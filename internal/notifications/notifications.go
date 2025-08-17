package notifications

import (
	"context"
	"log/slog"
	"net/http"

	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

// GroupsModule interface defines the methods needed from the groups module
type GroupsModule interface {
	RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
}

type Module struct {
	*module.BaseModule
	groupsModule GroupsModule
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, groupsModule GroupsModule) *Module {
	return &Module{
		BaseModule:   module.NewBaseModule("notifications", mongodb, redis, sdeService),
		groupsModule: groupsModule,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	
	// Protected notification endpoints
	r.With(m.groupsModule.RequireGranularPermission("notifications", "messages", "read")).Get("/", m.getNotificationsHandler)
	r.With(m.groupsModule.RequireGranularPermission("notifications", "messages", "write")).Post("/", m.sendNotificationHandler)
	r.With(m.groupsModule.RequireGranularPermission("notifications", "messages", "write")).Put("/{id}", m.markReadHandler)
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