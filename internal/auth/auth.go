package auth

import (
	"context"
	"log/slog"
	"net/http"

	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	return &Module{
		BaseModule: module.NewBaseModule("auth", mongodb, redis, sdeService),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Post("/login", m.loginHandler)
	r.Post("/register", m.registerHandler)
	r.Get("/status", m.statusHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting auth-specific background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Add auth-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Auth background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Auth background tasks stopped")
			return
		default:
			// Auth-specific background work would go here
			// For now, just wait
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}

func (m *Module) loginHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Login attempt", slog.String("remote_addr", r.RemoteAddr), slog.String("module", m.Name()))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Auth module - login endpoint","status":"not_implemented"}`))
}

func (m *Module) registerHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Registration attempt", slog.String("remote_addr", r.RemoteAddr), slog.String("module", m.Name()))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Auth module - register endpoint","status":"not_implemented"}`))
}

func (m *Module) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"module":"auth","status":"running","version":"1.0.0"}`))
}