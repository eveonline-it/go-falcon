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
type GroupServiceInterface interface {
	// GetUserGroups gets all unique groups that any character belonging to a user_id belongs to
	GetUserGroups(ctx context.Context, userID string) ([]GroupInfo, error)
	// GetCharacterGroups gets all groups a character belongs to
	GetCharacterGroups(ctx context.Context, characterID int64) ([]GroupInfo, error)
}

// GroupInfo represents group information returned by the groups service
type GroupInfo struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	SystemName  *string `json:"system_name,omitempty"`
	EVEEntityID *int64  `json:"eve_entity_id,omitempty"`
	IsActive    bool    `json:"is_active"`
}

// CorporationServiceInterface defines the interface for corporation service operations
type CorporationServiceInterface interface {
	// GetCorporationInfo gets basic corporation information by ID
	GetCorporationInfo(ctx context.Context, corporationID int) (*CorporationInfo, error)
}

// CorporationInfo represents corporation information returned by the corporation service
type CorporationInfo struct {
	CorporationID int    `json:"corporation_id"`
	Name          string `json:"name"`
	Ticker        string `json:"ticker"`
}

// SiteSettingsServiceInterface defines the interface for site settings service operations
type SiteSettingsServiceInterface interface {
	// GetManagedCorporations gets the list of managed corporations with enabled status
	GetManagedCorporations(ctx context.Context) ([]ManagedCorporation, error)
	// GetManagedAlliances gets the list of managed alliances with enabled status
	GetManagedAlliances(ctx context.Context) ([]ManagedAlliance, error)
}

// ManagedCorporation represents a managed corporation from site settings
type ManagedCorporation struct {
	CorporationID int64  `json:"corporation_id"`
	Name          string `json:"name"`
	Ticker        string `json:"ticker"`
	Enabled       bool   `json:"enabled"`
	Position      int    `json:"position"`
}

// ManagedAlliance represents a managed alliance from site settings
type ManagedAlliance struct {
	AllianceID int64  `json:"alliance_id"`
	Name       string `json:"name"`
	Ticker     string `json:"ticker"`
	Enabled    bool   `json:"enabled"`
	Position   int    `json:"position"`
}

// GroupsServiceAdapter adapts the real groups service to our interface
type GroupsServiceAdapter struct {
	// We'll store a function interface instead of a concrete service to avoid import cycles
	getUserGroupsFunc      func(ctx context.Context, userID string) ([]GroupInfo, error)
	getCharacterGroupsFunc func(ctx context.Context, characterID int64) ([]GroupInfo, error)
}

// NewGroupsServiceAdapter creates an adapter that bridges the groups service
func NewGroupsServiceAdapter(
	getUserGroupsFunc func(ctx context.Context, userID string) ([]GroupInfo, error),
	getCharacterGroupsFunc func(ctx context.Context, characterID int64) ([]GroupInfo, error),
) GroupServiceInterface {
	return &GroupsServiceAdapter{
		getUserGroupsFunc:      getUserGroupsFunc,
		getCharacterGroupsFunc: getCharacterGroupsFunc,
	}
}

// GetUserGroups implements GroupServiceInterface
func (a *GroupsServiceAdapter) GetUserGroups(ctx context.Context, userID string) ([]GroupInfo, error) {
	if a.getUserGroupsFunc == nil {
		return []GroupInfo{}, nil // Gracefully handle nil function
	}
	return a.getUserGroupsFunc(ctx, userID)
}

// GetCharacterGroups implements GroupServiceInterface
func (a *GroupsServiceAdapter) GetCharacterGroups(ctx context.Context, characterID int64) ([]GroupInfo, error) {
	if a.getCharacterGroupsFunc == nil {
		return []GroupInfo{}, nil // Gracefully handle nil function
	}
	return a.getCharacterGroupsFunc(ctx, characterID)
}

// CorporationServiceAdapter adapts the real corporation service to our interface
type CorporationServiceAdapter struct {
	getCorporationInfoFunc func(ctx context.Context, corporationID int) (*CorporationInfo, error)
}

// NewCorporationServiceAdapter creates an adapter that bridges the corporation service
func NewCorporationServiceAdapter(
	getCorporationInfoFunc func(ctx context.Context, corporationID int) (*CorporationInfo, error),
) CorporationServiceInterface {
	return &CorporationServiceAdapter{
		getCorporationInfoFunc: getCorporationInfoFunc,
	}
}

// GetCorporationInfo implements CorporationServiceInterface
func (a *CorporationServiceAdapter) GetCorporationInfo(ctx context.Context, corporationID int) (*CorporationInfo, error) {
	if a.getCorporationInfoFunc == nil {
		return nil, fmt.Errorf("corporation service not available")
	}
	return a.getCorporationInfoFunc(ctx, corporationID)
}

// SiteSettingsServiceAdapter adapts the real site settings service to our interface
type SiteSettingsServiceAdapter struct {
	getManagedCorporationsFunc func(ctx context.Context) ([]ManagedCorporation, error)
	getManagedAlliancesFunc    func(ctx context.Context) ([]ManagedAlliance, error)
}

// NewSiteSettingsServiceAdapter creates an adapter that bridges the site settings service
func NewSiteSettingsServiceAdapter(
	getManagedCorporationsFunc func(ctx context.Context) ([]ManagedCorporation, error),
	getManagedAlliancesFunc func(ctx context.Context) ([]ManagedAlliance, error),
) SiteSettingsServiceInterface {
	return &SiteSettingsServiceAdapter{
		getManagedCorporationsFunc: getManagedCorporationsFunc,
		getManagedAlliancesFunc:    getManagedAlliancesFunc,
	}
}

// GetManagedCorporations implements SiteSettingsServiceInterface
func (a *SiteSettingsServiceAdapter) GetManagedCorporations(ctx context.Context) ([]ManagedCorporation, error) {
	if a.getManagedCorporationsFunc == nil {
		return []ManagedCorporation{}, nil // Gracefully handle nil function
	}
	return a.getManagedCorporationsFunc(ctx)
}

// GetManagedAlliances implements SiteSettingsServiceInterface
func (a *SiteSettingsServiceAdapter) GetManagedAlliances(ctx context.Context) ([]ManagedAlliance, error) {
	if a.getManagedAlliancesFunc == nil {
		return []ManagedAlliance{}, nil // Gracefully handle nil function
	}
	return a.getManagedAlliancesFunc(ctx)
}

// Service handles sitemap business logic
type Service struct {
	db                  *mongo.Database
	repository          *Repository
	permissionManager   *permissions.PermissionManager
	groupService        GroupServiceInterface
	corporationService  CorporationServiceInterface
	siteSettingsService SiteSettingsServiceInterface
}

// NewService creates a new sitemap service
func NewService(db *mongo.Database, permissionManager *permissions.PermissionManager, groupService GroupServiceInterface, corporationService CorporationServiceInterface, siteSettingsService SiteSettingsServiceInterface) *Service {
	return &Service{
		db:                  db,
		repository:          NewRepository(db),
		permissionManager:   permissionManager,
		groupService:        groupService,
		corporationService:  corporationService,
		siteSettingsService: siteSettingsService,
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

// sortRoutesByPositionAndOrder sorts routes by nav_order globally (not grouped by position)
func (s *Service) sortRoutesByPositionAndOrder(routes []models.Route) []models.Route {
	// Sort by nav_order globally across all positions
	// This ensures the navigation respects the intended order regardless of nav_position
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[i].NavOrder > routes[j].NavOrder {
				routes[i], routes[j] = routes[j], routes[i]
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

	// Extract user permissions and groups if we have a user context
	userPermissions := []string{}
	userGroups := []string{}

	// For authenticated requests, we need to pass user information to extract permissions/groups
	// This will be updated when we add user context to the method signature

	return &models.SitemapResponse{
		Routes:          routeConfigs,
		Navigation:      navigation,
		UserPermissions: userPermissions,
		UserGroups:      userGroups,
		Features:        make(map[string]bool),
	}, nil
}

// GetUserRoutesWithAuth returns user-specific sitemap with permission and group filtering
func (s *Service) GetUserRoutesWithAuth(ctx context.Context, input *dto.GetUserRoutesInput, userID string, characterID int64) (*models.SitemapResponse, error) {
	// Build filter based on parameters
	filter := bson.M{}
	if !input.IncludeDisabled {
		filter["is_enabled"] = true
	}

	routes, err := s.repository.GetRoutes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	// Generate dynamic corporation dashboard routes
	corporationRoutes, err := s.generateCorporationRoutes(ctx, characterID)
	if err != nil {
		// Log error but don't fail the entire request
		fmt.Printf("⚠️  [DEBUG] Warning: Failed to generate corporation routes: %v\n", err)
		corporationRoutes = []models.Route{}
	}

	// Generate dynamic alliance dashboard routes
	allianceRoutes, err := s.generateAllianceRoutes(ctx, characterID)
	if err != nil {
		// Log error but don't fail the entire request
		fmt.Printf("⚠️  [DEBUG] Warning: Failed to generate alliance routes: %v\n", err)
		allianceRoutes = []models.Route{}
	}

	// Merge static and dynamic routes
	allRoutes := append(routes, corporationRoutes...)
	allRoutes = append(allRoutes, allianceRoutes...)
	fmt.Printf("📋 [DEBUG] Merged %d static + %d corp + %d alliance = %d total routes\n", len(routes), len(corporationRoutes), len(allianceRoutes), len(allRoutes))

	// Extract user permissions and groups
	userPermissions, userGroups, err := s.extractUserContext(ctx, userID, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract user context: %w", err)
	}

	// Check if user is super admin
	isSuperAdmin := false
	for _, group := range userGroups {
		if group == "Super Administrator" {
			isSuperAdmin = true
			break
		}
	}

	// Filter routes based on user access
	accessibleRoutes := []models.Route{}
	for _, route := range allRoutes {
		if s.checkRouteAccess(route, userPermissions, userGroups, isSuperAdmin) {
			accessibleRoutes = append(accessibleRoutes, route)
		}
	}

	// Build route configs with hierarchy from filtered routes
	routeConfigs := s.buildRouteConfigs(accessibleRoutes)

	// Build navigation with folder support from filtered routes
	navigation := s.buildHierarchicalNavigation(accessibleRoutes, input.IncludeHidden, input.MaxDepth)

	// Apply folder expansion settings
	if input.ExpandFolders {
		s.expandAllFolders(navigation)
	}

	return &models.SitemapResponse{
		Routes:          routeConfigs,
		Navigation:      navigation,
		UserPermissions: userPermissions,
		UserGroups:      userGroups,
		Features:        s.getUserFeatures(ctx, characterID),
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

// generateCorporationRoutes creates dynamic corporation dashboard routes based on user access
func (s *Service) generateCorporationRoutes(ctx context.Context, characterID int64) ([]models.Route, error) {
	var corporationRoutes []models.Route

	// Get managed corporations from site settings
	fmt.Printf("🔍 [DEBUG] Generating corporation routes for character %d\n", characterID)
	managedCorps, err := s.siteSettingsService.GetManagedCorporations(ctx)
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to get managed corporations: %v\n", err)
		return nil, fmt.Errorf("failed to get managed corporations: %w", err)
	}

	fmt.Printf("📊 [DEBUG] Found %d managed corporations total\n", len(managedCorps))

	// Filter to only enabled corporations
	var enabledCorps []ManagedCorporation
	for _, corp := range managedCorps {
		if corp.Enabled {
			enabledCorps = append(enabledCorps, corp)
			fmt.Printf("✅ [DEBUG] Enabled corporation: %s [%s] (ID: %d)\n", corp.Name, corp.Ticker, corp.CorporationID)
		} else {
			fmt.Printf("❌ [DEBUG] Disabled corporation: %s [%s] (ID: %d)\n", corp.Name, corp.Ticker, corp.CorporationID)
		}
	}

	fmt.Printf("🎯 [DEBUG] Found %d enabled corporations\n", len(enabledCorps))

	if len(enabledCorps) == 0 {
		fmt.Printf("⚠️  [DEBUG] No enabled corporations found - returning empty routes\n")
		return corporationRoutes, nil // No enabled corporations
	}

	// Get user's character corporation ID (this would need to be passed from auth context)
	// For now, we'll generate routes for all enabled corporations and let the access control filter them
	// TODO: In the future, we could add logic to determine which corporations the user has access to

	// Generate dynamic routes for each enabled corporation
	for _, corp := range enabledCorps {
		route := models.Route{
			ID:          primitive.NewObjectID(),
			RouteID:     fmt.Sprintf("corp-dashboard-%d", corp.CorporationID),
			Path:        fmt.Sprintf("/corporations/%d/dashboard", corp.CorporationID),
			Component:   "CorporationDashboard",
			Name:        corp.Name,
			Type:        models.RouteTypeProtected, // Requires specific permissions
			ParentID:    stringPtr("folder-corporation"),
			NavPosition: models.NavMain,
			NavOrder:    100 + corp.Position, // Start from 100 to avoid conflicts with static routes
			ShowInNav:   true,
			Title:       fmt.Sprintf("%s Corporation Dashboard", corp.Name),
			Description: stringPtr(fmt.Sprintf("Dashboard for %s [%s] corporation", corp.Name, corp.Ticker)),
			Group:       stringPtr("corporation"),
			LazyLoad:    true,
			Exact:       false,
			NewTab:      false,
			IsEnabled:   true,
			Icon:        stringPtr("building"),

			// Corporation-specific permissions: user must be in the corporation OR have corporation management permissions
			RequiredPermissions: []string{},                                    // We'll handle access logic via route access checker
			RequiredGroups:      []string{fmt.Sprintf("corp_%s", corp.Ticker)}, // Corporation-specific group

			// Metadata
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),

			// Add corporation-specific props
			Props: map[string]interface{}{
				"corporationId":   corp.CorporationID,
				"corporationName": corp.Name,
				"ticker":          corp.Ticker,
				"isDynamic":       true,
			},
		}

		corporationRoutes = append(corporationRoutes, route)
		fmt.Printf("🚀 [DEBUG] Generated route: %s -> %s\n", route.RouteID, route.Path)
	}

	fmt.Printf("✅ [DEBUG] Successfully generated %d corporation dashboard routes\n", len(corporationRoutes))
	return corporationRoutes, nil
}

// stringPtr returns a pointer to the given string (helper function)
func stringPtr(s string) *string {
	return &s
}

// generateAllianceRoutes creates dynamic alliance dashboard routes based on user access
func (s *Service) generateAllianceRoutes(ctx context.Context, characterID int64) ([]models.Route, error) {
	var allianceRoutes []models.Route

	// Get managed alliances from site settings
	fmt.Printf("🔍 [DEBUG] Generating alliance routes for character %d\n", characterID)
	managedAlliances, err := s.siteSettingsService.GetManagedAlliances(ctx)
	if err != nil {
		fmt.Printf("❌ [DEBUG] Failed to get managed alliances: %v\n", err)
		return nil, fmt.Errorf("failed to get managed alliances: %w", err)
	}

	fmt.Printf("📊 [DEBUG] Found %d managed alliances total\n", len(managedAlliances))

	// Filter to only enabled alliances
	var enabledAlliances []ManagedAlliance
	for _, alliance := range managedAlliances {
		if alliance.Enabled {
			enabledAlliances = append(enabledAlliances, alliance)
			fmt.Printf("✅ [DEBUG] Enabled alliance: %s [%s] (ID: %d)\n", alliance.Name, alliance.Ticker, alliance.AllianceID)
		} else {
			fmt.Printf("❌ [DEBUG] Disabled alliance: %s [%s] (ID: %d)\n", alliance.Name, alliance.Ticker, alliance.AllianceID)
		}
	}

	fmt.Printf("🎯 [DEBUG] Found %d enabled alliances\n", len(enabledAlliances))

	if len(enabledAlliances) == 0 {
		fmt.Printf("⚠️  [DEBUG] No enabled alliances found - returning empty routes\n")
		return allianceRoutes, nil // No enabled alliances
	}

	// Get user's character alliance ID (this would need to be passed from auth context)
	// For now, we'll generate routes for all enabled alliances and let the access control filter them
	// TODO: In the future, we could add logic to determine which alliances the user has access to

	// Generate dynamic routes for each enabled alliance
	for _, alliance := range enabledAlliances {
		route := models.Route{
			ID:          primitive.NewObjectID(),
			RouteID:     fmt.Sprintf("alliance-dashboard-%d", alliance.AllianceID),
			Path:        fmt.Sprintf("/alliances/%d/dashboard", alliance.AllianceID),
			Component:   "AllianceDashboard",
			Name:        alliance.Name,
			Type:        models.RouteTypeProtected, // Requires specific permissions
			ParentID:    stringPtr("folder-alliance"),
			NavPosition: models.NavMain,
			NavOrder:    200 + alliance.Position, // Start from 200 to avoid conflicts with static routes and corporation routes
			ShowInNav:   true,
			Title:       fmt.Sprintf("%s Alliance Dashboard", alliance.Name),
			Description: stringPtr(fmt.Sprintf("Dashboard for %s [%s] alliance", alliance.Name, alliance.Ticker)),
			Group:       stringPtr("alliance"),
			LazyLoad:    true,
			Exact:       false,
			NewTab:      false,
			IsEnabled:   true,
			Icon:        stringPtr("users"),

			// Alliance-specific permissions: user must be in the alliance OR have alliance management permissions
			RequiredPermissions: []string{},                                            // We'll handle access logic via route access checker
			RequiredGroups:      []string{fmt.Sprintf("alliance_%s", alliance.Ticker)}, // Alliance-specific group

			// Metadata
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),

			// Add alliance-specific props
			Props: map[string]interface{}{
				"allianceId":   alliance.AllianceID,
				"allianceName": alliance.Name,
				"ticker":       alliance.Ticker,
				"isDynamic":    true,
			},
		}

		allianceRoutes = append(allianceRoutes, route)
		fmt.Printf("🚀 [DEBUG] Generated route: %s -> %s\n", route.RouteID, route.Path)
	}

	fmt.Printf("✅ [DEBUG] Successfully generated %d alliance dashboard routes\n", len(allianceRoutes))
	return allianceRoutes, nil
}

// extractUserContext extracts user permissions and groups from authentication context
func (s *Service) extractUserContext(ctx context.Context, userID string, characterID int64) ([]string, []string, error) {
	userPermissions := []string{}
	userGroups := []string{}

	// Extract user groups from groups service if available
	if s.groupService != nil {
		groups, err := s.groupService.GetUserGroups(ctx, userID)
		if err != nil {
			// Log error but don't fail - gracefully degrade to no group restrictions
			fmt.Printf("Warning: Failed to get user groups for user %s: %v\n", userID, err)
		} else {
			// Convert groups to string array
			for _, group := range groups {
				if group.IsActive {
					userGroups = append(userGroups, group.Name)
				}
			}
		}
	}

	// Extract user permissions from permission manager if available
	if s.permissionManager != nil {
		// Get all available permissions and check which ones the user has
		// Note: This could be expensive - consider caching in production
		allPermissions := s.permissionManager.GetAllPermissions()
		for permissionID := range allPermissions {
			hasPermission, err := s.permissionManager.HasPermission(ctx, characterID, permissionID)
			if err != nil {
				// Log error but continue checking other permissions
				fmt.Printf("Warning: Failed to check permission %s for character %d: %v\n", permissionID, characterID, err)
				continue
			}
			if hasPermission {
				userPermissions = append(userPermissions, permissionID)
			}
		}
	}

	return userPermissions, userGroups, nil
}

// checkRouteAccess checks if user has access to a route based on permissions and groups
func (s *Service) checkRouteAccess(route models.Route, userPermissions, userGroups []string, isSuperAdmin bool) bool {
	// Super admins have access to everything
	if isSuperAdmin {
		return true
	}

	// Check if route has any restrictions first
	hasPermissionRestrictions := len(route.RequiredPermissions) > 0
	hasGroupRestrictions := len(route.RequiredGroups) > 0

	// Public routes are accessible to everyone ONLY if they have no group/permission restrictions
	if route.Type == models.RouteTypePublic && !hasPermissionRestrictions && !hasGroupRestrictions {
		return true
	}

	// If route has no restrictions, it's accessible to authenticated users
	if !hasPermissionRestrictions && !hasGroupRestrictions {
		return true // No restrictions
	}

	// Check group restrictions (OR logic - user needs ANY of the required groups)
	if hasGroupRestrictions {
		for _, requiredGroup := range route.RequiredGroups {
			for _, userGroup := range userGroups {
				if userGroup == requiredGroup {
					return true // User has a required group
				}
			}
		}
	}

	// Check permission restrictions (AND logic - user needs ALL required permissions)
	if hasPermissionRestrictions {
		for _, requiredPerm := range route.RequiredPermissions {
			hasPermission := false
			for _, userPerm := range userPermissions {
				if userPerm == requiredPerm {
					hasPermission = true
					break
				}
			}
			if !hasPermission {
				return false // User lacks a required permission
			}
		}
		// If we get here, user has all required permissions
		return true
	}

	// If only group restrictions and user has no matching groups, deny access
	if hasGroupRestrictions && !hasPermissionRestrictions {
		return false
	}

	return false
}

// CreateRoute creates a new route
func (s *Service) CreateRoute(ctx context.Context, input *dto.CreateRouteInput) (*models.Route, error) {
	// Check if route ID already exists
	existing, _ := s.repository.GetRouteByRouteID(ctx, input.Body.RouteID)
	if existing != nil {
		return nil, fmt.Errorf("route with ID %s already exists", input.Body.RouteID)
	}

	// Calculate depth based on parent
	depth := 0
	if input.Body.ParentID != nil && *input.Body.ParentID != "" {
		parent, err := s.repository.GetRouteByRouteID(ctx, *input.Body.ParentID)
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
	isFolder := input.Body.Type == models.RouteTypeFolder

	// Validate is_folder if provided by client
	if input.Body.IsFolder != nil {
		if *input.Body.IsFolder != isFolder {
			return nil, fmt.Errorf("is_folder value (%v) doesn't match route type (%s). Folder routes must have type='folder'", *input.Body.IsFolder, input.Body.Type)
		}
	}

	route := &models.Route{
		RouteID:             input.Body.RouteID,
		Path:                input.Body.Path,
		Component:           input.Body.Component,
		Name:                input.Body.Name,
		Icon:                input.Body.Icon,
		Type:                input.Body.Type,
		ParentID:            input.Body.ParentID,
		NavPosition:         input.Body.NavPosition,
		NavOrder:            input.Body.NavOrder,
		ShowInNav:           input.Body.ShowInNav,
		RequiredPermissions: input.Body.RequiredPermissions,
		RequiredGroups:      input.Body.RequiredGroups,
		Title:               input.Body.Title,
		Description:         input.Body.Description,
		Keywords:            input.Body.Keywords,
		Group:               input.Body.Group,
		FeatureFlags:        input.Body.FeatureFlags,
		IsEnabled:           input.Body.IsEnabled,
		Props:               input.Body.Props,
		LazyLoad:            input.Body.LazyLoad,
		Exact:               input.Body.Exact,
		NewTab:              input.Body.NewTab,
		BadgeType:           input.Body.BadgeType,
		BadgeText:           input.Body.BadgeText,

		// Folder-specific fields
		IsFolder:      isFolder,
		Depth:         depth,
		ChildrenCount: 0,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Build folder path
	if isFolder || input.Body.ParentID != nil {
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
	if input.Body.ParentID != nil && *input.Body.ParentID != "" {
		s.repository.UpdateChildrenCount(ctx, *input.Body.ParentID)
	}

	return route, nil
}

// UpdateRoute updates an existing route
func (s *Service) UpdateRoute(ctx context.Context, routeID string, body *dto.UpdateRouteBody) (*models.Route, error) {
	// Get existing route (handles both ObjectID and route_id)
	route, err := s.GetRouteByID(ctx, routeID)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}

	// Update fields if provided
	updateDoc := bson.M{"updated_at": time.Now()}

	// Core fields
	if body.Name != nil {
		updateDoc["name"] = *body.Name
	}
	if body.Path != nil {
		updateDoc["path"] = *body.Path
	}
	if body.Component != nil {
		updateDoc["component"] = *body.Component
	}
	if body.Icon != nil {
		updateDoc["icon"] = *body.Icon
	}
	if body.Type != nil {
		updateDoc["type"] = *body.Type
	}
	if body.ParentID != nil {
		updateDoc["parent_id"] = *body.ParentID
		// Rebuild folder path when parent changes
		if isFolder := route.Type == models.RouteTypeFolder; isFolder || *body.ParentID != "" {
			route.ParentID = body.ParentID
			folderPath, err := s.buildFolderPathForRoute(ctx, route)
			if err == nil {
				updateDoc["folder_path"] = folderPath
			}
		}
	}

	// Navigation fields
	if body.NavPosition != nil {
		updateDoc["nav_position"] = *body.NavPosition
	}
	if body.NavOrder != nil {
		updateDoc["nav_order"] = *body.NavOrder
	}
	if body.ShowInNav != nil {
		updateDoc["show_in_nav"] = *body.ShowInNav
	}

	// Permissions
	if body.RequiredPermissions != nil {
		updateDoc["required_permissions"] = body.RequiredPermissions
	}
	if body.RequiredGroups != nil {
		updateDoc["required_groups"] = body.RequiredGroups
	}

	// Metadata
	if body.Title != nil {
		updateDoc["title"] = *body.Title
	}
	if body.Description != nil {
		updateDoc["description"] = *body.Description
	}
	if body.Keywords != nil {
		updateDoc["keywords"] = body.Keywords
	}
	if body.Group != nil {
		updateDoc["group"] = *body.Group
	}

	// Feature flags
	if body.FeatureFlags != nil {
		updateDoc["feature_flags"] = body.FeatureFlags
	}
	if body.IsEnabled != nil {
		updateDoc["is_enabled"] = *body.IsEnabled
	}

	// React-specific
	if body.Props != nil {
		updateDoc["props"] = body.Props
	}
	if body.LazyLoad != nil {
		updateDoc["lazy_load"] = *body.LazyLoad
	}
	if body.Exact != nil {
		updateDoc["exact"] = *body.Exact
	}
	if body.NewTab != nil {
		updateDoc["newtab"] = *body.NewTab
	}

	// Badge
	if body.BadgeType != nil {
		updateDoc["badge_type"] = *body.BadgeType
	}
	if body.BadgeText != nil {
		updateDoc["badge_text"] = *body.BadgeText
	}

	err = s.repository.UpdateRoute(ctx, route.ID, updateDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	// Update parent's children count if parent changed
	if body.ParentID != nil && route.ParentID != nil && *route.ParentID != *body.ParentID {
		// Update old parent
		if *route.ParentID != "" {
			s.repository.UpdateChildrenCount(ctx, *route.ParentID)
		}
		// Update new parent
		if *body.ParentID != "" {
			s.repository.UpdateChildrenCount(ctx, *body.ParentID)
		}
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
func (s *Service) GetRoutes(ctx context.Context, input *dto.ListRoutesInput) ([]models.Route, error) {
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

	opts := options.Find().SetSort(sortFields)

	routes, err := s.repository.GetRoutesWithOptions(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	// Apply hierarchical sorting only if requested (default)
	if input.Sort == "" || input.Sort == "hierarchical" {
		routes = s.sortRoutesHierarchicallyForAdmin(routes)
	}

	return routes, nil
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
