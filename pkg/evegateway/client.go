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
	"go-falcon/pkg/evegateway/character"
	"go-falcon/pkg/evegateway/corporation"

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
	Status      StatusClient
	Character   CharacterClient
	Universe    UniverseClient
	Alliance    AllianceClient
	Corporation CorporationClient
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
	GetCharactersAffiliation(ctx context.Context, characterIDs []int) ([]map[string]interface{}, error)
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

// CorporationClient interface for corporation operations
type CorporationClient interface {
	// Basic Corporation Information
	GetCorporationInfo(ctx context.Context, corporationID int) (*corporation.CorporationInfoResponse, error)
	GetCorporationInfoWithCache(ctx context.Context, corporationID int) (*corporation.CorporationInfoResult, error)
	GetCorporationIcons(ctx context.Context, corporationID int) (*corporation.CorporationIcons, error)
	GetCorporationIconsWithCache(ctx context.Context, corporationID int) (*corporation.CorporationIconsResult, error)
	GetCorporationAllianceHistory(ctx context.Context, corporationID int) ([]corporation.CorporationAllianceHistory, error)
	GetCorporationAllianceHistoryWithCache(ctx context.Context, corporationID int) (*corporation.CorporationAllianceHistoryResult, error)

	// Corporation Members (requires authentication)
	GetCorporationMembers(ctx context.Context, corporationID int, token string) ([]corporation.CorporationMember, error)
	GetCorporationMembersWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationMembersResult, error)
	GetCorporationMemberTracking(ctx context.Context, corporationID int, token string) ([]corporation.CorporationMemberTracking, error)
	GetCorporationMemberTrackingWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationMemberTrackingResult, error)
	GetCorporationMemberRoles(ctx context.Context, corporationID int, token string) ([]corporation.CorporationMemberRoles, error)
	GetCorporationMemberRolesWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationRolesResult, error)

	// Corporation Structures and Assets (requires authentication)
	GetCorporationStructures(ctx context.Context, corporationID int, token string) ([]corporation.CorporationStructure, error)
	GetCorporationStructuresWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationStructuresResult, error)

	// Corporation Relationships
	GetCorporationStandings(ctx context.Context, corporationID int, token string) ([]corporation.CorporationStanding, error)
	GetCorporationStandingsWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationStandingsResult, error)

	// Corporation Finances (requires authentication)
	GetCorporationWallets(ctx context.Context, corporationID int, token string) ([]corporation.CorporationWallet, error)
	GetCorporationWalletsWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationWalletResult, error)
}

// NewClient creates a new EVE Online ESI client
func NewClient() *Client {
	var transport http.RoundTripper = http.DefaultTransport
	
	// Only add OpenTelemetry instrumentation if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
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
	characterClientDirect := character.NewCharacterClient(httpClient, "https://esi.evetech.net", userAgent, cacheManager, retryClient)
	characterClient := &characterClientImpl{client: characterClientDirect}
	universeClient := &universeClientImpl{cacheManager, retryClient, httpClient, "https://esi.evetech.net", userAgent}
	allianceClientDirect := alliance.NewAllianceClient(httpClient, "https://esi.evetech.net", userAgent, cacheManager, retryClient)
	allianceClient := &allianceClientImpl{client: allianceClientDirect}
	corporationClientDirect := corporation.NewCorporationClient(httpClient, "https://esi.evetech.net", userAgent, cacheManager, retryClient)
	corporationClient := &corporationClientImpl{client: corporationClientDirect}
	
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
		Corporation:  corporationClient,
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
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
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
	
	// Delegate to character client
	return c.Character.GetCharacterInfo(ctx, characterID)
}

// GetCorporationInfo retrieves corporation information from EVE ESI
func (c *Client) GetCorporationInfo(ctx context.Context, corporationID int) (map[string]any, error) {
	slog.Info("Requesting corporation info from ESI", slog.Int("corporation_id", corporationID))
	
	// Get corporation info from typed client
	corpInfo, err := c.Corporation.GetCorporationInfo(ctx, corporationID)
	if err != nil {
		return nil, err
	}
	
	// Convert structured response to map for backward compatibility
	result := map[string]any{
		"corporation_id":   corpInfo.CorporationID,
		"name":            corpInfo.Name,
		"ticker":          corpInfo.Ticker,
		"description":     corpInfo.Description,
		"ceo_id":          corpInfo.CEOCharacterID,
		"creator_id":      corpInfo.CreatorID,
		"date_founded":    corpInfo.DateFounded.Format("2006-01-02T15:04:05Z"),
		"member_count":    corpInfo.MemberCount,
		"tax_rate":        corpInfo.TaxRate,
	}
	
	// Add optional fields if they exist (check for non-zero values due to omitempty)
	if corpInfo.URL != "" {
		result["url"] = corpInfo.URL
	}
	if corpInfo.AllianceID != 0 {
		result["alliance_id"] = corpInfo.AllianceID
	}
	if corpInfo.FactionID != 0 {
		result["faction_id"] = corpInfo.FactionID
	}
	if corpInfo.HomeStationID != 0 {
		result["home_station_id"] = corpInfo.HomeStationID
	}
	if corpInfo.Shares != 0 {
		result["shares"] = corpInfo.Shares
	}
	// WarEligible is a bool, so we include it always since false is valid
	result["war_eligible"] = corpInfo.WarEligible
	
	return result, nil
}

// GetAllianceInfo retrieves alliance information from EVE ESI
func (c *Client) GetAllianceInfo(ctx context.Context, allianceID int) (map[string]any, error) {
	slog.Info("Requesting alliance info from ESI", slog.Int("alliance_id", allianceID))
	
	// Delegate to the alliance client
	return c.Alliance.GetAllianceInfo(ctx, int64(allianceID))
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
	client character.Client
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
	charInfo, err := c.client.GetCharacterInfo(ctx, characterID)
	if err != nil {
		return nil, err
	}
	
	// Convert structured response to map for backward compatibility
	result := map[string]any{
		"character_id":    charInfo.CharacterID,
		"name":           charInfo.Name,
		"description":    charInfo.Description,
		"corporation_id": charInfo.CorporationID,
		"birthday":       charInfo.Birthday,
		"gender":         charInfo.Gender,
		"race_id":        charInfo.RaceID,
		"bloodline_id":   charInfo.BloodlineID,
		"security_status": charInfo.SecurityStatus,
	}
	
	// Add optional fields if they exist
	if charInfo.AllianceID != 0 {
		result["alliance_id"] = charInfo.AllianceID
	}
	if charInfo.AncestryID != 0 {
		result["ancestry_id"] = charInfo.AncestryID
	}
	if charInfo.FactionID != 0 {
		result["faction_id"] = charInfo.FactionID
	}
	
	return result, nil
}

func (c *characterClientImpl) GetCharacterPortrait(ctx context.Context, characterID int) (map[string]any, error) {
	portrait, err := c.client.GetCharacterPortrait(ctx, characterID)
	if err != nil {
		return nil, err
	}
	
	// Convert structured response to map for backward compatibility
	return map[string]any{
		"px64x64":   portrait.Px64x64,
		"px128x128": portrait.Px128x128,
		"px256x256": portrait.Px256x256,
		"px512x512": portrait.Px512x512,
	}, nil
}

func (c *characterClientImpl) GetCharactersAffiliation(ctx context.Context, characterIDs []int) ([]map[string]interface{}, error) {
	affiliations, err := c.client.GetCharactersAffiliation(ctx, characterIDs)
	if err != nil {
		return nil, err
	}
	
	// Convert structured response to map for backward compatibility
	result := make([]map[string]interface{}, len(affiliations))
	for i, aff := range affiliations {
		result[i] = map[string]interface{}{
			"character_id":   aff.CharacterID,
			"corporation_id": aff.CorporationID,
		}
		if aff.AllianceID != 0 {
			result[i]["alliance_id"] = aff.AllianceID
		}
		if aff.FactionID != 0 {
			result[i]["faction_id"] = aff.FactionID
		}
	}
	
	return result, nil
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

// corporationClientImpl wraps the corporation.Client to match the interface
type corporationClientImpl struct {
	client corporation.Client
}

func (c *corporationClientImpl) GetCorporationInfo(ctx context.Context, corporationID int) (*corporation.CorporationInfoResponse, error) {
	return c.client.GetCorporationInfo(ctx, corporationID)
}

func (c *corporationClientImpl) GetCorporationInfoWithCache(ctx context.Context, corporationID int) (*corporation.CorporationInfoResult, error) {
	return c.client.GetCorporationInfoWithCache(ctx, corporationID)
}

func (c *corporationClientImpl) GetCorporationIcons(ctx context.Context, corporationID int) (*corporation.CorporationIcons, error) {
	return c.client.GetCorporationIcons(ctx, corporationID)
}

func (c *corporationClientImpl) GetCorporationIconsWithCache(ctx context.Context, corporationID int) (*corporation.CorporationIconsResult, error) {
	return c.client.GetCorporationIconsWithCache(ctx, corporationID)
}

func (c *corporationClientImpl) GetCorporationAllianceHistory(ctx context.Context, corporationID int) ([]corporation.CorporationAllianceHistory, error) {
	return c.client.GetCorporationAllianceHistory(ctx, corporationID)
}

func (c *corporationClientImpl) GetCorporationAllianceHistoryWithCache(ctx context.Context, corporationID int) (*corporation.CorporationAllianceHistoryResult, error) {
	return c.client.GetCorporationAllianceHistoryWithCache(ctx, corporationID)
}

func (c *corporationClientImpl) GetCorporationMembers(ctx context.Context, corporationID int, token string) ([]corporation.CorporationMember, error) {
	return c.client.GetCorporationMembers(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationMembersWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationMembersResult, error) {
	return c.client.GetCorporationMembersWithCache(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationMemberTracking(ctx context.Context, corporationID int, token string) ([]corporation.CorporationMemberTracking, error) {
	return c.client.GetCorporationMemberTracking(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationMemberTrackingWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationMemberTrackingResult, error) {
	return c.client.GetCorporationMemberTrackingWithCache(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationMemberRoles(ctx context.Context, corporationID int, token string) ([]corporation.CorporationMemberRoles, error) {
	return c.client.GetCorporationMemberRoles(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationMemberRolesWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationRolesResult, error) {
	return c.client.GetCorporationMemberRolesWithCache(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationStructures(ctx context.Context, corporationID int, token string) ([]corporation.CorporationStructure, error) {
	return c.client.GetCorporationStructures(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationStructuresWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationStructuresResult, error) {
	return c.client.GetCorporationStructuresWithCache(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationStandings(ctx context.Context, corporationID int, token string) ([]corporation.CorporationStanding, error) {
	return c.client.GetCorporationStandings(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationStandingsWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationStandingsResult, error) {
	return c.client.GetCorporationStandingsWithCache(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationWallets(ctx context.Context, corporationID int, token string) ([]corporation.CorporationWallet, error) {
	return c.client.GetCorporationWallets(ctx, corporationID, token)
}

func (c *corporationClientImpl) GetCorporationWalletsWithCache(ctx context.Context, corporationID int, token string) (*corporation.CorporationWalletResult, error) {
	return c.client.GetCorporationWalletsWithCache(ctx, corporationID, token)
}

