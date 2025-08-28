package migrations

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	Register(Migration{
		Version:     "004_create_character_indexes",
		Description: "Create indexes for characters collection",
		Up:          up004,
		Down:        down004,
	})
}

func up004(ctx context.Context, db *mongo.Database) error {
	charactersCollection := db.Collection("characters")

	// Non-text indexes
	indexes := []mongo.IndexModel{
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
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	}

	// Create regular indexes with ordered=false to continue on duplicates
	opts := options.CreateIndexes().SetMaxTime(30 * time.Second)
	_, err := charactersCollection.Indexes().CreateMany(ctx, indexes, opts)
	if err != nil {
		// Check if error is due to existing indexes (acceptable)
		if !mongo.IsDuplicateKeyError(err) && !isIndexExistsError(err) {
			return err
		}
	}

	// Handle text index separately (might already exist with different name)
	textIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "character_name", Value: "text"}},
		Options: options.Index().SetDefaultLanguage("english"),
	}

	_, err = charactersCollection.Indexes().CreateOne(ctx, textIndex)
	if err != nil {
		// If text index already exists with different name, that's OK
		if !isIndexExistsError(err) && !isTextIndexConflictError(err) {
			return err
		}
	}

	return nil
}

// Helper function to check if error is due to text index conflict
func isTextIndexConflictError(err error) bool {
	return strings.Contains(err.Error(), "IndexOptionsConflict") ||
		strings.Contains(err.Error(), "equivalent index already exists")
}

// Note: isIndexExistsError helper function defined in helpers.go

func down004(ctx context.Context, db *mongo.Database) error {
	charactersCollection := db.Collection("characters")
	if _, err := charactersCollection.Indexes().DropAll(ctx); err != nil {
		return err
	}
	return nil
}
