package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	groupsDTO "go-falcon/internal/groups/dto"
	"go-falcon/internal/mapservice/dto"
	"go-falcon/internal/mapservice/models"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GroupsService interface for groups service dependency
type GroupsService interface {
	GetUserGroups(ctx context.Context, input *groupsDTO.GetUserGroupsInput) (*groupsDTO.UserGroupsOutput, error)
	GetCharacterGroups(ctx context.Context, input *groupsDTO.GetCharacterGroupsInput) (*groupsDTO.CharacterGroupsOutput, error)
}

type MapService struct {
	db              *mongo.Database
	redis           *redis.Client
	sde             *sde.Service
	SDEService      *sde.Service     // Public accessor for routes
	wormholeService *WormholeService // Wormhole operations
	groupsService   GroupsService    // Optional groups service for access control
}

func NewMapService(db *mongo.Database, redis *redis.Client, sdeService *sde.Service) *MapService {
	wormholeService := NewWormholeService(db, redis, sdeService)
	return &MapService{
		db:              db,
		redis:           redis,
		sde:             sdeService,
		SDEService:      sdeService, // Expose for routes
		wormholeService: wormholeService,
		groupsService:   nil, // Set later via SetGroupsService
	}
}

// SetGroupsService sets the groups service for access control
func (s *MapService) SetGroupsService(groupsService GroupsService) {
	s.groupsService = groupsService
}

// GetGroupsService returns the groups service
func (s *MapService) GetGroupsService() GroupsService {
	return s.groupsService
}

// GetUserGroupIDs retrieves the group IDs for a user
func (s *MapService) GetUserGroupIDs(ctx context.Context, userID string) ([]primitive.ObjectID, error) {
	if s.groupsService == nil {
		// Return empty slice if groups service is not available
		return []primitive.ObjectID{}, nil
	}

	// Create input for groups service
	input := &groupsDTO.GetUserGroupsInput{
		UserID: userID,
	}

	// Call groups service to get user groups
	output, err := s.groupsService.GetUserGroups(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Convert group ID strings to ObjectIDs
	var groupIDs []primitive.ObjectID
	for _, group := range output.Body.Groups {
		objectID, err := primitive.ObjectIDFromHex(group.ID)
		if err != nil {
			// Log the error but continue with other groups
			continue
		}
		groupIDs = append(groupIDs, objectID)
	}

	return groupIDs, nil
}

// GetUserDefaultGroupID retrieves a default group ID for a user (for creating new items)
func (s *MapService) GetUserDefaultGroupID(ctx context.Context, userID string) (*primitive.ObjectID, error) {
	if s.groupsService == nil {
		// Return nil if groups service is not available
		return nil, nil
	}

	// Get user's groups to find a suitable default
	groupIDs, err := s.GetUserGroupIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups for default: %w", err)
	}

	// For now, return the first group if available
	// In the future, this could be enhanced to prioritize corporation/alliance groups
	if len(groupIDs) > 0 {
		return &groupIDs[0], nil
	}

	// No groups available - return nil
	return nil, nil
}

// Signature Management

func (s *MapService) CreateSignature(ctx context.Context, userID primitive.ObjectID, userName string, groupID *primitive.ObjectID, input dto.CreateSignatureInput) (*models.MapSignature, error) {
	// Validate system exists
	if _, err := s.sde.GetSolarSystem(int(input.SystemID)); err != nil {
		return nil, fmt.Errorf("invalid system ID: %w", err)
	}

	signature := &models.MapSignature{
		SystemID:      input.SystemID,
		SignatureID:   input.SignatureID,
		Type:          input.Type,
		Name:          input.Name,
		Description:   input.Description,
		Strength:      input.Strength,
		CreatedBy:     userID,
		CreatedByName: userName,
		SharingLevel:  input.SharingLevel,
		GroupID:       groupID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set expiration if specified
	if input.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(input.ExpiresIn) * time.Hour)
		signature.ExpiresAt = &expiresAt
	}

	// Check for existing signature with same ID in system
	filter := bson.M{
		"system_id":    input.SystemID,
		"signature_id": input.SignatureID,
		"$or": []bson.M{
			{"expires_at": bson.M{"$gt": time.Now()}},
			{"expires_at": nil},
		},
	}

	var existing models.MapSignature
	err := s.db.Collection("map_signatures").FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		// Update existing signature instead of creating duplicate
		update := bson.M{
			"$set": bson.M{
				"type":            signature.Type,
				"name":            signature.Name,
				"description":     signature.Description,
				"strength":        signature.Strength,
				"updated_by":      userID,
				"updated_by_name": userName,
				"updated_at":      time.Now(),
			},
		}

		if signature.ExpiresAt != nil {
			update["$set"].(bson.M)["expires_at"] = signature.ExpiresAt
		}

		err = s.db.Collection("map_signatures").FindOneAndUpdate(
			ctx,
			bson.M{"_id": existing.ID},
			update,
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(signature)

		if err != nil {
			return nil, fmt.Errorf("failed to update signature: %w", err)
		}
		return signature, nil
	}

	// Create new signature
	result, err := s.db.Collection("map_signatures").InsertOne(ctx, signature)
	if err != nil {
		return nil, fmt.Errorf("failed to create signature: %w", err)
	}

	signature.ID = result.InsertedID.(primitive.ObjectID)
	return signature, nil
}

func (s *MapService) GetSignatures(ctx context.Context, userID primitive.ObjectID, groupIDs []primitive.ObjectID, input dto.GetSignaturesInput) ([]models.MapSignature, error) {
	filter := bson.M{}

	// Filter by system if specified
	if input.SystemID > 0 {
		filter["system_id"] = input.SystemID
	}

	// Filter by type if specified
	if input.Type != "" {
		filter["type"] = input.Type
	}

	// Handle expiration filter
	if !input.IncludeExpired {
		filter["$or"] = []bson.M{
			{"expires_at": bson.M{"$gt": time.Now()}},
			{"expires_at": nil},
		}
	}

	// Handle sharing level filter
	if input.SharingLevel != "all" && input.SharingLevel != "" {
		filter["sharing_level"] = input.SharingLevel
	}

	// Apply visibility rules
	visibilityFilter := bson.M{
		"$or": []bson.M{
			{"created_by": userID},
			{"sharing_level": "alliance"},
			{
				"$and": []bson.M{
					{"sharing_level": "corporation"},
					{"group_id": bson.M{"$in": groupIDs}},
				},
			},
		},
	}

	// Combine filters
	if len(filter) > 0 {
		filter = bson.M{"$and": []bson.M{filter, visibilityFilter}}
	} else {
		filter = visibilityFilter
	}

	cursor, err := s.db.Collection("map_signatures").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get signatures: %w", err)
	}
	defer cursor.Close(ctx)

	var signatures []models.MapSignature
	if err := cursor.All(ctx, &signatures); err != nil {
		return nil, fmt.Errorf("failed to decode signatures: %w", err)
	}

	return signatures, nil
}

func (s *MapService) UpdateSignature(ctx context.Context, signatureID primitive.ObjectID, userID primitive.ObjectID, userName string, input dto.UpdateSignatureInput) error {
	update := bson.M{
		"$set": bson.M{
			"updated_by":      userID,
			"updated_by_name": userName,
			"updated_at":      time.Now(),
		},
	}

	// Add fields to update if provided
	if input.Type != nil {
		update["$set"].(bson.M)["type"] = *input.Type
	}
	if input.Name != nil {
		update["$set"].(bson.M)["name"] = *input.Name
	}
	if input.Description != nil {
		update["$set"].(bson.M)["description"] = *input.Description
	}
	if input.Strength != nil {
		update["$set"].(bson.M)["strength"] = *input.Strength
	}

	result, err := s.db.Collection("map_signatures").UpdateOne(
		ctx,
		bson.M{"_id": signatureID},
		update,
	)

	if err != nil {
		return fmt.Errorf("failed to update signature: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("signature not found or no changes made")
	}

	return nil
}

func (s *MapService) DeleteSignature(ctx context.Context, signatureID primitive.ObjectID, userID primitive.ObjectID) error {
	// Only allow deletion by creator or based on permissions
	filter := bson.M{
		"_id": signatureID,
		"$or": []bson.M{
			{"created_by": userID},
			// Add admin permission check here if needed
		},
	}

	result, err := s.db.Collection("map_signatures").DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete signature: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("signature not found or insufficient permissions")
	}

	return nil
}

// Batch Operations

func (s *MapService) BatchUpdateSignatures(ctx context.Context, userID primitive.ObjectID, userName string, groupID *primitive.ObjectID, input dto.BatchSignatureInput) (*dto.BatchSignatureOutput, error) {
	output := &dto.BatchSignatureOutput{
		Created: []dto.SignatureOutput{},
		Updated: []dto.SignatureOutput{},
		Deleted: []string{},
		Errors:  []dto.BatchError{},
	}

	// Get existing signatures for the system
	existingFilter := bson.M{
		"system_id": input.SystemID,
		"$or": []bson.M{
			{"expires_at": bson.M{"$gt": time.Now()}},
			{"expires_at": nil},
		},
	}

	cursor, err := s.db.Collection("map_signatures").Find(ctx, existingFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing signatures: %w", err)
	}
	defer cursor.Close(ctx)

	existingMap := make(map[string]*models.MapSignature)
	for cursor.Next(ctx) {
		var sig models.MapSignature
		if err := cursor.Decode(&sig); err != nil {
			continue
		}
		existingMap[sig.SignatureID] = &sig
	}

	// Process each signature in the batch
	processedIDs := make(map[string]bool)
	for _, sigInput := range input.Signatures {
		processedIDs[sigInput.SignatureID] = true

		if existing, exists := existingMap[sigInput.SignatureID]; exists {
			// Update existing signature
			updateInput := dto.UpdateSignatureInput{
				Type:        &sigInput.Type,
				Name:        &sigInput.Name,
				Description: &sigInput.Description,
				Strength:    &sigInput.Strength,
			}

			if err := s.UpdateSignature(ctx, existing.ID, userID, userName, updateInput); err != nil {
				output.Errors = append(output.Errors, dto.BatchError{
					SignatureID: sigInput.SignatureID,
					Error:       err.Error(),
				})
			} else {
				// Get updated signature for output
				var updated models.MapSignature
				s.db.Collection("map_signatures").FindOne(ctx, bson.M{"_id": existing.ID}).Decode(&updated)
				system, _ := s.sde.GetSolarSystem(int(updated.SystemID))
				systemName := GetSystemName(s.sde, system)
				output.Updated = append(output.Updated, dto.SignatureToOutput(&updated, systemName))
			}
		} else {
			// Create new signature
			created, err := s.CreateSignature(ctx, userID, userName, groupID, sigInput)
			if err != nil {
				output.Errors = append(output.Errors, dto.BatchError{
					SignatureID: sigInput.SignatureID,
					Error:       err.Error(),
				})
			} else {
				system, _ := s.sde.GetSolarSystem(int(created.SystemID))
				systemName := GetSystemName(s.sde, system)
				output.Created = append(output.Created, dto.SignatureToOutput(created, systemName))
			}
		}
	}

	// Delete old signatures if requested
	if input.DeleteOld {
		for sigID, existing := range existingMap {
			if !processedIDs[sigID] {
				if err := s.DeleteSignature(ctx, existing.ID, userID); err != nil {
					output.Errors = append(output.Errors, dto.BatchError{
						SignatureID: sigID,
						Error:       fmt.Sprintf("failed to delete: %v", err),
					})
				} else {
					output.Deleted = append(output.Deleted, existing.ID.Hex())
				}
			}
		}
	}

	return output, nil
}

// System Activity

func (s *MapService) GetSystemActivity(ctx context.Context, systemIDs []int32) ([]models.MapSystemActivity, error) {
	// Check cache first
	activities := make([]models.MapSystemActivity, 0, len(systemIDs))

	// TODO: Implement Redis caching
	// For now, get from MongoDB

	filter := bson.M{
		"system_id":  bson.M{"$in": systemIDs},
		"updated_at": bson.M{"$gte": time.Now().Add(-2 * time.Hour)}, // Only recent data
	}

	cursor, err := s.db.Collection("map_system_activity").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get system activity: %w", err)
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &activities); err != nil {
		return nil, fmt.Errorf("failed to decode activities: %w", err)
	}

	return activities, nil
}

// Region Data

func (s *MapService) GetRegionSystems(ctx context.Context, regionID int32) (*dto.MapRegionOutput, error) {
	// Check Redis cache first
	cacheKey := fmt.Sprintf("map:region:%d", regionID)
	if s.redis != nil {
		cachedData, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil && cachedData != "" {
			// Found in cache, unmarshal and return
			var result dto.MapRegionOutput
			if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
				return &result, nil
			}
		}
		// If not in cache or error, continue to fetch from MongoDB
	}

	// Create result array for map elements (nodes and edges)
	mapElements := make([]dto.MapElement, 0)

	// Fetch from MongoDB collections

	// Get all systems in the region from map-nodes collection
	cursor, err := s.db.Collection("map-nodes").Find(ctx, bson.M{"region_id": regionID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch systems from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	systemIDs := make(map[int32]bool)

	// Process all systems as nodes
	for cursor.Next(ctx) {
		var node struct {
			SystemID        int32   `bson:"system_id"`
			SystemName      string  `bson:"system_name"`
			RegionID        int32   `bson:"region_id"`
			RegionName      string  `bson:"region_name"`
			ConstellationID int32   `bson:"constellation_id"`
			SecStatus       float32 `bson:"secstatus"`
			Planets         []struct {
				PlanetID int32   `bson:"planet_id"`
				Moons    []int32 `bson:"moons"`
			} `bson:"planets"`
			Stations []int32 `bson:"stations"`
		}

		if err := cursor.Decode(&node); err != nil {
			continue
		}

		// Get position from map-positions collection if available
		var position *dto.MapPosition
		var posData struct {
			PositionX float64 `bson:"positionX"`
			PositionY float64 `bson:"positionY"`
		}
		if err := s.db.Collection("map-positions").FindOne(ctx, bson.M{"system_id": node.SystemID}).Decode(&posData); err == nil {
			position = &dto.MapPosition{
				X: posData.PositionX,
				Y: posData.PositionY,
			}
		} else {
			// Fallback to 0,0 if no position data
			position = &dto.MapPosition{X: 0, Y: 0}
		}

		// Build planets data
		planets := make([]dto.PlanetData, 0)
		temperate := 0
		for _, planet := range node.Planets {
			// We don't have typeID in the current data, so using 0
			// In production you might want to fetch this from SDE or another source
			planets = append(planets, dto.PlanetData{
				PlanetID: planet.PlanetID,
				TypeID:   0, // TODO: Get actual type ID from SDE or database
			})
		}

		// Create node data
		nodeData := dto.MapNodeData{
			ID:             node.SystemID,
			Name:           node.SystemName,
			Label:          node.SystemName,
			RegionID:       node.RegionID,
			RegionName:     node.RegionName,
			SecStatus:      node.SecStatus,
			Regional:       false, // TODO: Determine if system is regional
			AllianceID:     nil,
			FactionID:      nil,
			SunPower:       0,
			EquinoxPlanets: []interface{}{},
			Planets:        planets,
			Temperate:      temperate,
		}

		// Add node element
		mapElements = append(mapElements, dto.MapElement{
			Group:    "nodes",
			Data:     nodeData,
			Classes:  "system",
			Position: position,
		})

		systemIDs[node.SystemID] = true
	}

	// Get connections from map-edges collection
	edgeCursor, err := s.db.Collection("map-edges").Find(ctx, bson.M{"region_id": regionID})
	if err == nil {
		defer edgeCursor.Close(ctx)

		for edgeCursor.Next(ctx) {
			var edge struct {
				ID     int32 `bson:"id"`
				Source int32 `bson:"source"`
				Target int32 `bson:"target"`
			}

			if err := edgeCursor.Decode(&edge); err != nil {
				continue
			}

			// Only include connections where both systems are in this region
			if systemIDs[edge.Source] && systemIDs[edge.Target] {
				// Create edge data
				edgeData := dto.MapEdgeData{
					ID:     edge.ID,
					Source: edge.Source,
					Target: edge.Target,
				}

				// Add edge element
				mapElements = append(mapElements, dto.MapElement{
					Group:   "edges",
					Data:    edgeData,
					Classes: "systemEdge",
				})
			}
		}
	}

	// Create the response with the elements
	result := &dto.MapRegionOutput{
		Elements: mapElements,
	}

	// Cache the result in Redis for future use (cache for 24 hours since region data rarely changes)
	if s.redis != nil {
		if data, err := json.Marshal(result); err == nil {
			s.redis.Set(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return result, nil
}

// Search Systems

func (s *MapService) SearchSystems(ctx context.Context, query string, limit int) ([]dto.SearchSystemOutput, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("search:systems:%s:%d", strings.ToLower(query), limit)
	if cachedResults, err := s.getSearchFromCache(ctx, cacheKey); err == nil {
		return cachedResults, nil
	}

	// Use SDE service for searching
	allSystems, err := s.sde.GetAllSolarSystems()
	if err != nil {
		return nil, fmt.Errorf("failed to get solar systems: %w", err)
	}

	results := make([]dto.SearchSystemOutput, 0)
	count := 0

	// First pass: exact matches
	for _, system := range allSystems {
		if count >= limit {
			break
		}

		systemName := GetSystemName(s.sde, system)
		if systemName == query {
			constellationID := getConstellationIDForSystem(s.sde, system)
			constellation, _ := s.sde.GetConstellation(int(constellationID))
			var regionName, constellationName string

			if constellation != nil {
				constellationName = GetConstellationName(s.sde, constellation)
				regionID := getRegionIDForConstellation(s.sde, constellation)
				if region, err := s.sde.GetRegion(int(regionID)); err == nil {
					regionName = GetRegionName(s.sde, region)
				}
			}

			results = append(results, dto.SearchSystemOutput{
				SystemID:          int32(system.SolarSystemID),
				SystemName:        GetSystemName(s.sde, system),
				RegionName:        regionName,
				ConstellationName: constellationName,
				Security:          float32(system.Security),
				MatchType:         "exact",
			})
			count++
		}
	}

	// Second pass: starts with
	for _, system := range allSystems {
		if count >= limit {
			break
		}

		systemName := GetSystemName(s.sde, system)
		if len(systemName) >= len(query) && systemName[:len(query)] == query && systemName != query {
			constellationID := getConstellationIDForSystem(s.sde, system)
			constellation, _ := s.sde.GetConstellation(int(constellationID))
			var regionName, constellationName string

			if constellation != nil {
				constellationName = GetConstellationName(s.sde, constellation)
				regionID := getRegionIDForConstellation(s.sde, constellation)
				if region, err := s.sde.GetRegion(int(regionID)); err == nil {
					regionName = GetRegionName(s.sde, region)
				}
			}

			results = append(results, dto.SearchSystemOutput{
				SystemID:          int32(system.SolarSystemID),
				SystemName:        GetSystemName(s.sde, system),
				RegionName:        regionName,
				ConstellationName: constellationName,
				Security:          float32(system.Security),
				MatchType:         "starts_with",
			})
			count++
		}
	}

	// Third pass: contains
	for _, system := range allSystems {
		if count >= limit {
			break
		}

		// Skip if already added
		alreadyAdded := false
		for _, r := range results {
			if r.SystemID == int32(system.SolarSystemID) {
				alreadyAdded = true
				break
			}
		}

		systemName := GetSystemName(s.sde, system)
		if !alreadyAdded && len(query) > 0 {
			// Case-insensitive contains check
			if containsIgnoreCase(systemName, query) {
				constellationID := getConstellationIDForSystem(s.sde, system)
				constellation, _ := s.sde.GetConstellation(int(constellationID))
				var regionName, constellationName string

				if constellation != nil {
					constellationName = GetConstellationName(s.sde, constellation)
					regionID := getRegionIDForConstellation(s.sde, constellation)
					if region, err := s.sde.GetRegion(int(regionID)); err == nil {
						regionName = GetRegionName(s.sde, region)
					}
				}

				results = append(results, dto.SearchSystemOutput{
					SystemID:          int32(system.SolarSystemID),
					SystemName:        systemName,
					RegionName:        regionName,
					ConstellationName: constellationName,
					Security:          float32(system.Security),
					MatchType:         "contains",
				})
				count++
			}
		}
	}

	// Cache the results for 1 hour
	s.cacheSearchResults(ctx, cacheKey, results)

	return results, nil
}

// Cache helper methods for search

func (s *MapService) getSearchFromCache(ctx context.Context, key string) ([]dto.SearchSystemOutput, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var results []dto.SearchSystemOutput
	if err := json.Unmarshal([]byte(data), &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *MapService) cacheSearchResults(ctx context.Context, key string, results []dto.SearchSystemOutput) {
	if s.redis == nil {
		return
	}

	data, err := json.Marshal(results)
	if err != nil {
		return
	}

	// Cache for 1 hour since system data doesn't change often
	s.redis.Set(ctx, key, data, 1*time.Hour)
}

// Helper function for case-insensitive contains
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// GetSDE returns the SDE service for external access
func (s *MapService) GetSDE() *sde.Service {
	return s.sde
}

// GetSignatureByID retrieves a specific signature by ID with proper access control
func (s *MapService) GetSignatureByID(ctx context.Context, signatureID primitive.ObjectID, userID primitive.ObjectID, groupIDs []primitive.ObjectID) (*models.MapSignature, error) {
	// Build visibility filter
	visibilityFilter := bson.M{
		"$or": []bson.M{
			{"created_by": userID},
			{"sharing_level": "alliance"},
			{
				"$and": []bson.M{
					{"sharing_level": "corporation"},
					{"group_id": bson.M{"$in": groupIDs}},
				},
			},
		},
	}

	filter := bson.M{
		"_id": signatureID,
		"$and": []bson.M{
			visibilityFilter,
		},
	}

	var signature models.MapSignature
	err := s.db.Collection("map_signatures").FindOne(ctx, filter).Decode(&signature)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("signature not found")
		}
		return nil, err
	}

	return &signature, nil
}

// UpdateSignature with the correct signature for the routes (returns updated signature)
func (s *MapService) UpdateSignatureForRoute(ctx context.Context, signatureID primitive.ObjectID, userID primitive.ObjectID, input dto.UpdateSignatureInput) (*models.MapSignature, error) {
	// Check if user has permission to update this signature
	filter := bson.M{
		"_id":        signatureID,
		"created_by": userID, // Only allow updating own signatures for now
	}

	update := bson.M{
		"$set": bson.M{
			"updated_by": userID,
			"updated_at": time.Now(),
		},
	}

	// Update fields if provided
	if input.Type != nil {
		update["$set"].(bson.M)["type"] = *input.Type
	}
	if input.Name != nil {
		update["$set"].(bson.M)["name"] = *input.Name
	}
	if input.Description != nil {
		update["$set"].(bson.M)["description"] = *input.Description
	}
	if input.Strength != nil {
		update["$set"].(bson.M)["strength"] = *input.Strength
	}

	result, err := s.db.Collection("map_signatures").UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("signature not found or permission denied")
	}

	// Retrieve and return the updated signature
	var signature models.MapSignature
	err = s.db.Collection("map_signatures").FindOne(ctx, bson.M{"_id": signatureID}).Decode(&signature)
	if err != nil {
		return nil, err
	}

	return &signature, nil
}

// DeleteSignatureForRoute with the correct signature for the routes
func (s *MapService) DeleteSignatureForRoute(ctx context.Context, signatureID primitive.ObjectID, userID primitive.ObjectID) error {
	// Check if user has permission to delete this signature
	filter := bson.M{
		"_id":        signatureID,
		"created_by": userID, // Only allow deleting own signatures for now
	}

	result, err := s.db.Collection("map_signatures").DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("signature not found or permission denied")
	}

	return nil
}

// Module Status

func (s *MapService) GetModuleStatus(ctx context.Context) (*dto.MapStatusOutput, error) {
	statusData := dto.MapStatusResponse{
		Module:  "map",
		Status:  "healthy",
		Message: "Map module operational",
	}

	// Get statistics
	sigCount, _ := s.db.Collection("map_signatures").CountDocuments(ctx, bson.M{})
	whCount, _ := s.db.Collection("map_wormholes").CountDocuments(ctx, bson.M{})
	noteCount, _ := s.db.Collection("map_notes").CountDocuments(ctx, bson.M{})
	routeCount, _ := s.db.Collection("map_routes").CountDocuments(ctx, bson.M{})

	statusData.Stats.Signatures = int(sigCount)
	statusData.Stats.Wormholes = int(whCount)
	statusData.Stats.Notes = int(noteCount)
	statusData.Stats.CachedRoutes = int(routeCount)

	status := &dto.MapStatusOutput{
		Body: statusData,
	}

	// Check database connectivity
	if err := s.db.Client().Ping(ctx, nil); err != nil {
		status.Body.Status = "unhealthy"
		status.Body.Message = fmt.Sprintf("Database connection failed: %v", err)
	}

	// Check Redis connectivity
	if err := s.redis.Ping(ctx).Err(); err != nil {
		status.Body.Status = "unhealthy"
		status.Body.Message = fmt.Sprintf("Redis connection failed: %v", err)
	}

	return status, nil
}

// Wormhole Management (delegated to WormholeService)

func (s *MapService) CreateWormhole(ctx context.Context, userID primitive.ObjectID, userName string, groupID *primitive.ObjectID, input dto.CreateWormholeInput) (*models.MapWormhole, error) {
	return s.wormholeService.CreateWormhole(ctx, userID, userName, groupID, input)
}

func (s *MapService) GetWormholes(ctx context.Context, userID primitive.ObjectID, groupIDs []primitive.ObjectID, input dto.GetWormholesInput) ([]models.MapWormhole, error) {
	return s.wormholeService.GetWormholes(ctx, userID, groupIDs, input)
}

func (s *MapService) GetWormholeByID(ctx context.Context, wormholeID primitive.ObjectID, userID primitive.ObjectID, groupIDs []primitive.ObjectID) (*models.MapWormhole, error) {
	// First get all wormholes the user can access
	wormholes, err := s.wormholeService.GetWormholes(ctx, userID, groupIDs, dto.GetWormholesInput{IncludeExpired: true})
	if err != nil {
		return nil, err
	}

	// Find the specific wormhole
	for _, wh := range wormholes {
		if wh.ID == wormholeID {
			return &wh, nil
		}
	}

	return nil, fmt.Errorf("wormhole not found or access denied")
}

func (s *MapService) UpdateWormholeForRoute(ctx context.Context, wormholeID primitive.ObjectID, userID primitive.ObjectID, input dto.UpdateWormholeInput) (*models.MapWormhole, error) {
	// Update the wormhole
	err := s.wormholeService.UpdateWormhole(ctx, wormholeID, userID, "", input)
	if err != nil {
		return nil, err
	}

	// Return the updated wormhole
	var wormhole models.MapWormhole
	err = s.db.Collection("map_wormholes").FindOne(ctx, bson.M{"_id": wormholeID}).Decode(&wormhole)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated wormhole: %w", err)
	}

	return &wormhole, nil
}

func (s *MapService) DeleteWormholeForRoute(ctx context.Context, wormholeID primitive.ObjectID, userID primitive.ObjectID) error {
	return s.wormholeService.DeleteWormhole(ctx, wormholeID, userID)
}

func (s *MapService) GetWormholeStaticInfo(ctx context.Context, whType string) (*models.WormholeStatic, error) {
	return s.wormholeService.GetStaticInfo(ctx, whType)
}

func (s *MapService) BatchUpdateWormholes(ctx context.Context, userID primitive.ObjectID, userName string, groupID *primitive.ObjectID, input dto.BatchWormholeInput) (*dto.BatchWormholeOutput, error) {
	result := &dto.BatchWormholeOutput{
		Created: []dto.WormholeOutput{},
		Updated: []dto.WormholeOutput{},
		Deleted: []string{},
		Errors:  []dto.BatchError{},
	}

	// Create new wormholes
	for _, whInput := range input.Wormholes {
		wormhole, err := s.wormholeService.CreateWormhole(ctx, userID, userName, groupID, whInput)
		if err != nil {
			result.Errors = append(result.Errors, dto.BatchError{
				SignatureID: whInput.FromSignatureID,
				Error:       err.Error(),
			})
			continue
		}

		// Get system names for output
		fromSystem, _ := s.sde.GetSolarSystem(int(wormhole.FromSystemID))
		toSystem, _ := s.sde.GetSolarSystem(int(wormhole.ToSystemID))
		fromSystemName := GetSystemName(s.sde, fromSystem)
		toSystemName := GetSystemName(s.sde, toSystem)

		// Get wormhole static info if available
		var staticInfo *models.WormholeStatic
		if wormhole.WormholeType != "" {
			staticInfo, _ = s.wormholeService.GetStaticInfo(ctx, wormhole.WormholeType)
		}

		output := dto.WormholeToOutput(wormhole, fromSystemName, toSystemName, staticInfo)
		result.Created = append(result.Created, output)
	}

	// TODO: Implement delete old logic if requested
	if input.DeleteOld {
		// Implementation would depend on requirements for which wormholes to delete
	}

	return result, nil
}
