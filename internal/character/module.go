package character

import (
	"context"
	"log"

	"go-falcon/internal/character/routes"
	"go-falcon/internal/character/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the character module
type Module struct {
	*module.BaseModule
	service    *services.Service
	eveGateway *evegateway.Client
}

// New creates a new character module instance
func New(mongodb *database.MongoDB, redis *database.Redis, eveGateway *evegateway.Client) *Module {
	service := services.NewService(mongodb, eveGateway)

	return &Module{
		BaseModule: module.NewBaseModule("character", mongodb, redis),
		service:    service,
		eveGateway: eveGateway,
	}
}

// Routes is kept for compatibility
func (m *Module) Routes(r chi.Router) {
	// Character module uses only Huma v2 routes
}

// RegisterHumaRoutes registers the Huma v2 routes (legacy)
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	// Character module uses unified routes only
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterCharacterRoutes(api, basePath, m.service)
	log.Printf("Character module unified routes registered at %s", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// No background tasks needed for now
}