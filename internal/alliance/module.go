package alliance

import (
	"context"
	"log/slog"
	"time"

	"go-falcon/internal/alliance/routes"
	"go-falcon/internal/alliance/services"
	authServices "go-falcon/internal/auth/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the alliance module
type Module struct {
	*module.BaseModule
	service         *services.Service
	routes          *routes.Module
	allianceAdapter *middleware.AllianceAdapter
}

// NewModule creates a new alliance module instance
func NewModule(mongodb *database.MongoDB, redis *database.Redis, eveClient *evegateway.Client, authService *authServices.AuthService, permissionManager *permissions.PermissionManager) *Module {
	// Initialize repository and service
	repository := services.NewRepository(mongodb)
	service := services.NewService(repository, eveClient)

	// Initialize centralized permission middleware with debug logging for migration
	permissionMiddleware := middleware.NewPermissionMiddleware(
		authService,
		permissionManager,
		middleware.WithDebugLogging(),
	)

	// Create alliance-specific adapter
	allianceAdapter := middleware.NewAllianceAdapter(permissionMiddleware)

	// Initialize routes
	routesModule := routes.NewModule(service, allianceAdapter)

	// Create the module
	m := &Module{
		BaseModule:      module.NewBaseModule("alliance", mongodb, redis),
		service:         service,
		routes:          routesModule,
		allianceAdapter: allianceAdapter,
	}

	slog.Info("Alliance module initialized with centralized middleware", "name", m.Name())

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

// RegisterPermissions registers alliance-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager *permissions.PermissionManager) error {
	alliancePermissions := []permissions.Permission{
		{
			ID:          "alliance:info:view",
			Service:     "alliance",
			Resource:    "info",
			Action:      "view",
			IsStatic:    false,
			Name:        "View Alliance Information",
			Description: "View detailed EVE alliance profiles and information",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "alliance:list:access",
			Service:     "alliance",
			Resource:    "list",
			Action:      "access",
			IsStatic:    false,
			Name:        "List Alliances",
			Description: "Access the list of all active EVE alliances",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "alliance:corporations:view",
			Service:     "alliance",
			Resource:    "corporations",
			Action:      "view",
			IsStatic:    false,
			Name:        "View Alliance Corporations",
			Description: "View the list of corporations that belong to an alliance",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "alliance:data:manage",
			Service:     "alliance",
			Resource:    "data",
			Action:      "manage",
			IsStatic:    false,
			Name:        "Manage Alliance Data",
			Description: "Import and manage alliance data from EVE ESI",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
	}

	return permissionManager.RegisterServicePermissions(ctx, alliancePermissions)
}
