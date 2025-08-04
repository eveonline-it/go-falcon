package alliance

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

// Client interface for alliance-related ESI operations
type Client interface {
	GetAlliances(ctx context.Context) ([]int64, error)
	GetAllianceInfo(ctx context.Context, allianceID int64) (*AllianceInfoResponse, error)
	GetAllianceContacts(ctx context.Context, allianceID int64, token string) ([]AllianceContact, error)
	GetAllianceContactLabels(ctx context.Context, allianceID int64, token string) ([]AllianceContactLabel, error)
	GetAllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error)
	GetAllianceIcons(ctx context.Context, allianceID int64) (*AllianceIconsResponse, error)
}

// AllianceInfoResponse represents alliance public information
type AllianceInfoResponse struct {
	CreatorCorporationID  int64     `json:"creator_corporation_id"`
	CreatorID             int64     `json:"creator_id"`
	DateFounded           time.Time `json:"date_founded"`
	ExecutorCorporationID *int64    `json:"executor_corporation_id,omitempty"`
	FactionID             *int64    `json:"faction_id,omitempty"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
}

// AllianceContact represents an alliance contact
type AllianceContact struct {
	ContactID   int64     `json:"contact_id"`
	ContactType string    `json:"contact_type"` // character, corporation, alliance, faction
	LabelIDs    []int64   `json:"label_ids,omitempty"`
	Standing    float64   `json:"standing"`
}

// AllianceContactLabel represents an alliance contact label
type AllianceContactLabel struct {
	LabelID   int64  `json:"label_id"`
	LabelName string `json:"label_name"`
}

// AllianceIconsResponse represents alliance icon URLs
type AllianceIconsResponse struct {
	Px64x64   *string `json:"px64x64,omitempty"`
	Px128x128 *string `json:"px128x128,omitempty"`
}

// AllianceClient implements alliance-related ESI operations
type AllianceClient struct {
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

// NewAllianceClient creates a new alliance client
func NewAllianceClient(httpClient *http.Client, baseURL, userAgent string, cacheManager CacheManager, retryClient RetryClient) Client {
	return &AllianceClient{
		httpClient:   httpClient,
		baseURL:      baseURL,
		userAgent:    userAgent,
		cacheManager: cacheManager,
		retryClient:  retryClient,
	}
}

// GetAlliances retrieves list of all active player alliances
func (c *AllianceClient) GetAlliances(ctx context.Context) ([]int64, error) {
	var span trace.Span
	endpoint := "/alliances"
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/alliance")
		ctx, span = tracer.Start(ctx, "alliance.GetAlliances")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "alliances"),
			attribute.String("esi.base_url", c.baseURL),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting alliances list from ESI")

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var alliances []int64
		if err := json.Unmarshal(cachedData, &alliances); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached alliances data")
			return alliances, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create request")
		}
		slog.ErrorContext(ctx, "Failed to create alliances request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
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
		slog.ErrorContext(ctx, "Failed to call ESI alliances endpoint", "error", err)
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
		c.cacheManager.RefreshExpiry(cacheKey, resp.Header)
		
		if cachedData, found, err := c.cacheManager.GetForNotModified(cacheKey); err == nil && found {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit - not modified")
			}
			slog.InfoContext(ctx, "Alliances not modified, refreshed cache expiry")

			var alliances []int64
			if err := json.Unmarshal(cachedData, &alliances); err != nil {
				return nil, fmt.Errorf("failed to parse cached response: %w", err)
			}
			return alliances, nil
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available")
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for alliances")
		}
	}

	if resp.StatusCode != http.StatusOK {
		if span != nil {
			span.SetStatus(codes.Error, "ESI returned error status")
		}
		slog.ErrorContext(ctx, "ESI alliances endpoint returned error", "status_code", resp.StatusCode)
		return nil, fmt.Errorf("ESI returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to read response")
		}
		slog.ErrorContext(ctx, "Failed to read alliances response", "error", err)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("http.response_size", len(body)),
			attribute.Bool("cache.hit", false),
		)
	}

	c.cacheManager.Set(cacheKey, body, resp.Header)

	var alliances []int64
	if err := json.Unmarshal(body, &alliances); err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to parse response")
		}
		slog.ErrorContext(ctx, "Failed to parse alliances response", "error", err)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.Int("esi.alliances_count", len(alliances)),
		)
		span.SetStatus(codes.Ok, "successfully retrieved alliances")
	}

	slog.InfoContext(ctx, "Successfully retrieved alliances",
		slog.Int("count", len(alliances)))

	return alliances, nil
}

// GetAllianceInfo retrieves alliance public information
func (c *AllianceClient) GetAllianceInfo(ctx context.Context, allianceID int64) (*AllianceInfoResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/alliances/%d", allianceID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/alliance")
		ctx, span = tracer.Start(ctx, "alliance.GetAllianceInfo")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "alliance_info"),
			attribute.Int64("esi.alliance_id", allianceID),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting alliance info from ESI", "alliance_id", allianceID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var alliance AllianceInfoResponse
		if err := json.Unmarshal(cachedData, &alliance); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
				span.SetStatus(codes.Ok, "cache hit")
			}
			slog.InfoContext(ctx, "Using cached alliance data", "alliance_id", allianceID)
			return &alliance, nil
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
			var alliance AllianceInfoResponse
			if err := json.Unmarshal(cachedData, &alliance); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "Alliance info not modified, using cached data", "alliance_id", allianceID)
				return &alliance, nil
			}
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "alliance_id", allianceID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for alliance %d", allianceID)
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

	var alliance AllianceInfoResponse
	if err := json.Unmarshal(body, &alliance); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(
			attribute.String("esi.alliance_name", alliance.Name),
			attribute.String("esi.alliance_ticker", alliance.Ticker),
		)
		span.SetStatus(codes.Ok, "successfully retrieved alliance info")
	}

	return &alliance, nil
}

// GetAllianceContacts retrieves alliance contacts (requires authentication)
func (c *AllianceClient) GetAllianceContacts(ctx context.Context, allianceID int64, token string) ([]AllianceContact, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/alliances/%d/contacts", allianceID)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/alliance")
		ctx, span = tracer.Start(ctx, "alliance.GetAllianceContacts")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "alliance_contacts"),
			attribute.Int64("esi.alliance_id", allianceID),
		)
	}

	slog.InfoContext(ctx, "Requesting alliance contacts from ESI", "alliance_id", allianceID)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

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

	var contacts []AllianceContact
	if err := json.Unmarshal(body, &contacts); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(attribute.Int("esi.contacts_count", len(contacts)))
		span.SetStatus(codes.Ok, "successfully retrieved alliance contacts")
	}

	return contacts, nil
}

// GetAllianceContactLabels retrieves alliance contact labels (requires authentication)
func (c *AllianceClient) GetAllianceContactLabels(ctx context.Context, allianceID int64, token string) ([]AllianceContactLabel, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/alliances/%d/contacts/labels", allianceID)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/alliance")
		ctx, span = tracer.Start(ctx, "alliance.GetAllianceContactLabels")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "alliance_contact_labels"),
			attribute.Int64("esi.alliance_id", allianceID),
		)
	}

	slog.InfoContext(ctx, "Requesting alliance contact labels from ESI", "alliance_id", allianceID)

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.retryClient.DoWithRetry(ctx, req, 3)
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to call ESI: %w", err)
	}
	defer resp.Body.Close()

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

	var labels []AllianceContactLabel
	if err := json.Unmarshal(body, &labels); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(attribute.Int("esi.labels_count", len(labels)))
		span.SetStatus(codes.Ok, "successfully retrieved alliance contact labels")
	}

	return labels, nil
}

// GetAllianceCorporations retrieves list of alliance member corporations
func (c *AllianceClient) GetAllianceCorporations(ctx context.Context, allianceID int64) ([]int64, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/alliances/%d/corporations", allianceID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/alliance")
		ctx, span = tracer.Start(ctx, "alliance.GetAllianceCorporations")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "alliance_corporations"),
			attribute.Int64("esi.alliance_id", allianceID),
			attribute.String("cache.key", cacheKey),
		)
	}

	slog.InfoContext(ctx, "Requesting alliance corporations from ESI", "alliance_id", allianceID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var corporations []int64
		if err := json.Unmarshal(cachedData, &corporations); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
			}
			return corporations, nil
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
			var corporations []int64
			if err := json.Unmarshal(cachedData, &corporations); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "Alliance corporations not modified, using cached data", "alliance_id", allianceID)
				return corporations, nil
			}
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "alliance_id", allianceID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for alliance %d corporations", allianceID)
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

	var corporations []int64
	if err := json.Unmarshal(body, &corporations); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetAttributes(attribute.Int("esi.corporations_count", len(corporations)))
		span.SetStatus(codes.Ok, "successfully retrieved alliance corporations")
	}

	return corporations, nil
}

// GetAllianceIcons retrieves alliance icon URLs
func (c *AllianceClient) GetAllianceIcons(ctx context.Context, allianceID int64) (*AllianceIconsResponse, error) {
	var span trace.Span
	endpoint := fmt.Sprintf("/alliances/%d/icons", allianceID)
	cacheKey := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	if config.GetBoolEnv("ENABLE_TELEMETRY", true) {
		tracer := otel.Tracer("go-falcon/evegate/alliance")
		ctx, span = tracer.Start(ctx, "alliance.GetAllianceIcons")
		defer span.End()

		span.SetAttributes(
			attribute.String("esi.endpoint", "alliance_icons"),
			attribute.Int64("esi.alliance_id", allianceID),
		)
	}

	slog.InfoContext(ctx, "Requesting alliance icons from ESI", "alliance_id", allianceID)

	// Check cache first
	if cachedData, found, err := c.cacheManager.Get(cacheKey); err == nil && found {
		var icons AllianceIconsResponse
		if err := json.Unmarshal(cachedData, &icons); err == nil {
			if span != nil {
				span.SetAttributes(attribute.Bool("cache.hit", true))
			}
			return &icons, nil
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
			var icons AllianceIconsResponse
			if err := json.Unmarshal(cachedData, &icons); err == nil {
				if span != nil {
					span.SetAttributes(attribute.Bool("cache.hit", true))
					span.SetStatus(codes.Ok, "cache hit - not modified")
				}
				slog.InfoContext(ctx, "Alliance icons not modified, using cached data", "alliance_id", allianceID)
				return &icons, nil
			}
		} else {
			// 304 but no cached data - this shouldn't happen, but handle gracefully
			if span != nil {
				span.SetStatus(codes.Error, "304 response but no cached data available")
			}
			slog.WarnContext(ctx, "Received 304 Not Modified but no cached data available", "alliance_id", allianceID)
			return nil, fmt.Errorf("ESI returned 304 Not Modified but no cached data is available for alliance %d icons", allianceID)
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

	var icons AllianceIconsResponse
	if err := json.Unmarshal(body, &icons); err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if span != nil {
		span.SetStatus(codes.Ok, "successfully retrieved alliance icons")
	}

	return &icons, nil
}