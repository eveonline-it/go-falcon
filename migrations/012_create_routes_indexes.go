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
		Version:     "012_create_routes_indexes",
		Description: "Create indexes for routes collection (dynamic routing system)",
		Up:          up012,
		Down:        down012,
	})
}

func up012(ctx context.Context, db *mongo.Database) error {
	routesCollection := db.Collection("routes")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "route_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "path", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "type", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "parent_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "nav_position", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "nav_order", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "module", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
		// Compound index for navigation ordering
		{
			Keys: bson.D{
				{Key: "nav_position", Value: 1},
				{Key: "nav_order", Value: 1},
			},
		},
		// Compound index for enabled routes by type
		{
			Keys: bson.D{
				{Key: "is_enabled", Value: 1},
				{Key: "type", Value: 1},
			},
		},
		// Compound index for hierarchical navigation
		{
			Keys: bson.D{
				{Key: "parent_id", Value: 1},
				{Key: "nav_order", Value: 1},
			},
		},
		// Compound index for module routes
		{
			Keys: bson.D{
				{Key: "module", Value: 1},
				{Key: "is_enabled", Value: 1},
			},
		},
	}

	// Create indexes with timeout and error handling for existing indexes
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := routesCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	return nil
}

func down012(ctx context.Context, db *mongo.Database) error {
	routesCollection := db.Collection("routes")
	if _, err := routesCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
