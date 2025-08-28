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
		Version:     "007_create_auth_states_indexes",
		Description: "Create indexes for auth_states collection (EVE SSO states)",
		Up:          up007,
		Down:        down007,
	})
}

func up007(ctx context.Context, db *mongo.Database) error {
	authStatesCollection := db.Collection("auth_states")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "state", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "redirect_uri", Value: 1}},
		},
		// TTL index for automatic cleanup of expired states
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // TTL based on expires_at field
		},
	}

	// Create indexes with timeout and error handling for existing indexes
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := authStatesCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	return nil
}

func down007(ctx context.Context, db *mongo.Database) error {
	authStatesCollection := db.Collection("auth_states")
	if _, err := authStatesCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}

// Note: isIndexExistsError helper function defined in 006_create_user_profiles_indexes.go
