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
		Version:     "008_create_permissions_indexes",
		Description: "Create indexes for permissions collection (permission system)",
		Up:          up008,
		Down:        down008,
	})
}

func up008(ctx context.Context, db *mongo.Database) error {
	permissionsCollection := db.Collection("permissions")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "service", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "resource", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "action", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_static", Value: 1}},
		},
		// Compound index for permission lookups
		{
			Keys: bson.D{
				{Key: "service", Value: 1},
				{Key: "resource", Value: 1},
				{Key: "action", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		// Index for searching permissions by name
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	// Create indexes with timeout and error handling for existing indexes
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := permissionsCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	return nil
}

func down008(ctx context.Context, db *mongo.Database) error {
	permissionsCollection := db.Collection("permissions")
	if _, err := permissionsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
