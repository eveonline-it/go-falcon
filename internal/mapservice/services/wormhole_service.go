package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go-falcon/internal/mapservice/dto"
	"go-falcon/internal/mapservice/models"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WormholeService struct {
	db         *mongo.Database
	redis      *redis.Client
	sde        *sde.Service
	SDEService *sde.Service // Public accessor for routes
}

func NewWormholeService(db *mongo.Database, redis *redis.Client, sdeService *sde.Service) *WormholeService {
	return &WormholeService{
		db:         db,
		redis:      redis,
		sde:        sdeService,
		SDEService: sdeService, // Expose for routes
	}
}

// InitializeStaticData loads static wormhole data
func (s *WormholeService) InitializeStaticData(ctx context.Context) error {
	// Static wormhole type data
	staticWormholes := []models.WormholeStatic{
		// High-sec statics
		{ID: "A641", LeadsTo: "HS", MaxMass: 2000000000, JumpMass: 1000000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to High Security Space"},
		{ID: "B041", LeadsTo: "HS", MaxMass: 5000000000, JumpMass: 2000000000, MassRegenRate: 0, Lifetime: 48, Description: "Wormhole to High Security Space"},

		// Low-sec statics
		{ID: "A239", LeadsTo: "LS", MaxMass: 2000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Low Security Space"},
		{ID: "B449", LeadsTo: "LS", MaxMass: 2000000000, JumpMass: 1000000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Low Security Space"},

		// Null-sec statics
		{ID: "A009", LeadsTo: "NS", MaxMass: 5000000000, JumpMass: 2000000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Null Security Space"},
		{ID: "C248", LeadsTo: "NS", MaxMass: 5000000000, JumpMass: 2000000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Null Security Space"},

		// C1 statics
		{ID: "B274", LeadsTo: "C1", MaxMass: 2000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 1 Space"},
		{ID: "C247", LeadsTo: "C1", MaxMass: 5000000000, JumpMass: 2000000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Class 1 Space"},

		// C2 statics
		{ID: "D382", LeadsTo: "C2", MaxMass: 2000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Class 2 Space"},
		{ID: "O477", LeadsTo: "C2", MaxMass: 2000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 2 Space"},

		// C3 statics
		{ID: "M267", LeadsTo: "C3", MaxMass: 1000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Class 3 Space"},
		{ID: "O883", LeadsTo: "C3", MaxMass: 3000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 3 Space"},

		// C4 statics
		{ID: "E175", LeadsTo: "C4", MaxMass: 2000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Class 4 Space"},
		{ID: "O128", LeadsTo: "C4", MaxMass: 1000000000, JumpMass: 300000000, MassRegenRate: 100000000, Lifetime: 24, Description: "Wormhole to Class 4 Space"},

		// C5 statics
		{ID: "H296", LeadsTo: "C5", MaxMass: 3000000000, JumpMass: 1350000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 5 Space"},
		{ID: "V911", LeadsTo: "C5", MaxMass: 3500000000, JumpMass: 1350000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 5 Space"},

		// C6 statics
		{ID: "H900", LeadsTo: "C6", MaxMass: 3000000000, JumpMass: 1350000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 6 Space"},
		{ID: "U210", LeadsTo: "C6", MaxMass: 3000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Class 6 Space"},

		// Special wormholes
		{ID: "K162", LeadsTo: "Unknown", MaxMass: 0, JumpMass: 0, MassRegenRate: 0, Lifetime: 0, Description: "Exit wormhole (other side of connection)"},

		// Drifter wormholes
		{ID: "C414", LeadsTo: "Drifter", MaxMass: 750000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 16, Description: "Drifter wormhole"},
		{ID: "R474", LeadsTo: "Drifter", MaxMass: 3000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Drifter wormhole"},

		// Thera connections
		{ID: "F135", LeadsTo: "Thera", MaxMass: 750000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Thera"},
		{ID: "L031", LeadsTo: "Thera", MaxMass: 3000000000, JumpMass: 1350000000, MassRegenRate: 0, Lifetime: 16, Description: "Wormhole to Thera"},

		// Frigate-only holes
		{ID: "E004", LeadsTo: "C1", MaxMass: 1000000000, JumpMass: 5000000, MassRegenRate: 0, Lifetime: 16, Description: "Small ship wormhole to C1"},
		{ID: "L005", LeadsTo: "C2", MaxMass: 1000000000, JumpMass: 5000000, MassRegenRate: 0, Lifetime: 16, Description: "Small ship wormhole to C2"},
		{ID: "Z006", LeadsTo: "C3", MaxMass: 1000000000, JumpMass: 5000000, MassRegenRate: 0, Lifetime: 16, Description: "Small ship wormhole to C3"},

		// Capital escalation holes
		{ID: "X702", LeadsTo: "NS", MaxMass: 1000000000, JumpMass: 300000000, MassRegenRate: 0, Lifetime: 24, Description: "Capital escalation hole"},

		// Shattered/Special wormholes
		{ID: "J244", LeadsTo: "LS", MaxMass: 1000000000, JumpMass: 20000000, MassRegenRate: 0, Lifetime: 24, Description: "Wormhole to Low Security from C1/C2/C3"},
		{ID: "Z971", LeadsTo: "C1", MaxMass: 100000000, JumpMass: 20000000, MassRegenRate: 0, Lifetime: 16, Description: "Small ship wormhole to C1"},
	}

	// Insert or update static wormhole data
	for _, wh := range staticWormholes {
		filter := bson.M{"_id": wh.ID}
		update := bson.M{"$set": wh}
		opts := options.Update().SetUpsert(true)

		if _, err := s.db.Collection("map_wormhole_statics").UpdateOne(ctx, filter, update, opts); err != nil {
			return fmt.Errorf("failed to insert wormhole static %s: %w", wh.ID, err)
		}
	}

	return nil
}

// GetStaticInfo retrieves static information for a wormhole type
func (s *WormholeService) GetStaticInfo(ctx context.Context, whType string) (*models.WormholeStatic, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("wormhole:static:%s", whType)
	if cachedStatic, err := s.getStaticFromCache(ctx, cacheKey); err == nil {
		return cachedStatic, nil
	}

	// Query database
	var static models.WormholeStatic
	err := s.db.Collection("map_wormhole_statics").FindOne(ctx, bson.M{"_id": whType}).Decode(&static)
	if err != nil {
		return nil, err
	}

	// Cache the result (static data doesn't change, so cache for 24 hours)
	s.cacheStaticInfo(ctx, cacheKey, &static)

	return &static, nil
}

// CreateWormhole creates a new wormhole connection
func (s *WormholeService) CreateWormhole(ctx context.Context, userID primitive.ObjectID, userName string, groupID *primitive.ObjectID, input dto.CreateWormholeInput) (*models.MapWormhole, error) {
	// Validate systems exist
	if _, err := s.sde.GetSolarSystem(int(input.FromSystemID)); err != nil {
		return nil, fmt.Errorf("invalid from system ID: %w", err)
	}
	if _, err := s.sde.GetSolarSystem(int(input.ToSystemID)); err != nil {
		return nil, fmt.Errorf("invalid to system ID: %w", err)
	}

	wormhole := &models.MapWormhole{
		FromSystemID:    input.FromSystemID,
		ToSystemID:      input.ToSystemID,
		FromSignatureID: input.FromSignatureID,
		ToSignatureID:   input.ToSignatureID,
		WormholeType:    input.WormholeType,
		MassStatus:      input.MassStatus,
		TimeStatus:      input.TimeStatus,
		CreatedBy:       userID,
		CreatedByName:   userName,
		SharingLevel:    input.SharingLevel,
		GroupID:         groupID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Get static info if wormhole type is known
	if input.WormholeType != "" {
		if static, err := s.GetStaticInfo(ctx, input.WormholeType); err == nil {
			wormhole.MaxMass = static.MaxMass
			wormhole.JumpMass = static.JumpMass
			wormhole.MassRegenRate = static.MassRegenRate
			wormhole.RemainingMass = static.MaxMass // Start with full mass

			// Set expiration based on lifetime
			if static.Lifetime > 0 {
				expiresAt := time.Now().Add(time.Duration(static.Lifetime) * time.Hour)

				// EOL wormholes have 4 hours or less remaining
				if input.TimeStatus == "eol" {
					expiresAt = time.Now().Add(4 * time.Hour)
				}

				wormhole.ExpiresAt = &expiresAt
			}
		}
	} else {
		// Default values for unknown wormhole types
		wormhole.MaxMass = 2000000000
		wormhole.JumpMass = 300000000
		wormhole.RemainingMass = 2000000000

		// Default lifetime
		expiresAt := time.Now().Add(24 * time.Hour)
		if input.TimeStatus == "eol" {
			expiresAt = time.Now().Add(4 * time.Hour)
		}
		wormhole.ExpiresAt = &expiresAt
	}

	// Adjust remaining mass based on mass status
	switch input.MassStatus {
	case "destabilized":
		wormhole.RemainingMass = wormhole.MaxMass / 2
	case "critical":
		wormhole.RemainingMass = wormhole.MaxMass / 10
	}

	// Check for existing connection
	filter := bson.M{
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{
						"from_system_id":    input.FromSystemID,
						"from_signature_id": input.FromSignatureID,
					},
					{
						"to_system_id":    input.FromSystemID,
						"to_signature_id": input.FromSignatureID,
					},
				},
			},
			{
				"$or": []bson.M{
					{"expires_at": bson.M{"$gt": time.Now()}},
					{"expires_at": nil},
				},
			},
		},
	}

	var existing models.MapWormhole
	err := s.db.Collection("map_wormholes").FindOne(ctx, filter).Decode(&existing)
	if err == nil {
		// Update existing wormhole
		update := bson.M{
			"$set": bson.M{
				"to_system_id":    input.ToSystemID,
				"to_signature_id": input.ToSignatureID,
				"wormhole_type":   input.WormholeType,
				"mass_status":     input.MassStatus,
				"time_status":     input.TimeStatus,
				"updated_by":      userID,
				"updated_by_name": userName,
				"updated_at":      time.Now(),
			},
		}

		if wormhole.ExpiresAt != nil {
			update["$set"].(bson.M)["expires_at"] = wormhole.ExpiresAt
		}

		err = s.db.Collection("map_wormholes").FindOneAndUpdate(
			ctx,
			bson.M{"_id": existing.ID},
			update,
			options.FindOneAndUpdate().SetReturnDocument(options.After),
		).Decode(wormhole)

		if err != nil {
			return nil, fmt.Errorf("failed to update wormhole: %w", err)
		}
		return wormhole, nil
	}

	// Create new wormhole
	result, err := s.db.Collection("map_wormholes").InsertOne(ctx, wormhole)
	if err != nil {
		return nil, fmt.Errorf("failed to create wormhole: %w", err)
	}

	wormhole.ID = result.InsertedID.(primitive.ObjectID)
	return wormhole, nil
}

// GetWormholes retrieves wormhole connections
func (s *WormholeService) GetWormholes(ctx context.Context, userID primitive.ObjectID, groupIDs []primitive.ObjectID, input dto.GetWormholesInput) ([]models.MapWormhole, error) {
	filter := bson.M{}

	// Filter by system if specified
	if input.SystemID > 0 {
		filter["$or"] = []bson.M{
			{"from_system_id": input.SystemID},
			{"to_system_id": input.SystemID},
		}
	}

	// Handle expiration filter
	if !input.IncludeExpired {
		expirationFilter := bson.M{
			"$or": []bson.M{
				{"expires_at": bson.M{"$gt": time.Now()}},
				{"expires_at": nil},
			},
		}

		if len(filter) > 0 {
			filter = bson.M{"$and": []bson.M{filter, expirationFilter}}
		} else {
			filter = expirationFilter
		}
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

	cursor, err := s.db.Collection("map_wormholes").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get wormholes: %w", err)
	}
	defer cursor.Close(ctx)

	var wormholes []models.MapWormhole
	if err := cursor.All(ctx, &wormholes); err != nil {
		return nil, fmt.Errorf("failed to decode wormholes: %w", err)
	}

	return wormholes, nil
}

// UpdateWormhole updates a wormhole connection
func (s *WormholeService) UpdateWormhole(ctx context.Context, wormholeID primitive.ObjectID, userID primitive.ObjectID, userName string, input dto.UpdateWormholeInput) error {
	update := bson.M{
		"$set": bson.M{
			"updated_by":      userID,
			"updated_by_name": userName,
			"updated_at":      time.Now(),
		},
	}

	// Add fields to update if provided
	if input.ToSignatureID != nil {
		update["$set"].(bson.M)["to_signature_id"] = *input.ToSignatureID
	}
	if input.WormholeType != nil {
		update["$set"].(bson.M)["wormhole_type"] = *input.WormholeType

		// Update mass/lifetime info if type changed
		if static, err := s.GetStaticInfo(ctx, *input.WormholeType); err == nil {
			update["$set"].(bson.M)["max_mass"] = static.MaxMass
			update["$set"].(bson.M)["jump_mass"] = static.JumpMass
			update["$set"].(bson.M)["mass_regen_rate"] = static.MassRegenRate
		}
	}
	if input.MassStatus != nil {
		update["$set"].(bson.M)["mass_status"] = *input.MassStatus

		// Adjust remaining mass estimate
		var wh models.MapWormhole
		s.db.Collection("map_wormholes").FindOne(ctx, bson.M{"_id": wormholeID}).Decode(&wh)

		switch *input.MassStatus {
		case "stable":
			update["$set"].(bson.M)["remaining_mass"] = wh.MaxMass
		case "destabilized":
			update["$set"].(bson.M)["remaining_mass"] = wh.MaxMass / 2
		case "critical":
			update["$set"].(bson.M)["remaining_mass"] = wh.MaxMass / 10
		}
	}
	if input.TimeStatus != nil {
		update["$set"].(bson.M)["time_status"] = *input.TimeStatus

		// Update expiration for EOL
		if *input.TimeStatus == "eol" {
			expiresAt := time.Now().Add(4 * time.Hour)
			update["$set"].(bson.M)["expires_at"] = expiresAt
		}
	}

	result, err := s.db.Collection("map_wormholes").UpdateOne(
		ctx,
		bson.M{"_id": wormholeID},
		update,
	)

	if err != nil {
		return fmt.Errorf("failed to update wormhole: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("wormhole not found or no changes made")
	}

	return nil
}

// DeleteWormhole removes a wormhole connection
func (s *WormholeService) DeleteWormhole(ctx context.Context, wormholeID primitive.ObjectID, userID primitive.ObjectID) error {
	filter := bson.M{
		"_id": wormholeID,
		"$or": []bson.M{
			{"created_by": userID},
			// Add admin permission check here if needed
		},
	}

	result, err := s.db.Collection("map_wormholes").DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete wormhole: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("wormhole not found or insufficient permissions")
	}

	return nil
}

// RecordJump records a jump through a wormhole and updates mass
func (s *WormholeService) RecordJump(ctx context.Context, wormholeID primitive.ObjectID, shipMass int64) error {
	// Get current wormhole state
	var wormhole models.MapWormhole
	err := s.db.Collection("map_wormholes").FindOne(ctx, bson.M{"_id": wormholeID}).Decode(&wormhole)
	if err != nil {
		return fmt.Errorf("wormhole not found: %w", err)
	}

	// Update remaining mass
	newRemainingMass := wormhole.RemainingMass - shipMass
	if newRemainingMass < 0 {
		newRemainingMass = 0
	}

	// Determine new mass status
	massStatus := "stable"
	percentRemaining := float64(newRemainingMass) / float64(wormhole.MaxMass) * 100

	if percentRemaining <= 10 {
		massStatus = "critical"
	} else if percentRemaining <= 50 {
		massStatus = "destabilized"
	}

	// Update wormhole
	update := bson.M{
		"$set": bson.M{
			"remaining_mass": newRemainingMass,
			"mass_status":    massStatus,
			"updated_at":     time.Now(),
		},
	}

	_, err = s.db.Collection("map_wormholes").UpdateOne(ctx, bson.M{"_id": wormholeID}, update)
	return err
}

// Cache helper methods for wormhole static info

func (s *WormholeService) getStaticFromCache(ctx context.Context, key string) (*models.WormholeStatic, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var static models.WormholeStatic
	if err := json.Unmarshal([]byte(data), &static); err != nil {
		return nil, err
	}

	return &static, nil
}

func (s *WormholeService) cacheStaticInfo(ctx context.Context, key string, static *models.WormholeStatic) {
	if s.redis == nil {
		return
	}

	data, err := json.Marshal(static)
	if err != nil {
		return
	}

	// Cache for 24 hours since static wormhole data never changes
	s.redis.Set(ctx, key, data, 24*time.Hour)
}
