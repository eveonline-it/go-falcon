package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/sitemap/dto"
	"go-falcon/internal/sitemap/models"
	"go-falcon/pkg/permissions"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GroupServiceInterface defines the interface for group service operations
// For now, use interface{} to avoid interface mismatch - can be refined later
type GroupServiceInterface interface{}

// Service handles sitemap business logic
type Service struct {
	db                *mongo.Database
	repository        *Repository
	permissionManager *permissions.PermissionManager
	groupService      GroupServiceInterface
}

// NewService creates a new sitemap service
func NewService(db *mongo.Database, permissionManager *permissions.PermissionManager, groupService GroupServiceInterface) *Service {
	return &Service{
		db:                db,
		repository:        NewRepository(db),
		permissionManager: permissionManager,
		groupService:      groupService,
	}
}

// GetAllEnabledRoutes returns all enabled routes regardless of type (for testing/debugging)
func (s *Service) GetAllEnabledRoutes(ctx context.Context, includeDisabled, includeHidden bool) (*models.SitemapResponse, error) {
	// Build filter based on parameters
	filter := bson.M{}
	if !includeDisabled {
		filter["is_enabled"] = true
	}

	routes, err := s.repository.GetRoutes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get all routes: %w", err)
	}

	// Build route configs
	routeConfigs := s.buildRouteConfigs(routes)

	// Build navigation
	navigation := s.buildNavigation(routes, includeHidden)

	return &models.SitemapResponse{
		Routes:          routeConfigs,
		Navigation:      navigation,
		UserPermissions: []string{}, // Empty for now
		UserGroups:      []string{}, // Empty for now
		Features:        make(map[string]bool),
	}, nil
}

// buildRouteConfigs converts routes to frontend-consumable format
func (s *Service) buildRouteConfigs(routes []models.Route) []models.RouteConfig {
	// Build parent-child map
	childMap := make(map[string][]models.Route)
	rootRoutes := []models.Route{}

	for _, route := range routes {
		if route.ParentID != nil && *route.ParentID != "" {
			childMap[*route.ParentID] = append(childMap[*route.ParentID], route)
		} else {
			rootRoutes = append(rootRoutes, route)
		}
	}

	// Build hierarchical structure
	var buildConfigs func([]models.Route) []models.RouteConfig
	buildConfigs = func(routeList []models.Route) []models.RouteConfig {
		configs := make([]models.RouteConfig, 0, len(routeList))
		for _, route := range routeList {
			config := models.RouteConfig{
				ID:          route.RouteID,
				Path:        route.Path,
				Component:   route.Component,
				Name:        route.Name,
				Icon:        route.Icon,
				Title:       route.Title,
				Permissions: route.RequiredPermissions,
				Props:       route.Props,
				LazyLoad:    route.LazyLoad,
				Accessible:  true, // Already filtered
				Meta: &models.RouteMeta{
					Title:       route.Title,
					Icon:        route.Icon,
					Group:       route.Group,
					Description: route.Description,
				},
			}

			// Add children if any
			if children, exists := childMap[route.RouteID]; exists {
				config.Children = buildConfigs(children)
			}

			configs = append(configs, config)
		}
		return configs
	}

	return buildConfigs(rootRoutes)
}

// buildNavigation creates navigation structure from routes
func (s *Service) buildNavigation(routes []models.Route, includeHidden bool) []models.NavigationGroup {
	// Group routes by navigation position
	navGroups := make(map[models.NavigationPosition][]models.Route)

	for _, route := range routes {
		if !route.ShowInNav && !includeHidden {
			continue
		}
		if route.NavPosition == models.NavHidden && !includeHidden {
			continue
		}
		navGroups[route.NavPosition] = append(navGroups[route.NavPosition], route)
	}

	// Build navigation groups
	var navigation []models.NavigationGroup

	// Process each navigation position in order
	positions := []models.NavigationPosition{models.NavMain, models.NavUser, models.NavAdmin, models.NavFooter}

	for _, pos := range positions {
		if routes, exists := navGroups[pos]; exists && len(routes) > 0 {
			// Sort routes by NavOrder
			sortedRoutes := s.sortRoutesByOrder(routes)

			// Group by route.Group if specified
			groupedItems := s.groupNavigationItems(sortedRoutes)

			navigation = append(navigation, groupedItems...)
		}
	}

	return navigation
}

// sortRoutesByOrder sorts routes by their navigation order
func (s *Service) sortRoutesByOrder(routes []models.Route) []models.Route {
	// Simple bubble sort for small arrays
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[i].NavOrder > routes[j].NavOrder {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}
	return routes
}

// groupNavigationItems groups navigation items by their group field
func (s *Service) groupNavigationItems(routes []models.Route) []models.NavigationGroup {
	// Map to store grouped items and track minimum nav_order for each group
	groupMap := make(map[string][]models.NavItem)
	groupOrder := make(map[string]int) // Track minimum nav_order for each group
	var ungrouped []models.NavItem

	for _, route := range routes {
		navItem := models.NavItem{
			RouteID: route.RouteID,
			Name:    route.Name,
			To:      route.Path,
			Icon:    route.Icon,
			Active:  route.IsEnabled,
			Exact:   route.Exact,
			NewTab:  route.NewTab,
		}

		// Add badge if specified
		if route.BadgeType != nil && route.BadgeText != nil {
			navItem.Badge = &models.Badge{
				Type: *route.BadgeType,
				Text: *route.BadgeText,
			}
		}

		// Group items
		if route.Group != nil && *route.Group != "" {
			groupName := *route.Group
			groupMap[groupName] = append(groupMap[groupName], navItem)

			// Track the minimum nav_order for this group
			if existingOrder, exists := groupOrder[groupName]; !exists || route.NavOrder < existingOrder {
				groupOrder[groupName] = route.NavOrder
			}
		} else {
			ungrouped = append(ungrouped, navItem)
		}
	}

	// Build navigation groups sorted by group order
	var navGroups []models.NavigationGroup

	// Create a sorted list of group names by their nav_order
	type GroupInfo struct {
		Name  string
		Order int
	}
	var sortedGroups []GroupInfo
	for groupName, order := range groupOrder {
		sortedGroups = append(sortedGroups, GroupInfo{Name: groupName, Order: order})
	}

	// Sort groups by their minimum nav_order
	for i := 0; i < len(sortedGroups); i++ {
		for j := i + 1; j < len(sortedGroups); j++ {
			if sortedGroups[i].Order > sortedGroups[j].Order {
				sortedGroups[i], sortedGroups[j] = sortedGroups[j], sortedGroups[i]
			}
		}
	}

	// Add grouped items in correct order
	for _, group := range sortedGroups {
		if items, exists := groupMap[group.Name]; exists {
			navGroups = append(navGroups, models.NavigationGroup{
				Label: group.Name,
				Items: items,
			})
		}
	}

	// Add ungrouped items
	if len(ungrouped) > 0 {
		navGroups = append(navGroups, models.NavigationGroup{
			Label:        "General",
			LabelDisable: true,
			Items:        ungrouped,
		})
	}

	return navGroups
}

// getUserFeatures returns feature flags for a user
func (s *Service) getUserFeatures(ctx context.Context, characterID int64) map[string]bool {
	// This could be expanded to check user settings, environment variables, etc.
	features := make(map[string]bool)

	// Example feature flags
	features["darkMode"] = true
	features["advancedAnalytics"] = true
	features["betaFeatures"] = false

	return features
}

// CreateRoute creates a new route
func (s *Service) CreateRoute(ctx context.Context, input *dto.CreateRouteInput) (*models.Route, error) {
	// Check if route ID already exists
	existing, _ := s.repository.GetRouteByRouteID(ctx, input.RouteID)
	if existing != nil {
		return nil, fmt.Errorf("route with ID %s already exists", input.RouteID)
	}

	route := &models.Route{
		RouteID:             input.RouteID,
		Path:                input.Path,
		Component:           input.Component,
		Name:                input.Name,
		Icon:                input.Icon,
		Type:                input.Type,
		ParentID:            input.ParentID,
		NavPosition:         input.NavPosition,
		NavOrder:            input.NavOrder,
		ShowInNav:           input.ShowInNav,
		RequiredPermissions: input.RequiredPermissions,
		RequiredGroups:      input.RequiredGroups,
		Title:               input.Title,
		Description:         input.Description,
		Keywords:            input.Keywords,
		Group:               input.Group,
		FeatureFlags:        input.FeatureFlags,
		IsEnabled:           input.IsEnabled,
		Props:               input.Props,
		LazyLoad:            input.LazyLoad,
		Exact:               input.Exact,
		NewTab:              input.NewTab,
		BadgeType:           input.BadgeType,
		BadgeText:           input.BadgeText,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	id, err := s.repository.CreateRoute(ctx, route)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	route.ID = id
	return route, nil
}

// UpdateRoute updates an existing route
func (s *Service) UpdateRoute(ctx context.Context, routeID string, input *dto.UpdateRouteInput) (*models.Route, error) {
	// Get existing route (handles both ObjectID and route_id)
	route, err := s.GetRouteByID(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}

	// Update fields if provided
	updateDoc := bson.M{"updated_at": time.Now()}

	if input.Path != nil {
		updateDoc["path"] = *input.Path
	}
	if input.Component != nil {
		updateDoc["component"] = *input.Component
	}
	if input.Name != nil {
		updateDoc["name"] = *input.Name
	}
	if input.Icon != nil {
		updateDoc["icon"] = *input.Icon
	}
	if input.Type != nil {
		updateDoc["type"] = *input.Type
	}
	if input.ParentID != nil {
		updateDoc["parent_id"] = *input.ParentID
	}
	if input.NavPosition != nil {
		updateDoc["nav_position"] = *input.NavPosition
	}
	if input.NavOrder != nil {
		updateDoc["nav_order"] = *input.NavOrder
	}
	if input.ShowInNav != nil {
		updateDoc["show_in_nav"] = *input.ShowInNav
	}
	if len(input.RequiredPermissions) > 0 {
		updateDoc["required_permissions"] = input.RequiredPermissions
	}
	if len(input.RequiredGroups) > 0 {
		updateDoc["required_groups"] = input.RequiredGroups
	}
	if input.Title != nil {
		updateDoc["title"] = *input.Title
	}
	if input.Description != nil {
		updateDoc["description"] = *input.Description
	}
	if len(input.Keywords) > 0 {
		updateDoc["keywords"] = input.Keywords
	}
	if input.Group != nil {
		updateDoc["group"] = *input.Group
	}
	if len(input.FeatureFlags) > 0 {
		updateDoc["feature_flags"] = input.FeatureFlags
	}
	if input.IsEnabled != nil {
		updateDoc["is_enabled"] = *input.IsEnabled
	}
	if input.Props != nil {
		updateDoc["props"] = input.Props
	}
	if input.LazyLoad != nil {
		updateDoc["lazy_load"] = *input.LazyLoad
	}
	if input.Exact != nil {
		updateDoc["exact"] = *input.Exact
	}
	if input.NewTab != nil {
		updateDoc["newtab"] = *input.NewTab
	}
	if input.BadgeType != nil {
		updateDoc["badge_type"] = *input.BadgeType
	}
	if input.BadgeText != nil {
		updateDoc["badge_text"] = *input.BadgeText
	}

	err = s.repository.UpdateRoute(ctx, route.ID, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	// Get updated route
	return s.repository.GetRouteByID(ctx, route.ID)
}

// DeleteRoute deletes a route and its children
func (s *Service) DeleteRoute(ctx context.Context, routeID string) (int, error) {
	// Get route to check if it exists (handles both ObjectID and route_id)
	route, err := s.GetRouteByID(ctx, routeID)
	if err != nil {
		return 0, fmt.Errorf("route not found: %w", err)
	}

	// Delete route and all children
	deleted, err := s.repository.DeleteRouteAndChildren(ctx, route.RouteID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete route: %w", err)
	}

	return deleted, nil
}

// GetRoutes lists routes with filtering
func (s *Service) GetRoutes(ctx context.Context, input *dto.ListRoutesInput) ([]models.Route, int64, error) {
	filter := bson.M{}

	// Handle string-based filters with "all" as no filter
	if input.Type != "" && input.Type != "all" {
		filter["type"] = input.Type
	}
	if input.Group != "" {
		filter["group"] = input.Group
	}
	if input.IsEnabled != "" && input.IsEnabled != "all" {
		if input.IsEnabled == "true" {
			filter["is_enabled"] = true
		} else if input.IsEnabled == "false" {
			filter["is_enabled"] = false
		}
	}
	if input.ShowInNav != "" && input.ShowInNav != "all" {
		if input.ShowInNav == "true" {
			filter["show_in_nav"] = true
		} else if input.ShowInNav == "false" {
			filter["show_in_nav"] = false
		}
	}
	if input.NavPosition != "" && input.NavPosition != "all" {
		filter["nav_position"] = input.NavPosition
	}

	opts := options.Find().
		SetSkip(int64((input.Page - 1) * input.Limit)).
		SetLimit(int64(input.Limit)).
		SetSort(bson.D{{"nav_order", 1}, {"created_at", -1}})

	routes, err := s.repository.GetRoutesWithOptions(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get routes: %w", err)
	}

	count, err := s.repository.CountRoutes(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count routes: %w", err)
	}

	return routes, count, nil
}

// GetRouteByID gets a single route by ID
func (s *Service) GetRouteByID(ctx context.Context, id string) (*models.Route, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		// Try as route_id instead
		return s.repository.GetRouteByRouteID(ctx, id)
	}
	return s.repository.GetRouteByID(ctx, objID)
}

// BulkUpdateOrder updates navigation order for multiple routes
func (s *Service) BulkUpdateOrder(ctx context.Context, updates []dto.OrderUpdate) (int, int, []string) {
	updated := 0
	failed := 0
	var errors []string

	for _, update := range updates {
		err := s.repository.UpdateRouteOrder(ctx, update.RouteID, update.NavOrder)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Failed to update %s: %v", update.RouteID, err))
		} else {
			updated++
		}
	}

	return updated, failed, errors
}

// GetStatus returns the module health status
func (s *Service) GetStatus(ctx context.Context) *dto.StatusResponse {
	// Check database connection
	err := s.db.Client().Ping(ctx, nil)
	if err != nil {
		return &dto.StatusResponse{
			Module:  "sitemap",
			Status:  "unhealthy",
			Message: fmt.Sprintf("Database connection failed: %v", err),
		}
	}

	// Check if routes collection is accessible
	count, err := s.repository.CountRoutes(ctx, bson.M{})
	if err != nil {
		return &dto.StatusResponse{
			Module:  "sitemap",
			Status:  "unhealthy",
			Message: fmt.Sprintf("Cannot access routes collection: %v", err),
		}
	}

	return &dto.StatusResponse{
		Module:  "sitemap",
		Status:  "healthy",
		Message: fmt.Sprintf("Managing %d routes", count),
	}
}
