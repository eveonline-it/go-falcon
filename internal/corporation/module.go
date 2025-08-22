package corporation

import (
	"context"
	"log/slog"

	"go-falcon/internal/corporation/routes"
	"go-falcon/internal/corporation/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the corporation module
type Module struct {
	*module.BaseModule
	service *services.Service
	routes  *routes.Module
}

// NewModule creates a new corporation module instance
func NewModule(mongodb *database.MongoDB, redis *database.Redis, eveClient *evegateway.Client) *Module {
	// Initialize repository and service
	repository := services.NewRepository(mongodb)
	service := services.NewService(repository, eveClient)
	
	// Initialize routes
	routesModule := routes.NewModule(service)
	
	// Create the module
	m := &Module{
		BaseModule: module.NewBaseModule("corporation", mongodb, redis),
		service:    service,
		routes:     routesModule,
	}
	
	slog.Info("Corporation module initialized", "name", m.Name())
	
	return m
}

// RegisterUnifiedRoutes registers all corporation routes with the provided Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	slog.Info("Registering corporation unified routes", "basePath", basePath)
	
	// Register all routes through the routes module with basePath
	m.routes.RegisterUnifiedRoutes(api, basePath)
	
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