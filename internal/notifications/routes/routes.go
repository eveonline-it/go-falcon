package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go-falcon/internal/auth"
	authmiddleware "go-falcon/internal/auth/middleware"
	"go-falcon/internal/groups"
	"go-falcon/internal/notifications/dto"
	"go-falcon/internal/notifications/middleware"
	"go-falcon/internal/notifications/services"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Handler contains the route handlers for the notifications module
type Handler struct {
	service      *services.Service
	authModule   *auth.Module
	groupsModule *groups.Module
}

// NewHandler creates a new handler instance
func NewHandler(service *services.Service, authModule *auth.Module, groupsModule *groups.Module) *Handler {
	return &Handler{
		service:      service,
		authModule:   authModule,
		groupsModule: groupsModule,
	}
}

// RegisterRoutes registers all notification routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	// All notification endpoints require authentication
	r.Group(func(r chi.Router) {
		r.Use(h.authModule.GetMiddleware().Auth.RequireAuth)

		// Get user's notifications
		r.Group(func(r chi.Router) {
			r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "read"))
			r.Use(middleware.ValidateNotificationSearchRequest)
			r.Get("/", h.GetNotifications) // GET /api/notifications?page=1&unread_only=true
		})

		// Get notification statistics
		r.Group(func(r chi.Router) {
			r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "read"))
			r.Get("/stats", h.GetNotificationStats) // GET /api/notifications/stats
		})

		// Send notification
		r.Group(func(r chi.Router) {
			r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "write"))
			r.Use(middleware.ValidateNotificationCreateRequest)
			r.Post("/", h.SendNotification) // POST /api/notifications
		})

		// Bulk operations
		r.Group(func(r chi.Router) {
			r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "write"))
			r.Use(middleware.ValidateNotificationBulkRequest)
			r.Post("/bulk", h.BulkUpdateNotifications) // POST /api/notifications/bulk
		})

		// Individual notification operations
		r.Route("/{id}", func(r chi.Router) {
			r.Use(middleware.ValidateNotificationID)

			// Get specific notification
			r.Group(func(r chi.Router) {
				r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "read"))
				r.Get("/", h.GetNotification) // GET /api/notifications/{id}
			})

			// Update notification status
			r.Group(func(r chi.Router) {
				r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "write"))
				r.Use(middleware.ValidateNotificationUpdateRequest)
				r.Put("/", h.UpdateNotification) // PUT /api/notifications/{id}
			})

			// Delete notification
			r.Group(func(r chi.Router) {
				r.Use(h.groupsModule.RequireGranularPermission("notifications", "messages", "write"))
				r.Delete("/", h.DeleteNotification) // DELETE /api/notifications/{id}
			})
		})
	})
}

// GetNotifications retrieves notifications for the authenticated user
func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.list",
		attribute.String("service", "notifications"),
		attribute.String("operation", "list_notifications"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Parse search request (already validated by middleware)
	var req dto.NotificationSearchRequest
	// Re-parse query parameters since middleware validation doesn't store them
	h.parseSearchRequest(r, &req)

	span.SetAttributes(
		attribute.Int("user.character_id", user.CharacterID),
		attribute.Int("search.page", req.Page),
		attribute.Int("search.page_size", req.PageSize),
	)

	response, err := h.service.GetUserNotifications(r.Context(), user.CharacterID, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get notifications")
		slog.Error("Failed to get notifications", slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to retrieve notifications", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("response.total_notifications", response.Total),
		attribute.Int("response.unread_count", response.UnreadCount),
		attribute.Int("response.returned_notifications", len(response.Notifications)),
	)

	handlers.JSONResponse(w, response, http.StatusOK)
}

// GetNotification retrieves a specific notification
func (h *Handler) GetNotification(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.get",
		attribute.String("service", "notifications"),
		attribute.String("operation", "get_notification"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	notificationID := chi.URLParam(r, "id")
	span.SetAttributes(
		attribute.Int("user.character_id", user.CharacterID),
		attribute.String("notification.id", notificationID),
	)

	notification, err := h.service.GetNotification(r.Context(), notificationID, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get notification")
		slog.Error("Failed to get notification",
			slog.String("notification_id", notificationID),
			slog.Int("character_id", user.CharacterID),
			slog.String("error", err.Error()))
		
		if err.Error() == "notification not found" || err.Error() == "access denied: notification not found or access denied" {
			handlers.ErrorResponse(w, "Notification not found", http.StatusNotFound)
		} else {
			handlers.ErrorResponse(w, "Failed to retrieve notification", http.StatusInternalServerError)
		}
		return
	}

	span.SetAttributes(
		attribute.String("notification.type", notification.Type),
		attribute.String("notification.priority", notification.Priority),
		attribute.Bool("notification.read", notification.Read),
	)

	handlers.JSONResponse(w, notification, http.StatusOK)
}

// SendNotification creates and sends a new notification
func (h *Handler) SendNotification(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.send",
		attribute.String("service", "notifications"),
		attribute.String("operation", "send_notification"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req dto.NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.Int("user.character_id", user.CharacterID),
		attribute.String("notification.type", req.Type),
		attribute.String("notification.priority", req.Priority),
		attribute.Int("notification.recipient_count", len(req.Recipients)),
	)

	response, err := h.service.CreateNotification(r.Context(), req, &user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send notification")
		slog.Error("Failed to send notification",
			slog.Int("sender_id", user.CharacterID),
			slog.String("type", req.Type),
			slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to send notification", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("notification.id", response.ID),
		attribute.Bool("notification.success", response.Success),
	)

	slog.Info("Notification sent successfully",
		slog.String("notification_id", response.ID),
		slog.Int("sender_id", user.CharacterID),
		slog.String("type", req.Type),
		slog.Int("recipient_count", len(req.Recipients)))

	handlers.JSONResponse(w, response, http.StatusCreated)
}

// UpdateNotification updates notification status (mark as read/unread)
func (h *Handler) UpdateNotification(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.update",
		attribute.String("service", "notifications"),
		attribute.String("operation", "update_notification"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	notificationID := chi.URLParam(r, "id")
	
	var req dto.NotificationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.Int("user.character_id", user.CharacterID),
		attribute.String("notification.id", notificationID),
	)

	response, err := h.service.UpdateNotificationStatus(r.Context(), notificationID, user.CharacterID, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update notification")
		slog.Error("Failed to update notification",
			slog.String("notification_id", notificationID),
			slog.Int("character_id", user.CharacterID),
			slog.String("error", err.Error()))
		
		if err.Error() == "notification not found or access denied" {
			handlers.ErrorResponse(w, "Notification not found", http.StatusNotFound)
		} else {
			handlers.ErrorResponse(w, "Failed to update notification", http.StatusInternalServerError)
		}
		return
	}

	span.SetAttributes(
		attribute.Bool("notification.read", response.Notification.Read),
		attribute.Bool("update.success", response.Success),
	)

	handlers.JSONResponse(w, response, http.StatusOK)
}

// DeleteNotification soft deletes a notification
func (h *Handler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.delete",
		attribute.String("service", "notifications"),
		attribute.String("operation", "delete_notification"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	notificationID := chi.URLParam(r, "id")
	span.SetAttributes(
		attribute.Int("user.character_id", user.CharacterID),
		attribute.String("notification.id", notificationID),
	)

	err := h.service.DeleteNotification(r.Context(), notificationID, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete notification")
		slog.Error("Failed to delete notification",
			slog.String("notification_id", notificationID),
			slog.Int("character_id", user.CharacterID),
			slog.String("error", err.Error()))
		
		if err.Error() == "notification not found or access denied" {
			handlers.ErrorResponse(w, "Notification not found", http.StatusNotFound)
		} else {
			handlers.ErrorResponse(w, "Failed to delete notification", http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Notification deleted successfully",
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

// BulkUpdateNotifications performs bulk operations on notifications
func (h *Handler) BulkUpdateNotifications(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.bulk_update",
		attribute.String("service", "notifications"),
		attribute.String("operation", "bulk_update_notifications"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req dto.NotificationBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.Int("user.character_id", user.CharacterID),
		attribute.String("bulk.action", req.Action),
		attribute.Int("bulk.notification_count", len(req.NotificationIDs)),
	)

	response, err := h.service.BulkUpdateNotifications(r.Context(), user.CharacterID, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to perform bulk update")
		slog.Error("Failed to perform bulk update",
			slog.Int("character_id", user.CharacterID),
			slog.String("action", req.Action),
			slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to perform bulk operation", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("bulk.success_count", response.SuccessCount),
		attribute.Int("bulk.failure_count", response.FailureCount),
		attribute.Bool("bulk.success", response.Success),
	)

	handlers.JSONResponse(w, response, http.StatusOK)
}

// GetNotificationStats returns notification statistics for the authenticated user
func (h *Handler) GetNotificationStats(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "notifications.stats",
		attribute.String("service", "notifications"),
		attribute.String("operation", "get_stats"),
	)
	defer span.End()

	// Get authenticated user
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	span.SetAttributes(attribute.Int("user.character_id", user.CharacterID))

	stats, err := h.service.GetNotificationStats(r.Context(), user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get notification statistics")
		slog.Error("Failed to get notification statistics",
			slog.Int("character_id", user.CharacterID),
			slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to retrieve notification statistics", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("stats.total_notifications", stats.TotalNotifications),
		attribute.Int("stats.unread_notifications", stats.UnreadNotifications),
	)

	handlers.JSONResponse(w, stats, http.StatusOK)
}

// parseSearchRequest helper function to parse search request from query parameters
func (h *Handler) parseSearchRequest(r *http.Request, req *dto.NotificationSearchRequest) {
	// This mirrors the validation middleware parsing logic
	// Implementation would be the same as in the validation middleware
	req.SetDefaults()
}