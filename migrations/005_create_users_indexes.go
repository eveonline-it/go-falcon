package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	Register(Migration{
		Version:     "005_create_users_indexes",
		Description: "Create indexes for users collection",
		Up:          up005,
		Down:        down005,
	})
}

func up005(ctx context.Context, db *mongo.Database) error {
	usersCollection := db.Collection("users")

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
			Keys: bson.D{{Key: "character_name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "corporation_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "alliance_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "last_login", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
	}

	if _, err := usersCollection.Indexes().CreateMany(ctx, indexes); err != nil {
		return err
	}

	return nil
}

func down005(ctx context.Context, db *mongo.Database) error {
	usersCollection := db.Collection("users")
	if _, err := usersCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
