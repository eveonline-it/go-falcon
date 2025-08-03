package auth

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
	r.Post("/login", m.loginHandler)
	r.Post("/register", m.registerHandler)
	r.Get("/status", m.statusHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting auth background tasks")
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Auth background tasks stopped due to context cancellation")
			return
		case <-m.stopCh:
			slog.Info("Auth background tasks stopped")
			return
		default:
			// Auth background processing would go here
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
	slog.Info("Auth health check requested",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("module", "auth"),
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","module":"auth"}`))
}

func (m *Module) loginHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Login attempt", slog.String("remote_addr", r.RemoteAddr))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Auth module - login endpoint","status":"not_implemented"}`))
}

func (m *Module) registerHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Registration attempt", slog.String("remote_addr", r.RemoteAddr))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Auth module - register endpoint","status":"not_implemented"}`))
}

func (m *Module) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"module":"auth","status":"running","version":"1.0.0"}`))
}