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
	GetCharacterSkillQueue(ctx context.Context, characterID int, token string) ([]SkillQueueItem, error)
	GetCharacterSkillQueueWithCache(ctx context.Context, characterID int, token string) (*SkillQueueResult, error)
	GetCharacterSkills(ctx context.Context, characterID int, token string) (*CharacterSkillsResponse, error)
	GetCharacterSkillsWithCache(ctx context.Context, characterID int, token string) (*CharacterSkillsResult, error)
	GetCharacterCorporationHistory(ctx context.Context, characterID int) ([]CorporationHistoryEntry, error)
	GetCharacterCorporationHistoryWithCache(ctx context.Context, characterID int) (*CorporationHistoryResult, error)
	GetCharacterClones(ctx context.Context, characterID int, token string) (*ClonesResponse, error)
	GetCharacterClonesWithCache(ctx context.Context, characterID int, token string) (*ClonesResult, error)
	GetCharacterImplants(ctx context.Context, characterID int, token string) ([]int, error)
	GetCharacterImplantsWithCache(ctx context.Context, characterID int, token string) (*ImplantsResult, error)
	GetCharacterLocation(ctx context.Context, characterID int, token string) (*LocationResponse, error)
	GetCharacterLocationWithCache(ctx context.Context, characterID int, token string) (*LocationResult, error)
	GetCharacterFatigue(ctx context.Context, characterID int, token string) (*FatigueResponse, error)
	GetCharacterFatigueWithCache(ctx context.Context, characterID int, token string) (*FatigueResult, error)
	GetCharacterOnline(ctx context.Context, characterID int, token string) (*OnlineResponse, error)
	GetCharacterOnlineWithCache(ctx context.Context, characterID int, token string) (*OnlineResult, error)
	GetCharacterShip(ctx context.Context, characterID int, token string) (*ShipResponse, error)
	GetCharacterShipWithCache(ctx context.Context, characterID int, token string) (*ShipResult, error)
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

// SkillQueueItem represents a single skill in the character's skill queue
type SkillQueueItem struct {
	SkillID         int        `json:"skill_id"`
	FinishedLevel   int        `json:"finished_level"`
	QueuePosition   int        `json:"queue_position"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	FinishDate      *time.Time `json:"finish_date,omitempty"`
	TrainingStartSP *int       `json:"training_start_sp,omitempty"`
	LevelEndSP      *int       `json:"level_end_sp,omitempty"`
	LevelStartSP    *int       `json:"level_start_sp,omitempty"`
}

// SkillQueueResult contains skill queue and cache information
type SkillQueueResult struct {
	Data  []SkillQueueItem `json:"data"`
	Cache CacheInfo        `json:"cache"`
}

// Skill represents a single trained skill
type Skill struct {
	SkillID            int `json:"skill_id"`
	SkillpointsInSkill int `json:"skillpoints_in_skill"`
	TrainedSkillLevel  int `json:"trained_skill_level"`
	ActiveSkillLevel   int `json:"active_skill_level"`
}

// CharacterSkillsResponse represents character skills information
type CharacterSkillsResponse struct {
	Skills        []Skill `json:"skills"`
	TotalSP       int64   `json:"total_sp"`
	UnallocatedSP *int    `json:"unallocated_sp,omitempty"`
}

// CharacterSkillsResult contains skills and cache information
type CharacterSkillsResult struct {
	Data  *CharacterSkillsResponse `json:"data"`
	Cache CacheInfo                `json:"cache"`
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

// GetCharacterSkillQueue retrieves character skill queue from ESI
func (c *CharacterClient) GetCharacterSkillQueue(ctx context.Context, characterID int, token string) ([]SkillQueueItem, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/skillqueue/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterSkillQueue")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.skillqueue"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character skill queue from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var skillQueue []SkillQueueItem
		if err := json.Unmarshal(cachedData, &skillQueue); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character skill queue", "character_id", characterID)
			return skillQueue, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character skill queue request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI character skill queue endpoint", "error", err)
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
			slog.InfoContext(ctx, "Character skill queue not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var skillQueue []SkillQueueItem
			if err := json.Unmarshal(cachedData, &skillQueue); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return skillQueue, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d skill queue", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character skill queue endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character skill queue response", "error", err)
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

	var skillQueue []SkillQueueItem
	if err := json.Unmarshal(body, &skillQueue); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character skill queue response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("skill_queue.length", len(skillQueue)),
		)
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character skill queue from ESI", "character_id", characterID, "queue_length", len(skillQueue))
	return skillQueue, nil
}

// GetCharacterSkillQueueWithCache retrieves character skill queue with cache information
func (c *CharacterClient) GetCharacterSkillQueueWithCache(ctx context.Context, characterID int, token string) (*SkillQueueResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/skillqueue/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterSkillQueue(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &SkillQueueResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCharacterSkills retrieves character skills from ESI
func (c *CharacterClient) GetCharacterSkills(ctx context.Context, characterID int, token string) (*CharacterSkillsResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/skills/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterSkills")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.skills"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character skills from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var skills CharacterSkillsResponse
		if err := json.Unmarshal(cachedData, &skills); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character skills", "character_id", characterID)
			return &skills, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character skills request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI character skills endpoint", "error", err)
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
			slog.InfoContext(ctx, "Character skills not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var skills CharacterSkillsResponse
			if err := json.Unmarshal(cachedData, &skills); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &skills, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d skills", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character skills endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character skills response", "error", err)
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

	var skills CharacterSkillsResponse
	if err := json.Unmarshal(body, &skills); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character skills response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("skills.count", len(skills.Skills)),
			attribute.Int64("skills.total_sp", skills.TotalSP),
		)
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character skills from ESI", "character_id", characterID, "skill_count", len(skills.Skills))
	return &skills, nil
}

// GetCharacterSkillsWithCache retrieves character skills with cache information
func (c *CharacterClient) GetCharacterSkillsWithCache(ctx context.Context, characterID int, token string) (*CharacterSkillsResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/skills/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterSkills(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &CharacterSkillsResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// CorporationHistoryEntry represents a single corporation history entry
type CorporationHistoryEntry struct {
	CorporationID int    `json:"corporation_id"`
	IsDeleted     bool   `json:"is_deleted,omitempty"`
	RecordID      int    `json:"record_id"`
	StartDate     string `json:"start_date"`
}

// CorporationHistoryResult represents the result of a corporation history request with cache info
type CorporationHistoryResult struct {
	Data  []CorporationHistoryEntry
	Cache CacheInfo
}

// GetCharacterCorporationHistory retrieves the character's corporation history from ESI
func (c *CharacterClient) GetCharacterCorporationHistory(ctx context.Context, characterID int) ([]CorporationHistoryEntry, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/corporationhistory/", characterID)

	// Create an absolute URL
	absoluteURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Start tracing if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterCorporationHistory")
		defer span.End()
	}

	// Create a GET request
	req, err := http.NewRequestWithContext(ctx, "GET", absoluteURL, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create request")
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to execute request")
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
			span.SetStatus(codes.Error, fmt.Sprintf("Unexpected status code: %d", resp.StatusCode))
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response body
	var history []CorporationHistoryEntry
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&history); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to decode response")
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the response if caching is enabled
	if c.cacheManager != nil && len(history) > 0 {
		cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
		if cacheData, err := json.Marshal(history); err == nil {
			// Use the response headers for caching
			c.cacheManager.Set(cacheKey, cacheData, resp.Header)
		}
	}

	if span != nil {
		span.SetAttributes(attribute.Int("corporation_history.count", len(history)))
		span.SetStatus(codes.Ok, "Successfully retrieved corporation history")
	}

	return history, nil
}

// GetCharacterCorporationHistoryWithCache retrieves character corporation history with cache information
func (c *CharacterClient) GetCharacterCorporationHistoryWithCache(ctx context.Context, characterID int) (*CorporationHistoryResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/corporationhistory/", characterID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Check if data is cached and get expiry
	cachedData, cached, cacheExpiry, err := c.cacheManager.GetWithExpiry(cacheKey)

	// If cached and no error, try to use it
	if err == nil && cached {
		var history []CorporationHistoryEntry
		if err := json.Unmarshal(cachedData, &history); err == nil {
			return &CorporationHistoryResult{
				Data:  history,
				Cache: CacheInfo{Cached: true, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Fetch from ESI
	data, err := c.GetCharacterCorporationHistory(ctx, characterID)
	if err != nil {
		return nil, err
	}

	// Calculate expiry time (24 hours from now as default)
	expiryTime := time.Now().Add(24 * time.Hour)
	return &CorporationHistoryResult{
		Data:  data,
		Cache: CacheInfo{Cached: false, ExpiresAt: &expiryTime},
	}, nil
}

// ClonesResponse represents the character's clone information from ESI
type ClonesResponse struct {
	HomeLocation          *HomeLocation `json:"home_location,omitempty"`
	JumpClones            []JumpClone   `json:"jump_clones"`
	LastCloneJumpDate     *time.Time    `json:"last_clone_jump_date,omitempty"`
	LastStationChangeDate *time.Time    `json:"last_station_change_date,omitempty"`
}

// HomeLocation represents the character's home location
type HomeLocation struct {
	LocationID   int64  `json:"location_id"`
	LocationType string `json:"location_type"`
}

// JumpClone represents a single jump clone
type JumpClone struct {
	Implants     []int  `json:"implants"`
	JumpCloneID  int    `json:"jump_clone_id"`
	LocationID   int64  `json:"location_id"`
	LocationType string `json:"location_type"`
	Name         string `json:"name,omitempty"`
}

// ClonesResult represents the result of a clones request with cache info
type ClonesResult struct {
	Data  *ClonesResponse
	Cache CacheInfo
}

// ImplantsResult represents the result of an implants request with cache info
type ImplantsResult struct {
	Data  []int
	Cache CacheInfo
}

// GetCharacterClones retrieves the character's clone information from ESI
func (c *CharacterClient) GetCharacterClones(ctx context.Context, characterID int, token string) (*ClonesResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/clones/", characterID)

	// Create an absolute URL
	absoluteURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Start tracing if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterClones")
		defer span.End()
	}

	// Create a GET request
	req, err := http.NewRequestWithContext(ctx, "GET", absoluteURL, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create request")
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to execute request")
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
			span.SetStatus(codes.Error, fmt.Sprintf("Unexpected status code: %d", resp.StatusCode))
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response body
	var clones ClonesResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&clones); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to decode response")
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the response if caching is enabled
	if c.cacheManager != nil {
		cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)
		if cacheData, err := json.Marshal(clones); err == nil {
			// Use the response headers for caching
			c.cacheManager.Set(cacheKey, cacheData, resp.Header)
		}
	}

	if span != nil {
		span.SetAttributes(attribute.Int("jump_clones.count", len(clones.JumpClones)))
		span.SetStatus(codes.Ok, "Successfully retrieved character clones")
	}

	return &clones, nil
}

// GetCharacterClonesWithCache retrieves character clones with cache information
func (c *CharacterClient) GetCharacterClonesWithCache(ctx context.Context, characterID int, token string) (*ClonesResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/clones/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	cachedData, cached, cacheExpiry, err := c.cacheManager.GetWithExpiry(cacheKey)

	// If cached and no error, try to use it
	if err == nil && cached {
		var clones ClonesResponse
		if err := json.Unmarshal(cachedData, &clones); err == nil {
			return &ClonesResult{
				Data:  &clones,
				Cache: CacheInfo{Cached: true, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Fetch from ESI
	data, err := c.GetCharacterClones(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	// Calculate expiry time (1 hour from now as default)
	expiryTime := time.Now().Add(1 * time.Hour)
	return &ClonesResult{
		Data:  data,
		Cache: CacheInfo{Cached: false, ExpiresAt: &expiryTime},
	}, nil
}

// GetCharacterImplants retrieves the character's active implants from ESI
func (c *CharacterClient) GetCharacterImplants(ctx context.Context, characterID int, token string) ([]int, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/implants/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterImplants")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.implants"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character implants from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var implants []int
		if err := json.Unmarshal(cachedData, &implants); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character implants", "character_id", characterID)
			return implants, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character implants request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI character implants endpoint", "error", err)
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
			slog.InfoContext(ctx, "Character implants not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var implants []int
			if err := json.Unmarshal(cachedData, &implants); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return implants, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d implants", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character implants endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character implants response", "error", err)
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

	var implants []int
	if err := json.Unmarshal(body, &implants); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character implants response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("implants.count", len(implants)),
		)
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character implants from ESI", "character_id", characterID, "implant_count", len(implants))
	return implants, nil
}

// GetCharacterImplantsWithCache retrieves character implants with cache information
func (c *CharacterClient) GetCharacterImplantsWithCache(ctx context.Context, characterID int, token string) (*ImplantsResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/implants/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterImplants(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &ImplantsResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// LocationResponse represents the character's current location
type LocationResponse struct {
	SolarSystemID int    `json:"solar_system_id"`
	StationID     *int   `json:"station_id,omitempty"`
	StructureID   *int64 `json:"structure_id,omitempty"`
}

// LocationResult represents the result of a location request with cache info
type LocationResult struct {
	Data  *LocationResponse
	Cache CacheInfo
}

// FatigueResponse represents the character's jump fatigue information
type FatigueResponse struct {
	JumpFatigueExpireDate *time.Time `json:"jump_fatigue_expire_date,omitempty"`
	LastJumpDate          *time.Time `json:"last_jump_date,omitempty"`
	LastUpdateDate        *time.Time `json:"last_update_date,omitempty"`
}

// FatigueResult represents the result of a fatigue request with cache info
type FatigueResult struct {
	Data  *FatigueResponse
	Cache CacheInfo
}

// OnlineResponse represents the character's online status information
type OnlineResponse struct {
	Online      bool       `json:"online"`
	LastLogin   *time.Time `json:"last_login,omitempty"`
	LastLogout  *time.Time `json:"last_logout,omitempty"`
	LoginsToday *int       `json:"logins,omitempty"`
}

// OnlineResult represents the result of an online status request with cache info
type OnlineResult struct {
	Data  *OnlineResponse
	Cache CacheInfo
}

// ShipResponse represents the character's current ship information
type ShipResponse struct {
	ShipItemID int64  `json:"ship_item_id"`
	ShipName   string `json:"ship_name"`
	ShipTypeID int    `json:"ship_type_id"`
}

// ShipResult represents the result of a ship request with cache info
type ShipResult struct {
	Data  *ShipResponse
	Cache CacheInfo
}

// GetCharacterLocation retrieves the character's current location from ESI
func (c *CharacterClient) GetCharacterLocation(ctx context.Context, characterID int, token string) (*LocationResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/location/", characterID)

	// Create an absolute URL
	absoluteURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Start tracing if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterLocation")
		defer span.End()
	}

	// Create a GET request
	req, err := http.NewRequestWithContext(ctx, "GET", absoluteURL, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create request")
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to execute request")
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
			span.SetStatus(codes.Error, fmt.Sprintf("Unexpected status code: %d", resp.StatusCode))
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response body
	var location LocationResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&location); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to decode response")
		}
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the response if caching is enabled
	if c.cacheManager != nil {
		cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)
		if cacheData, err := json.Marshal(location); err == nil {
			// Use the response headers for caching (5 seconds for location)
			c.cacheManager.Set(cacheKey, cacheData, resp.Header)
		}
	}

	if span != nil {
		span.SetAttributes(attribute.Int("solar_system_id", location.SolarSystemID))
		span.SetStatus(codes.Ok, "Successfully retrieved character location")
	}

	return &location, nil
}

// GetCharacterLocationWithCache retrieves character location with cache information
func (c *CharacterClient) GetCharacterLocationWithCache(ctx context.Context, characterID int, token string) (*LocationResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/location/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	cachedData, cached, cacheExpiry, err := c.cacheManager.GetWithExpiry(cacheKey)

	// If cached and no error, try to use it
	if err == nil && cached {
		var location LocationResponse
		if err := json.Unmarshal(cachedData, &location); err == nil {
			return &LocationResult{
				Data:  &location,
				Cache: CacheInfo{Cached: true, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Fetch from ESI
	data, err := c.GetCharacterLocation(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &LocationResult{
		Data:  data,
		Cache: CacheInfo{Cached: false, ExpiresAt: &now},
	}, nil
}

// GetCharacterFatigue retrieves the character's jump fatigue information from ESI
func (c *CharacterClient) GetCharacterFatigue(ctx context.Context, characterID int, token string) (*FatigueResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/fatigue/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterFatigue")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.fatigue"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character fatigue from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var fatigue FatigueResponse
		if err := json.Unmarshal(cachedData, &fatigue); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character fatigue", "character_id", characterID)
			return &fatigue, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character fatigue request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI character fatigue endpoint", "error", err)
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
			slog.InfoContext(ctx, "Character fatigue not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var fatigue FatigueResponse
			if err := json.Unmarshal(cachedData, &fatigue); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &fatigue, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d fatigue", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character fatigue endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character fatigue response", "error", err)
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

	var fatigue FatigueResponse
	if err := json.Unmarshal(body, &fatigue); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character fatigue response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character fatigue from ESI", "character_id", characterID)
	return &fatigue, nil
}

// GetCharacterFatigueWithCache retrieves character fatigue with cache information
func (c *CharacterClient) GetCharacterFatigueWithCache(ctx context.Context, characterID int, token string) (*FatigueResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/fatigue/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterFatigue(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &FatigueResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCharacterOnline retrieves the character's online status information from ESI
func (c *CharacterClient) GetCharacterOnline(ctx context.Context, characterID int, token string) (*OnlineResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/online/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterOnline")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.online"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character online status from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var online OnlineResponse
		if err := json.Unmarshal(cachedData, &online); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character online status", "character_id", characterID)
			return &online, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character online request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI character online endpoint", "error", err)
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
			slog.InfoContext(ctx, "Character online status not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var online OnlineResponse
			if err := json.Unmarshal(cachedData, &online); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &online, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d online status", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character online endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character online response", "error", err)
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

	var online OnlineResponse
	if err := json.Unmarshal(body, &online); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character online response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Bool("online.status", online.Online),
		)
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character online status from ESI", "character_id", characterID, "online", online.Online)
	return &online, nil
}

// GetCharacterOnlineWithCache retrieves character online status with cache information
func (c *CharacterClient) GetCharacterOnlineWithCache(ctx context.Context, characterID int, token string) (*OnlineResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/online/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterOnline(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &OnlineResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCharacterShip retrieves the character's current ship information from ESI
func (c *CharacterClient) GetCharacterShip(ctx context.Context, characterID int, token string) (*ShipResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/ship/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/character")
		ctx, span = tracer.Start(ctx, "character.GetCharacterShip")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character.ship"),
			attribute.Int("esi.character_id", characterID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
			attribute.Bool("auth.required", true),
		)
	}

	slog.InfoContext(ctx, "Requesting character ship from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var ship ShipResponse
		if err := json.Unmarshal(cachedData, &ship); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character ship", "character_id", characterID)
			return &ship, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create character ship request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI character ship endpoint", "error", err)
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
			slog.InfoContext(ctx, "Character ship not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var ship ShipResponse
			if err := json.Unmarshal(cachedData, &ship); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &ship, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "character_id", characterID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for character %d ship", characterID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI character ship endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read character ship response", "error", err)
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

	var ship ShipResponse
	if err := json.Unmarshal(body, &ship); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse character ship response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.String("ship.name", ship.ShipName),
			attribute.Int("ship.type_id", ship.ShipTypeID),
		)
		span.SetStatus(codes.Ok, "success")
	}

	slog.InfoContext(ctx, "Successfully fetched character ship from ESI", "character_id", characterID, "ship_name", ship.ShipName)
	return &ship, nil
}

// GetCharacterShipWithCache retrieves character ship with cache information
func (c *CharacterClient) GetCharacterShipWithCache(ctx context.Context, characterID int, token string) (*ShipResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/ship/", characterID)
	cacheKey := fmt.Sprintf("%s%s:%s", c.baseURL, endpoint, token)

	// Check if data is cached and get expiry
	_, cached, cacheExpiry, _ := c.cacheManager.GetWithExpiry(cacheKey)

	data, err := c.GetCharacterShip(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	return &ShipResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}
