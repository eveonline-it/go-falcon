package sde

import (
	"context"
	"log/slog"
	"log"

	"go-falcon/internal/sde/routes"
	"go-falcon/internal/sde/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"github.com/danielgtaylor/huma/v2"
)

// Module represents the SDE module
type Module struct {
	*module.BaseModule
	service    *services.Service
	routes     *routes.Routes
}

// NewModule creates a new SDE module instance
func NewModule(
	mongodb *database.MongoDB,
	redis *database.Redis,
	sdeService *sde.Service,
) *Module {
	// Create repository
	repo := services.NewRepository(mongodb, redis)

	// Create service
	service := services.NewService(repo, sdeService)

	return &Module{
		BaseModule: module.NewBaseModule("sde", mongodb, redis, sdeService),
		service:    service,
		routes:     nil, // Will be created when needed
	}
}

// Routes is kept for compatibility - sde now uses Huma v2 routes only
func (m *Module) Routes(r chi.Router) {
	// SDE module now uses only Huma v2 routes - call RegisterHumaRoutes instead
	m.RegisterHumaRoutes(r)
}

// RegisterHumaRoutes registers the Huma v2 routes
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.service, r)
	}
}

// StartBackgroundTasks starts SDE-specific background tasks
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting SDE background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// SDE-specific background tasks would go here
	for {
		select {
		case <-ctx.Done():
			slog.Info("SDE background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("SDE background tasks stopped")
			return
		default:
			// SDE-specific background work would go here
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}

// CheckSDEUpdate checks for SDE updates (for scheduler integration)
func (m *Module) CheckSDEUpdate(ctx context.Context) error {
	// Placeholder implementation for scheduler compatibility
	// The actual SDE update checking would go here
	slog.Info("SDE update check requested")
	return nil
}
// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterSDERoutes(api, basePath, m.service)
	log.Printf("SDE module unified routes registered at %s", basePath)
}
