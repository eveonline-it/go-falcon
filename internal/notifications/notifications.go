package notifications

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
	r.Get("/", m.getNotificationsHandler)
	r.Post("/", m.sendNotificationHandler)
	r.Put("/{id}", m.markReadHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting notifications background tasks")
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Notifications background tasks stopped due to context cancellation")
			return
		case <-m.stopCh:
			slog.Info("Notifications background tasks stopped")
			return
		default:
			// Notifications background processing would go here
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
	slog.Info("Notifications health check requested",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("module", "notifications"),
	)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","module":"notifications"}`))
}

func (m *Module) getNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Notifications module - list notifications","status":"not_implemented"}`))
}

func (m *Module) sendNotificationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Notifications module - send notification","status":"not_implemented"}`))
}

func (m *Module) markReadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Notifications module - mark as read","status":"not_implemented"}`))
}