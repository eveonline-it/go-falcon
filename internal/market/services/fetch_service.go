package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"go-falcon/internal/market/models"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"
)

// FetchService handles ESI market data fetching with pagination support
type FetchService struct {
	repository *Repository
	eveGateway *evegateway.Client
	sdeService sde.SDEService

	// Configuration
	maxConcurrentWorkers int
	requestDelay         time.Duration
	fetchTimeout         time.Duration
}

// NewFetchService creates a new fetch service instance
func NewFetchService(repository *Repository, eveGateway *evegateway.Client, sdeService sde.SDEService) *FetchService {
	return &FetchService{
		repository:           repository,
		eveGateway:           eveGateway,
		sdeService:           sdeService,
		maxConcurrentWorkers: 8,
		requestDelay:         200 * time.Millisecond,
		fetchTimeout:         90 * time.Minute, // Increased timeout for large market data sets
	}
}

// FetchAllRegionalOrders fetches market orders for all regions with parallel processing
func (f *FetchService) FetchAllRegionalOrders(ctx context.Context, force bool) error {
	startTime := time.Now()
	slog.Info("Starting full regional market data fetch", "force", force)

	// Create context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, f.fetchTimeout)
	defer cancel()

	// Get all regions from SDE
	regions, err := f.sdeService.GetAllRegions()
	if err != nil {
		return fmt.Errorf("failed to get regions from SDE: %w", err)
	}
	if len(regions) == 0 {
		return fmt.Errorf("no regions found in SDE service")
	}

	slog.Info("Found regions for market fetch", "count", len(regions))

	// Create worker pool for parallel processing
	regionChan := make(chan *sde.Region, len(regions))
	resultChan := make(chan *RegionFetchResult, len(regions))

	// Start workers
	var wg sync.WaitGroup
	slog.Info("Starting worker pool", "workers", f.maxConcurrentWorkers, "timeout", f.fetchTimeout)
	for i := 0; i < f.maxConcurrentWorkers; i++ {
		wg.Add(1)
		workerID := i
		slog.Info("Starting worker", "worker_id", workerID)
		go f.regionWorker(fetchCtx, &wg, regionChan, resultChan, force, workerID)
	}

	// Send regions to workers
	go func() {
		defer close(regionChan)
		slog.Info("Sending regions to workers", "total_regions", len(regions))
		for i, region := range regions {
			if i < 5 { // Log first 5 regions
				slog.Info("Sending region to worker", "region_id", region.RegionID, "index", i)
			}
			regionChan <- region
		}
		slog.Info("Finished sending all regions to workers")
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make([]*RegionFetchResult, 0, len(regions))
	slog.Info("Collecting results from workers")
	resultCount := 0
	for result := range resultChan {
		resultCount++
		if result.Error != nil {
			slog.Info("Received result with error", "region_id", result.RegionID, "error", result.Error, "count", resultCount)
		} else {
			slog.Info("Received successful result", "region_id", result.RegionID, "orders", len(result.Orders), "skipped", result.Skipped, "count", resultCount)
		}
		results = append(results, result)
		if resultCount%10 == 0 { // Progress update every 10 results
			slog.Info("Results collection progress", "collected", resultCount, "expected", len(regions))
		}
	}
	slog.Info("Finished collecting results", "total_results", len(results))

	// Analyze results and perform atomic swap if successful enough
	return f.processResults(fetchCtx, results, startTime)
}

// FetchRegionOrders fetches market orders for a specific region
func (f *FetchService) FetchRegionOrders(ctx context.Context, regionID int, force bool) error {
	startTime := time.Now()
	slog.Info("Fetching market orders for region", "region_id", regionID, "force", force)

	// Check if we need to fetch (unless forced)
	if !force {
		status, err := f.repository.GetFetchStatus(ctx, regionID)
		if err == nil && status != nil {
			// Skip if recently fetched (within last hour)
			if time.Since(status.LastFetchTime) < time.Hour && status.Status == "success" {
				slog.Info("Skipping region fetch - recently updated", "region_id", regionID, "last_fetch", status.LastFetchTime)
				return nil
			}
		}
	}

	// Get region from SDE
	region, err := f.sdeService.GetRegion(regionID)
	if err != nil {
		return fmt.Errorf("failed to get region %d from SDE: %w", regionID, err)
	}
	if region == nil {
		return fmt.Errorf("region %d not found in SDE", regionID)
	}

	// Perform fetch
	slog.Info("Starting fetchRegionData", "region_id", regionID)
	result := f.fetchRegionData(ctx, region)
	slog.Info("Completed fetchRegionData", "region_id", regionID, "orders", len(result.Orders), "error", result.Error)

	// Process single region result
	if result.Error != nil {
		slog.Error("FetchRegionData returned error", "region_id", regionID, "error", result.Error)
		return result.Error
	}

	// Store orders directly to live collection for single region updates
	slog.Info("Starting BulkUpsertOrders for single region", "region_id", regionID, "orders_count", len(result.Orders))
	err = f.repository.BulkUpsertOrders(ctx, "market_orders", result.Orders)
	if err != nil {
		slog.Error("BulkUpsertOrders failed for single region", "region_id", regionID, "error", err)
		return fmt.Errorf("failed to store orders for region %d: %w", regionID, err)
	}
	slog.Info("Completed BulkUpsertOrders for single region", "region_id", regionID)

	// Update fetch status
	slog.Info("Creating fetch status record", "region_id", regionID)
	status := &models.MarketFetchStatus{
		RegionID:        regionID,
		RegionName:      fmt.Sprintf("Region %d", region.RegionID),
		LastFetchTime:   result.FetchTime,
		NextFetchTime:   result.FetchTime.Add(time.Hour), // Next fetch in 1 hour
		Status:          "success",
		OrderCount:      len(result.Orders),
		PaginationMode:  string(result.PaginationMode),
		FetchDurationMs: time.Since(startTime).Milliseconds(),
		ESIRequestCount: result.ESIRequestCount,
	}

	slog.Info("Upserting fetch status", "region_id", regionID)
	err = f.repository.UpsertFetchStatus(ctx, status)
	if err != nil {
		slog.Error("Failed to update fetch status", "region_id", regionID, "error", err)
	} else {
		slog.Info("Successfully updated fetch status", "region_id", regionID)
	}

	slog.Info("Successfully fetched region orders",
		"region_id", regionID,
		"orders", len(result.Orders),
		"duration_ms", time.Since(startTime).Milliseconds())

	return nil
}

// regionWorker processes regions in parallel
func (f *FetchService) regionWorker(ctx context.Context, wg *sync.WaitGroup, regionChan <-chan *sde.Region, resultChan chan<- *RegionFetchResult, force bool, workerID int) {
	defer wg.Done()
	slog.Info("Worker started", "worker_id", workerID)

	regionsProcessed := 0
	for region := range regionChan {
		regionsProcessed++
		slog.Debug("Worker processing region", "worker_id", workerID, "region_id", region.RegionID, "regions_processed", regionsProcessed)

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			slog.Debug("Worker cancelled", "worker_id", workerID, "region_id", region.RegionID)
			resultChan <- &RegionFetchResult{
				RegionID: region.RegionID,
				Error:    ctx.Err(),
			}
			return
		default:
		}

		// Check if we need to fetch (unless forced)
		if !force {
			slog.Debug("Checking fetch status", "worker_id", workerID, "region_id", region.RegionID)
			status, err := f.repository.GetFetchStatus(ctx, region.RegionID)
			if err != nil {
				slog.Debug("Failed to get fetch status, will fetch anyway", "worker_id", workerID, "region_id", region.RegionID, "error", err)
			} else if status != nil {
				// Skip if recently fetched (within last hour)
				timeSinceLastFetch := time.Since(status.LastFetchTime)
				slog.Debug("Found existing fetch status", "worker_id", workerID, "region_id", region.RegionID, "last_fetch", status.LastFetchTime, "time_since", timeSinceLastFetch, "status", status.Status)
				if timeSinceLastFetch < time.Hour && status.Status == "success" {
					slog.Debug("Skipping region - recently updated", "worker_id", workerID, "region_id", region.RegionID)
					resultChan <- &RegionFetchResult{
						RegionID: region.RegionID,
						Orders:   []models.MarketOrder{},
						Skipped:  true,
					}
					continue
				}
			} else {
				slog.Debug("No existing fetch status found, will fetch", "worker_id", workerID, "region_id", region.RegionID)
			}
		}

		// Add delay between requests to respect ESI rate limits
		slog.Debug("Adding delay before fetch", "worker_id", workerID, "region_id", region.RegionID, "delay", f.requestDelay)
		time.Sleep(f.requestDelay)

		// Fetch region data
		slog.Debug("Starting region data fetch", "worker_id", workerID, "region_id", region.RegionID)
		result := f.fetchRegionData(ctx, region)
		if result.Error != nil {
			slog.Debug("Region fetch completed with error", "worker_id", workerID, "region_id", region.RegionID, "error", result.Error)
		} else {
			slog.Debug("Region fetch completed successfully", "worker_id", workerID, "region_id", region.RegionID, "orders", len(result.Orders))
		}
		resultChan <- result
	}
	slog.Debug("Worker finished", "worker_id", workerID, "regions_processed", regionsProcessed)
}

// fetchRegionData fetches market data for a single region with pagination support
func (f *FetchService) fetchRegionData(ctx context.Context, region *sde.Region) *RegionFetchResult {
	startTime := time.Now()
	slog.Debug("Starting fetchRegionData", "region_id", region.RegionID)
	result := &RegionFetchResult{
		RegionID:       region.RegionID,
		FetchTime:      startTime,
		Orders:         []models.MarketOrder{},
		PaginationMode: models.PaginationModeOffset, // Default to offset
	}

	// Try to detect pagination mode
	paginationParams := models.PaginationParams{}

	// Fetch all order types (buy and sell)
	for _, orderType := range []string{"buy", "sell"} {
		slog.Debug("Fetching orders by type", "region_id", region.RegionID, "order_type", orderType)
		orders, requestCount, paginationMode, err := f.fetchRegionOrdersByType(ctx, region.RegionID, orderType, paginationParams)
		if err != nil {
			slog.Debug("Failed to fetch orders by type", "region_id", region.RegionID, "order_type", orderType, "error", err)
			result.Error = fmt.Errorf("failed to fetch %s orders for region %d: %w", orderType, region.RegionID, err)
			return result
		}

		slog.Debug("Successfully fetched orders by type", "region_id", region.RegionID, "order_type", orderType, "orders", len(orders), "requests", requestCount)
		result.Orders = append(result.Orders, orders...)
		result.ESIRequestCount += requestCount
		result.PaginationMode = paginationMode
	}

	result.FetchDuration = time.Since(startTime)

	slog.Info("Completed fetchRegionData processing",
		"region_id", region.RegionID,
		"orders", len(result.Orders),
		"requests", result.ESIRequestCount,
		"duration_ms", result.FetchDuration.Milliseconds(),
		"pagination_mode", result.PaginationMode)

	return result
}

// fetchRegionOrdersByType fetches orders for a specific region and order type using evegateway
func (f *FetchService) fetchRegionOrdersByType(ctx context.Context, regionID int, orderType string, params models.PaginationParams) ([]models.MarketOrder, int, models.PaginationMode, error) {
	var allOrders []models.MarketOrder
	requestCount := 0
	paginationMode := models.PaginationModeOffset

	// Start with page 1 for offset-based pagination
	page := 1

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return allOrders, requestCount, paginationMode, ctx.Err()
		default:
		}

		// Use evegateway market client to fetch orders
		slog.Debug("Making ESI request", "region_id", regionID, "order_type", orderType, "page", page, "request_count", requestCount+1)
		esiOrders, err := f.eveGateway.Market.GetMarketOrders(ctx, regionID, orderType, page)
		if err != nil {
			slog.Debug("ESI request failed", "region_id", regionID, "order_type", orderType, "page", page, "error", err)
			return allOrders, requestCount, paginationMode, fmt.Errorf("ESI request failed: %w", err)
		}
		slog.Debug("ESI request successful", "region_id", regionID, "order_type", orderType, "page", page, "orders_received", len(esiOrders))

		requestCount++

		// Convert ESI response to internal market order format
		orders, err := f.convertESIOrdersToModels(esiOrders, regionID)
		if err != nil {
			return allOrders, requestCount, paginationMode, fmt.Errorf("failed to convert ESI orders: %w", err)
		}

		allOrders = append(allOrders, orders...)

		// Check if we need to continue pagination
		// Simple pagination: if we get less than expected, we're done
		slog.Debug("Pagination check", "region_id", regionID, "order_type", orderType, "page", page, "orders_received", len(esiOrders), "total_orders_so_far", len(allOrders))
		if len(esiOrders) == 0 {
			slog.Debug("No more orders, stopping pagination", "region_id", regionID, "order_type", orderType, "final_page", page)
			break // No more pages
		}

		page++

		// Safety check to prevent infinite loops
		if requestCount > 1000 {
			slog.Warn("Too many requests for region, stopping", "region_id", regionID, "order_type", orderType, "requests", requestCount)
			break
		}

		// Break if we get less than a full page (indicates last page)
		if len(esiOrders) < 1000 { // ESI typically returns up to 1000 orders per page
			slog.Debug("Less than full page, stopping pagination", "region_id", regionID, "order_type", orderType, "orders_in_page", len(esiOrders), "final_page", page-1)
			break
		}
	}

	return allOrders, requestCount, paginationMode, nil
}

// convertESIOrdersToModels converts evegateway ESI response to internal model format
func (f *FetchService) convertESIOrdersToModels(esiOrders []map[string]any, regionID int) ([]models.MarketOrder, error) {
	orders := make([]models.MarketOrder, 0, len(esiOrders))
	fetchedAt := time.Now()

	for i, esiOrder := range esiOrders {
		// Debug: log first ESI order to see actual structure
		if i == 0 {
			slog.Info("First ESI order structure for debugging", "region_id", regionID, "esi_order", esiOrder)
			// Also check specific problematic fields
			if issuedField, exists := esiOrder["issued"]; exists {
				slog.Info("Issued field debug", "type", fmt.Sprintf("%T", issuedField), "value", issuedField)
			} else {
				slog.Info("Issued field is missing from ESI response")
			}
		}

		order := models.MarketOrder{
			RegionID:  regionID,
			FetchedAt: fetchedAt,
			CreatedAt: fetchedAt,
			UpdatedAt: fetchedAt,
		}

		// Extract fields from ESI response with flexible type conversion
		if orderID := f.extractInt64(esiOrder, "order_id"); orderID != 0 {
			order.OrderID = orderID
		}
		if typeID := f.extractInt(esiOrder, "type_id"); typeID != 0 {
			order.TypeID = typeID
		}
		if locationID := f.extractInt64(esiOrder, "location_id"); locationID != 0 {
			order.LocationID = locationID
		}
		if volumeTotal := f.extractInt(esiOrder, "volume_total"); volumeTotal != 0 {
			order.VolumeTotal = volumeTotal
		}
		if volumeRemain := f.extractInt(esiOrder, "volume_remain"); volumeRemain != 0 {
			order.VolumeRemain = volumeRemain
		}
		if minVolume := f.extractInt(esiOrder, "min_volume"); minVolume >= 0 {
			order.MinVolume = minVolume
		}
		if price, ok := esiOrder["price"].(float64); ok {
			order.Price = price
		}
		if isBuyOrder, ok := esiOrder["is_buy_order"].(bool); ok {
			order.IsBuyOrder = isBuyOrder
		}
		if duration := f.extractInt(esiOrder, "duration"); duration != 0 {
			order.Duration = duration
		}
		// Parse issued date with flexible format handling
		if issued := f.extractTimeString(esiOrder, "issued"); !issued.IsZero() {
			order.Issued = issued
		}
		if orderRange, ok := esiOrder["range"].(string); ok {
			order.Range = orderRange
		}

		// Resolve system_id from location_id using SDE service
		if order.LocationID != 0 {
			if systemID := f.resolveSystemIDFromLocation(order.LocationID); systemID != 0 {
				order.SystemID = systemID
				// Debug: log first few successful resolutions
				if i < 3 {
					slog.Info("System ID resolved", "location_id", order.LocationID, "system_id", systemID)
				}
			} else if i < 3 {
				slog.Info("Could not resolve system ID", "location_id", order.LocationID)
			}
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// Helper functions for flexible type extraction from ESI responses

// extractInt64 safely extracts int64 from various numeric types
func (f *FetchService) extractInt64(data map[string]any, field string) int64 {
	value, exists := data[field]
	if !exists {
		return 0
	}

	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	default:
		slog.Debug("Unexpected type for field", "field", field, "type", fmt.Sprintf("%T", v), "value", v)
	}
	return 0
}

// extractInt safely extracts int from various numeric types
func (f *FetchService) extractInt(data map[string]any, field string) int {
	return int(f.extractInt64(data, field))
}

// extractTimeString safely extracts and parses time from ESI response
func (f *FetchService) extractTimeString(data map[string]any, field string) time.Time {
	value, exists := data[field]
	if !exists {
		return time.Time{}
	}

	timeStr, ok := value.(string)
	if !ok {
		slog.Debug("Time field is not a string", "field", field, "type", fmt.Sprintf("%T", value), "value", value)
		return time.Time{}
	}

	// Try different time formats that ESI might use
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,       // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05Z", // Simple UTC format
		"2006-01-02T15:04:05",  // Without timezone
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, timeStr); err == nil {
			return parsedTime
		}
	}

	slog.Debug("Failed to parse time field", "field", field, "value", timeStr)
	return time.Time{}
}

// resolveSystemIDFromLocation resolves system_id from location_id using SDE
func (f *FetchService) resolveSystemIDFromLocation(locationID int64) int {
	// First try as a station ID
	if station, err := f.sdeService.GetStaStation(int(locationID)); err == nil && station != nil {
		return station.SolarSystemID
	}

	// If not a station, it might be a structure ID
	// Structure IDs are typically > 1000000000000 (1 trillion)
	if locationID > 1000000000000 {
		// For structure IDs, we can't resolve system_id from SDE alone
		// as structures are player-built and not in static data
		// This would require additional ESI calls or a structures cache
		slog.Debug("Structure location detected, cannot resolve system_id from SDE", "location_id", locationID)
		return 0
	}

	// If it's neither a known station nor a structure, log it
	slog.Debug("Unknown location_id, cannot resolve system_id", "location_id", locationID)
	return 0
}

// processResults analyzes fetch results and performs atomic collection swap
func (f *FetchService) processResults(ctx context.Context, results []*RegionFetchResult, startTime time.Time) error {
	slog.Debug("Processing fetch results", "total_results", len(results))
	successful := 0
	failed := 0
	skipped := 0
	totalOrders := 0

	// Analyze results
	for i, result := range results {
		slog.Debug("Analyzing result", "index", i, "region_id", result.RegionID, "skipped", result.Skipped, "error", result.Error, "orders", len(result.Orders))
		if result.Skipped {
			skipped++
		} else if result.Error != nil {
			failed++
			slog.Error("Region fetch failed", "region_id", result.RegionID, "error", result.Error)
		} else {
			successful++
			totalOrders += len(result.Orders)
		}
	}

	slog.Info("Market fetch results summary",
		"successful", successful,
		"failed", failed,
		"skipped", skipped,
		"total_orders", totalOrders)

	// Decide whether to perform atomic swap
	successRate := float64(0)
	if successful+failed > 0 {
		successRate = float64(successful) / float64(successful+failed)
	}
	shouldSwap := successful > 0 && successRate >= 0.8 // 80% success rate

	slog.Debug("Atomic swap decision", "should_swap", shouldSwap, "successful", successful, "failed", failed, "success_rate", successRate, "total_orders", totalOrders)

	if shouldSwap && totalOrders > 0 {
		slog.Debug("Performing atomic swap")
		return f.performAtomicSwap(ctx, results, startTime)
	} else {
		slog.Warn("Not performing atomic swap due to low success rate or no new data",
			"success_rate", successRate,
			"total_orders", totalOrders,
			"should_swap", shouldSwap)

		// Still update fetch statuses for failed regions
		slog.Debug("Updating fetch statuses without atomic swap")
		f.updateFetchStatuses(ctx, results, startTime)
		return nil
	}
}

// performAtomicSwap performs atomic collection swapping for successful regions
func (f *FetchService) performAtomicSwap(ctx context.Context, results []*RegionFetchResult, startTime time.Time) error {
	slog.Info("Starting atomic collection swap")

	// Step 1: Store all successful data in temporary collection
	tempCollection := "market_orders_temp"

	// Drop temp collection if it exists
	f.repository.DropCollection(ctx, tempCollection)

	// Count total orders and regions for progress tracking
	totalOrders := 0
	successfulRegions := 0
	for _, result := range results {
		if result.Error == nil && !result.Skipped && len(result.Orders) > 0 {
			totalOrders += len(result.Orders)
			successfulRegions++
		}
	}

	slog.Info("Starting bulk insert to temp collection",
		"temp_collection", tempCollection,
		"total_orders", totalOrders,
		"regions_to_process", successfulRegions)

	// Insert all successful orders into temp collection
	processedRegions := 0
	processedOrders := 0
	for _, result := range results {
		if result.Error == nil && !result.Skipped && len(result.Orders) > 0 {
			processedRegions++
			processedOrders += len(result.Orders)

			slog.Info("Inserting region orders to temp collection",
				"region_id", result.RegionID,
				"orders", len(result.Orders),
				"progress", fmt.Sprintf("%d/%d regions", processedRegions, successfulRegions),
				"total_orders_so_far", processedOrders)

			err := f.repository.BulkUpsertOrders(ctx, tempCollection, result.Orders)
			if err != nil {
				slog.Error("Failed to insert orders into temp collection",
					"region_id", result.RegionID, "orders", len(result.Orders), "error", err)
				// Continue with other regions
			} else {
				slog.Info("Successfully inserted region orders",
					"region_id", result.RegionID,
					"orders", len(result.Orders))
			}
		}
	}

	// Step 2: Validate temp collection has reasonable data
	tempStats, err := f.repository.GetCollectionStats(ctx, tempCollection)
	if err != nil {
		return fmt.Errorf("failed to get temp collection stats: %w", err)
	}

	tempCount := int64(0)
	if count, ok := tempStats["count"].(int64); ok {
		tempCount = count
	}

	if tempCount == 0 {
		return fmt.Errorf("temp collection is empty, aborting swap")
	}

	slog.Info("Temp collection validation passed", "orders", tempCount)

	// Step 3: Atomic swap
	oldCollection := "market_orders_old"
	liveCollection := "market_orders"

	// Drop old backup collection if it exists
	f.repository.DropCollection(ctx, oldCollection)

	// Rename live to old (backup)
	err = f.repository.RenameCollection(ctx, liveCollection, oldCollection)
	if err != nil {
		slog.Error("Failed to backup live collection", "error", err)
		// Continue anyway - new data is better than old
	}

	// Rename temp to live
	err = f.repository.RenameCollection(ctx, tempCollection, liveCollection)
	if err != nil {
		// This is critical - try to restore backup
		slog.Error("CRITICAL: Failed to swap temp to live collection", "error", err)

		// Try to restore from backup
		if restoreErr := f.repository.RenameCollection(ctx, oldCollection, liveCollection); restoreErr != nil {
			slog.Error("CRITICAL: Failed to restore backup collection", "restore_error", restoreErr)
		}

		return fmt.Errorf("atomic swap failed: %w", err)
	}

	// Step 4: Cleanup old backup
	f.repository.DropCollection(ctx, oldCollection)

	slog.Info("Atomic collection swap completed successfully",
		"orders", tempCount,
		"duration_ms", time.Since(startTime).Milliseconds())

	// Step 5: Update fetch statuses
	f.updateFetchStatuses(ctx, results, startTime)

	return nil
}

// updateFetchStatuses updates fetch status records for all regions
func (f *FetchService) updateFetchStatuses(ctx context.Context, results []*RegionFetchResult, startTime time.Time) {
	for _, result := range results {
		status := &models.MarketFetchStatus{
			RegionID:        result.RegionID,
			LastFetchTime:   result.FetchTime,
			NextFetchTime:   result.FetchTime.Add(time.Hour), // Next fetch in 1 hour
			OrderCount:      len(result.Orders),
			FetchDurationMs: result.FetchDuration.Milliseconds(),
			ESIRequestCount: result.ESIRequestCount,
			PaginationMode:  string(result.PaginationMode),
		}

		// Get region name from SDE
		if region, err := f.sdeService.GetRegion(result.RegionID); err == nil && region != nil {
			status.RegionName = fmt.Sprintf("Region %d", region.RegionID)
		}

		if result.Skipped {
			status.Status = "skipped"
		} else if result.Error != nil {
			status.Status = "failed"
			status.ErrorMessage = result.Error.Error()
		} else {
			status.Status = "success"
		}

		err := f.repository.UpsertFetchStatus(ctx, status)
		if err != nil {
			slog.Error("Failed to update fetch status", "region_id", result.RegionID, "error", err)
		}
	}
}

// RegionFetchResult represents the result of fetching data for a single region
type RegionFetchResult struct {
	RegionID        int                   `json:"region_id"`
	Orders          []models.MarketOrder  `json:"orders"`
	FetchTime       time.Time             `json:"fetch_time"`
	FetchDuration   time.Duration         `json:"fetch_duration"`
	ESIRequestCount int                   `json:"esi_request_count"`
	PaginationMode  models.PaginationMode `json:"pagination_mode"`
	Error           error                 `json:"error,omitempty"`
	Skipped         bool                  `json:"skipped"`
}
