package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-falcon/internal/killmails/models"
	killmailsService "go-falcon/internal/killmails/services"
	websocketServices "go-falcon/internal/websocket/services"
	"go-falcon/internal/zkillboard/dto"
	zkbModels "go-falcon/internal/zkillboard/models"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"
)

// KillmailProcessor handles the killmail processing pipeline
type KillmailProcessor struct {
	killmailRepo     *killmailsService.Repository
	zkbRepo          *Repository
	aggregator       *Aggregator
	esiClient        evegateway.KillmailClient
	websocketService *websocketServices.WebSocketService
	sdeService       sde.SDEService
	charStatsService *killmailsService.CharStatsService

	// Batch processing
	batchSize  int
	batchMu    sync.Mutex
	batch      []*ProcessedKillmail
	batchTimer *time.Timer
}

// ProcessedKillmail represents a fully processed killmail
type ProcessedKillmail struct {
	Killmail    *models.Killmail
	ZKBMetadata *zkbModels.ZKBMetadata
	Package     *dto.RedisQPackage
}

// Using getEnvAsInt from redisq_consumer.go

// NewKillmailProcessor creates a new killmail processor
func NewKillmailProcessor(
	killmailRepo *killmailsService.Repository,
	zkbRepo *Repository,
	aggregator *Aggregator,
	esiClient evegateway.KillmailClient,
	websocketService *websocketServices.WebSocketService,
	sdeService sde.SDEService,
	charStatsService *killmailsService.CharStatsService,
) *KillmailProcessor {
	batchSize := getEnvAsInt("ZKB_BATCH_SIZE", 10)

	return &KillmailProcessor{
		killmailRepo:     killmailRepo,
		zkbRepo:          zkbRepo,
		aggregator:       aggregator,
		esiClient:        esiClient,
		websocketService: websocketService,
		sdeService:       sdeService,
		charStatsService: charStatsService,
		batchSize:        batchSize,
		batch:            make([]*ProcessedKillmail, 0, batchSize),
	}
}

// ProcessKillmail processes a single killmail from RedisQ
func (p *KillmailProcessor) ProcessKillmail(ctx context.Context, pkg *dto.RedisQPackage) error {
	// Check for duplicate
	exists, err := p.killmailRepo.Exists(ctx, pkg.KillID, "")
	if err != nil {
		return fmt.Errorf("failed to check killmail existence: %w", err)
	}

	if exists {
		slog.Debug("Killmail already exists, skipping", "killmail_id", pkg.KillID)
		return nil
	}

	// Parse the ESI killmail from raw JSON
	var esiKillmail dto.ESIKillmail
	if err := json.Unmarshal(pkg.Killmail, &esiKillmail); err != nil {
		return fmt.Errorf("failed to parse ESI killmail: %w", err)
	}

	// Convert to internal killmail model
	killmail := p.convertToKillmail(&esiKillmail, pkg.ZKB.Hash)

	// Create ZKB metadata
	zkbMetadata := &zkbModels.ZKBMetadata{
		KillmailID:     pkg.KillID,
		LocationID:     pkg.ZKB.LocationID,
		Hash:           pkg.ZKB.Hash,
		FittedValue:    pkg.ZKB.FittedValue,
		DroppedValue:   pkg.ZKB.DroppedValue,
		DestroyedValue: pkg.ZKB.DestroyedValue,
		TotalValue:     pkg.ZKB.TotalValue,
		Points:         pkg.ZKB.Points,
		NPC:            pkg.ZKB.NPC,
		Solo:           pkg.ZKB.Solo,
		Awox:           pkg.ZKB.Awox,
		Labels:         pkg.ZKB.Labels,
		Href:           pkg.ZKB.Href,
		ProcessedAt:    time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Add to batch
	p.addToBatch(&ProcessedKillmail{
		Killmail:    killmail,
		ZKBMetadata: zkbMetadata,
		Package:     pkg,
	})

	// Process batch if full
	if p.shouldFlushBatch() {
		return p.flushBatch(ctx)
	}

	// Set timer for batch timeout
	p.setBatchTimer(ctx)

	return nil
}

// convertToKillmail converts ESI killmail to internal model
func (p *KillmailProcessor) convertToKillmail(esiKm *dto.ESIKillmail, hash string) *models.Killmail {
	km := &models.Killmail{
		KillmailID:    esiKm.KillmailID,
		KillmailTime:  esiKm.KillmailTime,
		KillmailHash:  hash,
		SolarSystemID: int64(esiKm.SolarSystemID),
		MoonID:        nil, // Not provided in RedisQ
		WarID:         nil, // Not provided in RedisQ
	}

	// Convert victim with type conversions
	var charID *int64
	if esiKm.Victim.CharacterID != nil {
		val := int64(*esiKm.Victim.CharacterID)
		charID = &val
	}
	var corpID *int64
	corpIDVal := int64(esiKm.Victim.CorporationID)
	corpID = &corpIDVal
	var allianceID *int64
	if esiKm.Victim.AllianceID != nil {
		val := int64(*esiKm.Victim.AllianceID)
		allianceID = &val
	}

	km.Victim = models.Victim{
		CharacterID:   charID,
		CorporationID: corpID,
		AllianceID:    allianceID,
		ShipTypeID:    int64(esiKm.Victim.ShipTypeID),
		DamageTaken:   int64(esiKm.Victim.DamageTaken),
	}

	// Convert position if present
	if esiKm.Victim.Position != nil {
		km.Victim.Position = &models.Position{
			X: esiKm.Victim.Position.X,
			Y: esiKm.Victim.Position.Y,
			Z: esiKm.Victim.Position.Z,
		}
	}

	// Convert items
	km.Victim.Items = p.convertItems(esiKm.Victim.Items)

	// Convert attackers with type conversions
	km.Attackers = make([]models.Attacker, len(esiKm.Attackers))
	for i, att := range esiKm.Attackers {
		var attCharID *int64
		if att.CharacterID != nil {
			val := int64(*att.CharacterID)
			attCharID = &val
		}
		var attCorpID *int64
		if att.CorporationID != nil {
			val := int64(*att.CorporationID)
			attCorpID = &val
		}
		var attAllianceID *int64
		if att.AllianceID != nil {
			val := int64(*att.AllianceID)
			attAllianceID = &val
		}
		var attShipTypeID *int64
		if att.ShipTypeID != nil {
			val := int64(*att.ShipTypeID)
			attShipTypeID = &val
		}
		var attWeaponTypeID *int64
		if att.WeaponTypeID != nil {
			val := int64(*att.WeaponTypeID)
			attWeaponTypeID = &val
		}

		km.Attackers[i] = models.Attacker{
			CharacterID:    attCharID,
			CorporationID:  attCorpID,
			AllianceID:     attAllianceID,
			ShipTypeID:     attShipTypeID,
			WeaponTypeID:   attWeaponTypeID,
			DamageDone:     int64(att.DamageDone),
			FinalBlow:      att.FinalBlow,
			SecurityStatus: float64(att.SecurityStatus),
		}
	}

	// Killmail model doesn't have CreatedAt/UpdatedAt fields
	// Timestamps are handled by MongoDB or other mechanisms

	return km
}

// convertItems recursively converts ESI items to internal model
func (p *KillmailProcessor) convertItems(esiItems []dto.ESIItem) []models.Item {
	if len(esiItems) == 0 {
		return nil
	}

	items := make([]models.Item, len(esiItems))
	for i, esiItem := range esiItems {
		var qtyDropped *int64
		if esiItem.QuantityDropped != nil {
			val := int64(*esiItem.QuantityDropped)
			qtyDropped = &val
		}
		var qtyDestroyed *int64
		if esiItem.QuantityDestroyed != nil {
			val := int64(*esiItem.QuantityDestroyed)
			qtyDestroyed = &val
		}

		item := models.Item{
			ItemTypeID:        int64(esiItem.ItemTypeID),
			Singleton:         int64(esiItem.SingletonID),
			Flag:              int64(esiItem.Flag),
			QuantityDropped:   qtyDropped,
			QuantityDestroyed: qtyDestroyed,
		}

		// Recursively convert nested items
		if len(esiItem.Items) > 0 {
			item.Items = p.convertItems(esiItem.Items)
		}

		items[i] = item
	}

	return items
}

// addToBatch adds a processed killmail to the batch
func (p *KillmailProcessor) addToBatch(processed *ProcessedKillmail) {
	p.batchMu.Lock()
	defer p.batchMu.Unlock()

	p.batch = append(p.batch, processed)
}

// shouldFlushBatch checks if the batch should be flushed
func (p *KillmailProcessor) shouldFlushBatch() bool {
	p.batchMu.Lock()
	defer p.batchMu.Unlock()

	return len(p.batch) >= p.batchSize
}

// setBatchTimer sets a timer to flush the batch after a timeout
func (p *KillmailProcessor) setBatchTimer(ctx context.Context) {
	p.batchMu.Lock()
	defer p.batchMu.Unlock()

	// Cancel existing timer if present
	if p.batchTimer != nil {
		p.batchTimer.Stop()
	}

	// Set new timer for 5 seconds
	p.batchTimer = time.AfterFunc(5*time.Second, func() {
		if err := p.flushBatch(ctx); err != nil {
			slog.Error("Failed to flush batch on timer", "error", err)
		}
	})
}

// flushBatch processes and stores all killmails in the batch
func (p *KillmailProcessor) flushBatch(ctx context.Context) error {
	p.batchMu.Lock()
	defer p.batchMu.Unlock()

	// Cancel timer
	if p.batchTimer != nil {
		p.batchTimer.Stop()
		p.batchTimer = nil
	}

	// Check if batch is empty
	if len(p.batch) == 0 {
		return nil
	}

	// Extract data for batch operations
	killmails := make([]*models.Killmail, len(p.batch))
	zkbMetadata := make([]*zkbModels.ZKBMetadata, len(p.batch))

	for i, processed := range p.batch {
		killmails[i] = processed.Killmail
		zkbMetadata[i] = processed.ZKBMetadata
	}

	// Batch insert killmails
	if err := p.killmailRepo.CreateMany(ctx, killmails); err != nil {
		return fmt.Errorf("failed to insert killmails: %w", err)
	}

	// Batch insert ZKB metadata
	if err := p.zkbRepo.SaveZKBMetadataBatch(ctx, zkbMetadata); err != nil {
		return fmt.Errorf("failed to insert ZKB metadata: %w", err)
	}

	// Update timeseries aggregations and character stats
	for _, processed := range p.batch {
		if err := p.aggregator.UpdateTimeseries(ctx, processed.Killmail, processed.ZKBMetadata); err != nil {
			slog.Error("Failed to update timeseries", "error", err, "killmail_id", processed.Killmail.KillmailID)
		}

		// Update character stats for tracked ship categories
		if err := p.charStatsService.UpdateFromKillmail(ctx, processed.Killmail); err != nil {
			slog.Error("Failed to update character stats", "error", err, "killmail_id", processed.Killmail.KillmailID)
		}

		// Emit WebSocket event
		p.emitKillmailEvent(processed)
	}

	slog.Info("Batch processed successfully", "count", len(p.batch))

	// Clear batch
	p.batch = p.batch[:0]

	return nil
}

// emitKillmailEvent sends a WebSocket notification for a new killmail
func (p *KillmailProcessor) emitKillmailEvent(processed *ProcessedKillmail) {
	// Get system name from SDE
	systemName := ""
	if system, err := p.sdeService.GetSolarSystem(int(processed.Killmail.SolarSystemID)); err == nil {
		// SolarSystem doesn't have a direct name field
		// TODO: Resolve system name through NameID lookup
		systemName = fmt.Sprintf("System %d", processed.Killmail.SolarSystemID)
		_ = system // Acknowledge we got the system but can't use name yet
	}

	// Get ship type name from SDE
	shipTypeName := ""
	if shipType, err := p.sdeService.GetType(fmt.Sprintf("%d", processed.Killmail.Victim.ShipTypeID)); err == nil {
		if enName, ok := shipType.Name["en"]; ok {
			shipTypeName = enName
		}
	}

	// Get victim name (character, corp, or "Unknown")
	victimName := "Unknown"
	if processed.Killmail.Victim.CharacterID != nil {
		// TODO: Resolve character name from database or ESI
		victimName = fmt.Sprintf("Character %d", *processed.Killmail.Victim.CharacterID)
	} else {
		// TODO: Resolve corporation name
		victimName = fmt.Sprintf("Corporation %d", processed.Killmail.Victim.CorporationID)
	}

	// Create event payload
	event := map[string]interface{}{
		"killmail_id":     processed.Killmail.KillmailID,
		"timestamp":       processed.Killmail.KillmailTime,
		"solar_system_id": processed.Killmail.SolarSystemID,
		"system_name":     systemName,
		"victim_name":     victimName,
		"ship_type_id":    processed.Killmail.Victim.ShipTypeID,
		"ship_type_name":  shipTypeName,
		"total_value":     processed.ZKBMetadata.TotalValue,
		"points":          processed.ZKBMetadata.Points,
		"solo":            processed.ZKBMetadata.Solo,
		"npc":             processed.ZKBMetadata.NPC,
		"href":            processed.ZKBMetadata.Href,
	}

	// Emit to WebSocket (event prepared but WebSocket integration pending)
	if p.websocketService != nil {
		// TODO: Implement proper WebSocket broadcast method
		slog.Info("Would emit killmail event via WebSocket", "killmail_id", processed.Killmail.KillmailID, "event_data", len(event) > 0)
	}
}

// Flush ensures any remaining killmails in the batch are processed
func (p *KillmailProcessor) Flush(ctx context.Context) error {
	return p.flushBatch(ctx)
}
