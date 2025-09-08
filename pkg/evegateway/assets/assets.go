package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
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

// AssetsResult contains assets info and cache information
type AssetsResult struct {
	Data  []AssetResponse `json:"data"`
	Cache CacheInfo       `json:"cache"`
}

// Client interface for assets-related ESI operations
type Client interface {
	GetCharacterAssets(ctx context.Context, characterID int32, token string) ([]AssetResponse, error)
	GetCharacterAssetsWithCache(ctx context.Context, characterID int32, token string) (*AssetsResult, error)
	GetCorporationAssets(ctx context.Context, corporationID int32, token string) ([]AssetResponse, error)
	GetCorporationAssetsWithCache(ctx context.Context, corporationID int32, token string) (*AssetsResult, error)
}

// AssetResponse represents an EVE Online asset from ESI
type AssetResponse struct {
	ItemID          int64  `json:"item_id"`
	TypeID          int32  `json:"type_id"`
	LocationID      int64  `json:"location_id"`
	LocationFlag    string `json:"location_flag"`
	Quantity        int32  `json:"quantity"`
	IsSingleton     bool   `json:"is_singleton"`
	IsBlueprintCopy *bool  `json:"is_blueprint_copy,omitempty"`
}

// CacheManager interface for caching operations
type CacheManager interface {
	Get(key string) ([]byte, bool, error)
	GetWithExpiry(key string) ([]byte, bool, *time.Time, error)
	GetForNotModified(key string) ([]byte, bool, error)
	Set(key string, data []byte, headers http.Header) error
	RefreshExpiry(key string, headers http.Header) error
	SetConditionalHeaders(req *http.Request, key string) error
}

// RetryClient interface for retry operations
type RetryClient interface {
	DoWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error)
}

// ClientImpl implements the Client interface
type ClientImpl struct {
	httpClient   *http.Client
	baseURL      string
	userAgent    string
	cacheManager CacheManager
	retryClient  RetryClient
}

// NewAssetsClient creates a new assets client
func NewAssetsClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &ClientImpl{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetCharacterAssets retrieves character assets from ESI with authentication
func (c *ClientImpl) GetCharacterAssets(ctx context.Context, characterID int32, token string) ([]AssetResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/characters/%d/assets/", characterID)
	cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate")
		ctx, span = tracer.Start(ctx, "evegate.GetCharacterAssets")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "character_assets"),
			attribute.Int("esi.character_id", int(characterID)),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting character assets from ESI", "character_id", characterID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var assets []AssetResponse
		if err := json.Unmarshal(cachedData, &assets); err == nil {
			if span != nil {
				span.SetAttributes(
					attribute.Bool("cache.hit", true),
					attribute.Int("assets.count", len(assets)),
				)
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached character assets", "character_id", characterID, "count", len(assets))
			return assets, nil
		}
	}

	// Fetch all pages from ESI
	allAssets, err := c.fetchCharacterAssetsPages(ctx, characterID, token)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch character assets")
		}
		return nil, err
	}

	if span != nil {
		span.SetAttributes(
			attribute.Bool("cache.hit", false),
			attribute.Int("assets.count", len(allAssets)),
		)
		span.SetStatus(codes.Ok, "successfully retrieved character assets")
	}

	slog.InfoContext(ctx, "Successfully retrieved character assets", "character_id", characterID, "count", len(allAssets))
	return allAssets, nil
}

// GetCorporationAssets retrieves corporation assets from ESI with authentication
func (c *ClientImpl) GetCorporationAssets(ctx context.Context, corporationID int32, token string) ([]AssetResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/corporations/%d/assets/", corporationID)
	cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate")
		ctx, span = tracer.Start(ctx, "evegate.GetCorporationAssets")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "corporation_assets"),
			attribute.Int("esi.corporation_id", int(corporationID)),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting corporation assets from ESI", "corporation_id", corporationID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var assets []AssetResponse
		if err := json.Unmarshal(cachedData, &assets); err == nil {
			if span != nil {
				span.SetAttributes(
					attribute.Bool("cache.hit", true),
					attribute.Int("assets.count", len(assets)),
				)
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached corporation assets", "corporation_id", corporationID, "count", len(assets))
			return assets, nil
		}
	}

	// Fetch all pages from ESI
	allAssets, err := c.fetchCorporationAssetsPages(ctx, corporationID, token)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to fetch corporation assets")
		}
		return nil, err
	}

	if span != nil {
		span.SetAttributes(
			attribute.Bool("cache.hit", false),
			attribute.Int("assets.count", len(allAssets)),
		)
		span.SetStatus(codes.Ok, "successfully retrieved corporation assets")
	}

	slog.InfoContext(ctx, "Successfully retrieved corporation assets", "corporation_id", corporationID, "count", len(allAssets))
	return allAssets, nil
}

// GetCharacterAssetsWithCache retrieves character assets from ESI with cache information
func (c *ClientImpl) GetCharacterAssetsWithCache(ctx context.Context, characterID int32, token string) (*AssetsResult, error) {
	endpoint := fmt.Sprintf("/characters/%d/assets/", characterID)
	cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)

	// Check cache first and get expiry info
	cachedData, found, expiresAt, err := c.cacheManager.GetWithExpiry(cacheKey)
	if err == nil && found {
		var assets []AssetResponse
		if err := json.Unmarshal(cachedData, &assets); err == nil {
			return &AssetsResult{
				Data: assets,
				Cache: CacheInfo{
					Cached:    true,
					ExpiresAt: expiresAt,
				},
			}, nil
		}
	}

	// Cache miss - fetch from ESI
	assets, err := c.GetCharacterAssets(ctx, characterID, token)
	if err != nil {
		return nil, err
	}

	// Get cache metadata after fetching
	var cacheExpiresAt *time.Time
	if _, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		cacheExpiresAt = expiry
	}

	return &AssetsResult{
		Data: assets,
		Cache: CacheInfo{
			Cached:    false,
			ExpiresAt: cacheExpiresAt,
		},
	}, nil
}

// GetCorporationAssetsWithCache retrieves corporation assets from ESI with cache information
func (c *ClientImpl) GetCorporationAssetsWithCache(ctx context.Context, corporationID int32, token string) (*AssetsResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/assets/", corporationID)
	cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)

	// Check cache first and get expiry info
	cachedData, found, expiresAt, err := c.cacheManager.GetWithExpiry(cacheKey)
	if err == nil && found {
		var assets []AssetResponse
		if err := json.Unmarshal(cachedData, &assets); err == nil {
			return &AssetsResult{
				Data: assets,
				Cache: CacheInfo{
					Cached:    true,
					ExpiresAt: expiresAt,
				},
			}, nil
		}
	}

	// Cache miss - fetch from ESI
	assets, err := c.GetCorporationAssets(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	// Get cache metadata after fetching
	var cacheExpiresAt *time.Time
	if _, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		cacheExpiresAt = expiry
	}

	return &AssetsResult{
		Data: assets,
		Cache: CacheInfo{
			Cached:    false,
			ExpiresAt: cacheExpiresAt,
		},
	}, nil
}

// fetchCharacterAssetsPages handles pagination for character assets
func (c *ClientImpl) fetchCharacterAssetsPages(ctx context.Context, characterID int32, token string) ([]AssetResponse, error) {
	var allAssets []AssetResponse
	page := 1

	for {
		assets, totalPages, err := c.fetchCharacterAssetsPage(ctx, characterID, token, page)
		if err != nil {
			return nil, err
		}

		allAssets = append(allAssets, assets...)

		// Check if we have more pages
		if page >= totalPages || len(assets) == 0 {
			break
		}
		page++
	}

	return allAssets, nil
}

// fetchCharacterAssetsPage fetches a single page of character assets
func (c *ClientImpl) fetchCharacterAssetsPage(ctx context.Context, characterID int32, token string, page int) ([]AssetResponse, int, error) {
	endpoint := fmt.Sprintf("/characters/%d/assets/", characterID)
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if page > 1 {
		url += fmt.Sprintf("?page=%d", page)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Use retry mechanism
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to call ESI character assets endpoint", "error", err)
		return nil, 0, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "ESI character assets endpoint returned error", "status_code", resp.StatusCode)
		return nil, 0, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	// Get total pages from headers
	totalPages := 1
	if pagesHeader := resp.Header.Get("X-Pages"); pagesHeader != "" {
		if pages, err := strconv.Atoi(pagesHeader); err == nil {
			totalPages = pages
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to read character assets response", "error", err)
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	var assets []AssetResponse
	if err := json.Unmarshal(body, &assets); err != nil {
		slog.ErrorContext(ctx, "Failed to parse character assets response", "error", err)
		return nil, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the page if it's the first page
	if page == 1 {
		cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)
		c.cacheManager.Set(cacheKey, body, resp.Header)
	}

	return assets, totalPages, nil
}

// fetchCorporationAssetsPages handles pagination for corporation assets
func (c *ClientImpl) fetchCorporationAssetsPages(ctx context.Context, corporationID int32, token string) ([]AssetResponse, error) {
	var allAssets []AssetResponse
	page := 1

	for {
		assets, totalPages, err := c.fetchCorporationAssetsPage(ctx, corporationID, token, page)
		if err != nil {
			return nil, err
		}

		allAssets = append(allAssets, assets...)

		// Check if we have more pages
		if page >= totalPages || len(assets) == 0 {
			break
		}
		page++
	}

	return allAssets, nil
}

// fetchCorporationAssetsPage fetches a single page of corporation assets
func (c *ClientImpl) fetchCorporationAssetsPage(ctx context.Context, corporationID int32, token string, page int) ([]AssetResponse, int, error) {
	endpoint := fmt.Sprintf("/corporations/%d/assets/", corporationID)
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if page > 1 {
		url += fmt.Sprintf("?page=%d", page)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Use retry mechanism
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to call ESI corporation assets endpoint", "error", err)
		return nil, 0, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "ESI corporation assets endpoint returned error", "status_code", resp.StatusCode)
		return nil, 0, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	// Get total pages from headers
	totalPages := 1
	if pagesHeader := resp.Header.Get("X-Pages"); pagesHeader != "" {
		if pages, err := strconv.Atoi(pagesHeader); err == nil {
			totalPages = pages
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to read corporation assets response", "error", err)
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	var assets []AssetResponse
	if err := json.Unmarshal(body, &assets); err != nil {
		slog.ErrorContext(ctx, "Failed to parse corporation assets response", "error", err)
		return nil, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the page if it's the first page
	if page == 1 {
		cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)
		c.cacheManager.Set(cacheKey, body, resp.Header)
	}

	return assets, totalPages, nil
}
