package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/market/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository provides database operations for market data
type Repository struct {
	mongodb          *database.MongoDB
	ordersCollection *mongo.Collection
	statusCollection *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:          mongodb,
		ordersCollection: mongodb.Database.Collection("market_orders"),
		statusCollection: mongodb.Database.Collection("market_fetch_status"),
	}
}

// Market Orders Operations

// GetOrdersByLocation retrieves market orders for a specific location
func (r *Repository) GetOrdersByLocation(ctx context.Context, locationID int64, typeID *int, orderType string, page, limit int) ([]models.MarketOrder, int64, error) {
	filter := bson.M{"location_id": locationID}

	if typeID != nil {
		filter["type_id"] = *typeID
	}

	if orderType != "all" {
		filter["is_buy_order"] = orderType == "buy"
	}

	return r.getOrdersWithPagination(ctx, filter, page, limit)
}

// GetOrdersByRegion retrieves market orders for a specific region
func (r *Repository) GetOrdersByRegion(ctx context.Context, regionID int, typeID *int, orderType string, page, limit int) ([]models.MarketOrder, int64, error) {
	filter := bson.M{"region_id": regionID}

	if typeID != nil {
		filter["type_id"] = *typeID
	}

	if orderType != "all" {
		filter["is_buy_order"] = orderType == "buy"
	}

	return r.getOrdersWithPagination(ctx, filter, page, limit)
}

// GetOrdersByType retrieves market orders for a specific item type
func (r *Repository) GetOrdersByType(ctx context.Context, typeID int, regionID *int, orderType string, page, limit int) ([]models.MarketOrder, int64, error) {
	filter := bson.M{"type_id": typeID}

	if regionID != nil {
		filter["region_id"] = *regionID
	}

	if orderType != "all" {
		filter["is_buy_order"] = orderType == "buy"
	}

	return r.getOrdersWithPagination(ctx, filter, page, limit)
}

// SearchOrders performs advanced search on market orders
func (r *Repository) SearchOrders(ctx context.Context, filter bson.M, sortBy string, sortOrder int, page, limit int) ([]models.MarketOrder, int64, error) {
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))

	// Set sort order
	sortMap := bson.M{}
	if sortBy != "" {
		sortMap[sortBy] = sortOrder
	}
	opts.SetSort(sortMap)

	cursor, err := r.ordersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find orders: %w", err)
	}
	defer cursor.Close(ctx)

	var orders []models.MarketOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, fmt.Errorf("failed to decode orders: %w", err)
	}

	// Get total count
	total, err := r.ordersCollection.CountDocuments(ctx, filter)
	if err != nil {
		return orders, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	return orders, total, nil
}

// getOrdersWithPagination is a helper method for paginated queries
func (r *Repository) getOrdersWithPagination(ctx context.Context, filter bson.M, page, limit int) ([]models.MarketOrder, int64, error) {
	opts := options.Find()
	opts.SetSkip(int64((page - 1) * limit))
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.M{"price": 1}) // Default sort by price

	cursor, err := r.ordersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find orders: %w", err)
	}
	defer cursor.Close(ctx)

	var orders []models.MarketOrder
	if err := cursor.All(ctx, &orders); err != nil {
		return nil, 0, fmt.Errorf("failed to decode orders: %w", err)
	}

	// Get total count
	total, err := r.ordersCollection.CountDocuments(ctx, filter)
	if err != nil {
		return orders, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	return orders, total, nil
}

// GetRegionSummary retrieves summary statistics for a region
func (r *Repository) GetRegionSummary(ctx context.Context, regionID int) (*models.MarketRegionSummary, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"region_id": regionID}},
		{"$group": bson.M{
			"_id":          "$region_id",
			"total_orders": bson.M{"$sum": 1},
			"buy_orders":   bson.M{"$sum": bson.M{"$cond": bson.A{"$is_buy_order", 1, 0}}},
			"sell_orders":  bson.M{"$sum": bson.M{"$cond": bson.A{"$is_buy_order", 0, 1}}},
			"unique_types": bson.M{"$addToSet": "$type_id"},
			"last_updated": bson.M{"$max": "$fetched_at"},
		}},
		{"$addFields": bson.M{
			"unique_types": bson.M{"$size": "$unique_types"},
		}},
	}

	cursor, err := r.ordersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate region summary: %w", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No data found
	}

	data := result[0]
	summary := &models.MarketRegionSummary{
		RegionID:    regionID,
		TotalOrders: int(data["total_orders"].(int32)),
		BuyOrders:   int(data["buy_orders"].(int32)),
		SellOrders:  int(data["sell_orders"].(int32)),
		UniqueTypes: int(data["unique_types"].(int32)),
		LastUpdated: data["last_updated"].(primitive.DateTime).Time(),
	}

	return summary, nil
}

// BulkUpsertOrders performs bulk upsert of market orders
func (r *Repository) BulkUpsertOrders(ctx context.Context, collectionName string, orders []models.MarketOrder) error {
	if len(orders) == 0 {
		return nil
	}

	collection := r.mongodb.Database.Collection(collectionName)

	var operations []mongo.WriteModel
	for _, order := range orders {
		now := time.Now()
		if order.CreatedAt.IsZero() {
			order.CreatedAt = now
		}
		order.UpdatedAt = now

		filter := bson.M{"order_id": order.OrderID}
		update := bson.M{"$set": order}

		operation := mongo.NewUpdateOneModel()
		operation.SetFilter(filter)
		operation.SetUpdate(update)
		operation.SetUpsert(true)

		operations = append(operations, operation)
	}

	// Execute bulk write in batches to avoid memory issues
	batchSize := 1000
	for i := 0; i < len(operations); i += batchSize {
		end := i + batchSize
		if end > len(operations) {
			end = len(operations)
		}

		batch := operations[i:end]
		opts := options.BulkWrite().SetOrdered(false)

		_, err := collection.BulkWrite(ctx, batch, opts)
		if err != nil {
			return fmt.Errorf("failed to bulk write orders (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// Collection Management for Atomic Swapping

// RenameCollection atomically renames a collection
func (r *Repository) RenameCollection(ctx context.Context, fromName, toName string) error {
	cmd := bson.D{{Key: "renameCollection", Value: fmt.Sprintf("%s.%s", r.mongodb.Database.Name(), fromName)}, {Key: "to", Value: fmt.Sprintf("%s.%s", r.mongodb.Database.Name(), toName)}}

	// Use admin database for renameCollection command
	adminDB := r.mongodb.Client.Database("admin")
	return adminDB.RunCommand(ctx, cmd).Err()
}

// DropCollection drops a collection
func (r *Repository) DropCollection(ctx context.Context, collectionName string) error {
	collection := r.mongodb.Database.Collection(collectionName)
	return collection.Drop(ctx)
}

// GetCollectionStats returns statistics about a collection
func (r *Repository) GetCollectionStats(ctx context.Context, collectionName string) (bson.M, error) {
	cmd := bson.D{{Key: "collStats", Value: collectionName}}
	var result bson.M

	err := r.mongodb.Database.RunCommand(ctx, cmd).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	return result, nil
}

// Fetch Status Operations

// GetFetchStatus retrieves the fetch status for a specific region
func (r *Repository) GetFetchStatus(ctx context.Context, regionID int) (*models.MarketFetchStatus, error) {
	filter := bson.M{"region_id": regionID}

	var status models.MarketFetchStatus
	err := r.statusCollection.FindOne(ctx, filter).Decode(&status)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found, not an error
		}
		return nil, fmt.Errorf("failed to get fetch status: %w", err)
	}

	return &status, nil
}

// UpsertFetchStatus updates or creates a fetch status record
func (r *Repository) UpsertFetchStatus(ctx context.Context, status *models.MarketFetchStatus) error {
	now := time.Now()
	if status.CreatedAt.IsZero() {
		status.CreatedAt = now
	}
	status.UpdatedAt = now

	filter := bson.M{"region_id": status.RegionID}
	update := bson.M{"$set": status}
	opts := options.Update().SetUpsert(true)

	_, err := r.statusCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert fetch status: %w", err)
	}

	return nil
}

// GetAllFetchStatuses retrieves fetch status for all regions
func (r *Repository) GetAllFetchStatuses(ctx context.Context) ([]models.MarketFetchStatus, error) {
	cursor, err := r.statusCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find fetch statuses: %w", err)
	}
	defer cursor.Close(ctx)

	var statuses []models.MarketFetchStatus
	if err := cursor.All(ctx, &statuses); err != nil {
		return nil, fmt.Errorf("failed to decode fetch statuses: %w", err)
	}

	return statuses, nil
}

// GetOverallStats retrieves overall market statistics
func (r *Repository) GetOverallStats(ctx context.Context) (map[string]interface{}, error) {
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":              nil,
			"total_orders":     bson.M{"$sum": 1},
			"buy_orders":       bson.M{"$sum": bson.M{"$cond": bson.A{"$is_buy_order", 1, 0}}},
			"sell_orders":      bson.M{"$sum": bson.M{"$cond": bson.A{"$is_buy_order", 0, 1}}},
			"unique_types":     bson.M{"$addToSet": "$type_id"},
			"unique_regions":   bson.M{"$addToSet": "$region_id"},
			"unique_locations": bson.M{"$addToSet": "$location_id"},
			"oldest_data":      bson.M{"$min": "$fetched_at"},
			"newest_data":      bson.M{"$max": "$fetched_at"},
		}},
		{"$addFields": bson.M{
			"unique_types":     bson.M{"$size": "$unique_types"},
			"unique_regions":   bson.M{"$size": "$unique_regions"},
			"unique_locations": bson.M{"$size": "$unique_locations"},
		}},
	}

	cursor, err := r.ordersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate overall stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation result: %w", err)
	}

	if len(result) == 0 {
		return map[string]interface{}{}, nil
	}

	return result[0], nil
}

// CreateIndexes creates necessary database indexes for optimal performance
func (r *Repository) CreateIndexes(ctx context.Context) error {
	// Market orders indexes
	ordersIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "order_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("order_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "type_id", Value: 1}, {Key: "location_id", Value: 1}},
			Options: options.Index().SetName("type_location_compound"),
		},
		{
			Keys:    bson.D{{Key: "location_id", Value: 1}, {Key: "is_buy_order", Value: 1}},
			Options: options.Index().SetName("location_order_type"),
		},
		{
			Keys:    bson.D{{Key: "region_id", Value: 1}, {Key: "type_id", Value: 1}},
			Options: options.Index().SetName("region_type_compound"),
		},
		{
			Keys:    bson.D{{Key: "fetched_at", Value: 1}},
			Options: options.Index().SetName("fetched_at_timestamp"),
		},
		{
			Keys:    bson.D{{Key: "price", Value: 1}},
			Options: options.Index().SetName("price_sort"),
		},
	}

	_, err := r.ordersCollection.Indexes().CreateMany(ctx, ordersIndexes)
	if err != nil {
		return fmt.Errorf("failed to create market orders indexes: %w", err)
	}

	// Fetch status indexes
	statusIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "region_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("region_id_unique"),
		},
		{
			Keys:    bson.D{{Key: "last_fetch_time", Value: 1}},
			Options: options.Index().SetName("last_fetch_time"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_filter"),
		},
	}

	_, err = r.statusCollection.Indexes().CreateMany(ctx, statusIndexes)
	if err != nil {
		return fmt.Errorf("failed to create fetch status indexes: %w", err)
	}

	return nil
}
