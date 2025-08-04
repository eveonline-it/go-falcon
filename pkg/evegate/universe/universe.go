package universe

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

// Client interface for universe-related ESI operations
type Client interface {
	GetSystemInfo(ctx context.Context, systemID int) (*SystemInfoResponse, error)
	GetStationInfo(ctx context.Context, stationID int) (*StationInfoResponse, error)
}

// SystemInfoResponse represents solar system information
type SystemInfoResponse struct {
	SystemID        int     `json:"system_id"`
	Name            string  `json:"name"`
	ConstellationID int     `json:"constellation_id"`
	SecurityStatus  float64 `json:"security_status"`
	SecurityClass   string  `json:"security_class,omitempty"`
	StarID          int     `json:"star_id,omitempty"`
	Stargates       []int   `json:"stargates,omitempty"`
	Stations        []int   `json:"stations,omitempty"`
	Planets         []int   `json:"planets,omitempty"`
}

// StationInfoResponse represents station information
type StationInfoResponse struct {
	StationID       int     `json:"station_id"`
	Name            string  `json:"name"`
	SystemID        int     `json:"system_id"`
	TypeID          int     `json:"type_id"`
	Race            string  `json:"race,omitempty"`
	Owner           int     `json:"owner,omitempty"`
	MaxDockableShip int     `json:"max_dockable_ship_volume,omitempty"`
	OfficeRentalCost int    `json:"office_rental_cost,omitempty"`
	ReprocessingEff float64 `json:"reprocessing_efficiency,omitempty"`
}

// UniverseClient implements universe-related ESI operations
type UniverseClient struct {
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

// NewUniverseClient creates a new universe client
func NewUniverseClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &UniverseClient{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetSystemInfo retrieves solar system information from ESI
func (c *UniverseClient) GetSystemInfo(ctx context.Context, systemID int) (*SystemInfoResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/universe/systems/%d/", systemID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/universe")
		ctx, span = tracer.Start(ctx, "universe.GetSystemInfo")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "universe_system"),
			attribute.Int("esi.system_id", systemID),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting system info from ESI", "system_id", systemID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var system SystemInfoResponse
		if err := json.Unmarshal(cachedData, &system); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached system data", "system_id", systemID)
			return &system, nil
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
		c.cacheManager.RefreshExpiry(cacheKey, resp.Header)
		
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			var system SystemInfoResponse
			if err := json.Unmarshal(cachedData, &system); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "System info not modified, using cached data", "system_id", systemID)
				return &system, nil
			}
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "system_id", systemID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for system %d", systemID)
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

	var system SystemInfoResponse
	if err := json.Unmarshal(body, &system); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.String("esi.system_name", system.Name),
			attribute.Float64("esi.security_status", system.SecurityStatus),
		)
		span.SetStatus(codes.Ok, "successfully retrieved system info")
	}

	return &system, nil
}

// GetStationInfo retrieves station information from ESI
func (c *UniverseClient) GetStationInfo(ctx context.Context, stationID int) (*StationInfoResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/universe/stations/%d/", stationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/universe")
		ctx, span = tracer.Start(ctx, "universe.GetStationInfo")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "universe_station"),
			attribute.Int("esi.station_id", stationID),
		)
	}

	slog.InfoContext(ctx, "Requesting station info from ESI", "station_id", stationID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var station StationInfoResponse
		if err := json.Unmarshal(cachedData, &station); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
			}
			return &station, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
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
		}
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		c.cacheManager.RefreshExpiry(cacheKey, resp.Header)
		
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			var station StationInfoResponse
			if err := json.Unmarshal(cachedData, &station); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "Station info not modified, using cached data", "station_id", stationID)
				return &station, nil
			}
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "station_id", stationID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for station %d", stationID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	c.cacheManager.Set(cacheKey, body, resp.Header)

	var station StationInfoResponse
	if err := json.Unmarshal(body, &station); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetStatus(codes.Ok, "successfully retrieved station info")
	}

	return &station, nil
}