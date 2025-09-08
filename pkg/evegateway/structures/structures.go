package structures

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

// StructureResult contains structure info and cache information
type StructureResult struct {
	Data  *StructureResponse `json:"data"`
	Cache CacheInfo          `json:"cache"`
}

// Client interface for structures-related ESI operations
type Client interface {
	GetStructure(ctx context.Context, structureID int64, token string) (*StructureResponse, error)
	GetStructureWithCache(ctx context.Context, structureID int64, token string) (*StructureResult, error)
}

// StructureResponse represents an EVE Online structure from ESI
type StructureResponse struct {
	Name            string     `json:"name"`
	OwnerID         int32      `json:"owner_id"`
	Position        Position   `json:"position"`
	SolarSystemID   int32      `json:"solar_system_id"`
	TypeID          int32      `json:"type_id"`
	Services        []string   `json:"services,omitempty"`
	State           string     `json:"state,omitempty"`
	StateTimerStart *time.Time `json:"state_timer_start,omitempty"`
	StateTimerEnd   *time.Time `json:"state_timer_end,omitempty"`
	FuelExpires     *time.Time `json:"fuel_expires,omitempty"`
	UnanchorsAt     *time.Time `json:"unanchors_at,omitempty"`
}

// Position represents 3D coordinates
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
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

// NewStructuresClient creates a new structures client
func NewStructuresClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &ClientImpl{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetStructure retrieves structure information from ESI
func (c *ClientImpl) GetStructure(ctx context.Context, structureID int64, token string) (*StructureResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/universe/structures/%d/", structureID)
	cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate")
		ctx, span = tracer.Start(ctx, "evegate.GetStructure")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "universe_structure"),
			attribute.Int64("esi.structure_id", structureID),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting structure info from ESI", "structure_id", structureID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var structure StructureResponse
		if err := json.Unmarshal(cachedData, &structure); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached structure data", "structure_id", structureID)
			return &structure, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%s", c.baseURL, endpoint), nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create ESI structure request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Add conditional headers if we have cached data
	c.cacheManager.SetConditionalHeaders(req, cacheKey)

	if span != nil {
		span.SetAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
		)
	}

	// Use retry mechanism with exponential backoff
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3) // Max 3 retries
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
		slog.ErrorContext(ctx, "Failed to call ESI structure endpoint", "error", err)
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode),
			attribute.String("http.status_text", resp.Status),
		)
	}

	// Handle 304 Not Modified - return cached data
	if resp.StatusCode == http.StatusNotModified {
		// Refresh the cache expiry time
		c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

		// Get cached data (even if expired)
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			slog.InfoContext(ctx, "ESI structure not modified, refreshed cache expiry")

			var structure StructureResponse
			if err := json.Unmarshal(cachedData, &structure); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &structure, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI structure endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read ESI structure response", "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.response_size", len(body)),
			attribute.Bool("cache.hit", false),
		)
	}

	// Update cache with new data
	c.cacheManager.Set(cacheKey, body, resp.Header)

	var structure StructureResponse
	if err := json.Unmarshal(body, &structure); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse ESI structure response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.String("structure.name", structure.Name),
			attribute.Int("structure.owner_id", int(structure.OwnerID)),
			attribute.Int("structure.solar_system_id", int(structure.SolarSystemID)),
		)
		span.SetStatus(codes.Ok, "successfully retrieved ESI structure")
	}

	slog.InfoContext(ctx, "Successfully retrieved ESI structure",
		slog.Int64("structure_id", structureID),
		slog.String("name", structure.Name),
		slog.Any("owner_id", structure.OwnerID),
		slog.Any("solar_system_id", structure.SolarSystemID))

	return &structure, nil
}

// GetStructureWithCache retrieves structure information from ESI with cache information
func (c *ClientImpl) GetStructureWithCache(ctx context.Context, structureID int64, token string) (*StructureResult, error) {
	endpoint := fmt.Sprintf("/universe/structures/%d/", structureID)
	cacheKey := fmt.Sprintf("%s%s?token=%s", c.baseURL, endpoint, token)

	// Check cache first and get expiry info
	cachedData, found, expiresAt, err := c.cacheManager.GetWithExpiry(cacheKey)
	if err == nil && found {
		var structure StructureResponse
		if err := json.Unmarshal(cachedData, &structure); err == nil {
			return &StructureResult{
				Data: &structure,
				Cache: CacheInfo{
					Cached:    true,
					ExpiresAt: expiresAt,
				},
			}, nil
		}
	}

	// Cache miss - fetch from ESI
	structure, err := c.GetStructure(ctx, structureID, token)
	if err != nil {
		return nil, err
	}

	// Get cache metadata after fetching
	var cacheExpiresAt *time.Time
	if _, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		cacheExpiresAt = expiry
	}

	return &StructureResult{
		Data: structure,
		Cache: CacheInfo{
			Cached:    false,
			ExpiresAt: cacheExpiresAt,
		},
	}, nil
}
