package corporation

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

// Corporation information results with cache information
type CorporationInfoResult struct {
	Data  *CorporationInfoResponse `json:"data"`
	Cache CacheInfo                `json:"cache"`
}

type CorporationMembersResult struct {
	Data  []CorporationMember `json:"data"`
	Cache CacheInfo           `json:"cache"`
}

type CorporationStructuresResult struct {
	Data  []CorporationStructure `json:"data"`
	Cache CacheInfo              `json:"cache"`
}

type CorporationAllianceHistoryResult struct {
	Data  []CorporationAllianceHistory `json:"data"`
	Cache CacheInfo                    `json:"cache"`
}

type CorporationIconsResult struct {
	Data  *CorporationIcons `json:"data"`
	Cache CacheInfo         `json:"cache"`
}

type CorporationStandingsResult struct {
	Data  []CorporationStanding `json:"data"`
	Cache CacheInfo             `json:"cache"`
}

type CorporationWalletResult struct {
	Data  []CorporationWallet `json:"data"`
	Cache CacheInfo           `json:"cache"`
}

type CorporationMemberTrackingResult struct {
	Data  []CorporationMemberTracking `json:"data"`
	Cache CacheInfo                   `json:"cache"`
}

type CorporationRolesResult struct {
	Data  []CorporationMemberRoles `json:"data"`
	Cache CacheInfo                `json:"cache"`
}

// Client interface for corporation-related ESI operations
type Client interface {
	// Basic Corporation Information
	GetCorporationInfo(ctx context.Context, corporationID int) (*CorporationInfoResponse, error)
	GetCorporationInfoWithCache(ctx context.Context, corporationID int) (*CorporationInfoResult, error)
	GetCorporationIcons(ctx context.Context, corporationID int) (*CorporationIcons, error)
	GetCorporationIconsWithCache(ctx context.Context, corporationID int) (*CorporationIconsResult, error)
	GetCorporationAllianceHistory(ctx context.Context, corporationID int) ([]CorporationAllianceHistory, error)
	GetCorporationAllianceHistoryWithCache(ctx context.Context, corporationID int) (*CorporationAllianceHistoryResult, error)

	// Corporation Members (requires authentication)
	GetCorporationMembers(ctx context.Context, corporationID int, token string) ([]CorporationMember, error)
	GetCorporationMembersWithCache(ctx context.Context, corporationID int, token string) (*CorporationMembersResult, error)
	GetCorporationMemberTracking(ctx context.Context, corporationID int, token string) ([]CorporationMemberTracking, error)
	GetCorporationMemberTrackingWithCache(ctx context.Context, corporationID int, token string) (*CorporationMemberTrackingResult, error)
	GetCorporationMemberRoles(ctx context.Context, corporationID int, token string) ([]CorporationMemberRoles, error)
	GetCorporationMemberRolesWithCache(ctx context.Context, corporationID int, token string) (*CorporationRolesResult, error)

	// Corporation Structures and Assets (requires authentication)
	GetCorporationStructures(ctx context.Context, corporationID int, token string) ([]CorporationStructure, error)
	GetCorporationStructuresWithCache(ctx context.Context, corporationID int, token string) (*CorporationStructuresResult, error)

	// Corporation Relationships
	GetCorporationStandings(ctx context.Context, corporationID int, token string) ([]CorporationStanding, error)
	GetCorporationStandingsWithCache(ctx context.Context, corporationID int, token string) (*CorporationStandingsResult, error)

	// Corporation Finances (requires authentication)
	GetCorporationWallets(ctx context.Context, corporationID int, token string) ([]CorporationWallet, error)
	GetCorporationWalletsWithCache(ctx context.Context, corporationID int, token string) (*CorporationWalletResult, error)
}

// CorporationInfoResponse represents corporation public information
type CorporationInfoResponse struct {
	CorporationID  int       `json:"corporation_id"`
	Name           string    `json:"name"`
	Ticker         string    `json:"ticker"`
	Description    string    `json:"description"`
	URL            string    `json:"url,omitempty"`
	AllianceID     int       `json:"alliance_id,omitempty"`
	CEOCharacterID int       `json:"ceo_id"`
	CreatorID      int       `json:"creator_id"`
	DateFounded    time.Time `json:"date_founded"`
	FactionID      int       `json:"faction_id,omitempty"`
	HomeStationID  int       `json:"home_station_id,omitempty"`
	MemberCount    int       `json:"member_count"`
	Shares         int64     `json:"shares,omitempty"`
	TaxRate        float64   `json:"tax_rate"`
	WarEligible    bool      `json:"war_eligible,omitempty"`
}

// CorporationMember represents a corporation member
type CorporationMember struct {
	CharacterID int `json:"character_id"`
}

// CorporationStructure represents a corporation structure
type CorporationStructure struct {
	StructureID      int64     `json:"structure_id"`
	TypeID           int       `json:"type_id"`
	SystemID         int       `json:"system_id"`
	ProfileID        int       `json:"profile_id"`
	FuelExpires      time.Time `json:"fuel_expires,omitempty"`
	StateTimerStart  time.Time `json:"state_timer_start,omitempty"`
	StateTimerEnd    time.Time `json:"state_timer_end,omitempty"`
	UnanchorsAt      time.Time `json:"unanchors_at,omitempty"`
	State            string    `json:"state"`
	ReinforceHour    int       `json:"reinforce_hour,omitempty"`
	ReinforceWeekday int       `json:"reinforce_weekday,omitempty"`
	CorporationID    int       `json:"corporation_id"`
	Services         []Service `json:"services,omitempty"`
}

// Service represents a structure service
type Service struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// CorporationAllianceHistory represents corporation alliance history entry
type CorporationAllianceHistory struct {
	AllianceID int       `json:"alliance_id,omitempty"`
	IsDeleted  bool      `json:"is_deleted,omitempty"`
	RecordID   int       `json:"record_id"`
	StartDate  time.Time `json:"start_date"`
}

// CorporationIcons represents corporation icon URLs
type CorporationIcons struct {
	Px64x64   string `json:"px64x64"`
	Px128x128 string `json:"px128x128"`
	Px256x256 string `json:"px256x256"`
}

// CorporationStanding represents a corporation standing
type CorporationStanding struct {
	FromID   int     `json:"from_id"`
	FromType string  `json:"from_type"`
	Standing float64 `json:"standing"`
}

// CorporationWallet represents a corporation wallet division
type CorporationWallet struct {
	Division int     `json:"division"`
	Balance  float64 `json:"balance"`
}

// CorporationMemberTracking represents member tracking information
type CorporationMemberTracking struct {
	BaseID      int       `json:"base_id,omitempty"`
	CharacterID int       `json:"character_id"`
	LocationID  int64     `json:"location_id,omitempty"`
	LogoffDate  time.Time `json:"logoff_date,omitempty"`
	LogonDate   time.Time `json:"logon_date,omitempty"`
	ShipTypeID  int       `json:"ship_type_id,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty"`
}

// CorporationMemberRoles represents member roles information
type CorporationMemberRoles struct {
	CharacterID           int      `json:"character_id"`
	GrantedRoles          []string `json:"grantable_roles,omitempty"`
	GrantableRoles        []string `json:"grantable_roles_at_base,omitempty"`
	GrantableRolesAtHQ    []string `json:"grantable_roles_at_hq,omitempty"`
	GrantableRolesAtOther []string `json:"grantable_roles_at_other,omitempty"`
	Roles                 []string `json:"roles,omitempty"`
	RolesAtBase           []string `json:"roles_at_base,omitempty"`
	RolesAtHQ             []string `json:"roles_at_hq,omitempty"`
	RolesAtOther          []string `json:"roles_at_other,omitempty"`
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

// CorporationClient implements corporation-related ESI operations
type CorporationClient struct {
	httpClient   *http.Client
	baseURL      string
	userAgent    string
	cacheManager CacheManager
	retryClient  RetryClient
}

// NewCorporationClient creates a new corporation client
func NewCorporationClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &CorporationClient{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetCorporationInfo retrieves corporation public information from ESI
func (c *CorporationClient) GetCorporationInfo(ctx context.Context, corporationID int) (*CorporationInfoResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/corporations/%d/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegateway/corporation")
		ctx, span = tracer.Start(ctx, "corporation.GetCorporationInfo")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "corporation"),
			attribute.Int("esi.corporation_id", corporationID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting corporation info from ESI", "corporation_id", corporationID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var corporation CorporationInfoResponse
		if err := json.Unmarshal(cachedData, &corporation); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached corporation data", "corporation_id", corporationID)
			return &corporation, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create corporation info request", "error", err)
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
		slog.ErrorContext(ctx, "Failed to call ESI corporation endpoint", "error", err)
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
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			slog.InfoContext(ctx, "Corporation info not modified, using cached data")

			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var corporation CorporationInfoResponse
			if err := json.Unmarshal(cachedData, &corporation); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return &corporation, nil
		} else {
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "corporation_id", corporationID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for corporation %d", corporationID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI corporation endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read corporation info response", "error", err)
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

	var corporation CorporationInfoResponse
	if err := json.Unmarshal(body, &corporation); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse corporation info response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.String("esi.corporation_name", corporation.Name),
			attribute.String("esi.corporation_ticker", corporation.Ticker),
			attribute.Int("esi.member_count", corporation.MemberCount),
		)
		span.SetStatus(codes.Ok, "successfully retrieved corporation info")
	}

	slog.InfoContext(ctx, "Successfully retrieved corporation info",
		slog.Int("corporation_id", corporation.CorporationID),
		slog.String("name", corporation.Name),
		slog.String("ticker", corporation.Ticker),
		slog.Int("member_count", corporation.MemberCount))

	return &corporation, nil
}

// GetCorporationInfoWithCache retrieves corporation public information from ESI with cache info
func (c *CorporationClient) GetCorporationInfoWithCache(ctx context.Context, corporationID int) (*CorporationInfoResult, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/corporations/%d/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegateway/corporation")
		ctx, span = tracer.Start(ctx, "corporation.GetCorporationInfoWithCache")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "corporation"),
			attribute.Int("esi.corporation_id", corporationID),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting corporation info from ESI with cache info", "corporation_id", corporationID)

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var corporation CorporationInfoResponse
		if err := json.Unmarshal(cachedData, &corporation); err == nil {
			cached = true
			cacheExpiry = expiry
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached corporation data", "corporation_id", corporationID)
			return &CorporationInfoResult{
				Data:  &corporation,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationInfo(ctx, corporationID)
	if err != nil {
		return nil, err
	}

	return &CorporationInfoResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationIcons retrieves corporation icon URLs from ESI
func (c *CorporationClient) GetCorporationIcons(ctx context.Context, corporationID int) (*CorporationIcons, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/corporations/%d/icons/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegateway/corporation")
		ctx, span = tracer.Start(ctx, "corporation.GetCorporationIcons")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "corporation_icons"),
			attribute.Int("esi.corporation_id", corporationID),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting corporation icons from ESI", "corporation_id", corporationID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var icons CorporationIcons
		if err := json.Unmarshal(cachedData, &icons); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached corporation icons data", "corporation_id", corporationID)
			return &icons, nil
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
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			var icons CorporationIcons
			if err := json.Unmarshal(cachedData, &icons); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "Corporation icons not modified, using cached data")
				c.cacheManager.RefreshExpiry(cacheKey, resp.Header)
				return &icons, nil
			}
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

	var icons CorporationIcons
	if err := json.Unmarshal(body, &icons); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetStatus(codes.Ok, "successfully retrieved corporation icons")
	}

	return &icons, nil
}

// GetCorporationIconsWithCache retrieves corporation icon URLs from ESI with cache info
func (c *CorporationClient) GetCorporationIconsWithCache(ctx context.Context, corporationID int) (*CorporationIconsResult, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/corporations/%d/icons/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Only create spans if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegateway/corporation")
		ctx, span = tracer.Start(ctx, "corporation.GetCorporationIconsWithCache")
		defer span.End()
	}

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var icons CorporationIcons
		if err := json.Unmarshal(cachedData, &icons); err == nil {
			cached = true
			cacheExpiry = expiry
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			return &CorporationIconsResult{
				Data:  &icons,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationIcons(ctx, corporationID)
	if err != nil {
		return nil, err
	}

	return &CorporationIconsResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// Helper method to make authenticated requests
func (c *CorporationClient) makeAuthenticatedRequest(ctx context.Context, endpoint, token string, cacheKey string) ([]byte, error) {
	var span trace.Span

	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		tracer := otel.Tracer("go-falcon/evegateway/corporation")
		ctx, span = tracer.Start(ctx, "corporation.makeAuthenticatedRequest")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", endpoint),
			attribute.String("cache.key", cacheKey),
		)
	}

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		if span != nil {
			span.SetAttributes(attribute.Bool("cache.hit", true))
			span.SetStatus(codes.Ok, "cache hit")
		}
		return cachedData, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
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

	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to call ESI")
		}
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
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)
			return cachedData, nil
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
			span.SetStatus(codes.Error, "failed to read response")
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.response_size", len(body)),
			attribute.Bool("cache.hit", false),
		)
		span.SetStatus(codes.Ok, "successfully retrieved data")
	}

	// Update cache
	c.cacheManager.Set(cacheKey, body, resp.Header)

	return body, nil
}

// GetCorporationAllianceHistory retrieves corporation alliance history from ESI
func (c *CorporationClient) GetCorporationAllianceHistory(ctx context.Context, corporationID int) ([]CorporationAllianceHistory, error) {
	endpoint := fmt.Sprintf("/corporations/%d/alliancehistory/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var history []CorporationAllianceHistory
		if err := json.Unmarshal(cachedData, &history); err == nil {
			return history, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers for public endpoint
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	// Add conditional headers if we have cached data
	c.cacheManager.SetConditionalHeaders(req, cacheKey)

	// Use retry mechanism
	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			// Refresh the expiry since ESI confirmed data is still valid
			c.cacheManager.RefreshExpiry(cacheKey, resp.Header)

			var history []CorporationAllianceHistory
			if err := json.Unmarshal(cachedData, &history); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return history, nil
		} else {
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for corporation %d", corporationID)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Update cache
	c.cacheManager.Set(cacheKey, body, resp.Header)

	var history []CorporationAllianceHistory
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return history, nil
}

// GetCorporationAllianceHistoryWithCache retrieves corporation alliance history from ESI with cache info
func (c *CorporationClient) GetCorporationAllianceHistoryWithCache(ctx context.Context, corporationID int) (*CorporationAllianceHistoryResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/alliancehistory/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var history []CorporationAllianceHistory
		if err := json.Unmarshal(cachedData, &history); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationAllianceHistoryResult{
				Data:  history,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationAllianceHistory(ctx, corporationID)
	if err != nil {
		return nil, err
	}

	return &CorporationAllianceHistoryResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationMembers retrieves corporation members from ESI (requires authentication)
func (c *CorporationClient) GetCorporationMembers(ctx context.Context, corporationID int, token string) ([]CorporationMember, error) {
	endpoint := fmt.Sprintf("/corporations/%d/members/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	body, err := c.makeAuthenticatedRequest(ctx, endpoint, token, cacheKey)
	if err != nil {
		return nil, err
	}

	// ESI returns a simple array of character IDs (integers)
	var characterIDs []int
	if err := json.Unmarshal(body, &characterIDs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to CorporationMember structs
	members := make([]CorporationMember, len(characterIDs))
	for i, characterID := range characterIDs {
		members[i] = CorporationMember{
			CharacterID: characterID,
		}
	}

	return members, nil
}

// GetCorporationMembersWithCache retrieves corporation members from ESI with cache info
func (c *CorporationClient) GetCorporationMembersWithCache(ctx context.Context, corporationID int, token string) (*CorporationMembersResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/members/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var members []CorporationMember
		if err := json.Unmarshal(cachedData, &members); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationMembersResult{
				Data:  members,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationMembers(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	return &CorporationMembersResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationMemberTracking retrieves corporation member tracking from ESI (requires authentication)
func (c *CorporationClient) GetCorporationMemberTracking(ctx context.Context, corporationID int, token string) ([]CorporationMemberTracking, error) {
	endpoint := fmt.Sprintf("/corporations/%d/membertracking/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	body, err := c.makeAuthenticatedRequest(ctx, endpoint, token, cacheKey)
	if err != nil {
		return nil, err
	}

	var tracking []CorporationMemberTracking
	if err := json.Unmarshal(body, &tracking); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return tracking, nil
}

// GetCorporationMemberTrackingWithCache retrieves corporation member tracking from ESI with cache info
func (c *CorporationClient) GetCorporationMemberTrackingWithCache(ctx context.Context, corporationID int, token string) (*CorporationMemberTrackingResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/membertracking/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var tracking []CorporationMemberTracking
		if err := json.Unmarshal(cachedData, &tracking); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationMemberTrackingResult{
				Data:  tracking,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationMemberTracking(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	return &CorporationMemberTrackingResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationMemberRoles retrieves corporation member roles from ESI (requires authentication)
func (c *CorporationClient) GetCorporationMemberRoles(ctx context.Context, corporationID int, token string) ([]CorporationMemberRoles, error) {
	endpoint := fmt.Sprintf("/corporations/%d/roles/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	body, err := c.makeAuthenticatedRequest(ctx, endpoint, token, cacheKey)
	if err != nil {
		return nil, err
	}

	var roles []CorporationMemberRoles
	if err := json.Unmarshal(body, &roles); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return roles, nil
}

// GetCorporationMemberRolesWithCache retrieves corporation member roles from ESI with cache info
func (c *CorporationClient) GetCorporationMemberRolesWithCache(ctx context.Context, corporationID int, token string) (*CorporationRolesResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/roles/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var roles []CorporationMemberRoles
		if err := json.Unmarshal(cachedData, &roles); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationRolesResult{
				Data:  roles,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationMemberRoles(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	return &CorporationRolesResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationStructures retrieves corporation structures from ESI (requires authentication)
func (c *CorporationClient) GetCorporationStructures(ctx context.Context, corporationID int, token string) ([]CorporationStructure, error) {
	endpoint := fmt.Sprintf("/corporations/%d/structures/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	body, err := c.makeAuthenticatedRequest(ctx, endpoint, token, cacheKey)
	if err != nil {
		return nil, err
	}

	var structures []CorporationStructure
	if err := json.Unmarshal(body, &structures); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return structures, nil
}

// GetCorporationStructuresWithCache retrieves corporation structures from ESI with cache info
func (c *CorporationClient) GetCorporationStructuresWithCache(ctx context.Context, corporationID int, token string) (*CorporationStructuresResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/structures/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var structures []CorporationStructure
		if err := json.Unmarshal(cachedData, &structures); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationStructuresResult{
				Data:  structures,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationStructures(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	return &CorporationStructuresResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationStandings retrieves corporation standings from ESI (requires authentication)
func (c *CorporationClient) GetCorporationStandings(ctx context.Context, corporationID int, token string) ([]CorporationStanding, error) {
	endpoint := fmt.Sprintf("/corporations/%d/standings/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	body, err := c.makeAuthenticatedRequest(ctx, endpoint, token, cacheKey)
	if err != nil {
		return nil, err
	}

	var standings []CorporationStanding
	if err := json.Unmarshal(body, &standings); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return standings, nil
}

// GetCorporationStandingsWithCache retrieves corporation standings from ESI with cache info
func (c *CorporationClient) GetCorporationStandingsWithCache(ctx context.Context, corporationID int, token string) (*CorporationStandingsResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/standings/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var standings []CorporationStanding
		if err := json.Unmarshal(cachedData, &standings); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationStandingsResult{
				Data:  standings,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationStandings(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	return &CorporationStandingsResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}

// GetCorporationWallets retrieves corporation wallet balances from ESI (requires authentication)
func (c *CorporationClient) GetCorporationWallets(ctx context.Context, corporationID int, token string) ([]CorporationWallet, error) {
	endpoint := fmt.Sprintf("/corporations/%d/wallets/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	body, err := c.makeAuthenticatedRequest(ctx, endpoint, token, cacheKey)
	if err != nil {
		return nil, err
	}

	var wallets []CorporationWallet
	if err := json.Unmarshal(body, &wallets); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return wallets, nil
}

// GetCorporationWalletsWithCache retrieves corporation wallet balances from ESI with cache info
func (c *CorporationClient) GetCorporationWalletsWithCache(ctx context.Context, corporationID int, token string) (*CorporationWalletResult, error) {
	endpoint := fmt.Sprintf("/corporations/%d/wallets/", corporationID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	cached := false
	var cacheExpiry *time.Time

	// Check cache first
	if cachedData, found, expiry, err := c.cacheManager.GetWithExpiry(cacheKey); err == nil && found {
		var wallets []CorporationWallet
		if err := json.Unmarshal(cachedData, &wallets); err == nil {
			cached = true
			cacheExpiry = expiry
			return &CorporationWalletResult{
				Data:  wallets,
				Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
			}, nil
		}
	}

	// Get fresh data
	data, err := c.GetCorporationWallets(ctx, corporationID, token)
	if err != nil {
		return nil, err
	}

	return &CorporationWalletResult{
		Data:  data,
		Cache: CacheInfo{Cached: cached, ExpiresAt: cacheExpiry},
	}, nil
}
