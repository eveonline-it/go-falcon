package alliance

import (
	"log/slog"

	"go-falcon/internal/alliance/middleware"
	"go-falcon/internal/alliance/routes"
	"go-falcon/internal/alliance/services"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the alliance module
type Module struct {
	*module.BaseModule
	service    *services.Service
	routes     *routes.Module
	middleware *middleware.AuthMiddleware
}

// NewModule creates a new alliance module instance
func NewModule(mongodb *database.MongoDB, redis *database.Redis, eveClient *evegateway.Client, authService *authServices.AuthService, permissionManager *permissions.PermissionManager) *Module {
	// Initialize repository and service
	repository := services.NewRepository(mongodb)
	service := services.NewService(repository, eveClient)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, permissionManager)

	// Initialize routes
	routesModule := routes.NewModule(service, authMiddleware)

	// Create the module
	m := &Module{
		BaseModule: module.NewBaseModule("alliance", mongodb, redis),
		service:    service,
		routes:     routesModule,
		middleware: authMiddleware,
	}

	slog.Info("Alliance module initialized", "name", m.Name())

	return m
}

// RegisterUnifiedRoutes registers all alliance routes with the provided Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	slog.Info("Registering alliance unified routes", "basePath", basePath)

	// Register all routes through the routes module with basePath
	m.routes.RegisterUnifiedRoutes(api, basePath)

	slog.Info("Alliance unified routes registered successfully", "basePath", basePath)
}

// Name returns the module name
func (m *Module) Name() string {
	return m.BaseModule.Name()
}

// Version returns the module version
func (m *Module) Version() string {
	return "1.0.0"
}

// Description returns the module description
func (m *Module) Description() string {
	return "Alliance Management"
}

// Routes is kept for compatibility
func (m *Module) Routes(r chi.Router) {
	// Alliance module uses only Huma v2 routes
}

// GetService returns the alliance service for testing or external access
func (m *Module) GetService() *services.Service {
	return m.service
}
