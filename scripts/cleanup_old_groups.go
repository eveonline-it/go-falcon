package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
)

func main() {
	fmt.Println("🧹 Cleaning up incorrectly created groups...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to MongoDB
	mongodb, err := database.NewMongoDB(cfg.DatabaseURL, cfg.DatabaseName)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close()

	ctx := context.Background()
	groupsCollection := mongodb.Database.Collection("groups")

	// Find groups with old naming convention (Corp_ID, Alliance_ID)
	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": "^Corp_[0-9]+$"}},
			{"name": bson.M{"$regex": "^Alliance_[0-9]+$"}},
		},
	}

	cursor, err := groupsCollection.Find(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to find old groups: %v", err)
	}
	defer cursor.Close(ctx)

	var oldGroups []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Name string             `bson:"name"`
		Type string             `bson:"type"`
	}

	if err := cursor.All(ctx, &oldGroups); err != nil {
		log.Fatalf("Failed to decode old groups: %v", err)
	}

	if len(oldGroups) == 0 {
		fmt.Println("✅ No old groups found to clean up")
		return
	}

	fmt.Printf("Found %d groups to clean up:\n", len(oldGroups))
	for _, group := range oldGroups {
		fmt.Printf("  - %s (%s) - ID: %s\n", group.Name, group.Type, group.ID.Hex())
	}

	// Confirm cleanup
	fmt.Print("\nDo you want to delete these groups? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
		fmt.Println("❌ Cleanup cancelled")
		return
	}

	// Delete the groups and their memberships
	for _, group := range oldGroups {
		// First delete memberships
		membershipFilter := bson.M{"group_id": group.ID}
		membershipResult, err := mongodb.Database.Collection("group_memberships").DeleteMany(ctx, membershipFilter)
		if err != nil {
			fmt.Printf("⚠️  Failed to delete memberships for group %s: %v\n", group.Name, err)
		} else {
			fmt.Printf("🗑️  Deleted %d memberships for group %s\n", membershipResult.DeletedCount, group.Name)
		}

		// Then delete the group
		groupFilter := bson.M{"_id": group.ID}
		groupResult, err := groupsCollection.DeleteOne(ctx, groupFilter)
		if err != nil {
			fmt.Printf("❌ Failed to delete group %s: %v\n", group.Name, err)
		} else if groupResult.DeletedCount > 0 {
			fmt.Printf("✅ Deleted group: %s\n", group.Name)
		} else {
			fmt.Printf("⚠️  Group %s not found (may have been deleted already)\n", group.Name)
		}
	}

	fmt.Printf("\n🎉 Cleanup completed! Deleted %d old groups.\n", len(oldGroups))
}