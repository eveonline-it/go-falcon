package services

import (
	"context"
	"strings"
	"time"

	"go-falcon/internal/corporation/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for corporations
type Repository struct {
	mongodb    *database.MongoDB
	collection *mongo.Collection
}

// NewRepository creates a new corporation repository
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:    mongodb,
		collection: mongodb.Database.Collection(models.CorporationCollection),
	}
}

// GetCorporationByID retrieves a corporation by its ID from the database
func (r *Repository) GetCorporationByID(ctx context.Context, corporationID int) (*models.Corporation, error) {
	var corporation models.Corporation
	filter := bson.M{"corporation_id": corporationID, "deleted_at": bson.M{"$exists": false}}

	err := r.collection.FindOne(ctx, filter).Decode(&corporation)
	if err != nil {
		return nil, err
	}

	return &corporation, nil
}

// CreateCorporation creates a new corporation record in the database
func (r *Repository) CreateCorporation(ctx context.Context, corporation *models.Corporation) error {
	corporation.CreatedAt = time.Now().UTC()
	corporation.UpdatedAt = time.Now().UTC()

	_, err := r.collection.InsertOne(ctx, corporation)
	return err
}

// UpdateCorporation updates an existing corporation record
func (r *Repository) UpdateCorporation(ctx context.Context, corporation *models.Corporation) error {
	corporation.UpdatedAt = time.Now().UTC()

	filter := bson.M{"corporation_id": corporation.CorporationID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{"$set": corporation}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// GetAllCorporationIDs retrieves all corporation IDs from the database
func (r *Repository) GetAllCorporationIDs(ctx context.Context) ([]int, error) {
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}

	// Only project the corporation_id field for efficiency
	projection := bson.M{"corporation_id": 1}
	findOptions := options.Find().SetProjection(projection)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var corporationIDs []int
	for cursor.Next(ctx) {
		var doc struct {
			CorporationID int `bson:"corporation_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue // Skip invalid documents
		}
		corporationIDs = append(corporationIDs, doc.CorporationID)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return corporationIDs, nil
}

// SearchCorporationsByName searches corporations by name using optimized search strategies
func (r *Repository) SearchCorporationsByName(ctx context.Context, name string) ([]*models.Corporation, error) {
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
			// Also search in ticker field for corporation ticker searches
			regexPattern := strings.ToLower(name)

			filter = bson.M{
				"$or": []bson.M{
					{
						"name": bson.M{
							"$regex":   regexPattern,
							"$options": "i", // case-insensitive
						},
					},
					{
						"ticker": bson.M{
							"$regex":   regexPattern,
							"$options": "i", // case-insensitive
						},
					},
				},
			}
			// Sort by member count (descending) for relevance and limit
			findOptions = options.Find().
				SetSort(bson.M{"member_count": -1}).
				SetLimit(50) // Limit results for performance
		}
	} else {
		// For very short queries, use prefix search only on both name and ticker
		filter = bson.M{
			"$or": []bson.M{
				{
					"name": bson.M{
						"$regex":   "^" + strings.ToLower(name),
						"$options": "i",
					},
				},
				{
					"ticker": bson.M{
						"$regex":   "^" + strings.ToLower(name),
						"$options": "i",
					},
				},
			},
		}
		findOptions = options.Find().
			SetSort(bson.M{"member_count": -1}).
			SetLimit(20) // Smaller limit for short queries
	}

	// Add soft delete filter and handle legacy ID field mapping
	filter["deleted_at"] = bson.M{"$exists": false}

	// Convert name and ticker searches to handle both corporation_id and legacy id field
	if len(name) >= 3 && !strings.Contains(name, " ") && len(strings.Fields(name)) <= 1 {
		// For single-word searches, we need to ensure both corporation_id and legacy id are available
		// But we keep the existing filter as-is for now since the search conversion will handle legacy IDs
	}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var corporations []*models.Corporation
	if err := cursor.All(ctx, &corporations); err != nil {
		return nil, err
	}

	return corporations, nil
}

// GetCEOIDsFromEnabledCorporations retrieves CEO character IDs from all enabled (non-deleted) corporations
func (r *Repository) GetCEOIDsFromEnabledCorporations(ctx context.Context) ([]int, error) {
	filter := bson.M{"deleted_at": bson.M{"$exists": false}, "ceo_character_id": bson.M{"$gt": 0}}

	// Only project the ceo_character_id field for efficiency
	projection := bson.M{"ceo_character_id": 1}
	findOptions := options.Find().SetProjection(projection)

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ceoIDs []int
	for cursor.Next(ctx) {
		var doc struct {
			CEOCharacterID int `bson:"ceo_character_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue // Skip invalid documents
		}
		ceoIDs = append(ceoIDs, doc.CEOCharacterID)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return ceoIDs, nil
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.mongodb.Client.Ping(ctx, nil)
}
