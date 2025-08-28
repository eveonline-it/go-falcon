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
		Version:     "003_seed_system_groups",
		Description: "Create initial system groups (super_admin, authenticated, guest)",
		Up:          up003,
		Down:        down003,
	})
}

func up003(ctx context.Context, db *mongo.Database) error {
	groupsCollection := db.Collection("groups")

	systemGroups := []interface{}{
		bson.M{
			"name":        "Super Administrator",
			"description": "System administrators with full access",
			"type":        "system",
			"system_name": "super_admin",
			"is_active":   true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		},
		bson.M{
			"name":        "Authenticated Users",
			"description": "Users who registered with EVE Online scopes",
			"type":        "system",
			"system_name": "authenticated",
			"is_active":   true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		},
		bson.M{
			"name":        "Guest Users",
			"description": "Users who logged in without EVE scopes",
			"type":        "system",
			"system_name": "guest",
			"is_active":   true,
			"created_at":  time.Now(),
			"updated_at":  time.Now(),
		},
	}

	// Use InsertMany with ordered=false to skip duplicates
	opts := options.InsertMany().SetOrdered(false)
	_, err := groupsCollection.InsertMany(ctx, systemGroups, opts)

	// Ignore duplicate key errors
	if err != nil {
		if !mongo.IsDuplicateKeyError(err) {
			return err
		}
	}

	return nil
}

func down003(ctx context.Context, db *mongo.Database) error {
	groupsCollection := db.Collection("groups")

	// Remove system groups
	_, err := groupsCollection.DeleteMany(ctx, bson.M{
		"system_name": bson.M{
			"$in": []string{"super_admin", "authenticated", "guest"},
		},
	})

	return err
}
