package services

import (
	"context"
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

// CreateIndexes creates necessary database indexes for the alliances collection
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "alliance_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	
	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}