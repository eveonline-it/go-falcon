package status

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

// Client interface for status-related ESI operations
type Client interface {
	GetServerStatus(ctx context.Context) (*ServerStatusResponse, error)
}

// ServerStatusResponse represents the EVE Online server status
type ServerStatusResponse struct {
	Players       int       `json:"players"`
	ServerVersion string    `json:"server_version"`
	StartTime     time.Time `json:"start_time"`
}

// StatusClient implements status-related ESI operations
type StatusClient struct {
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
	Set(key string, data []byte, headers http.Header) error
	RefreshExpiry(key string, headers http.Header) error
	SetConditionalHeaders(req *http.Request, key string) error
}

// RetryClient interface for retry operations
type RetryClient interface {
	DoWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error)
}

// NewStatusClient creates a new status client
func NewStatusClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &StatusClient{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetServerStatus retrieves EVE Online server status from ESI
func (c *StatusClient) GetServerStatus(ctx context.Context) (*ServerStatusResponse, error) {
	var span trace.Span
	endpoint := "/status"
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegate/status")
		ctx, span = tracer.Start(ctx, "status.GetServerStatus")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "status"),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("http.user_agent", c.userAgent),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting server status from ESI")

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var status ServerStatusResponse
		if err := json.Unmarshal(cachedData, &status); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached ESI status data")
			return &status, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create ESI status request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI status endpoint", "error", err)
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
		// Refresh the cache expiry time
		c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

		// Get cached data (even if expired)
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			slog.InfoContext(ctx, "ESI status not modified, refreshed cache expiry")

			var status ServerStatusResponse
			if err := json.Unmarshal(cachedData, &status); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &status, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI status endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read ESI status response", "error", err)
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

	var status ServerStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse ESI status response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("esi.players", status.Players),
			attribute.String("esi.server_version", status.ServerVersion),
			attribute.String("esi.start_time", status.StartTime.Format(time.RFC3339)),
		)
		span.SetStatus(codes.Ok, "successfully retrieved ESI status")
	}

	slog.InfoContext(ctx, "Successfully retrieved ESI status",
		slog.Int("players", status.Players),
		slog.String("server_version", status.ServerVersion),
		slog.Time("start_time", status.StartTime))

	return &status, nil
}
