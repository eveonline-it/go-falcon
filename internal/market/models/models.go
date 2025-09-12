package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MarketOrder represents a market order from EVE Online
type MarketOrder struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrderID      int64              `bson:"order_id" json:"order_id"`
	TypeID       int                `bson:"type_id" json:"type_id"`
	LocationID   int64              `bson:"location_id" json:"location_id"`
	RegionID     int                `bson:"region_id" json:"region_id"`
	SystemID     int                `bson:"system_id" json:"system_id"`
	IsBuyOrder   bool               `bson:"is_buy_order" json:"is_buy_order"`
	Price        float64            `bson:"price" json:"price"`
	VolumeRemain int                `bson:"volume_remain" json:"volume_remain"`
	VolumeTotal  int                `bson:"volume_total" json:"volume_total"`
	Duration     int                `bson:"duration" json:"duration"`
	Issued       time.Time          `bson:"issued" json:"issued"`
	MinVolume    int                `bson:"min_volume" json:"min_volume"`
	Range        string             `bson:"range" json:"range"`
	FetchedAt    time.Time          `bson:"fetched_at" json:"fetched_at"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name
func (m *MarketOrder) CollectionName() string {
	return "market_orders"
}

// TempCollectionName returns the temporary collection name for atomic swaps
func (m *MarketOrder) TempCollectionName() string {
	return "market_orders_temp"
}

// PaginationMode represents the type of pagination being used
type PaginationMode string

const (
	PaginationModeOffset PaginationMode = "offset"
	PaginationModeToken  PaginationMode = "token"
	PaginationModeAuto   PaginationMode = "auto"
	PaginationModeMixed  PaginationMode = "mixed"
)

// PaginationParams holds parameters for both offset and token-based pagination
type PaginationParams struct {
	// Offset-based pagination (current system)
	Page *int `json:"page,omitempty"`

	// Token-based pagination (future system)
	Before *string `json:"before,omitempty"`
	After  *string `json:"after,omitempty"`

	// Common parameters
	Limit *int `json:"limit,omitempty"`
}

// PaginationInfo contains pagination metadata from ESI responses
type PaginationInfo struct {
	Mode        PaginationMode `json:"mode"`
	CurrentPage *int           `json:"current_page,omitempty"`
	TotalPages  *int           `json:"total_pages,omitempty"`
	Before      *string        `json:"before,omitempty"`
	After       *string        `json:"after,omitempty"`
	HasMore     bool           `json:"has_more"`
}

// MarketFetchStatus tracks the status of market data fetches per region
type MarketFetchStatus struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	RegionID      int                `bson:"region_id" json:"region_id"`
	RegionName    string             `bson:"region_name" json:"region_name"`
	LastFetchTime time.Time          `bson:"last_fetch_time" json:"last_fetch_time"`
	NextFetchTime time.Time          `bson:"next_fetch_time" json:"next_fetch_time"`
	Status        string             `bson:"status" json:"status"` // "success", "partial", "failed", "in_progress"
	OrderCount    int                `bson:"order_count" json:"order_count"`
	ErrorMessage  string             `bson:"error_message,omitempty" json:"error_message,omitempty"`

	// Pagination metadata
	PaginationMode  string  `bson:"pagination_mode" json:"pagination_mode"` // "offset", "token", "mixed"
	LastPageFetched *int    `bson:"last_page_fetched,omitempty" json:"last_page_fetched,omitempty"`
	LastBeforeToken *string `bson:"last_before_token,omitempty" json:"last_before_token,omitempty"`
	LastAfterToken  *string `bson:"last_after_token,omitempty" json:"last_after_token,omitempty"`

	// Data quality tracking
	OldestOrderTime *time.Time `bson:"oldest_order_time,omitempty" json:"oldest_order_time,omitempty"`
	NewestOrderTime *time.Time `bson:"newest_order_time,omitempty" json:"newest_order_time,omitempty"`
	DuplicateCount  int        `bson:"duplicate_count" json:"duplicate_count"`

	// Performance metrics
	FetchDurationMs int64 `bson:"fetch_duration_ms" json:"fetch_duration_ms"`
	ESIRequestCount int   `bson:"esi_request_count" json:"esi_request_count"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name
func (s *MarketFetchStatus) CollectionName() string {
	return "market_fetch_status"
}

// MarketDataBatch represents a batch of market orders with pagination info
type MarketDataBatch struct {
	Orders         []MarketOrder   `json:"orders"`
	PaginationInfo *PaginationInfo `json:"pagination_info"`
	RegionID       int             `json:"region_id"`
	FetchTime      time.Time       `json:"fetch_time"`
	RequestCount   int             `json:"request_count"`
}

// MarketRegionSummary provides aggregate information about a region's market
type MarketRegionSummary struct {
	RegionID    int       `json:"region_id"`
	RegionName  string    `json:"region_name"`
	TotalOrders int       `json:"total_orders"`
	BuyOrders   int       `json:"buy_orders"`
	SellOrders  int       `json:"sell_orders"`
	UniqueTypes int       `json:"unique_types"`
	LastUpdated time.Time `json:"last_updated"`
}

// MarketLocationInfo represents information about a market location
type MarketLocationInfo struct {
	LocationID  int64  `json:"location_id"`
	Name        string `json:"name"`
	RegionID    int    `json:"region_id"`
	RegionName  string `json:"region_name"`
	SystemID    int    `json:"system_id"`
	SystemName  string `json:"system_name"`
	TypeID      int    `json:"type_id,omitempty"`
	TypeName    string `json:"type_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
}

// FetchStatistics represents overall market fetching statistics
type FetchStatistics struct {
	TotalRegions        int            `json:"total_regions"`
	SuccessfulRegions   int            `json:"successful_regions"`
	FailedRegions       int            `json:"failed_regions"`
	PartialRegions      int            `json:"partial_regions"`
	TotalOrders         int            `json:"total_orders"`
	LastFullFetch       time.Time      `json:"last_full_fetch"`
	NextScheduledFetch  time.Time      `json:"next_scheduled_fetch"`
	AverageFetchTime    int64          `json:"average_fetch_time_ms"`
	PaginationBreakdown map[string]int `json:"pagination_breakdown"`
}
