package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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

// MarketOrdersResult contains market orders and cache information
type MarketOrdersResult struct {
	Data  []MarketOrderResponse `json:"data"`
	Cache CacheInfo             `json:"cache"`
	Pages *PaginationInfo       `json:"pagination,omitempty"`
}

// MarketHistoryResult contains market history and cache information
type MarketHistoryResult struct {
	Data  []MarketHistoryResponse `json:"data"`
	Cache CacheInfo               `json:"cache"`
}

// MarketStatsResult contains market statistics and cache information
type MarketStatsResult struct {
	Data  []MarketStatsResponse `json:"data"`
	Cache CacheInfo             `json:"cache"`
}

// MarketTypesResult contains market types and cache information
type MarketTypesResult struct {
	Data  []int     `json:"data"`
	Cache CacheInfo `json:"cache"`
}

// Client interface for market-related ESI operations
type Client interface {
	// Market Orders
	GetMarketOrders(ctx context.Context, regionID int, orderType string, page int) ([]MarketOrderResponse, error)
	GetMarketOrdersWithCache(ctx context.Context, regionID int, orderType string, page int) (*MarketOrdersResult, error)
	GetMarketOrdersWithPagination(ctx context.Context, regionID int, orderType string, params PaginationParams) (*MarketOrdersResult, error)

	// Market History
	GetMarketHistory(ctx context.Context, regionID int, typeID int) ([]MarketHistoryResponse, error)
	GetMarketHistoryWithCache(ctx context.Context, regionID int, typeID int) (*MarketHistoryResult, error)

	// Market Statistics
	GetMarketStats(ctx context.Context, regionID int) ([]MarketStatsResponse, error)
	GetMarketStatsWithCache(ctx context.Context, regionID int) (*MarketStatsResult, error)

	// Market Types
	GetMarketTypes(ctx context.Context, regionID int) ([]int, error)
	GetMarketTypesWithCache(ctx context.Context, regionID int) (*MarketTypesResult, error)

	// Structure Market Orders (requires authentication)
	GetStructureOrders(ctx context.Context, structureID int64, token string, page int) ([]MarketOrderResponse, error)
	GetStructureOrdersWithCache(ctx context.Context, structureID int64, token string, page int) (*MarketOrdersResult, error)
}

// MarketOrderResponse represents a market order from ESI
type MarketOrderResponse struct {
	OrderID      int64     `json:"order_id"`
	TypeID       int       `json:"type_id"`
	LocationID   int64     `json:"location_id"`
	VolumeTotal  int       `json:"volume_total"`
	VolumeRemain int       `json:"volume_remain"`
	MinVolume    int       `json:"min_volume"`
	Price        float64   `json:"price"`
	IsBuyOrder   bool      `json:"is_buy_order"`
	Duration     int       `json:"duration"`
	Issued       time.Time `json:"issued"`
	Range        string    `json:"range"`
}

// MarketHistoryResponse represents market history data from ESI
type MarketHistoryResponse struct {
	Date       string  `json:"date"` // YYYY-MM-DD format
	OrderCount int64   `json:"order_count"`
	Volume     int64   `json:"volume"`
	Highest    float64 `json:"highest"`
	Average    float64 `json:"average"`
	Lowest     float64 `json:"lowest"`
}

// MarketStatsResponse represents market statistics from ESI
type MarketStatsResponse struct {
	TypeID           int     `json:"type_id"`
	SellOrderCount   int     `json:"sell_order_count"`
	SellVolumeRemain int64   `json:"sell_volume_remain"`
	SellOrdersMin    float64 `json:"sell_orders_min"`
	SellOrdersMax    float64 `json:"sell_orders_max"`
	BuyOrderCount    int     `json:"buy_order_count"`
	BuyVolumeRemain  int64   `json:"buy_volume_remain"`
	BuyOrdersMin     float64 `json:"buy_orders_min"`
	BuyOrdersMax     float64 `json:"buy_orders_max"`
}

// PaginationParams defines parameters for pagination
type PaginationParams struct {
	// Current offset-based pagination
	Page int `json:"page,omitempty"`

	// Future token-based pagination (when available)
	Before *string `json:"before,omitempty"`
	After  *string `json:"after,omitempty"`
	Limit  *int    `json:"limit,omitempty"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	// Current page information
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages,omitempty"`

	// Token-based pagination (future)
	Before *string `json:"before,omitempty"`
	After  *string `json:"after,omitempty"`

	// Pagination mode detection
	Mode PaginationMode `json:"mode"`
}

// PaginationMode defines the type of pagination used
type PaginationMode string

const (
	PaginationModeOffset PaginationMode = "offset"
	PaginationModeToken  PaginationMode = "token"
	PaginationModeHybrid PaginationMode = "hybrid"
)

// MarketClient implements the market ESI client
type MarketClient struct {
	httpClient   *http.Client
	baseURL      string
	userAgent    string
	cacheManager CacheManager
}

// CacheManager interface - kept for interface compatibility but not used for market data
type CacheManager interface {
	Get(key string) ([]byte, bool, error)
	GetWithExpiry(key string) ([]byte, bool, *time.Time, error)
	GetForNotModified(key string) ([]byte, bool, error)
	Set(key string, data []byte, headers http.Header) error
	RefreshExpiry(key string, headers http.Header) error
}

// NewMarketClient creates a new market ESI client
func NewMarketClient(httpClient *http.Client, cacheManager CacheManager) Client {
	return &MarketClient{
		httpClient:   httpClient,
		baseURL:      "https://esi.evetech.net",
		userAgent:    config.GetEnv("ESI_USER_AGENT", "go-falcon/1.0.0 contact@example.com"),
		cacheManager: cacheManager,
	}
}

// GetMarketOrders fetches market orders for a region
func (c *MarketClient) GetMarketOrders(ctx context.Context, regionID int, orderType string, page int) ([]MarketOrderResponse, error) {
	result, err := c.GetMarketOrdersWithCache(ctx, regionID, orderType, page)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetMarketOrdersWithCache fetches market orders with cache information
func (c *MarketClient) GetMarketOrdersWithCache(ctx context.Context, regionID int, orderType string, page int) (*MarketOrdersResult, error) {
	params := PaginationParams{Page: page}
	return c.GetMarketOrdersWithPagination(ctx, regionID, orderType, params)
}

// GetMarketOrdersWithPagination fetches market orders with pagination support
func (c *MarketClient) GetMarketOrdersWithPagination(ctx context.Context, regionID int, orderType string, params PaginationParams) (*MarketOrdersResult, error) {
	tracer := otel.Tracer("evegateway.market")
	ctx, span := tracer.Start(ctx, "GetMarketOrdersWithPagination",
		trace.WithAttributes(
			attribute.Int("region_id", regionID),
			attribute.String("order_type", orderType),
			attribute.Int("page", params.Page),
		),
	)
	defer span.End()

	// Skip caching for market data - it's too large and changes frequently

	// Build request URL
	path := fmt.Sprintf("/v1/markets/%d/orders/", regionID)
	url := c.baseURL + path

	// Add query parameters
	queryParams := []string{}
	if orderType != "all" {
		queryParams = append(queryParams, fmt.Sprintf("order_type=%s", orderType))
	}

	// Add pagination parameters
	if params.Before != nil {
		queryParams = append(queryParams, fmt.Sprintf("before=%s", *params.Before))
	} else if params.After != nil {
		queryParams = append(queryParams, fmt.Sprintf("after=%s", *params.After))
	} else if params.Page > 0 {
		queryParams = append(queryParams, fmt.Sprintf("page=%d", params.Page))
	}

	if params.Limit != nil && *params.Limit > 0 {
		queryParams = append(queryParams, fmt.Sprintf("limit=%d", *params.Limit))
	}

	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ESI API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var orders []MarketOrderResponse
	if err := json.Unmarshal(body, &orders); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Parse pagination headers
	paginationInfo := c.parsePaginationHeaders(resp.Header)
	paginationInfo.CurrentPage = params.Page
	if params.Page == 0 {
		paginationInfo.CurrentPage = 1
	}

	// Determine expiration from cache headers
	expiresAt := c.parseExpiresHeader(resp.Header)

	span.SetAttributes(
		attribute.Int("orders_count", len(orders)),
	)

	slog.Debug("Market orders fetched from ESI",
		"region_id", regionID,
		"order_type", orderType,
		"orders_count", len(orders),
		"expires_at", expiresAt)

	return &MarketOrdersResult{
		Data: orders,
		Cache: CacheInfo{
			Cached: false,
		},
		Pages: paginationInfo,
	}, nil
}

// GetMarketHistory fetches market history for a region and type
func (c *MarketClient) GetMarketHistory(ctx context.Context, regionID int, typeID int) ([]MarketHistoryResponse, error) {
	result, err := c.GetMarketHistoryWithCache(ctx, regionID, typeID)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetMarketHistoryWithCache fetches market history with cache information
func (c *MarketClient) GetMarketHistoryWithCache(ctx context.Context, regionID int, typeID int) (*MarketHistoryResult, error) {
	tracer := otel.Tracer("evegateway.market")
	ctx, span := tracer.Start(ctx, "GetMarketHistoryWithCache",
		trace.WithAttributes(
			attribute.Int("region_id", regionID),
			attribute.Int("type_id", typeID),
		),
	)
	defer span.End()

	// Build request URL
	url := fmt.Sprintf("%s/v1/markets/%d/history/?type_id=%d", c.baseURL, regionID, typeID)

	return c.fetchMarketHistory(ctx, url, span)
}

// GetMarketStats fetches market statistics for a region
func (c *MarketClient) GetMarketStats(ctx context.Context, regionID int) ([]MarketStatsResponse, error) {
	result, err := c.GetMarketStatsWithCache(ctx, regionID)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetMarketStatsWithCache fetches market statistics with cache information
func (c *MarketClient) GetMarketStatsWithCache(ctx context.Context, regionID int) (*MarketStatsResult, error) {
	tracer := otel.Tracer("evegateway.market")
	ctx, span := tracer.Start(ctx, "GetMarketStatsWithCache",
		trace.WithAttributes(
			attribute.Int("region_id", regionID),
		),
	)
	defer span.End()

	// Build request URL
	url := fmt.Sprintf("%s/v1/markets/%d/stats/", c.baseURL, regionID)

	return c.fetchMarketStats(ctx, url, span)
}

// GetMarketTypes fetches available market types for a region
func (c *MarketClient) GetMarketTypes(ctx context.Context, regionID int) ([]int, error) {
	result, err := c.GetMarketTypesWithCache(ctx, regionID)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetMarketTypesWithCache fetches market types with cache information
func (c *MarketClient) GetMarketTypesWithCache(ctx context.Context, regionID int) (*MarketTypesResult, error) {
	tracer := otel.Tracer("evegateway.market")
	ctx, span := tracer.Start(ctx, "GetMarketTypesWithCache",
		trace.WithAttributes(
			attribute.Int("region_id", regionID),
		),
	)
	defer span.End()

	// Skip caching for market data - it's too large and changes frequently

	url := fmt.Sprintf("%s/v1/markets/%d/types/", c.baseURL, regionID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ESI API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var types []int
	if err := json.Unmarshal(body, &types); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	span.SetAttributes(
		attribute.Int("types_count", len(types)),
	)

	return &MarketTypesResult{
		Data: types,
		Cache: CacheInfo{
			Cached: false,
		},
	}, nil
}

// GetStructureOrders fetches market orders for a structure (requires authentication)
func (c *MarketClient) GetStructureOrders(ctx context.Context, structureID int64, token string, page int) ([]MarketOrderResponse, error) {
	result, err := c.GetStructureOrdersWithCache(ctx, structureID, token, page)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetStructureOrdersWithCache fetches structure orders with cache information
func (c *MarketClient) GetStructureOrdersWithCache(ctx context.Context, structureID int64, token string, page int) (*MarketOrdersResult, error) {
	tracer := otel.Tracer("evegateway.market")
	ctx, span := tracer.Start(ctx, "GetStructureOrdersWithCache",
		trace.WithAttributes(
			attribute.Int64("structure_id", structureID),
			attribute.Int("page", page),
		),
	)
	defer span.End()

	// Build request URL
	url := fmt.Sprintf("%s/v1/markets/structures/%d/", c.baseURL, structureID)
	if page > 1 {
		url += fmt.Sprintf("?page=%d", page)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Skip caching for market data - it's too large and changes frequently

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ESI API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var orders []MarketOrderResponse
	if err := json.Unmarshal(body, &orders); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Parse pagination headers
	paginationInfo := c.parsePaginationHeaders(resp.Header)
	paginationInfo.CurrentPage = page
	if page == 0 {
		paginationInfo.CurrentPage = 1
	}

	span.SetAttributes(
		attribute.Int("orders_count", len(orders)),
	)

	return &MarketOrdersResult{
		Data: orders,
		Cache: CacheInfo{
			Cached: false,
		},
		Pages: paginationInfo,
	}, nil
}

// fetchMarketHistory is a helper for fetching market history data
func (c *MarketClient) fetchMarketHistory(ctx context.Context, url string, span trace.Span) (*MarketHistoryResult, error) {
	// Skip caching for market data - it's too large and changes frequently

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ESI API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data []MarketHistoryResponse
	if err := json.Unmarshal(body, &data); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	span.SetAttributes(
		attribute.Int("data_count", len(data)),
	)

	return &MarketHistoryResult{
		Data: data,
		Cache: CacheInfo{
			Cached: false,
		},
	}, nil
}

// fetchMarketStats is a helper for fetching market statistics data
func (c *MarketClient) fetchMarketStats(ctx context.Context, url string, span trace.Span) (*MarketStatsResult, error) {
	// Skip caching for market data - it's too large and changes frequently

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("ESI API returned status %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data []MarketStatsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	span.SetAttributes(
		attribute.Int("data_count", len(data)),
	)

	return &MarketStatsResult{
		Data: data,
		Cache: CacheInfo{
			Cached: false,
		},
	}, nil
}

// parsePaginationHeaders parses pagination information from response headers
func (c *MarketClient) parsePaginationHeaders(headers http.Header) *PaginationInfo {
	info := &PaginationInfo{
		Mode: PaginationModeOffset, // Default to offset mode
	}

	// Check for X-Pages header (current ESI standard)
	if pages := headers.Get("X-Pages"); pages != "" {
		if totalPages, err := strconv.Atoi(pages); err == nil {
			info.TotalPages = totalPages
		}
	}

	// Check for future token-based pagination headers
	if before := headers.Get("X-Before"); before != "" {
		info.Before = &before
		info.Mode = PaginationModeToken
	}

	if after := headers.Get("X-After"); after != "" {
		info.After = &after
		info.Mode = PaginationModeToken
	}

	// If both token and offset headers present, it's hybrid mode
	if info.TotalPages > 0 && (info.Before != nil || info.After != nil) {
		info.Mode = PaginationModeHybrid
	}

	return info
}

// parseExpiresHeader parses expiration information from response headers
func (c *MarketClient) parseExpiresHeader(headers http.Header) time.Time {
	// Default cache duration if no expires header
	defaultExpiration := time.Now().Add(5 * time.Minute)

	// Check Expires header
	if expires := headers.Get("Expires"); expires != "" {
		if t, err := http.ParseTime(expires); err == nil {
			return t
		}
	}

	// Check Cache-Control max-age
	if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		parts := strings.Split(cacheControl, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "max-age=") {
				if seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age=")); err == nil {
					return time.Now().Add(time.Duration(seconds) * time.Second)
				}
			}
		}
	}

	return defaultExpiration
}
