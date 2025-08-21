package services

import (
	"context"
	"log"
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
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB, eveGateway *evegateway.Client) *Service {
	return &Service{
		repository: NewRepository(mongodb),
		eveGateway: eveGateway,
	}
}

// CreateIndexes creates database indexes for optimal performance
func (s *Service) CreateIndexes(ctx context.Context) error {
	return s.repository.CreateIndexes(ctx)
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