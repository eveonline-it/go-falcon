package services

import (
	"context"
	"fmt"
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
		// Folder-specific indexes
		{
			Keys: bson.M{"is_folder": 1},
		},
		// Index on folder_path for path-based queries
		{
			Keys: bson.M{"folder_path": 1},
		},
		// Index on depth for depth-based filtering
		{
			Keys: bson.M{"depth": 1},
		},
		// Compound index for folder hierarchical queries
		{
			Keys: bson.M{"parent_id": 1, "depth": 1, "nav_order": 1},
		},
		// Compound index for folder type filtering
		{
			Keys: bson.M{"is_folder": 1, "nav_position": 1, "is_enabled": 1},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// Folder-specific repository methods

// GetFolders gets all folders matching a filter
func (r *Repository) GetFolders(ctx context.Context, filter bson.M) ([]models.Route, error) {
	// Add is_folder: true to the filter
	filter["is_folder"] = true

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"depth", 1}, {"nav_order", 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var folders []models.Route
	if err = cursor.All(ctx, &folders); err != nil {
		return nil, err
	}
	return folders, nil
}

// GetFolderChildren gets direct children of a folder
func (r *Repository) GetFolderChildren(ctx context.Context, folderID string, includeDisabled bool) ([]models.Route, error) {
	filter := bson.M{"parent_id": folderID}
	if !includeDisabled {
		filter["is_enabled"] = true
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"nav_order", 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var children []models.Route
	if err = cursor.All(ctx, &children); err != nil {
		return nil, err
	}
	return children, nil
}

// GetFolderPath builds the full path for a folder
func (r *Repository) GetFolderPath(ctx context.Context, folderID string) (string, error) {
	if folderID == "" {
		return "/", nil
	}

	folder, err := r.GetRouteByRouteID(ctx, folderID)
	if err != nil {
		return "", err
	}

	if folder.ParentID == nil || *folder.ParentID == "" {
		return "/" + folder.Name, nil
	}

	parentPath, err := r.GetFolderPath(ctx, *folder.ParentID)
	if err != nil {
		return "", err
	}

	if parentPath == "/" {
		return "/" + folder.Name, nil
	}
	return parentPath + "/" + folder.Name, nil
}

// UpdateFolderPath updates the folder_path field for a route
func (r *Repository) UpdateFolderPath(ctx context.Context, routeID string, path string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"route_id": routeID},
		bson.M{
			"$set": bson.M{
				"folder_path": path,
				"updated_at":  time.Now(),
			},
		},
	)
	return err
}

// UpdateChildrenCount updates the children_count field for a folder
func (r *Repository) UpdateChildrenCount(ctx context.Context, folderID string) error {
	// Count direct children
	count, err := r.collection.CountDocuments(ctx, bson.M{"parent_id": folderID, "is_enabled": true})
	if err != nil {
		return err
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"route_id": folderID},
		bson.M{
			"$set": bson.M{
				"children_count": int(count),
				"updated_at":     time.Now(),
			},
		},
	)
	return err
}

// MoveRouteToFolder moves a route/folder to a new parent
func (r *Repository) MoveRouteToFolder(ctx context.Context, routeID string, newParentID *string, newOrder *int) error {
	updateDoc := bson.M{"updated_at": time.Now()}

	if newParentID != nil {
		updateDoc["parent_id"] = *newParentID
	} else {
		updateDoc["parent_id"] = nil
	}

	if newOrder != nil {
		updateDoc["nav_order"] = *newOrder
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"route_id": routeID},
		bson.M{"$set": updateDoc},
	)
	return err
}

// GetRoutesByDepth gets routes at a specific depth level
func (r *Repository) GetRoutesByDepth(ctx context.Context, depth int, filter bson.M) ([]models.Route, error) {
	if filter == nil {
		filter = bson.M{}
	}
	filter["depth"] = depth

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

// GetMaxDepth gets the maximum depth used in the hierarchy
func (r *Repository) GetMaxDepth(ctx context.Context) (int, error) {
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":       nil,
			"max_depth": bson.M{"$max": "$depth"},
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err = cursor.All(ctx, &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}

	maxDepth, ok := result[0]["max_depth"].(int32)
	if !ok {
		return 0, nil
	}

	return int(maxDepth), nil
}

// ValidateDepth checks if adding a child would exceed max depth
func (r *Repository) ValidateDepth(ctx context.Context, parentID string, maxDepth int) (bool, int, error) {
	if parentID == "" {
		return true, 0, nil // Root level is always valid
	}

	parent, err := r.GetRouteByRouteID(ctx, parentID)
	if err != nil {
		return false, 0, err
	}

	newDepth := parent.Depth + 1
	return newDepth <= maxDepth, newDepth, nil
}

// GetFolderStats returns statistics about folder usage
func (r *Repository) GetFolderStats(ctx context.Context) (*models.FolderStats, error) {
	// Count total folders and routes
	totalFolders, err := r.collection.CountDocuments(ctx, bson.M{"is_folder": true})
	if err != nil {
		return nil, err
	}

	totalRoutes, err := r.collection.CountDocuments(ctx, bson.M{"is_folder": false})
	if err != nil {
		return nil, err
	}

	// Get max depth
	maxDepth, err := r.GetMaxDepth(ctx)
	if err != nil {
		return nil, err
	}

	// Count folders by depth
	pipeline := []bson.M{
		{"$match": bson.M{"is_folder": true}},
		{"$group": bson.M{
			"_id":   "$depth",
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"_id": 1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var depthResults []bson.M
	if err = cursor.All(ctx, &depthResults); err != nil {
		return nil, err
	}

	foldersByDepth := make(map[int]int)
	for _, result := range depthResults {
		depth := int(result["_id"].(int32))
		count := int(result["count"].(int32))
		foldersByDepth[depth] = count
	}

	// Find empty folders
	emptyFolders, err := r.getEmptyFolders(ctx)
	if err != nil {
		return nil, err
	}

	stats := &models.FolderStats{
		TotalFolders:      int(totalFolders),
		TotalRoutes:       int(totalRoutes),
		MaxDepthUsed:      maxDepth,
		FoldersByDepth:    foldersByDepth,
		EmptyFolders:      emptyFolders,
		DepthDistribution: make(map[string]int),
	}

	// Build depth distribution labels
	for depth, count := range foldersByDepth {
		stats.DepthDistribution[fmt.Sprintf("Level %d", depth)] = count
	}

	return stats, nil
}

// getEmptyFolders returns folder IDs that have no children
func (r *Repository) getEmptyFolders(ctx context.Context) ([]string, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"is_folder": true}},
		{"$lookup": bson.M{
			"from":         models.RoutesCollection,
			"localField":   "route_id",
			"foreignField": "parent_id",
			"as":           "children",
		}},
		{"$match": bson.M{"children": bson.M{"$size": 0}}},
		{"$project": bson.M{"route_id": 1}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	var emptyFolders []string
	for _, result := range results {
		emptyFolders = append(emptyFolders, result["route_id"].(string))
	}

	return emptyFolders, nil
}
