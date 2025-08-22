package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/internal/alliance/dto"
	"go-falcon/internal/alliance/models"
	"go-falcon/pkg/evegateway"

	"go.mongodb.org/mongo-driver/mongo"
)

// Service handles alliance business logic
type Service struct {
	repository *Repository
	eveClient  *evegateway.Client
}

// NewService creates a new alliance service
func NewService(repository *Repository, eveClient *evegateway.Client) *Service {
	return &Service{
		repository: repository,
		eveClient:  eveClient,
	}
}

// GetAllianceInfo retrieves alliance information, first checking the database,
// then falling back to EVE ESI if not found or data is stale
func (s *Service) GetAllianceInfo(ctx context.Context, allianceID int) (*dto.AllianceInfoOutput, error) {
	slog.InfoContext(ctx, "Getting alliance info", "alliance_id", allianceID)
	
	// Try to get from database first
	alliance, err := s.repository.GetAllianceByID(ctx, allianceID)
	if err != nil && err != mongo.ErrNoDocuments {
		slog.ErrorContext(ctx, "Failed to get alliance from database", "error", err)
		// Continue to fetch from ESI
	}
	
	// If not found in database or data might be stale, fetch from ESI
	if alliance == nil || err == mongo.ErrNoDocuments {
		slog.InfoContext(ctx, "Alliance not found in database, fetching from ESI", "alliance_id", allianceID)
		
		// Get alliance info from EVE ESI
		esiData, err := s.eveClient.GetAllianceInfo(ctx, allianceID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get alliance from ESI", "error", err)
			return nil, fmt.Errorf("failed to get alliance information: %w", err)
		}
		
		// Convert ESI data to our model
		alliance = s.convertESIDataToModel(esiData, allianceID)
		
		// Save to database for future use
		if err := s.repository.UpdateAlliance(ctx, alliance); err != nil {
			slog.WarnContext(ctx, "Failed to save alliance to database", "error", err)
			// Don't fail the request if we can't save to DB
		}
	}
	
	// Convert model to output DTO
	return s.convertModelToOutput(alliance), nil
}

// convertESIDataToModel converts ESI response data to our alliance model
func (s *Service) convertESIDataToModel(esiData map[string]any, allianceID int) *models.Alliance {
	now := time.Now().UTC()
	alliance := &models.Alliance{
		AllianceID: allianceID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	
	
	if name, ok := esiData["name"].(string); ok {
		alliance.Name = name
	}
	if ticker, ok := esiData["ticker"].(string); ok {
		alliance.Ticker = ticker
	}
	
	// Handle date_founded - could be time.Time from alliance client or string from direct ESI
	if dateFounded, ok := esiData["date_founded"].(time.Time); ok {
		alliance.DateFounded = dateFounded
	} else if dateFoundedStr, ok := esiData["date_founded"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, dateFoundedStr); err == nil {
			alliance.DateFounded = parsedTime
		}
	}
	
	// Handle numeric fields - alliance client returns int64, fallback to float64/int
	if creatorCorpIDInt64, ok := esiData["creator_corporation_id"].(int64); ok {
		alliance.CreatorCorporationID = int(creatorCorpIDInt64)
	} else if creatorCorpIDFloat, ok := esiData["creator_corporation_id"].(float64); ok {
		alliance.CreatorCorporationID = int(creatorCorpIDFloat)
	} else if creatorCorpID, ok := esiData["creator_corporation_id"].(int); ok {
		alliance.CreatorCorporationID = creatorCorpID
	}
	
	if creatorCharIDInt64, ok := esiData["creator_id"].(int64); ok {
		alliance.CreatorCharacterID = int(creatorCharIDInt64)
	} else if creatorCharIDFloat, ok := esiData["creator_id"].(float64); ok {
		alliance.CreatorCharacterID = int(creatorCharIDFloat)
	} else if creatorCharID, ok := esiData["creator_id"].(int); ok {
		alliance.CreatorCharacterID = creatorCharID
	}
	
	// Handle executor_corporation_id - can be *int64 from alliance client
	if executorCorpIDPtr, ok := esiData["executor_corporation_id"].(*int64); ok && executorCorpIDPtr != nil {
		executorCorpIDInt := int(*executorCorpIDPtr)
		alliance.ExecutorCorporationID = &executorCorpIDInt
	} else if executorCorpIDInt64, ok := esiData["executor_corporation_id"].(int64); ok {
		executorCorpIDInt := int(executorCorpIDInt64)
		alliance.ExecutorCorporationID = &executorCorpIDInt
	} else if executorCorpIDFloat, ok := esiData["executor_corporation_id"].(float64); ok {
		executorCorpIDInt := int(executorCorpIDFloat)
		alliance.ExecutorCorporationID = &executorCorpIDInt
	} else if executorCorpID, ok := esiData["executor_corporation_id"].(int); ok {
		alliance.ExecutorCorporationID = &executorCorpID
	}
	
	// Handle faction_id - check all possible types: *int64, int64, float64, int
	if esiData["faction_id"] != nil {
		if factionIDPtr, ok := esiData["faction_id"].(*int64); ok && factionIDPtr != nil {
			factionIDInt := int(*factionIDPtr)
			alliance.FactionID = &factionIDInt
		} else if factionIDInt64, ok := esiData["faction_id"].(int64); ok {
			factionIDInt := int(factionIDInt64)
			alliance.FactionID = &factionIDInt
		} else if factionIDFloat, ok := esiData["faction_id"].(float64); ok {
			factionIDInt := int(factionIDFloat)
			alliance.FactionID = &factionIDInt
		} else if factionID, ok := esiData["faction_id"].(int); ok {
			alliance.FactionID = &factionID
		}
	}
	
	return alliance
}

// convertModelToOutput converts alliance model to output DTO according to ESI specification
func (s *Service) convertModelToOutput(alliance *models.Alliance) *dto.AllianceInfoOutput {
	allianceInfo := dto.AllianceInfo{
		Name:                  alliance.Name,
		CreatorID:             alliance.CreatorCharacterID, // Map CreatorCharacterID to CreatorID per ESI spec
		CreatorCorporationID:  alliance.CreatorCorporationID,
		Ticker:                alliance.Ticker,
		DateFounded:           alliance.DateFounded,
		ExecutorCorporationID: alliance.ExecutorCorporationID,
		FactionID:             alliance.FactionID,
	}
	
	return &dto.AllianceInfoOutput{
		Body: allianceInfo,
	}
}

// GetAllAlliances retrieves list of all active alliance IDs from ESI
func (s *Service) GetAllAlliances(ctx context.Context) (*dto.AllianceListOutput, error) {
	slog.InfoContext(ctx, "Getting all alliances list from ESI")
	
	// Get all alliance IDs from EVE ESI
	allianceIDs, err := s.eveClient.Alliance.GetAlliances(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get alliances from ESI", "error", err)
		return nil, fmt.Errorf("failed to get alliances: %w", err)
	}
	
	slog.InfoContext(ctx, "Successfully retrieved alliances list", "count", len(allianceIDs))
	
	return &dto.AllianceListOutput{
		Body: allianceIDs,
	}, nil
}

// GetAllianceCorporations retrieves list of corporation IDs that are members of the specified alliance
func (s *Service) GetAllianceCorporations(ctx context.Context, allianceID int) (*dto.AllianceCorporationsOutput, error) {
	slog.InfoContext(ctx, "Getting alliance member corporations", "alliance_id", allianceID)
	
	// Get alliance member corporations from EVE ESI
	corporationIDs, err := s.eveClient.Alliance.GetAllianceCorporations(ctx, int64(allianceID))
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get alliance corporations from ESI", "error", err, "alliance_id", allianceID)
		return nil, fmt.Errorf("failed to get alliance corporations: %w", err)
	}
	
	slog.InfoContext(ctx, "Successfully retrieved alliance corporations", "alliance_id", allianceID, "count", len(corporationIDs))
	
	return &dto.AllianceCorporationsOutput{
		Body: corporationIDs,
	}, nil
}

// SearchAlliancesByName searches alliances by name or ticker
func (s *Service) SearchAlliancesByName(ctx context.Context, name string) (*dto.SearchAlliancesByNameOutput, error) {
	slog.InfoContext(ctx, "Searching alliances by name", "name", name)
	
	alliances, err := s.repository.SearchAlliancesByName(ctx, name)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to search alliances", "error", err)
		return nil, fmt.Errorf("failed to search alliances: %w", err)
	}
	
	// Convert models to search DTOs
	searchResults := make([]dto.AllianceSearchInfo, len(alliances))
	for i, alliance := range alliances {
		searchResults[i] = dto.AllianceSearchInfo{
			AllianceID:            alliance.AllianceID,
			Name:                  alliance.Name,
			Ticker:                alliance.Ticker,
			ExecutorCorporationID: alliance.ExecutorCorporationID,
			DateFounded:           alliance.DateFounded,
			UpdatedAt:             alliance.UpdatedAt,
		}
	}
	
	result := &dto.SearchAlliancesByNameOutput{
		Body: dto.SearchAlliancesResult{
			Alliances: searchResults,
			Count:     len(searchResults),
		},
	}
	
	slog.InfoContext(ctx, "Alliance search completed", "count", len(searchResults))
	return result, nil
}

// BulkImportAlliances retrieves all alliance IDs from ESI and imports detailed information for each
func (s *Service) BulkImportAlliances(ctx context.Context) (*dto.BulkImportAlliancesOutput, error) {
	slog.InfoContext(ctx, "Starting bulk alliance import operation")
	
	stats := &dto.BulkImportStats{}
	
	// Step 1: Get all alliance IDs from ESI
	slog.InfoContext(ctx, "Fetching all alliance IDs from ESI")
	allianceIDs, err := s.eveClient.Alliance.GetAlliances(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get alliance list from ESI", "error", err)
		return nil, fmt.Errorf("failed to get alliance list: %w", err)
	}
	
	stats.TotalAlliances = len(allianceIDs)
	slog.InfoContext(ctx, "Retrieved alliance IDs", "count", stats.TotalAlliances)
	
	// Step 2: Process each alliance with rate limiting
	const batchSize = 10 // Process in batches to avoid overwhelming ESI
	const delayBetweenRequests = 200 * time.Millisecond // Respect ESI rate limits
	
	for i, allianceID := range allianceIDs {
		// Add delay between requests to respect ESI rate limits
		if i > 0 && i%batchSize == 0 {
			slog.InfoContext(ctx, "Batch completed, pausing for rate limiting", "processed", i, "total", stats.TotalAlliances)
			time.Sleep(delayBetweenRequests * 5) // Longer pause between batches
		} else if i > 0 {
			time.Sleep(delayBetweenRequests)
		}
		
		stats.Processed++
		
		// Check if alliance already exists in database
		existingAlliance, err := s.repository.GetAllianceByID(ctx, int(allianceID))
		isUpdate := (existingAlliance != nil && err == nil)
		
		// Fetch alliance information from ESI
		esiData, err := s.eveClient.GetAllianceInfo(ctx, int(allianceID))
		if err != nil {
			stats.Failed++
			slog.WarnContext(ctx, "Failed to fetch alliance from ESI", 
				"alliance_id", allianceID, "error", err, 
				"progress", fmt.Sprintf("%d/%d", stats.Processed, stats.TotalAlliances))
			continue
		}
		
		// Convert ESI data to alliance model
		alliance := s.convertESIDataToModel(esiData, int(allianceID))
		
		// Save/update alliance in database
		if err := s.repository.UpdateAlliance(ctx, alliance); err != nil {
			stats.Failed++
			slog.ErrorContext(ctx, "Failed to save alliance to database", 
				"alliance_id", allianceID, "error", err)
			continue
		}
		
		// Update statistics
		if isUpdate {
			stats.Updated++
			slog.DebugContext(ctx, "Updated alliance", 
				"alliance_id", allianceID, "name", alliance.Name, "ticker", alliance.Ticker,
				"progress", fmt.Sprintf("%d/%d", stats.Processed, stats.TotalAlliances))
		} else {
			stats.Created++
			slog.InfoContext(ctx, "Created alliance", 
				"alliance_id", allianceID, "name", alliance.Name, "ticker", alliance.Ticker,
				"progress", fmt.Sprintf("%d/%d", stats.Processed, stats.TotalAlliances))
		}
		
		// Log progress every 50 alliances
		if stats.Processed%50 == 0 {
			slog.InfoContext(ctx, "Bulk import progress", 
				"processed", stats.Processed, "total", stats.TotalAlliances,
				"created", stats.Created, "updated", stats.Updated, "failed", stats.Failed)
		}
	}
	
	slog.InfoContext(ctx, "Bulk alliance import completed", 
		"total", stats.TotalAlliances, "processed", stats.Processed,
		"created", stats.Created, "updated", stats.Updated, 
		"failed", stats.Failed, "skipped", stats.Skipped)
	
	return &dto.BulkImportAlliancesOutput{
		Body: *stats,
	}, nil
}