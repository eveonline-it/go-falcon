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

// buildRouteConfigs converts routes to frontend-consumable format (flat array)
func (s *Service) buildRouteConfigs(routes []models.Route) []models.RouteConfig {
	// Build flat array of routes without children
	configs := make([]models.RouteConfig, 0, len(routes))

	for _, route := range routes {
		// Skip folders as they're not actual routes
		if route.Type == models.RouteTypeFolder {
			continue
		}

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

		// Don't include children - keep it flat
		configs = append(configs, config)
	}

	return configs
}

// buildNavigation creates navigation structure from routes with folder support
func (s *Service) buildNavigation(routes []models.Route, includeHidden bool) []models.NavigationGroup {
	return s.buildHierarchicalNavigation(routes, includeHidden, models.MaxFolderDepth)
}

// buildHierarchicalNavigation creates hierarchical navigation structure with folders
func (s *Service) buildHierarchicalNavigation(routes []models.Route, includeHidden bool, maxDepth int) []models.NavigationGroup {
	// Filter routes for navigation
	var navRoutes []models.Route
	for _, route := range routes {
		if !route.ShowInNav && !includeHidden {
			continue
		}
		if route.NavPosition == models.NavHidden && !includeHidden {
			continue
		}
		// Skip routes that exceed max depth
		if route.Depth > maxDepth {
			continue
		}
		navRoutes = append(navRoutes, route)
	}

	// Sort all routes by navigation position first, then by nav order
	sortedRoutes := s.sortRoutesByPositionAndOrder(navRoutes)

	// Build hierarchical navigation items directly from the folder structure
	hierarchicalItems := s.buildHierarchicalNavItems(sortedRoutes, maxDepth)

	// Convert nav items directly to navigation groups based on folder structure
	return s.convertNavItemsToGroups(hierarchicalItems)
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

// buildHierarchicalNavItems creates hierarchical navigation items with folder support
func (s *Service) buildHierarchicalNavItems(routes []models.Route, maxDepth int) []models.NavItem {
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

	// Build hierarchical structure recursively
	var buildNavItems func([]models.Route, int) []models.NavItem
	buildNavItems = func(routeList []models.Route, currentDepth int) []models.NavItem {
		if currentDepth > maxDepth {
			return []models.NavItem{}
		}

		items := make([]models.NavItem, 0, len(routeList))

		for _, route := range routeList {
			item := models.NavItem{
				RouteID: route.RouteID,
				Name:    route.Name,
				Icon:    route.Icon,
				Active:  route.IsEnabled,
				Exact:   route.Exact,
				NewTab:  route.NewTab,
				Depth:   route.Depth,
			}

			// Handle folders vs regular routes differently
			if route.IsFolder {
				item.IsFolder = true
				item.To = "" // Folders don't have routes

				// Set expanded state
				if route.IsExpanded != nil {
					item.IsOpen = *route.IsExpanded
				} else {
					item.IsOpen = false // Default to closed
				}

				// Use appropriate folder icon
				if route.Icon != nil {
					if item.IsOpen {
						item.Icon = models.DefaultOpenFolderIcon
					} else {
						item.Icon = models.DefaultFolderIcon
					}
				}
			} else {
				item.IsFolder = false
				item.To = route.Path
			}

			// Add badge if specified
			if route.BadgeType != nil && route.BadgeText != nil {
				item.Badge = &models.Badge{
					Type: *route.BadgeType,
					Text: *route.BadgeText,
				}
			}

			// Add children if any
			if children, exists := childMap[route.RouteID]; exists {
				// IMPORTANT: Sort children by nav_order before building nav items
				sortedChildren := s.sortRoutesByOrder(children)
				childItems := buildNavItems(sortedChildren, currentDepth+1)
				item.Children = childItems
				item.HasChildren = len(childItems) > 0
			}

			items = append(items, item)
		}

		return items
	}

	return buildNavItems(rootRoutes, 0)
}

// sortRoutesByPositionAndOrder sorts routes by navigation position first, then by nav order
func (s *Service) sortRoutesByPositionAndOrder(routes []models.Route) []models.Route {
	// Define position priority order
	positionPriority := map[models.NavigationPosition]int{
		models.NavMain:   1,
		models.NavUser:   2,
		models.NavAdmin:  3,
		models.NavFooter: 4,
		models.NavHidden: 5,
	}

	// Sort by position first, then nav order
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			iPriority := positionPriority[routes[i].NavPosition]
			jPriority := positionPriority[routes[j].NavPosition]

			// First sort by position
			if iPriority > jPriority {
				routes[i], routes[j] = routes[j], routes[i]
			} else if iPriority == jPriority {
				// Same position, sort by nav order
				if routes[i].NavOrder > routes[j].NavOrder {
					routes[i], routes[j] = routes[j], routes[i]
				}
			}
		}
	}
	return routes
}

// convertNavItemsToGroups converts hierarchical nav items directly to navigation groups
func (s *Service) convertNavItemsToGroups(items []models.NavItem) []models.NavigationGroup {
	// Each top-level item becomes a navigation group
	var groups []models.NavigationGroup

	for _, item := range items {
		// Only create groups for folders (top-level containers)
		if item.IsFolder {
			groups = append(groups, models.NavigationGroup{
				Label:        item.Name,
				LabelDisable: false,         // Show folder names
				Items:        item.Children, // Direct children
			})
		} else {
			// Non-folder top-level items go into a general group
			groups = append(groups, models.NavigationGroup{
				Label:        "Navigation",
				LabelDisable: true,
				Items:        []models.NavItem{item},
			})
		}
	}

	return groups
}

// GetUserRoutesWithFolders returns user-specific sitemap with folder support
func (s *Service) GetUserRoutesWithFolders(ctx context.Context, input *dto.GetUserRoutesInput) (*models.SitemapResponse, error) {
	// Build filter based on parameters
	filter := bson.M{}
	if !input.IncludeDisabled {
		filter["is_enabled"] = true
	}

	routes, err := s.repository.GetRoutes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	// Build route configs with hierarchy
	routeConfigs := s.buildRouteConfigs(routes)

	// Build navigation with folder support
	navigation := s.buildHierarchicalNavigation(routes, input.IncludeHidden, input.MaxDepth)

	// Apply folder expansion settings
	if input.ExpandFolders {
		s.expandAllFolders(navigation)
	}

	return &models.SitemapResponse{
		Routes:          routeConfigs,
		Navigation:      navigation,
		UserPermissions: []string{}, // TODO: Implement permission extraction
		UserGroups:      []string{}, // TODO: Implement group extraction
		Features:        make(map[string]bool),
	}, nil
}

// expandAllFolders recursively expands all folders in navigation
func (s *Service) expandAllFolders(navigation []models.NavigationGroup) {
	for i := range navigation {
		s.expandFoldersInItems(navigation[i].Items)
	}
}

// expandFoldersInItems recursively expands folders in navigation items
func (s *Service) expandFoldersInItems(items []models.NavItem) {
	for i := range items {
		if items[i].IsFolder {
			items[i].IsOpen = true
		}
		if len(items[i].Children) > 0 {
			s.expandFoldersInItems(items[i].Children)
		}
	}
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

	// Calculate depth based on parent
	depth := 0
	if input.ParentID != nil && *input.ParentID != "" {
		parent, err := s.repository.GetRouteByRouteID(ctx, *input.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent route not found: %w", err)
		}
		depth = parent.Depth + 1

		// Validate depth limits
		if depth > models.MaxFolderDepth {
			return nil, fmt.Errorf("route depth would exceed maximum of %d levels", models.MaxFolderDepth)
		}
	}

	// Determine if this is a folder
	isFolder := input.Type == models.RouteTypeFolder

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

		// Folder-specific fields
		IsFolder:      isFolder,
		Depth:         depth,
		ChildrenCount: 0,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Build folder path
	if isFolder || input.ParentID != nil {
		folderPath, err := s.buildFolderPathForRoute(ctx, route)
		if err == nil {
			route.FolderPath = folderPath
		}
	}

	id, err := s.repository.CreateRoute(ctx, route)
	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	route.ID = id

	// Update parent's children count if applicable
	if input.ParentID != nil && *input.ParentID != "" {
		s.repository.UpdateChildrenCount(ctx, *input.ParentID)
	}

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

	// Determine sort order based on input
	var sortFields bson.D
	switch input.Sort {
	case "flat":
		// Flat sorting - just by nav_order
		sortFields = bson.D{{"nav_order", 1}, {"created_at", -1}}
	case "nav_order":
		// Sort by nav_order only
		sortFields = bson.D{{"nav_order", 1}}
	case "created_at":
		// Sort by creation date
		sortFields = bson.D{{"created_at", -1}}
	default: // "hierarchical" or empty
		// Hierarchical sorting: depth first, then nav_order
		sortFields = bson.D{{"depth", 1}, {"nav_order", 1}, {"created_at", -1}}
	}

	opts := options.Find().
		SetSkip(int64((input.Page - 1) * input.Limit)).
		SetLimit(int64(input.Limit)).
		SetSort(sortFields)

	routes, err := s.repository.GetRoutesWithOptions(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get routes: %w", err)
	}

	// Apply hierarchical sorting only if requested (default)
	if input.Sort == "" || input.Sort == "hierarchical" {
		routes = s.sortRoutesHierarchicallyForAdmin(routes)
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

	if len(updates) == 0 {
		return updated, failed, errors
	}

	for _, update := range updates {
		err := s.repository.UpdateRouteOrder(ctx, update.RouteID, update.NavOrder)
		if err != nil {
			failed++
			errors = append(errors, fmt.Sprintf("Failed to update %s: %v", update.RouteID, err))
		} else {
			updated++
		}
	}

	// Force refresh any cached navigation data by triggering a test fetch
	if updated > 0 {
		go func() {
			// This will force the system to rebuild navigation with new order
			_, _ = s.GetUserRoutesWithFolders(ctx, &dto.GetUserRoutesInput{
				IncludeDisabled: false,
				IncludeHidden:   false,
				MaxDepth:        5,
				ExpandFolders:   false,
			})
		}()
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

// Folder-specific service methods

// CreateFolder creates a new folder
func (s *Service) CreateFolder(ctx context.Context, input *dto.CreateFolderInput) (*models.Route, error) {
	// Check if folder ID already exists
	existing, _ := s.repository.GetRouteByRouteID(ctx, input.RouteID)
	if existing != nil {
		return nil, fmt.Errorf("folder with ID %s already exists", input.RouteID)
	}

	// Validate depth limits
	depth := 0
	if input.ParentID != nil && *input.ParentID != "" {
		valid, newDepth, err := s.repository.ValidateDepth(ctx, *input.ParentID, models.MaxFolderDepth)
		if err != nil {
			return nil, fmt.Errorf("failed to validate depth: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("folder depth would exceed maximum of %d levels", models.MaxFolderDepth)
		}
		depth = newDepth
	}

	// Set default icon if not provided
	icon := input.Icon
	if icon == nil || *icon == "" {
		defaultIcon := models.DefaultFolderIcon
		icon = &defaultIcon
	}

	// Set default expanded state
	expanded := input.IsExpanded
	if expanded == nil {
		defaultExpanded := false
		expanded = &defaultExpanded
	}

	folder := &models.Route{
		RouteID:             input.RouteID,
		Path:                "/folder/" + input.RouteID, // Folders don't have real paths
		Component:           "Folder",                   // Special component for folders
		Name:                input.Name,
		Icon:                icon,
		Type:                models.RouteTypeFolder,
		ParentID:            input.ParentID,
		NavPosition:         input.NavPosition,
		NavOrder:            input.NavOrder,
		ShowInNav:           input.ShowInNav,
		RequiredPermissions: input.RequiredPermissions,
		RequiredGroups:      input.RequiredGroups,
		Title:               input.Name,
		Description:         input.Description,
		IsEnabled:           input.IsEnabled,
		LazyLoad:            false, // Folders are not lazy loaded
		IsFolder:            true,
		Depth:               depth,
		ChildrenCount:       0,
		IsExpanded:          expanded,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Build and set folder path
	folderPath, err := s.buildFolderPath(ctx, folder)
	if err != nil {
		return nil, fmt.Errorf("failed to build folder path: %w", err)
	}
	folder.FolderPath = folderPath

	id, err := s.repository.CreateRoute(ctx, folder)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	folder.ID = id

	// Update parent's children count if applicable
	if input.ParentID != nil && *input.ParentID != "" {
		s.repository.UpdateChildrenCount(ctx, *input.ParentID)
	}

	return folder, nil
}

// UpdateFolder updates an existing folder
func (s *Service) UpdateFolder(ctx context.Context, folderID string, input *dto.UpdateFolderInput) (*models.Route, error) {
	// Get existing folder
	folder, err := s.GetRouteByID(ctx, folderID)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}

	if !folder.IsFolder {
		return nil, fmt.Errorf("route %s is not a folder", folderID)
	}

	// Build update document
	updateDoc := bson.M{"updated_at": time.Now()}

	if input.Name != nil {
		updateDoc["name"] = *input.Name
		updateDoc["title"] = *input.Name // Keep title in sync with name for folders
	}
	if input.ParentID != nil {
		// Validate new parent doesn't create circular reference
		if err := s.validateNoCircularReference(ctx, folder.RouteID, *input.ParentID); err != nil {
			return nil, err
		}

		// Validate depth limits
		if *input.ParentID != "" {
			valid, newDepth, err := s.repository.ValidateDepth(ctx, *input.ParentID, models.MaxFolderDepth)
			if err != nil {
				return nil, fmt.Errorf("failed to validate depth: %w", err)
			}
			if !valid {
				return nil, fmt.Errorf("folder depth would exceed maximum of %d levels", models.MaxFolderDepth)
			}
			updateDoc["depth"] = newDepth
		} else {
			updateDoc["depth"] = 0 // Moving to root
		}

		oldParentID := folder.ParentID
		updateDoc["parent_id"] = *input.ParentID

		// Update children count for old and new parents
		if oldParentID != nil && *oldParentID != "" {
			go s.repository.UpdateChildrenCount(ctx, *oldParentID)
		}
		if *input.ParentID != "" {
			go s.repository.UpdateChildrenCount(ctx, *input.ParentID)
		}
	}
	if input.Icon != nil {
		updateDoc["icon"] = *input.Icon
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
	if input.Description != nil {
		updateDoc["description"] = *input.Description
	}
	if input.IsExpanded != nil {
		updateDoc["is_expanded"] = *input.IsExpanded
	}
	if input.IsEnabled != nil {
		updateDoc["is_enabled"] = *input.IsEnabled
	}
	if len(input.RequiredPermissions) > 0 {
		updateDoc["required_permissions"] = input.RequiredPermissions
	}
	if len(input.RequiredGroups) > 0 {
		updateDoc["required_groups"] = input.RequiredGroups
	}

	err = s.repository.UpdateRoute(ctx, folder.ID, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update folder: %w", err)
	}

	// Rebuild folder path if name or parent changed
	if input.Name != nil || input.ParentID != nil {
		updatedFolder, _ := s.repository.GetRouteByID(ctx, folder.ID)
		if updatedFolder != nil {
			folderPath, _ := s.buildFolderPath(ctx, updatedFolder)
			s.repository.UpdateFolderPath(ctx, updatedFolder.RouteID, folderPath)
		}
	}

	return s.repository.GetRouteByID(ctx, folder.ID)
}

// MoveToFolder moves a route or folder to a different parent folder
func (s *Service) MoveToFolder(ctx context.Context, itemID string, input *dto.MoveFolderInput) (*dto.MoveFolderResponse, error) {
	// Get the item being moved
	item, err := s.repository.GetRouteByRouteID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}

	oldParentID := ""
	if item.ParentID != nil {
		oldParentID = *item.ParentID
	}

	newParentID := ""
	if input.NewParentID != nil {
		newParentID = *input.NewParentID
	}

	// Validate no circular reference if moving a folder
	if item.IsFolder && newParentID != "" {
		if err := s.validateNoCircularReference(ctx, itemID, newParentID); err != nil {
			return nil, err
		}
	}

	// Validate depth limits
	newDepth := 0
	if newParentID != "" {
		valid, depth, err := s.repository.ValidateDepth(ctx, newParentID, models.MaxFolderDepth)
		if err != nil {
			return nil, fmt.Errorf("failed to validate depth: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("move would exceed maximum depth of %d levels", models.MaxFolderDepth)
		}
		newDepth = depth
	}

	// Perform the move
	err = s.repository.MoveRouteToFolder(ctx, itemID, input.NewParentID, input.NavOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to move item: %w", err)
	}

	// Update depth for the moved item and all its children
	if err := s.updateItemDepth(ctx, itemID, newDepth); err != nil {
		return nil, fmt.Errorf("failed to update item depth: %w", err)
	}

	// Update children count for old and new parents
	if oldParentID != "" {
		s.repository.UpdateChildrenCount(ctx, oldParentID)
	}
	if newParentID != "" {
		s.repository.UpdateChildrenCount(ctx, newParentID)
	}

	// Build new folder path
	updatedItem, _ := s.repository.GetRouteByRouteID(ctx, itemID)
	newPath := ""
	if updatedItem != nil {
		newPath, _ = s.buildFolderPath(ctx, updatedItem)
		s.repository.UpdateFolderPath(ctx, itemID, newPath)
	}

	return &dto.MoveFolderResponse{
		Message:   fmt.Sprintf("Successfully moved %s", item.Name),
		ItemMoved: itemID,
		OldParent: oldParentID,
		NewParent: newParentID,
		NewPath:   newPath,
	}, nil
}

// GetFolderChildren gets children of a folder
func (s *Service) GetFolderChildren(ctx context.Context, input *dto.FolderChildrenInput) (*dto.FolderChildrenResponse, error) {
	// Get folder info
	folder, err := s.repository.GetRouteByRouteID(ctx, input.FolderID)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}

	if !folder.IsFolder {
		return nil, fmt.Errorf("route %s is not a folder", input.FolderID)
	}

	// Get direct children
	children, err := s.repository.GetFolderChildren(ctx, input.FolderID, input.IncludeDisabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}

	// If recursive, get all descendants
	if input.Recursive && input.MaxDepth > 1 {
		allChildren, err := s.getFolderDescendants(ctx, input.FolderID, input.MaxDepth, input.IncludeDisabled)
		if err != nil {
			return nil, fmt.Errorf("failed to get descendants: %w", err)
		}
		children = allChildren
	}

	// Check if folder has subfolders
	hasSubfolders := false
	for _, child := range children {
		if child.IsFolder {
			hasSubfolders = true
			break
		}
	}

	folderPath, _ := s.buildFolderPath(ctx, folder)

	return &dto.FolderChildrenResponse{
		FolderID:      input.FolderID,
		FolderName:    folder.Name,
		FolderPath:    folderPath,
		Children:      children,
		TotalChildren: len(children),
		Depth:         folder.Depth,
		HasSubfolders: hasSubfolders,
	}, nil
}

// BulkMove moves multiple items to a target folder
func (s *Service) BulkMove(ctx context.Context, input *dto.BulkMoveInput) (*dto.BulkMoveResponse, error) {
	var moved []string
	var failed []string
	var errors []string

	targetFolderID := ""
	if input.TargetFolderID != nil {
		targetFolderID = *input.TargetFolderID
	}

	// Validate target folder exists if specified
	if targetFolderID != "" {
		target, err := s.repository.GetRouteByRouteID(ctx, targetFolderID)
		if err != nil {
			return nil, fmt.Errorf("target folder not found: %w", err)
		}
		if !target.IsFolder {
			return nil, fmt.Errorf("target %s is not a folder", targetFolderID)
		}
	}

	// Move each item
	for i, itemID := range input.ItemIDs {
		moveInput := &dto.MoveFolderInput{
			NewParentID: input.TargetFolderID,
			NavOrder:    &i, // Use index as default order
		}

		_, err := s.MoveToFolder(ctx, itemID, moveInput)
		if err != nil {
			failed = append(failed, itemID)
			errors = append(errors, fmt.Sprintf("%s: %v", itemID, err))
		} else {
			moved = append(moved, itemID)
		}
	}

	return &dto.BulkMoveResponse{
		TargetFolder: targetFolderID,
		ItemsMoved:   moved,
		ItemsFailed:  failed,
		TotalMoved:   len(moved),
		TotalFailed:  len(failed),
		Errors:       errors,
		Message:      fmt.Sprintf("Moved %d items, %d failed", len(moved), len(failed)),
	}, nil
}

// GetFolderStats returns folder usage statistics
func (s *Service) GetFolderStats(ctx context.Context) (*models.FolderStats, error) {
	return s.repository.GetFolderStats(ctx)
}

// Helper methods

// buildFolderPath builds the full path for a folder
func (s *Service) buildFolderPath(ctx context.Context, folder *models.Route) (string, error) {
	return s.repository.GetFolderPath(ctx, folder.RouteID)
}

// buildFolderPathForRoute builds folder path for a route that may not be saved yet
func (s *Service) buildFolderPathForRoute(ctx context.Context, route *models.Route) (string, error) {
	if route.ParentID == nil || *route.ParentID == "" {
		if route.IsFolder {
			return "/" + route.Name, nil
		}
		return "/", nil
	}

	parentPath, err := s.repository.GetFolderPath(ctx, *route.ParentID)
	if err != nil {
		return "", err
	}

	if route.IsFolder {
		if parentPath == "/" {
			return "/" + route.Name, nil
		}
		return parentPath + "/" + route.Name, nil
	}

	return parentPath, nil
}

// validateNoCircularReference ensures moving a folder won't create circular references
func (s *Service) validateNoCircularReference(ctx context.Context, folderID, newParentID string) error {
	if folderID == newParentID {
		return fmt.Errorf("folder cannot be its own parent")
	}

	// Check if newParentID is a descendant of folderID
	current := newParentID
	visited := make(map[string]bool)

	for current != "" {
		if visited[current] {
			break // Circular reference in existing data, but not caused by this move
		}
		visited[current] = true

		if current == folderID {
			return fmt.Errorf("cannot move folder into its own descendant")
		}

		parent, err := s.repository.GetRouteByRouteID(ctx, current)
		if err != nil {
			break // Parent not found, safe to move
		}

		if parent.ParentID == nil {
			break
		}
		current = *parent.ParentID
	}

	return nil
}

// updateItemDepth recursively updates depth for item and all children
func (s *Service) updateItemDepth(ctx context.Context, itemID string, newDepth int) error {
	// Update the item itself
	_, err := s.repository.collection.UpdateOne(
		ctx,
		bson.M{"route_id": itemID},
		bson.M{"$set": bson.M{"depth": newDepth, "updated_at": time.Now()}},
	)
	if err != nil {
		return err
	}

	// Get children and update their depth recursively
	children, err := s.repository.GetFolderChildren(ctx, itemID, true)
	if err != nil {
		return nil // No children or error getting them, continue
	}

	for _, child := range children {
		if err := s.updateItemDepth(ctx, child.RouteID, newDepth+1); err != nil {
			continue // Continue with other children even if one fails
		}
	}

	return nil
}

// getFolderDescendants gets all descendants of a folder up to maxDepth
func (s *Service) getFolderDescendants(ctx context.Context, folderID string, maxDepth int, includeDisabled bool) ([]models.Route, error) {
	var allDescendants []models.Route
	currentLevel := []string{folderID}

	for depth := 1; depth <= maxDepth && len(currentLevel) > 0; depth++ {
		var nextLevel []string

		for _, parentID := range currentLevel {
			children, err := s.repository.GetFolderChildren(ctx, parentID, includeDisabled)
			if err != nil {
				continue
			}

			for _, child := range children {
				allDescendants = append(allDescendants, child)
				if child.IsFolder {
					nextLevel = append(nextLevel, child.RouteID)
				}
			}
		}

		currentLevel = nextLevel
	}

	return allDescendants, nil
}

// sortRoutesHierarchicallyForAdmin sorts routes for hierarchical display in admin interface
func (s *Service) sortRoutesHierarchicallyForAdmin(routes []models.Route) []models.Route {
	if len(routes) == 0 {
		return routes
	}

	// Build parent-child relationships
	parentChildMap := make(map[string][]models.Route)
	rootRoutes := []models.Route{}

	for _, route := range routes {
		if route.ParentID == nil || *route.ParentID == "" {
			rootRoutes = append(rootRoutes, route)
		} else {
			parentChildMap[*route.ParentID] = append(parentChildMap[*route.ParentID], route)
		}
	}

	// Sort root routes by nav_order
	rootRoutes = s.sortRoutesByOrder(rootRoutes)

	// Sort children for each parent
	for parentID, children := range parentChildMap {
		parentChildMap[parentID] = s.sortRoutesByOrder(children)
	}

	// Build final hierarchical list for admin display
	var result []models.Route

	var addRouteAndChildren func(models.Route)
	addRouteAndChildren = func(route models.Route) {
		// Add the route itself
		result = append(result, route)

		// Add its children recursively
		if children, exists := parentChildMap[route.RouteID]; exists {
			for _, child := range children {
				addRouteAndChildren(child)
			}
		}
	}

	// Process all root routes and their children
	for _, root := range rootRoutes {
		addRouteAndChildren(root)
	}

	return result
}
