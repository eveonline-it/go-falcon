package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go-falcon/internal/structures/models"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Cache keys
	structureCachePrefix = "c:structure:"
	structureCacheTTL    = 2 * time.Hour

	// NPC station threshold
	npcStationThreshold = 100000000
)

// getFloat64FromInterface safely extracts float64 from interface{}
func getFloat64FromInterface(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// StructureService handles structure operations
type StructureService struct {
	db         *mongo.Database
	redis      *redis.Client
	eveGateway *evegateway.Client
	sdeService sde.SDEService
}

// NewStructureService creates a new structure service
func NewStructureService(db *mongo.Database, redis *redis.Client, eveGateway *evegateway.Client, sdeService sde.SDEService) *StructureService {
	return &StructureService{
		db:         db,
		redis:      redis,
		eveGateway: eveGateway,
		sdeService: sdeService,
	}
}

// GetStructure retrieves structure information
func (s *StructureService) GetStructure(ctx context.Context, structureID int64, token string) (*models.Structure, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%d", structureCachePrefix, structureID)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var structure models.Structure
		if err := json.Unmarshal([]byte(cached), &structure); err == nil {
			return &structure, nil
		}
	}

	// Check if it's an NPC station or player structure
	if structureID < npcStationThreshold {
		return s.getNPCStation(ctx, structureID)
	}

	return s.getPlayerStructure(ctx, structureID, token)
}

// getNPCStation retrieves NPC station information
func (s *StructureService) getNPCStation(ctx context.Context, stationID int64) (*models.Structure, error) {
	// Get station info from SDE
	station, err := s.sdeService.GetStaStation(int(stationID))
	if err != nil {
		return nil, fmt.Errorf("station not found in SDE: %w", err)
	}

	// Get type name if possible
	typeName := ""
	if typeInfo, err := s.sdeService.GetType(fmt.Sprintf("%d", station.StationTypeID)); err == nil {
		if name, ok := typeInfo.Name["en"]; ok {
			typeName = name
		}
	}

	// Create structure model
	structure := &models.Structure{
		StructureID:     stationID,
		Name:            station.StationName,
		OwnerID:         int32(station.CorporationID),
		SolarSystemID:   int32(station.SolarSystemID),
		ConstellationID: int32(station.ConstellationID),
		RegionID:        int32(station.RegionID),
		TypeID:          int32(station.StationTypeID),
		TypeName:        typeName,
		IsNPCStation:    true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Try to get solar system name
	if solarSystem, err := s.sdeService.GetSolarSystem(station.SolarSystemID); err == nil {
		// Solar system doesn't have a direct name field, but we can use the station name to infer it
		// Station names typically include the system name
		structure.SolarSystemID = int32(solarSystem.SolarSystemID)
	}

	// Try to get constellation name
	if constellation, err := s.sdeService.GetConstellation(station.ConstellationID); err == nil {
		// Constellation also uses NameID, we'd need a name lookup service
		_ = constellation // Suppress unused variable warning
	}

	// Try to get region name
	if region, err := s.sdeService.GetRegion(station.RegionID); err == nil {
		// Region also uses NameID, we'd need a name lookup service
		_ = region // Suppress unused variable warning
	}

	// Save to database
	if err := s.saveStructure(ctx, structure); err != nil {
		return nil, err
	}

	// Cache the result
	s.cacheStructure(ctx, structure)

	return structure, nil
}

// getPlayerStructure retrieves player structure information from ESI
func (s *StructureService) getPlayerStructure(ctx context.Context, structureID int64, token string) (*models.Structure, error) {
	// Check database first
	structure, err := s.getStructureFromDB(ctx, structureID)
	if err == nil && structure != nil {
		// Check if data is fresh (less than 1 hour old)
		if time.Since(structure.UpdatedAt) < time.Hour {
			return structure, nil
		}
	}

	// Fetch from ESI
	esiStructure, err := s.eveGateway.Structures.GetStructure(ctx, structureID, token)
	if err != nil {
		// Check if it's an authentication error (401) - fail immediately
		if strings.Contains(err.Error(), "status 401") {
			return nil, fmt.Errorf("authentication failed for structure %d: %w", structureID, err)
		}

		// Check if it's a forbidden error (403) - character doesn't have access
		if strings.Contains(err.Error(), "status 403") {
			// Log that we're skipping this structure due to access restrictions
			slog.DebugContext(ctx, "Structure access denied",
				"structure_id", structureID,
				"reason", "Character lacks docking rights (403)")

			// For forbidden errors, return cached data if available, otherwise a specific error
			if structure != nil {
				return structure, nil
			}
			return nil, fmt.Errorf("access denied to structure %d: character lacks docking rights", structureID)
		}

		// For other errors, if we have cached data, return it
		if structure != nil {
			return structure, nil
		}
		return nil, fmt.Errorf("failed to fetch structure from ESI: %w", err)
	}

	// Parse ESI structure response from map[string]any
	// The adapter returns int32 values, but we need to handle both int32 and float64 due to JSON unmarshaling
	name, _ := esiStructure["name"].(string)

	// Handle owner_id - try int32 first (from adapter), then float64 (from JSON)
	var ownerID int32
	if val, ok := esiStructure["owner_id"].(int32); ok {
		ownerID = val
	} else if val, ok := esiStructure["owner_id"].(float64); ok {
		ownerID = int32(val)
	}

	// Handle solar_system_id
	var solarSystemID int32
	if val, ok := esiStructure["solar_system_id"].(int32); ok {
		solarSystemID = val
	} else if val, ok := esiStructure["solar_system_id"].(float64); ok {
		solarSystemID = int32(val)
	}

	// Handle type_id
	var typeID int32
	if val, ok := esiStructure["type_id"].(int32); ok {
		typeID = val
	} else if val, ok := esiStructure["type_id"].(float64); ok {
		typeID = int32(val)
	}

	// Get type name from SDE
	typeName := ""
	if typeInfo, err := s.sdeService.GetType(fmt.Sprintf("%d", typeID)); err == nil {
		if name, ok := typeInfo.Name["en"]; ok {
			typeName = name
		}
	}

	// Parse position if present
	var position *models.Position
	if positionData, ok := esiStructure["position"].(map[string]interface{}); ok {
		position = &models.Position{
			X: getFloat64FromInterface(positionData["x"]),
			Y: getFloat64FromInterface(positionData["y"]),
			Z: getFloat64FromInterface(positionData["z"]),
		}
	}

	// Parse optional string slice fields
	var services []string
	if servicesData, ok := esiStructure["services"].([]interface{}); ok {
		for _, service := range servicesData {
			if serviceStr, ok := service.(string); ok {
				services = append(services, serviceStr)
			}
		}
	}

	state, _ := esiStructure["state"].(string)

	// Parse optional time fields
	var fuelExpires *time.Time
	if fuelExpiresStr, ok := esiStructure["fuel_expires"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, fuelExpiresStr); err == nil {
			fuelExpires = &parsed
		}
	}

	var stateTimerStart *time.Time
	if timerStartStr, ok := esiStructure["state_timer_start"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, timerStartStr); err == nil {
			stateTimerStart = &parsed
		}
	}

	var stateTimerEnd *time.Time
	if timerEndStr, ok := esiStructure["state_timer_end"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, timerEndStr); err == nil {
			stateTimerEnd = &parsed
		}
	}

	var unanchorsAt *time.Time
	if unanchorsStr, ok := esiStructure["unanchors_at"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, unanchorsStr); err == nil {
			unanchorsAt = &parsed
		}
	}

	// Create or update structure model
	if structure == nil {
		structure = &models.Structure{}
	}

	structure.StructureID = structureID
	structure.Name = name
	structure.OwnerID = ownerID
	structure.Position = position
	structure.SolarSystemID = solarSystemID
	structure.TypeID = typeID
	structure.TypeName = typeName
	structure.IsNPCStation = structureID < npcStationThreshold
	structure.Services = services
	structure.State = state
	structure.FuelExpires = fuelExpires
	structure.StateTimerStart = stateTimerStart
	structure.StateTimerEnd = stateTimerEnd
	structure.UnanchorsAt = unanchorsAt
	structure.UpdatedAt = time.Now()

	// Try to get solar system info to determine constellation and region
	if solarSystem, err := s.sdeService.GetSolarSystem(int(solarSystemID)); err == nil {
		// The SDE solar system doesn't have constellation/region references directly
		// We would need to enhance the SDE service or maintain a lookup table
		_ = solarSystem // Suppress unused variable warning
	}

	// Save to database
	if err := s.saveStructure(ctx, structure); err != nil {
		return nil, err
	}

	// TODO: Update access record - requires character ID extraction from token

	// Cache the result
	s.cacheStructure(ctx, structure)

	return structure, nil
}

// getStructureFromDB retrieves structure from database
func (s *StructureService) getStructureFromDB(ctx context.Context, structureID int64) (*models.Structure, error) {
	var structure models.Structure
	err := s.db.Collection(models.StructuresCollection).FindOne(ctx, bson.M{"structure_id": structureID}).Decode(&structure)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &structure, nil
}

// saveStructure saves or updates a structure in the database
func (s *StructureService) saveStructure(ctx context.Context, structure *models.Structure) error {
	if structure.ID.IsZero() {
		structure.ID = primitive.NewObjectID()
		structure.CreatedAt = time.Now()
	}
	structure.UpdatedAt = time.Now()

	opts := options.Replace().SetUpsert(true)
	_, err := s.db.Collection(models.StructuresCollection).ReplaceOne(
		ctx,
		bson.M{"structure_id": structure.StructureID},
		structure,
		opts,
	)
	return err
}

// cacheStructure caches structure data in Redis
func (s *StructureService) cacheStructure(ctx context.Context, structure *models.Structure) {
	cacheKey := fmt.Sprintf("%s%d", structureCachePrefix, structure.StructureID)
	data, err := json.Marshal(structure)
	if err == nil {
		s.redis.Set(ctx, cacheKey, data, structureCacheTTL)
	}
}

// updateStructureAccess updates structure access record
func (s *StructureService) updateStructureAccess(ctx context.Context, structureID int64, characterID int32, hasAccess bool) {
	access := &models.StructureAccess{
		StructureID: structureID,
		CharacterID: characterID,
		HasAccess:   hasAccess,
		LastChecked: time.Now(),
		UpdatedAt:   time.Now(),
	}

	opts := options.Replace().SetUpsert(true)
	s.db.Collection(models.StructureAccessCollection).ReplaceOne(
		ctx,
		bson.M{
			"structure_id": structureID,
			"character_id": characterID,
		},
		access,
		opts,
	)
}

// GetStructuresBySystem retrieves all structures in a solar system
func (s *StructureService) GetStructuresBySystem(ctx context.Context, solarSystemID int32) ([]*models.Structure, error) {
	cursor, err := s.db.Collection(models.StructuresCollection).Find(ctx, bson.M{"solar_system_id": solarSystemID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var structures []*models.Structure
	if err := cursor.All(ctx, &structures); err != nil {
		return nil, err
	}

	return structures, nil
}

// GetStructuresByOwner retrieves all structures owned by a corporation
func (s *StructureService) GetStructuresByOwner(ctx context.Context, ownerID int32) ([]*models.Structure, error) {
	cursor, err := s.db.Collection(models.StructuresCollection).Find(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var structures []*models.Structure
	if err := cursor.All(ctx, &structures); err != nil {
		return nil, err
	}

	return structures, nil
}

// BulkRefreshStructures refreshes multiple structures
func (s *StructureService) BulkRefreshStructures(ctx context.Context, structureIDs []int64, token string) ([]int64, []int64, error) {
	var refreshed []int64
	var failed []int64

	for _, structureID := range structureIDs {
		_, err := s.GetStructure(ctx, structureID, token)
		if err != nil {
			// If authentication fails, stop processing immediately
			if strings.Contains(err.Error(), "authentication failed") {
				return refreshed, failed, fmt.Errorf("authentication failed during bulk refresh: %w", err)
			}
			failed = append(failed, structureID)
		} else {
			refreshed = append(refreshed, structureID)
		}
	}

	return refreshed, failed, nil
}

// GetCharacterAccessibleStructures returns structures a character has access to
func (s *StructureService) GetCharacterAccessibleStructures(ctx context.Context, characterID int32) ([]*models.Structure, error) {
	// First get all structure IDs the character has access to
	cursor, err := s.db.Collection(models.StructureAccessCollection).Find(ctx, bson.M{
		"character_id": characterID,
		"has_access":   true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accessRecords []models.StructureAccess
	if err := cursor.All(ctx, &accessRecords); err != nil {
		return nil, err
	}

	if len(accessRecords) == 0 {
		return []*models.Structure{}, nil
	}

	// Get structure IDs
	structureIDs := make([]int64, len(accessRecords))
	for i, record := range accessRecords {
		structureIDs[i] = record.StructureID
	}

	// Fetch structures
	structureCursor, err := s.db.Collection(models.StructuresCollection).Find(ctx, bson.M{
		"structure_id": bson.M{"$in": structureIDs},
	})
	if err != nil {
		return nil, err
	}
	defer structureCursor.Close(ctx)

	var structures []*models.Structure
	if err := structureCursor.All(ctx, &structures); err != nil {
		return nil, err
	}

	return structures, nil
}
