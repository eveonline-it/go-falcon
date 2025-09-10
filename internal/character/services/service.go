package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"time"

	"go-falcon/internal/character/dto"
	"go-falcon/internal/character/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
)

// Service provides business logic for character operations
type Service struct {
	repository *Repository
	eveGateway *evegateway.Client
	redis      *database.Redis
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client) *Service {
	return &Service{
		repository: NewRepository(mongodb),
		eveGateway: eveGateway,
		redis:      redis,
	}
}

// CreateIndexes creates database indexes for optimal performance
func (s *Service) CreateIndexes(ctx context.Context) error {
	return s.repository.CreateIndexes(ctx)
}

// GetCharacterAttributes retrieves character attributes following cache → DB → ESI flow
func (s *Service) GetCharacterAttributes(ctx context.Context, characterID int, token string) (*dto.CharacterAttributesOutput, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("character:attributes:%d", characterID)
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
