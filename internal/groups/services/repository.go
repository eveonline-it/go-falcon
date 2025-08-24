package services

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-falcon/internal/groups/models"
	"go-falcon/pkg/database"
)

// Repository handles database operations for groups
type Repository struct {
	db                    *database.MongoDB
	groupsCollection      *mongo.Collection
	membershipsCollection *mongo.Collection
	charactersCollection  *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(db *database.MongoDB) *Repository {
	return &Repository{
		db:                    db,
		groupsCollection:      db.Database.Collection(models.GroupsCollection),
		membershipsCollection: db.Database.Collection(models.MembershipsCollection),
		charactersCollection:  db.Database.Collection("characters"),
	}
}

// CreateIndexes creates the necessary database indexes
func (r *Repository) CreateIndexes(ctx context.Context) error {
	// Groups collection indexes
	groupIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "system_name", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.D{{Key: "eve_entity_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
	}

	// Membership collection indexes
	membershipIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "group_id", Value: 1}, {Key: "character_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "character_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "group_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
	}

	// Create indexes for groups collection
	if _, err := r.groupsCollection.Indexes().CreateMany(ctx, groupIndexes); err != nil {
		return fmt.Errorf("failed to create group indexes: %w", err)
	}

	// Create indexes for memberships collection
	if _, err := r.membershipsCollection.Indexes().CreateMany(ctx, membershipIndexes); err != nil {
		return fmt.Errorf("failed to create membership indexes: %w", err)
	}

	return nil
}

// CreateGroup creates a new group
func (r *Repository) CreateGroup(ctx context.Context, group *models.Group) error {
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	result, err := r.groupsCollection.InsertOne(ctx, group)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	group.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetGroupByID retrieves a group by its ID
func (r *Repository) GetGroupByID(ctx context.Context, id primitive.ObjectID) (*models.Group, error) {
	var group models.Group
	err := r.groupsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return &group, nil
}

// GetGroupByName retrieves a group by its name
func (r *Repository) GetGroupByName(ctx context.Context, name string) (*models.Group, error) {
	var group models.Group
	err := r.groupsCollection.FindOne(ctx, bson.M{"name": name}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get group by name: %w", err)
	}
	return &group, nil
}

// GetGroupBySystemName retrieves a system group by its system name
func (r *Repository) GetGroupBySystemName(ctx context.Context, systemName string) (*models.Group, error) {
	var group models.Group
	err := r.groupsCollection.FindOne(ctx, bson.M{"system_name": systemName}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get system group: %w", err)
	}
	return &group, nil
}

// GetGroupByEVEEntityID retrieves a group by EVE entity ID (corporation or alliance)
func (r *Repository) GetGroupByEVEEntityID(ctx context.Context, eveEntityID int64) (*models.Group, error) {
	filter := bson.M{"eve_entity_id": eveEntityID}

	var group models.Group
	err := r.groupsCollection.FindOne(ctx, filter).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get group by EVE entity ID: %w", err)
	}

	return &group, nil
}

// ListGroups retrieves groups with filtering and pagination
func (r *Repository) ListGroups(ctx context.Context, filter bson.M, page, limit int) ([]models.Group, int64, error) {
	// Get total count
	total, err := r.groupsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count groups: %w", err)
	}

	// Get paginated results
	skip := (page - 1) * limit
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.groupsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find groups: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, 0, fmt.Errorf("failed to decode groups: %w", err)
	}

	return groups, total, nil
}

// UpdateGroup updates a group
func (r *Repository) UpdateGroup(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	update["updated_at"] = time.Now()

	result, err := r.groupsCollection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}

// DeleteGroup deletes a group
func (r *Repository) DeleteGroup(ctx context.Context, id primitive.ObjectID) error {
	// First, delete all memberships for this group
	_, err := r.membershipsCollection.DeleteMany(ctx, bson.M{"group_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete group memberships: %w", err)
	}

	// Then delete the group
	result, err := r.groupsCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}

// AddMembership adds a character to a group
func (r *Repository) AddMembership(ctx context.Context, membership *models.GroupMembership) error {
	now := time.Now()

	// Use upsert to handle duplicates gracefully
	filter := bson.M{
		"group_id":     membership.GroupID,
		"character_id": membership.CharacterID,
	}

	update := bson.M{
		"$set": bson.M{
			"is_active":  membership.IsActive,
			"added_by":   membership.AddedBy,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"added_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := r.membershipsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to add membership: %w", err)
	}

	if result.UpsertedID != nil {
		membership.ID = result.UpsertedID.(primitive.ObjectID)
	}

	return nil
}

// RemoveMembership removes a character from a group
func (r *Repository) RemoveMembership(ctx context.Context, groupID primitive.ObjectID, characterID int64) error {
	result, err := r.membershipsCollection.DeleteOne(ctx, bson.M{
		"group_id":     groupID,
		"character_id": characterID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove membership: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("membership not found")
	}

	return nil
}

// GetMembership retrieves a specific membership
func (r *Repository) GetMembership(ctx context.Context, groupID primitive.ObjectID, characterID int64) (*models.GroupMembership, error) {
	var membership models.GroupMembership
	err := r.membershipsCollection.FindOne(ctx, bson.M{
		"group_id":     groupID,
		"character_id": characterID,
	}).Decode(&membership)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get membership: %w", err)
	}
	return &membership, nil
}

// ListMemberships retrieves memberships for a group with pagination
func (r *Repository) ListMemberships(ctx context.Context, groupID primitive.ObjectID, filter bson.M, page, limit int) ([]models.GroupMembership, int64, error) {
	// Add group filter
	filter["group_id"] = groupID

	// Get total count
	total, err := r.membershipsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count memberships: %w", err)
	}

	// Get paginated results
	skip := (page - 1) * limit
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "added_at", Value: -1}})

	cursor, err := r.membershipsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find memberships: %w", err)
	}
	defer cursor.Close(ctx)

	var memberships []models.GroupMembership
	if err := cursor.All(ctx, &memberships); err != nil {
		return nil, 0, fmt.Errorf("failed to decode memberships: %w", err)
	}

	return memberships, total, nil
}

// GetCharacterGroups retrieves all groups a character belongs to
func (r *Repository) GetCharacterGroups(ctx context.Context, characterID int64, filter bson.M) ([]models.Group, error) {
	// First, get all active memberships for the character
	membershipFilter := bson.M{
		"character_id": characterID,
		"is_active":    true,
	}

	cursor, err := r.membershipsCollection.Find(ctx, membershipFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find character memberships: %w", err)
	}
	defer cursor.Close(ctx)

	var memberships []models.GroupMembership
	if err := cursor.All(ctx, &memberships); err != nil {
		return nil, fmt.Errorf("failed to decode memberships: %w", err)
	}

	if len(memberships) == 0 {
		return []models.Group{}, nil
	}

	// Extract group IDs
	groupIDs := make([]primitive.ObjectID, len(memberships))
	for i, membership := range memberships {
		groupIDs[i] = membership.GroupID
	}

	// Get groups
	groupFilter := bson.M{"_id": bson.M{"$in": groupIDs}}
	// Merge additional filters
	for k, v := range filter {
		groupFilter[k] = v
	}

	cursor, err = r.groupsCollection.Find(ctx, groupFilter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find character groups: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode character groups: %w", err)
	}

	return groups, nil
}

// GetGroupMemberCount returns the number of active members in a group
func (r *Repository) GetGroupMemberCount(ctx context.Context, groupID primitive.ObjectID) (int64, error) {
	filter := bson.M{
		"group_id":  groupID,
		"is_active": true,
	}

	count, err := r.membershipsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count group members: %w", err)
	}

	return count, nil
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.db.Client.Ping(ctx, nil)
}

// GetCharacterNames fetches character names for a list of character IDs
func (r *Repository) GetCharacterNames(ctx context.Context, characterIDs []int64) (map[int64]string, error) {
	// Build filter for character IDs
	filter := bson.M{
		"character_id": bson.M{"$in": characterIDs},
	}

	// Query only the fields we need
	projection := bson.M{
		"character_id": 1,
		"name":         1,
	}

	cursor, err := r.charactersCollection.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer cursor.Close(ctx)

	// Build the map of character_id -> name
	names := make(map[int64]string)
	for cursor.Next(ctx) {
		var char struct {
			CharacterID int64  `bson:"character_id"`
			Name        string `bson:"name"`
		}
		if err := cursor.Decode(&char); err != nil {
			continue // Skip characters we can't decode
		}
		names[char.CharacterID] = char.Name
	}

	return names, nil
}

// GetCharacterIDsByUserID fetches all character IDs for a given user_id
func (r *Repository) GetCharacterIDsByUserID(ctx context.Context, userID string) ([]int64, error) {
	// Query the user_profiles collection (same as used by auth/users modules)
	userProfilesCollection := r.db.Database.Collection("user_profiles")

	// Build filter for user_id
	filter := bson.M{"user_id": userID}

	// Query only the character_id field we need
	projection := bson.M{"character_id": 1}

	cursor, err := userProfilesCollection.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, fmt.Errorf("failed to query user profiles: %w", err)
	}
	defer cursor.Close(ctx)

	// Build the list of character IDs
	var characterIDs []int64
	for cursor.Next(ctx) {
		var userProfile struct {
			CharacterID int64 `bson:"character_id"`
		}
		if err := cursor.Decode(&userProfile); err != nil {
			continue // Skip profiles we can't decode
		}
		characterIDs = append(characterIDs, int64(userProfile.CharacterID))
	}

	return characterIDs, nil
}
