package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-falcon/internal/users/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// ValidateCharacterID validates and extracts character ID from URL parameter
func ValidateCharacterID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "users.validate_character_id",
			attribute.String("service", "users"),
			attribute.String("operation", "validate_character_id"),
		)
		defer span.End()

		characterIDStr := chi.URLParam(r, "character_id")
		if characterIDStr == "" {
			span.SetStatus(codes.Error, "Missing character_id parameter")
			handlers.ErrorResponse(w, "Missing character_id parameter", http.StatusBadRequest)
			return
		}

		characterID, err := strconv.Atoi(characterIDStr)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid character_id parameter")
			handlers.ErrorResponse(w, "Invalid character_id parameter", http.StatusBadRequest)
			return
		}

		if characterID <= 0 {
			span.SetStatus(codes.Error, "Character ID must be positive")
			handlers.ErrorResponse(w, "Character ID must be positive", http.StatusBadRequest)
			return
		}

		span.SetAttributes(attribute.Int("user.character_id", characterID))
		next.ServeHTTP(w, r)
	})
}

// ValidateUserID validates and extracts user ID from URL parameter
func ValidateUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "users.validate_user_id",
			attribute.String("service", "users"),
			attribute.String("operation", "validate_user_id"),
		)
		defer span.End()

		userID := chi.URLParam(r, "user_id")
		if userID == "" {
			span.SetStatus(codes.Error, "Missing user_id parameter")
			handlers.ErrorResponse(w, "Missing user_id parameter", http.StatusBadRequest)
			return
		}

		span.SetAttributes(attribute.String("user.user_id", userID))
		next.ServeHTTP(w, r)
	})
}

// ValidateUserSearchRequest validates query parameters for user search
func ValidateUserSearchRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "users.validate_search_request",
			attribute.String("service", "users"),
			attribute.String("operation", "validate_search_request"),
		)
		defer span.End()

		// Parse query parameters
		var req dto.UserSearchRequest
		
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
		
		// Parse search and filters
		req.Query = r.URL.Query().Get("query")
		req.SortBy = r.URL.Query().Get("sort_by")
		req.SortOrder = r.URL.Query().Get("sort_order")
		
		if enabledStr := r.URL.Query().Get("enabled"); enabledStr != "" {
			if enabled, err := strconv.ParseBool(enabledStr); err == nil {
				req.Enabled = &enabled
			}
		}
		if bannedStr := r.URL.Query().Get("banned"); bannedStr != "" {
			if banned, err := strconv.ParseBool(bannedStr); err == nil {
				req.Banned = &banned
			}
		}
		if invalidStr := r.URL.Query().Get("invalid"); invalidStr != "" {
			if invalid, err := strconv.ParseBool(invalidStr); err == nil {
				req.Invalid = &invalid
			}
		}
		if positionStr := r.URL.Query().Get("position"); positionStr != "" {
			if position, err := strconv.Atoi(positionStr); err == nil {
				req.Position = &position
			}
		}

		// Set defaults
		req.SetDefaults()

		// Validate the request
		if err := dto.ValidateUserSearchRequest(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid search request")
			handlers.ErrorResponse(w, "Invalid search parameters: "+err.Error(), http.StatusBadRequest)
			return
		}

		span.SetAttributes(
			attribute.String("search.query", req.Query),
			attribute.Int("pagination.page", req.Page),
			attribute.Int("pagination.page_size", req.PageSize),
		)

		next.ServeHTTP(w, r)
	})
}

// ValidateUserUpdateRequest validates the JSON body for user updates
func ValidateUserUpdateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span, r := handlers.StartHTTPSpan(r, "users.validate_update_request",
			attribute.String("service", "users"),
			attribute.String("operation", "validate_update_request"),
		)
		defer span.End()

		var req dto.UserUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")
			handlers.ErrorResponse(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate the request
		if err := dto.ValidateUserUpdateRequest(&req); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid update request")
			handlers.ErrorResponse(w, "Invalid update request: "+err.Error(), http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}