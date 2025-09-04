package killmails

import (
	"context"

	"go-falcon/internal/killmails/routes"
	"go-falcon/internal/killmails/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the killmails module
type Module struct {
	*module.BaseModule
	service    *services.Service
	repository *services.Repository
	eveGateway *evegateway.Client
}

// New creates a new killmails module instance
func New(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client) *Module {
	repository := services.NewRepository(mongodb)
	service := services.NewService(repository, eveGateway)

	return &Module{
		BaseModule: module.NewBaseModule("killmails", mongodb, redis),
		service:    service,
		repository: repository,
		eveGateway: eveGateway,
	}
}

// RegisterUnifiedRoutes registers all killmails routes with the unified API gateway
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterKillmailRoutes(api, basePath, m.service)
}

// Routes registers routes on a Chi router (implements module.Module interface)
func (m *Module) Routes(r chi.Router) {
	// Killmails module uses only Huma v2 unified routes
}

// Initialize performs module initialization tasks
func (m *Module) Initialize(ctx context.Context) error {
	// Create database indexes for optimal performance
	if err := m.repository.CreateIndexes(ctx); err != nil {
		return err
	}

	return nil
}

// GetService returns the service instance for this module
func (m *Module) GetService() *services.Service {
	return m.service
}

// GetRepository returns the repository instance for this module
func (m *Module) GetRepository() *services.Repository {
	return m.repository
}
