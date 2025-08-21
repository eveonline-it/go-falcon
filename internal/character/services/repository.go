package services

import (
	"context"
	"strings"
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

// SearchCharactersByName searches characters by name using optimized search strategies
func (r *Repository) SearchCharactersByName(ctx context.Context, name string) ([]*models.Character, error) {
	var filter bson.M
	var findOptions *options.FindOptions
	
	// Use different search strategies based on the search pattern
	if len(name) >= 3 {
		// For partial matches, try text search first (faster for full-text queries)
		// If the query looks like a text search (multiple words or special characters)
		if strings.Contains(name, " ") || len(strings.Fields(name)) > 1 {
			// Use text search for multi-word queries
			filter = bson.M{
				"$text": bson.M{
					"$search": name,
				},
			}
			// Sort by text score for relevance
			findOptions = options.Find().
				SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
				SetSort(bson.M{"score": bson.M{"$meta": "textScore"}}).
				SetLimit(50) // Limit results for performance
		} else {
			// Use case-insensitive regex for single-word prefix/contains search
			// Optimize with anchored regex for prefix search if possible
			regexPattern := "^" + strings.ToLower(name) // Start with prefix search
			if !strings.HasPrefix(name, "^") {
				regexPattern = strings.ToLower(name) // Contains search
			}
			
			filter = bson.M{
				"name": bson.M{
					"$regex":   regexPattern,
					"$options": "i", // case-insensitive
				},
			}
			// Sort by name for consistent results and limit
			findOptions = options.Find().
				SetSort(bson.M{"name": 1}).
				SetLimit(50) // Limit results for performance
		}
	} else {
		// For very short queries, use prefix search only
		filter = bson.M{
			"name": bson.M{
				"$regex":   "^" + strings.ToLower(name),
				"$options": "i",
			},
		}
		findOptions = options.Find().
			SetSort(bson.M{"name": 1}).
			SetLimit(20) // Smaller limit for short queries
	}
	
	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var characters []*models.Character
	if err := cursor.All(ctx, &characters); err != nil {
		return nil, err
	}
	
	return characters, nil
}

// CreateIndexes creates necessary database indexes for the characters collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	// Create unique index on character_id
	characterIDIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "character_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	
	// Create text index on name field for full-text search (multi-word queries)
	nameTextIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: "text"}},
		Options: options.Index().SetName("name_text"),
	}
	
	// Create case-insensitive index on name for prefix/regex searches
	// This supports both prefix searches (^pattern) and general regex searches
	nameRegularIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: 1}},
		Options: options.Index().
			SetName("name_regular").
			SetCollation(&options.Collation{
				Locale:   "en",
				Strength: 2, // Case-insensitive comparison
			}),
	}
	
	// Create compound index for efficient sorting with search
	nameWithTimestampIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
			{Key: "created_at", Value: -1}, // Newest first as secondary sort
		},
		Options: options.Index().
			SetName("name_created_compound").
			SetCollation(&options.Collation{
				Locale:   "en",
				Strength: 2, // Case-insensitive
			}),
	}
	
	indexModels := []mongo.IndexModel{
		characterIDIndex,
		nameTextIndex,
		nameRegularIndex,
		nameWithTimestampIndex,
	}
	
	_, err := r.collection.Indexes().CreateMany(ctx, indexModels)
	return err
}