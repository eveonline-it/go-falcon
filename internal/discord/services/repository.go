package services

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-falcon/internal/discord/models"
	"go-falcon/pkg/database"
)

// Repository handles database operations for the Discord module
type Repository struct {
	db *database.MongoDB
}

// NewRepository creates a new Discord repository
func NewRepository(db *database.MongoDB) *Repository {
	return &Repository{
		db: db,
	}
}

// CreateIndexes creates database indexes for Discord collections
func (r *Repository) CreateIndexes(ctx context.Context) error {
	// Discord users collection indexes
	usersCollection := r.db.Collection(models.DiscordUsersCollection)
	usersIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "discord_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "discord_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "is_active", Value: 1}}},
		{Keys: bson.D{{Key: "token_expiry", Value: 1}}},
	}
	if _, err := usersCollection.Indexes().CreateMany(ctx, usersIndexes); err != nil {
		return fmt.Errorf("failed to create discord users indexes: %w", err)
	}

	// Guild configs collection indexes
	guildsCollection := r.db.Collection(models.DiscordGuildConfigsCollection)
	guildsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "guild_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "is_enabled", Value: 1}}},
	}
	if _, err := guildsCollection.Indexes().CreateMany(ctx, guildsIndexes); err != nil {
		return fmt.Errorf("failed to create guild configs indexes: %w", err)
	}

	// Role mappings collection indexes
	mappingsCollection := r.db.Collection(models.DiscordRoleMappingsCollection)
	mappingsIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "guild_id", Value: 1}}},
		{Keys: bson.D{{Key: "group_id", Value: 1}}},
		{Keys: bson.D{{Key: "guild_id", Value: 1}, {Key: "group_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "discord_role_id", Value: 1}}},
		{Keys: bson.D{{Key: "is_active", Value: 1}}},
	}
	if _, err := mappingsCollection.Indexes().CreateMany(ctx, mappingsIndexes); err != nil {
		return fmt.Errorf("failed to create role mappings indexes: %w", err)
	}

	// Sync status collection indexes
	statusCollection := r.db.Collection(models.DiscordSyncStatusCollection)
	statusIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "guild_id", Value: 1}}},
		{Keys: bson.D{{Key: "last_sync_at", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	}
	if _, err := statusCollection.Indexes().CreateMany(ctx, statusIndexes); err != nil {
		return fmt.Errorf("failed to create sync status indexes: %w", err)
	}

	// OAuth states collection indexes
	statesCollection := r.db.Collection(models.DiscordOAuthStatesCollection)
	statesIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "state", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)}, // TTL index
	}
	if _, err := statesCollection.Indexes().CreateMany(ctx, statesIndexes); err != nil {
		return fmt.Errorf("failed to create oauth states indexes: %w", err)
	}

	return nil
}

// Discord Users Operations

// CreateDiscordUser creates a new Discord user record
func (r *Repository) CreateDiscordUser(ctx context.Context, user *models.DiscordUser) error {
	user.LinkedAt = time.Now()
	user.UpdatedAt = time.Now()

	collection := r.db.Collection(models.DiscordUsersCollection)
	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create discord user: %w", err)
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetDiscordUserByID gets a Discord user by database ID
func (r *Repository) GetDiscordUserByID(ctx context.Context, id primitive.ObjectID) (*models.DiscordUser, error) {
	collection := r.db.Collection(models.DiscordUsersCollection)
	var user models.DiscordUser

	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get discord user by ID: %w", err)
	}

	return &user, nil
}

// GetDiscordUserByDiscordID gets a Discord user by Discord ID
func (r *Repository) GetDiscordUserByDiscordID(ctx context.Context, discordID string) (*models.DiscordUser, error) {
	collection := r.db.Collection(models.DiscordUsersCollection)
	var user models.DiscordUser

	err := collection.FindOne(ctx, bson.M{"discord_id": discordID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get discord user by discord ID: %w", err)
	}

	return &user, nil
}

// GetDiscordUsersByUserID gets all Discord users linked to a Go Falcon user
func (r *Repository) GetDiscordUsersByUserID(ctx context.Context, userID string) ([]*models.DiscordUser, error) {
	collection := r.db.Collection(models.DiscordUsersCollection)

	cursor, err := collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to find discord users by user ID: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.DiscordUser
	for cursor.Next(ctx) {
		var user models.DiscordUser
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode discord user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

// UpdateDiscordUser updates a Discord user record
func (r *Repository) UpdateDiscordUser(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	update["updated_at"] = time.Now()

	collection := r.db.Collection(models.DiscordUsersCollection)
	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("failed to update discord user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("discord user not found")
	}

	return nil
}

// DeleteDiscordUser deletes a Discord user record
func (r *Repository) DeleteDiscordUser(ctx context.Context, id primitive.ObjectID) error {
	collection := r.db.Collection(models.DiscordUsersCollection)
	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete discord user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("discord user not found")
	}

	return nil
}

// ListDiscordUsers lists Discord users with pagination and filtering
func (r *Repository) ListDiscordUsers(ctx context.Context, filter bson.M, page, limit int) ([]*models.DiscordUser, int64, error) {
	collection := r.db.Collection(models.DiscordUsersCollection)

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count discord users: %w", err)
	}

	// Get paginated results
	skip := (page - 1) * limit
	opts := options.Find().
		SetSort(bson.D{{Key: "linked_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find discord users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.DiscordUser
	for cursor.Next(ctx) {
		var user models.DiscordUser
		if err := cursor.Decode(&user); err != nil {
			return nil, 0, fmt.Errorf("failed to decode discord user: %w", err)
		}
		users = append(users, &user)
	}

	return users, total, nil
}

// Guild Configuration Operations

// CreateGuildConfig creates a new Discord guild configuration
func (r *Repository) CreateGuildConfig(ctx context.Context, config *models.DiscordGuildConfig) error {
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	collection := r.db.Collection(models.DiscordGuildConfigsCollection)
	result, err := collection.InsertOne(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create guild config: %w", err)
	}

	config.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetGuildConfigByGuildID gets guild configuration by Discord guild ID
func (r *Repository) GetGuildConfigByGuildID(ctx context.Context, guildID string) (*models.DiscordGuildConfig, error) {
	collection := r.db.Collection(models.DiscordGuildConfigsCollection)
	var config models.DiscordGuildConfig

	err := collection.FindOne(ctx, bson.M{"guild_id": guildID}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get guild config: %w", err)
	}

	return &config, nil
}

// UpdateGuildConfig updates a guild configuration
func (r *Repository) UpdateGuildConfig(ctx context.Context, guildID string, update bson.M) error {
	update["updated_at"] = time.Now()

	collection := r.db.Collection(models.DiscordGuildConfigsCollection)
	result, err := collection.UpdateOne(ctx, bson.M{"guild_id": guildID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("failed to update guild config: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("guild config not found")
	}

	return nil
}

// DeleteGuildConfig deletes a guild configuration
func (r *Repository) DeleteGuildConfig(ctx context.Context, guildID string) error {
	collection := r.db.Collection(models.DiscordGuildConfigsCollection)
	result, err := collection.DeleteOne(ctx, bson.M{"guild_id": guildID})
	if err != nil {
		return fmt.Errorf("failed to delete guild config: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("guild config not found")
	}

	return nil
}

// ListGuildConfigs lists guild configurations with pagination and filtering
func (r *Repository) ListGuildConfigs(ctx context.Context, filter bson.M, page, limit int) ([]*models.DiscordGuildConfig, int64, error) {
	collection := r.db.Collection(models.DiscordGuildConfigsCollection)

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count guild configs: %w", err)
	}

	// Get paginated results
	skip := (page - 1) * limit
	opts := options.Find().
		SetSort(bson.D{{Key: "guild_name", Value: 1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find guild configs: %w", err)
	}
	defer cursor.Close(ctx)

	var configs []*models.DiscordGuildConfig
	for cursor.Next(ctx) {
		var config models.DiscordGuildConfig
		if err := cursor.Decode(&config); err != nil {
			return nil, 0, fmt.Errorf("failed to decode guild config: %w", err)
		}
		configs = append(configs, &config)
	}

	return configs, total, nil
}

// Role Mapping Operations

// CreateRoleMapping creates a new role mapping
func (r *Repository) CreateRoleMapping(ctx context.Context, mapping *models.DiscordRoleMapping) error {
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()

	collection := r.db.Collection(models.DiscordRoleMappingsCollection)
	result, err := collection.InsertOne(ctx, mapping)
	if err != nil {
		return fmt.Errorf("failed to create role mapping: %w", err)
	}

	mapping.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetRoleMappingByID gets a role mapping by database ID
func (r *Repository) GetRoleMappingByID(ctx context.Context, id primitive.ObjectID) (*models.DiscordRoleMapping, error) {
	collection := r.db.Collection(models.DiscordRoleMappingsCollection)
	var mapping models.DiscordRoleMapping

	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&mapping)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get role mapping: %w", err)
	}

	return &mapping, nil
}

// GetRoleMappingsByGuildID gets all role mappings for a guild
func (r *Repository) GetRoleMappingsByGuildID(ctx context.Context, guildID string) ([]*models.DiscordRoleMapping, error) {
	collection := r.db.Collection(models.DiscordRoleMappingsCollection)

	cursor, err := collection.Find(ctx, bson.M{"guild_id": guildID, "is_active": true})
	if err != nil {
		return nil, fmt.Errorf("failed to find role mappings: %w", err)
	}
	defer cursor.Close(ctx)

	var mappings []*models.DiscordRoleMapping
	for cursor.Next(ctx) {
		var mapping models.DiscordRoleMapping
		if err := cursor.Decode(&mapping); err != nil {
			return nil, fmt.Errorf("failed to decode role mapping: %w", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, nil
}

// UpdateRoleMapping updates a role mapping
func (r *Repository) UpdateRoleMapping(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	update["updated_at"] = time.Now()

	collection := r.db.Collection(models.DiscordRoleMappingsCollection)
	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("failed to update role mapping: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("role mapping not found")
	}

	return nil
}

// DeleteRoleMapping deletes a role mapping
func (r *Repository) DeleteRoleMapping(ctx context.Context, id primitive.ObjectID) error {
	collection := r.db.Collection(models.DiscordRoleMappingsCollection)
	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete role mapping: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("role mapping not found")
	}

	return nil
}

// ListRoleMappings lists role mappings with pagination and filtering
func (r *Repository) ListRoleMappings(ctx context.Context, filter bson.M, page, limit int) ([]*models.DiscordRoleMapping, int64, error) {
	collection := r.db.Collection(models.DiscordRoleMappingsCollection)

	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count role mappings: %w", err)
	}

	// Get paginated results
	skip := (page - 1) * limit
	opts := options.Find().
		SetSort(bson.D{{Key: "group_name", Value: 1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find role mappings: %w", err)
	}
	defer cursor.Close(ctx)

	var mappings []*models.DiscordRoleMapping
	for cursor.Next(ctx) {
		var mapping models.DiscordRoleMapping
		if err := cursor.Decode(&mapping); err != nil {
			return nil, 0, fmt.Errorf("failed to decode role mapping: %w", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, total, nil
}

// GetRoleMappingsByGroupIDs gets role mappings for specific group IDs
func (r *Repository) GetRoleMappingsByGroupIDs(ctx context.Context, groupIDs []primitive.ObjectID) ([]*models.DiscordRoleMapping, error) {
	collection := r.db.Collection(models.DiscordRoleMappingsCollection)

	filter := bson.M{
		"group_id":  bson.M{"$in": groupIDs},
		"is_active": true,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find role mappings: %w", err)
	}
	defer cursor.Close(ctx)

	var mappings []*models.DiscordRoleMapping
	for cursor.Next(ctx) {
		var mapping models.DiscordRoleMapping
		if err := cursor.Decode(&mapping); err != nil {
			return nil, fmt.Errorf("failed to decode role mapping: %w", err)
		}
		mappings = append(mappings, &mapping)
	}

	return mappings, nil
}

// OAuth State Operations

// CreateOAuthState creates a new OAuth state record
func (r *Repository) CreateOAuthState(ctx context.Context, state *models.DiscordOAuthState) error {
	state.CreatedAt = time.Now()

	collection := r.db.Collection(models.DiscordOAuthStatesCollection)
	result, err := collection.InsertOne(ctx, state)
	if err != nil {
		return fmt.Errorf("failed to create oauth state: %w", err)
	}

	state.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetOAuthState gets and deletes an OAuth state (one-time use)
func (r *Repository) GetOAuthState(ctx context.Context, state string) (*models.DiscordOAuthState, error) {
	collection := r.db.Collection(models.DiscordOAuthStatesCollection)

	// Find and delete in one operation
	var oauthState models.DiscordOAuthState
	err := collection.FindOneAndDelete(ctx, bson.M{
		"state":      state,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&oauthState)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get oauth state: %w", err)
	}

	return &oauthState, nil
}

// Sync Status Operations

// CreateSyncStatus creates a new sync status record
func (r *Repository) CreateSyncStatus(ctx context.Context, status *models.DiscordSyncStatus) error {
	status.CreatedAt = time.Now()

	collection := r.db.Collection(models.DiscordSyncStatusCollection)
	result, err := collection.InsertOne(ctx, status)
	if err != nil {
		return fmt.Errorf("failed to create sync status: %w", err)
	}

	status.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// UpdateSyncStatus updates a sync status record
func (r *Repository) UpdateSyncStatus(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	collection := r.db.Collection(models.DiscordSyncStatusCollection)
	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("sync status not found")
	}

	return nil
}

// GetRecentSyncStatus gets recent sync status records
func (r *Repository) GetRecentSyncStatus(ctx context.Context, guildID string, limit int) ([]*models.DiscordSyncStatus, error) {
	collection := r.db.Collection(models.DiscordSyncStatusCollection)

	filter := bson.M{}
	if guildID != "" {
		filter["guild_id"] = guildID
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "last_sync_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find sync status: %w", err)
	}
	defer cursor.Close(ctx)

	var statuses []*models.DiscordSyncStatus
	for cursor.Next(ctx) {
		var status models.DiscordSyncStatus
		if err := cursor.Decode(&status); err != nil {
			return nil, fmt.Errorf("failed to decode sync status: %w", err)
		}
		statuses = append(statuses, &status)
	}

	return statuses, nil
}

// Health Check

// CheckHealth checks database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.db.Client.Ping(ctx, nil)
}
