package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/internal/corporation/dto"
	"go-falcon/internal/corporation/models"
	"go-falcon/pkg/evegateway"

	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles corporation business logic
type Service struct {
	repository *Repository
	eveClient  *evegateway.Client
}

// NewService creates a new corporation service
func NewService(repository *Repository, eveClient *evegateway.Client) *Service {
	return &Service{
		repository: repository,
		eveClient:  eveClient,
	}
}

// GetCorporationInfo retrieves corporation information, first checking the database,
// then falling back to EVE ESI if not found or data is stale
func (s *Service) GetCorporationInfo(ctx context.Context, corporationID int) (*dto.CorporationInfoOutput, error) {
	slog.InfoContext(ctx, "Getting corporation info", "corporation_id", corporationID)
	
	// Try to get from database first
	corporation, err := s.repository.GetCorporationByID(ctx, corporationID)
	if err != nil && err != mongo.ErrNoDocuments {
		slog.ErrorContext(ctx, "Failed to get corporation from database", "error", err)
		// Continue to fetch from ESI
	}
	
	// If not found in database or data might be stale, fetch from ESI
	if corporation == nil || err == mongo.ErrNoDocuments {
		slog.InfoContext(ctx, "Corporation not found in database, fetching from ESI", "corporation_id", corporationID)
		
		// Get corporation info from EVE ESI
		esiData, err := s.eveClient.GetCorporationInfo(ctx, corporationID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get corporation from ESI", "error", err)
			return nil, fmt.Errorf("failed to get corporation information: %w", err)
		}
		
		// Convert ESI data to our model
		corporation = s.convertESIDataToModel(esiData, corporationID)
		
		// Save to database for future use
		if err := s.repository.UpdateCorporation(ctx, corporation); err != nil {
			slog.WarnContext(ctx, "Failed to save corporation to database", "error", err)
			// Don't fail the request if we can't save to DB
		}
	}
	
	// Convert model to output DTO
	return s.convertModelToOutput(corporation), nil
}

// convertESIDataToModel converts ESI response data to our corporation model
func (s *Service) convertESIDataToModel(esiData map[string]any, corporationID int) *models.Corporation {
	now := time.Now().UTC()
	corporation := &models.Corporation{
		CorporationID: corporationID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	
	if name, ok := esiData["name"].(string); ok {
		corporation.Name = name
	}
	if ticker, ok := esiData["ticker"].(string); ok {
		corporation.Ticker = ticker
	}
	if description, ok := esiData["description"].(string); ok {
		corporation.Description = description
	}
	if url, ok := esiData["url"].(string); ok && url != "" {
		corporation.URL = &url
	}
	// Handle date_founded
	if dateFounded, ok := esiData["date_founded"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, dateFounded); err == nil {
			corporation.DateFounded = parsedTime
		}
	}
	
	// Handle numeric fields - can be int, float64, or string depending on source
	if allianceID, ok := esiData["alliance_id"].(int); ok {
		corporation.AllianceID = &allianceID
	} else if allianceIDFloat, ok := esiData["alliance_id"].(float64); ok {
		allianceIDInt := int(allianceIDFloat)
		corporation.AllianceID = &allianceIDInt
	}
	
	if ceoID, ok := esiData["ceo_id"].(int); ok {
		corporation.CEOCharacterID = ceoID
	} else if ceoIDFloat, ok := esiData["ceo_id"].(float64); ok {
		corporation.CEOCharacterID = int(ceoIDFloat)
	}
	
	if creatorID, ok := esiData["creator_id"].(int); ok {
		corporation.CreatorID = creatorID
	} else if creatorIDFloat, ok := esiData["creator_id"].(float64); ok {
		corporation.CreatorID = int(creatorIDFloat)
	}
	if factionID, ok := esiData["faction_id"].(int); ok {
		corporation.FactionID = &factionID
	} else if factionIDFloat, ok := esiData["faction_id"].(float64); ok {
		factionIDInt := int(factionIDFloat)
		corporation.FactionID = &factionIDInt
	}
	if homeStationID, ok := esiData["home_station_id"].(int); ok {
		corporation.HomeStationID = &homeStationID
	} else if homeStationIDFloat, ok := esiData["home_station_id"].(float64); ok {
		homeStationIDInt := int(homeStationIDFloat)
		corporation.HomeStationID = &homeStationIDInt
	}
	if memberCount, ok := esiData["member_count"].(int); ok {
		corporation.MemberCount = memberCount
	} else if memberCountFloat, ok := esiData["member_count"].(float64); ok {
		corporation.MemberCount = int(memberCountFloat)
	}
	if shares, ok := esiData["shares"].(int64); ok {
		corporation.Shares = &shares
	} else if sharesInt, ok := esiData["shares"].(int); ok {
		sharesInt64 := int64(sharesInt)
		corporation.Shares = &sharesInt64
	} else if sharesFloat, ok := esiData["shares"].(float64); ok {
		sharesInt64 := int64(sharesFloat)
		corporation.Shares = &sharesInt64
	}
	if taxRate, ok := esiData["tax_rate"].(float64); ok {
		corporation.TaxRate = taxRate
	}
	if warEligible, ok := esiData["war_eligible"].(bool); ok {
		corporation.WarEligible = &warEligible
	}
	
	return corporation
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// convertModelToOutput converts corporation model to output DTO
func (s *Service) convertModelToOutput(corporation *models.Corporation) *dto.CorporationInfoOutput {
	corporationInfo := dto.CorporationInfo{
		AllianceID:      corporation.AllianceID,
		CEOCharacterID:  corporation.CEOCharacterID,
		CreatorID:       corporation.CreatorID,
		DateFounded:     corporation.DateFounded,
		Description:     corporation.Description,
		FactionID:       corporation.FactionID,
		HomeStationID:   corporation.HomeStationID,
		MemberCount:     corporation.MemberCount,
		Name:            corporation.Name,
		Shares:          corporation.Shares,
		TaxRate:         corporation.TaxRate,
		Ticker:          corporation.Ticker,
		URL:             corporation.URL,
		WarEligible:     corporation.WarEligible,
	}
	
	return &dto.CorporationInfoOutput{
		Body: corporationInfo,
	}
}

// SearchCorporationsByName searches corporations by name or ticker
func (s *Service) SearchCorporationsByName(ctx context.Context, name string) (*dto.SearchCorporationsByNameOutput, error) {
	slog.InfoContext(ctx, "Searching corporations by name", "name", name)
	
	corporations, err := s.repository.SearchCorporationsByName(ctx, name)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to search corporations", "error", err)
		return nil, fmt.Errorf("failed to search corporations: %w", err)
	}
	
	// Convert models to search DTOs
	searchResults := make([]dto.CorporationSearchInfo, len(corporations))
	for i, corp := range corporations {
		searchResults[i] = dto.CorporationSearchInfo{
			CorporationID: corp.CorporationID,
			Name:          corp.Name,
			Ticker:        corp.Ticker,
			MemberCount:   corp.MemberCount,
			AllianceID:    corp.AllianceID,
			UpdatedAt:     corp.UpdatedAt,
		}
	}
	
	result := &dto.SearchCorporationsByNameOutput{
		Body: dto.SearchCorporationsResult{
			Corporations: searchResults,
			Count:        len(searchResults),
		},
	}
	
	slog.InfoContext(ctx, "Corporation search completed", "count", len(searchResults))
	return result, nil
}

// UpdateAllCorporations updates all corporations in the database by fetching fresh data from ESI
func (s *Service) UpdateAllCorporations(ctx context.Context, concurrentWorkers int) error {
	slog.InfoContext(ctx, "Starting update of all corporations", "concurrent_workers", concurrentWorkers)
	
	// Get all corporation IDs from database
	corporationIDs, err := s.repository.GetAllCorporationIDs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get corporation IDs from database", "error", err)
		return fmt.Errorf("failed to get corporation IDs: %w", err)
	}
	
	totalCount := len(corporationIDs)
	if totalCount == 0 {
		slog.InfoContext(ctx, "No corporations found in database to update")
		return nil
	}
	
	slog.InfoContext(ctx, "Found corporations to update", "total_count", totalCount)
	
	// Create channels for work distribution
	type updateResult struct {
		corporationID int
		success       bool
		err           error
	}
	
	jobs := make(chan int, totalCount)
	results := make(chan updateResult, totalCount)
	
	// Start workers
	for w := 1; w <= concurrentWorkers; w++ {
		go func(workerID int) {
			for corporationID := range jobs {
				// Create a new context for each update to avoid context cancellation issues
				updateCtx := context.Background()
				
				// Get corporation info from EVE ESI
				esiData, err := s.eveClient.GetCorporationInfo(updateCtx, corporationID)
				if err != nil {
					slog.WarnContext(updateCtx, "Failed to get corporation from ESI", 
						"worker_id", workerID,
						"corporation_id", corporationID,
						"error", err)
					results <- updateResult{
						corporationID: corporationID,
						success:       false,
						err:           err,
					}
					continue
				}
				
				// Convert ESI data to our model
				corporation := s.convertESIDataToModel(esiData, corporationID)
				
				// Update in database
				if err := s.repository.UpdateCorporation(updateCtx, corporation); err != nil {
					slog.WarnContext(updateCtx, "Failed to update corporation in database",
						"worker_id", workerID,
						"corporation_id", corporationID,
						"error", err)
					results <- updateResult{
						corporationID: corporationID,
						success:       false,
						err:           err,
					}
					continue
				}
				
				slog.DebugContext(updateCtx, "Successfully updated corporation",
					"worker_id", workerID,
					"corporation_id", corporationID,
					"corporation_name", corporation.Name)
				
				results <- updateResult{
					corporationID: corporationID,
					success:       true,
					err:           nil,
				}
				
				// Small delay to respect ESI rate limits
				time.Sleep(50 * time.Millisecond)
			}
		}(w)
	}
	
	// Send all jobs
	for _, corporationID := range corporationIDs {
		jobs <- corporationID
	}
	close(jobs)
	
	// Collect results
	successCount := 0
	failureCount := 0
	
	for i := 0; i < totalCount; i++ {
		result := <-results
		if result.success {
			successCount++
		} else {
			failureCount++
		}
		
		// Log progress every 100 corporations
		if (i+1)%100 == 0 || (i+1) == totalCount {
			slog.InfoContext(ctx, "Corporation update progress",
				"processed", i+1,
				"total", totalCount,
				"success", successCount,
				"failures", failureCount,
				"progress_percent", fmt.Sprintf("%.1f%%", float64(i+1)/float64(totalCount)*100))
		}
	}
	
	slog.InfoContext(ctx, "Completed updating all corporations",
		"total_processed", totalCount,
		"successful", successCount,
		"failed", failureCount)
	
	if failureCount > 0 {
		return fmt.Errorf("failed to update %d out of %d corporations", failureCount, totalCount)
	}
	
	return nil
}

// GetStatus returns the health status of the corporation module
func (s *Service) GetStatus(ctx context.Context) *dto.CorporationStatusResponse {
	// Check database connectivity
	if err := s.repository.CheckHealth(ctx); err != nil {
		return &dto.CorporationStatusResponse{
			Module:  "corporation",
			Status:  "unhealthy",
			Message: "Database connection failed: " + err.Error(),
		}
	}

	return &dto.CorporationStatusResponse{
		Module: "corporation",
		Status: "healthy",
	}
}