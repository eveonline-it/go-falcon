package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/site_settings/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for site settings
type Repository struct {
	collection *mongo.Collection
	db         *mongo.Database
}

// NewRepository creates a new repository instance
func NewRepository(db *mongo.Database) *Repository {
	collection := db.Collection(models.SiteSettingsCollection)
	return &Repository{
		collection: collection,
		db:         db,
	}
}

// CreateIndexes creates necessary database indexes
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_public", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}, {Key: "is_public", Value: 1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// Create creates a new site setting
func (r *Repository) Create(ctx context.Context, setting *models.SiteSetting) error {
	setting.CreatedAt = time.Now()
	setting.UpdatedAt = setting.CreatedAt
	
	result, err := r.collection.InsertOne(ctx, setting)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("setting key '%s' already exists", setting.Key)
		}
		return fmt.Errorf("failed to create setting: %w", err)
	}
	
	setting.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByKey retrieves a setting by its key
func (r *Repository) GetByKey(ctx context.Context, key string) (*models.SiteSetting, error) {
	var setting models.SiteSetting
	err := r.collection.FindOne(ctx, bson.M{"key": key}).Decode(&setting)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("setting with key '%s' not found", key)
		}
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}
	return &setting, nil
}

// Update updates an existing site setting
func (r *Repository) Update(ctx context.Context, key string, updates bson.M, updatedBy int64) (*models.SiteSetting, error) {
	updates["updated_at"] = time.Now()
	updates["updated_by"] = updatedBy
	
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var setting models.SiteSetting
	
	err := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"key": key},
		bson.M{"$set": updates},
		opts,
	).Decode(&setting)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("setting with key '%s' not found", key)
		}
		return nil, fmt.Errorf("failed to update setting: %w", err)
	}
	
	return &setting, nil
}

// Delete deletes a site setting by key
func (r *Repository) Delete(ctx context.Context, key string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"key": key})
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("setting with key '%s' not found", key)
	}
	
	return nil
}

// ListSettings returns a paginated list of settings with optional filters
func (r *Repository) ListSettings(ctx context.Context, category string, isPublic *bool, isActive *bool, page, limit int) ([]*models.SiteSetting, int, error) {
	// Build filter
	filter := bson.M{}
	if category != "" {
		filter["category"] = category
	}
	if isPublic != nil {
		filter["is_public"] = *isPublic
	}
	if isActive != nil {
		filter["is_active"] = *isActive
	}

	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count settings: %w", err)
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find with pagination
	opts := options.Find().
		SetSort(bson.D{{Key: "category", Value: 1}, {Key: "key", Value: 1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find settings: %w", err)
	}
	defer cursor.Close(ctx)

	var settings []*models.SiteSetting
	for cursor.Next(ctx) {
		var setting models.SiteSetting
		if err := cursor.Decode(&setting); err != nil {
			return nil, 0, fmt.Errorf("failed to decode setting: %w", err)
		}
		settings = append(settings, &setting)
	}

	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}

	return settings, int(total), nil
}

// GetPublicSettings returns all public settings
func (r *Repository) GetPublicSettings(ctx context.Context, category string, page, limit int) ([]*models.SiteSetting, int, error) {
	isPublic := true
	isActive := true
	return r.ListSettings(ctx, category, &isPublic, &isActive, page, limit)
}

// InitializeDefaults creates default settings if they don't exist
func (r *Repository) InitializeDefaults(ctx context.Context) error {
	for _, defaultSetting := range models.DefaultSiteSettings {
		// Check if setting already exists
		_, err := r.GetByKey(ctx, defaultSetting.Key)
		if err == nil {
			// Setting already exists, skip
			continue
		}

		// Create default setting
		setting := defaultSetting
		setting.CreatedAt = time.Now()
		setting.UpdatedAt = setting.CreatedAt

		_, err = r.collection.InsertOne(ctx, &setting)
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to initialize default setting '%s': %w", setting.Key, err)
		}
	}

	return nil
}

// SettingExists checks if a setting with the given key exists
func (r *Repository) SettingExists(ctx context.Context, key string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"key": key})
	if err != nil {
		return false, fmt.Errorf("failed to check if setting exists: %w", err)
	}
	return count > 0, nil
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.db.Client().Ping(ctx, nil)
}

// List retrieves settings with optional filters and pagination
func (r *Repository) List(ctx context.Context, category string, isPublic, isActive *bool, characterID *int64, page, limit int) ([]*models.SiteSetting, error) {
	filter := bson.M{}
	
	if category != "" {
		filter["category"] = category
	}
	if isPublic != nil {
		filter["is_public"] = *isPublic
	}
	if isActive != nil {
		filter["is_active"] = *isActive
	}

	// Calculate skip based on page
	skip := (page - 1) * limit
	
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "key", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	defer cursor.Close(ctx)

	var settings []*models.SiteSetting
	if err := cursor.All(ctx, &settings); err != nil {
		return nil, fmt.Errorf("failed to decode settings: %w", err)
	}

	return settings, nil
}

// GetAllianceTicker fetches alliance ticker from the alliances collection
func (r *Repository) GetAllianceTicker(ctx context.Context, allianceID int64) (string, error) {
	alliancesCollection := r.db.Collection("alliances")
	
	var result struct {
		Ticker string `bson:"ticker"`
	}
	
	err := alliancesCollection.FindOne(ctx, bson.M{"alliance_id": allianceID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil // Alliance not found
		}
		return "", err
	}
	
	return result.Ticker, nil
}