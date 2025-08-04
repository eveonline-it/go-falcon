package evegateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go-falcon/pkg/config"
	"go-falcon/pkg/evegateway/alliance"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Client represents an EVE Online ESI client with all category clients
type Client struct {
	httpClient  *http.Client
	baseURL     string
	userAgent   string
	cacheManager CacheManager
	retryClient  RetryClient
	errorLimits *ESIErrorLimits
	limitsMutex sync.RWMutex
	
	// Category clients
	Status    StatusClient
	Character CharacterClient
	Universe  UniverseClient
	Alliance  AllianceClient
}

// ESIStatusResponse represents the EVE Online server status
type ESIStatusResponse struct {
	Players       int       `json:"players"`
	ServerVersion string    `json:"server_version"`
	StartTime     time.Time `json:"start_time"`
}

// StatusClient interface for status operations
type StatusClient interface {
	GetServerStatus(ctx context.Context) (*ESIStatusResponse, error)
}

// CharacterClient interface for character operations  
type CharacterClient interface {
	GetCharacterInfo(ctx context.Context, characterID int) (map[string]any, error)
	GetCharacterPortrait(ctx context.Context, characterID int) (map[string]any, error)
}

// UniverseClient interface for universe operations
type UniverseClient interface {
	GetSystemInfo(ctx context.Context, systemID int) (map[string]any, error)
	GetStationInfo(ctx context.Context, stationID int) (map[string]any, error)
}

// AllianceClient interface for alliance operations
type AllianceClient interface {
	GetAlliances(ctx context.Context) ([]int64, error)
	GetAllianceInfo(ctx context.Context, allianceID int64) (map[string]any, error)
	GetAllianceContacts(ctx context.Context, allianceID int64, token string) ([]map[string]any, error)
	GetAllianceContactLabels(ctx context.Context, allianceID int64, token string) ([]map[string]any, error)
	GetAllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error)
	GetAllianceIcons(ctx context.Context, allianceID int64) (map[string]any, error)
}

// NewClient creates a new EVE Online ESI client
func NewClient() *Client {
	var transport http.RoundTripper = http.DefaultTransport
	
	// Only add OpenTelemetry instrumentation if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		transport = otelhttp.NewTransport(http.DefaultTransport, 
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Host)
			}),
		)
	}
	
	// ESI-compliant User-Agent header with contact information
	userAgent := config.GetEnv("ESI_USER_AGENT", "go-falcon/1.0.0 contact@example.com")
	
	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	
	cacheManager := NewDefaultCacheManager()
	errorLimits := &ESIErrorLimits{}
	limitsMutex := &sync.RWMutex{}
	retryClient := NewDefaultRetryClient(httpClient, errorLimits, limitsMutex)
	
	// Create category clients using the shared infrastructure
	statusClient := &statusClientImpl{cacheManager, retryClient, httpClient, "https://esi.evetech.net", userAgent}
	characterClient := &characterClientImpl{cacheManager, retryClient, httpClient, "https://esi.evetech.net", userAgent}
	universeClient := &universeClientImpl{cacheManager, retryClient, httpClient, "https://esi.evetech.net", userAgent}
	allianceClientDirect := alliance.NewAllianceClient(httpClient, "https://esi.evetech.net", userAgent, cacheManager, retryClient)
	allianceClient := &allianceClientImpl{client: allianceClientDirect}
	
	return &Client{
		httpClient:   httpClient,
		baseURL:      "https://esi.evetech.net",
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
		errorLimits:  errorLimits,
		limitsMutex:  sync.RWMutex{},
		Status:       statusClient,
		Character:    characterClient,
		Universe:     universeClient,
		Alliance:     allianceClient,
	}
}

// HTTPClient returns the underlying HTTP client for advanced usage
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// GetServerStatus retrieves EVE Online server status from ESI with proper caching
func (c *Client) GetServerStatus(ctx context.Context) (*ESIStatusResponse, error) {
	var span trace.Span
	endpoint := "/status"
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	
	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate")
		ctx, span = tracer.Start(ctx, "evegate.GetServerStatus")
		defer span.End()
		
		span.SetAttributes(
			attribute.String("esi.endpoint", "status"),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("http.user_agent", c.userAgent),
			attribute.String("cache.key", cacheKey),
		)
	}
	
	slog.InfoContext(ctx, "Requesting server status from ESI")
	
	// Error limits are checked in the retry client
	
	// Check cache first
	cachedData, exists, err := c.cacheManager.Get(cacheKey)
	if err == nil && exists {
		var status ESIStatusResponse
		if err := json.Unmarshal(cachedData, &status); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached ESI status data")
			return &status, nil
		}
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", cacheKey, nil)
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
	
	// Use retry mechanism with exponential backoff
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3) // Max 3 retries
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
		slog.ErrorContext(ctx, "Failed to call ESI status endpoint", "error", err)
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()
	
	// Error limits are updated in the retry client
	
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
			slog.InfoContext(ctx, "ESI status not modified, refreshed cache expiry")
			
			var status ESIStatusResponse
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
	
	// Update cache with new data
	c.cacheManager.Set(cacheKey, body, resp.Header)
	
	var status ESIStatusResponse
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

// GetCharacterInfo retrieves character information from EVE ESI
func (c *Client) GetCharacterInfo(ctx context.Context, characterID int) (map[string]any, error) {
	slog.Info("Requesting character info from ESI", slog.Int("character_id", characterID))
	
	// TODO: Implement actual ESI call
	return map[string]any{"character_id": characterID, "name": "placeholder"}, nil
}

// GetCorporationInfo retrieves corporation information from EVE ESI
func (c *Client) GetCorporationInfo(ctx context.Context, corporationID int) (map[string]any, error) {
	slog.Info("Requesting corporation info from ESI", slog.Int("corporation_id", corporationID))
	
	// TODO: Implement actual ESI call
	return map[string]any{"corporation_id": corporationID, "name": "placeholder"}, nil
}

// GetAllianceInfo retrieves alliance information from EVE ESI
func (c *Client) GetAllianceInfo(ctx context.Context, allianceID int) (map[string]any, error) {
	slog.Info("Requesting alliance info from ESI", slog.Int("alliance_id", allianceID))
	
	// TODO: Implement actual ESI call
	return map[string]any{"alliance_id": allianceID, "name": "placeholder"}, nil
}

// RefreshToken refreshes an EVE SSO access token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (map[string]any, error) {
	slog.Info("Refreshing EVE SSO token")
	
	// TODO: Implement actual token refresh
	return map[string]any{"access_token": "placeholder", "expires_in": 1200}, nil
}

// Implementation structs for category clients
type statusClientImpl struct {
	cacheManager CacheManager
	retryClient  RetryClient
	httpClient   *http.Client
	baseURL      string
	userAgent    string
}

type characterClientImpl struct {
	cacheManager CacheManager
	retryClient  RetryClient
	httpClient   *http.Client
	baseURL      string
	userAgent    string
}

type universeClientImpl struct {
	cacheManager CacheManager
	retryClient  RetryClient
	httpClient   *http.Client
	baseURL      string
	userAgent    string
}

// StatusClient implementation
func (s *statusClientImpl) GetServerStatus(ctx context.Context) (*ESIStatusResponse, error) {
	// Delegate to the existing GetServerStatus method logic - for backward compatibility
	// In practice, users should use the status package directly
	return &ESIStatusResponse{}, fmt.Errorf("moved to status package - use status client directly")
}

// CharacterClient implementation
func (c *characterClientImpl) GetCharacterInfo(ctx context.Context, characterID int) (map[string]any, error) {
	slog.InfoContext(ctx, "Character info request delegated to character package", "character_id", characterID)
	return map[string]any{"character_id": characterID, "name": "use character package"}, nil
}

func (c *characterClientImpl) GetCharacterPortrait(ctx context.Context, characterID int) (map[string]any, error) {
	slog.InfoContext(ctx, "Character portrait request delegated to character package", "character_id", characterID)
	return map[string]any{"character_id": characterID, "portrait": "use character package"}, nil
}

// UniverseClient implementation
func (u *universeClientImpl) GetSystemInfo(ctx context.Context, systemID int) (map[string]any, error) {
	slog.InfoContext(ctx, "System info request delegated to universe package", "system_id", systemID)
	return map[string]any{"system_id": systemID, "name": "use universe package"}, nil
}

func (u *universeClientImpl) GetStationInfo(ctx context.Context, stationID int) (map[string]any, error) {
	slog.InfoContext(ctx, "Station info request delegated to universe package", "station_id", stationID)
	return map[string]any{"station_id": stationID, "name": "use universe package"}, nil
}

// Alliance client adapter
type allianceClientImpl struct {
	client alliance.Client
}

func (a *allianceClientImpl) GetAlliances(ctx context.Context) ([]int64, error) {
	return a.client.GetAlliances(ctx)
}

func (a *allianceClientImpl) GetAllianceInfo(ctx context.Context, allianceID int64) (map[string]any, error) {
	info, err := a.client.GetAllianceInfo(ctx, allianceID)
	if err != nil {
		return nil, err
	}
	// Convert to map[string]any for backward compatibility
	return map[string]any{
		"creator_corporation_id":  info.CreatorCorporationID,
		"creator_id":             info.CreatorID,
		"date_founded":           info.DateFounded,
		"executor_corporation_id": info.ExecutorCorporationID,
		"faction_id":             info.FactionID,
		"name":                   info.Name,
		"ticker":                 info.Ticker,
	}, nil
}

func (a *allianceClientImpl) GetAllianceContacts(ctx context.Context, allianceID int64, token string) ([]map[string]any, error) {
	contacts, err := a.client.GetAllianceContacts(ctx, allianceID, token)
	if err != nil {
		return nil, err
	}
	// Convert to []map[string]any for backward compatibility
	result := make([]map[string]any, len(contacts))
	for i, contact := range contacts {
		result[i] = map[string]any{
			"contact_id":   contact.ContactID,
			"contact_type": contact.ContactType,
			"label_ids":    contact.LabelIDs,
			"standing":     contact.Standing,
		}
	}
	return result, nil
}

func (a *allianceClientImpl) GetAllianceContactLabels(ctx context.Context, allianceID int64, token string) ([]map[string]any, error) {
	labels, err := a.client.GetAllianceContactLabels(ctx, allianceID, token)
	if err != nil {
		return nil, err
	}
	// Convert to []map[string]any for backward compatibility
	result := make([]map[string]any, len(labels))
	for i, label := range labels {
		result[i] = map[string]any{
			"label_id":   label.LabelID,
			"label_name": label.LabelName,
		}
	}
	return result, nil
}

func (a *allianceClientImpl) GetAllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error) {
	return a.client.GetAllianceCorporations(ctx, allianceID)
}

func (a *allianceClientImpl) GetAllianceIcons(ctx context.Context, allianceID int64) (map[string]any, error) {
	icons, err := a.client.GetAllianceIcons(ctx, allianceID)
	if err != nil {
		return nil, err
	}
	// Convert to map[string]any for backward compatibility
	result := map[string]any{}
	if icons.Px64x64 != nil {
		result["px64x64"] = *icons.Px64x64
	}
	if icons.Px128x128 != nil {
		result["px128x128"] = *icons.Px128x128
	}
	return result, nil
}

