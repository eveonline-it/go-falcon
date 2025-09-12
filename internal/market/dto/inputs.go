package dto

// GetStationOrdersInput represents the input for getting orders from a specific station/structure
type GetStationOrdersInput struct {
	LocationID int64  `path:"location_id" validate:"required" minimum:"30000001" maximum:"999999999999999999" doc:"Station or structure ID"`
	TypeID     int    `query:"type_id,omitempty" minimum:"1" maximum:"2147483647" doc:"Filter by item type ID (optional)"`
	OrderType  string `query:"order_type" enum:"buy,sell,all" default:"all" doc:"Filter by order type"`
	Page       int    `query:"page,omitempty" minimum:"1" maximum:"10000" default:"1" doc:"Page number for pagination"`
	Limit      int    `query:"limit" minimum:"1" maximum:"10000" default:"1000" doc:"Number of orders per page"`
}

// GetRegionOrdersInput represents the input for getting orders from a specific region
type GetRegionOrdersInput struct {
	RegionID  int    `path:"region_id" validate:"required" minimum:"10000001" maximum:"11000033" doc:"Region ID"`
	TypeID    int    `query:"type_id,omitempty" minimum:"1" maximum:"2147483647" doc:"Filter by item type ID (optional)"`
	OrderType string `query:"order_type" enum:"buy,sell,all" default:"all" doc:"Filter by order type"`
	Page      int    `query:"page,omitempty" minimum:"1" maximum:"10000" default:"1" doc:"Page number for pagination"`
	Limit     int    `query:"limit" minimum:"1" maximum:"10000" default:"1000" doc:"Number of orders per page"`
}

// GetItemOrdersInput represents the input for getting orders for a specific item
type GetItemOrdersInput struct {
	TypeID    int    `path:"type_id" validate:"required" minimum:"1" maximum:"2147483647" doc:"Item type ID"`
	RegionID  int    `query:"region_id,omitempty" minimum:"10000001" maximum:"11000033" doc:"Filter by region ID (optional)"`
	OrderType string `query:"order_type" enum:"buy,sell,all" default:"all" doc:"Filter by order type"`
	Page      int    `query:"page,omitempty" minimum:"1" maximum:"10000" default:"1" doc:"Page number for pagination"`
	Limit     int    `query:"limit" minimum:"1" maximum:"10000" default:"1000" doc:"Number of orders per page"`
}

// GetMarketStatusInput represents the input for getting market module status
type GetMarketStatusInput struct {
	// No parameters needed for status endpoint
}

// GetRegionSummaryInput represents the input for getting region market summary
type GetRegionSummaryInput struct {
	RegionID int `path:"region_id" validate:"required" minimum:"10000001" maximum:"11000033" doc:"Region ID"`
}

// SearchOrdersInput represents the input for searching market orders
type SearchOrdersInput struct {
	TypeID     int     `query:"type_id,omitempty" minimum:"1" maximum:"2147483647" doc:"Filter by item type ID"`
	RegionID   int     `query:"region_id,omitempty" minimum:"10000001" maximum:"11000033" doc:"Filter by region ID"`
	SystemID   int     `query:"system_id,omitempty" minimum:"30000001" maximum:"32000000" doc:"Filter by solar system ID"`
	LocationID int64   `query:"location_id,omitempty" minimum:"30000001" maximum:"999999999999999999" doc:"Filter by station/structure ID"`
	OrderType  string  `query:"order_type" enum:"buy,sell,all" default:"all" doc:"Filter by order type"`
	MinPrice   float64 `query:"min_price,omitempty" minimum:"0.01" doc:"Minimum price filter"`
	MaxPrice   float64 `query:"max_price,omitempty" minimum:"0.01" doc:"Maximum price filter"`
	MinVolume  int     `query:"min_volume,omitempty" minimum:"1" doc:"Minimum volume filter"`
	MaxVolume  int     `query:"max_volume,omitempty" minimum:"1" doc:"Maximum volume filter"`
	Page       int     `query:"page,omitempty" minimum:"1" maximum:"10000" default:"1" doc:"Page number for pagination"`
	Limit      int     `query:"limit" minimum:"1" maximum:"10000" default:"1000" doc:"Number of orders per page"`
	SortBy     string  `query:"sort_by" enum:"price,volume,issued,location" default:"price" doc:"Sort field"`
	SortOrder  string  `query:"sort_order" enum:"asc,desc" default:"asc" doc:"Sort order"`
}

// TriggerFetchInput represents the input for manually triggering a market data fetch
type TriggerFetchInput struct {
	RegionID int  `query:"region_id,omitempty" minimum:"10000001" maximum:"11000033" doc:"Specific region to fetch (optional, fetches all if not specified)"`
	Force    bool `query:"force,omitempty" doc:"Force fetch even if recently updated"`
}
