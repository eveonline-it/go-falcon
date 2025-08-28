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
		Version:     "013_create_site_settings_indexes_and_seed",
		Description: "Create indexes and seed data for site_settings collection",
		Up:          up013,
		Down:        down013,
	})
}

func up013(ctx context.Context, db *mongo.Database) error {
	siteSettingsCollection := db.Collection("site_settings")

	// Create indexes first
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_public", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		// Compound index for public settings
		{
			Keys: bson.D{
				{Key: "is_public", Value: 1},
				{Key: "is_active", Value: 1},
			},
		},
		// Compound index for settings by category
		{
			Keys: bson.D{
				{Key: "category", Value: 1},
				{Key: "is_active", Value: 1},
			},
		},
	}

	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := siteSettingsCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	// Seed default site settings
	now := time.Now()
	defaultSettings := []interface{}{
		bson.M{
			"key":         "site_name",
			"value":       "Go Falcon API Gateway",
			"description": "The name of the site displayed in the UI",
			"category":    "general",
			"type":        "string",
			"is_public":   true,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "maintenance_mode",
			"value":       false,
			"description": "Enable maintenance mode to disable API access",
			"category":    "system",
			"type":        "boolean",
			"is_public":   true,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "max_users",
			"value":       1000,
			"description": "Maximum number of registered users allowed",
			"category":    "limits",
			"type":        "number",
			"is_public":   false,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "api_rate_limit",
			"value":       100,
			"description": "API requests per minute limit per user",
			"category":    "limits",
			"type":        "number",
			"is_public":   false,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "registration_enabled",
			"value":       true,
			"description": "Allow new user registrations",
			"category":    "auth",
			"type":        "boolean",
			"is_public":   true,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "contact_info",
			"value":       "admin@example.com",
			"description": "Contact information for support",
			"category":    "general",
			"type":        "string",
			"is_public":   true,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "managed_corporations",
			"value":       []interface{}{},
			"description": "List of managed corporations with enabled status",
			"category":    "eve",
			"type":        "array",
			"is_public":   false,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		bson.M{
			"key":         "managed_alliances",
			"value":       []interface{}{},
			"description": "List of managed alliances with enabled status",
			"category":    "eve",
			"type":        "array",
			"is_public":   false,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
	}

	// Use insertMany with ordered=false to skip duplicates
	insertOpts := options.InsertMany().SetOrdered(false)
	_, err = siteSettingsCollection.InsertMany(ctx, defaultSettings, insertOpts)
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return err
	}

	return nil
}

func down013(ctx context.Context, db *mongo.Database) error {
	siteSettingsCollection := db.Collection("site_settings")

	// Remove seeded settings
	settingKeys := []string{
		"site_name", "maintenance_mode", "max_users", "api_rate_limit",
		"registration_enabled", "contact_info", "managed_corporations", "managed_alliances",
	}

	_, err := siteSettingsCollection.DeleteMany(ctx, bson.M{
		"key": bson.M{"$in": settingKeys},
	})
	if err != nil {
		return err
	}

	// Drop indexes
	if _, err := siteSettingsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}

	return nil
}
