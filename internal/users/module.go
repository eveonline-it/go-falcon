package users

import (
	"context"
	"log/slog"

	"go-falcon/internal/auth"
	"go-falcon/internal/groups"
	"go-falcon/internal/users/routes"
	"go-falcon/internal/users/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

// Module represents the users module
type Module struct {
	*module.BaseModule
	service      *services.Service
	handler      *routes.Handler
	authModule   *auth.Module
	groupsModule *groups.Module
}

// New creates a new users module instance
func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, authModule *auth.Module, groupsModule *groups.Module) *Module {
	service := services.NewService(mongodb)
	handler := routes.NewHandler(service, authModule, groupsModule)

	return &Module{
		BaseModule:   module.NewBaseModule("users", mongodb, redis, sdeService),
		service:      service,
		handler:      handler,
		authModule:   authModule,
		groupsModule: groupsModule,
	}
}

// Routes registers the module's routes
func (m *Module) Routes(r chi.Router) {
	// Register health check route
	m.RegisterHealthRoute(r)
	
	// Register users-specific routes
	m.handler.RegisterRoutes(r)
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