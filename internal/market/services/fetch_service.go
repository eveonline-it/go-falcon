package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
		fetchTimeout:         45 * time.Minute,
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
	for i := 0; i < f.maxConcurrentWorkers; i++ {
		wg.Add(1)
		go f.regionWorker(fetchCtx, &wg, regionChan, resultChan, force)
	}

	// Send regions to workers
	go func() {
		defer close(regionChan)
		for _, region := range regions {
			regionChan <- region
		}
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make([]*RegionFetchResult, 0, len(regions))
	for result := range resultChan {
		results = append(results, result)
	}

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
	result := f.fetchRegionData(ctx, region)

	// Process single region result
	if result.Error != nil {
		return result.Error
	}

	// Store orders directly to live collection for single region updates
	err = f.repository.BulkUpsertOrders(ctx, "market_orders", result.Orders)
	if err != nil {
		return fmt.Errorf("failed to store orders for region %d: %w", regionID, err)
	}

	// Update fetch status
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

	err = f.repository.UpsertFetchStatus(ctx, status)
	if err != nil {
		slog.Error("Failed to update fetch status", "region_id", regionID, "error", err)
	}

	slog.Info("Successfully fetched region orders",
		"region_id", regionID,
		"orders", len(result.Orders),
		"duration_ms", time.Since(startTime).Milliseconds())

	return nil
}

// regionWorker processes regions in parallel
func (f *FetchService) regionWorker(ctx context.Context, wg *sync.WaitGroup, regionChan <-chan *sde.Region, resultChan chan<- *RegionFetchResult, force bool) {
	defer wg.Done()

	for region := range regionChan {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			resultChan <- &RegionFetchResult{
				RegionID: region.RegionID,
				Error:    ctx.Err(),
			}
			return
		default:
		}

		// Check if we need to fetch (unless forced)
		if !force {
			status, err := f.repository.GetFetchStatus(ctx, region.RegionID)
			if err == nil && status != nil {
				// Skip if recently fetched (within last hour)
				if time.Since(status.LastFetchTime) < time.Hour && status.Status == "success" {
					slog.Debug("Skipping region - recently updated", "region_id", region.RegionID)
					resultChan <- &RegionFetchResult{
						RegionID: region.RegionID,
						Orders:   []models.MarketOrder{},
						Skipped:  true,
					}
					continue
				}
			}
		}

		// Add delay between requests to respect ESI rate limits
		time.Sleep(f.requestDelay)

		// Fetch region data
		result := f.fetchRegionData(ctx, region)
		resultChan <- result
	}
}

// fetchRegionData fetches market data for a single region with pagination support
func (f *FetchService) fetchRegionData(ctx context.Context, region *sde.Region) *RegionFetchResult {
	startTime := time.Now()
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
		orders, requestCount, paginationMode, err := f.fetchRegionOrdersByType(ctx, region.RegionID, orderType, paginationParams)
		if err != nil {
			result.Error = fmt.Errorf("failed to fetch %s orders for region %d: %w", orderType, region.RegionID, err)
			return result
		}

		result.Orders = append(result.Orders, orders...)
		result.ESIRequestCount += requestCount
		result.PaginationMode = paginationMode
	}

	result.FetchDuration = time.Since(startTime)

	slog.Debug("Fetched region data",
		"region_id", region.RegionID,
		"orders", len(result.Orders),
		"requests", result.ESIRequestCount,
		"duration_ms", result.FetchDuration.Milliseconds(),
		"pagination_mode", result.PaginationMode)

	return result
}

// fetchRegionOrdersByType fetches orders for a specific region and order type with pagination
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

		// Prepare ESI request parameters
		esiParams := map[string]interface{}{
			"order_type": orderType,
		}

		// Add pagination parameters
		if params.Before != nil || params.After != nil {
			// Token-based pagination
			paginationMode = models.PaginationModeToken
			if params.Before != nil {
				esiParams["before"] = *params.Before
			}
			if params.After != nil {
				esiParams["after"] = *params.After
			}
		} else {
			// Offset-based pagination
			esiParams["page"] = page
		}

		// Make ESI request
		path := fmt.Sprintf("/v1/markets/%d/orders/", regionID)
		response, err := f.makeESIRequest(ctx, path, esiParams)
		if err != nil {
			return allOrders, requestCount, paginationMode, fmt.Errorf("ESI request failed: %w", err)
		}

		requestCount++

		// Parse response
		orders, paginationInfo, err := f.parseMarketOrdersResponse(response, regionID)
		if err != nil {
			return allOrders, requestCount, paginationMode, fmt.Errorf("failed to parse ESI response: %w", err)
		}

		allOrders = append(allOrders, orders...)

		// Check if we need to continue pagination
		if paginationInfo.Mode == models.PaginationModeToken {
			paginationMode = models.PaginationModeToken
			if paginationInfo.Before != nil {
				params.Before = paginationInfo.Before
			} else {
				break // No more pages
			}
		} else {
			// Offset-based pagination
			if len(orders) == 0 || !paginationInfo.HasMore {
				break // No more pages
			}
			page++
		}

		// Safety check to prevent infinite loops
		if requestCount > 1000 {
			slog.Warn("Too many requests for region, stopping", "region_id", regionID, "order_type", orderType, "requests", requestCount)
			break
		}
	}

	return allOrders, requestCount, paginationMode, nil
}

// parseMarketOrdersResponse parses ESI market orders response
func (f *FetchService) parseMarketOrdersResponse(response interface{}, regionID int) ([]models.MarketOrder, *models.PaginationInfo, error) {
	// TODO: Implement proper ESI response parsing
	// This is a placeholder - in reality you would parse the actual ESI JSON response

	orders := []models.MarketOrder{}
	paginationInfo := &models.PaginationInfo{
		Mode:    models.PaginationModeOffset,
		HasMore: false,
	}

	// For now, return empty orders to allow the system to work
	// In production, you would implement proper JSON unmarshaling here

	return orders, paginationInfo, nil
}

// processResults analyzes fetch results and performs atomic collection swap
func (f *FetchService) processResults(ctx context.Context, results []*RegionFetchResult, startTime time.Time) error {
	successful := 0
	failed := 0
	skipped := 0
	totalOrders := 0

	// Analyze results
	for _, result := range results {
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
	shouldSwap := successful > 0 && float64(successful)/float64(successful+failed) >= 0.8 // 80% success rate

	if shouldSwap && totalOrders > 0 {
		return f.performAtomicSwap(ctx, results, startTime)
	} else {
		slog.Warn("Not performing atomic swap due to low success rate or no new data",
			"success_rate", float64(successful)/float64(successful+failed),
			"total_orders", totalOrders)

		// Still update fetch statuses for failed regions
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

	// Insert all successful orders into temp collection
	for _, result := range results {
		if result.Error == nil && !result.Skipped && len(result.Orders) > 0 {
			err := f.repository.BulkUpsertOrders(ctx, tempCollection, result.Orders)
			if err != nil {
				slog.Error("Failed to insert orders into temp collection",
					"region_id", result.RegionID, "error", err)
				// Continue with other regions
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

// makeESIRequest makes an HTTP request to the EVE ESI API
func (f *FetchService) makeESIRequest(ctx context.Context, path string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Build URL with parameters
	url := "https://esi.evetech.net" + path
	if len(params) > 0 {
		url += "?"
		first := true
		for key, value := range params {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%v", key, value)
			first = false
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("User-Agent", "go-falcon/1.0.0")
	req.Header.Set("Accept", "application/json")

	// Make request using the evegateway HTTP client
	client := f.eveGateway.HTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}
