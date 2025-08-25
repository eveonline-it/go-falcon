package services

import (
	"context"
	"time"

	"go-falcon/internal/sitemap/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for routes
type Repository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

// NewRepository creates a new repository
func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		db:         db,
		collection: db.Collection(models.RoutesCollection),
	}
}

// CreateRoute creates a new route
func (r *Repository) CreateRoute(ctx context.Context, route *models.Route) (primitive.ObjectID, error) {
	result, err := r.collection.InsertOne(ctx, route)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

// GetRouteByID gets a route by its MongoDB ID
func (r *Repository) GetRouteByID(ctx context.Context, id primitive.ObjectID) (*models.Route, error) {
	var route models.Route
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&route)
	if err != nil {
		return nil, err
	}
	return &route, nil
}

// GetRouteByRouteID gets a route by its route_id field
func (r *Repository) GetRouteByRouteID(ctx context.Context, routeID string) (*models.Route, error) {
	var route models.Route
	err := r.collection.FindOne(ctx, bson.M{"route_id": routeID}).Decode(&route)
	if err != nil {
		return nil, err
	}
	return &route, nil
}

// GetRoutes gets routes with a filter
func (r *Repository) GetRoutes(ctx context.Context, filter bson.M) ([]models.Route, error) {
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"nav_order", 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []models.Route
	if err = cursor.All(ctx, &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

// GetRoutesWithOptions gets routes with filter and options
func (r *Repository) GetRoutesWithOptions(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]models.Route, error) {
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var routes []models.Route
	if err = cursor.All(ctx, &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

// UpdateRoute updates a route
func (r *Repository) UpdateRoute(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
	)
	return err
}

// UpdateRouteOrder updates the navigation order of a route
func (r *Repository) UpdateRouteOrder(ctx context.Context, routeID string, order int) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"route_id": routeID},
		bson.M{
			"$set": bson.M{
				"nav_order":  order,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}

// DeleteRoute deletes a route
func (r *Repository) DeleteRoute(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteRouteAndChildren deletes a route and all its children
func (r *Repository) DeleteRouteAndChildren(ctx context.Context, routeID string) (int, error) {
	// Find all children recursively
	routeIDs := []string{routeID}
	allRouteIDs := []string{routeID}

	for len(routeIDs) > 0 {
		// Find children of current routes
		cursor, err := r.collection.Find(ctx, bson.M{"parent_id": bson.M{"$in": routeIDs}})
		if err != nil {
			return 0, err
		}

		var children []models.Route
		if err = cursor.All(ctx, &children); err != nil {
			cursor.Close(ctx)
			return 0, err
		}
		cursor.Close(ctx)

		// Collect child IDs for next iteration
		routeIDs = []string{}
		for _, child := range children {
			routeIDs = append(routeIDs, child.RouteID)
			allRouteIDs = append(allRouteIDs, child.RouteID)
		}
	}

	// Delete all routes
	result, err := r.collection.DeleteMany(ctx, bson.M{"route_id": bson.M{"$in": allRouteIDs}})
	if err != nil {
		return 0, err
	}

	return int(result.DeletedCount), nil
}

// CountRoutes counts routes matching a filter
func (r *Repository) CountRoutes(ctx context.Context, filter bson.M) (int64, error) {
	return r.collection.CountDocuments(ctx, filter)
}

// CreateIndexes creates database indexes for optimal performance
func (r *Repository) CreateIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// Unique index on route_id
		{
			Keys:    bson.M{"route_id": 1},
			Options: options.Index().SetUnique(true),
		},
		// Index on type for filtering
		{
			Keys: bson.M{"type": 1},
		},
		// Index on is_enabled for filtering
		{
			Keys: bson.M{"is_enabled": 1},
		},
		// Index on nav_position and nav_order for navigation queries
		{
			Keys: bson.M{"nav_position": 1, "nav_order": 1},
		},
		// Index on parent_id for hierarchical queries
		{
			Keys: bson.M{"parent_id": 1},
		},
		// Index on group for grouping
		{
			Keys: bson.M{"group": 1},
		},
		// Compound index for common queries
		{
			Keys: bson.M{"is_enabled": 1, "type": 1, "nav_position": 1},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}
