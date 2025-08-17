package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/dev/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles data access for Dev operations
type Repository struct {
	mongodb *database.MongoDB
	redis   *database.Redis
}

// NewRepository creates a new Dev repository
func NewRepository(mongodb *database.MongoDB, redis *database.Redis) *Repository {
	return &Repository{
		mongodb: mongodb,
		redis:   redis,
	}
}

// Test execution operations

// CreateTestExecution creates a new test execution record
func (r *Repository) CreateTestExecution(ctx context.Context, execution *models.TestExecution) error {
	collection := r.mongodb.Database.Collection(models.TestExecutionsCollection)
	
	execution.CreatedAt = time.Now()
	execution.UpdatedAt = time.Now()
	
	result, err := collection.InsertOne(ctx, execution)
	if err != nil {
		return fmt.Errorf("failed to create test execution: %w", err)
	}
	
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		execution.ID = oid
	}
	
	return nil
}

// UpdateTestExecution updates an existing test execution
func (r *Repository) UpdateTestExecution(ctx context.Context, execution *models.TestExecution) error {
	collection := r.mongodb.Database.Collection(models.TestExecutionsCollection)
	
	execution.UpdatedAt = time.Now()
	
	filter := bson.M{"_id": execution.ID}
	update := bson.M{"$set": execution}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update test execution: %w", err)
	}
	
	return nil
}

// GetTestExecution retrieves a test execution by ID
func (r *Repository) GetTestExecution(ctx context.Context, id primitive.ObjectID) (*models.TestExecution, error) {
	collection := r.mongodb.Database.Collection(models.TestExecutionsCollection)
	
	var execution models.TestExecution
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&execution)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("test execution not found")
		}
		return nil, fmt.Errorf("failed to get test execution: %w", err)
	}
	
	return &execution, nil
}

// ListTestExecutions retrieves test executions with pagination
func (r *Repository) ListTestExecutions(ctx context.Context, userID int, limit, offset int) ([]models.TestExecution, int, error) {
	collection := r.mongodb.Database.Collection(models.TestExecutionsCollection)
	
	filter := bson.M{}
	if userID > 0 {
		filter["user_id"] = userID
	}
	
	// Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count test executions: %w", err)
	}
	
	// Find documents with pagination
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find test executions: %w", err)
	}
	defer cursor.Close(ctx)
	
	var executions []models.TestExecution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, 0, fmt.Errorf("failed to decode test executions: %w", err)
	}
	
	return executions, int(total), nil
}

// Cache metrics operations

// UpsertCacheMetrics inserts or updates cache metrics
func (r *Repository) UpsertCacheMetrics(ctx context.Context, metrics *models.CacheMetrics) error {
	collection := r.mongodb.Database.Collection(models.CacheMetricsCollection)
	
	metrics.Timestamp = time.Now()
	
	filter := bson.M{"cache_type": metrics.CacheType}
	update := bson.M{
		"$set": metrics,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert cache metrics: %w", err)
	}
	
	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			metrics.ID = oid
		}
	}
	
	return nil
}

// GetCacheMetrics retrieves cache metrics by type
func (r *Repository) GetCacheMetrics(ctx context.Context, cacheType string) (*models.CacheMetrics, error) {
	collection := r.mongodb.Database.Collection(models.CacheMetricsCollection)
	
	var metrics models.CacheMetrics
	err := collection.FindOne(ctx, bson.M{"cache_type": cacheType}).Decode(&metrics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cache metrics: %w", err)
	}
	
	return &metrics, nil
}

// ESI metrics operations

// UpsertESIMetrics inserts or updates ESI metrics
func (r *Repository) UpsertESIMetrics(ctx context.Context, metrics *models.ESIMetrics) error {
	collection := r.mongodb.Database.Collection(models.ESIMetricsCollection)
	
	metrics.Timestamp = time.Now()
	
	filter := bson.M{"endpoint": metrics.Endpoint}
	update := bson.M{
		"$set": metrics,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert ESI metrics: %w", err)
	}
	
	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			metrics.ID = oid
		}
	}
	
	return nil
}

// GetESIMetrics retrieves ESI metrics by endpoint
func (r *Repository) GetESIMetrics(ctx context.Context, endpoint string) (*models.ESIMetrics, error) {
	collection := r.mongodb.Database.Collection(models.ESIMetricsCollection)
	
	var metrics models.ESIMetrics
	err := collection.FindOne(ctx, bson.M{"endpoint": endpoint}).Decode(&metrics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get ESI metrics: %w", err)
	}
	
	return &metrics, nil
}

// ListESIMetrics retrieves all ESI metrics
func (r *Repository) ListESIMetrics(ctx context.Context) ([]models.ESIMetrics, error) {
	collection := r.mongodb.Database.Collection(models.ESIMetricsCollection)
	
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find ESI metrics: %w", err)
	}
	defer cursor.Close(ctx)
	
	var metrics []models.ESIMetrics
	if err := cursor.All(ctx, &metrics); err != nil {
		return nil, fmt.Errorf("failed to decode ESI metrics: %w", err)
	}
	
	return metrics, nil
}

// Performance test operations

// CreatePerformanceTest creates a new performance test
func (r *Repository) CreatePerformanceTest(ctx context.Context, test *models.PerformanceTest) error {
	collection := r.mongodb.Database.Collection(models.PerformanceTestsCollection)
	
	test.CreatedAt = time.Now()
	test.UpdatedAt = time.Now()
	
	result, err := collection.InsertOne(ctx, test)
	if err != nil {
		return fmt.Errorf("failed to create performance test: %w", err)
	}
	
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		test.ID = oid
	}
	
	return nil
}

// GetPerformanceTest retrieves a performance test by ID
func (r *Repository) GetPerformanceTest(ctx context.Context, id primitive.ObjectID) (*models.PerformanceTest, error) {
	collection := r.mongodb.Database.Collection(models.PerformanceTestsCollection)
	
	var test models.PerformanceTest
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&test)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("performance test not found")
		}
		return nil, fmt.Errorf("failed to get performance test: %w", err)
	}
	
	return &test, nil
}

// ListPerformanceTests retrieves performance tests
func (r *Repository) ListPerformanceTests(ctx context.Context, activeOnly bool) ([]models.PerformanceTest, error) {
	collection := r.mongodb.Database.Collection(models.PerformanceTestsCollection)
	
	filter := bson.M{}
	if activeOnly {
		filter["is_active"] = true
	}
	
	opts := options.Find().SetSort(bson.M{"created_at": -1})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find performance tests: %w", err)
	}
	defer cursor.Close(ctx)
	
	var tests []models.PerformanceTest
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, fmt.Errorf("failed to decode performance tests: %w", err)
	}
	
	return tests, nil
}

// Test result operations

// CreateTestResult creates a new test result
func (r *Repository) CreateTestResult(ctx context.Context, result *models.TestResult) error {
	collection := r.mongodb.Database.Collection(models.TestResultsCollection)
	
	result.CreatedAt = time.Now()
	
	insertResult, err := collection.InsertOne(ctx, result)
	if err != nil {
		return fmt.Errorf("failed to create test result: %w", err)
	}
	
	if oid, ok := insertResult.InsertedID.(primitive.ObjectID); ok {
		result.ID = oid
	}
	
	return nil
}

// GetTestResults retrieves test results for a test
func (r *Repository) GetTestResults(ctx context.Context, testID primitive.ObjectID, limit int) ([]models.TestResult, error) {
	collection := r.mongodb.Database.Collection(models.TestResultsCollection)
	
	filter := bson.M{"test_id": testID}
	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetLimit(int64(limit))
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find test results: %w", err)
	}
	defer cursor.Close(ctx)
	
	var results []models.TestResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode test results: %w", err)
	}
	
	return results, nil
}

// Debug session operations

// CreateDebugSession creates a new debug session
func (r *Repository) CreateDebugSession(ctx context.Context, session *models.DebugSession) error {
	collection := r.mongodb.Database.Collection(models.DebugSessionsCollection)
	
	session.CreatedAt = time.Now()
	session.UpdatedAt = time.Now()
	
	result, err := collection.InsertOne(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to create debug session: %w", err)
	}
	
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		session.ID = oid
	}
	
	return nil
}

// UpdateDebugSession updates a debug session
func (r *Repository) UpdateDebugSession(ctx context.Context, session *models.DebugSession) error {
	collection := r.mongodb.Database.Collection(models.DebugSessionsCollection)
	
	session.UpdatedAt = time.Now()
	
	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": session}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update debug session: %w", err)
	}
	
	return nil
}

// GetDebugSession retrieves a debug session by ID
func (r *Repository) GetDebugSession(ctx context.Context, id primitive.ObjectID) (*models.DebugSession, error) {
	collection := r.mongodb.Database.Collection(models.DebugSessionsCollection)
	
	var session models.DebugSession
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("debug session not found")
		}
		return nil, fmt.Errorf("failed to get debug session: %w", err)
	}
	
	return &session, nil
}

// Component status operations

// UpsertComponentStatus inserts or updates component status
func (r *Repository) UpsertComponentStatus(ctx context.Context, status *models.ComponentStatus) error {
	collection := r.mongodb.Database.Collection(models.ComponentStatusCollection)
	
	status.UpdatedAt = time.Now()
	
	filter := bson.M{"component": status.Component}
	update := bson.M{
		"$set": status,
		"$setOnInsert": bson.M{
			"created_at": time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert component status: %w", err)
	}
	
	if result.UpsertedID != nil {
		if oid, ok := result.UpsertedID.(primitive.ObjectID); ok {
			status.ID = oid
		}
	}
	
	return nil
}

// GetComponentStatus retrieves component status
func (r *Repository) GetComponentStatus(ctx context.Context, component string) (*models.ComponentStatus, error) {
	collection := r.mongodb.Database.Collection(models.ComponentStatusCollection)
	
	var status models.ComponentStatus
	err := collection.FindOne(ctx, bson.M{"component": component}).Decode(&status)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get component status: %w", err)
	}
	
	return &status, nil
}

// ListComponentStatuses retrieves all component statuses
func (r *Repository) ListComponentStatuses(ctx context.Context) ([]models.ComponentStatus, error) {
	collection := r.mongodb.Database.Collection(models.ComponentStatusCollection)
	
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find component statuses: %w", err)
	}
	defer cursor.Close(ctx)
	
	var statuses []models.ComponentStatus
	if err := cursor.All(ctx, &statuses); err != nil {
		return nil, fmt.Errorf("failed to decode component statuses: %w", err)
	}
	
	return statuses, nil
}

// Mock data template operations

// CreateMockDataTemplate creates a new mock data template
func (r *Repository) CreateMockDataTemplate(ctx context.Context, template *models.MockDataTemplate) error {
	collection := r.mongodb.Database.Collection(models.MockDataTemplatesCollection)
	
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	
	result, err := collection.InsertOne(ctx, template)
	if err != nil {
		return fmt.Errorf("failed to create mock data template: %w", err)
	}
	
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		template.ID = oid
	}
	
	return nil
}

// GetMockDataTemplate retrieves a mock data template by data type
func (r *Repository) GetMockDataTemplate(ctx context.Context, dataType string) (*models.MockDataTemplate, error) {
	collection := r.mongodb.Database.Collection(models.MockDataTemplatesCollection)
	
	var template models.MockDataTemplate
	err := collection.FindOne(ctx, bson.M{"data_type": dataType, "is_active": true}).Decode(&template)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get mock data template: %w", err)
	}
	
	return &template, nil
}

// Redis cache operations

// SetCache stores a value in Redis cache
func (r *Repository) SetCache(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	err := r.redis.Client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

// GetCache retrieves a value from Redis cache
func (r *Repository) GetCache(ctx context.Context, key string) (string, error) {
	result := r.redis.Client.Get(ctx, key)
	if result.Err() != nil {
		return "", result.Err()
	}
	return result.Val(), nil
}

// DeleteCache deletes a value from Redis cache
func (r *Repository) DeleteCache(ctx context.Context, key string) error {
	err := r.redis.Client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

// GetCacheStats retrieves cache statistics from Redis
func (r *Repository) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info := r.redis.Client.Info(ctx, "memory", "stats")
	if info.Err() != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", info.Err())
	}
	
	// Parse Redis info output into structured data
	stats := make(map[string]interface{})
	stats["redis_info"] = info.Val()
	
	// Get keyspace info
	keyspace := r.redis.Client.Info(ctx, "keyspace")
	if keyspace.Err() == nil {
		stats["keyspace"] = keyspace.Val()
	}
	
	return stats, nil
}