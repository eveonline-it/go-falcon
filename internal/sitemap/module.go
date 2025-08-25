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

// SeedDefaultRoutes populates the database with default routes based on existing React frontend
// This should be called during initial setup
func (m *Module) SeedDefaultRoutes(ctx context.Context) error {
	log.Printf("🌱 Seeding default routes for sitemap module...")

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

// getDefaultRoutes returns the default routes matching ~/react-falcon frontend structure
func (m *Module) getDefaultRoutes() []dto.CreateRouteInput {
	// This represents the routes from your existing React frontend at ~/react-falcon
	// Based on the siteMaps.ts structure found in src/routes/

	return []dto.CreateRouteInput{
		// Dashboard Routes
		{
			RouteID:     "dashboard-default",
			Path:        "/",
			Component:   "DefaultDashboard",
			Name:        "Dashboard",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    1,
			ShowInNav:   true,
			Title:       "Dashboard",
			Group:       stringPtr("dashboard"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("chart-pie"),
		},
		{
			RouteID:             "dashboard-analytics",
			Path:                "/dashboard/analytics",
			Component:           "AnalyticsDashboard",
			Name:                "Analytics",
			Type:                "protected",
			RequiredPermissions: []string{"analytics.view", "dashboard.access"},
			NavPosition:         "main",
			NavOrder:            2,
			ShowInNav:           true,
			Title:               "Analytics Dashboard",
			Group:               stringPtr("dashboard"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("chart-line"),
		},
		{
			RouteID:             "dashboard-crm",
			Path:                "/dashboard/crm",
			Component:           "CrmDashboard",
			Name:                "CRM",
			Type:                "protected",
			RequiredPermissions: []string{"crm.view", "dashboard.access"},
			NavPosition:         "main",
			NavOrder:            3,
			ShowInNav:           true,
			Title:               "CRM Dashboard",
			Group:               stringPtr("dashboard"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("users"),
		},
		{
			RouteID:             "dashboard-saas",
			Path:                "/dashboard/saas",
			Component:           "SaasDashboard",
			Name:                "SaaS",
			Type:                "protected",
			RequiredPermissions: []string{"saas.view", "dashboard.access"},
			NavPosition:         "main",
			NavOrder:            4,
			ShowInNav:           true,
			Title:               "SaaS Dashboard",
			Group:               stringPtr("dashboard"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("rocket"),
		},
		{
			RouteID:             "dashboard-project-management",
			Path:                "/dashboard/project-management",
			Component:           "ProjectManagementDashboard",
			Name:                "Management",
			Type:                "protected",
			RequiredPermissions: []string{"projects.view", "dashboard.access"},
			NavPosition:         "main",
			NavOrder:            5,
			ShowInNav:           true,
			Title:               "Project Management",
			Group:               stringPtr("dashboard"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("tasks"),
		},
		{
			RouteID:             "dashboard-support-desk",
			Path:                "/dashboard/support-desk",
			Component:           "SupportDeskDashboard",
			Name:                "Support Desk",
			Type:                "protected",
			RequiredPermissions: []string{"support.view", "dashboard.access"},
			NavPosition:         "main",
			NavOrder:            6,
			ShowInNav:           true,
			Title:               "Support Desk",
			Group:               stringPtr("dashboard"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("headset"),
		},

		// App Routes
		{
			RouteID:     "app-calendar",
			Path:        "/app/calendar",
			Component:   "Calendar",
			Name:        "Calendar",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    10,
			ShowInNav:   true,
			Title:       "Calendar",
			Group:       stringPtr("app"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("calendar-alt"),
		},
		{
			RouteID:     "app-chat",
			Path:        "/app/chat",
			Component:   "Chat",
			Name:        "Chat",
			Type:        "auth",
			NavPosition: "main",
			NavOrder:    11,
			ShowInNav:   true,
			Title:       "Chat",
			Group:       stringPtr("app"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("comments"),
		},
		{
			RouteID:             "app-kanban",
			Path:                "/app/kanban",
			Component:           "Kanban",
			Name:                "Kanban",
			Type:                "protected",
			RequiredPermissions: []string{"kanban.view"},
			NavPosition:         "main",
			NavOrder:            12,
			ShowInNav:           true,
			Title:               "Kanban Board",
			Group:               stringPtr("app"),
			LazyLoad:            true,
			IsEnabled:           true,
			Icon:                stringPtr("clipboard-list"),
		},

		// Email Routes
		{
			RouteID:             "app-email-inbox",
			Path:                "/app/email/inbox",
			Component:           "EmailInbox",
			Name:                "Inbox",
			Type:                "protected",
			RequiredPermissions: []string{"email.view"},
			ParentID:            stringPtr("app-email"),
			NavPosition:         "main",
			NavOrder:            13,
			ShowInNav:           true,
			Title:               "Email Inbox",
			Group:               stringPtr("app"),
			LazyLoad:            true,
			IsEnabled:           true,
		},

		// User Routes
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
			Group:       stringPtr("user"),
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
			Group:       stringPtr("user"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("id-card"),
		},

		// Admin Routes
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
			Group:       stringPtr("admin"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("users-cog"),
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
			Group:       stringPtr("admin"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("users"),
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
			Group:       stringPtr("admin"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("key"),
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
			Group:       stringPtr("admin"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("clock"),
		},
		{
			RouteID:     "admin-settings",
			Path:        "/admin/settings",
			Component:   "SettingsAdmin",
			Name:        "Settings",
			Type:        "admin",
			NavPosition: "admin",
			NavOrder:    5,
			ShowInNav:   true,
			Title:       "System Settings",
			Group:       stringPtr("admin"),
			LazyLoad:    true,
			IsEnabled:   true,
			Icon:        stringPtr("cogs"),
		},

		// Public Routes
		{
			RouteID:     "public-landing",
			Path:        "/landing",
			Component:   "Landing",
			Name:        "Landing",
			Type:        "public",
			NavPosition: "footer",
			NavOrder:    1,
			ShowInNav:   true,
			Title:       "Landing Page",
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
