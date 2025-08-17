package notifications

import (
	"context"
	"log/slog"
	"time"

	"go-falcon/internal/auth"
	"go-falcon/internal/groups"
	"go-falcon/internal/notifications/routes"
	"go-falcon/internal/notifications/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

// Module represents the notifications module
type Module struct {
	*module.BaseModule
	service      *services.Service
	handler      *routes.Handler
	authModule   *auth.Module
	groupsModule *groups.Module
}

// New creates a new notifications module instance
func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, authModule *auth.Module, groupsModule *groups.Module) *Module {
	service := services.NewService(mongodb)
	handler := routes.NewHandler(service, authModule, groupsModule)

	return &Module{
		BaseModule:   module.NewBaseModule("notifications", mongodb, redis, sdeService),
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
	
	// Register notifications-specific routes
	m.handler.RegisterRoutes(r)
}

// StartBackgroundTasks starts background processes for the notifications module
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting notifications background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Start notification-specific background tasks
	go m.cleanupExpiredNotifications(ctx)
	
	// Main background task loop
	for {
		select {
		case <-ctx.Done():
			slog.Info("Notifications background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Notifications background tasks stopped")
			return
		default:
			// Notifications-specific background work
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			case <-time.After(30 * time.Minute): // Check every 30 minutes
				// Periodic maintenance tasks could go here
			}
		}
	}
}

// cleanupExpiredNotifications runs periodic cleanup of expired notifications
func (m *Module) cleanupExpiredNotifications(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification cleanup task stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Notification cleanup task stopped")
			return
		case <-ticker.C:
			slog.Info("Starting expired notifications cleanup")
			
			count, err := m.service.CleanupExpiredNotifications(ctx)
			if err != nil {
				slog.Error("Failed to cleanup expired notifications", slog.String("error", err.Error()))
			} else if count > 0 {
				slog.Info("Cleaned up expired notifications", slog.Int64("count", count))
			}
		}
	}
}

// NotificationService interface for other modules to send notifications
type NotificationService interface {
	SendSystemNotification(ctx context.Context, title, message string, recipients []int) error
	SendAlertNotification(ctx context.Context, title, message string, recipients []int, priority string) error
}

// SendSystemNotification sends a system notification (for use by other modules)
func (m *Module) SendSystemNotification(ctx context.Context, title, message string, recipients []int) error {
	_, err := m.service.SendSystemNotification(ctx, title, message, recipients)
	return err
}

// SendAlertNotification sends an alert notification (for use by other modules)
func (m *Module) SendAlertNotification(ctx context.Context, title, message string, recipients []int, priority string) error {
	_, err := m.service.SendAlertNotification(ctx, title, message, recipients, priority)
	return err
}