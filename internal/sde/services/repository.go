package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-falcon/internal/sde/models"
	"go-falcon/pkg/database"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles data access for SDE operations
type Repository struct {
	mongodb *database.MongoDB
	redis   *database.Redis
}

// NewRepository creates a new SDE repository
func NewRepository(mongodb *database.MongoDB, redis *database.Redis) *Repository {
	return &Repository{
		mongodb: mongodb,
		redis:   redis,
	}
}

// Collection names
const (
	statusCollection        = "sde_status"
	historyCollection      = "sde_update_history"
	notificationsCollection = "sde_notifications"
	configCollection       = "sde_config"
	statisticsCollection   = "sde_statistics"
	indexCollection        = "sde_indexes"
)

// GetStatus retrieves the current SDE status
func (r *Repository) GetStatus(ctx context.Context) (*models.SDEStatus, error) {
	collection := r.mongodb.Database.Collection(statusCollection)
	
	var status models.SDEStatus
	err := collection.FindOne(ctx, bson.M{}).Decode(&status)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default status if not found
			return &models.SDEStatus{
				CurrentHash:    "",
				LatestHash:     "",
				IsUpToDate:     false,
				IsProcessing:   false,
				Progress:       0.0,
				LastError:      "",
				LastCheck:      time.Time{},
				LastUpdate:     time.Time{},
				FilesProcessed: 0,
				TotalFiles:     0,
				CurrentStage:   "",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get SDE status: %w", err)
	}

	return &status, nil
}

// UpdateStatus updates the SDE status
func (r *Repository) UpdateStatus(ctx context.Context, status *models.SDEStatus) error {
	collection := r.mongodb.Database.Collection(statusCollection)
	
	status.UpdatedAt = time.Now()
	
	filter := bson.M{}
	if !status.ID.IsZero() {
		filter["_id"] = status.ID
	}

	update := bson.M{
		"$set": status,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update SDE status: %w", err)
	}

	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			status.ID = oid
		}
	}

	return nil
}

// CreateUpdateHistory creates a new update history entry
func (r *Repository) CreateUpdateHistory(ctx context.Context, history *models.SDEUpdateHistory) (primitive.ObjectID, error) {
	collection := r.mongodb.Database.Collection(historyCollection)
	
	history.CreatedAt = time.Now()
	
	result, err := collection.InsertOne(ctx, history)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to create update history: %w", err)
	}

	return result.InsertedID.(primitive.ObjectID), nil
}

// UpdateHistory updates an existing history entry
func (r *Repository) UpdateHistory(ctx context.Context, history *models.SDEUpdateHistory) error {
	collection := r.mongodb.Database.Collection(historyCollection)
	
	filter := bson.M{"_id": history.ID}
	update := bson.M{
		"$set": bson.M{
			"end_time": history.EndTime,
			"duration": history.Duration,
			"success":  history.Success,
			"error":    history.Error,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update history: %w", err)
	}

	return nil
}

// GetHistory retrieves update history with pagination
func (r *Repository) GetHistory(ctx context.Context, page, pageSize int, startTime, endTime *time.Time, success *bool) ([]models.SDEUpdateHistory, int, error) {
	collection := r.mongodb.Database.Collection(historyCollection)
	
	// Build filter
	filter := bson.M{}
	if startTime != nil {
		filter["start_time"] = bson.M{"$gte": *startTime}
	}
	if endTime != nil {
		if filter["start_time"] != nil {
			filter["start_time"].(bson.M)["$lte"] = *endTime
		} else {
			filter["start_time"] = bson.M{"$lte": *endTime}
		}
	}
	if success != nil {
		filter["success"] = *success
	}

	// Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history documents: %w", err)
	}

	// Calculate skip
	skip := (page - 1) * pageSize

	// Find documents with pagination
	opts := options.Find().
		SetSort(bson.M{"start_time": -1}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find history documents: %w", err)
	}
	defer cursor.Close(ctx)

	var history []models.SDEUpdateHistory
	if err := cursor.All(ctx, &history); err != nil {
		return nil, 0, fmt.Errorf("failed to decode history documents: %w", err)
	}

	return history, int(total), nil
}

// CreateNotification creates a new notification
func (r *Repository) CreateNotification(ctx context.Context, notification *models.SDENotification) error {
	collection := r.mongodb.Database.Collection(notificationsCollection)
	
	notification.CreatedAt = time.Now()
	
	_, err := collection.InsertOne(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// GetNotifications retrieves notifications with pagination and filtering
func (r *Repository) GetNotifications(ctx context.Context, page, pageSize int, notificationType string, isRead *bool, startTime, endTime *time.Time) ([]models.SDENotification, int, error) {
	collection := r.mongodb.Database.Collection(notificationsCollection)
	
	// Build filter
	filter := bson.M{}
	if notificationType != "" {
		filter["type"] = notificationType
	}
	if isRead != nil {
		filter["is_read"] = *isRead
	}
	if startTime != nil {
		filter["created_at"] = bson.M{"$gte": *startTime}
	}
	if endTime != nil {
		if filter["created_at"] != nil {
			filter["created_at"].(bson.M)["$lte"] = *endTime
		} else {
			filter["created_at"] = bson.M{"$lte": *endTime}
		}
	}

	// Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notification documents: %w", err)
	}

	// Calculate skip
	skip := (page - 1) * pageSize

	// Find documents with pagination
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notification documents: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []models.SDENotification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, 0, fmt.Errorf("failed to decode notification documents: %w", err)
	}

	return notifications, int(total), nil
}

// MarkNotificationsRead marks multiple notifications as read
func (r *Repository) MarkNotificationsRead(ctx context.Context, notificationIDs []primitive.ObjectID) error {
	collection := r.mongodb.Database.Collection(notificationsCollection)
	
	filter := bson.M{"_id": bson.M{"$in": notificationIDs}}
	update := bson.M{
		"$set": bson.M{
			"is_read": true,
			"read_at": time.Now(),
		},
	}

	_, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark notifications as read: %w", err)
	}

	return nil
}

// GetConfig retrieves SDE configuration
func (r *Repository) GetConfig(ctx context.Context) (*models.SDEConfig, error) {
	collection := r.mongodb.Database.Collection(configCollection)
	
	var config models.SDEConfig
	err := collection.FindOne(ctx, bson.M{}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, mongo.ErrNoDocuments
		}
		return nil, fmt.Errorf("failed to get SDE config: %w", err)
	}

	return &config, nil
}

// UpdateConfig updates SDE configuration
func (r *Repository) UpdateConfig(ctx context.Context, config *models.SDEConfig) error {
	collection := r.mongodb.Database.Collection(configCollection)
	
	config.UpdatedAt = time.Now()
	
	filter := bson.M{}
	if !config.ID.IsZero() {
		filter["_id"] = config.ID
	}

	update := bson.M{
		"$set": config,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update SDE config: %w", err)
	}

	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			config.ID = oid
		}
	}

	return nil
}

// GetStatistics retrieves SDE statistics
func (r *Repository) GetStatistics(ctx context.Context) (*models.SDEStatistics, error) {
	collection := r.mongodb.Database.Collection(statisticsCollection)
	
	var stats models.SDEStatistics
	err := collection.FindOne(ctx, bson.M{}).Decode(&stats)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default statistics if not found
			return &models.SDEStatistics{
				TotalEntities:    0,
				EntitiesByType:   make(map[string]int),
				LastUpdate:       time.Time{},
				DataSize:         0,
				IndexSize:        0,
				ProcessingStats:  models.ProcessingStats{},
				PerformanceStats: models.PerformanceStats{},
				UpdatedAt:        time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get SDE statistics: %w", err)
	}

	return &stats, nil
}

// UpdateStatistics updates SDE statistics
func (r *Repository) UpdateStatistics(ctx context.Context, stats *models.SDEStatistics) error {
	collection := r.mongodb.Database.Collection(statisticsCollection)
	
	stats.UpdatedAt = time.Now()
	
	filter := bson.M{}
	if !stats.ID.IsZero() {
		filter["_id"] = stats.ID
	}

	update := bson.M{
		"$set": stats,
		"$setOnInsert": bson.M{
			"updated_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update SDE statistics: %w", err)
	}

	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			stats.ID = oid
		}
	}

	return nil
}

// GetIndexInfo retrieves information about a specific index
func (r *Repository) GetIndexInfo(ctx context.Context, indexType string) (*models.SDEIndex, error) {
	collection := r.mongodb.Database.Collection(indexCollection)
	
	filter := bson.M{"type": indexType, "is_active": true}
	
	var index models.SDEIndex
	err := collection.FindOne(ctx, filter).Decode(&index)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Index not found
		}
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}

	return &index, nil
}

// UpdateIndexInfo updates information about a search index
func (r *Repository) UpdateIndexInfo(ctx context.Context, indexType string, itemCount int, buildTime time.Duration) error {
	collection := r.mongodb.Database.Collection(indexCollection)
	
	now := time.Now()
	redisKey := fmt.Sprintf("sde:index:%s", indexType)
	
	filter := bson.M{"type": indexType}
	update := bson.M{
		"$set": bson.M{
			"type":       indexType,
			"name":       fmt.Sprintf("%s index", strings.Title(indexType)),
			"redis_key":  redisKey,
			"item_count": itemCount,
			"last_built": now,
			"build_time": buildTime,
			"is_active":  true,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update index info: %w", err)
	}

	return nil
}

// Redis operations

// GetRedisStatus gets status from Redis
func (r *Repository) GetRedisStatus(ctx context.Context) (map[string]interface{}, error) {
	result := r.redis.Client.Get(ctx, models.RedisKeyStatus)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get Redis status: %w", result.Err())
	}

	var status map[string]interface{}
	if err := result.Scan(&status); err != nil {
		return nil, fmt.Errorf("failed to decode Redis status: %w", err)
	}

	return status, nil
}

// SetRedisStatus sets status in Redis
func (r *Repository) SetRedisStatus(ctx context.Context, status map[string]interface{}) error {
	err := r.redis.Client.Set(ctx, models.RedisKeyStatus, status, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set Redis status: %w", err)
	}

	return nil
}

// GetRedisProgress gets progress from Redis
func (r *Repository) GetRedisProgress(ctx context.Context) (*models.ProgressState, error) {
	result := r.redis.Client.Get(ctx, models.RedisKeyProgress)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get Redis progress: %w", result.Err())
	}

	var progress models.ProgressState
	if err := result.Scan(&progress); err != nil {
		return nil, fmt.Errorf("failed to decode Redis progress: %w", err)
	}

	return &progress, nil
}

// SetRedisProgress sets progress in Redis
func (r *Repository) SetRedisProgress(ctx context.Context, progress *models.ProgressState) error {
	err := r.redis.Client.Set(ctx, models.RedisKeyProgress, progress, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set Redis progress: %w", err)
	}

	return nil
}

// GetCurrentHash gets the current SDE hash from Redis
func (r *Repository) GetCurrentHash(ctx context.Context) (string, error) {
	result := r.redis.Client.Get(ctx, models.RedisKeyCurrentHash)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to get current hash: %w", result.Err())
	}

	return result.Val(), nil
}

// SetCurrentHash sets the current SDE hash in Redis
func (r *Repository) SetCurrentHash(ctx context.Context, hash string) error {
	err := r.redis.Client.Set(ctx, models.RedisKeyCurrentHash, hash, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set current hash: %w", err)
	}

	return nil
}

// DeleteSDEKeys removes all SDE-related keys from Redis
func (r *Repository) DeleteSDEKeys(ctx context.Context, pattern string) error {
	// Get all keys matching the pattern
	keys, err := r.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get SDE keys: %w", err)
	}

	if len(keys) == 0 {
		return nil // No keys to delete
	}

	// Delete keys in batches to avoid blocking Redis
	batchSize := 1000
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		if err := r.redis.Client.Del(ctx, batch...).Err(); err != nil {
			return fmt.Errorf("failed to delete SDE keys batch: %w", err)
		}
	}

	return nil
}

// GetRedisKeysByPattern gets keys matching a pattern
func (r *Repository) GetRedisKeysByPattern(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis keys: %w", err)
	}

	return keys, nil
}

// CountRedisKeysByPattern counts keys matching a pattern
func (r *Repository) CountRedisKeysByPattern(ctx context.Context, pattern string) (int, error) {
	keys, err := r.GetRedisKeysByPattern(ctx, pattern)
	if err != nil {
		return 0, err
	}

	return len(keys), nil
}

// GetRedisMemoryUsage gets memory usage for a specific key
func (r *Repository) GetRedisMemoryUsage(ctx context.Context, key string) (int64, error) {
	result := r.redis.Client.MemoryUsage(ctx, key)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get memory usage for key %s: %w", key, result.Err())
	}

	return result.Val(), nil
}

// SetRedisEntity stores an entity in Redis
func (r *Repository) SetRedisEntity(ctx context.Context, key string, data interface{}) error {
	err := r.redis.Client.Set(ctx, key, data, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set Redis entity %s: %w", key, err)
	}

	return nil
}

// GetRedisEntity retrieves an entity from Redis
func (r *Repository) GetRedisEntity(ctx context.Context, key string) (interface{}, error) {
	result := r.redis.Client.Get(ctx, key)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get Redis entity %s: %w", key, result.Err())
	}

	var data interface{}
	if err := result.Scan(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Redis entity %s: %w", key, err)
	}

	return data, nil
}

// SetSearchIndex stores a search index in Redis
func (r *Repository) SetSearchIndex(ctx context.Context, indexKey string, index map[string]string) error {
	// Use HSET to store the index as a hash
	pipe := r.redis.Client.Pipeline()
	
	// Clear existing index
	pipe.Del(ctx, indexKey)
	
	// Set new index entries
	if len(index) > 0 {
		args := make([]interface{}, 0, len(index)*2)
		for field, value := range index {
			args = append(args, field, value)
		}
		pipe.HSet(ctx, indexKey, args...)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set search index %s: %w", indexKey, err)
	}

	return nil
}

// SearchIndex performs a search using Redis hash index
func (r *Repository) SearchIndex(ctx context.Context, indexKey, pattern string) (map[string]string, error) {
	// For exact match, use HGET
	if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") {
		result := r.redis.Client.HGet(ctx, indexKey, strings.ToLower(pattern))
		if result.Err() != nil {
			if result.Err() == redis.Nil {
				return make(map[string]string), nil
			}
			return nil, fmt.Errorf("failed to search index %s: %w", indexKey, result.Err())
		}
		
		matches := make(map[string]string)
		matches[strings.ToLower(pattern)] = result.Val()
		return matches, nil
	}

	// For pattern matching, get all and filter
	allFields, err := r.redis.Client.HGetAll(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all index entries for %s: %w", indexKey, err)
	}

	matches := make(map[string]string)
	lowerPattern := strings.ToLower(pattern)
	
	for field, value := range allFields {
		if strings.Contains(field, lowerPattern) {
			matches[field] = value
		}
	}

	return matches, nil
}