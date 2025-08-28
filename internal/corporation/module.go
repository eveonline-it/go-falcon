package corporation

import (
	"context"
	"log/slog"
	"time"

	"go-falcon/internal/auth"
	characterServices "go-falcon/internal/character/services"
	"go-falcon/internal/corporation/routes"
	"go-falcon/internal/corporation/services"
	groupsServices "go-falcon/internal/groups/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"
	"go-falcon/pkg/sde"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the corporation module
type Module struct {
	*module.BaseModule
	service      *services.Service
	routes       *routes.Module
	authModule   *auth.Module
	groupService *groupsServices.Service
}

// NewModule creates a new corporation module instance
func NewModule(mongodb *database.MongoDB, redis *database.Redis, eveClient *evegateway.Client, authModule *auth.Module, characterService *characterServices.Service, sdeService sde.SDEService) *Module {
	// Initialize repository and service
	repository := services.NewRepository(mongodb)
	service := services.NewService(repository, eveClient, characterService, sdeService)

	// Initialize routes
	routesModule := routes.NewModule(service)

	// Create the module
	m := &Module{
		BaseModule:   module.NewBaseModule("corporation", mongodb, redis),
		service:      service,
		routes:       routesModule,
		authModule:   authModule,
		groupService: nil, // Will be set after groups module initialization
	}

	slog.Info("Corporation module initialized", "name", m.Name())

	return m
}

// SetGroupService sets the groups service dependency
func (m *Module) SetGroupService(groupService *groupsServices.Service) {
	m.groupService = groupService
}

// RegisterUnifiedRoutes registers all corporation routes with the provided Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	slog.Info("Registering corporation unified routes", "basePath", basePath)

	// Create corporation adapter using centralized middleware if auth module is available
	var corporationAdapter *middleware.CorporationAdapter
	if m.authModule != nil {
		authService := m.authModule.GetAuthService()
		if authService != nil {
			// Get permission manager from groups service if available
			var permissionManager *permissions.PermissionManager
			if m.groupService != nil {
				permissionManager = m.groupService.GetPermissionManager()
			}

			// Create centralized permission middleware
			permissionMiddleware := middleware.NewPermissionMiddleware(authService, permissionManager)
			// Create corporation adapter
			corporationAdapter = middleware.NewCorporationAdapter(permissionMiddleware)
		}
	}

	// Register all routes through the routes module with basePath and corporation adapter
	m.routes.RegisterUnifiedRoutes(api, basePath, corporationAdapter)

	slog.Info("Corporation unified routes registered successfully", "basePath", basePath)
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
	return "Corporation Management"
}

// Routes is kept for compatibility
func (m *Module) Routes(r chi.Router) {
	// Corporation module uses only Huma v2 routes
}

// GetService returns the corporation service for testing or external access
func (m *Module) GetService() *services.Service {
	return m.service
}

// UpdateAllCorporations implements the scheduler's CorporationModule interface
func (m *Module) UpdateAllCorporations(ctx context.Context, concurrentWorkers int) error {
	return m.service.UpdateAllCorporations(ctx, concurrentWorkers)
}

// ValidateCEOTokens implements the scheduler's CorporationModule interface for CEO token validation
func (m *Module) ValidateCEOTokens(ctx context.Context) error {
	return m.service.ValidateCEOTokens(ctx)
}

// RegisterPermissions registers corporation-specific permissions
func (m *Module) RegisterPermissions(ctx context.Context, permissionManager *permissions.PermissionManager) error {
	corporationPermissions := []permissions.Permission{
		{
			ID:          "corporation:info:view",
			Service:     "corporation",
			Resource:    "info",
			Action:      "view",
			IsStatic:    false,
			Name:        "View Corporation Information",
			Description: "View detailed EVE corporation profiles and information",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "corporation:search:access",
			Service:     "corporation",
			Resource:    "search",
			Action:      "access",
			IsStatic:    false,
			Name:        "Search Corporations",
			Description: "Search for corporations by name or ticker and access corporation listings",
			Category:    "Content Management",
			CreatedAt:   time.Now(),
		},
		{
			ID:          "corporation:data:manage",
			Service:     "corporation",
			Resource:    "data",
			Action:      "manage",
			IsStatic:    false,
			Name:        "Manage Corporation Data",
			Description: "Trigger corporation data updates and manage corporation information",
			Category:    "System Administration",
			CreatedAt:   time.Now(),
		},
	}

	return permissionManager.RegisterServicePermissions(ctx, corporationPermissions)
}
