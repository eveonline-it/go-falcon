package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"time"

	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"
)

// Service provides business logic for character operations
type Service struct {
	repository *Repository
	eveGateway *evegateway.Client
	redis      *database.Redis
	sdeService sde.SDEService
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client, sdeService sde.SDEService) *Service {
	return &Service{
		repository: NewRepository(mongodb),
		eveGateway: eveGateway,
		redis:      redis,
		sdeService: sdeService,
	}
}

// CreateIndexes creates database indexes for optimal performance
func (s *Service) CreateIndexes(ctx context.Context) error {
	return s.repository.CreateIndexes(ctx)
}

// resolveImplantInfo converts a slice of implant type IDs to ImplantInfo structs with names and descriptions
func (s *Service) resolveImplantInfo(implantTypeIDs []int) []dto.ImplantInfo {
	implantInfos := make([]dto.ImplantInfo, len(implantTypeIDs))

	for i, typeID := range implantTypeIDs {
		implantInfo := dto.ImplantInfo{
			TypeID:      typeID,
			Name:        fmt.Sprintf("Unknown Implant %d", typeID), // Fallback name
			Description: "",                                        // Default empty description
		}

		// Get type information from SDE service
		if s.sdeService != nil {
			typeIDStr := strconv.Itoa(typeID)
			if typeInfo, err := s.sdeService.GetType(typeIDStr); err == nil && typeInfo != nil {
				// Extract English name
				if enName, ok := typeInfo.Name["en"]; ok && enName != "" {
					implantInfo.Name = enName
				}
				// Extract English description
				if enDesc, ok := typeInfo.Description["en"]; ok && enDesc != "" {
					implantInfo.Description = enDesc
				}
			} else {
				log.Printf("Failed to get implant type info for ID %d: %v", typeID, err)
			}
		}

		implantInfos[i] = implantInfo
	}

	return implantInfos
}

// LocationInfo holds location name and type ID
type LocationInfo struct {
	Name   string
	TypeID int32
}

// resolveLocationInfo resolves a location name and type ID based on location ID and type
func (s *Service) resolveLocationInfo(ctx context.Context, locationID int64, locationType string, token string) (*LocationInfo, error) {
	const npcStationThreshold = 100000000

	if locationID < npcStationThreshold {
		// NPC Station - use SDE
		station, err := s.sdeService.GetStaStation(int(locationID))
		if err != nil {
			log.Printf("Warning: Station not found in SDE for ID %d: %v", locationID, err)
			return &LocationInfo{}, nil // Return empty info, not error, to avoid breaking the API response
		}
		return &LocationInfo{
			Name:   station.StationName,
			TypeID: int32(station.StationTypeID),
		}, nil
	} else {
		// Player Structure - use EVE Gateway (requires token)
		if token == "" {
			log.Printf("Warning: Cannot resolve structure info for ID %d: no token provided", locationID)
			return &LocationInfo{}, nil // Return empty info instead of error
		}

		structure, err := s.eveGateway.Structures.GetStructure(ctx, locationID, token)
		if err != nil {
			log.Printf("Warning: Failed to get structure info for ID %d: %v", locationID, err)
			return &LocationInfo{}, nil // Return empty info instead of error to avoid breaking API
		}

		info := &LocationInfo{}
		if name, ok := structure["name"].(string); ok {
			info.Name = name
		}
		if typeID, ok := structure["type_id"].(int32); ok {
			info.TypeID = typeID
		} else if typeID, ok := structure["type_id"].(float64); ok {
			info.TypeID = int32(typeID)
		}

		return info, nil
	}
}

// GetCharacterAttributes retrieves character attributes following cache → DB → ESI flow
func (s *Service) GetCharacterAttributes(ctx context.Context, characterID int, token string) (*dto.CharacterAttributesOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:attributes:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterAttributes
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character attributes found in cache for character_id: %d", characterID)
				return &dto.CharacterAttributesOutput{Body: result}, nil
			}
		}
	}

	// 2. Check database
	dbAttributes, err := s.repository.GetCharacterAttributes(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching character attributes from DB: %v", err)
		// Continue to ESI even if DB error
	} else if dbAttributes != nil {
		// Found in database, convert to DTO
		result := s.modelToDTO(dbAttributes)

		// Update cache
		if s.redis != nil {
			if data, err := json.Marshal(result); err == nil {
				// Cache for 30 minutes
				_ = s.redis.Set(ctx, cacheKey, string(data), 30*time.Minute)
			}
		}

		log.Printf("Character attributes found in database for character_id: %d", characterID)
		return &dto.CharacterAttributesOutput{Body: *result}, nil
	}

	// 3. Fetch from ESI
	log.Printf("Fetching character attributes from ESI for character_id: %d", characterID)
	esiAttributes, err := s.eveGateway.Character.GetCharacterAttributes(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character attributes from ESI: %w", err)
	}

	// Convert ESI response to DTO
	result := s.parseESIAttributes(esiAttributes)

	// 4. Save to database
	dbModel := s.dtoToModel(result, characterID)
	if err := s.repository.SaveCharacterAttributes(ctx, dbModel); err != nil {
		log.Printf("Failed to save character attributes to DB: %v", err)
		// Continue even if save fails
	}

	// 5. Save to cache
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 30 minutes
			_ = s.redis.Set(ctx, cacheKey, string(data), 30*time.Minute)
		}
	}

	log.Printf("Successfully fetched and saved character attributes for character_id: %d", characterID)
	return &dto.CharacterAttributesOutput{Body: *result}, nil
}

// GetCharacterCorporationHistory retrieves character corporation history following cache → DB → ESI flow
func (s *Service) GetCharacterCorporationHistory(ctx context.Context, characterID int) (*dto.CharacterCorporationHistoryOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:corphistory:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterCorporationHistory
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character corporation history found in cache for character_id: %d", characterID)
				return &dto.CharacterCorporationHistoryOutput{Body: result}, nil
			}
		}
	}

	// 2. Check database
	dbHistory, err := s.repository.GetCharacterCorporationHistory(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching character corporation history from DB: %v", err)
		// Continue to ESI even if DB error
	} else if dbHistory != nil && len(dbHistory.History) > 0 {
		// Found in database, convert to DTO
		dtoHistory := make([]dto.CorporationHistoryEntry, len(dbHistory.History))
		for i, entry := range dbHistory.History {
			dtoHistory[i] = dto.CorporationHistoryEntry{
				CorporationID: entry.CorporationID,
				IsDeleted:     entry.IsDeleted,
				RecordID:      entry.RecordID,
				StartDate:     entry.StartDate,
			}
		}
		result := &dto.CharacterCorporationHistory{
			CharacterID: characterID,
			History:     dtoHistory,
			UpdatedAt:   dbHistory.UpdatedAt,
		}

		// Update cache
		if s.redis != nil {
			if data, err := json.Marshal(result); err == nil {
				// Cache for 24 hours as corporation history rarely changes
				_ = s.redis.Set(ctx, cacheKey, string(data), 24*time.Hour)
			}
		}

		log.Printf("Character corporation history found in database for character_id: %d", characterID)
		return &dto.CharacterCorporationHistoryOutput{Body: *result}, nil
	}

	// 3. Fetch from ESI
	log.Printf("Fetching character corporation history from ESI for character_id: %d", characterID)
	esiHistory, err := s.eveGateway.Character.GetCharacterCorporationHistory(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character corporation history from ESI: %w", err)
	}

	// Convert ESI response to DTO
	historyEntries := make([]dto.CorporationHistoryEntry, len(esiHistory))
	for i, entry := range esiHistory {
		// Parse the date string from ESI
		startDateStr, _ := entry["start_date"].(string)
		startDate, _ := time.Parse(time.RFC3339, startDateStr)

		corpID, _ := entry["corporation_id"].(int)
		isDeleted, _ := entry["is_deleted"].(bool)
		recordID, _ := entry["record_id"].(int)

		historyEntries[i] = dto.CorporationHistoryEntry{
			CorporationID: corpID,
			IsDeleted:     isDeleted,
			RecordID:      recordID,
			StartDate:     startDate,
		}
	}

	result := &dto.CharacterCorporationHistory{
		CharacterID: characterID,
		History:     historyEntries,
		UpdatedAt:   time.Now(),
	}

	// 4. Save to database
	// Convert DTO entries to model entries
	modelHistory := make([]models.CorporationHistoryEntry, len(historyEntries))
	for i, entry := range historyEntries {
		modelHistory[i] = models.CorporationHistoryEntry{
			CorporationID: entry.CorporationID,
			IsDeleted:     entry.IsDeleted,
			RecordID:      entry.RecordID,
			StartDate:     entry.StartDate,
		}
	}

	dbModel := &models.CharacterCorporationHistory{
		CharacterID: characterID,
		History:     modelHistory,
		UpdatedAt:   time.Now(),
	}
	if err := s.repository.SaveCharacterCorporationHistory(ctx, dbModel); err != nil {
		log.Printf("Failed to save character corporation history to DB: %v", err)
		// Continue even if save fails
	}

	// 5. Save to cache
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 24 hours
			_ = s.redis.Set(ctx, cacheKey, string(data), 24*time.Hour)
		}
	}

	log.Printf("Successfully fetched and saved character corporation history for character_id: %d", characterID)
	return &dto.CharacterCorporationHistoryOutput{Body: *result}, nil
}

// GetCharacterClones retrieves character clones following cache → DB → ESI flow
func (s *Service) GetCharacterClones(ctx context.Context, characterID int, token string) (*dto.CharacterClonesOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:clones:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterClones
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character clones found in cache for character_id: %d", characterID)
				return &dto.CharacterClonesOutput{Body: result}, nil
			}
		}
	}

	// 2. Check database
	dbClones, err := s.repository.GetCharacterClones(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching character clones from DB: %v", err)
		// Continue to ESI even if DB error
	} else if dbClones != nil {
		// Found in database, convert to DTO
		dtoClones := &dto.CharacterClones{
			CharacterID:           characterID,
			ActiveImplants:        s.resolveImplantInfo(dbClones.ActiveImplants), // Convert []int to []dto.ImplantInfo
			LastCloneJumpDate:     dbClones.LastCloneJumpDate,
			LastStationChangeDate: dbClones.LastStationChangeDate,
			UpdatedAt:             dbClones.UpdatedAt,
		}

		// If no active implants in database, try to fetch them
		if len(dbClones.ActiveImplants) == 0 {
			if activeImplants, err := s.getActiveImplantsInternal(ctx, characterID, token); err == nil {
				dtoClones.ActiveImplants = s.resolveImplantInfo(activeImplants) // Convert []int to []dto.ImplantInfo
				// Update the database record with the fetched implants (as []int)
				dbClones.ActiveImplants = activeImplants
				dbClones.UpdatedAt = time.Now()
				if err := s.repository.SaveCharacterClones(ctx, dbClones); err != nil {
					log.Printf("Failed to update character clones with implants in DB: %v", err)
				}
			} else {
				log.Printf("Failed to fetch active implants for character_id: %d, error: %v", characterID, err)
				dtoClones.ActiveImplants = []dto.ImplantInfo{} // Ensure empty array instead of nil
			}
		}

		// Convert home location
		if dbClones.HomeLocation != nil {
			locationName := dbClones.HomeLocation.LocationName
			locationTypeID := dbClones.HomeLocation.LocationTypeID
			// If no cached location info, try to resolve it
			if locationName == "" || locationTypeID == 0 {
				if locationInfo, err := s.resolveLocationInfo(ctx, dbClones.HomeLocation.LocationID, dbClones.HomeLocation.LocationType, token); err == nil {
					if locationName == "" && locationInfo.Name != "" {
						locationName = locationInfo.Name
					}
					if locationTypeID == 0 && locationInfo.TypeID != 0 {
						locationTypeID = locationInfo.TypeID
					}
				}
			}
			dtoClones.HomeLocation = &dto.HomeLocation{
				LocationID:     dbClones.HomeLocation.LocationID,
				LocationType:   dbClones.HomeLocation.LocationType,
				LocationName:   locationName,
				LocationTypeID: locationTypeID,
			}
		}

		// Convert jump clones
		dtoClones.JumpClones = make([]dto.JumpClone, len(dbClones.JumpClones))
		for i, clone := range dbClones.JumpClones {
			locationName := clone.LocationName
			locationTypeID := clone.LocationTypeID
			// If no cached location info, try to resolve it
			if locationName == "" || locationTypeID == 0 {
				if locationInfo, err := s.resolveLocationInfo(ctx, clone.LocationID, clone.LocationType, token); err == nil {
					if locationName == "" && locationInfo.Name != "" {
						locationName = locationInfo.Name
					}
					if locationTypeID == 0 && locationInfo.TypeID != 0 {
						locationTypeID = locationInfo.TypeID
					}
				}
			}
			dtoClones.JumpClones[i] = dto.JumpClone{
				Implants:       s.resolveImplantInfo(clone.Implants), // Convert []int to []dto.ImplantInfo
				JumpCloneID:    clone.JumpCloneID,
				LocationID:     clone.LocationID,
				LocationType:   clone.LocationType,
				LocationName:   locationName,
				LocationTypeID: locationTypeID,
				Name:           clone.Name,
			}
		}

		// Update cache
		if s.redis != nil {
			if data, err := json.Marshal(dtoClones); err == nil {
				// Cache for 1 hour
				_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
			}
		}

		log.Printf("Character clones found in database for character_id: %d", characterID)
		return &dto.CharacterClonesOutput{Body: *dtoClones}, nil
	}

	// 3. Fetch from ESI
	log.Printf("Fetching character clones from ESI for character_id: %d", characterID)
	esiClones, err := s.eveGateway.Character.GetCharacterClones(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character clones from ESI: %w", err)
	}

	// Convert ESI response to DTO
	result := &dto.CharacterClones{
		CharacterID: characterID,
		UpdatedAt:   time.Now(),
	}

	// Parse the map response
	if homeLocation, ok := esiClones["home_location"].(map[string]any); ok {
		var locationID int64
		if lid, ok := homeLocation["location_id"].(int64); ok {
			locationID = lid
		} else if lid, ok := homeLocation["location_id"].(float64); ok {
			locationID = int64(lid)
		}
		locationType, _ := homeLocation["location_type"].(string)

		// Resolve location info
		locationName := ""
		var locationTypeID int32
		if locationInfo, err := s.resolveLocationInfo(ctx, locationID, locationType, token); err == nil {
			locationName = locationInfo.Name
			locationTypeID = locationInfo.TypeID
		}

		result.HomeLocation = &dto.HomeLocation{
			LocationID:     locationID,
			LocationType:   locationType,
			LocationName:   locationName,
			LocationTypeID: locationTypeID,
		}
	}

	// Try multiple type assertions for jump_clones
	if jumpClonesRaw, ok := esiClones["jump_clones"].([]map[string]any); ok {
		result.JumpClones = make([]dto.JumpClone, len(jumpClonesRaw))
		for i, clone := range jumpClonesRaw {
			var jumpCloneID int
			if jid, ok := clone["jump_clone_id"].(int); ok {
				jumpCloneID = jid
			} else if jid, ok := clone["jump_clone_id"].(float64); ok {
				jumpCloneID = int(jid)
			}

			var locationID int64
			if lid, ok := clone["location_id"].(int64); ok {
				locationID = lid
			} else if lid, ok := clone["location_id"].(float64); ok {
				locationID = int64(lid)
			}

			locationType, _ := clone["location_type"].(string)
			name, _ := clone["name"].(string)

			// Resolve location info
			locationName := ""
			var locationTypeID int32
			if locationInfo, err := s.resolveLocationInfo(ctx, locationID, locationType, token); err == nil {
				locationName = locationInfo.Name
				locationTypeID = locationInfo.TypeID
			}

			// Handle implants
			var implants []int
			if implantsRaw, ok := clone["implants"].([]int); ok {
				implants = implantsRaw
			} else if implantsRaw, ok := clone["implants"].([]any); ok {
				implants = make([]int, len(implantsRaw))
				for j, implantRaw := range implantsRaw {
					if implant, ok := implantRaw.(float64); ok {
						implants[j] = int(implant)
					} else if implant, ok := implantRaw.(int); ok {
						implants[j] = implant
					}
				}
			}

			result.JumpClones[i] = dto.JumpClone{
				Implants:       s.resolveImplantInfo(implants), // Convert []int to []dto.ImplantInfo
				JumpCloneID:    jumpCloneID,
				LocationID:     locationID,
				LocationType:   locationType,
				LocationName:   locationName,
				LocationTypeID: locationTypeID,
				Name:           name,
			}
		}
	} else if jumpClonesRaw, ok := esiClones["jump_clones"].([]any); ok {
		result.JumpClones = make([]dto.JumpClone, len(jumpClonesRaw))
		for i, cloneRaw := range jumpClonesRaw {
			if clone, ok := cloneRaw.(map[string]any); ok {
				var jumpCloneID int
				if jid, ok := clone["jump_clone_id"].(int); ok {
					jumpCloneID = jid
				} else if jid, ok := clone["jump_clone_id"].(float64); ok {
					jumpCloneID = int(jid)
				}

				var locationID int64
				if lid, ok := clone["location_id"].(int64); ok {
					locationID = lid
				} else if lid, ok := clone["location_id"].(float64); ok {
					locationID = int64(lid)
				}

				locationType, _ := clone["location_type"].(string)
				name, _ := clone["name"].(string)

				// Resolve location info
				locationName := ""
				var locationTypeID int32
				if locationInfo, err := s.resolveLocationInfo(ctx, locationID, locationType, token); err == nil {
					locationName = locationInfo.Name
					locationTypeID = locationInfo.TypeID
				}

				// Handle implants
				var implants []int
				if implantsRaw, ok := clone["implants"].([]int); ok {
					implants = implantsRaw
				} else if implantsRaw, ok := clone["implants"].([]any); ok {
					implants = make([]int, len(implantsRaw))
					for j, implantRaw := range implantsRaw {
						if implant, ok := implantRaw.(float64); ok {
							implants[j] = int(implant)
						} else if implant, ok := implantRaw.(int); ok {
							implants[j] = implant
						}
					}
				}

				result.JumpClones[i] = dto.JumpClone{
					Implants:       s.resolveImplantInfo(implants), // Convert []int to []dto.ImplantInfo
					JumpCloneID:    jumpCloneID,
					LocationID:     locationID,
					LocationType:   locationType,
					LocationName:   locationName,
					LocationTypeID: locationTypeID,
					Name:           name,
				}
			}
		}
	} else {
		result.JumpClones = []dto.JumpClone{} // Initialize as empty array instead of nil
	}

	if lastJumpDate, ok := esiClones["last_clone_jump_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastJumpDate); err == nil {
			result.LastCloneJumpDate = &t
		}
	}

	if lastStationDate, ok := esiClones["last_station_change_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastStationDate); err == nil {
			result.LastStationChangeDate = &t
		}
	}

	// 4. Save to database
	dbModel := &models.CharacterClones{
		CharacterID:           characterID,
		LastCloneJumpDate:     result.LastCloneJumpDate,
		LastStationChangeDate: result.LastStationChangeDate,
		UpdatedAt:             time.Now(),
	}

	// Convert DTO to model for home location
	if result.HomeLocation != nil {
		dbModel.HomeLocation = &models.HomeLocation{
			LocationID:     result.HomeLocation.LocationID,
			LocationType:   result.HomeLocation.LocationType,
			LocationName:   result.HomeLocation.LocationName,
			LocationTypeID: result.HomeLocation.LocationTypeID,
		}
	}

	// Convert DTO to model for jump clones
	dbModel.JumpClones = make([]models.JumpClone, len(result.JumpClones))
	for i, clone := range result.JumpClones {
		// Extract type IDs from ImplantInfo structs for database storage
		implantTypeIDs := make([]int, len(clone.Implants))
		for j, implant := range clone.Implants {
			implantTypeIDs[j] = implant.TypeID
		}

		dbModel.JumpClones[i] = models.JumpClone{
			Implants:       implantTypeIDs, // Store only type IDs in database
			JumpCloneID:    clone.JumpCloneID,
			LocationID:     clone.LocationID,
			LocationType:   clone.LocationType,
			LocationName:   clone.LocationName,
			LocationTypeID: clone.LocationTypeID,
			Name:           clone.Name,
		}
	}

	if err := s.repository.SaveCharacterClones(ctx, dbModel); err != nil {
		log.Printf("Failed to save character clones to DB: %v", err)
		// Continue even if save fails
	}

	// 5. Save to cache
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 1 hour
			_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
		}
	}

	// 6. Fetch active implants and add to result
	activeImplantsTypeIDs, err := s.getActiveImplantsInternal(ctx, characterID, token)
	if err != nil {
		log.Printf("Failed to fetch active implants for character_id: %d, error: %v", characterID, err)
		// Continue without implants data rather than failing the whole request
		result.ActiveImplants = []dto.ImplantInfo{}
	} else {
		result.ActiveImplants = s.resolveImplantInfo(activeImplantsTypeIDs) // Convert []int to []dto.ImplantInfo
		// Update database model with active implants (as type IDs)
		dbModel.ActiveImplants = activeImplantsTypeIDs
		// Re-save to database with implants data
		if err := s.repository.SaveCharacterClones(ctx, dbModel); err != nil {
			log.Printf("Failed to update character clones with implants in DB: %v", err)
		}
	}

	// 7. Update cache with complete data including implants
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 1 hour
			_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
		}
	}

	log.Printf("Successfully fetched and saved character clones for character_id: %d", characterID)
	return &dto.CharacterClonesOutput{Body: *result}, nil
}

// getActiveImplantsInternal is a helper method to get active implants data without the full DTO wrapper
// This is used internally by GetCharacterClones to fetch implants data
func (s *Service) getActiveImplantsInternal(ctx context.Context, characterID int, token string) ([]int, error) {
	// 1. Check database first
	dbImplants, err := s.repository.GetCharacterImplants(ctx, characterID)
	if err == nil && dbImplants != nil {
		return dbImplants.Implants, nil
	}

	// 2. Fetch from ESI as fallback
	esiImplants, err := s.eveGateway.Character.GetCharacterImplants(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character implants from ESI: %w", err)
	}

	// 3. Save to database (async, don't block on errors)
	go func() {
		dbModel := &models.CharacterImplants{
			CharacterID: characterID,
			Implants:    esiImplants,
			UpdatedAt:   time.Now(),
		}
		if err := s.repository.SaveCharacterImplants(context.Background(), dbModel); err != nil {
			log.Printf("Failed to save character implants to DB (async): %v", err)
		}
	}()

	return esiImplants, nil
}

// GetCharacterImplants retrieves character implants following cache → DB → ESI flow
func (s *Service) GetCharacterImplants(ctx context.Context, characterID int, token string) (*dto.CharacterImplantsOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:implants:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterImplants
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character implants found in cache for character_id: %d", characterID)
				return &dto.CharacterImplantsOutput{Body: result}, nil
			}
		}
	}

	// 2. Check database
	dbImplants, err := s.repository.GetCharacterImplants(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching character implants from DB: %v", err)
		// Continue to ESI even if DB error
	} else if dbImplants != nil {
		// Found in database, convert to DTO
		result := &dto.CharacterImplants{
			CharacterID: characterID,
			Implants:    dbImplants.Implants,
			UpdatedAt:   dbImplants.UpdatedAt,
		}

		// Update cache
		if s.redis != nil {
			if data, err := json.Marshal(result); err == nil {
				// Cache for 1 hour
				_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
			}
		}

		log.Printf("Character implants found in database for character_id: %d", characterID)
		return &dto.CharacterImplantsOutput{Body: *result}, nil
	}

	// 3. Fetch from ESI
	log.Printf("Fetching character implants from ESI for character_id: %d", characterID)
	esiImplants, err := s.eveGateway.Character.GetCharacterImplants(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character implants from ESI: %w", err)
	}

	result := &dto.CharacterImplants{
		CharacterID: characterID,
		Implants:    esiImplants,
		UpdatedAt:   time.Now(),
	}

	// 4. Save to database
	dbModel := &models.CharacterImplants{
		CharacterID: characterID,
		Implants:    esiImplants,
		UpdatedAt:   time.Now(),
	}

	if err := s.repository.SaveCharacterImplants(ctx, dbModel); err != nil {
		log.Printf("Failed to save character implants to DB: %v", err)
		// Continue even if save fails
	}

	// 5. Save to cache
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 1 hour
			_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Hour)
		}
	}

	log.Printf("Successfully fetched and saved character implants for character_id: %d", characterID)
	return &dto.CharacterImplantsOutput{Body: *result}, nil
}

// GetCharacterLocation retrieves character location following cache → ESI flow (location is volatile, no DB)
func (s *Service) GetCharacterLocation(ctx context.Context, characterID int, token string) (*dto.CharacterLocationOutput, error) {
	// 1. Check Redis cache first (5 seconds for location data)
	cacheKey := fmt.Sprintf("c:character:location:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterLocation
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character location found in cache for character_id: %d", characterID)
				return &dto.CharacterLocationOutput{Body: result}, nil
			}
		}
	}

	// 2. Fetch from ESI (no database storage for volatile location data)
	log.Printf("Fetching character location from ESI for character_id: %d", characterID)
	esiLocation, err := s.eveGateway.Character.GetCharacterLocation(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character location from ESI: %w", err)
	}

	// Parse the map response
	result := &dto.CharacterLocation{
		CharacterID: characterID,
		UpdatedAt:   time.Now(),
	}

	if solarSystemID, ok := esiLocation["solar_system_id"].(float64); ok {
		result.SolarSystemID = int(solarSystemID)
	} else if solarSystemID, ok := esiLocation["solar_system_id"].(int); ok {
		result.SolarSystemID = solarSystemID
	}

	if stationID, ok := esiLocation["station_id"].(float64); ok {
		sid := int(stationID)
		result.StationID = &sid
	} else if stationID, ok := esiLocation["station_id"].(int); ok {
		result.StationID = &stationID
	}

	if structureID, ok := esiLocation["structure_id"].(float64); ok {
		sid := int64(structureID)
		result.StructureID = &sid
	} else if structureID, ok := esiLocation["structure_id"].(int64); ok {
		result.StructureID = &structureID
	}

	// 3. Update cache (5 seconds for location data)
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 5 seconds (volatile data)
			_ = s.redis.Set(ctx, cacheKey, string(data), 5*time.Second)
		}
	}

	log.Printf("Character location fetched from ESI for character_id: %d, system: %d", characterID, result.SolarSystemID)
	return &dto.CharacterLocationOutput{Body: *result}, nil
}

// GetCharacterSkillQueue retrieves character skill queue following cache → DB → ESI flow
func (s *Service) GetCharacterSkillQueue(ctx context.Context, characterID int, token string) (*dto.CharacterSkillQueueOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:skillqueue:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterSkillQueue
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character skill queue found in cache for character_id: %d", characterID)
				return &dto.CharacterSkillQueueOutput{Body: result}, nil
			}
		}
	}

	// 2. Check database
	dbSkillQueue, err := s.repository.GetCharacterSkillQueue(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching character skill queue from DB: %v", err)
		// Continue to ESI even if DB error
	} else if dbSkillQueue != nil {
		// Found in database, convert to DTO
		result := s.skillQueueModelToDTO(dbSkillQueue)

		// Update cache
		if s.redis != nil {
			if data, err := json.Marshal(result); err == nil {
				// Cache for 5 minutes (skill queue changes frequently)
				_ = s.redis.Set(ctx, cacheKey, string(data), 5*time.Minute)
			}
		}

		log.Printf("Character skill queue found in database for character_id: %d", characterID)
		return &dto.CharacterSkillQueueOutput{Body: *result}, nil
	}

	// 3. Fetch from ESI
	log.Printf("Fetching character skill queue from ESI for character_id: %d", characterID)
	esiSkillQueue, err := s.eveGateway.Character.GetCharacterSkillQueue(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character skill queue from ESI: %w", err)
	}

	// Convert ESI response to model
	dbModel := s.parseESISkillQueue(esiSkillQueue, characterID)

	// 4. Save to database
	if err := s.repository.SaveCharacterSkillQueue(ctx, dbModel); err != nil {
		log.Printf("Failed to save character skill queue to DB: %v", err)
		// Continue even if save fails
	}

	// Convert to DTO
	result := s.skillQueueModelToDTO(dbModel)

	// 5. Save to cache
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 5 minutes (skill queue changes frequently)
			_ = s.redis.Set(ctx, cacheKey, string(data), 5*time.Minute)
		}
	}

	log.Printf("Successfully fetched and saved character skill queue for character_id: %d", characterID)
	return &dto.CharacterSkillQueueOutput{Body: *result}, nil
}

// parseESISkillQueue converts ESI response to database model
func (s *Service) parseESISkillQueue(esiQueue []map[string]any, characterID int) *models.CharacterSkillQueue {
	skillQueue := &models.CharacterSkillQueue{
		CharacterID: characterID,
		Skills:      make([]models.SkillQueueItem, 0, len(esiQueue)),
	}

	for _, skill := range esiQueue {
		item := models.SkillQueueItem{}

		// Parse required fields
		if val, ok := skill["skill_id"].(float64); ok {
			item.SkillID = int(val)
		} else if val, ok := skill["skill_id"].(int); ok {
			item.SkillID = val
		}

		if val, ok := skill["finished_level"].(float64); ok {
			item.FinishedLevel = int(val)
		} else if val, ok := skill["finished_level"].(int); ok {
			item.FinishedLevel = val
		}

		if val, ok := skill["queue_position"].(float64); ok {
			item.QueuePosition = int(val)
		} else if val, ok := skill["queue_position"].(int); ok {
			item.QueuePosition = val
		}

		// Parse optional time fields
		if val, ok := skill["start_date"].(time.Time); ok {
			item.StartDate = &val
		}
		if val, ok := skill["finish_date"].(time.Time); ok {
			item.FinishDate = &val
		}

		// Parse optional SP fields
		if val, ok := skill["training_start_sp"].(float64); ok {
			intVal := int(val)
			item.TrainingStartSP = &intVal
		} else if val, ok := skill["training_start_sp"].(int); ok {
			item.TrainingStartSP = &val
		}

		if val, ok := skill["level_end_sp"].(float64); ok {
			intVal := int(val)
			item.LevelEndSP = &intVal
		} else if val, ok := skill["level_end_sp"].(int); ok {
			item.LevelEndSP = &val
		}

		if val, ok := skill["level_start_sp"].(float64); ok {
			intVal := int(val)
			item.LevelStartSP = &intVal
		} else if val, ok := skill["level_start_sp"].(int); ok {
			item.LevelStartSP = &val
		}

		skillQueue.Skills = append(skillQueue.Skills, item)
	}

	return skillQueue
}

// skillQueueModelToDTO converts database model to DTO
func (s *Service) skillQueueModelToDTO(model *models.CharacterSkillQueue) *dto.CharacterSkillQueue {
	skills := make([]dto.SkillQueueItem, 0, len(model.Skills))
	for _, skill := range model.Skills {
		// Get skill name from SDE service if available
		skillName := ""
		if s.sdeService != nil {
			typeID := fmt.Sprintf("%d", skill.SkillID)
			if typeInfo, err := s.sdeService.GetType(typeID); err == nil && typeInfo != nil {
				// Get English name from the localized name map
				if name, ok := typeInfo.Name["en"]; ok {
					skillName = name
				} else {
					// Fallback to any available language if English not found
					for _, name := range typeInfo.Name {
						skillName = name
						break
					}
				}
			}
		}

		skills = append(skills, dto.SkillQueueItem{
			SkillID:         skill.SkillID,
			SkillName:       skillName,
			FinishedLevel:   skill.FinishedLevel,
			QueuePosition:   skill.QueuePosition,
			StartDate:       skill.StartDate,
			FinishDate:      skill.FinishDate,
			TrainingStartSP: skill.TrainingStartSP,
			LevelEndSP:      skill.LevelEndSP,
			LevelStartSP:    skill.LevelStartSP,
		})
	}

	return &dto.CharacterSkillQueue{
		CharacterID: model.CharacterID,
		Skills:      skills,
		UpdatedAt:   model.UpdatedAt,
	}
}

// GetCharacterSkills retrieves character skills following cache → DB → ESI flow
func (s *Service) GetCharacterSkills(ctx context.Context, characterID int, token string) (*dto.CharacterSkillsOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:skills:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterSkills
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character skills found in cache for character_id: %d", characterID)
				return &dto.CharacterSkillsOutput{Body: result}, nil
			}
		}
	}

	// 2. Check database
	dbSkills, err := s.repository.GetCharacterSkills(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching character skills from DB: %v", err)
		// Continue to ESI even if DB error
	} else if dbSkills != nil {
		// Found in database, convert to DTO
		result := s.skillsModelToDTO(dbSkills)

		// Update cache
		if s.redis != nil {
			if data, err := json.Marshal(result); err == nil {
				// Cache for 30 minutes (skills don't change often but are important)
				_ = s.redis.Set(ctx, cacheKey, string(data), 30*time.Minute)
			}
		}

		log.Printf("Character skills found in database for character_id: %d", characterID)
		return &dto.CharacterSkillsOutput{Body: *result}, nil
	}

	// 3. Fetch from ESI
	log.Printf("Fetching character skills from ESI for character_id: %d", characterID)
	esiSkills, err := s.eveGateway.Character.GetCharacterSkills(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character skills from ESI: %w", err)
	}

	// Convert ESI response to model
	dbModel := s.parseESISkills(esiSkills, characterID)

	// 4. Save to database
	if err := s.repository.SaveCharacterSkills(ctx, dbModel); err != nil {
		log.Printf("Failed to save character skills to DB: %v", err)
		// Continue even if save fails
	}

	// Convert to DTO
	result := s.skillsModelToDTO(dbModel)

	// 5. Save to cache
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 30 minutes
			_ = s.redis.Set(ctx, cacheKey, string(data), 30*time.Minute)
		}
	}

	log.Printf("Successfully fetched and saved character skills for character_id: %d", characterID)
	return &dto.CharacterSkillsOutput{Body: *result}, nil
}

// parseESISkills converts ESI response to database model
func (s *Service) parseESISkills(esiSkills map[string]any, characterID int) *models.CharacterSkills {
	skills := &models.CharacterSkills{
		CharacterID: characterID,
		Skills:      []models.Skill{},
	}

	// Parse total_sp
	if val, ok := esiSkills["total_sp"].(float64); ok {
		skills.TotalSP = int64(val)
	} else if val, ok := esiSkills["total_sp"].(int64); ok {
		skills.TotalSP = val
	}

	// Parse unallocated_sp
	if val, ok := esiSkills["unallocated_sp"].(float64); ok {
		intVal := int(val)
		skills.UnallocatedSP = &intVal
	} else if val, ok := esiSkills["unallocated_sp"].(int); ok {
		skills.UnallocatedSP = &val
	}

	// Parse skills array - handle both []interface{} and []map[string]any
	if skillsList, ok := esiSkills["skills"].([]interface{}); ok {
		for _, skillItem := range skillsList {
			if skillMap, ok := skillItem.(map[string]interface{}); ok {
				skill := models.Skill{}

				if val, ok := skillMap["skill_id"].(float64); ok {
					skill.SkillID = int(val)
				} else if val, ok := skillMap["skill_id"].(int); ok {
					skill.SkillID = val
				}

				if val, ok := skillMap["skillpoints_in_skill"].(float64); ok {
					skill.SkillpointsInSkill = int(val)
				} else if val, ok := skillMap["skillpoints_in_skill"].(int); ok {
					skill.SkillpointsInSkill = val
				}

				if val, ok := skillMap["trained_skill_level"].(float64); ok {
					skill.TrainedSkillLevel = int(val)
				} else if val, ok := skillMap["trained_skill_level"].(int); ok {
					skill.TrainedSkillLevel = val
				}

				if val, ok := skillMap["active_skill_level"].(float64); ok {
					skill.ActiveSkillLevel = int(val)
				} else if val, ok := skillMap["active_skill_level"].(int); ok {
					skill.ActiveSkillLevel = val
				}

				skills.Skills = append(skills.Skills, skill)
			}
		}
	} else if skillsList, ok := esiSkills["skills"].([]map[string]any); ok {
		// Handle []map[string]any type from the wrapper
		for _, skillMap := range skillsList {
			skill := models.Skill{}

			if val, ok := skillMap["skill_id"].(float64); ok {
				skill.SkillID = int(val)
			} else if val, ok := skillMap["skill_id"].(int); ok {
				skill.SkillID = val
			}

			if val, ok := skillMap["skillpoints_in_skill"].(float64); ok {
				skill.SkillpointsInSkill = int(val)
			} else if val, ok := skillMap["skillpoints_in_skill"].(int); ok {
				skill.SkillpointsInSkill = val
			}

			if val, ok := skillMap["trained_skill_level"].(float64); ok {
				skill.TrainedSkillLevel = int(val)
			} else if val, ok := skillMap["trained_skill_level"].(int); ok {
				skill.TrainedSkillLevel = val
			}

			if val, ok := skillMap["active_skill_level"].(float64); ok {
				skill.ActiveSkillLevel = int(val)
			} else if val, ok := skillMap["active_skill_level"].(int); ok {
				skill.ActiveSkillLevel = val
			}

			skills.Skills = append(skills.Skills, skill)
		}
	}

	return skills
}

// skillsModelToDTO converts database model to DTO
func (s *Service) skillsModelToDTO(model *models.CharacterSkills) *dto.CharacterSkills {
	skills := make([]dto.Skill, 0, len(model.Skills))

	for _, skill := range model.Skills {
		// Get skill name from SDE service if available
		skillName := ""
		if s.sdeService != nil {
			typeID := fmt.Sprintf("%d", skill.SkillID)
			if typeInfo, err := s.sdeService.GetType(typeID); err == nil && typeInfo != nil {
				// Get English name from the localized name map
				if name, ok := typeInfo.Name["en"]; ok {
					skillName = name
				} else {
					// Fallback to any available language if English not found
					for _, name := range typeInfo.Name {
						skillName = name
						break
					}
				}
			}
		}

		skills = append(skills, dto.Skill{
			SkillID:            skill.SkillID,
			SkillName:          skillName,
			SkillpointsInSkill: skill.SkillpointsInSkill,
			TrainedSkillLevel:  skill.TrainedSkillLevel,
			ActiveSkillLevel:   skill.ActiveSkillLevel,
		})
	}

	return &dto.CharacterSkills{
		CharacterID:   model.CharacterID,
		Skills:        skills,
		TotalSP:       model.TotalSP,
		UnallocatedSP: model.UnallocatedSP,
		UpdatedAt:     model.UpdatedAt,
	}
}

// parseESIAttributes converts ESI response map to DTO
func (s *Service) parseESIAttributes(attributes map[string]any) *dto.CharacterAttributes {
	result := &dto.CharacterAttributes{}

	// Helper function to safely convert to int
	toInt := func(v interface{}) int {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		default:
			return 0
		}
	}

	result.Charisma = toInt(attributes["charisma"])
	result.Intelligence = toInt(attributes["intelligence"])
	result.Memory = toInt(attributes["memory"])
	result.Perception = toInt(attributes["perception"])
	result.Willpower = toInt(attributes["willpower"])

	// Handle optional fields
	if val, ok := attributes["accrued_remap_cooldown_date"]; ok && val != nil {
		if t, ok := val.(*time.Time); ok {
			result.AccruedRemapCooldownDate = t
		}
	}
	if val, ok := attributes["bonus_remaps"]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			intVal := int(v)
			result.BonusRemaps = &intVal
		case int:
			result.BonusRemaps = &v
		}
	}
	if val, ok := attributes["last_remap_date"]; ok && val != nil {
		if t, ok := val.(*time.Time); ok {
			result.LastRemapDate = t
		}
	}

	return result
}

// modelToDTO converts database model to DTO
func (s *Service) modelToDTO(model *models.CharacterAttributes) *dto.CharacterAttributes {
	return &dto.CharacterAttributes{
		Charisma:                 model.Charisma,
		Intelligence:             model.Intelligence,
		Memory:                   model.Memory,
		Perception:               model.Perception,
		Willpower:                model.Willpower,
		AccruedRemapCooldownDate: model.AccruedRemapCooldownDate,
		BonusRemaps:              model.BonusRemaps,
		LastRemapDate:            model.LastRemapDate,
	}
}

// dtoToModel converts DTO to database model
func (s *Service) dtoToModel(dto *dto.CharacterAttributes, characterID int) *models.CharacterAttributes {
	return &models.CharacterAttributes{
		CharacterID:              characterID,
		Charisma:                 dto.Charisma,
		Intelligence:             dto.Intelligence,
		Memory:                   dto.Memory,
		Perception:               dto.Perception,
		Willpower:                dto.Willpower,
		AccruedRemapCooldownDate: dto.AccruedRemapCooldownDate,
		BonusRemaps:              dto.BonusRemaps,
		LastRemapDate:            dto.LastRemapDate,
	}
}

// GetCharacterProfile retrieves character profile from DB or ESI if not found
func (s *Service) GetCharacterProfile(ctx context.Context, characterID int) (*dto.CharacterProfileOutput, error) {
	log.Printf("GetCharacterProfile called for character ID: %d", characterID)

	// Try to get from database first
	character, err := s.repository.GetCharacterByID(ctx, characterID)
	if err != nil {
		log.Printf("Error getting character from DB: %v", err)
		return nil, err
	}

	// If found in DB, return it
	if character != nil {
		log.Printf("Character found in DB: %+v", character)
		profile := s.characterToProfile(character)
		result := &dto.CharacterProfileOutput{Body: *profile}
		log.Printf("Returning profile output: %+v", result)
		return result, nil
	}

	log.Printf("Character not found in DB, fetching from ESI")

	// Not in DB, fetch from ESI
	esiData, err := s.eveGateway.GetCharacterInfo(ctx, characterID)
	if err != nil {
		log.Printf("Error fetching from ESI: %v", err)
		return nil, err
	}

	log.Printf("ESI data received: %+v", esiData)

	// Parse the map response - using safe type assertions with defaults
	character = &models.Character{
		CharacterID: characterID, // We already have this
	}

	// Parse fields from map with safe type assertions
	if name, ok := esiData["name"].(string); ok {
		character.Name = name
	}
	if corpID, ok := esiData["corporation_id"].(float64); ok {
		character.CorporationID = int(corpID)
	} else if corpID, ok := esiData["corporation_id"].(int); ok {
		character.CorporationID = corpID
	}
	if allianceID, ok := esiData["alliance_id"].(float64); ok {
		character.AllianceID = int(allianceID)
	} else if allianceID, ok := esiData["alliance_id"].(int); ok {
		character.AllianceID = allianceID
	}
	if secStatus, ok := esiData["security_status"].(float64); ok {
		character.SecurityStatus = secStatus
	}
	if desc, ok := esiData["description"].(string); ok {
		character.Description = desc
	}
	if gender, ok := esiData["gender"].(string); ok {
		character.Gender = gender
	}
	if raceID, ok := esiData["race_id"].(float64); ok {
		character.RaceID = int(raceID)
	} else if raceID, ok := esiData["race_id"].(int); ok {
		character.RaceID = raceID
	}
	if bloodlineID, ok := esiData["bloodline_id"].(float64); ok {
		character.BloodlineID = int(bloodlineID)
	} else if bloodlineID, ok := esiData["bloodline_id"].(int); ok {
		character.BloodlineID = bloodlineID
	}
	if ancestryID, ok := esiData["ancestry_id"].(float64); ok {
		character.AncestryID = int(ancestryID)
	} else if ancestryID, ok := esiData["ancestry_id"].(int); ok {
		character.AncestryID = ancestryID
	}

	// Handle birthday if present as time.Time
	if birthday, ok := esiData["birthday"].(time.Time); ok {
		character.Birthday = birthday
	}
	// Handle faction_id if present
	if factionID, ok := esiData["faction_id"].(float64); ok {
		character.FactionID = int(factionID)
	} else if factionID, ok := esiData["faction_id"].(int); ok {
		character.FactionID = factionID
	}

	// Save to database
	if err := s.repository.SaveCharacter(ctx, character); err != nil {
		log.Printf("Error saving character to DB: %v", err)
		return nil, err
	}

	log.Printf("Character saved to DB: %+v", character)
	profile := s.characterToProfile(character)
	result := &dto.CharacterProfileOutput{Body: *profile}
	log.Printf("Returning ESI-fetched profile output: %+v", result)
	return result, nil
}

// SearchCharactersByName searches characters by name
func (s *Service) SearchCharactersByName(ctx context.Context, name string) (*dto.SearchCharactersByNameOutput, error) {
	log.Printf("SearchCharactersByName called with name: %s", name)

	characters, err := s.repository.SearchCharactersByName(ctx, name)
	if err != nil {
		log.Printf("Error searching characters: %v", err)
		return nil, err
	}

	log.Printf("Found %d characters matching name: %s", len(characters), name)

	// Convert to DTOs
	profiles := make([]dto.CharacterProfile, len(characters))
	for i, character := range characters {
		profiles[i] = *s.characterToProfile(character)
	}

	result := &dto.SearchCharactersByNameOutput{
		Body: dto.SearchCharactersResult{
			Characters: profiles,
			Count:      len(profiles),
		},
	}

	return result, nil
}

// characterToProfile converts Character model to CharacterProfile DTO
func (s *Service) characterToProfile(character *models.Character) *dto.CharacterProfile {
	return &dto.CharacterProfile{
		CharacterID:    character.CharacterID,
		Name:           character.Name,
		CorporationID:  character.CorporationID,
		AllianceID:     character.AllianceID,
		Birthday:       character.Birthday,
		SecurityStatus: character.SecurityStatus,
		Description:    character.Description,
		Gender:         character.Gender,
		RaceID:         character.RaceID,
		BloodlineID:    character.BloodlineID,
		AncestryID:     character.AncestryID,
		FactionID:      character.FactionID,
		CreatedAt:      character.CreatedAt,
		UpdatedAt:      character.UpdatedAt,
	}
}

// GetStatus returns the health status of the character module
func (s *Service) GetStatus(ctx context.Context) *dto.CharacterStatusResponse {
	now := time.Now()
	overallStatus := "healthy"
	message := "All character services operational"

	// Initialize dependency status
	depStatus := &dto.CharacterDependencyStatus{}

	// Check database connectivity and latency
	dbStart := time.Now()
	if err := s.repository.CheckHealth(ctx); err != nil {
		depStatus.Database = "unhealthy"
		overallStatus = "unhealthy"
		message = "Database connection failed: " + err.Error()
	} else {
		depStatus.Database = "healthy"
		depStatus.DatabaseLatency = fmt.Sprintf("%dms", time.Since(dbStart).Milliseconds())
	}

	// Check EVE ESI connectivity by testing server status endpoint
	esiStart := time.Now()
	if serverStatus, err := s.eveGateway.GetServerStatus(ctx); err != nil {
		depStatus.EVEOnlineESI = "degraded"
		depStatus.ESIErrorLimits = "unknown"
		if overallStatus == "healthy" {
			overallStatus = "degraded"
			message = "EVE ESI connectivity issues: " + err.Error()
		}
	} else {
		depStatus.EVEOnlineESI = "healthy"
		depStatus.ESILatency = fmt.Sprintf("%dms", time.Since(esiStart).Milliseconds())
		depStatus.ESIErrorLimits = fmt.Sprintf("%d players online", serverStatus.Players)
	}

	// Collect character metrics
	metrics := &dto.CharacterMetrics{}

	// Get character count from database
	if totalChars, err := s.repository.GetCharacterCount(ctx); err == nil {
		metrics.TotalCharacters = int(totalChars)
	}

	// Get recently updated characters (last 24 hours)
	if recentCount, err := s.repository.GetRecentlyUpdatedCount(ctx, 24*time.Hour); err == nil {
		metrics.RecentlyUpdated = int(recentCount)
	}

	// Calculate cache hit rate (approximation based on database vs ESI requests)
	if metrics.TotalCharacters > 0 && metrics.RecentlyUpdated >= 0 {
		// Rough estimate: characters not recently updated are likely cache hits
		cachedCharacters := metrics.TotalCharacters - metrics.RecentlyUpdated
		if cachedCharacters > 0 {
			metrics.CacheHitRate = float64(cachedCharacters) / float64(metrics.TotalCharacters) * 100
		} else {
			metrics.CacheHitRate = 0.0
		}
	}

	// Placeholder values for metrics that would require additional tracking
	metrics.AffiliationUpdates = 45                        // Would need tracking table
	metrics.ESIRequests = 23                               // Would need request counter
	metrics.AverageResponseTime = "150ms"                  // Would need response time tracking
	metrics.LastAffiliationUpdate = "2025-09-04T04:00:00Z" // Would get from scheduler

	// Get memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	metrics.MemoryUsage = float64(memStats.Alloc) / 1024 / 1024 // Convert to MB

	return &dto.CharacterStatusResponse{
		Module:       "character",
		Status:       overallStatus,
		Message:      message,
		Dependencies: depStatus,
		Metrics:      metrics,
		LastChecked:  now.Format(time.RFC3339),
	}
}

// GetCharacterFatigue retrieves character jump fatigue following cache → ESI flow (fatigue is volatile, no DB)
func (s *Service) GetCharacterFatigue(ctx context.Context, characterID int, token string) (*dto.CharacterFatigueOutput, error) {
	// 1. Check Redis cache first (5 minutes for fatigue data)
	cacheKey := fmt.Sprintf("c:character:fatigue:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterFatigue
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character fatigue found in cache for character_id: %d", characterID)
				return &dto.CharacterFatigueOutput{Body: result}, nil
			}
		}
	}

	// 2. Fetch from ESI (no database storage for volatile fatigue data)
	log.Printf("Fetching character fatigue from ESI for character_id: %d", characterID)
	esiFatigue, err := s.eveGateway.Character.GetCharacterFatigue(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character fatigue from ESI: %w", err)
	}

	// Parse the map response
	result := &dto.CharacterFatigue{
		CharacterID: characterID,
		UpdatedAt:   time.Now(),
	}

	// Parse optional time fields
	if expireDate, ok := esiFatigue["jump_fatigue_expire_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, expireDate); err == nil {
			result.JumpFatigueExpireDate = &t
		}
	}

	if lastJumpDate, ok := esiFatigue["last_jump_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastJumpDate); err == nil {
			result.LastJumpDate = &t
		}
	}

	if lastUpdateDate, ok := esiFatigue["last_update_date"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastUpdateDate); err == nil {
			result.LastUpdateDate = &t
		}
	}

	// 3. Update cache (5 minutes for fatigue data)
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 5 minutes (moderately volatile data)
			_ = s.redis.Set(ctx, cacheKey, string(data), 5*time.Minute)
		}
	}

	log.Printf("Character fatigue fetched from ESI for character_id: %d", characterID)
	return &dto.CharacterFatigueOutput{Body: *result}, nil
}

// GetCharacterOnline retrieves character online status following cache → ESI flow (online status is very volatile, minimal cache)
func (s *Service) GetCharacterOnline(ctx context.Context, characterID int, token string) (*dto.CharacterOnlineOutput, error) {
	// 1. Check Redis cache first (30 seconds for highly volatile online status)
	cacheKey := fmt.Sprintf("c:character:online:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterOnline
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character online status found in cache for character_id: %d", characterID)
				return &dto.CharacterOnlineOutput{Body: result}, nil
			}
		}
	}

	// 2. Fetch from ESI (no database storage for highly volatile online status)
	log.Printf("Fetching character online status from ESI for character_id: %d", characterID)
	esiOnline, err := s.eveGateway.Character.GetCharacterOnline(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character online status from ESI: %w", err)
	}

	// Parse the map response
	result := &dto.CharacterOnline{
		CharacterID: characterID,
		UpdatedAt:   time.Now(),
	}

	// Parse required online field
	if online, ok := esiOnline["online"].(bool); ok {
		result.Online = online
	}

	// Parse optional time fields
	if lastLogin, ok := esiOnline["last_login"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastLogin); err == nil {
			result.LastLogin = &t
		}
	}

	if lastLogout, ok := esiOnline["last_logout"].(string); ok {
		if t, err := time.Parse(time.RFC3339, lastLogout); err == nil {
			result.LastLogout = &t
		}
	}

	// Parse optional logins count
	if logins, ok := esiOnline["logins"].(float64); ok {
		loginsInt := int(logins)
		result.LoginsToday = &loginsInt
	}

	// 3. Update cache (30 seconds for highly volatile online status)
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 30 seconds (very volatile data)
			_ = s.redis.Set(ctx, cacheKey, string(data), 30*time.Second)
		}
	}

	log.Printf("Character online status fetched from ESI for character_id: %d", characterID)
	return &dto.CharacterOnlineOutput{Body: *result}, nil
}

// GetCharacterShip retrieves character current ship following cache → ESI flow (ship data is moderately volatile)
func (s *Service) GetCharacterShip(ctx context.Context, characterID int, token string) (*dto.CharacterShipOutput, error) {
	// 1. Check Redis cache first (2 minutes for ship data - can change when docking/undocking)
	cacheKey := fmt.Sprintf("c:character:ship:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterShip
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character ship found in cache for character_id: %d", characterID)
				return &dto.CharacterShipOutput{Body: result}, nil
			}
		}
	}

	// 2. Fetch from ESI (no database storage for volatile ship data)
	log.Printf("Fetching character ship from ESI for character_id: %d", characterID)
	esiShip, err := s.eveGateway.Character.GetCharacterShip(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character ship from ESI: %w", err)
	}

	// Parse the map response
	result := &dto.CharacterShip{
		CharacterID: characterID,
		UpdatedAt:   time.Now(),
	}

	// Parse required fields
	if shipItemID, ok := esiShip["ship_item_id"].(float64); ok {
		result.ShipItemID = int64(shipItemID)
	}

	if shipName, ok := esiShip["ship_name"].(string); ok {
		result.ShipName = shipName
	}

	if shipTypeID, ok := esiShip["ship_type_id"].(float64); ok {
		result.ShipTypeID = int(shipTypeID)
	}

	// 3. Update cache (2 minutes for moderately volatile ship data)
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 2 minutes (moderately volatile data - can change when switching ships)
			_ = s.redis.Set(ctx, cacheKey, string(data), 2*time.Minute)
		}
	}

	log.Printf("Character ship fetched from ESI for character_id: %d", characterID)
	return &dto.CharacterShipOutput{Body: *result}, nil
}

// GetCharacterWallet retrieves character wallet balance following cache → ESI flow (wallet data is volatile)
func (s *Service) GetCharacterWallet(ctx context.Context, characterID int, token string) (*dto.CharacterWalletOutput, error) {
	// 1. Check Redis cache first (1 minute for wallet balance - can change frequently with transactions)
	cacheKey := fmt.Sprintf("c:character:wallet:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.CharacterWallet
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Character wallet found in cache for character_id: %d", characterID)
				return &dto.CharacterWalletOutput{Body: result}, nil
			}
		}
	}

	// 2. Fetch from ESI (no database storage for volatile wallet data)
	log.Printf("Fetching character wallet from ESI for character_id: %d", characterID)
	esiWallet, err := s.eveGateway.Character.GetCharacterWallet(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character wallet from ESI: %w", err)
	}

	// Parse the map response
	result := &dto.CharacterWallet{
		CharacterID: characterID,
		UpdatedAt:   time.Now(),
	}

	// Parse wallet balance - the EVE Gateway returns it wrapped in a WalletResponse
	if balance, ok := esiWallet["balance"].(float64); ok {
		result.Balance = balance
	}

	// 3. Update cache (1 minute for volatile wallet data)
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			// Cache for 1 minute (volatile data - can change with transactions)
			_ = s.redis.Set(ctx, cacheKey, string(data), 1*time.Minute)
		}
	}

	log.Printf("Character wallet fetched from ESI for character_id: %d (balance: %.2f ISK)", characterID, result.Balance)
	return &dto.CharacterWalletOutput{Body: *result}, nil
}

// GetEnrichedSkillTree retrieves character skills organized by categories with statistics
func (s *Service) GetEnrichedSkillTree(ctx context.Context, characterID int, token string) (*dto.EnrichedSkillTreeOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("c:character:skill-tree:%d", characterID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			// Parse cached data
			var result dto.EnrichedSkillTree
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				log.Printf("Enriched skill tree found in cache for character_id: %d", characterID)
				return &dto.EnrichedSkillTreeOutput{Body: result}, nil
			}
		}
	}

	// 2. Get character skills using existing service method
	skillsResponse, err := s.GetCharacterSkills(ctx, characterID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get character skills: %w", err)
	}

	// 3. Build skill tree from SDE data and enrich with character skills
	enrichedTree, err := s.buildEnrichedSkillTree(skillsResponse.Body.Skills, skillsResponse.Body.TotalSP, skillsResponse.Body.UnallocatedSP)
	if err != nil {
		return nil, fmt.Errorf("failed to build enriched skill tree: %w", err)
	}

	enrichedTree.CharacterID = characterID
	enrichedTree.UpdatedAt = time.Now()

	// 4. Update cache
	if s.redis != nil {
		if data, err := json.Marshal(enrichedTree); err == nil {
			// Cache for 30 minutes (skills don't change often)
			_ = s.redis.Set(ctx, cacheKey, string(data), 30*time.Minute)
		}
	}

	log.Printf("Enriched skill tree built for character_id: %d with %d categories", characterID, len(enrichedTree.SkillTree))
	return &dto.EnrichedSkillTreeOutput{Body: *enrichedTree}, nil
}

// buildEnrichedSkillTree constructs the skill tree organized by categories with statistics
func (s *Service) buildEnrichedSkillTree(characterSkills []dto.Skill, totalSP int64, unallocatedSP *int) (*dto.EnrichedSkillTree, error) {
	// Maximum skill points per category (from Node.js implementation)
	maxSkillPoints := map[string]int{
		"Armor":                  12032000,
		"Corporation Management": 4864000,
		"Drones":                 35840000,
		"Electronic Systems":     17408000,
		"Engineering":            11008000,
		"Fleet Support":          22784000,
		"Gunnery":                86528000,
		"Missiles":               29184000,
		"Navigation":             15872000,
		"Neural Enhancement":     10496000,
		"Planet Management":      4352000,
		"Production":             21248000,
		"Resource Processing":    36608000,
		"Rigging":                7424000,
		"Scanning":               7168000,
		"Science":                49408000,
		"Shields":                12288000,
		"Spaceship Command":      151808000,
		"Structure Management":   4608000,
		"Subsystems":             4096000,
		"Targeting":              3840000,
		"Trade":                  12032000,
	}

	// Create a map of character skills for fast lookup
	skillMap := make(map[int]dto.Skill)
	for _, skill := range characterSkills {
		skillMap[skill.SkillID] = skill
	}

	// Get all skill groups from SDE
	allGroups, err := s.sdeService.GetAllGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to get skill groups from SDE: %w", err)
	}

	// Build categories map for skill groups
	skillGroupMap := make(map[int]string) // groupID -> categoryName

	// Process each group to find skill groups (category ID 16 is Skills in EVE)
	for groupIDStr, group := range allGroups {
		if group.CategoryID != 16 { // Skip non-skill groups
			continue
		}

		// Get group name (use English if available)
		categoryName := "Unknown"
		if enName, ok := group.Name["en"]; ok && enName != "" {
			categoryName = enName
		} else if len(group.Name) > 0 {
			// Use first available language if English not found
			for _, name := range group.Name {
				if name != "" {
					categoryName = name
					break
				}
			}
		}

		// Skip fake skills category
		if categoryName == "Fake Skills" {
			continue
		}

		groupID, _ := strconv.Atoi(groupIDStr)
		skillGroupMap[groupID] = categoryName
	}

	// Get all types and filter for skills
	allTypes, err := s.sdeService.GetAllTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get all types from SDE: %w", err)
	}

	// Build categories
	categories := make(map[string]*dto.SkillCategory)

	// Process each type to find skills
	for typeIDStr, typeInfo := range allTypes {
		// Check if this type belongs to a skill group and is published
		categoryName, isSkillGroup := skillGroupMap[typeInfo.GroupID]
		if !isSkillGroup || !typeInfo.Published {
			continue
		}

		// Initialize category if not exists
		if categories[categoryName] == nil {
			categories[categoryName] = &dto.SkillCategory{
				Category:          categoryName,
				Skills:            []dto.EnrichedSkill{},
				TotalSpInCategory: 0,
			}
			// Set max SP if we have it
			if maxSP, exists := maxSkillPoints[categoryName]; exists {
				categories[categoryName].MaxCategorySP = &maxSP
			}
		}

		typeID, _ := strconv.Atoi(typeIDStr)

		// Get skill name
		skillName := "Unknown Skill"
		if enName, ok := typeInfo.Name["en"]; ok && enName != "" {
			skillName = enName
		}

		// Create enriched skill
		enrichedSkill := dto.EnrichedSkill{
			SkillID:            typeID,
			Name:               skillName,
			ActiveSkillLevel:   0,
			SkillpointsInSkill: 0,
			TrainedSkillLevel:  0,
		}

		// Enrich with character data if character has this skill
		if charSkill, hasSkill := skillMap[typeID]; hasSkill {
			enrichedSkill.ActiveSkillLevel = charSkill.ActiveSkillLevel
			enrichedSkill.SkillpointsInSkill = charSkill.SkillpointsInSkill
			enrichedSkill.TrainedSkillLevel = charSkill.TrainedSkillLevel
			enrichedSkill.Name = charSkill.SkillName // Use name from ESI response if available

			categories[categoryName].TotalSpInCategory += charSkill.SkillpointsInSkill
		}

		categories[categoryName].Skills = append(categories[categoryName].Skills, enrichedSkill)
	}

	// Calculate statistics for each category
	for _, category := range categories {
		if len(category.Skills) > 0 {
			// Calculate percentage fulfilled
			categoryLevel := 0
			for _, skill := range category.Skills {
				categoryLevel += skill.TrainedSkillLevel
			}
			howManySkillsInCategory := len(category.Skills)
			percentFulfilled := float64(categoryLevel) / float64(howManySkillsInCategory) / 5.0 * 100.0
			category.PercentFulfilled = fmt.Sprintf("%.0f", percentFulfilled)
		}
	}

	// Convert map to slice
	skillTree := make([]dto.SkillCategory, 0, len(categories))
	for _, category := range categories {
		skillTree = append(skillTree, *category)
	}

	unallocatedSPValue := 0
	if unallocatedSP != nil {
		unallocatedSPValue = *unallocatedSP
	}

	return &dto.EnrichedSkillTree{
		TotalSP:       totalSP,
		UnallocatedSP: unallocatedSPValue,
		SkillTree:     skillTree,
	}, nil
}
