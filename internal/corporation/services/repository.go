package services

import (
	"context"
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