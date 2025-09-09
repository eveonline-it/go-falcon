package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go-falcon/internal/assets/models"
	structureModels "go-falcon/internal/structures/models"
	"go-falcon/internal/structures/services"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AssetService handles asset operations
type AssetService struct {
	db               *mongo.Database
	eveGateway       *evegateway.Client
	sdeService       sde.SDEService
	structureService *services.StructureService
	redis            *redis.Client
	structureTracker *StructureAccessTracker
}

// NewAssetService creates a new asset service
func NewAssetService(db *mongo.Database, eveGateway *evegateway.Client, sdeService sde.SDEService, structureService *services.StructureService, redis *redis.Client) *AssetService {
	return &AssetService{
		db:               db,
		eveGateway:       eveGateway,
		sdeService:       sdeService,
		structureService: structureService,
		redis:            redis,
		structureTracker: NewStructureAccessTracker(redis),
	}
}

// GetCharacterAssets retrieves character assets from database only
func (s *AssetService) GetCharacterAssets(ctx context.Context, characterID int32, token string, locationID *int64) ([]*models.Asset, int, error) {
	// Get assets from database only - no ESI queries
	filter := bson.M{"character_id": characterID}
	if locationID != nil {
		filter["location_id"] = *locationID
	}

	cursor, err := s.db.Collection(models.AssetsCollection).Find(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query database: %w", err)
	}
	defer cursor.Close(ctx)

	var assets []*models.Asset
	if err := cursor.All(ctx, &assets); err != nil {
		return nil, 0, fmt.Errorf("failed to decode assets: %w", err)
	}

	// Return all assets without pagination
	total := len(assets)
	return assets, total, nil
}

// GetCorporationAssets retrieves corporation assets from database only
func (s *AssetService) GetCorporationAssets(ctx context.Context, corporationID, characterID int32, token string, locationID *int64, division *int) ([]*models.Asset, int, error) {
	// Get assets from database only - no ESI queries
	filter := bson.M{"corporation_id": corporationID}
	if locationID != nil {
		filter["location_id"] = *locationID
	}
	if division != nil {
		filter["location_flag"] = fmt.Sprintf("CorpSAG%d", *division)
	}

	cursor, err := s.db.Collection(models.AssetsCollection).Find(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query database: %w", err)
	}
	defer cursor.Close(ctx)

	var assets []*models.Asset
	if err := cursor.All(ctx, &assets); err != nil {
		return nil, 0, fmt.Errorf("failed to decode assets: %w", err)
	}

	// Return all assets without pagination
	total := len(assets)
	return assets, total, nil
}

// isContainer checks if a type ID is a container
func (s *AssetService) isContainer(typeID int32) bool {
	for _, containerID := range models.ContainerTypeIDs {
		if typeID == containerID {
			return true
		}
	}

	// Check if it's a ship (ships can contain items)
	// We would need to check the group or category ID to determine if it's a ship
	// For now, we'll just check containers

	return false
}

// determineLocationType determines the type of location
func (s *AssetService) determineLocationType(locationID int64) string {
	if locationID < 100000000 {
		return models.LocationTypeStation
	} else if locationID < 2000000000000 {
		return models.LocationTypeStructure
	} else {
		return models.LocationTypeOther
	}
}

// RefreshCharacterAssets forces a refresh of character assets from ESI
func (s *AssetService) RefreshCharacterAssets(ctx context.Context, characterID int32, token string) (int, int, int, error) {
	// Check ESI error limits before starting
	if err := s.eveGateway.CheckErrorLimits(); err != nil {
		return 0, 0, 0, fmt.Errorf("cannot refresh assets: %w", err)
	}

	// Log current error limits
	limits := s.eveGateway.GetErrorLimits()
	slog.InfoContext(ctx, "Starting asset refresh",
		"character_id", characterID,
		"esi_errors_remaining", limits.Remain,
		"esi_error_reset", limits.Reset)
	// Get existing assets for statistics before refresh
	var existingAssets []*models.Asset
	cursor, err := s.db.Collection(models.AssetsCollection).Find(ctx, bson.M{"character_id": characterID})
	if err == nil {
		cursor.All(ctx, &existingAssets)
		cursor.Close(ctx)
	}

	// Fetch fresh data from ESI
	esiAssets, err := s.eveGateway.Assets.GetCharacterAssets(ctx, characterID, token)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to fetch assets from ESI: %w", err)
	}

	// Process assets in memory (don't save to DB yet)
	newAssets, err := s.processESIAssetsInMemory(ctx, esiAssets, characterID, 0, token)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to process ESI assets: %w", err)
	}

	// Calculate statistics before making changes
	updated := len(newAssets)
	newItems := 0
	removedItems := 0

	// Create maps for comparison
	existingMap := make(map[int64]bool)
	for _, asset := range existingAssets {
		existingMap[asset.ItemID] = true
	}

	newMap := make(map[int64]bool)
	for _, asset := range newAssets {
		newMap[asset.ItemID] = true
		if !existingMap[asset.ItemID] {
			newItems++
		}
	}

	// Count removed items
	for _, asset := range existingAssets {
		if !newMap[asset.ItemID] {
			removedItems++
		}
	}

	// Atomic replacement: Delete old assets, then insert new ones
	// 1. Delete all existing character assets
	_, err = s.db.Collection(models.AssetsCollection).DeleteMany(ctx, bson.M{
		"character_id": characterID,
	})
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to delete existing assets: %w", err)
	}

	// 2. Insert all new assets (if any)
	if len(newAssets) > 0 {
		err = s.insertAssets(ctx, newAssets)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to insert new assets: %w", err)
		}
	}

	return updated, newItems, removedItems, nil
}

// processESIAssetsInMemory processes raw ESI assets and enriches them with additional data (memory only)
func (s *AssetService) processESIAssetsInMemory(ctx context.Context, esiAssets []map[string]any, characterID, corporationID int32, token string) ([]*models.Asset, error) {
	assets := make([]*models.Asset, 0, len(esiAssets))

	// Create a map for container hierarchy
	containerMap := make(map[int64]*models.Asset)

	// Track structures we've already tried to fetch to avoid repeated 403s
	failedStructures := make(map[int64]bool)

	// First pass: create all assets and identify containers
	for _, esiAsset := range esiAssets {
		// Parse fields from map[string]any
		itemID, _ := esiAsset["item_id"].(int64)
		if itemID == 0 {
			if itemIDFloat, ok := esiAsset["item_id"].(float64); ok {
				itemID = int64(itemIDFloat)
			}
		}

		typeID, _ := esiAsset["type_id"].(int32)
		if typeID == 0 {
			if typeIDFloat, ok := esiAsset["type_id"].(float64); ok {
				typeID = int32(typeIDFloat)
			}
		}

		locationID, _ := esiAsset["location_id"].(int64)
		if locationID == 0 {
			if locationIDFloat, ok := esiAsset["location_id"].(float64); ok {
				locationID = int64(locationIDFloat)
			}
		}

		locationFlag, _ := esiAsset["location_flag"].(string)

		quantity, _ := esiAsset["quantity"].(int32)
		if quantity == 0 {
			if quantityFloat, ok := esiAsset["quantity"].(float64); ok {
				quantity = int32(quantityFloat)
			}
		}

		isSingleton, _ := esiAsset["is_singleton"].(bool)

		var isBlueprintCopy bool
		if blueprintCopyPtr, ok := esiAsset["is_blueprint_copy"].(*bool); ok && blueprintCopyPtr != nil {
			isBlueprintCopy = *blueprintCopyPtr
		} else if blueprintCopyBool, ok := esiAsset["is_blueprint_copy"].(bool); ok {
			isBlueprintCopy = blueprintCopyBool
		}

		asset := &models.Asset{
			CharacterID:     characterID,
			CorporationID:   corporationID,
			ItemID:          itemID,
			TypeID:          typeID,
			LocationID:      locationID,
			LocationFlag:    locationFlag,
			Quantity:        quantity,
			IsSingleton:     isSingleton,
			IsBlueprintCopy: isBlueprintCopy,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		// Check if it's a container
		asset.IsContainer = s.isContainer(asset.TypeID)

		// Determine location type
		asset.LocationType = s.determineLocationType(asset.LocationID)

		// Get type information from SDE
		if typeInfo, err := s.sdeService.GetType(fmt.Sprintf("%d", asset.TypeID)); err == nil {
			if name, ok := typeInfo.Name["en"]; ok {
				asset.TypeName = name
			}
		}

		assets = append(assets, asset)
		containerMap[asset.ItemID] = asset
	}

	// Second pass: establish parent-child relationships
	for _, asset := range assets {
		// Check if this item is inside a container
		if parent, exists := containerMap[asset.LocationID]; exists {
			asset.ParentItemID = &parent.ItemID
			asset.LocationID = parent.LocationID // Use parent's location
		}
	}

	// Third pass: Collect unique structures and intelligently check them
	uniqueStructures := make(map[int64]bool)
	for _, asset := range assets {
		uniqueStructures[asset.LocationID] = true
	}

	slog.InfoContext(ctx, "Preparing to enrich assets with structure data",
		"total_assets", len(assets),
		"unique_structures", len(uniqueStructures))

	// Get known failed structures from Redis tracker
	knownFailedStructures := make(map[int64]bool)
	if s.structureTracker != nil {
		// Get structures to retry based on intelligent selection
		retryStructures, err := s.structureTracker.GetRetryStructures(ctx, characterID, 10)
		if err != nil {
			slog.WarnContext(ctx, "Failed to get retry structures from tracker",
				"error", err)
		} else if len(retryStructures) > 0 {
			slog.DebugContext(ctx, "Selected structures for retry",
				"character_id", characterID,
				"retry_count", len(retryStructures),
				"retry_structure_ids", retryStructures)
		}

		// Mark known failed structures (except those selected for retry)
		checkedInRedis := 0
		foundInRedis := 0
		for structureID := range uniqueStructures {
			// Check if this structure is known to be inaccessible
			key := fmt.Sprintf("falcon:assets:failed_structures:%d:%d", characterID, structureID)
			checkedInRedis++
			if exists, _ := s.redis.Exists(ctx, key).Result(); exists > 0 {
				foundInRedis++
				// Check if it's selected for retry
				isRetry := false
				for _, retryID := range retryStructures {
					if retryID == structureID {
						isRetry = true
						break
					}
				}
				if !isRetry {
					knownFailedStructures[structureID] = true
					slog.DebugContext(ctx, "Structure marked as known forbidden (will skip)",
						"character_id", characterID,
						"structure_id", structureID,
						"redis_key", key)
				} else {
					slog.DebugContext(ctx, "Forbidden structure selected for retry",
						"character_id", characterID,
						"structure_id", structureID,
						"redis_key", key)
				}
			}
		}

		if foundInRedis > 0 {
			slog.DebugContext(ctx, "Redis forbidden structure check summary",
				"character_id", characterID,
				"structures_checked", checkedInRedis,
				"forbidden_found", foundInRedis,
				"will_skip", len(knownFailedStructures))
		}
	}

	// Pre-fetch structure data with intelligent filtering
	structureCache := make(map[int64]*structureModels.Structure)
	structuresChecked := 0
	new403Errors := 0
	max403Errors := 20 // Stop after 20 new 403 errors to protect ESI limits

	for locationID := range uniqueStructures {
		// Skip known failed structures (unless selected for retry)
		if knownFailedStructures[locationID] {
			slog.DebugContext(ctx, "Skipping known forbidden structure",
				"character_id", characterID,
				"structure_id", locationID,
				"reason", "Previously failed with 403, not selected for retry")
			continue
		}

		// Stop if we've hit too many new 403 errors
		if new403Errors >= max403Errors {
			slog.WarnContext(ctx, "Stopping structure enrichment - too many 403 errors",
				"new_403_errors", new403Errors,
				"max_allowed", max403Errors)
			break
		}

		// Check error limits before EVERY structure call
		if err := s.eveGateway.CheckErrorLimits(); err != nil {
			slog.WarnContext(ctx, "Stopping structure enrichment due to ESI error limit",
				"error", err,
				"structures_checked", structuresChecked,
				"structures_total", len(uniqueStructures))
			break
		}

		// Try to get structure data
		structure, err := s.structureService.GetStructure(ctx, locationID, token)
		if err != nil {
			if strings.Contains(err.Error(), "authentication failed") {
				return nil, fmt.Errorf("authentication failed during structure fetch: %w", err)
			}
			if strings.Contains(err.Error(), "access denied") {
				new403Errors++
				failedStructures[locationID] = true

				slog.DebugContext(ctx, "New forbidden structure encountered (403)",
					"character_id", characterID,
					"structure_id", locationID,
					"new_403_count", new403Errors,
					"error", err.Error())

				// Record failed access in Redis tracker
				if s.structureTracker != nil {
					if err := s.structureTracker.RecordFailedAccess(ctx, characterID, locationID, "403 Forbidden - Access denied"); err != nil {
						slog.WarnContext(ctx, "Failed to record structure access failure",
							"structure_id", locationID,
							"error", err)
					} else {
						slog.DebugContext(ctx, "Recorded forbidden structure in Redis",
							"character_id", characterID,
							"structure_id", locationID,
							"redis_key", fmt.Sprintf("falcon:assets:failed_structures:%d:%d", characterID, locationID))
					}
				}

				// Check if we're getting too many 403s in a row
				if new403Errors > 10 && float64(new403Errors)/float64(structuresChecked) > 0.5 {
					slog.WarnContext(ctx, "Too many access denied errors, stopping structure enrichment",
						"new_403_errors", new403Errors,
						"structures_checked", structuresChecked)
					break
				}
			}
			// Continue with next structure
		} else if structure != nil {
			structureCache[locationID] = structure

			// Record successful access (might have been previously failed)
			if s.structureTracker != nil {
				if err := s.structureTracker.RecordSuccessfulAccess(ctx, characterID, locationID); err != nil {
					slog.WarnContext(ctx, "Failed to record structure access success",
						"structure_id", locationID,
						"error", err)
				}
			}
		}
		structuresChecked++
	}

	// Update ESI error budget tracker
	if s.structureTracker != nil && new403Errors > 0 {
		if err := s.structureTracker.IncrementESIErrors(ctx, new403Errors); err != nil {
			slog.WarnContext(ctx, "Failed to update ESI error budget",
				"error", err)
		}
	}

	// Fourth pass: Apply structure data to assets
	for _, asset := range assets {
		if structure, exists := structureCache[asset.LocationID]; exists {
			asset.LocationName = structure.Name
			asset.SolarSystemID = structure.SolarSystemID
			asset.SolarSystemName = structure.SolarSystemName
			asset.RegionID = structure.RegionID
			asset.RegionName = structure.RegionName
		}
		// Assets without structure data will just have the location ID
	}

	// Log summary
	if len(failedStructures) > 0 || len(knownFailedStructures) > 0 {
		slog.InfoContext(ctx, "Asset enrichment completed with structure access tracking",
			"total_assets", len(assets),
			"new_inaccessible", len(failedStructures),
			"known_inaccessible", len(knownFailedStructures),
			"structures_checked", structuresChecked,
			"reason", "Character lacks docking rights (403)")

		// Debug log with more details
		if len(failedStructures) > 0 {
			failedIDs := make([]int64, 0, len(failedStructures))
			for id := range failedStructures {
				failedIDs = append(failedIDs, id)
			}
			slog.DebugContext(ctx, "New forbidden structures encountered in this refresh",
				"character_id", characterID,
				"structure_ids", failedIDs)
		}

		if len(knownFailedStructures) > 0 {
			skippedIDs := make([]int64, 0, len(knownFailedStructures))
			for id := range knownFailedStructures {
				skippedIDs = append(skippedIDs, id)
			}
			slog.DebugContext(ctx, "Known forbidden structures skipped in this refresh",
				"character_id", characterID,
				"structure_ids", skippedIDs)
		}
	}

	// Update tracker metrics
	if s.structureTracker != nil {
		s.structureTracker.updateMetrics(ctx, "total_structures_checked", structuresChecked)
	}

	return assets, nil
}

// insertAssets performs a bulk insert of assets into the database
func (s *AssetService) insertAssets(ctx context.Context, assets []*models.Asset) error {
	if len(assets) == 0 {
		return nil
	}

	// Convert to interface slice for InsertMany
	documents := make([]interface{}, len(assets))
	for i, asset := range assets {
		// Ensure each asset has a new ObjectID
		asset.ID = primitive.NewObjectID()
		documents[i] = asset
	}

	// Insert all assets in one operation
	_, err := s.db.Collection(models.AssetsCollection).InsertMany(ctx, documents)
	return err
}

// GetAssetSummary returns a summary of assets
func (s *AssetService) GetAssetSummary(ctx context.Context, characterID, corporationID int32) (*models.AssetSnapshot, error) {
	filter := bson.M{}
	if characterID > 0 {
		filter["character_id"] = characterID
	}
	if corporationID > 0 {
		filter["corporation_id"] = corporationID
	}

	// Aggregate assets
	pipeline := []bson.M{
		{"$match": filter},
		{"$group": bson.M{
			"_id":          "$location_id",
			"total_value":  bson.M{"$sum": "$total_value"},
			"item_count":   bson.M{"$sum": "$quantity"},
			"unique_types": bson.M{"$addToSet": "$type_id"},
		}},
		{"$group": bson.M{
			"_id":          nil,
			"total_value":  bson.M{"$sum": "$total_value"},
			"item_count":   bson.M{"$sum": "$item_count"},
			"unique_types": bson.M{"$addToSet": "$unique_types"},
		}},
	}

	cursor, err := s.db.Collection(models.AssetsCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return &models.AssetSnapshot{
			CharacterID:   characterID,
			CorporationID: corporationID,
			SnapshotTime:  time.Now(),
		}, nil
	}

	// Create snapshot
	snapshot := &models.AssetSnapshot{
		ID:            primitive.NewObjectID(),
		CharacterID:   characterID,
		CorporationID: corporationID,
		TotalValue:    result[0]["total_value"].(float64),
		ItemCount:     result[0]["item_count"].(int32),
		UniqueTypes:   int32(len(result[0]["unique_types"].([]interface{}))),
		SnapshotTime:  time.Now(),
		CreatedAt:     time.Now(),
	}

	// Save snapshot
	s.db.Collection(models.AssetSnapshotsCollection).InsertOne(ctx, snapshot)

	return snapshot, nil
}

// Asset tracking methods

// CreateAssetTracking creates a new asset tracking configuration
func (s *AssetService) CreateAssetTracking(ctx context.Context, tracking *models.AssetTracking) error {
	tracking.ID = primitive.NewObjectID()
	tracking.CreatedAt = time.Now()
	tracking.UpdatedAt = time.Now()

	_, err := s.db.Collection(models.AssetTrackingCollection).InsertOne(ctx, tracking)
	return err
}

// UpdateAssetTracking updates an existing asset tracking configuration
func (s *AssetService) UpdateAssetTracking(ctx context.Context, trackingID string, updates bson.M) error {
	objectID, err := primitive.ObjectIDFromHex(trackingID)
	if err != nil {
		return err
	}

	updates["updated_at"] = time.Now()

	_, err = s.db.Collection(models.AssetTrackingCollection).UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)
	return err
}

// DeleteAssetTracking deletes an asset tracking configuration
func (s *AssetService) DeleteAssetTracking(ctx context.Context, trackingID string) error {
	objectID, err := primitive.ObjectIDFromHex(trackingID)
	if err != nil {
		return err
	}

	_, err = s.db.Collection(models.AssetTrackingCollection).DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// GetAssetTracking retrieves asset tracking configurations
func (s *AssetService) GetAssetTracking(ctx context.Context, filter bson.M) ([]*models.AssetTracking, error) {
	cursor, err := s.db.Collection(models.AssetTrackingCollection).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var trackings []*models.AssetTracking
	if err := cursor.All(ctx, &trackings); err != nil {
		return nil, err
	}

	return trackings, nil
}

// GetStructureAccessStats returns statistics about failed structure access for monitoring
func (s *AssetService) GetStructureAccessStats(ctx context.Context, characterID *int32) (map[string]interface{}, error) {
	if s.structureTracker == nil {
		return nil, fmt.Errorf("structure tracker not initialized")
	}

	stats := make(map[string]interface{})

	if characterID != nil {
		// Get stats for specific character
		charStats, err := s.structureTracker.GetFailedStructureStats(ctx, *characterID)
		if err != nil {
			return nil, fmt.Errorf("failed to get character stats: %w", err)
		}
		stats["character_stats"] = charStats
		stats["character_id"] = *characterID
	} else {
		// Get global stats
		pattern := "falcon:assets:failed_structures:*"
		keys, err := s.redis.Keys(ctx, pattern).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get failed structure keys: %w", err)
		}

		// Aggregate stats
		characterMap := make(map[int32]int)
		totalFailed := len(keys)

		for _, key := range keys {
			var charID int32
			var structID int64
			fmt.Sscanf(key, "falcon:assets:failed_structures:%d:%d", &charID, &structID)
			if charID > 0 {
				characterMap[charID]++
			}
		}

		stats["total_failed_structures"] = totalFailed
		stats["affected_characters"] = len(characterMap)
		stats["character_breakdown"] = characterMap
	}

	// Add current error budget info
	stats["remaining_error_budget"] = s.structureTracker.GetRemainingErrorBudget(ctx)

	// Get today's metrics
	date := time.Now().Format("2006-01-02")
	metricsKey := fmt.Sprintf("falcon:assets:metrics:%s", date)
	metricsData, err := s.redis.Get(ctx, metricsKey).Result()
	if err == nil {
		var metrics map[string]interface{}
		if err := json.Unmarshal([]byte(metricsData), &metrics); err == nil {
			stats["today_metrics"] = metrics
		}
	}

	return stats, nil
}

// ProcessStructureAccessRetry processes scheduled retries for failed structure access
func (s *AssetService) ProcessStructureAccessRetry(ctx context.Context) error {
	if s.structureTracker == nil {
		return fmt.Errorf("structure tracker not initialized")
	}

	slog.InfoContext(ctx, "Starting scheduled structure access retry processing")

	// Get all unique character IDs from failed structures
	pattern := "falcon:assets:failed_structures:*"
	keys, err := s.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get failed structure keys: %w", err)
	}

	// Track unique characters
	characterMap := make(map[int32]bool)
	for _, key := range keys {
		var characterID int32
		var structureID int64
		fmt.Sscanf(key, "falcon:assets:failed_structures:%d:%d", &characterID, &structureID)
		if characterID > 0 {
			characterMap[characterID] = true
		}
	}

	slog.InfoContext(ctx, "Found characters with failed structures",
		"character_count", len(characterMap),
		"total_failed_structures", len(keys))

	// Process retries for each character (limit to 5 characters per run)
	processedCharacters := 0
	totalRetries := 0

	for characterID := range characterMap {
		if processedCharacters >= 5 {
			break // Limit processing to avoid long-running tasks
		}

		// Get retry candidates for this character
		retryStructures, err := s.structureTracker.GetRetryStructures(ctx, characterID, 5)
		if err != nil {
			slog.WarnContext(ctx, "Failed to get retry structures",
				"character_id", characterID,
				"error", err)
			continue
		}

		if len(retryStructures) == 0 {
			continue
		}

		slog.InfoContext(ctx, "Processing structure retries for character",
			"character_id", characterID,
			"retry_count", len(retryStructures))

		// Process each retry structure
		for _, structureID := range retryStructures {
			totalRetries++

			// Note: Actual retry would happen during next asset refresh
			// This task just selects and marks structures for retry
			slog.DebugContext(ctx, "Marked structure for retry",
				"character_id", characterID,
				"structure_id", structureID)
		}

		processedCharacters++
	}

	// Get and log remaining error budget
	remainingBudget := s.structureTracker.GetRemainingErrorBudget(ctx)

	slog.InfoContext(ctx, "Completed scheduled structure access retry processing",
		"characters_processed", processedCharacters,
		"total_retries_marked", totalRetries,
		"remaining_error_budget", remainingBudget)

	return nil
}

// ProcessAssetTracking processes active asset tracking configurations
func (s *AssetService) ProcessAssetTracking(ctx context.Context) error {
	// Get all enabled tracking configurations
	trackings, err := s.GetAssetTracking(ctx, bson.M{"enabled": true})
	if err != nil {
		return err
	}

	for _, tracking := range trackings {
		// Get assets for tracked locations
		filter := bson.M{
			"character_id": tracking.CharacterID,
			"location_id":  bson.M{"$in": tracking.LocationIDs},
		}

		if tracking.CorporationID > 0 {
			filter["corporation_id"] = tracking.CorporationID
		}

		if len(tracking.TypeIDs) > 0 {
			filter["type_id"] = bson.M{"$in": tracking.TypeIDs}
		}

		// Calculate total value
		pipeline := []bson.M{
			{"$match": filter},
			{"$group": bson.M{
				"_id":         nil,
				"total_value": bson.M{"$sum": "$total_value"},
			}},
		}

		cursor, err := s.db.Collection(models.AssetsCollection).Aggregate(ctx, pipeline)
		if err != nil {
			continue
		}

		var result []bson.M
		cursor.All(ctx, &result)
		cursor.Close(ctx)

		if len(result) > 0 {
			totalValue := result[0]["total_value"].(float64)

			// Check threshold
			if tracking.NotifyThreshold > 0 && tracking.LastValue > 0 {
				change := totalValue - tracking.LastValue
				if change > tracking.NotifyThreshold || change < -tracking.NotifyThreshold {
					// TODO: Send notification
				}
			}

			// Update tracking
			s.UpdateAssetTracking(ctx, tracking.ID.Hex(), bson.M{
				"last_checked": time.Now(),
				"last_value":   totalValue,
			})
		}
	}

	return nil
}
