package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go-falcon/internal/character/dto"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
)

const (
	// ESI_BATCH_SIZE is the maximum number of character IDs per ESI request
	ESI_BATCH_SIZE = 1000
	// PROCESSING_BATCH_SIZE is the number of characters to process per cycle
	PROCESSING_BATCH_SIZE = 5000
	// PARALLEL_WORKERS is the number of concurrent ESI requests (reduced to avoid rate limits)
	PARALLEL_WORKERS = 1
)

// UpdateService handles character affiliation updates
type UpdateService struct {
	repository *Repository
	eveGateway *evegateway.Client
}

// NewUpdateService creates a new update service instance
func NewUpdateService(mongodb *database.MongoDB, eveGateway *evegateway.Client) *UpdateService {
	return &UpdateService{
		repository: NewRepository(mongodb),
		eveGateway: eveGateway,
	}
}

// UpdateAllAffiliations updates affiliations for all characters in the database
func (s *UpdateService) UpdateAllAffiliations(ctx context.Context) (*dto.AffiliationUpdateStats, error) {
	startTime := time.Now()
	stats := &dto.AffiliationUpdateStats{}

	log.Printf("Starting character affiliation update process")

	// Get all character IDs from database
	characterIDs, err := s.repository.GetAllCharacterIDs(ctx)
	if err != nil {
		log.Printf("Error fetching character IDs: %v", err)
		return nil, fmt.Errorf("failed to fetch character IDs: %w", err)
	}

	stats.TotalCharacters = len(characterIDs)
	log.Printf("Found %d characters to update", stats.TotalCharacters)

	if stats.TotalCharacters == 0 {
		stats.Duration = int(time.Since(startTime).Seconds())
		return stats, nil
	}

	// Process characters in batches
	batches := s.createBatches(characterIDs, ESI_BATCH_SIZE)
	stats.BatchesProcessed = len(batches)

	// Create worker pool for parallel processing
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, PARALLEL_WORKERS)
	resultsChan := make(chan *batchResult, len(batches))

	for i, batch := range batches {
		wg.Add(1)
		go func(batchNum int, charIDs []int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Printf("Processing batch %d/%d with %d characters", batchNum+1, len(batches), len(charIDs))

			result := s.processBatch(ctx, charIDs)
			resultsChan <- result
		}(i, batch)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(resultsChan)

	// Aggregate results
	for result := range resultsChan {
		stats.UpdatedCharacters += result.Updated
		stats.FailedCharacters += result.Failed
		stats.SkippedCharacters += result.Skipped
	}

	stats.Duration = int(time.Since(startTime).Seconds())

	log.Printf("🎯 AFFILIATION UPDATE COMPLETED: %d updated, %d failed, %d skipped in %d seconds (processed %d batches)",
		stats.UpdatedCharacters, stats.FailedCharacters, stats.SkippedCharacters, stats.Duration, stats.BatchesProcessed)

	return stats, nil
}

// UpdateCharacterAffiliations updates affiliations for specific characters
func (s *UpdateService) UpdateCharacterAffiliations(ctx context.Context, characterIDs []int) (*dto.AffiliationUpdateStats, error) {
	startTime := time.Now()
	stats := &dto.AffiliationUpdateStats{
		TotalCharacters: len(characterIDs),
	}

	if len(characterIDs) == 0 {
		return stats, nil
	}

	log.Printf("Updating affiliations for %d characters", len(characterIDs))

	// Process in batches
	batches := s.createBatches(characterIDs, ESI_BATCH_SIZE)
	stats.BatchesProcessed = len(batches)

	for i, batch := range batches {
		log.Printf("Processing batch %d/%d", i+1, len(batches))
		result := s.processBatch(ctx, batch)
		stats.UpdatedCharacters += result.Updated
		stats.FailedCharacters += result.Failed
		stats.SkippedCharacters += result.Skipped
	}

	stats.Duration = int(time.Since(startTime).Seconds())
	return stats, nil
}

// batchResult represents the result of processing a batch
type batchResult struct {
	Updated int
	Failed  int
	Skipped int
}

// processBatch processes a single batch of character IDs
func (s *UpdateService) processBatch(ctx context.Context, characterIDs []int) *batchResult {
	result := &batchResult{}

	log.Printf("🌐 Calling ESI /characters/affiliation/ for %d character IDs", len(characterIDs))

	// Call ESI to get affiliations
	affiliations, err := s.eveGateway.Character.GetCharactersAffiliation(ctx, characterIDs)
	if err != nil {
		log.Printf("❌ ERROR fetching affiliations from ESI: %v", err)
		result.Failed = len(characterIDs)
		return result
	}

	log.Printf("✅ ESI returned %d affiliations for %d requested characters", len(affiliations), len(characterIDs))

	// Create a map for quick lookup
	affiliationMap := make(map[int]dto.CharacterAffiliation)
	for _, affMap := range affiliations {
		charID, ok := affMap["character_id"].(int)
		if !ok {
			// Try float64 conversion (JSON numbers are often floats)
			if charIDFloat, ok := affMap["character_id"].(float64); ok {
				charID = int(charIDFloat)
			} else {
				continue
			}
		}

		corpID, ok := affMap["corporation_id"].(int)
		if !ok {
			if corpIDFloat, ok := affMap["corporation_id"].(float64); ok {
				corpID = int(corpIDFloat)
			}
		}

		aff := dto.CharacterAffiliation{
			CharacterID:   charID,
			CorporationID: corpID,
		}

		if allianceID, ok := affMap["alliance_id"]; ok {
			if allianceIDInt, ok := allianceID.(int); ok {
				aff.AllianceID = allianceIDInt
			} else if allianceIDFloat, ok := allianceID.(float64); ok {
				aff.AllianceID = int(allianceIDFloat)
			}
		}

		if factionID, ok := affMap["faction_id"]; ok {
			if factionIDInt, ok := factionID.(int); ok {
				aff.FactionID = factionIDInt
			} else if factionIDFloat, ok := factionID.(float64); ok {
				aff.FactionID = int(factionIDFloat)
			}
		}

		affiliationMap[charID] = aff
	}

	// Update each character in the database
	for _, charID := range characterIDs {
		aff, found := affiliationMap[charID]
		if !found {
			log.Printf("⚠️  Character %d not found in ESI response, skipping", charID)
			result.Skipped++
			continue
		}

		// Update the character in the database
		if err := s.repository.UpdateCharacterAffiliation(ctx, &aff); err != nil {
			log.Printf("❌ Error updating character %d: %v", charID, err)
			result.Failed++
		} else {
			result.Updated++
		}
	}

	log.Printf("📊 Batch completed: %d updated, %d failed, %d skipped", result.Updated, result.Failed, result.Skipped)
	return result
}

// createBatches splits character IDs into batches
func (s *UpdateService) createBatches(characterIDs []int, batchSize int) [][]int {
	var batches [][]int

	for i := 0; i < len(characterIDs); i += batchSize {
		end := i + batchSize
		if end > len(characterIDs) {
			end = len(characterIDs)
		}
		batches = append(batches, characterIDs[i:end])
	}

	return batches
}

// GetCharacterAffiliation gets the current affiliation for a specific character
func (s *UpdateService) GetCharacterAffiliation(ctx context.Context, characterID int) (*dto.CharacterAffiliation, error) {
	// Try to get from database first
	character, err := s.repository.GetCharacterByID(ctx, characterID)
	if err != nil {
		return nil, err
	}

	if character != nil {
		return &dto.CharacterAffiliation{
			CharacterID:   character.CharacterID,
			CorporationID: character.CorporationID,
			AllianceID:    character.AllianceID,
			FactionID:     character.FactionID,
		}, nil
	}

	// Not in database, fetch from ESI
	affiliations, err := s.eveGateway.Character.GetCharactersAffiliation(ctx, []int{characterID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch affiliation from ESI: %w", err)
	}

	if len(affiliations) == 0 {
		return nil, fmt.Errorf("character %d not found", characterID)
	}

	affMap := affiliations[0]
	result := &dto.CharacterAffiliation{
		CharacterID: characterID,
	}

	if corpID, ok := affMap["corporation_id"].(float64); ok {
		result.CorporationID = int(corpID)
	} else if corpID, ok := affMap["corporation_id"].(int); ok {
		result.CorporationID = corpID
	}

	if allianceID, ok := affMap["alliance_id"].(float64); ok {
		result.AllianceID = int(allianceID)
	} else if allianceID, ok := affMap["alliance_id"].(int); ok {
		result.AllianceID = allianceID
	}

	if factionID, ok := affMap["faction_id"].(float64); ok {
		result.FactionID = int(factionID)
	} else if factionID, ok := affMap["faction_id"].(int); ok {
		result.FactionID = factionID
	}

	// Save to database for future use
	if err := s.repository.UpdateCharacterAffiliation(ctx, result); err != nil {
		log.Printf("Warning: failed to save affiliation to database: %v", err)
	}

	return result, nil
}

// RefreshCharacterAffiliation forces a refresh of a character's affiliation from ESI
func (s *UpdateService) RefreshCharacterAffiliation(ctx context.Context, characterID int) (*dto.CharacterAffiliation, error) {
	// Fetch from ESI
	affiliations, err := s.eveGateway.Character.GetCharactersAffiliation(ctx, []int{characterID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch affiliation from ESI: %w", err)
	}

	if len(affiliations) == 0 {
		return nil, fmt.Errorf("character %d not found", characterID)
	}

	affMap := affiliations[0]
	result := &dto.CharacterAffiliation{
		CharacterID: characterID,
	}

	if corpID, ok := affMap["corporation_id"].(float64); ok {
		result.CorporationID = int(corpID)
	} else if corpID, ok := affMap["corporation_id"].(int); ok {
		result.CorporationID = corpID
	}

	if allianceID, ok := affMap["alliance_id"].(float64); ok {
		result.AllianceID = int(allianceID)
	} else if allianceID, ok := affMap["alliance_id"].(int); ok {
		result.AllianceID = allianceID
	}

	if factionID, ok := affMap["faction_id"].(float64); ok {
		result.FactionID = int(factionID)
	} else if factionID, ok := affMap["faction_id"].(int); ok {
		result.FactionID = factionID
	}

	// Update in database
	if err := s.repository.UpdateCharacterAffiliation(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to update affiliation in database: %w", err)
	}

	return result, nil
}

// ValidateCharacters checks if characters exist in ESI
func (s *UpdateService) ValidateCharacters(ctx context.Context, characterIDs []int) ([]int, []int, error) {
	var validIDs, invalidIDs []int

	// Process in batches
	batches := s.createBatches(characterIDs, ESI_BATCH_SIZE)

	for _, batch := range batches {
		affiliations, err := s.eveGateway.Character.GetCharactersAffiliation(ctx, batch)
		if err != nil {
			// If the entire batch fails, consider all as invalid
			invalidIDs = append(invalidIDs, batch...)
			continue
		}

		// Create a set of returned character IDs
		returnedIDs := make(map[int]bool)
		for _, affMap := range affiliations {
			charID, ok := affMap["character_id"].(int)
			if !ok {
				if charIDFloat, ok := affMap["character_id"].(float64); ok {
					charID = int(charIDFloat)
				} else {
					continue
				}
			}
			returnedIDs[charID] = true
			validIDs = append(validIDs, charID)
		}

		// Check for missing IDs
		for _, id := range batch {
			if !returnedIDs[id] {
				invalidIDs = append(invalidIDs, id)
			}
		}
	}

	return validIDs, invalidIDs, nil
}
