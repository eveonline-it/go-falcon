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

// GetUserSitemap returns routes and navigation filtered by user permissions
func (s *Service) GetUserSitemap(ctx context.Context, userID string, characterID int64, includeDisabled, includeHidden bool) (*models.SitemapResponse, error) {
	// Get user permissions
	userPermissions, err := s.getUserPermissions(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Get user groups - TODO: Fix interface integration
	var userGroups []string
	if s.groupService != nil {
		// For now, return empty groups - can be enhanced when interface is properly integrated
		userGroups = []string{}
	}

	// Check if user is super admin
	isSuperAdmin := false
	for _, group := range userGroups {
		if group == "Super Administrator" || group == "super_admin" {
			isSuperAdmin = true
			break
		}
	}

	// Fetch all enabled routes (or all routes for admin)
	filter := bson.M{}
	if !includeDisabled && !isSuperAdmin {
		filter["is_enabled"] = true
	}

	routes, err := s.repository.GetRoutes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	// Filter routes based on permissions
	accessibleRoutes := s.filterRoutesByPermissions(routes, userPermissions, userGroups, isSuperAdmin)

	// Build route configs for frontend
	routeConfigs := s.buildRouteConfigs(accessibleRoutes)

	// Build navigation structure
	navigation := s.buildNavigation(accessibleRoutes, includeHidden)

	// Get feature flags (could be based on user settings, environment, etc.)
	features := s.getUserFeatures(ctx, characterID)

	return &models.SitemapResponse{
		Routes:          routeConfigs,
		Navigation:      navigation,
		UserPermissions: userPermissions,
		UserGroups:      userGroups,
		Features:        features,
	}, nil
}

// GetPublicSitemap returns only public routes for unauthenticated users
func (s *Service) GetPublicSitemap(ctx context.Context) (*dto.PublicSitemapResponse, error) {
	// Fetch only public enabled routes
	filter := bson.M{
		"type":       models.RouteTypePublic,
		"is_enabled": true,
	}

	routes, err := s.repository.GetRoutes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get public routes: %w", err)
	}

	// Build route configs
	routeConfigs := s.buildRouteConfigs(routes)

	// Build navigation (only public items)
	navigation := s.buildNavigation(routes, false)

	return &dto.PublicSitemapResponse{
		Routes:     routeConfigs,
		Navigation: navigation,
	}, nil
}

// filterRoutesByPermissions filters routes based on user permissions and groups
func (s *Service) filterRoutesByPermissions(routes []models.Route, permissions []string, groups []string, isSuperAdmin bool) []models.Route {
	// Super admin gets all routes
	if isSuperAdmin {
		return routes
	}

	var accessible []models.Route
	permMap := make(map[string]bool)
	for _, p := range permissions {
		permMap[p] = true
	}

	groupMap := make(map[string]bool)
	for _, g := range groups {
		groupMap[g] = true
	}

	for _, route := range routes {
		// Check route type
		switch route.Type {
		case models.RouteTypePublic:
			accessible = append(accessible, route)

		case models.RouteTypeAuth:
			// Just needs authentication (user is authenticated if we're here)
			accessible = append(accessible, route)

		case models.RouteTypeProtected:
			// Check required permissions (AND logic)
			hasAllPerms := true
			for _, reqPerm := range route.RequiredPermissions {
				if !permMap[reqPerm] {
					hasAllPerms = false
					break
				}
			}

			// Check required groups (OR logic)
			hasAnyGroup := false
			if len(route.RequiredGroups) > 0 {
				for _, reqGroup := range route.RequiredGroups {
					if groupMap[reqGroup] {
						hasAnyGroup = true
						break
					}
				}
			}

			// Access granted if has all permissions OR any required group
			if hasAllPerms || (len(route.RequiredGroups) > 0 && hasAnyGroup) {
				accessible = append(accessible, route)
			}

		case models.RouteTypeAdmin:
			// Already handled by super admin check above
			continue
		}
	}

	return accessible
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
	// Map to store grouped items
	groupMap := make(map[string][]models.NavItem)
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
			groupMap[*route.Group] = append(groupMap[*route.Group], navItem)
		} else {
			ungrouped = append(ungrouped, navItem)
		}
	}

	// Build navigation groups
	var navGroups []models.NavigationGroup

	// Add grouped items
	for groupName, items := range groupMap {
		navGroups = append(navGroups, models.NavigationGroup{
			Label: groupName,
			Items: items,
		})
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
	// Get existing route
	route, err := s.repository.GetRouteByRouteID(ctx, routeID)
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
	// Get route to check if it exists
	route, err := s.repository.GetRouteByRouteID(ctx, routeID)
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
		SetSort(bson.M{"nav_order": 1, "created_at": -1})

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

// GetRouteStats returns statistics about routes
func (s *Service) GetRouteStats(ctx context.Context) (*dto.RouteStatsResponse, error) {
	stats, err := s.repository.GetRouteStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get route statistics: %w", err)
	}
	return stats, nil
}

// CheckRouteAccess checks if a user can access a specific route
func (s *Service) CheckRouteAccess(ctx context.Context, routeID string, characterID int64) (*dto.RouteAccessResponse, error) {
	// Get route
	route, err := s.repository.GetRouteByRouteID(ctx, routeID)
	if err != nil {
		return &dto.RouteAccessResponse{
			RouteID:    routeID,
			Accessible: false,
			Reason:     "Route not found",
		}, nil
	}

	// Get user permissions
	userPermissions, err := s.getUserPermissions(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check access
	response := &dto.RouteAccessResponse{
		RouteID:    route.RouteID,
		Path:       route.Path,
		Accessible: true,
	}

	// Check based on route type
	switch route.Type {
	case models.RouteTypePublic:
		// Always accessible
		return response, nil

	case models.RouteTypeAuth:
		// Just needs authentication (already authenticated if here)
		return response, nil

	case models.RouteTypeProtected:
		// Check permissions
		permMap := make(map[string]bool)
		for _, p := range userPermissions {
			permMap[p] = true
		}

		var missing []string
		for _, reqPerm := range route.RequiredPermissions {
			if !permMap[reqPerm] {
				missing = append(missing, reqPerm)
			}
		}

		if len(missing) > 0 {
			response.Accessible = false
			response.Reason = "Missing required permissions"
			response.Missing = missing
		}

	case models.RouteTypeAdmin:
		// Check for super admin
		response.Accessible = false
		response.Reason = "Admin access required"
	}

	return response, nil
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

// getUserPermissions gets all permissions for a character by checking against available permissions
func (s *Service) getUserPermissions(ctx context.Context, characterID int64) ([]string, error) {
	// Get all available permissions
	allPermissions := s.permissionManager.GetAllPermissions()

	var userPermissions []string

	// Check each permission to see if user has it
	for permissionID := range allPermissions {
		hasPermission, err := s.permissionManager.HasPermission(ctx, characterID, permissionID)
		if err != nil {
			// Log error but continue with other permissions
			continue
		}

		if hasPermission {
			userPermissions = append(userPermissions, permissionID)
		}
	}

	return userPermissions, nil
}
