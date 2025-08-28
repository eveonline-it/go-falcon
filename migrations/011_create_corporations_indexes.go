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
		Version:     "011_create_corporations_indexes",
		Description: "Create indexes for corporations collection (EVE corporation data)",
		Up:          up011,
		Down:        down011,
	})
}

func up011(ctx context.Context, db *mongo.Database) error {
	corporationsCollection := db.Collection("corporations")

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "corporation_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "ticker", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "alliance_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "ceo_character_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "member_count", Value: -1}}, // Descending for largest first
		},
		{
			Keys: bson.D{{Key: "date_founded", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "faction_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "home_station_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
		// Text search index for corporation name
		{
			Keys:    bson.D{{Key: "name", Value: "text"}},
			Options: options.Index().SetDefaultLanguage("english"),
		},
		// Compound index for alliance corporations
		{
			Keys: bson.D{
				{Key: "alliance_id", Value: 1},
				{Key: "member_count", Value: -1},
			},
		},
		// Compound index for active corporations
		{
			Keys: bson.D{
				{Key: "faction_id", Value: 1},
				{Key: "date_founded", Value: -1},
			},
		},
	}

	// Create regular indexes first
	regularIndexes := indexes[:len(indexes)-3] // All except text and compound indexes
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := corporationsCollection.Indexes().CreateMany(ctx, regularIndexes, opts)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	// Create text index separately
	textIndex := indexes[len(indexes)-3] // The text index
	_, err = corporationsCollection.Indexes().CreateOne(ctx, textIndex)
	if err != nil && !isIndexExistsError(err) {
		return err
	}

	// Create compound indexes
	compoundIndexes := indexes[len(indexes)-2:] // Last 2 compound indexes
	for _, idx := range compoundIndexes {
		_, err = corporationsCollection.Indexes().CreateOne(ctx, idx)
		if err != nil && !isIndexExistsError(err) {
			return err
		}
	}

	return nil
}

func down011(ctx context.Context, db *mongo.Database) error {
	corporationsCollection := db.Collection("corporations")
	if _, err := corporationsCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
