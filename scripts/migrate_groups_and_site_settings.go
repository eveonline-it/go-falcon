package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
)

func main() {
	fmt.Println("🚀 Go Falcon - Groups and Site Settings Migration Script")
	fmt.Println("=========================================================")
	fmt.Println("This script migrates the database for the new group auto-join system")
	fmt.Println("with site settings-based enabled corporations and alliances.")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to MongoDB
	fmt.Println("📡 Connecting to MongoDB...")
	mongodb, err := database.NewMongoDB(cfg.DatabaseURL, cfg.DatabaseName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close()

	ctx := context.Background()

	// Confirm migration
	if !confirmMigration() {
		fmt.Println("❌ Migration cancelled by user")
		os.Exit(0)
	}

	fmt.Println("🔄 Starting migration...")

	// Step 1: Clean slate approach - Drop existing groups and memberships
	if err := dropExistingGroupData(ctx, mongodb); err != nil {
		log.Fatalf("Failed to drop existing group data: %v", err)
	}

	// Step 2: Create new database indexes for groups and site settings
	if err := createIndexes(ctx, mongodb); err != nil {
		log.Fatalf("Failed to create indexes: %v", err)
	}

	// Step 3: Initialize site settings with managed corporations/alliances structure
	if err := initializeSiteSettings(ctx, mongodb); err != nil {
		log.Fatalf("Failed to initialize site settings: %v", err)
	}

	// Step 4: Create system groups (super_admin, authenticated, guest)
	if err := createSystemGroups(ctx, mongodb); err != nil {
		log.Fatalf("Failed to create system groups: %v", err)
	}

	fmt.Println("✅ Migration completed successfully!")
	fmt.Println()
	fmt.Println("📋 Next steps:")
	fmt.Println("1. Add corporations and alliances via the Site Settings API")
	fmt.Println("2. Enable the entities you want to create auto-join groups")
	fmt.Println("3. Characters will automatically join relevant groups on login")
}

func confirmMigration() bool {
	fmt.Println("⚠️  WARNING: This migration will:")
	fmt.Println("   - DROP all existing groups and group memberships")
	fmt.Println("   - Create new groups collection with updated structure")
	fmt.Println("   - Initialize site settings for managed corporations/alliances")
	fmt.Println("   - Create system groups (super_admin, authenticated, guest)")
	fmt.Println()
	fmt.Print("Do you want to continue? (y/N): ")
	
	var response string
	fmt.Scanln(&response)
	
	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}

func dropExistingGroupData(ctx context.Context, mongodb *database.MongoDB) error {
	fmt.Println("🗑️  Dropping existing groups and memberships collections...")
	
	// Drop groups collection
	if err := mongodb.Database.Collection("groups").Drop(ctx); err != nil {
		fmt.Printf("   Note: groups collection drop failed (may not exist): %v\n", err)
	} else {
		fmt.Println("   ✅ Dropped 'groups' collection")
	}
	
	// Drop group_memberships collection
	if err := mongodb.Database.Collection("group_memberships").Drop(ctx); err != nil {
		fmt.Printf("   Note: group_memberships collection drop failed (may not exist): %v\n", err)
	} else {
		fmt.Println("   ✅ Dropped 'group_memberships' collection")
	}

	return nil
}

func createIndexes(ctx context.Context, mongodb *database.MongoDB) error {
	fmt.Println("📊 Creating database indexes...")

	// Groups collection indexes
	groupsCollection := mongodb.Database.Collection("groups")
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

	if _, err := groupsCollection.Indexes().CreateMany(ctx, groupIndexes); err != nil {
		return fmt.Errorf("failed to create group indexes: %w", err)
	}
	fmt.Println("   ✅ Created groups collection indexes")

	// Membership collection indexes
	membershipsCollection := mongodb.Database.Collection("group_memberships")
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

	if _, err := membershipsCollection.Indexes().CreateMany(ctx, membershipIndexes); err != nil {
		return fmt.Errorf("failed to create membership indexes: %w", err)
	}
	fmt.Println("   ✅ Created group_memberships collection indexes")

	// Site settings collection indexes
	settingsCollection := mongodb.Database.Collection("site_settings")
	settingsIndexes := []mongo.IndexModel{
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
			Keys: bson.D{{Key: "category", Value: 1}, {Key: "is_public", Value: 1}},
		},
	}

	if _, err := settingsCollection.Indexes().CreateMany(ctx, settingsIndexes); err != nil {
		return fmt.Errorf("failed to create site settings indexes: %w", err)
	}
	fmt.Println("   ✅ Created site_settings collection indexes")

	return nil
}

func initializeSiteSettings(ctx context.Context, mongodb *database.MongoDB) error {
	fmt.Println("⚙️  Initializing site settings...")

	settingsCollection := mongodb.Database.Collection("site_settings")
	now := time.Now()

	// Initialize managed_corporations setting
	managedCorpsFilter := bson.M{"key": "managed_corporations"}
	managedCorpsUpdate := bson.M{
		"$setOnInsert": bson.M{
			"key":         "managed_corporations",
			"value":       bson.M{"corporations": []interface{}{}},
			"type":        "object",
			"category":    "eve",
			"description": "Managed corporations with enable/disable status and ticker information",
			"is_public":   false,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
	}

	if _, err := settingsCollection.UpdateOne(ctx, managedCorpsFilter, managedCorpsUpdate, options.Update().SetUpsert(true)); err != nil {
		return fmt.Errorf("failed to initialize managed_corporations setting: %w", err)
	}
	fmt.Println("   ✅ Initialized 'managed_corporations' setting")

	// Initialize managed_alliances setting
	managedAlliancesFilter := bson.M{"key": "managed_alliances"}
	managedAlliancesUpdate := bson.M{
		"$setOnInsert": bson.M{
			"key":         "managed_alliances",
			"value":       bson.M{"alliances": []interface{}{}},
			"type":        "object",
			"category":    "eve",
			"description": "Managed alliances with enable/disable status and ticker information",
			"is_public":   false,
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
	}

	if _, err := settingsCollection.UpdateOne(ctx, managedAlliancesFilter, managedAlliancesUpdate, options.Update().SetUpsert(true)); err != nil {
		return fmt.Errorf("failed to initialize managed_alliances setting: %w", err)
	}
	fmt.Println("   ✅ Initialized 'managed_alliances' setting")

	return nil
}

func createSystemGroups(ctx context.Context, mongodb *database.MongoDB) error {
	fmt.Println("👥 Creating system groups...")

	groupsCollection := mongodb.Database.Collection("groups")
	now := time.Now()

	systemGroups := []bson.M{
		{
			"name":        "Super Administrator",
			"description": "Full administrative access to all system operations",
			"type":        "system",
			"system_name": "super_admin",
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		{
			"name":        "Authenticated Users",
			"description": "Users who have successfully authenticated with EVE SSO",
			"type":        "system",
			"system_name": "authenticated",
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
		{
			"name":        "Guest Users",
			"description": "Unauthenticated users with limited access",
			"type":        "system",
			"system_name": "guest",
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		},
	}

	for _, group := range systemGroups {
		filter := bson.M{"system_name": group["system_name"]}
		update := bson.M{"$setOnInsert": group}
		
		result, err := groupsCollection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("failed to create system group %s: %w", group["name"], err)
		}

		if result.UpsertedCount > 0 {
			fmt.Printf("   ✅ Created system group: %s\n", group["name"])
		} else {
			fmt.Printf("   ℹ️  System group already exists: %s\n", group["name"])
		}
	}

	return nil
}