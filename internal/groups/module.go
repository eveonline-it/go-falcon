package groups

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	authMiddleware "go-falcon/internal/auth/middleware"
	authServices "go-falcon/internal/auth/services"
	groupsMiddleware "go-falcon/internal/groups/middleware"
	"go-falcon/internal/groups/routes"
	"go-falcon/internal/groups/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
)

// Module represents the groups module
type Module struct {
	service    *services.Service
	middleware *groupsMiddleware.AuthMiddleware
	routes     *routes.Module
}

// AuthModule interface for auth module dependency
type AuthModule interface {
	GetMiddleware() *authMiddleware.Middleware
	GetAuthService() *authServices.AuthService // Auth service for character context resolution
}

// NewModule creates a new groups module
func NewModule(db *database.MongoDB, authModule AuthModule) (*Module, error) {
	// Create service
	service := services.NewService(db)

	// Get auth service for character context resolution
	authService := authModule.GetAuthService()
	if authService == nil {
		return nil, fmt.Errorf("auth service is required for groups module")
	}
	
	// Create auth middleware with character context resolution
	groupMiddleware := groupsMiddleware.NewAuthMiddleware(authService, service)
	slog.Info("Groups module initialized with character context middleware")

	// Create routes
	routesModule := routes.NewModule(service, groupMiddleware)

	return &Module{
		service:    service,
		middleware: groupMiddleware,
		routes:     routesModule,
	}, nil
}

// Initialize implements the module.Module interface
func (m *Module) Initialize(ctx context.Context) error {
	slog.Info("Initializing groups module")

	// Initialize service (create indexes and system groups)
	if err := m.service.InitializeService(ctx); err != nil {
		return fmt.Errorf("failed to initialize groups service: %w", err)
	}

	slog.Info("Groups module initialized successfully")
	return nil
}

// Routes implements module.Module interface - registers Chi routes (legacy)
func (m *Module) Routes(r chi.Router) {
	// For Phase 1, we only use the unified API, so this is a no-op
	slog.Info("Groups module routes called (using unified API instead)")
}

// StartBackgroundTasks implements module.Module interface
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting groups background tasks")
	// For Phase 1, no background tasks are needed
}

// Stop implements module.Module interface
func (m *Module) Stop() {
	slog.Info("Stopping groups module")
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	slog.Info("Registering groups routes")
	m.routes.RegisterUnifiedRoutes(api)
}

// Name implements the module.Module interface
func (m *Module) Name() string {
	return "groups"
}

// Close implements the module.Module interface
func (m *Module) Close() error {
	slog.Info("Closing groups module")
	return nil
}

// GetService returns the groups service for use by other modules
func (m *Module) GetService() *services.Service {
	return m.service
}

// GetMiddleware returns the groups middleware for use by other modules
func (m *Module) GetMiddleware() *groupsMiddleware.AuthMiddleware {
	return m.middleware
}

// Ensure Module implements the module.Module interface
var _ module.Module = (*Module)(nil)