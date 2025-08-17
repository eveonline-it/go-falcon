package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"go-falcon/internal/auth"
	authmiddleware "go-falcon/internal/auth/middleware"
	"go-falcon/internal/groups"
	"go-falcon/internal/users/dto"
	"go-falcon/internal/users/middleware"
	"go-falcon/internal/users/services"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Handler contains the route handlers for the users module
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

// RegisterRoutes registers all users routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	// Public endpoints - basic user information (no authentication required)
	r.Get("/stats", h.GetUserStats) // GET /api/users/stats

	// Administrative endpoints - require authentication and granular permissions
	r.Group(func(r chi.Router) {
		r.Use(h.groupsModule.RequireGranularPermission("users", "profiles", "read"))
		r.Use(middleware.ValidateUserSearchRequest)
		r.Get("/", h.ListUsers) // GET /api/users?page=1&page_size=20&query=search
	})

	r.Group(func(r chi.Router) {
		r.Use(h.groupsModule.RequireGranularPermission("users", "profiles", "read"))
		r.Use(middleware.ValidateCharacterID)
		r.Get("/{character_id}", h.GetUser) // GET /api/users/{character_id}
	})

	r.Group(func(r chi.Router) {
		r.Use(h.groupsModule.RequireGranularPermission("users", "profiles", "write"))
		r.Use(middleware.ValidateCharacterID)
		r.Use(middleware.ValidateUserUpdateRequest)
		r.Put("/{character_id}", h.UpdateUser) // PUT /api/users/{character_id}
	})

	// User-specific character management - requires authentication, users can view their own characters or admins can view any
	r.Group(func(r chi.Router) {
		r.Use(h.authModule.GetMiddleware().Auth.RequireAuth)
		r.Use(middleware.ValidateUserID)
		r.Get("/by-user-id/{user_id}/characters", h.ListCharacters) // GET /api/users/by-user-id/{user_id}/characters
	})
}

// GetUserStats returns user statistics
func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.stats",
		attribute.String("service", "users"),
		attribute.String("operation", "get_stats"),
	)
	defer span.End()

	stats, err := h.service.GetUserStats(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user statistics")
		slog.Error("Failed to get user statistics", slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to retrieve user statistics", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("stats.total_users", stats.TotalUsers),
		attribute.Int("stats.enabled_users", stats.EnabledUsers),
		attribute.Int("stats.banned_users", stats.BannedUsers),
	)

	handlers.JSONResponse(w, stats, http.StatusOK)
}

// ListUsers retrieves users with pagination and filtering
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.list",
		attribute.String("service", "users"),
		attribute.String("operation", "list_users"),
	)
	defer span.End()

	// Parse query parameters (already validated by middleware)
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

	span.SetAttributes(
		attribute.String("search.query", req.Query),
		attribute.Int("pagination.page", req.Page),
		attribute.Int("pagination.page_size", req.PageSize),
	)

	response, err := h.service.ListUsers(r.Context(), req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list users")
		slog.Error("Failed to list users", slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("response.total_users", response.Total),
		attribute.Int("response.returned_users", len(response.Users)),
	)

	handlers.JSONResponse(w, response, http.StatusOK)
}

// GetUser retrieves a specific user by character ID
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.get",
		attribute.String("service", "users"),
		attribute.String("operation", "get_user"),
	)
	defer span.End()

	characterIDStr := chi.URLParam(r, "character_id")
	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid character_id parameter")
		handlers.ErrorResponse(w, "Invalid character_id parameter", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("user.character_id", characterID))

	user, err := h.service.GetUser(r.Context(), characterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user")
		slog.Error("Failed to get user", 
			slog.Int("character_id", characterID),
			slog.String("error", err.Error()))
		
		if err.Error() == "user not found for character ID "+strconv.Itoa(characterID) {
			handlers.ErrorResponse(w, "User not found", http.StatusNotFound)
		} else {
			handlers.ErrorResponse(w, "Failed to retrieve user", http.StatusInternalServerError)
		}
		return
	}

	span.SetAttributes(
		attribute.String("user.character_name", user.CharacterName),
		attribute.String("user.user_id", user.UserID),
		attribute.Bool("user.enabled", user.Enabled),
	)

	// Convert to response DTO
	userResponse := dto.UserResponse{
		CharacterID:   user.CharacterID,
		UserID:        user.UserID,
		Enabled:       user.Enabled,
		Banned:        user.Banned,
		Invalid:       user.Invalid,
		Scopes:        user.Scopes,
		Position:      user.Position,
		Notes:         user.Notes,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		LastLogin:     user.LastLogin,
		CharacterName: user.CharacterName,
		Valid:         user.Valid,
	}

	handlers.JSONResponse(w, userResponse, http.StatusOK)
}

// UpdateUser updates a user's status and administrative fields
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.update",
		attribute.String("service", "users"),
		attribute.String("operation", "update_user"),
	)
	defer span.End()

	characterIDStr := chi.URLParam(r, "character_id")
	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid character_id parameter")
		handlers.ErrorResponse(w, "Invalid character_id parameter", http.StatusBadRequest)
		return
	}

	var req dto.UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("user.character_id", characterID))

	user, err := h.service.UpdateUser(r.Context(), characterID, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update user")
		slog.Error("Failed to update user", 
			slog.Int("character_id", characterID),
			slog.String("error", err.Error()))
		
		if err.Error() == "user not found for character ID "+strconv.Itoa(characterID) {
			handlers.ErrorResponse(w, "User not found", http.StatusNotFound)
		} else {
			handlers.ErrorResponse(w, "Failed to update user", http.StatusInternalServerError)
		}
		return
	}

	span.SetAttributes(
		attribute.String("user.character_name", user.CharacterName),
		attribute.Bool("user.enabled", user.Enabled),
		attribute.Bool("user.banned", user.Banned),
	)

	slog.Info("User updated successfully",
		slog.Int("character_id", characterID),
		slog.String("character_name", user.CharacterName))

	// Convert to response DTO
	userResponse := dto.UserResponse{
		CharacterID:   user.CharacterID,
		UserID:        user.UserID,
		Enabled:       user.Enabled,
		Banned:        user.Banned,
		Invalid:       user.Invalid,
		Scopes:        user.Scopes,
		Position:      user.Position,
		Notes:         user.Notes,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		LastLogin:     user.LastLogin,
		CharacterName: user.CharacterName,
		Valid:         user.Valid,
	}

	response := dto.UserUpdateResponse{
		Success: true,
		Message: "User updated successfully",
		User:    userResponse,
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}

// ListCharacters retrieves character summaries for a specific user
// Allows users to view their own characters or admins to view any user's characters
func (h *Handler) ListCharacters(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.characters",
		attribute.String("service", "users"),
		attribute.String("operation", "list_characters"),
	)
	defer span.End()

	userID := chi.URLParam(r, "user_id")

	// Get authenticated user from context
	user := authmiddleware.GetAuthenticatedUser(r)
	if user == nil {
		span.SetStatus(codes.Error, "User not authenticated")
		handlers.ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	span.SetAttributes(
		attribute.String("user.user_id", userID),
		attribute.String("auth.user_id", user.UserID),
		attribute.Int("auth.character_id", user.CharacterID),
	)

	// Check if user is requesting their own characters or if they have admin permissions
	if user.UserID != userID {
		// User is requesting someone else's characters - check granular admin permission
		allowed, err := h.groupsModule.CheckGranularPermission(r.Context(), user.CharacterID, "users", "profiles", "read")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Permission check failed")
			slog.Error("Granular permission check failed", slog.String("error", err.Error()))
			handlers.ErrorResponse(w, "Permission check failed", http.StatusInternalServerError)
			return
		}
		
		if !allowed {
			span.SetStatus(codes.Error, "Insufficient permissions")
			slog.Warn("User attempted to access other user's characters",
				slog.String("requesting_user_id", user.UserID),
				slog.String("target_user_id", userID),
				slog.String("character_name", user.CharacterName))
			handlers.ErrorResponse(w, "Access denied: can only view your own characters", http.StatusForbidden)
			return
		}
	}

	characters, err := h.service.ListCharacters(r.Context(), userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list characters")
		slog.Error("Failed to list characters", 
			slog.String("user_id", userID),
			slog.String("error", err.Error()))
		handlers.ErrorResponse(w, "Failed to retrieve characters", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("response.character_count", len(characters)))

	response := dto.CharacterListResponse{
		UserID:     userID,
		Characters: characters,
		Count:      len(characters),
	}

	handlers.JSONResponse(w, response, http.StatusOK)
}