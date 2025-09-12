package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go-falcon/internal/market/dto"
	"go-falcon/internal/market/models"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"

	"go.mongodb.org/mongo-driver/bson"
)

// Service provides business logic for market operations
type Service struct {
	repository   *Repository
	fetchService *FetchService
	eveGateway   *evegateway.Client
	sdeService   sde.SDEService
}

// NewService creates a new service instance
func NewService(repository *Repository, eveGateway *evegateway.Client, sdeService sde.SDEService) *Service {
	fetchService := NewFetchService(repository, eveGateway, sdeService)

	return &Service{
		repository:   repository,
		fetchService: fetchService,
		eveGateway:   eveGateway,
		sdeService:   sdeService,
	}
}

// GetStationOrders retrieves market orders for a specific station/structure
func (s *Service) GetStationOrders(ctx context.Context, locationID int64, typeID *int, orderType string, page, limit int) (*dto.MarketOrdersOutput, error) {
	// Get orders from database
	orders, total, err := s.repository.GetOrdersByLocation(ctx, locationID, typeID, orderType, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get station orders: %w", err)
	}

	// Convert to DTOs
	dtoOrders := make([]dto.MarketOrder, len(orders))
	for i, order := range orders {
		dtoOrders[i] = dto.MarketOrderFromModel(&order)

		// Enrich with names from SDE
		s.enrichOrderWithNames(&dtoOrders[i])
	}

	// Calculate pagination info
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Get location info
	locationInfo, err := s.getLocationInfo(locationID, dtoOrders)
	if err != nil {
		// Log error but don't fail the request
		// log.Printf("Failed to get location info for %d: %v", locationID, err)
	}

	// Generate summary
	summary := s.calculateOrderSummary(orders)

	result := &dto.MarketOrdersOutput{}
	result.Body.Orders = dtoOrders
	result.Body.LocationInfo = locationInfo
	result.Body.Pagination = &dto.PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  int(total),
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
	result.Body.Summary = summary

	if len(orders) > 0 {
		result.Body.LastUpdated = orders[0].FetchedAt
	} else {
		result.Body.LastUpdated = time.Now()
	}

	return result, nil
}

// GetRegionOrders retrieves market orders for a specific region
func (s *Service) GetRegionOrders(ctx context.Context, regionID int, typeID *int, orderType string, page, limit int) (*dto.MarketOrdersOutput, error) {
	// Get orders from database
	orders, total, err := s.repository.GetOrdersByRegion(ctx, regionID, typeID, orderType, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get region orders: %w", err)
	}

	// Convert to DTOs
	dtoOrders := make([]dto.MarketOrder, len(orders))
	for i, order := range orders {
		dtoOrders[i] = dto.MarketOrderFromModel(&order)

		// Enrich with names from SDE
		s.enrichOrderWithNames(&dtoOrders[i])
	}

	// Calculate pagination info
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Generate summary
	summary := s.calculateOrderSummary(orders)

	result := &dto.MarketOrdersOutput{}
	result.Body.Orders = dtoOrders
	result.Body.Pagination = &dto.PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  int(total),
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
	result.Body.Summary = summary

	if len(orders) > 0 {
		result.Body.LastUpdated = orders[0].FetchedAt
	} else {
		result.Body.LastUpdated = time.Now()
	}

	return result, nil
}

// GetItemOrders retrieves market orders for a specific item type
func (s *Service) GetItemOrders(ctx context.Context, typeID int, regionID *int, orderType string, page, limit int) (*dto.MarketOrdersOutput, error) {
	// Get orders from database
	orders, total, err := s.repository.GetOrdersByType(ctx, typeID, regionID, orderType, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get item orders: %w", err)
	}

	// Convert to DTOs
	dtoOrders := make([]dto.MarketOrder, len(orders))
	for i, order := range orders {
		dtoOrders[i] = dto.MarketOrderFromModel(&order)

		// Enrich with names from SDE
		s.enrichOrderWithNames(&dtoOrders[i])
	}

	// Calculate pagination info
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Generate summary
	summary := s.calculateOrderSummary(orders)

	result := &dto.MarketOrdersOutput{}
	result.Body.Orders = dtoOrders
	result.Body.Pagination = &dto.PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  int(total),
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
	result.Body.Summary = summary

	if len(orders) > 0 {
		result.Body.LastUpdated = orders[0].FetchedAt
	} else {
		result.Body.LastUpdated = time.Now()
	}

	return result, nil
}

// SearchOrders performs advanced search on market orders
func (s *Service) SearchOrders(ctx context.Context, input *dto.SearchOrdersInput) (*dto.MarketOrdersOutput, error) {
	// Build MongoDB filter
	filter := bson.M{}

	if input.TypeID != 0 {
		filter["type_id"] = input.TypeID
	}

	if input.RegionID != 0 {
		filter["region_id"] = input.RegionID
	}

	if input.SystemID != 0 {
		filter["system_id"] = input.SystemID
	}

	if input.LocationID != 0 {
		filter["location_id"] = input.LocationID
	}

	if input.OrderType != "all" {
		filter["is_buy_order"] = input.OrderType == "buy"
	}

	// Price filters
	if input.MinPrice != 0 || input.MaxPrice != 0 {
		priceFilter := bson.M{}
		if input.MinPrice != 0 {
			priceFilter["$gte"] = input.MinPrice
		}
		if input.MaxPrice != 0 {
			priceFilter["$lte"] = input.MaxPrice
		}
		filter["price"] = priceFilter
	}

	// Volume filters
	if input.MinVolume != 0 || input.MaxVolume != 0 {
		volumeFilter := bson.M{}
		if input.MinVolume != 0 {
			volumeFilter["$gte"] = input.MinVolume
		}
		if input.MaxVolume != 0 {
			volumeFilter["$lte"] = input.MaxVolume
		}
		filter["volume_remain"] = volumeFilter
	}

	// Set default values
	page := input.Page
	if page == 0 {
		page = 1
	}

	limit := input.Limit
	if limit == 0 {
		limit = 1000
	}

	// Sort order
	sortOrder := 1 // ascending
	if input.SortOrder == "desc" {
		sortOrder = -1
	}

	// Execute search
	orders, total, err := s.repository.SearchOrders(ctx, filter, input.SortBy, sortOrder, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search orders: %w", err)
	}

	// Convert to DTOs
	dtoOrders := make([]dto.MarketOrder, len(orders))
	for i, order := range orders {
		dtoOrders[i] = dto.MarketOrderFromModel(&order)

		// Enrich with names from SDE
		s.enrichOrderWithNames(&dtoOrders[i])
	}

	// Calculate pagination info
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// Generate summary
	summary := s.calculateOrderSummary(orders)

	result := &dto.MarketOrdersOutput{}
	result.Body.Orders = dtoOrders
	result.Body.Pagination = &dto.PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  int(total),
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
	result.Body.Summary = summary

	if len(orders) > 0 {
		result.Body.LastUpdated = orders[0].FetchedAt
	} else {
		result.Body.LastUpdated = time.Now()
	}

	return result, nil
}

// GetRegionSummary retrieves summary statistics for a region
func (s *Service) GetRegionSummary(ctx context.Context, regionID int) (*dto.RegionSummaryOutput, error) {
	summary, err := s.repository.GetRegionSummary(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region summary: %w", err)
	}

	if summary == nil {
		return nil, fmt.Errorf("no market data found for region %d", regionID)
	}

	// Get region name from SDE
	regionName := fmt.Sprintf("Region %d", regionID) // fallback
	if region, err := s.sdeService.GetRegion(regionID); err == nil && region != nil {
		regionName = fmt.Sprintf("Region %d", region.RegionID)
	}
	summary.RegionName = regionName

	result := &dto.RegionSummaryOutput{}
	result.Body.RegionID = summary.RegionID
	result.Body.RegionName = summary.RegionName
	result.Body.TotalOrders = summary.TotalOrders
	result.Body.BuyOrders = summary.BuyOrders
	result.Body.SellOrders = summary.SellOrders
	result.Body.UniqueTypes = summary.UniqueTypes
	result.Body.LastUpdated = summary.LastUpdated

	// TODO: Implement top trading items query
	result.Body.TopTrading = []dto.TypeTradingInfo{}
	result.Body.UniqueStations = 0 // TODO: Calculate from aggregation

	return result, nil
}

// GetMarketStatus retrieves the overall status of the market module
func (s *Service) GetMarketStatus(ctx context.Context) (*dto.MarketStatusOutput, error) {
	// Get fetch statuses for all regions
	statuses, err := s.repository.GetAllFetchStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fetch statuses: %w", err)
	}

	// Get overall statistics
	overallStats, err := s.repository.GetOverallStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall stats: %w", err)
	}

	// Calculate region statistics
	successful, failed, partial := 0, 0, 0
	paginationBreakdown := make(map[string]int)

	var lastFetch, nextFetch time.Time
	regionStatusList := make([]dto.FetchStatusInfo, len(statuses))

	for i, status := range statuses {
		switch status.Status {
		case "success":
			successful++
		case "failed":
			failed++
		case "partial":
			partial++
		}

		// Track pagination modes
		mode := status.PaginationMode
		if mode == "" {
			mode = "offset"
		}
		paginationBreakdown[mode]++

		// Find latest fetch time
		if lastFetch.IsZero() || status.LastFetchTime.After(lastFetch) {
			lastFetch = status.LastFetchTime
		}

		// Find earliest next fetch time
		if nextFetch.IsZero() || status.NextFetchTime.Before(nextFetch) {
			nextFetch = status.NextFetchTime
		}

		// Get region name from SDE
		regionName := fmt.Sprintf("Region %d", status.RegionID)
		if region, err := s.sdeService.GetRegion(status.RegionID); err == nil && region != nil {
			regionName = fmt.Sprintf("Region %d", region.RegionID)
		}

		regionStatusList[i] = dto.FetchStatusInfo{
			RegionID:       status.RegionID,
			RegionName:     regionName,
			Status:         status.Status,
			LastFetch:      status.LastFetchTime,
			NextFetch:      status.NextFetchTime,
			OrderCount:     status.OrderCount,
			FetchDuration:  status.FetchDurationMs,
			ErrorMessage:   status.ErrorMessage,
			PaginationMode: status.PaginationMode,
		}
	}

	// Determine overall module health
	moduleStatus := "healthy"
	if failed > 0 || partial > successful {
		moduleStatus = "degraded"
	}
	if successful == 0 {
		moduleStatus = "unhealthy"
	}

	// Determine pagination info
	currentMode := "offset"
	tokenSupported := false
	migrationStatus := "pending"

	if paginationBreakdown["token"] > 0 {
		tokenSupported = true
		if paginationBreakdown["token"] > paginationBreakdown["offset"] {
			currentMode = "token"
			migrationStatus = "completed"
		} else {
			currentMode = "mixed"
			migrationStatus = "in_progress"
		}
	}

	result := &dto.MarketStatusOutput{}
	result.Body.Module = "market"
	result.Body.Status = moduleStatus
	result.Body.LastFetch = lastFetch
	result.Body.NextFetch = nextFetch

	result.Body.RegionStats.Total = len(statuses)
	result.Body.RegionStats.Successful = successful
	result.Body.RegionStats.Failed = failed
	result.Body.RegionStats.Partial = partial
	result.Body.RegionStats.Breakdown = paginationBreakdown

	// Data stats from aggregation
	if totalOrders, ok := overallStats["total_orders"].(int32); ok {
		result.Body.DataStats.TotalOrders = int(totalOrders)
	}
	if oldestData, ok := overallStats["oldest_data"].(time.Time); ok {
		result.Body.DataStats.OldestData = oldestData
	}
	if newestData, ok := overallStats["newest_data"].(time.Time); ok {
		result.Body.DataStats.NewestData = newestData
	}

	// Get collection size
	collStats, err := s.repository.GetCollectionStats(ctx, "market_orders")
	if err == nil {
		if size, ok := collStats["size"].(int64); ok {
			result.Body.DataStats.CollectionSize = size
		}
	}

	result.Body.PaginationInfo.CurrentMode = currentMode
	result.Body.PaginationInfo.TokenSupported = tokenSupported
	result.Body.PaginationInfo.MigrationStatus = migrationStatus

	// TODO: Calculate performance metrics
	result.Body.Performance.AverageFetchTime = 0
	result.Body.Performance.TotalESIRequests = 0
	result.Body.Performance.RequestsPerHour = 0

	result.Body.RegionStatus = regionStatusList

	return result, nil
}

// TriggerFetch manually triggers a market data fetch
func (s *Service) TriggerFetch(ctx context.Context, regionID *int, force bool) (*dto.TriggerFetchOutput, error) {
	triggeredAt := time.Now()
	regionsCount := 0

	if regionID != nil {
		// Trigger fetch for specific region
		err := s.fetchService.FetchRegionOrders(ctx, *regionID, force)
		if err != nil {
			return nil, fmt.Errorf("failed to trigger fetch for region %d: %w", *regionID, err)
		}
		regionsCount = 1
	} else {
		// Trigger fetch for all regions
		err := s.fetchService.FetchAllRegionalOrders(ctx, force)
		if err != nil {
			return nil, fmt.Errorf("failed to trigger full fetch: %w", err)
		}

		// Get all regions count from SDE
		regions, err := s.sdeService.GetAllRegions()
		if err != nil {
			return nil, fmt.Errorf("failed to get regions from SDE: %w", err)
		}
		regionsCount = len(regions)
	}

	// Estimate completion time (rough calculation)
	estimatedTime := regionsCount * 2 // ~2 minutes per region
	if regionsCount > 10 {
		estimatedTime = regionsCount / 5 // Parallel processing reduces time
	}

	result := &dto.TriggerFetchOutput{}
	result.Body.Message = "Market data fetch triggered successfully"
	result.Body.TriggeredAt = triggeredAt
	result.Body.RegionsCount = regionsCount
	result.Body.EstimatedTime = estimatedTime

	return result, nil
}

// Helper methods

// enrichOrderWithNames adds names from SDE service
func (s *Service) enrichOrderWithNames(order *dto.MarketOrder) {
	// Get type name
	if typeInfo, err := s.sdeService.GetType(strconv.Itoa(order.TypeID)); err == nil && typeInfo != nil {
		if name, exists := typeInfo.Name["en"]; exists {
			order.TypeName = name
		} else {
			order.TypeName = fmt.Sprintf("Type %d", order.TypeID)
		}
	}

	// Get region name
	if region, err := s.sdeService.GetRegion(order.RegionID); err == nil && region != nil {
		order.RegionName = fmt.Sprintf("Region %d", region.RegionID)
	}

	// Get system name
	if system, err := s.sdeService.GetSolarSystem(order.SystemID); err == nil && system != nil {
		order.SystemName = fmt.Sprintf("System %d", system.SolarSystemID)
	}

	// Get location name (station/structure)
	if station, err := s.sdeService.GetStaStation(int(order.LocationID)); err == nil && station != nil {
		order.LocationName = station.StationName
	} else {
		// For player structures, we might need to use ESI or cache
		order.LocationName = fmt.Sprintf("Structure %d", order.LocationID)
	}
}

// getLocationInfo retrieves detailed location information
func (s *Service) getLocationInfo(locationID int64, orders []dto.MarketOrder) (*dto.LocationInfo, error) {
	info := &dto.LocationInfo{
		LocationID: locationID,
	}

	// Try to get info from first order
	if len(orders) > 0 {
		order := orders[0]
		info.RegionID = order.RegionID
		info.RegionName = order.RegionName
		info.SystemID = order.SystemID
		info.SystemName = order.SystemName
		info.Name = order.LocationName
	}

	// Check if it's a structure (ID > 1000000000000)
	info.IsStructure = locationID > 1000000000000

	// If we don't have name from orders, try SDE
	if info.Name == "" {
		if station, err := s.sdeService.GetStaStation(int(locationID)); err == nil && station != nil {
			info.Name = station.StationName
			if system, err := s.sdeService.GetSolarSystem(station.SolarSystemID); err == nil && system != nil {
				info.SystemID = system.SolarSystemID
				info.SystemName = fmt.Sprintf("System %d", system.SolarSystemID)
				info.RegionID = station.RegionID
				if region, err := s.sdeService.GetRegion(station.RegionID); err == nil && region != nil {
					info.RegionName = fmt.Sprintf("Region %d", region.RegionID)
				}
			}
		} else {
			info.Name = fmt.Sprintf("Structure %d", locationID)
		}
	}

	return info, nil
}

// calculateOrderSummary generates summary statistics for a set of orders
func (s *Service) calculateOrderSummary(orders []models.MarketOrder) *dto.OrderSummary {
	if len(orders) == 0 {
		return &dto.OrderSummary{}
	}

	summary := &dto.OrderSummary{
		TotalOrders: len(orders),
		UniqueTypes: 0,
	}

	var lowestSell, highestBuy *float64
	var totalPrice float64
	var totalVolume int64
	buyOrders, sellOrders := 0, 0
	typeMap := make(map[int]bool)

	for _, order := range orders {
		// Track order types
		if order.IsBuyOrder {
			buyOrders++
			if highestBuy == nil || order.Price > *highestBuy {
				highestBuy = &order.Price
			}
		} else {
			sellOrders++
			if lowestSell == nil || order.Price < *lowestSell {
				lowestSell = &order.Price
			}
		}

		// Calculate totals
		totalPrice += order.Price
		totalVolume += int64(order.VolumeRemain)
		typeMap[order.TypeID] = true
	}

	summary.BuyOrders = buyOrders
	summary.SellOrders = sellOrders
	summary.LowestSell = lowestSell
	summary.HighestBuy = highestBuy
	summary.TotalVolume = totalVolume
	summary.UniqueTypes = len(typeMap)

	// Calculate average price
	if len(orders) > 0 {
		avg := totalPrice / float64(len(orders))
		summary.AveragePrice = &avg
	}

	return summary
}

// CreateIndexes creates necessary database indexes
func (s *Service) CreateIndexes(ctx context.Context) error {
	return s.repository.CreateIndexes(ctx)
}
