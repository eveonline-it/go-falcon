package migrations

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	Register(Migration{
		Version:     "006_create_user_profiles_indexes",
		Description: "Create indexes for user_profiles collection (auth system)",
		Up:          up006,
		Down:        down006,
	})
}

func up006(ctx context.Context, db *mongo.Database) error {
	userProfilesCollection := db.Collection("user_profiles")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "character_owner_hash", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "character_name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "last_login", Value: -1}}, // Descending for recent first
		},
		{
			Keys: bson.D{{Key: "valid", Value: 1}}, // For active user queries
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "corporation_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "alliance_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "scopes", Value: 1}}, // For scope-based queries
		},
	}

	// Create indexes with timeout and error handling for existing indexes
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := userProfilesCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	return nil
}

func down006(ctx context.Context, db *mongo.Database) error {
	userProfilesCollection := db.Collection("user_profiles")
	if _, err := userProfilesCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}

// Note: isIndexExistsError helper function defined in helpers.go
