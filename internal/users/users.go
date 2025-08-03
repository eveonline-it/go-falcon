package users

import (
	"context"
	"log/slog"
	"net/http"

	"go-falcon/pkg/database"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	mongodb *database.MongoDB
	redis   *database.Redis
	stopCh  chan struct{}
}

func New(mongodb *database.MongoDB, redis *database.Redis) *Module {
	return &Module{
		mongodb: mongodb,
		redis:   redis,
		stopCh:  make(chan struct{}),
	}
}

func (m *Module) Routes(r chi.Router) {
	r.Get("/health", m.healthHandler)
	r.Get("/", m.getUsersHandler)
	r.Get("/{id}", m.getUserHandler)
	r.Post("/", m.createUserHandler)
	r.Put("/{id}", m.updateUserHandler)
	r.Delete("/{id}", m.deleteUserHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting users background tasks")
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Users background tasks stopped due to context cancellation")
			return
		case <-m.stopCh:
			slog.Info("Users background tasks stopped")
			return
		default:
			// Users background processing would go here
			// For now, just sleep to prevent busy loop
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			}
		}
	}
}

func (m *Module) Stop() {
	close(m.stopCh)
}

func (m *Module) healthHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Users health check requested",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("module", "users"),
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","module":"users"}`))
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