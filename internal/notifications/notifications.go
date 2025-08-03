package notifications

import (
	"context"
	"log/slog"
	"net/http"

	"go-falcon/pkg/database"
	"go-falcon/pkg/module"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
}

func New(mongodb *database.MongoDB, redis *database.Redis) *Module {
	return &Module{
		BaseModule: module.NewBaseModule("notifications", mongodb, redis),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Get("/", m.getNotificationsHandler)
	r.Post("/", m.sendNotificationHandler)
	r.Put("/{id}", m.markReadHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting notifications-specific background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Add notifications-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Notifications background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Notifications background tasks stopped")
			return
		default:
			// Notifications-specific background work would go here
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

// Health handler is now provided by BaseModule

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