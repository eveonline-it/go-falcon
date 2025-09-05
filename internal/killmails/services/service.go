package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/internal/killmails/models"
	"go-falcon/pkg/evegateway"
)

type Service struct {
	repository       *Repository
	eveGateway       *evegateway.Client
	charStatsService *CharStatsService
}

func NewService(repository *Repository, eveGateway *evegateway.Client, charStatsService *CharStatsService) *Service {
	return &Service{
		repository:       repository,
		eveGateway:       eveGateway,
		charStatsService: charStatsService,
	}
}

// GetKillmail retrieves a killmail by ID and hash, with database-first approach
func (s *Service) GetKillmail(ctx context.Context, killmailID int64, hash string) (*models.Killmail, error) {
	slog.InfoContext(ctx, "Fetching killmail", "killmail_id", killmailID, "hash", hash)

	// Try database first
	killmail, err := s.repository.GetByKillmailIDAndHash(ctx, killmailID, hash)
	if err != nil {
		slog.ErrorContext(ctx, "Error fetching killmail from database", "error", err)
		return nil, fmt.Errorf("database error: %w", err)
	}

	if killmail != nil {
		slog.InfoContext(ctx, "Killmail found in database", "killmail_id", killmailID)
		return killmail, nil
	}

	// If not in database, fetch from ESI
	slog.InfoContext(ctx, "Killmail not found in database, fetching from ESI", "killmail_id", killmailID)

	esiData, err := s.eveGateway.Killmails.GetKillmail(ctx, killmailID, hash)
	if err != nil {
		slog.ErrorContext(ctx, "Error fetching killmail from ESI", "error", err)
		return nil, fmt.Errorf("ESI error: %w", err)
	}

	// Convert ESI data to model and save to database
	killmail = s.convertESIDataToModel(esiData, hash)
	if err := s.repository.UpsertKillmail(ctx, killmail); err != nil {
		slog.WarnContext(ctx, "Failed to cache killmail in database", "error", err, "killmail_id", killmailID)
		// Continue even if caching fails
	}

	slog.InfoContext(ctx, "Killmail fetched from ESI and cached", "killmail_id", killmailID)
	return killmail, nil
}

// GetCharacterRecentKillmails fetches recent killmails for a character
func (s *Service) GetCharacterRecentKillmails(ctx context.Context, characterID int, token string, limit int) ([]models.KillmailRef, error) {
	slog.InfoContext(ctx, "Fetching recent character killmails", "character_id", characterID, "limit", limit)

	// Use ESI to get recent killmail references
	esiData, err := s.eveGateway.Killmails.GetCharacterRecentKillmails(ctx, characterID, token)
	if err != nil {
		slog.ErrorContext(ctx, "Error fetching recent character killmails from ESI", "error", err)
		return nil, fmt.Errorf("ESI error: %w", err)
	}

	// Convert to model format
	refs := make([]models.KillmailRef, len(esiData))
	for i, km := range esiData {
		refs[i] = models.KillmailRef{
			KillmailID:   km["killmail_id"].(int64),
			KillmailHash: km["killmail_hash"].(string),
		}
	}

	slog.InfoContext(ctx, "Recent character killmails fetched", "character_id", characterID, "count", len(refs))
	return refs, nil
}

// GetCorporationRecentKillmails fetches recent killmails for a corporation
func (s *Service) GetCorporationRecentKillmails(ctx context.Context, corporationID int, token string, limit int) ([]models.KillmailRef, error) {
	slog.InfoContext(ctx, "Fetching recent corporation killmails", "corporation_id", corporationID, "limit", limit)

	// Use ESI to get recent killmail references
	esiData, err := s.eveGateway.Killmails.GetCorporationRecentKillmails(ctx, corporationID, token)
	if err != nil {
		slog.ErrorContext(ctx, "Error fetching recent corporation killmails from ESI", "error", err)
		return nil, fmt.Errorf("ESI error: %w", err)
	}

	// Convert to model format
	refs := make([]models.KillmailRef, len(esiData))
	for i, km := range esiData {
		refs[i] = models.KillmailRef{
			KillmailID:   km["killmail_id"].(int64),
			KillmailHash: km["killmail_hash"].(string),
		}
	}

	slog.InfoContext(ctx, "Recent corporation killmails fetched", "corporation_id", corporationID, "count", len(refs))
	return refs, nil
}

// ImportKillmail imports a killmail by ID and hash
func (s *Service) ImportKillmail(ctx context.Context, killmailID int64, hash string) (*models.Killmail, error) {
	slog.InfoContext(ctx, "Importing killmail", "killmail_id", killmailID, "hash", hash)

	// Check if already exists
	existing, err := s.repository.GetByKillmailIDAndHash(ctx, killmailID, hash)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	if existing != nil {
		slog.InfoContext(ctx, "Killmail already exists", "killmail_id", killmailID)
		return existing, nil
	}

	// Fetch from ESI
	esiData, err := s.eveGateway.Killmails.GetKillmail(ctx, killmailID, hash)
	if err != nil {
		return nil, fmt.Errorf("ESI error: %w", err)
	}

	// Convert and save
	killmail := s.convertESIDataToModel(esiData, hash)
	if err := s.repository.UpsertKillmail(ctx, killmail); err != nil {
		return nil, fmt.Errorf("failed to save killmail: %w", err)
	}

	slog.InfoContext(ctx, "Killmail imported successfully", "killmail_id", killmailID)
	return killmail, nil
}

// GetRecentKillmailsByCharacter gets recent killmails for a character from database
func (s *Service) GetRecentKillmailsByCharacter(ctx context.Context, characterID int64, limit int) ([]models.Killmail, error) {
	return s.repository.GetRecentKillmailsByCharacter(ctx, characterID, limit)
}

// GetRecentKillmailsByCorporation gets recent killmails for a corporation from database
func (s *Service) GetRecentKillmailsByCorporation(ctx context.Context, corporationID int64, limit int) ([]models.Killmail, error) {
	return s.repository.GetRecentKillmailsByCorporation(ctx, corporationID, limit)
}

// GetRecentKillmailsByAlliance gets recent killmails for an alliance from database
func (s *Service) GetRecentKillmailsByAlliance(ctx context.Context, allianceID int64, limit int) ([]models.Killmail, error) {
	return s.repository.GetRecentKillmailsByAlliance(ctx, allianceID, limit)
}

// GetKillmailsBySystem gets killmails in a specific solar system
func (s *Service) GetKillmailsBySystem(ctx context.Context, systemID int64, since time.Time, limit int) ([]models.Killmail, error) {
	return s.repository.GetKillmailsBySystem(ctx, systemID, since, limit)
}

// GetKillmailStats returns basic statistics about killmails
func (s *Service) GetKillmailStats(ctx context.Context) (map[string]interface{}, error) {
	count, err := s.repository.CountKillmails(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_killmails": count,
		"collection":      models.KillmailsCollection,
	}, nil
}

// HealthCheck performs a health check for the service
func (s *Service) HealthCheck(ctx context.Context) error {
	// Check database connectivity by counting documents
	_, err := s.repository.CountKillmails(ctx)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check ESI connectivity
	if _, err := s.eveGateway.GetServerStatus(ctx); err != nil {
		return fmt.Errorf("ESI health check failed: %w", err)
	}

	return nil
}

// toInt64 safely converts interface{} to int64, handling both int64 and float64 types
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	default:
		return 0
	}
}

// toFloat64 safely converts interface{} to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	default:
		return 0
	}
}

// convertESIDataToModel converts ESI response data to internal model
func (s *Service) convertESIDataToModel(esiData map[string]any, hash string) *models.Killmail {
	killmail := &models.Killmail{
		KillmailID:    toInt64(esiData["killmail_id"]),
		KillmailHash:  hash,
		SolarSystemID: toInt64(esiData["solar_system_id"]),
	}

	// Parse killmail time - EVE Gateway returns time.Time object, not string
	if killmailTime, ok := esiData["killmail_time"].(time.Time); ok {
		killmail.KillmailTime = killmailTime
	} else {
		slog.Error("killmail_time not found or invalid type", "value", esiData["killmail_time"], "type", fmt.Sprintf("%T", esiData["killmail_time"]))
	}

	// Optional fields
	if moonID, ok := esiData["moon_id"]; ok {
		moonIDInt := toInt64(moonID)
		if moonIDInt != 0 {
			killmail.MoonID = &moonIDInt
		}
	}

	if warID, ok := esiData["war_id"]; ok {
		warIDInt := toInt64(warID)
		if warIDInt != 0 {
			killmail.WarID = &warIDInt
		}
	}

	// Convert victim
	if victimData, ok := esiData["victim"].(map[string]any); ok {
		killmail.Victim = s.convertVictim(victimData)
	}

	// Convert attackers
	if attackersData, ok := esiData["attackers"].([]any); ok {
		killmail.Attackers = s.convertAttackers(attackersData)
	}

	return killmail
}

func (s *Service) convertVictim(data map[string]any) models.Victim {
	victim := models.Victim{
		ShipTypeID:  toInt64(data["ship_type_id"]),
		DamageTaken: toInt64(data["damage_taken"]),
	}

	// Optional fields
	if charID, ok := data["character_id"]; ok {
		charIDInt := toInt64(charID)
		if charIDInt != 0 {
			victim.CharacterID = &charIDInt
		}
	}
	if corpID, ok := data["corporation_id"]; ok {
		corpIDInt := toInt64(corpID)
		if corpIDInt != 0 {
			victim.CorporationID = &corpIDInt
		}
	}
	if allianceID, ok := data["alliance_id"]; ok {
		allianceIDInt := toInt64(allianceID)
		if allianceIDInt != 0 {
			victim.AllianceID = &allianceIDInt
		}
	}
	if factionID, ok := data["faction_id"]; ok {
		factionIDInt := toInt64(factionID)
		if factionIDInt != 0 {
			victim.FactionID = &factionIDInt
		}
	}

	// Convert position
	if posData, ok := data["position"].(map[string]any); ok {
		victim.Position = &models.Position{
			X: toFloat64(posData["x"]),
			Y: toFloat64(posData["y"]),
			Z: toFloat64(posData["z"]),
		}
	}

	// Convert items
	if itemsData, ok := data["items"].([]any); ok {
		victim.Items = s.convertItems(itemsData)
	}

	return victim
}

func (s *Service) convertAttackers(data []any) []models.Attacker {
	attackers := make([]models.Attacker, len(data))

	for i, attackerData := range data {
		if attackerMap, ok := attackerData.(map[string]any); ok {
			attacker := models.Attacker{
				DamageDone:     toInt64(attackerMap["damage_done"]),
				FinalBlow:      attackerMap["final_blow"].(bool),
				SecurityStatus: toFloat64(attackerMap["security_status"]),
			}

			// Optional fields
			if charID, ok := attackerMap["character_id"]; ok {
				charIDInt := toInt64(charID)
				if charIDInt != 0 {
					attacker.CharacterID = &charIDInt
				}
			}
			if corpID, ok := attackerMap["corporation_id"]; ok {
				corpIDInt := toInt64(corpID)
				if corpIDInt != 0 {
					attacker.CorporationID = &corpIDInt
				}
			}
			if allianceID, ok := attackerMap["alliance_id"]; ok {
				allianceIDInt := toInt64(allianceID)
				if allianceIDInt != 0 {
					attacker.AllianceID = &allianceIDInt
				}
			}
			if factionID, ok := attackerMap["faction_id"]; ok {
				factionIDInt := toInt64(factionID)
				if factionIDInt != 0 {
					attacker.FactionID = &factionIDInt
				}
			}
			if shipTypeID, ok := attackerMap["ship_type_id"]; ok {
				shipTypeIDInt := toInt64(shipTypeID)
				if shipTypeIDInt != 0 {
					attacker.ShipTypeID = &shipTypeIDInt
				}
			}
			if weaponTypeID, ok := attackerMap["weapon_type_id"]; ok {
				weaponTypeIDInt := toInt64(weaponTypeID)
				if weaponTypeIDInt != 0 {
					attacker.WeaponTypeID = &weaponTypeIDInt
				}
			}

			attackers[i] = attacker
		}
	}

	return attackers
}

func (s *Service) convertItems(data []any) []models.Item {
	items := make([]models.Item, len(data))

	for i, itemData := range data {
		if itemMap, ok := itemData.(map[string]any); ok {
			item := models.Item{
				ItemTypeID: toInt64(itemMap["item_type_id"]),
				Flag:       toInt64(itemMap["flag"]),
				Singleton:  toInt64(itemMap["singleton"]),
			}

			// Optional fields
			if qtyDestroyed, ok := itemMap["quantity_destroyed"]; ok {
				qtyDestroyedInt := toInt64(qtyDestroyed)
				if qtyDestroyedInt != 0 {
					item.QuantityDestroyed = &qtyDestroyedInt
				}
			}
			if qtyDropped, ok := itemMap["quantity_dropped"]; ok {
				qtyDroppedInt := toInt64(qtyDropped)
				if qtyDroppedInt != 0 {
					item.QuantityDropped = &qtyDroppedInt
				}
			}

			// Convert nested items
			if nestedItems, ok := itemMap["items"].([]any); ok {
				item.Items = s.convertItems(nestedItems)
			}

			items[i] = item
		}
	}

	return items
}

// Character Stats Service Methods

// GetCharacterStats retrieves character killmail statistics
func (s *Service) GetCharacterStats(ctx context.Context, characterID int32) (*models.CharacterKillmailStats, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetCharacterStats(ctx, characterID)
}

// GetCharacterLastShipByCategory gets the last ship used by a character in a specific category
func (s *Service) GetCharacterLastShipByCategory(ctx context.Context, characterID int32, category string) (*int64, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetCharacterLastShipByCategory(ctx, characterID, category)
}

// GetCharactersByShipCategory returns characters who have used ships in a specific category
func (s *Service) GetCharactersByShipCategory(ctx context.Context, category string, limit int) ([]*models.CharacterKillmailStats, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetCharactersByShipCategory(ctx, category, limit)
}

// GetCharactersByShipType returns characters who last used a specific ship type
func (s *Service) GetCharactersByShipType(ctx context.Context, shipTypeID int64, limit int) ([]*models.CharacterKillmailStats, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetCharactersByShipType(ctx, shipTypeID, limit)
}

// GetRecentCharacterActivity returns characters with recent activity
func (s *Service) GetRecentCharacterActivity(ctx context.Context, since time.Time, limit int) ([]*models.CharacterKillmailStats, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetRecentCharacterActivity(ctx, since, limit)
}

// GetTrackedCategories returns all tracked ship categories
func (s *Service) GetTrackedCategories(ctx context.Context) ([]string, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetTrackedCategories(), nil
}

// GetCategoryStats returns statistics about character usage by category
func (s *Service) GetCategoryStats(ctx context.Context) (map[string]int64, error) {
	if s.charStatsService == nil {
		return nil, fmt.Errorf("character stats service not available")
	}
	return s.charStatsService.GetCategoryStats(ctx)
}
