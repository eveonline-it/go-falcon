package users

import (
	"context"
	"log"
	"log/slog"

	"go-falcon/internal/auth"
	"go-falcon/internal/users/routes"
	"go-falcon/internal/users/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// Module represents the users module
type Module struct {
	*module.BaseModule
	service      *services.Service
	routes       *routes.Routes
	authModule   *auth.Module
}

// New creates a new users module instance
func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, authModule *auth.Module, groupsModule interface{}) *Module {
	service := services.NewService(mongodb)

	return &Module{
		BaseModule:   module.NewBaseModule("users", mongodb, redis, sdeService),
		service:      service,
		routes:       nil, // Will be created when needed
		authModule:   authModule,
	}
}

// Routes is kept for compatibility - users now uses Huma v2 routes only
func (m *Module) Routes(r chi.Router) {
	// Users module now uses only Huma v2 routes - call RegisterHumaRoutes instead
	m.RegisterHumaRoutes(r)
}

// RegisterHumaRoutes registers the Huma v2 routes
func (m *Module) RegisterHumaRoutes(r chi.Router) {
	if m.routes == nil {
		m.routes = routes.NewRoutes(m.service, r)
	}
}

// RegisterUnifiedRoutes registers routes on the shared Huma API
func (m *Module) RegisterUnifiedRoutes(api huma.API, basePath string) {
	routes.RegisterUsersRoutes(api, basePath, m.service)
	log.Printf("Users module unified routes registered at %s", basePath)
}

// StartBackgroundTasks starts any background processes for the module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting users-specific background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Users module doesn't need specific background tasks currently
	// This could be extended in the future for user-specific maintenance tasks
	for {
		select {
		case <-ctx.Done():
			slog.Info("Users background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Users background tasks stopped")
			return
		default:
			// No specific background tasks for users module currently
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}