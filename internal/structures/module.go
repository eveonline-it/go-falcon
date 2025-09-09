package structures

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go-falcon/internal/structures/routes"
	structureservices "go-falcon/internal/structures/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/middleware"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"
)

// Module implements the structures module
type Module struct {
	*module.BaseModule
	service *structureservices.StructureService
	routes  *routes.StructureRoutes
}

// NewModule creates a new structures module
func NewModule(db *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client, sdeService sde.SDEService, authMiddleware *middleware.PermissionMiddleware, structureTracker structureservices.StructureAccessTracker, authRepository routes.AuthRepository) *Module {
	// Create service
	service := structureservices.NewStructureService(db.Database, redis.Client, eveGateway, sdeService, structureTracker)

	// Create routes
	structureRoutes := routes.NewStructureRoutes(service, authMiddleware, authRepository)

	// Create module
	m := &Module{
		BaseModule: module.NewBaseModule("structures", db, redis),
		service:    service,
		routes:     structureRoutes,
	}

	return m
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string, authService routes.AuthService, structuresAdapter *middleware.PermissionMiddleware) {
	routes.RegisterStructuresRoutes(api, basePath, m.service, structuresAdapter, authService)
}

// GetService returns the structure service
func (m *Module) GetService() *structureservices.StructureService {
	return m.service
}

// Initialize initializes the module
func (m *Module) Initialize(ctx context.Context) error {
	// Create indexes
	if err := m.createIndexes(ctx); err != nil {
		return err
	}

	return nil
}

// createIndexes creates database indexes
func (m *Module) createIndexes(ctx context.Context) error {
	// Add any necessary indexes for the structures collection
	// This is handled by the base module if needed
	return nil
}

// Routes implements the Module interface (legacy)
func (m *Module) Routes(r chi.Router) {
	// Legacy route registration - not used with unified API
}

// Shutdown gracefully shuts down the module
func (m *Module) Shutdown(ctx context.Context) error {
	// Any cleanup needed
	return nil
}
