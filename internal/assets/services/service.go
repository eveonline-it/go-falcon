package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go-falcon/internal/assets/models"
	structureModels "go-falcon/internal/structures/models"
	"go-falcon/internal/structures/services"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AssetService handles asset operations
type AssetService struct {
	db               *mongo.Database
	eveGateway       *evegateway.Client
	sdeService       sde.SDEService
	structureService *services.StructureService
}

// NewAssetService creates a new asset service
func NewAssetService(db *mongo.Database, eveGateway *evegateway.Client, sdeService sde.SDEService, structureService *services.StructureService) *AssetService {
	return &AssetService{
		db:               db,
		eveGateway:       eveGateway,
		sdeService:       sdeService,
		structureService: structureService,
	}
}

// GetCharacterAssets retrieves character assets
func (s *AssetService) GetCharacterAssets(ctx context.Context, characterID int32, token string, locationID *int64) ([]*models.Asset, int, error) {
	// Try to get assets from database first
	filter := bson.M{"character_id": characterID}
	cursor, err := s.db.Collection(models.AssetsCollection).Find(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query database: %w", err)
	}
	defer cursor.Close(ctx)

	var assets []*models.Asset
	if err := cursor.All(ctx, &assets); err != nil {
		return nil, 0, fmt.Errorf("failed to decode assets: %w", err)
	}

	// If no assets found or data is old, fetch from ESI
	if len(assets) == 0 || (len(assets) > 0 && time.Since(assets[0].UpdatedAt) > 30*time.Minute) {
		// Fetch from ESI
		esiAssets, err := s.eveGateway.Assets.GetCharacterAssets(ctx, characterID, token)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch assets from ESI: %w", err)
		}

		// Process and save assets
		assets, err = s.processESIAssets(ctx, esiAssets, characterID, 0, token)
		if err != nil {
			return nil, 0, err
		}
	}

	// Filter by location if specified
	if locationID != nil {
		filtered := make([]*models.Asset, 0)
		for _, asset := range assets {
			if asset.LocationID == *locationID {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}

	// Return all assets without pagination
	total := len(assets)
	return assets, total, nil
}

// GetCorporationAssets retrieves corporation assets
func (s *AssetService) GetCorporationAssets(ctx context.Context, corporationID, characterID int32, token string, locationID *int64, division *int) ([]*models.Asset, int, error) {
	// Try to get assets from database first
	filter := bson.M{"corporation_id": corporationID}
	cursor, err := s.db.Collection(models.AssetsCollection).Find(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query database: %w", err)
	}
	defer cursor.Close(ctx)

	var assets []*models.Asset
	if err := cursor.All(ctx, &assets); err != nil {
		return nil, 0, fmt.Errorf("failed to decode assets: %w", err)
	}

	// If no assets found or data is old, fetch from ESI
	if len(assets) == 0 || (len(assets) > 0 && time.Since(assets[0].UpdatedAt) > 30*time.Minute) {
		// Fetch from ESI (requires character with appropriate roles)
		esiAssets, err := s.eveGateway.Assets.GetCorporationAssets(ctx, corporationID, token)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch corporation assets from ESI: %w", err)
		}

		// Process and save assets
		assets, err = s.processESIAssets(ctx, esiAssets, characterID, corporationID, token)
		if err != nil {
			return nil, 0, err
		}
	}

	// Filter by location if specified
	if locationID != nil {
		filtered := make([]*models.Asset, 0)
		for _, asset := range assets {
			if asset.LocationID == *locationID {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}

	// Filter by division if specified
	if division != nil {
		divisionFlag := fmt.Sprintf("CorpSAG%d", *division)
		filtered := make([]*models.Asset, 0)
		for _, asset := range assets {
			if asset.LocationFlag == divisionFlag {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}

	// Return all assets without pagination
	total := len(assets)
	return assets, total, nil
}

// processESIAssets processes raw ESI assets and enriches them with additional data
func (s *AssetService) processESIAssets(ctx context.Context, esiAssets []map[string]any, characterID, corporationID int32, token string) ([]*models.Asset, error) {
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

	// Third pass: Collect unique structures and pre-fetch them
	uniqueStructures := make(map[int64]bool)
	for _, asset := range assets {
		uniqueStructures[asset.LocationID] = true
	}

	slog.InfoContext(ctx, "Preparing to enrich assets with structure data",
		"total_assets", len(assets),
		"unique_structures", len(uniqueStructures))

	// Pre-fetch structure data with aggressive error limit checking
	structureCache := make(map[int64]*structureModels.Structure)
	structuresChecked := 0
	max403Errors := 20 // Stop after 20 403 errors to protect ESI limits

	for locationID := range uniqueStructures {
		// Stop if we've hit too many 403 errors
		if len(failedStructures) >= max403Errors {
			slog.WarnContext(ctx, "Stopping structure enrichment - too many 403 errors",
				"failed_structures", len(failedStructures),
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
				failedStructures[locationID] = true
				// Check if we're getting too many 403s in a row
				if len(failedStructures) > 10 && float64(len(failedStructures))/float64(structuresChecked) > 0.5 {
					slog.WarnContext(ctx, "Too many access denied errors, stopping structure enrichment",
						"failed_structures", len(failedStructures),
						"structures_checked", structuresChecked)
					break
				}
			}
			// Continue with next structure
		} else if structure != nil {
			structureCache[locationID] = structure
		}
		structuresChecked++
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

	// Log summary of structures we couldn't access
	if len(failedStructures) > 0 {
		slog.InfoContext(ctx, "Asset enrichment completed with some structures inaccessible",
			"total_assets", len(assets),
			"inaccessible_structures", len(failedStructures),
			"reason", "Character lacks docking rights (403)")
	}

	// Save all assets to database
	if err := s.saveAssets(ctx, assets); err != nil {
		return nil, err
	}

	return assets, nil
}

// enrichLocationData enriches asset with location information
func (s *AssetService) enrichLocationData(ctx context.Context, asset *models.Asset, token string) error {
	// Get structure/station information
	structure, err := s.structureService.GetStructure(ctx, asset.LocationID, token)
	if err != nil {
		// Check if it's an authentication error (401) - these should stop processing
		if strings.Contains(err.Error(), "authentication failed") {
			return fmt.Errorf("authentication failed for structure %d: %w", asset.LocationID, err)
		}
		// For access denied (403), return the error so caller can track it
		if strings.Contains(err.Error(), "access denied") {
			return fmt.Errorf("access denied to structure %d", asset.LocationID)
		}
		// For other errors, just log and continue
		slog.DebugContext(ctx, "Could not fetch structure information",
			"structure_id", asset.LocationID,
			"error", err)
		return nil
	}

	if structure != nil {
		asset.LocationName = structure.Name
		asset.SolarSystemID = structure.SolarSystemID
		asset.SolarSystemName = structure.SolarSystemName
		asset.RegionID = structure.RegionID
		asset.RegionName = structure.RegionName
	}
	return nil
}

// enrichMarketData enriches asset with market price information
func (s *AssetService) enrichMarketData(ctx context.Context, asset *models.Asset) {
	// Market price enrichment removed - this would need to be implemented
	// with a market data service or external API if needed
	// For now, set default values
	asset.MarketPrice = 0
	asset.TotalValue = 0
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

// saveAssets saves assets to database
func (s *AssetService) saveAssets(ctx context.Context, assets []*models.Asset) error {
	if len(assets) == 0 {
		return nil
	}

	// Prepare bulk write operations
	operations := make([]mongo.WriteModel, len(assets))
	for i, asset := range assets {
		filter := bson.M{
			"character_id": asset.CharacterID,
			"item_id":      asset.ItemID,
		}

		// Create update document with all fields except _id
		update := bson.M{
			"$set": bson.M{
				"corporation_id":    asset.CorporationID,
				"type_id":           asset.TypeID,
				"type_name":         asset.TypeName,
				"location_id":       asset.LocationID,
				"location_type":     asset.LocationType,
				"location_flag":     asset.LocationFlag,
				"location_name":     asset.LocationName,
				"quantity":          asset.Quantity,
				"is_singleton":      asset.IsSingleton,
				"is_blueprint_copy": asset.IsBlueprintCopy,
				"name":              asset.Name,
				"market_price":      asset.MarketPrice,
				"total_value":       asset.TotalValue,
				"solar_system_id":   asset.SolarSystemID,
				"solar_system_name": asset.SolarSystemName,
				"region_id":         asset.RegionID,
				"region_name":       asset.RegionName,
				"parent_item_id":    asset.ParentItemID,
				"is_container":      asset.IsContainer,
				"created_at":        asset.CreatedAt,
				"updated_at":        asset.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"_id": primitive.NewObjectID(),
			},
		}

		operations[i] = mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)
	}

	// Execute bulk write
	opts := options.BulkWrite().SetOrdered(false)
	_, err := s.db.Collection(models.AssetsCollection).BulkWrite(ctx, operations, opts)
	return err
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

	// Third pass: Collect unique structures and pre-fetch them
	uniqueStructures := make(map[int64]bool)
	for _, asset := range assets {
		uniqueStructures[asset.LocationID] = true
	}

	slog.InfoContext(ctx, "Preparing to enrich assets with structure data",
		"total_assets", len(assets),
		"unique_structures", len(uniqueStructures))

	// Pre-fetch structure data with aggressive error limit checking
	structureCache := make(map[int64]*structureModels.Structure)
	structuresChecked := 0
	max403Errors := 20 // Stop after 20 403 errors to protect ESI limits

	for locationID := range uniqueStructures {
		// Stop if we've hit too many 403 errors
		if len(failedStructures) >= max403Errors {
			slog.WarnContext(ctx, "Stopping structure enrichment - too many 403 errors",
				"failed_structures", len(failedStructures),
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
				failedStructures[locationID] = true
				// Check if we're getting too many 403s in a row
				if len(failedStructures) > 10 && float64(len(failedStructures))/float64(structuresChecked) > 0.5 {
					slog.WarnContext(ctx, "Too many access denied errors, stopping structure enrichment",
						"failed_structures", len(failedStructures),
						"structures_checked", structuresChecked)
					break
				}
			}
			// Continue with next structure
		} else if structure != nil {
			structureCache[locationID] = structure
		}
		structuresChecked++
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

	// Log summary of structures we couldn't access
	if len(failedStructures) > 0 {
		slog.InfoContext(ctx, "Asset enrichment completed with some structures inaccessible",
			"total_assets", len(assets),
			"inaccessible_structures", len(failedStructures),
			"reason", "Character lacks docking rights (403)")
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
