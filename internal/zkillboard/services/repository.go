package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/zkillboard/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for ZKillboard data
type Repository struct {
	db                      *mongo.Database
	zkbMetadataCollection   *mongo.Collection
	timeseriesCollection    *mongo.Collection
	consumerStateCollection *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		db:                      db,
		zkbMetadataCollection:   db.Collection("zkb_metadata"),
		timeseriesCollection:    db.Collection("killmail_timeseries"),
		consumerStateCollection: db.Collection("zkb_consumer_state"),
	}
}

// CreateIndexes creates necessary database indexes
func (r *Repository) CreateIndexes(ctx context.Context) error {
	// ZKB metadata indexes
	zkbIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "killmail_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "processed_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "total_value", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "points", Value: -1}},
		},
		{
			Keys: bson.D{
				{Key: "solo", Value: 1},
				{Key: "npc", Value: 1},
			},
		},
	}

	if _, err := r.zkbMetadataCollection.Indexes().CreateMany(ctx, zkbIndexes); err != nil {
		return fmt.Errorf("failed to create zkb_metadata indexes: %w", err)
	}

	// Timeseries indexes
	timeseriesIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "period", Value: 1},
				{Key: "timestamp", Value: -1},
			},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys: bson.D{
				{Key: "solar_system_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "region_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "alliance_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "corporation_id", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		// TTL index for automatic cleanup (90 days default)
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60),
		},
	}

	if _, err := r.timeseriesCollection.Indexes().CreateMany(ctx, timeseriesIndexes); err != nil {
		return fmt.Errorf("failed to create timeseries indexes: %w", err)
	}

	// Consumer state indexes
	stateIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "queue_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	}

	if _, err := r.consumerStateCollection.Indexes().CreateMany(ctx, stateIndexes); err != nil {
		return fmt.Errorf("failed to create consumer state indexes: %w", err)
	}

	return nil
}

// SaveZKBMetadata saves ZKillboard metadata for a single killmail
func (r *Repository) SaveZKBMetadata(ctx context.Context, metadata *models.ZKBMetadata) error {
	metadata.UpdatedAt = time.Now()

	filter := bson.M{"killmail_id": metadata.KillmailID}
	update := bson.M{"$set": metadata}
	opts := options.Update().SetUpsert(true)

	_, err := r.zkbMetadataCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save ZKB metadata: %w", err)
	}

	return nil
}

// SaveZKBMetadataBatch saves multiple ZKB metadata entries
func (r *Repository) SaveZKBMetadataBatch(ctx context.Context, metadataList []*models.ZKBMetadata) error {
	if len(metadataList) == 0 {
		return nil
	}

	// Prepare bulk write operations
	writeModels := make([]mongo.WriteModel, len(metadataList))
	for i, metadata := range metadataList {
		metadata.UpdatedAt = time.Now()

		filter := bson.M{"killmail_id": metadata.KillmailID}
		update := bson.M{"$set": metadata}

		writeModels[i] = mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err := r.zkbMetadataCollection.BulkWrite(ctx, writeModels, opts)
	if err != nil {
		return fmt.Errorf("failed to batch save ZKB metadata: %w", err)
	}

	return nil
}

// GetZKBMetadata retrieves ZKB metadata for a killmail
func (r *Repository) GetZKBMetadata(ctx context.Context, killmailID int64) (*models.ZKBMetadata, error) {
	var metadata models.ZKBMetadata

	filter := bson.M{"killmail_id": killmailID}
	err := r.zkbMetadataCollection.FindOne(ctx, filter).Decode(&metadata)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get ZKB metadata: %w", err)
	}

	return &metadata, nil
}

// SaveTimeseries saves or updates timeseries data
func (r *Repository) SaveTimeseries(ctx context.Context, timeseries *models.KillmailTimeseries) error {
	timeseries.UpdatedAt = time.Now()

	// Create unique key based on period and dimensions
	filter := bson.M{
		"period":    timeseries.Period,
		"timestamp": timeseries.Timestamp,
	}

	// Add optional dimension filters
	if timeseries.SolarSystemID != 0 {
		filter["solar_system_id"] = timeseries.SolarSystemID
	}
	if timeseries.RegionID != 0 {
		filter["region_id"] = timeseries.RegionID
	}
	if timeseries.AllianceID != 0 {
		filter["alliance_id"] = timeseries.AllianceID
	}
	if timeseries.CorporationID != 0 {
		filter["corporation_id"] = timeseries.CorporationID
	}
	if timeseries.ShipTypeID != 0 {
		filter["ship_type_id"] = timeseries.ShipTypeID
	}

	update := bson.M{"$set": timeseries}
	opts := options.Update().SetUpsert(true)

	_, err := r.timeseriesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save timeseries: %w", err)
	}

	return nil
}

// IncrementTimeseries atomically increments timeseries counters
func (r *Repository) IncrementTimeseries(ctx context.Context, filter bson.M, increments bson.M) error {
	update := bson.M{
		"$inc": increments,
		"$set": bson.M{"updated_at": time.Now()},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.timeseriesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to increment timeseries: %w", err)
	}

	return nil
}

// GetTimeseries retrieves timeseries data based on filters
func (r *Repository) GetTimeseries(ctx context.Context, period string, start, end time.Time, filter bson.M) ([]*models.KillmailTimeseries, error) {
	// Build query
	query := bson.M{
		"period": period,
		"timestamp": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	// Merge additional filters
	for k, v := range filter {
		query[k] = v
	}

	// Execute query
	cursor, err := r.timeseriesCollection.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query timeseries: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*models.KillmailTimeseries
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode timeseries: %w", err)
	}

	return results, nil
}

// SaveConsumerState saves the consumer state
func (r *Repository) SaveConsumerState(ctx context.Context, state *models.ConsumerState) error {
	state.UpdatedAt = time.Now()

	filter := bson.M{"queue_id": state.QueueID}

	// If no ID, this is a new state
	if state.ID.IsZero() {
		state.ID = primitive.NewObjectID()
	}

	update := bson.M{"$set": state}
	opts := options.Update().SetUpsert(true)

	_, err := r.consumerStateCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save consumer state: %w", err)
	}

	return nil
}

// GetConsumerState retrieves the consumer state by queue ID
func (r *Repository) GetConsumerState(ctx context.Context, queueID string) (*models.ConsumerState, error) {
	var state models.ConsumerState

	filter := bson.M{"queue_id": queueID}
	err := r.consumerStateCollection.FindOne(ctx, filter).Decode(&state)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get consumer state: %w", err)
	}

	return &state, nil
}

// GetLatestConsumerState retrieves the most recent consumer state
func (r *Repository) GetLatestConsumerState(ctx context.Context) (*models.ConsumerState, error) {
	var state models.ConsumerState

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	err := r.consumerStateCollection.FindOne(ctx, bson.M{}, opts).Decode(&state)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest consumer state: %w", err)
	}

	return &state, nil
}

// GetRecentKillmails retrieves recent killmails with ZKB metadata
func (r *Repository) GetRecentKillmails(ctx context.Context, limit int) ([]bson.M, error) {
	// Aggregation pipeline to join killmails with ZKB metadata
	pipeline := []bson.M{
		{
			"$sort": bson.M{"processed_at": -1},
		},
		{
			"$limit": limit,
		},
		{
			"$lookup": bson.M{
				"from":         "killmails",
				"localField":   "killmail_id",
				"foreignField": "killmail_id",
				"as":           "killmail",
			},
		},
		{
			"$unwind": "$killmail",
		},
		{
			"$project": bson.M{
				"killmail_id":     1,
				"timestamp":       "$killmail.killmail_time",
				"solar_system_id": "$killmail.solar_system_id",
				"victim":          "$killmail.victim",
				"total_value":     1,
				"points":          1,
				"solo":            1,
				"npc":             1,
				"href":            1,
			},
		},
	}

	cursor, err := r.zkbMetadataCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent killmails: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode recent killmails: %w", err)
	}

	return results, nil
}

// GetStats retrieves killmail statistics for a period
func (r *Repository) GetStats(ctx context.Context, period string, start, end time.Time) (bson.M, error) {
	// Aggregation pipeline for statistics
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"processed_at": bson.M{
					"$gte": start,
					"$lte": end,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":             nil,
				"total_killmails": bson.M{"$sum": 1},
				"total_value":     bson.M{"$sum": "$total_value"},
				"npc_kills": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{"$npc", 1, 0},
					},
				},
				"solo_kills": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{"$solo", 1, 0},
					},
				},
				"avg_value": bson.M{"$avg": "$total_value"},
				"max_value": bson.M{"$max": "$total_value"},
			},
		},
	}

	cursor, err := r.zkbMetadataCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	if len(results) == 0 {
		return bson.M{
			"total_killmails": 0,
			"total_value":     0,
			"npc_kills":       0,
			"solo_kills":      0,
			"avg_value":       0,
			"max_value":       0,
		}, nil
	}

	return results[0], nil
}
