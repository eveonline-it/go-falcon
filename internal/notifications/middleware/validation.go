package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-falcon/internal/notifications/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// ValidateNotificationID validates and extracts notification ID from URL parameter
func ValidateNotificationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "notifications.validate_id",
			attribute.String("service", "notifications"),
			attribute.String("operation", "validate_notification_id"),
		)
		defer span.End()

		notificationID := chi.URLParam(r, "id")
		if notificationID == "" {
			span.SetStatus(codes.Error, "Missing notification ID parameter")
			handlers.ErrorResponse(w, "Missing notification ID parameter", http.StatusBadRequest)
			return
		}

		if len(notificationID) < 10 {
			span.SetStatus(codes.Error, "Invalid notification ID format")
			handlers.ErrorResponse(w, "Invalid notification ID format", http.StatusBadRequest)
			return
		}

		span.SetAttributes(attribute.String("notification.id", notificationID))
		next.ServeHTTP(w, r)
	})
}

// ValidateNotificationCreateRequest validates the JSON body for notification creation
func ValidateNotificationCreateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "notifications.validate_create_request",
			attribute.String("service", "notifications"),
			attribute.String("operation", "validate_create_request"),
		)
		defer span.End()

		var req dto.NotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			handlers.ErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Set defaults
		req.SetDefaults()

		// Validate the request
		if err := dto.ValidateNotificationRequest(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid notification request")
			handlers.ErrorResponse(w, "Invalid notification request: "+err.Error(), http.StatusBadRequest)
			return
		}

		span.SetAttributes(
			attribute.String("notification.type", req.Type),
			attribute.String("notification.priority", req.Priority),
			attribute.Int("notification.recipient_count", len(req.Recipients)),
		)

		next.ServeHTTP(w, r)
	})
}

// ValidateNotificationUpdateRequest validates the JSON body for notification updates
func ValidateNotificationUpdateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "notifications.validate_update_request",
			attribute.String("service", "notifications"),
			attribute.String("operation", "validate_update_request"),
		)
		defer span.End()

		var req dto.NotificationUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			handlers.ErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate the request
		if err := dto.ValidateNotificationUpdateRequest(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid update request")
			handlers.ErrorResponse(w, "Invalid update request: "+err.Error(), http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateNotificationBulkRequest validates the JSON body for bulk operations
func ValidateNotificationBulkRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "notifications.validate_bulk_request",
			attribute.String("service", "notifications"),
			attribute.String("operation", "validate_bulk_request"),
		)
		defer span.End()

		var req dto.NotificationBulkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			handlers.ErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate the request
		if err := dto.ValidateNotificationBulkRequest(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid bulk request")
			handlers.ErrorResponse(w, "Invalid bulk request: "+err.Error(), http.StatusBadRequest)
			return
		}

		span.SetAttributes(
			attribute.String("bulk.action", req.Action),
			attribute.Int("bulk.notification_count", len(req.NotificationIDs)),
		)

		next.ServeHTTP(w, r)
	})
}

// ValidateNotificationSearchRequest validates query parameters for notification search
func ValidateNotificationSearchRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "notifications.validate_search_request",
			attribute.String("service", "notifications"),
			attribute.String("operation", "validate_search_request"),
		)
		defer span.End()

		// Parse query parameters
		var req dto.NotificationSearchRequest

		// Parse pagination
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
				req.Page = page
			}
		}
		if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
			if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
				req.PageSize = pageSize
			}
		}

		// Parse filters
		if unreadOnlyStr := r.URL.Query().Get("unread_only"); unreadOnlyStr != "" {
			if unreadOnly, err := strconv.ParseBool(unreadOnlyStr); err == nil {
				req.UnreadOnly = &unreadOnly
			}
		}

		req.Type = r.URL.Query().Get("type")
		req.Priority = r.URL.Query().Get("priority")
		req.Category = r.URL.Query().Get("category")
		req.SortBy = r.URL.Query().Get("sort_by")
		req.SortOrder = r.URL.Query().Get("sort_order")

		// Set defaults
		req.SetDefaults()

		// Validate the request
		if err := dto.ValidateNotificationSearchRequest(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid search request")
			handlers.ErrorResponse(w, "Invalid search parameters: "+err.Error(), http.StatusBadRequest)
			return
		}

		span.SetAttributes(
			attribute.Int("search.page", req.Page),
			attribute.Int("search.page_size", req.PageSize),
			attribute.String("search.type", req.Type),
			attribute.String("search.priority", req.Priority),
		)

		next.ServeHTTP(w, r)
	})
}