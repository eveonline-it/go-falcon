package routes

import (
	"context"

	"go-falcon/internal/notifications/dto"
	"go-falcon/internal/notifications/services"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles Huma-based HTTP routing for the Notifications module
type Routes struct {
	service *services.Service
	api     huma.API
}

// NewRoutes creates a new Huma Notifications routes handler
func NewRoutes(service *services.Service, router chi.Router) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Notifications Module", "1.0.0")
	config.Info.Description = "Notification management and delivery system"
	
	api := humachi.New(router, config)

	hr := &Routes{
		service: service,
		api:     api,
	}

	// Register all routes
	hr.registerRoutes()

	return hr
}

// RegisterNotificationsRoutes registers notifications routes on a shared Huma API
func RegisterNotificationsRoutes(api huma.API, basePath string, service *services.Service) {
	// Notification management endpoints
	huma.Get(api, basePath+"/notifications", func(ctx context.Context, input *dto.NotificationListInput) (*dto.NotificationListOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("Notification listing not yet implemented")
	})

	huma.Post(api, basePath+"/notifications", func(ctx context.Context, input *dto.NotificationCreateInput) (*dto.NotificationCreateOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("Notification creation not yet implemented")
	})

	huma.Get(api, basePath+"/notifications/{notification_id}", func(ctx context.Context, input *dto.NotificationGetInput) (*dto.NotificationGetOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("Notification retrieval not yet implemented")
	})

	huma.Put(api, basePath+"/notifications/{notification_id}/read", func(ctx context.Context, input *dto.NotificationMarkReadInput) (*dto.NotificationMarkReadOutput, error) {
		// TODO: Implement once service method is available
		return nil, huma.Error501NotImplemented("Marking notification as read not yet implemented")
	})
}

// registerRoutes registers all Notifications module routes with Huma
func (hr *Routes) registerRoutes() {
	// Notification management endpoints
	huma.Get(hr.api, "/notifications", hr.listNotifications)
	huma.Post(hr.api, "/notifications", hr.createNotification)
	huma.Get(hr.api, "/notifications/{notification_id}", hr.getNotification)
	huma.Put(hr.api, "/notifications/{notification_id}/read", hr.markNotificationRead)
}

// Notification handlers

func (hr *Routes) listNotifications(ctx context.Context, input *dto.NotificationListInput) (*dto.NotificationListOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("Notification listing not yet implemented")
}

func (hr *Routes) createNotification(ctx context.Context, input *dto.NotificationCreateInput) (*dto.NotificationCreateOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("Notification creation not yet implemented")
}

func (hr *Routes) getNotification(ctx context.Context, input *dto.NotificationGetInput) (*dto.NotificationGetOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("Notification retrieval not yet implemented")
}

func (hr *Routes) markNotificationRead(ctx context.Context, input *dto.NotificationMarkReadInput) (*dto.NotificationMarkReadOutput, error) {
	// TODO: Implement once service method is available
	return nil, huma.Error501NotImplemented("Marking notification as read not yet implemented")
}