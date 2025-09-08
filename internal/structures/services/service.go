package services

import (
	"context"
	"encoding/json"
	"fmt"
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
func (s *StructureService) GetStructure(ctx context.Context, structureID int64, characterID int32) (*models.Structure, error) {
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

	return s.getPlayerStructure(ctx, structureID, characterID)
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
func (s *StructureService) getPlayerStructure(ctx context.Context, structureID int64, characterID int32) (*models.Structure, error) {
	// Check database first
	structure, err := s.getStructureFromDB(ctx, structureID)
	if err == nil && structure != nil {
		// Check if data is fresh (less than 1 hour old)
		if time.Since(structure.UpdatedAt) < time.Hour {
			return structure, nil
		}
	}

	// Fetch from ESI
	esiStructure, err := s.eveGateway.GetStructure(ctx, structureID, characterID)
	if err != nil {
		// If we have cached data and ESI fails, return cached
		if structure != nil {
			return structure, nil
		}
		return nil, fmt.Errorf("failed to fetch structure from ESI: %w", err)
	}

	// Get type name from SDE
	typeName := ""
	if typeInfo, err := s.sdeService.GetType(fmt.Sprintf("%d", esiStructure.TypeID)); err == nil {
		if name, ok := typeInfo.Name["en"]; ok {
			typeName = name
		}
	}

	// Create or update structure model
	if structure == nil {
		structure = &models.Structure{}
	}

	structure.StructureID = structureID
	structure.CharacterID = characterID
	structure.Name = esiStructure.Name
	structure.OwnerID = esiStructure.OwnerID
	structure.Position = &models.Position{
		X: esiStructure.Position.X,
		Y: esiStructure.Position.Y,
		Z: esiStructure.Position.Z,
	}
	structure.SolarSystemID = esiStructure.SolarSystemID
	structure.TypeID = esiStructure.TypeID
	structure.TypeName = typeName
	structure.IsNPCStation = false
	structure.Services = esiStructure.Services
	structure.State = esiStructure.State
	structure.FuelExpires = esiStructure.FuelExpires
	structure.StateTimerStart = esiStructure.StateTimerStart
	structure.StateTimerEnd = esiStructure.StateTimerEnd
	structure.UnanchorsAt = esiStructure.UnanchorsAt
	structure.UpdatedAt = time.Now()

	// Try to get solar system info to determine constellation and region
	if solarSystem, err := s.sdeService.GetSolarSystem(int(esiStructure.SolarSystemID)); err == nil {
		// The SDE solar system doesn't have constellation/region references directly
		// We would need to enhance the SDE service or maintain a lookup table
		_ = solarSystem // Suppress unused variable warning
	}

	// Save to database
	if err := s.saveStructure(ctx, structure); err != nil {
		return nil, err
	}

	// Update access record
	s.updateStructureAccess(ctx, structureID, characterID, true)

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
func (s *StructureService) BulkRefreshStructures(ctx context.Context, structureIDs []int64, characterID int32) ([]int64, []int64) {
	var refreshed []int64
	var failed []int64

	for _, structureID := range structureIDs {
		_, err := s.GetStructure(ctx, structureID, characterID)
		if err != nil {
			failed = append(failed, structureID)
		} else {
			refreshed = append(refreshed, structureID)
		}
	}

	return refreshed, failed
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
