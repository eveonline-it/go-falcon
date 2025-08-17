package sde

import (
	"context"
	"log/slog"

	"go-falcon/internal/sde/routes"
	"go-falcon/internal/sde/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

// Module represents the SDE module
type Module struct {
	*module.BaseModule
	service *services.Service
	routes  *routes.Routes
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

	// Create routes
	routesHandler := routes.NewRoutes(service)

	return &Module{
		BaseModule: module.NewBaseModule("sde", mongodb, redis, sdeService),
		service:    service,
		routes:     routesHandler,
	}
}

// Routes registers the module's routes
func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r)
	m.routes.RegisterRoutes(r)
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