package site_settings

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/site_settings/middleware"
	"go-falcon/internal/site_settings/routes"
	"go-falcon/internal/site_settings/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"

	authServices "go-falcon/internal/auth/services"
	groupsServices "go-falcon/internal/groups/services"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the site settings module
type Module struct {
	service    *services.Service
	middleware *middleware.AuthMiddleware
	routes     *routes.Module
}

// NewModule creates a new site settings module
func NewModule(db *database.MongoDB, authService *authServices.AuthService, groupsService *groupsServices.Service) (*Module, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	// Create service layer
	service := services.NewService(db.Database)

	var authMiddleware *middleware.AuthMiddleware
	var routesModule *routes.Module
	
	if authService != nil && groupsService != nil {
		// Create middleware with dependencies
		authMiddleware = middleware.NewAuthMiddleware(authService, groupsService)
		slog.Info("Site settings module initialized with auth and groups dependencies")
	} else {
		slog.Info("Site settings module initialized without dependencies (will be set later)")
	}

	// Create routes (middleware might be nil initially)
	routesModule = routes.NewModule(service, authMiddleware)

	return &Module{
		service:    service,
		middleware: authMiddleware,
		routes:     routesModule,
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

// SetAuthService sets the auth service dependency after module initialization
func (m *Module) SetAuthService(authService *authServices.AuthService) {
	// For now, just log that this was called
	// TODO: Implement proper dependency injection for middleware recreation
	slog.Info("Site settings auth service dependency set (middleware update needed)")
}

// SetGroupsService sets the groups service dependency after module initialization
func (m *Module) SetGroupsService(groupsService *groupsServices.Service) {
	// For now, just log that this was called
	// TODO: Implement proper dependency injection for middleware recreation
	slog.Info("Site settings groups service dependency set (middleware update needed)")
}

// Ensure Module implements the module.Module interface
var _ module.Module = (*Module)(nil)