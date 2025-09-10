package character

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/pkg/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// CacheInfo represents cache information for responses
type CacheInfo struct {
	Cached    bool       `json:"cached"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CharacterInfoResult contains character info and cache information
type CharacterInfoResult struct {
	Data  *CharacterInfoResponse `json:"data"`
	Cache CacheInfo              `json:"cache"`
}

// CharacterPortraitResult contains portrait info and cache information
type CharacterPortraitResult struct {
	Data  *CharacterPortraitResponse `json:"data"`
	Cache CacheInfo                  `json:"cache"`
}

// Client interface for character-related ESI operations
type Client interface {
	GetCharacterInfo(ctx context.Context, characterID int) (*CharacterInfoResponse, error)
	GetCharacterInfoWithCache(ctx context.Context, characterID int) (*CharacterInfoResult, error)
	GetCharacterPortrait(ctx context.Context, characterID int) (*CharacterPortraitResponse, error)
	GetCharacterPortraitWithCache(ctx context.Context, characterID int) (*CharacterPortraitResult, error)
	GetCharactersAffiliation(ctx context.Context, characterIDs []int) ([]CharacterAffiliation, error)
	GetCharactersAffiliationWithCache(ctx context.Context, characterIDs []int) (*CharacterAffiliationResult, error)
	GetCharacterAttributes(ctx context.Context, characterID int, token string) (*CharacterAttributesResponse, error)
	GetCharacterAttributesWithCache(ctx context.Context, characterID int, token string) (*CharacterAttributesResult, error)
}

// CharacterInfoResponse represents character public information
type CharacterInfoResponse struct {
	CharacterID    int       `json:"character_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CorporationID  int       `json:"corporation_id"`
	AllianceID     int       `json:"alliance_id,omitempty"`
	Birthday       time.Time `json:"birthday"`
	Gender         string    `json:"gender"`
	RaceID         int       `json:"race_id"`
	BloodlineID    int       `json:"bloodline_id"`
	AncestryID     int       `json:"ancestry_id,omitempty"`
	SecurityStatus float64   `json:"security_status,omitempty"`
	FactionID      int       `json:"faction_id,omitempty"`
}

// CharacterPortraitResponse represents character portrait URLs
type CharacterPortraitResponse struct {
	Px64x64   string `json:"px64x64"`
	Px128x128 string `json:"px128x128"`
	Px256x256 string `json:"px256x256"`
	Px512x512 string `json:"px512x512"`
}

// CharacterAttributesResponse represents character attributes
type CharacterAttributesResponse struct {
	Charisma                 int        `json:"charisma"`
	Intelligence             int        `json:"intelligence"`
	Memory                   int        `json:"memory"`
	Perception               int        `json:"perception"`
	Willpower                int        `json:"willpower"`
	AccruedRemapCooldownDate *time.Time `json:"accrued_remap_cooldown_date,omitempty"`
	BonusRemaps              *int       `json:"bonus_remaps,omitempty"`
	LastRemapDate            *time.Time `json:"last_remap_date,omitempty"`
}

// CharacterAttributesResult contains attributes and cache information
type CharacterAttributesResult struct {
	Data  *CharacterAttributesResponse `json:"data"`
	Cache CacheInfo                    `json:"cache"`
}

// CharacterClient implements character-related ESI operations
type CharacterClient struct {
	httpClient   *http.Client
	baseURL      string
	userAgent    string
	cacheManager CacheManager
	retryClient  RetryClient
}

// CacheManager interface for caching operations
type CacheManager interface {
	Get(key string) ([]byte, bool, error)
	GetWithExpiry(key string) ([]byte, bool, *time.Time, error)
	GetForNotModified(key string) ([]byte, bool, error)
	GetMetadata(key string) (map[string]interface{}, error)
	Set(key string, data []byte, headers http.Header) error
	RefreshExpiry(key string, headers http.Header) error
	SetConditionalHeaders(req *http.Request, key string) error
}

// RetryClient interface for retry operations
type RetryClient interface {
	DoWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error)
}

// NewCharacterClient creates a new character client
func NewCharacterClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &CharacterClient{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetCharacterInfo retrieves character public information from ESI
func (c *CharacterClient) GetCharacterInfo(ctx context.Context, characterID int) (*CharacterInfoResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/", characterID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterInfo")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting character info from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var character CharacterInfoResponse
		if err := json.Unmarshal(cachedData, &character); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character data", "character_id", characterID)
			return &character, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character info request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	// Add conditional headers if we have cached data
	c.cacheManager.SetConditionalHeaders(req, cacheKey)

	if span != nil {
		span.SetAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
		)
	}

	// Use retry mechanism
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
		slog.ErrorContext(ctx, "Failed to call ESI character endpoint", "error", err)
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode),
			attribute.String("http.status_text", resp.Status),
		)
	}

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		// Use GetForNotModified which returns cached data even if expired
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			slog.InfoContext(ctx, "Character info not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var character CharacterInfoResponse
			if err := json.Unmarshal(cachedData, &character); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &character, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character info response", "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.response_size", len(body)),
			attribute.Bool("cache.hit", false),
		)
	}

	// Update cache
	c.cacheManager.Set(cacheKey, body, resp.Header)

	var character CharacterInfoResponse
	if err := json.Unmarshal(body, &character); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character info response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.String("esi.character_name", character.Name),
			attribute.Int("esi.corporation_id", character.CorporationID),
		)
		span.SetStatus(codes.Ok, "successfully retrieved character info")
	}

	slog.InfoContext(ctx, "Successfully retrieved character info",
		slog.Int("character_id", character.CharacterID),
		slog.String("name", character.Name),
		slog.Int("corporation_id", character.CorporationID))

	return &character, nil
}

// GetCharacterInfoWithCache retrieves character public information from ESI with cache info
func (c *CharacterClient) GetCharacterInfoWithCache(ctx context.Context, characterID int) (*CharacterInfoResult, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/", characterID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterInfoWithCache")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting character info from ESI with cache info", "character_id", characterID)

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var character CharacterInfoResponse
		if err := json.Unmarshal(cachedData, &character); err == nil {
			cached = true
			cacheExpiry = expiry
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character data", "character_id", characterID)
			return &CharacterInfoResult{
				Data:  &character,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCharacterInfo(ctx, characterID)
	if err != nil {
		return nil, err
	}

	return &CharacterInfoResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCharacterPortrait retrieves character portrait URLs from ESI
func (c *CharacterClient) GetCharacterPortrait(ctx context.Context, characterID int) (*CharacterPortraitResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/portrait/", characterID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterPortrait")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character_portrait"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting character portrait from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var portrait CharacterPortraitResponse
		if err := json.Unmarshal(cachedData, &portrait); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character portrait data", "character_id", characterID)
			return &portrait, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	c.cacheManager.SetConditionalHeaders(req, cacheKey)

	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		// Use GetForNotModified which returns cached data even if expired
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			var portrait CharacterPortraitResponse
			if err := json.Unmarshal(cachedData, &portrait); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "Character portrait not modified, using cached data")

				// Refresh the expiry since ESI confirmed data is still valid
				c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

				return &portrait, nil
			}
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d portrait", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.cacheManager.Set(cacheKey, body, resp.Header)

	var portrait CharacterPortraitResponse
	if err := json.Unmarshal(body, &portrait); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetStatus(codes.Ok, "successfully retrieved character portrait")
	}

	return &portrait, nil
}

// GetCharacterPortraitWithCache retrieves character portrait URLs from ESI with cache info
func (c *CharacterClient) GetCharacterPortraitWithCache(ctx context.Context, characterID int) (*CharacterPortraitResult, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/portrait/", characterID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterPortraitWithCache")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character_portrait"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting character portrait from ESI with cache info", "character_id", characterID)

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var portrait CharacterPortraitResponse
		if err := json.Unmarshal(cachedData, &portrait); err == nil {
			cached = true
			cacheExpiry = expiry
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character portrait data", "character_id", characterID)
			return &CharacterPortraitResult{
				Data:  &portrait,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCharacterPortrait(ctx, characterID)
	if err != nil {
		return nil, err
	}

	return &CharacterPortraitResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCharacterAttributes retrieves character attributes from ESI
func (c *CharacterClient) GetCharacterAttributes(ctx context.Context, characterID int, token string) (*CharacterAttributesResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/attributes/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterAttributes")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.attributes"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character attributes from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var attributes CharacterAttributesResponse
		if err := json.Unmarshal(cachedData, &attributes); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character attributes", "character_id", characterID)
			return &attributes, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character attributes request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Add conditional headers if we have cached data
	c.cacheManager.SetConditionalHeaders(req, cacheKey)

	if span != nil {
		span.SetAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
		)
	}

	// Use retry mechanism
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
		slog.ErrorContext(ctx, "Failed to call ESI character attributes endpoint", "error", err)
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode),
			attribute.String("http.status_text", resp.Status),
		)
	}

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		// Use GetForNotModified which returns cached data even if expired
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			slog.InfoContext(ctx, "Character attributes not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var attributes CharacterAttributesResponse
			if err := json.Unmarshal(cachedData, &attributes); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &attributes, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character attributes endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character attributes response", "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.response_size", len(body)),
			attribute.Bool("cache.hit", false),
		)
	}

	// Update cache
	c.cacheManager.Set(cacheKey, body, resp.Header)

	var attributes CharacterAttributesResponse
	if err := json.Unmarshal(body, &attributes); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character attributes response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character attributes from ESI", "character_id", characterID)
	return &attributes, nil
}

// GetCharacterAttributesWithCache retrieves character attributes with cache information
func (c *CharacterClient) GetCharacterAttributesWithCache(ctx context.Context, characterID int, token string) (*CharacterAttributesResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/attributes/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterAttributes(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &CharacterAttributesResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}
