package users

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
		BaseModule: module.NewBaseModule("users", mongodb, redis, sdeService),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Get("/", m.getUsersHandler)
	r.Get("/{id}", m.getUserHandler)
	r.Post("/", m.createUserHandler)
	r.Put("/{id}", m.updateUserHandler)
	r.Delete("/{id}", m.deleteUserHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting users-specific background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Add users-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Users background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Users background tasks stopped")
			return
		default:
			// Users-specific background work would go here
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

func (m *Module) getUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Users module - list users","status":"not_implemented"}`))
}

func (m *Module) getUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Users module - get user","status":"not_implemented"}`))
}

func (m *Module) createUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Users module - create user","status":"not_implemented"}`))
}

func (m *Module) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Users module - update user","status":"not_implemented"}`))
}

func (m *Module) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Users module - delete user","status":"not_implemented"}`))
}