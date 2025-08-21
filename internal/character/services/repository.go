package services

import (
	"context"
	"time"

	"go-falcon/internal/character/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles data persistence for characters
type Repository struct {
	mongodb    *database.MongoDB
	collection *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:    mongodb,
		collection: mongodb.Database.Collection("characters"),
	}
}

// GetCharacterByID retrieves a character by character ID
func (r *Repository) GetCharacterByID(ctx context.Context, characterID int) (*models.Character, error) {
	filter := bson.M{"character_id": characterID}
	
	var character models.Character
	err := r.collection.FindOne(ctx, filter).Decode(&character)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	
	return &character, nil
}

// SaveCharacter saves or updates a character profile
func (r *Repository) SaveCharacter(ctx context.Context, character *models.Character) error {
	character.UpdatedAt = time.Now()
	if character.CreatedAt.IsZero() {
		character.CreatedAt = time.Now()
	}

	filter := bson.M{"character_id": character.CharacterID}
	update := bson.M{"$set": character}
	opts := options.Update().SetUpsert(true)
	
	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// CreateIndexes creates necessary database indexes for the characters collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "character_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	
	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}