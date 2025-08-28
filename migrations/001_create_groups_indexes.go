package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	Register(Migration{
		Version:     "001_create_groups_indexes",
		Description: "Create indexes for groups and group_memberships collections",
		Up:          up001,
		Down:        down001,
	})
}

func up001(ctx context.Context, db *mongo.Database) error {
	// Groups collection indexes
	groupsCollection := db.Collection("groups")
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
		return err
	}

	// Group memberships collection indexes
	membershipsCollection := db.Collection("group_memberships")
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
		return err
	}

	return nil
}

func down001(ctx context.Context, db *mongo.Database) error {
	// Drop all indexes except _id
	groupsCollection := db.Collection("groups")
	if _, err := groupsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}

	membershipsCollection := db.Collection("group_memberships")
	if _, err := membershipsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}

	return nil
}
