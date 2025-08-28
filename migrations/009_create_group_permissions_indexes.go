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
		Version:     "009_create_group_permissions_indexes",
		Description: "Create indexes for group_permissions collection (group-permission assignments)",
		Up:          up009,
		Down:        down009,
	})
}

func up009(ctx context.Context, db *mongo.Database) error {
	groupPermissionsCollection := db.Collection("group_permissions")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "group_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "permission_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
		// Compound index for group-permission relationships (unique)
		{
			Keys: bson.D{
				{Key: "group_id", Value: 1},
				{Key: "permission_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		// Index for permission-based queries
		{
			Keys: bson.D{
				{Key: "permission_id", Value: 1},
				{Key: "is_active", Value: 1},
			},
		},
		// Index for group-based queries
		{
			Keys: bson.D{
				{Key: "group_id", Value: 1},
				{Key: "is_active", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	}

	// Create indexes with timeout and error handling for existing indexes
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := groupPermissionsCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	return nil
}

func down009(ctx context.Context, db *mongo.Database) error {
	groupPermissionsCollection := db.Collection("group_permissions")
	if _, err := groupPermissionsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
