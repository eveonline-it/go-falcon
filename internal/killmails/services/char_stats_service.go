package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-falcon/internal/killmails/models"
	"go-falcon/pkg/sde"
)

// CharStatsService handles character statistics operations
type CharStatsService struct {
	charStatsRepo  *CharStatsRepository
	shipClassifier *sde.ShipClassifier
	sdeService     sde.SDEService
}

// NewCharStatsService creates a new character stats service
func NewCharStatsService(charStatsRepo *CharStatsRepository, sdeService sde.SDEService) *CharStatsService {
	shipClassifier := sde.NewShipClassifier(sdeService)
	return &CharStatsService{
		charStatsRepo:  charStatsRepo,
		shipClassifier: shipClassifier,
		sdeService:     sdeService,
	}
}

// UpdateFromKillmail updates character statistics from a killmail
func (s *CharStatsService) UpdateFromKillmail(ctx context.Context, killmail *models.Killmail) error {
	// Update victim stats (loss)
	if killmail.Victim.CharacterID != nil && *killmail.Victim.CharacterID != 0 {
		err := s.updateCharacterShipUsage(ctx, int32(*killmail.Victim.CharacterID), killmail, killmail.Victim.ShipTypeID, false)
		if err != nil {
			log.Printf("Failed to update victim stats for character %d: %v", *killmail.Victim.CharacterID, err)
		}
	}

	// Update attacker stats (kills)
	for _, attacker := range killmail.Attackers {
		if attacker.CharacterID != nil && *attacker.CharacterID != 0 && attacker.ShipTypeID != nil && *attacker.ShipTypeID != 0 {
			err := s.updateCharacterShipUsage(ctx, int32(*attacker.CharacterID), killmail, *attacker.ShipTypeID, true)
			if err != nil {
				log.Printf("Failed to update attacker stats for character %d: %v", *attacker.CharacterID, err)
			}
		}
	}

	return nil
}

// updateCharacterShipUsage updates character ship usage statistics
func (s *CharStatsService) updateCharacterShipUsage(ctx context.Context, characterID int32, killmail *models.Killmail, shipTypeID int64, isKill bool) error {
	// Get ship category
	category, err := s.shipClassifier.GetShipCategory(shipTypeID)
	if err != nil {
		return fmt.Errorf("failed to get ship category for type %d: %w", shipTypeID, err)
	}

	// Only track if it's a tracked category
	if category == "" {
		return nil
	}

	// Update the character stats with just the ship type ID
	err = s.charStatsRepo.UpdateLastShipUsed(ctx, characterID, category, shipTypeID)
	if err != nil {
		return fmt.Errorf("failed to update character %d ship usage: %w", characterID, err)
	}

	return nil
}

// GetCharacterStats retrieves character statistics
func (s *CharStatsService) GetCharacterStats(ctx context.Context, characterID int32) (*models.CharacterKillmailStats, error) {
	return s.charStatsRepo.GetCharacterStats(ctx, characterID)
}

// GetCharacterLastShipByCategory gets the last ship used by a character in a specific category
func (s *CharStatsService) GetCharacterLastShipByCategory(ctx context.Context, characterID int32, category string) (*int64, error) {
	return s.charStatsRepo.GetCharacterLastShipByCategory(ctx, characterID, category)
}

// GetCharactersByShipCategory returns characters who have used ships in a specific category
func (s *CharStatsService) GetCharactersByShipCategory(ctx context.Context, category string, limit int) ([]*models.CharacterKillmailStats, error) {
	return s.charStatsRepo.GetCharactersByShipCategory(ctx, category, limit)
}

// GetCharactersByShipType returns characters who last used a specific ship type
func (s *CharStatsService) GetCharactersByShipType(ctx context.Context, shipTypeID int64, limit int) ([]*models.CharacterKillmailStats, error) {
	return s.charStatsRepo.GetCharactersByShipType(ctx, shipTypeID, limit)
}

// GetRecentCharacterActivity returns characters with recent activity
func (s *CharStatsService) GetRecentCharacterActivity(ctx context.Context, since time.Time, limit int) ([]*models.CharacterKillmailStats, error) {
	return s.charStatsRepo.GetRecentCharacterActivity(ctx, since, limit)
}

// GetTrackedCategories returns all tracked ship categories
func (s *CharStatsService) GetTrackedCategories() []string {
	return s.shipClassifier.GetTrackedCategories()
}

// GetShipsByCategory returns all ships in a specific category
func (s *CharStatsService) GetShipsByCategory(category string) ([]*sde.Type, error) {
	return s.shipClassifier.GetShipsByCategory(category)
}

// ValidateConfiguration validates the ship classification setup
func (s *CharStatsService) ValidateConfiguration() error {
	return s.shipClassifier.ValidateShipCategories()
}

// CreateIndexes creates necessary database indexes
func (s *CharStatsService) CreateIndexes(ctx context.Context) error {
	return s.charStatsRepo.CreateIndexes(ctx)
}

// GetCategoryStats returns statistics about character usage by category
func (s *CharStatsService) GetCategoryStats(ctx context.Context) (map[string]int64, error) {
	return s.charStatsRepo.CountCharactersByCategory(ctx)
}
