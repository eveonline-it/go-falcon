package dto

import (
	"time"

	"go-falcon/internal/market/models"
)

// MarketOrder represents a market order in API responses
type MarketOrder struct {
	OrderID      int64     `json:"order_id" doc:"Unique order ID"`
	TypeID       int       `json:"type_id" doc:"Item type ID"`
	TypeName     string    `json:"type_name,omitempty" doc:"Item type name"`
	LocationID   int64     `json:"location_id" doc:"Station or structure ID"`
	LocationName string    `json:"location_name,omitempty" doc:"Station or structure name"`
	RegionID     int       `json:"region_id" doc:"Region ID"`
	RegionName   string    `json:"region_name,omitempty" doc:"Region name"`
	SystemID     int       `json:"system_id" doc:"Solar system ID"`
	SystemName   string    `json:"system_name,omitempty" doc:"Solar system name"`
	IsBuyOrder   bool      `json:"is_buy_order" doc:"True for buy orders, false for sell orders"`
	Price        float64   `json:"price" doc:"Price per unit"`
	VolumeRemain int       `json:"volume_remain" doc:"Remaining quantity"`
	VolumeTotal  int       `json:"volume_total" doc:"Original quantity"`
	Duration     int       `json:"duration" doc:"Order duration in days"`
	Issued       time.Time `json:"issued" doc:"Order issue date"`
	MinVolume    int       `json:"min_volume" doc:"Minimum volume for partial fulfillment"`
	Range        string    `json:"range" doc:"Order range (station, region, etc.)"`
	FetchedAt    time.Time `json:"fetched_at" doc:"When this data was last updated"`
}

// LocationInfo represents location information for market data
type LocationInfo struct {
	LocationID  int64  `json:"location_id" doc:"Station or structure ID"`
	Name        string `json:"name" doc:"Location name"`
	RegionID    int    `json:"region_id" doc:"Region ID"`
	RegionName  string `json:"region_name" doc:"Region name"`
	SystemID    int    `json:"system_id" doc:"Solar system ID"`
	SystemName  string `json:"system_name" doc:"Solar system name"`
	IsStructure bool   `json:"is_structure" doc:"True if this is a player structure"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	CurrentPage int  `json:"current_page" doc:"Current page number"`
	TotalPages  int  `json:"total_pages" doc:"Total number of pages"`
	TotalCount  int  `json:"total_count" doc:"Total number of records"`
	HasNext     bool `json:"has_next" doc:"Whether there are more pages"`
	HasPrev     bool `json:"has_prev" doc:"Whether there are previous pages"`
}

// MarketOrdersOutput represents the response for market orders queries
type MarketOrdersOutput struct {
	Body struct {
		Orders       []MarketOrder   `json:"orders" doc:"List of market orders"`
		LocationInfo *LocationInfo   `json:"location_info,omitempty" doc:"Location information"`
		Pagination   *PaginationInfo `json:"pagination" doc:"Pagination information"`
		LastUpdated  time.Time       `json:"last_updated" doc:"When this data was last updated"`
		Summary      *OrderSummary   `json:"summary,omitempty" doc:"Order summary statistics"`
	} `json:"body"`
}

// OrderSummary provides aggregate statistics for a set of orders
type OrderSummary struct {
	TotalOrders  int      `json:"total_orders" doc:"Total number of orders"`
	BuyOrders    int      `json:"buy_orders" doc:"Number of buy orders"`
	SellOrders   int      `json:"sell_orders" doc:"Number of sell orders"`
	LowestSell   *float64 `json:"lowest_sell,omitempty" doc:"Lowest sell price"`
	HighestBuy   *float64 `json:"highest_buy,omitempty" doc:"Highest buy price"`
	TotalVolume  int64    `json:"total_volume" doc:"Total volume across all orders"`
	AveragePrice *float64 `json:"average_price,omitempty" doc:"Average price"`
	UniqueTypes  int      `json:"unique_types" doc:"Number of unique item types"`
}

// RegionSummaryOutput represents the response for region market summary
type RegionSummaryOutput struct {
	Body struct {
		RegionID       int               `json:"region_id" doc:"Region ID"`
		RegionName     string            `json:"region_name" doc:"Region name"`
		TotalOrders    int               `json:"total_orders" doc:"Total market orders in region"`
		BuyOrders      int               `json:"buy_orders" doc:"Number of buy orders"`
		SellOrders     int               `json:"sell_orders" doc:"Number of sell orders"`
		UniqueTypes    int               `json:"unique_types" doc:"Number of unique item types being traded"`
		UniqueStations int               `json:"unique_stations" doc:"Number of unique trading locations"`
		LastUpdated    time.Time         `json:"last_updated" doc:"When this data was last updated"`
		TopTrading     []TypeTradingInfo `json:"top_trading" doc:"Most actively traded items"`
	} `json:"body"`
}

// TypeTradingInfo represents trading information for a specific item type
type TypeTradingInfo struct {
	TypeID      int      `json:"type_id" doc:"Item type ID"`
	TypeName    string   `json:"type_name" doc:"Item type name"`
	OrderCount  int      `json:"order_count" doc:"Number of active orders"`
	TotalVolume int64    `json:"total_volume" doc:"Total volume being traded"`
	LowestSell  *float64 `json:"lowest_sell,omitempty" doc:"Lowest sell price"`
	HighestBuy  *float64 `json:"highest_buy,omitempty" doc:"Highest buy price"`
}

// FetchStatusInfo represents the status of market data fetching for a region
type FetchStatusInfo struct {
	RegionID       int       `json:"region_id" doc:"Region ID"`
	RegionName     string    `json:"region_name" doc:"Region name"`
	Status         string    `json:"status" doc:"Fetch status (success, partial, failed, in_progress)"`
	LastFetch      time.Time `json:"last_fetch" doc:"Last successful fetch time"`
	NextFetch      time.Time `json:"next_fetch" doc:"Next scheduled fetch time"`
	OrderCount     int       `json:"order_count" doc:"Number of orders fetched"`
	FetchDuration  int64     `json:"fetch_duration_ms" doc:"Fetch duration in milliseconds"`
	ErrorMessage   string    `json:"error_message,omitempty" doc:"Error message if fetch failed"`
	PaginationMode string    `json:"pagination_mode" doc:"Pagination mode used (offset, token, mixed)"`
}

// MarketStatusOutput represents the response for market module status
type MarketStatusOutput struct {
	Body struct {
		Module      string    `json:"module" doc:"Module name"`
		Status      string    `json:"status" doc:"Overall module health status"`
		LastFetch   time.Time `json:"last_fetch" doc:"Last successful full fetch time"`
		NextFetch   time.Time `json:"next_fetch" doc:"Next scheduled fetch time"`
		RegionStats struct {
			Total      int            `json:"total" doc:"Total number of regions"`
			Successful int            `json:"successful" doc:"Successfully fetched regions"`
			Failed     int            `json:"failed" doc:"Failed regions"`
			Partial    int            `json:"partial" doc:"Partially successful regions"`
			Breakdown  map[string]int `json:"pagination_breakdown" doc:"Pagination mode breakdown"`
		} `json:"region_stats" doc:"Region-wise statistics"`
		DataStats struct {
			TotalOrders    int       `json:"total_orders" doc:"Total market orders in database"`
			OldestData     time.Time `json:"oldest_data" doc:"Timestamp of oldest order data"`
			NewestData     time.Time `json:"newest_data" doc:"Timestamp of newest order data"`
			CollectionSize int64     `json:"collection_size_bytes" doc:"Database collection size in bytes"`
		} `json:"data_stats" doc:"Data statistics"`
		PaginationInfo struct {
			CurrentMode     string `json:"current_mode" doc:"Current pagination mode (offset, token, mixed)"`
			TokenSupported  bool   `json:"token_support_detected" doc:"Whether token pagination is supported"`
			MigrationStatus string `json:"migration_status" doc:"Migration status (pending, in_progress, completed)"`
		} `json:"pagination_info" doc:"Pagination system information"`
		Performance struct {
			AverageFetchTime int64 `json:"average_fetch_time_ms" doc:"Average fetch time per region in milliseconds"`
			TotalESIRequests int64 `json:"total_esi_requests" doc:"Total ESI requests made"`
			RequestsPerHour  int64 `json:"requests_per_hour" doc:"ESI requests per hour (last 24h)"`
		} `json:"performance" doc:"Performance metrics"`
		RegionStatus []FetchStatusInfo `json:"region_status" doc:"Per-region fetch status"`
	} `json:"body"`
}

// TriggerFetchOutput represents the response for manual fetch triggers
type TriggerFetchOutput struct {
	Body struct {
		Message       string    `json:"message" doc:"Response message"`
		TriggeredAt   time.Time `json:"triggered_at" doc:"When the fetch was triggered"`
		RegionsCount  int       `json:"regions_count" doc:"Number of regions queued for fetching"`
		EstimatedTime int       `json:"estimated_time_minutes" doc:"Estimated completion time in minutes"`
	} `json:"body"`
}

// Helper function to convert model to DTO
func MarketOrderFromModel(order *models.MarketOrder) MarketOrder {
	return MarketOrder{
		OrderID:      order.OrderID,
		TypeID:       order.TypeID,
		LocationID:   order.LocationID,
		RegionID:     order.RegionID,
		SystemID:     order.SystemID,
		IsBuyOrder:   order.IsBuyOrder,
		Price:        order.Price,
		VolumeRemain: order.VolumeRemain,
		VolumeTotal:  order.VolumeTotal,
		Duration:     order.Duration,
		Issued:       order.Issued,
		MinVolume:    order.MinVolume,
		Range:        order.Range,
		FetchedAt:    order.FetchedAt,
	}
}
