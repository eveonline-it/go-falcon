package site_settings

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/site_settings/routes"
	"go-falcon/internal/site_settings/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"

	authServices "go-falcon/internal/auth/services"
	groupsServices "go-falcon/internal/groups/services"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the site settings module
type Module struct {
	service              *services.Service
	permissionMiddleware *middleware.PermissionMiddleware
	routes               *routes.Module
}

// NewModule creates a new site settings module
func NewModule(db *database.MongoDB, authService *authServices.AuthService, groupsService *groupsServices.Service) (*Module, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Create service layer
	service := services.NewService(db.Database)

	var permissionMiddleware *middleware.PermissionMiddleware
	var routesModule *routes.Module

	if authService != nil && groupsService != nil {
		// Create permission middleware with centralized system
		permissionManager := permissions.NewPermissionManager(db.Database)
		permissionMiddleware = middleware.NewPermissionMiddleware(authService, permissionManager)
		slog.Info("Site settings module initialized with centralized middleware")
	} else {
		slog.Info("Site settings module initialized without dependencies (will be set later)")
	}

	// Create routes (middleware might be nil initially)
	routesModule = routes.NewModule(service, permissionMiddleware)

	return &Module{
		service:              service,
		permissionMiddleware: permissionMiddleware,
		routes:               routesModule,
	}, nil
}

// Initialize initializes the module
func (m *Module) Initialize(ctx context.Context) error {
	slog.Info("Initializing site settings module")

	if err := m.service.InitializeModule(ctx); err != nil {
		return fmt.Errorf("failed to initialize site settings service: %w", err)
	}

	slog.Info("Site settings module initialized successfully")
	return nil
}

// Routes implements module.Module interface - registers Chi routes (legacy)
func (m *Module) Routes(r chi.Router) {
	// For Phase 1, we only use the unified API, so this is a no-op
	slog.Info("Site settings module routes called (using unified API instead)")
}

// StartBackgroundTasks implements module.Module interface
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting site settings background tasks")
	// For Phase 1, no background tasks are needed
}

// Stop implements module.Module interface
func (m *Module) Stop() {
	slog.Info("Stopping site settings module")
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	slog.Info("Registering site settings routes")
	m.routes.RegisterUnifiedRoutes(api)
}

// Name implements the module.Module interface
func (m *Module) Name() string {
	return "site_settings"
}

// GetService returns the site settings service for use by other modules
func (m *Module) GetService() *services.Service {
	return m.service
}

// SetDependencies sets both auth and groups service dependencies and recreates middleware
func (m *Module) SetDependencies(authService *authServices.AuthService, groupsService *groupsServices.Service) {
	if authService == nil || groupsService == nil {
		slog.Error("Cannot set site settings dependencies - auth or groups service is nil")
		return
	}

	// For now, use auth-only mode since we don't have direct database access here
	// In a proper refactor, we'd pass the permission manager from main.go
	m.permissionMiddleware = middleware.NewPermissionMiddleware(authService, nil)

	// Recreate routes with the new middleware
	m.routes = routes.NewModule(m.service, m.permissionMiddleware)
	slog.Info("Site settings middleware and routes updated with centralized system (auth-only mode)")
}

// SetDependenciesWithPermissions sets auth, groups service, and permission manager dependencies
func (m *Module) SetDependenciesWithPermissions(authService *authServices.AuthService, groupsService *groupsServices.Service, permissionManager *permissions.PermissionManager) {
	if authService == nil || groupsService == nil {
		slog.Error("Cannot set site settings dependencies - auth or groups service is nil")
		return
	}

	if permissionManager == nil {
		slog.Error("Cannot set site settings dependencies - permission manager is nil")
		return
	}

	// Create permission middleware with the shared permission manager
	m.permissionMiddleware = middleware.NewPermissionMiddleware(authService, permissionManager)

	// Recreate routes with the new middleware
	m.routes = routes.NewModule(m.service, m.permissionMiddleware)
	slog.Info("Site settings middleware and routes updated with centralized permission system")
}

// SetAuthService sets the auth service dependency after module initialization
func (m *Module) SetAuthService(authService *authServices.AuthService) {
	// This method is kept for backward compatibility but doesn't do anything
	// The actual work is done in SetDependencies when both services are available
	slog.Info("Site settings auth service dependency noted (waiting for groups service)")
}

// SetGroupsService sets the groups service dependency after module initialization
func (m *Module) SetGroupsService(groupsService *groupsServices.Service) {
	// This method is kept for backward compatibility but doesn't do anything
	// The actual work is done in SetDependencies when both services are available
	slog.Info("Site settings groups service dependency noted (waiting for auth service)")
}

// Ensure Module implements the module.Module interface
var _ module.Module = (*Module)(nil)
