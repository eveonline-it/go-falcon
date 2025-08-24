package services

import (
	"context"
	"strings"
	"time"

	"go-falcon/internal/alliance/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for alliances
type Repository struct {
	mongodb    *database.MongoDB
	collection *mongo.Collection
}

// NewRepository creates a new alliance repository
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:    mongodb,
		collection: mongodb.Database.Collection(models.AllianceCollection),
	}
}

// GetAllianceByID retrieves an alliance by its ID from the database
func (r *Repository) GetAllianceByID(ctx context.Context, allianceID int) (*models.Alliance, error) {
	var alliance models.Alliance
	filter := bson.M{"alliance_id": allianceID, "deleted_at": bson.M{"$exists": false}}

	err := r.collection.FindOne(ctx, filter).Decode(&alliance)
	if err != nil {
		return nil, err
	}

	return &alliance, nil
}

// CreateAlliance creates a new alliance record in the database
func (r *Repository) CreateAlliance(ctx context.Context, alliance *models.Alliance) error {
	alliance.CreatedAt = time.Now().UTC()
	alliance.UpdatedAt = time.Now().UTC()

	_, err := r.collection.InsertOne(ctx, alliance)
	return err
}

// UpdateAlliance updates an existing alliance record
func (r *Repository) UpdateAlliance(ctx context.Context, alliance *models.Alliance) error {
	alliance.UpdatedAt = time.Now().UTC()

	filter := bson.M{"alliance_id": alliance.AllianceID, "deleted_at": bson.M{"$exists": false}}
	update := bson.M{"$set": alliance}

	_, err := r.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// SearchAlliancesByName searches alliances by name using optimized search strategies
func (r *Repository) SearchAlliancesByName(ctx context.Context, name string) ([]*models.Alliance, error) {
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
			// Also search in ticker field for alliance ticker searches
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
			// Sort by date founded (descending) for relevance and limit
			findOptions = options.Find().
				SetSort(bson.M{"date_founded": -1}).
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
			SetSort(bson.M{"date_founded": -1}).
			SetLimit(20) // Smaller limit for short queries
	}

	// Add soft delete filter
	filter["deleted_at"] = bson.M{"$exists": false}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alliances []*models.Alliance
	if err := cursor.All(ctx, &alliances); err != nil {
		return nil, err
	}

	return alliances, nil
}

// CreateIndexes creates necessary database indexes for the alliances collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "alliance_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.mongodb.Client.Ping(ctx, nil)
}
