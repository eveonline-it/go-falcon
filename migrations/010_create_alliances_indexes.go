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
		Version:     "010_create_alliances_indexes",
		Description: "Create indexes for alliances collection (EVE alliance data)",
		Up:          up010,
		Down:        down010,
	})
}

func up010(ctx context.Context, db *mongo.Database) error {
	alliancesCollection := db.Collection("alliances")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "alliance_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "ticker", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "executor_corporation_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "date_founded", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "faction_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
		// Text search index for alliance name
		{
			Keys:    bson.D{{Key: "name", Value: "text"}},
			Options: options.Index().SetDefaultLanguage("english"),
		},
		// Compound index for active alliances
		{
			Keys: bson.D{
				{Key: "executor_corporation_id", Value: 1},
				{Key: "date_founded", Value: -1},
			},
		},
	}

	// Create regular indexes first
	regularIndexes := indexes[:len(indexes)-1] // All except text index
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := alliancesCollection.Indexes().CreateMany(ctx, regularIndexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	// Create text index separately
	textIndex := indexes[len(indexes)-2] // The text index
	_, err = alliancesCollection.Indexes().CreateOne(ctx, textIndex)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	return nil
}

func down010(ctx context.Context, db *mongo.Database) error {
	alliancesCollection := db.Collection("alliances")
	if _, err := alliancesCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
