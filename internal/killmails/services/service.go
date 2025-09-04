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
	repository *Repository
	eveGateway *evegateway.Client
}

func NewService(repository *Repository, eveGateway *evegateway.Client) *Service {
	return &Service{
		repository: repository,
		eveGateway: eveGateway,
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
		"last_updated":    time.Now(),
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

// convertESIDataToModel converts ESI response data to internal model
func (s *Service) convertESIDataToModel(esiData map[string]any, hash string) *models.Killmail {
	killmail := &models.Killmail{
		KillmailID:    int64(esiData["killmail_id"].(float64)),
		KillmailHash:  hash,
		SolarSystemID: int64(esiData["solar_system_id"].(float64)),
	}

	// Parse killmail time
	if timeStr, ok := esiData["killmail_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			killmail.KillmailTime = t
		}
	}

	// Optional fields
	if moonID, ok := esiData["moon_id"].(float64); ok {
		moonIDInt := int64(moonID)
		killmail.MoonID = &moonIDInt
	}

	if warID, ok := esiData["war_id"].(float64); ok {
		warIDInt := int64(warID)
		killmail.WarID = &warIDInt
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
		ShipTypeID:  int64(data["ship_type_id"].(float64)),
		DamageTaken: int64(data["damage_taken"].(float64)),
	}

	// Optional fields
	if charID, ok := data["character_id"].(float64); ok {
		charIDInt := int64(charID)
		victim.CharacterID = &charIDInt
	}
	if corpID, ok := data["corporation_id"].(float64); ok {
		corpIDInt := int64(corpID)
		victim.CorporationID = &corpIDInt
	}
	if allianceID, ok := data["alliance_id"].(float64); ok {
		allianceIDInt := int64(allianceID)
		victim.AllianceID = &allianceIDInt
	}
	if factionID, ok := data["faction_id"].(float64); ok {
		factionIDInt := int64(factionID)
		victim.FactionID = &factionIDInt
	}

	// Convert position
	if posData, ok := data["position"].(map[string]any); ok {
		victim.Position = &models.Position{
			X: posData["x"].(float64),
			Y: posData["y"].(float64),
			Z: posData["z"].(float64),
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
				DamageDone:     int64(attackerMap["damage_done"].(float64)),
				FinalBlow:      attackerMap["final_blow"].(bool),
				SecurityStatus: attackerMap["security_status"].(float64),
			}

			// Optional fields
			if charID, ok := attackerMap["character_id"].(float64); ok {
				charIDInt := int64(charID)
				attacker.CharacterID = &charIDInt
			}
			if corpID, ok := attackerMap["corporation_id"].(float64); ok {
				corpIDInt := int64(corpID)
				attacker.CorporationID = &corpIDInt
			}
			if allianceID, ok := attackerMap["alliance_id"].(float64); ok {
				allianceIDInt := int64(allianceID)
				attacker.AllianceID = &allianceIDInt
			}
			if factionID, ok := attackerMap["faction_id"].(float64); ok {
				factionIDInt := int64(factionID)
				attacker.FactionID = &factionIDInt
			}
			if shipTypeID, ok := attackerMap["ship_type_id"].(float64); ok {
				shipTypeIDInt := int64(shipTypeID)
				attacker.ShipTypeID = &shipTypeIDInt
			}
			if weaponTypeID, ok := attackerMap["weapon_type_id"].(float64); ok {
				weaponTypeIDInt := int64(weaponTypeID)
				attacker.WeaponTypeID = &weaponTypeIDInt
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
				ItemTypeID: int64(itemMap["item_type_id"].(float64)),
				Flag:       int64(itemMap["flag"].(float64)),
				Singleton:  int64(itemMap["singleton"].(float64)),
			}

			// Optional fields
			if qtyDestroyed, ok := itemMap["quantity_destroyed"].(float64); ok {
				qtyDestroyedInt := int64(qtyDestroyed)
				item.QuantityDestroyed = &qtyDestroyedInt
			}
			if qtyDropped, ok := itemMap["quantity_dropped"].(float64); ok {
				qtyDroppedInt := int64(qtyDropped)
				item.QuantityDropped = &qtyDroppedInt
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
