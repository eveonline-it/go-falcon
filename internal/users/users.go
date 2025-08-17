package users

import (
	"context"
	"log/slog"
	"net/http"

	"go-falcon/internal/auth"
	"go-falcon/internal/groups"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
	authModule   *auth.Module
	groupsModule *groups.Module
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, authModule *auth.Module, groupsModule *groups.Module) *Module {
	return &Module{
		BaseModule:   module.NewBaseModule("users", mongodb, redis, sdeService),
		authModule:   authModule,
		groupsModule: groupsModule,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler

	// Public endpoints - basic user information (no authentication required)
	r.Get("/stats", m.getUserStatsHandler)                    // GET /api/users/stats

	// Administrative endpoints - require authentication and granular permissions
	r.With(m.groupsModule.RequireGranularPermission("users", "profiles", "read")).Get("/", m.listUsersHandler) // GET /api/users?page=1&page_size=20&query=search
	r.With(m.groupsModule.RequireGranularPermission("users", "profiles", "read")).Get("/{character_id}", m.getUserHandler) // GET /api/users/{character_id}
	r.With(m.groupsModule.RequireGranularPermission("users", "profiles", "write")).Put("/{character_id}", m.updateUserHandler) // PUT /api/users/{character_id}
	
	// User-specific character management - requires authentication, users can view their own characters or admins can view any
	r.With(m.authModule.JWTMiddleware).Get("/by-user-id/{user_id}/characters", m.listCharactersHandler) // GET /api/users/by-user-id/{user_id}/characters
}

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

// statusHandler provides module status information
func (m *Module) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"module":"users","status":"running","version":"1.0.0"}`))
}