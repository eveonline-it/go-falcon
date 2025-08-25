package sitemap

import (
	"context"
	"log"
	"log/slog"

	"go-falcon/internal/sitemap/dto"
	"go-falcon/internal/sitemap/routes"
	"go-falcon/internal/sitemap/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// GroupServiceInterface defines the interface for group service operations
// For now, use interface{} to avoid interface mismatch - can be refined later
type GroupServiceInterface interface{}

// Module represents the sitemap module
type Module struct {
	*module.BaseModule
	service *services.Service
	routes  *routes.Routes
}

// NewModule creates a new sitemap module
func NewModule(mongodb *database.MongoDB, redis *database.Redis, permissionManager *permissions.PermissionManager, groupService GroupServiceInterface) (*Module, error) {
	// Create service with dependencies
	service := services.NewService(mongodb.Database, permissionManager, groupService)

	// Create routes
	moduleRoutes := routes.NewRoutes(service)

	// Create database indexes
	repository := services.NewRepository(mongodb.Database)
	ctx := context.Background()
	if err := repository.CreateIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to create sitemap indexes: %v", err)
	}

	return &Module{
		BaseModule: module.NewBaseModule("sitemap", mongodb, redis),
		service:    service,
		routes:     moduleRoutes,
	}, nil
}

// Routes implements the module interface for HTTP routes
func (m *Module) Routes(r chi.Router) {
	// Sitemap module uses unified Huma routes only
	// RegisterUnifiedRoutes should be called instead
	log.Printf("Sitemap module Routes() called - use RegisterUnifiedRoutes() for Huma v2 routes")
}

// RegisterUnifiedRoutes registers all module routes with the Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	basePath := "/sitemap"
	m.routes.RegisterUnifiedRoutes(api, basePath)
	log.Printf("🗺️  Sitemap module routes registered at %s", basePath)
}

// RegisterPermissions registers sitemap-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager *permissions.PermissionManager) error {
	sitemapPermissions := []permissions.Permission{
		{
			ID:          "sitemap:routes:view",
			Service:     "sitemap",
			Resource:    "routes",
			Action:      "view",
			Name:        "View Routes",
			Description: "View route configurations",
			Category:    "Sitemap Management",
		},
		{
			ID:          "sitemap:routes:manage",
			Service:     "sitemap",
			Resource:    "routes",
			Action:      "manage",
			Name:        "Manage Routes",
			Description: "Create, update, and delete route configurations",
			Category:    "Sitemap Management",
		},
		{
			ID:          "sitemap:navigation:manage",
			Service:     "sitemap",
			Resource:    "navigation",
			Action:      "manage",
			Name:        "Manage Navigation",
			Description: "Manage navigation structure and ordering",
			Category:    "Sitemap Management",
		},
		{
			ID:          "sitemap:admin:full",
			Service:     "sitemap",
			Resource:    "admin",
			Action:      "full",
			Name:        "Full Sitemap Administration",
			Description: "Complete administrative access to sitemap system",
			Category:    "System Administration",
		},
	}

	return permissionManager.RegisterServicePermissions(ctx, sitemapPermissions)
}

// GetService returns the sitemap service for external use
func (m *Module) GetService() *services.Service {
	return m.service
}

// StartBackgroundTasks starts any background processes for the sitemap module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting sitemap-specific background tasks")

	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)

	// Sitemap module doesn't need specific background tasks currently
	for {
		select {
		case <-ctx.Done():
			slog.Info("Sitemap background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Sitemap background tasks stopped")
			return
		default:
			// No specific background tasks for sitemap module currently
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}

// SeedDefaultRoutes populates the database with routes organized into 7 main categories
// This should be called during initial setup
func (m *Module) SeedDefaultRoutes(ctx context.Context) error {
	log.Printf("🌱 Seeding default routes with 7-category structure...")

	// Note: This would contain the default routes that match your React frontend
	// Located at ~/react-falcon (/home/tore/react-falcon)
	// These routes are based on the existing siteMaps.ts structure

	defaultRoutes := m.getDefaultRoutes()

	seeded := 0
	skipped := 0

	for _, route := range defaultRoutes {
		// Check if route already exists
		existing, _ := m.service.GetRouteByID(ctx, route.RouteID)
		if existing != nil {
			skipped++
			continue
		}

		// Create the route
		_, err := m.service.CreateRoute(ctx, &route)
		if err != nil {
			log.Printf("Failed to seed route %s: %v", route.RouteID, err)
			continue
		}
		seeded++
	}

	log.Printf("✅ Seeded %d routes, skipped %d existing routes", seeded, skipped)
	return nil
}

// getDefaultRoutes returns routes organized into 7 hardcoded main categories
// Based on the updated structure with fixed categories
func (m *Module) getDefaultRoutes() []dto.CreateRouteInput {
	return []dto.CreateRouteInput{
		// =============================================================================
		// 1. ADMINISTRATION CATEGORY
		// =============================================================================
		{
			RouteID:     "admin-users",
			Path:        "/admin/users",
			Component:   "UsersAdmin",
			Name:        "Users",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    1,
			ShowInNav:   true,
			Title:       "User Management",
			Group:       stringPtr("Administration"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("users"),
		},
		{
			RouteID:     "admin-groups",
			Path:        "/admin/groups",
			Component:   "GroupsAdmin",
			Name:        "Groups",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    2,
			ShowInNav:   true,
			Title:       "Group Management",
			Group:       stringPtr("Administration"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("users-cog"),
		},
		{
			RouteID:     "admin-permissions",
			Path:        "/admin/permissions",
			Component:   "PermissionsAdmin",
			Name:        "Permissions",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    3,
			ShowInNav:   true,
			Title:       "Permission Management",
			Group:       stringPtr("Administration"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("shield-alt"),
		},
		{
			RouteID:     "admin-scheduler",
			Path:        "/admin/scheduler",
			Component:   "SchedulerAdmin",
			Name:        "Scheduler",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    4,
			ShowInNav:   true,
			Title:       "Task Scheduler",
			Group:       stringPtr("Administration"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("clock"),
		},
		{
			RouteID:     "admin-sitemap",
			Path:        "/admin/sitemap",
			Component:   "SitemapAdmin",
			Name:        "Sitemap",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    5,
			ShowInNav:   true,
			Title:       "Sitemap Management",
			Group:       stringPtr("Administration"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("map"),
		},
		{
			RouteID:     "admin-settings",
			Path:        "/admin/settings",
			Component:   "SettingsAdmin",
			Name:        "Site Settings",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    6,
			ShowInNav:   true,
			Title:       "System Settings",
			Group:       stringPtr("Administration"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("cogs"),
		},

		// =============================================================================
		// 2. ALLIANCE CATEGORY
		// =============================================================================
		{
			RouteID:     "admin-alliances",
			Path:        "/admin/alliances",
			Component:   "AlliancesAdmin",
			Name:        "Alliance Management",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    10,
			ShowInNav:   true,
			Title:       "Alliance Administration",
			Group:       stringPtr("Alliance"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("handshake"),
		},

		// =============================================================================
		// 3. CORPORATION CATEGORY
		// =============================================================================
		{
			RouteID:     "admin-corporations",
			Path:        "/admin/corporations",
			Component:   "CorporationsAdmin",
			Name:        "Corporation Management",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    20,
			ShowInNav:   true,
			Title:       "Corporation Administration",
			Group:       stringPtr("Corporation"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("building"),
		},

		// =============================================================================
		// 4. PERSONAL CATEGORY
		// =============================================================================
		{
			RouteID:     "dashboard-default",
			Path:        "/",
			Component:   "DefaultDashboard",
			Name:        "Dashboard",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    1,
			ShowInNav:   true,
			Title:       "Personal Dashboard",
			Group:       stringPtr("Personal"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("tachometer-alt"),
			Exact:       true,
		},
		{
			RouteID:     "user-profile",
			Path:        "/user/profile",
			Component:   "UserProfile",
			Name:        "Profile",
			Type:        "auth",
			NavPosition: "user",
			NavOrder:    1,
			ShowInNav:   true,
			Title:       "User Profile",
			Group:       stringPtr("Personal"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("user"),
		},
		{
			RouteID:     "user-characters",
			Path:        "/user/characters",
			Component:   "Characters",
			Name:        "Characters",
			Type:        "auth",
			NavPosition: "user",
			NavOrder:    2,
			ShowInNav:   true,
			Title:       "My Characters",
			Group:       stringPtr("Personal"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("id-card"),
		},
		{
			RouteID:     "app-calendar",
			Path:        "/app/calendar",
			Component:   "Calendar",
			Name:        "Calendar",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    10,
			ShowInNav:   true,
			Title:       "Personal Calendar",
			Group:       stringPtr("Personal"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("calendar-alt"),
		},

		// =============================================================================
		// 5. ECONOMY CATEGORY
		// =============================================================================
		{
			RouteID:             "dashboard-analytics",
			Path:                "/dashboard/analytics",
			Component:           "AnalyticsDashboard",
			Name:                "Market Analytics",
			Type:                "protected",
			RequiredPermissions: []string{"analytics.view"},
			NavPosition:         "main",
			NavOrder:            30,
			ShowInNav:           true,
			Title:               "Market Analytics",
			Group:               stringPtr("Economy"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("chart-line"),
		},

		// =============================================================================
		// 6. UTILITIES CATEGORY
		// =============================================================================
		{
			RouteID:     "app-chat",
			Path:        "/app/chat",
			Component:   "Chat",
			Name:        "Chat",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    40,
			ShowInNav:   true,
			Title:       "Communication Chat",
			Group:       stringPtr("Utilities"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("comments"),
		},
		{
			RouteID:     "app-kanban",
			Path:        "/app/kanban",
			Component:   "Kanban",
			Name:        "Kanban Board",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    41,
			ShowInNav:   true,
			Title:       "Task Management",
			Group:       stringPtr("Utilities"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("clipboard-list"),
		},
		{
			RouteID:     "pages-test-sitemap",
			Path:        "/pages/test-sitemap",
			Component:   "TestSitemap",
			Name:        "Test Sitemap",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    42,
			ShowInNav:   true,
			Title:       "Sitemap Testing Tool",
			Group:       stringPtr("Utilities"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("vial"),
		},
		{
			RouteID:     "widgets",
			Path:        "/widgets",
			Component:   "Widgets",
			Name:        "Widgets",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    43,
			ShowInNav:   true,
			Title:       "UI Widgets",
			Group:       stringPtr("Utilities"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("puzzle-piece"),
		},

		// =============================================================================
		// 7. DOCUMENTATION CATEGORY
		// =============================================================================
		{
			RouteID:     "public-landing",
			Path:        "/landing",
			Component:   "Landing",
			Name:        "Landing",
			Type:        "public",
			NavPosition: "footer",
			NavOrder:    50,
			ShowInNav:   true,
			Title:       "Landing Page",
			Group:       stringPtr("Documentation"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("globe"),
		},
		{
			RouteID:     "changelog",
			Path:        "/changelog",
			Component:   "Changelog",
			Name:        "Changelog",
			Type:        "public",
			NavPosition: "footer",
			NavOrder:    51,
			ShowInNav:   true,
			Title:       "Version History",
			Group:       stringPtr("Documentation"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("code-branch"),
		},
		{
			RouteID:     "migration",
			Path:        "/migration",
			Component:   "Migration",
			Name:        "Migration Guide",
			Type:        "public",
			NavPosition: "footer",
			NavOrder:    52,
			ShowInNav:   true,
			Title:       "Migration Documentation",
			Group:       stringPtr("Documentation"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("sign-out-alt"),
		},

		// Error pages (hidden from navigation)
		{
			RouteID:     "error-404",
			Path:        "/errors/404",
			Component:   "Error404",
			Name:        "404 Not Found",
			Type:        "public",
			NavPosition: "hidden",
			NavOrder:    0,
			ShowInNav:   false,
			Title:       "Page Not Found",
			Group:       stringPtr("Documentation"),
			LazyLoad:    true,
			IsEnabled:   true,
		},
		{
			RouteID:     "error-500",
			Path:        "/errors/500",
			Component:   "Error500",
			Name:        "500 Server Error",
			Type:        "public",
			NavPosition: "hidden",
			NavOrder:    0,
			ShowInNav:   false,
			Title:       "Server Error",
			Group:       stringPtr("Documentation"),
			LazyLoad:    true,
			IsEnabled:   true,
		},
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Ensure Module implements the module interface
var _ module.Module = (*Module)(nil)
