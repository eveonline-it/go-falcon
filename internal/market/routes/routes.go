package routes

import (
	"context"

	"go-falcon/internal/market/dto"
	"go-falcon/internal/market/services"

	"github.com/danielgtaylor/huma/v2"
)

// RegisterMarketRoutes registers all market-related routes
func RegisterMarketRoutes(api huma.API, basePath string, service *services.Service) {
	// Market Orders Endpoints

	// Get orders for a specific station/structure
	huma.Register(api, huma.Operation{
		OperationID: "market-get-station-orders",
		Method:      "GET",
		Path:        basePath + "/orders/station/{location_id}",
		Summary:     "Get market orders for station/structure",
		Description: "Retrieve market orders for a specific station or structure. Data is sourced from database with hourly ESI updates.",
		Tags:        []string{"Market Orders"},
		Errors:      []int{400, 404, 500},
	}, func(ctx context.Context, input *dto.GetStationOrdersInput) (*dto.MarketOrdersOutput, error) {
		page := input.Page
		if page == 0 {
			page = 1
		}

		limit := input.Limit
		if limit == 0 {
			limit = 1000
		}

		// Convert TypeID to pointer for compatibility
		var typeID *int
		if input.TypeID != 0 {
			typeID = &input.TypeID
		}

		return service.GetStationOrders(ctx, input.LocationID, typeID, input.OrderType, page, limit)
	})

	// Get orders for a specific region
	huma.Register(api, huma.Operation{
		OperationID: "market-get-region-orders",
		Method:      "GET",
		Path:        basePath + "/orders/region/{region_id}",
		Summary:     "Get market orders for region",
		Description: "Retrieve market orders for a specific region. Data is sourced from database with hourly ESI updates.",
		Tags:        []string{"Market Orders"},
		Errors:      []int{400, 404, 500},
	}, func(ctx context.Context, input *dto.GetRegionOrdersInput) (*dto.MarketOrdersOutput, error) {
		page := input.Page
		if page == 0 {
			page = 1
		}

		limit := input.Limit
		if limit == 0 {
			limit = 1000
		}

		// Convert TypeID to pointer for compatibility
		var typeID *int
		if input.TypeID != 0 {
			typeID = &input.TypeID
		}

		return service.GetRegionOrders(ctx, input.RegionID, typeID, input.OrderType, page, limit)
	})

	// Get orders for a specific item type
	huma.Register(api, huma.Operation{
		OperationID: "market-get-item-orders",
		Method:      "GET",
		Path:        basePath + "/orders/item/{type_id}",
		Summary:     "Get market orders for item type",
		Description: "Retrieve market orders for a specific item type. Data is sourced from database with hourly ESI updates.",
		Tags:        []string{"Market Orders"},
		Errors:      []int{400, 404, 500},
	}, func(ctx context.Context, input *dto.GetItemOrdersInput) (*dto.MarketOrdersOutput, error) {
		page := input.Page
		if page == 0 {
			page = 1
		}

		limit := input.Limit
		if limit == 0 {
			limit = 1000
		}

		// Convert RegionID to pointer for compatibility
		var regionID *int
		if input.RegionID != 0 {
			regionID = &input.RegionID
		}

		return service.GetItemOrders(ctx, input.TypeID, regionID, input.OrderType, page, limit)
	})

	// Advanced search for market orders
	huma.Register(api, huma.Operation{
		OperationID: "market-search-orders",
		Method:      "GET",
		Path:        basePath + "/orders/search",
		Summary:     "Search market orders",
		Description: "Advanced search for market orders with multiple filter options. Supports filtering by type, region, system, location, price, and volume ranges.",
		Tags:        []string{"Market Orders"},
		Errors:      []int{400, 500},
	}, func(ctx context.Context, input *dto.SearchOrdersInput) (*dto.MarketOrdersOutput, error) {
		return service.SearchOrders(ctx, input)
	})

	// Market Summary Endpoints

	// Get region market summary
	huma.Register(api, huma.Operation{
		OperationID: "market-get-region-summary",
		Method:      "GET",
		Path:        basePath + "/summary/region/{region_id}",
		Summary:     "Get region market summary",
		Description: "Retrieve aggregate market statistics for a specific region including order counts, unique items, and trading activity.",
		Tags:        []string{"Market Summary"},
		Errors:      []int{400, 404, 500},
	}, func(ctx context.Context, input *dto.GetRegionSummaryInput) (*dto.RegionSummaryOutput, error) {
		return service.GetRegionSummary(ctx, input.RegionID)
	})

	// Module Status and Administration

	// Get market module status
	huma.Register(api, huma.Operation{
		OperationID: "market-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get market module status",
		Description: "Retrieve comprehensive status information for the market module including fetch statistics, data freshness, and system health.",
		Tags:        []string{"Module Status"},
		Errors:      []int{500},
	}, func(ctx context.Context, input *dto.GetMarketStatusInput) (*dto.MarketStatusOutput, error) {
		return service.GetMarketStatus(ctx)
	})

	// Manually trigger market data fetch
	huma.Register(api, huma.Operation{
		OperationID: "market-trigger-fetch",
		Method:      "POST",
		Path:        basePath + "/fetch/trigger",
		Summary:     "Trigger market data fetch",
		Description: "Manually trigger a market data fetch operation. Can fetch all regions or a specific region. Use 'force' parameter to bypass cache timing.",
		Tags:        []string{"Market Administration"},
		Errors:      []int{400, 500},
	}, func(ctx context.Context, input *dto.TriggerFetchInput) (*dto.TriggerFetchOutput, error) {
		force := input.Force

		// Convert RegionID to pointer for compatibility
		var regionID *int
		if input.RegionID != 0 {
			regionID = &input.RegionID
		}

		return service.TriggerFetch(ctx, regionID, force)
	})
}
