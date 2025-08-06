package users

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"go-falcon/internal/auth"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// getUserHandler retrieves a specific user by character ID
func (m *Module) getUserHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.get",
		attribute.String("service", "users"),
		attribute.String("operation", "get_user"),
	)
	defer span.End()

	characterIDStr := chi.URLParam(r, "character_id")
	if characterIDStr == "" {
		span.SetStatus(codes.Error, "Missing character_id parameter")
		http.Error(w, "Missing character_id parameter", http.StatusBadRequest)
		return
	}

	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid character_id parameter")
		http.Error(w, "Invalid character_id parameter", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("user.character_id", characterID))

	user, err := m.GetUser(r.Context(), characterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user")
		slog.Error("Failed to get user", 
			slog.Int("character_id", characterID),
			slog.String("error", err.Error()))
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	span.SetAttributes(
		attribute.String("user.character_name", user.CharacterName),
		attribute.String("user.user_id", user.UserID),
		attribute.Bool("user.enabled", user.Enabled),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// listUsersHandler retrieves users with pagination and filtering
func (m *Module) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.list",
		attribute.String("service", "users"),
		attribute.String("operation", "list_users"),
	)
	defer span.End()

	// Parse query parameters
	var req UserSearchRequest
	
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

	response, err := m.ListUsers(r.Context(), req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list users")
		slog.Error("Failed to list users", slog.String("error", err.Error()))
		http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("response.total_users", response.Total),
		attribute.Int("response.returned_users", len(response.Users)),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateUserHandler updates a user's status and administrative fields
func (m *Module) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.update",
		attribute.String("service", "users"),
		attribute.String("operation", "update_user"),
	)
	defer span.End()

	characterIDStr := chi.URLParam(r, "character_id")
	if characterIDStr == "" {
		span.SetStatus(codes.Error, "Missing character_id parameter")
		http.Error(w, "Missing character_id parameter", http.StatusBadRequest)
		return
	}

	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid character_id parameter")
		http.Error(w, "Invalid character_id parameter", http.StatusBadRequest)
		return
	}

	var req UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(attribute.Int("user.character_id", characterID))

	user, err := m.UpdateUser(r.Context(), characterID, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update user")
		slog.Error("Failed to update user", 
			slog.Int("character_id", characterID),
			slog.String("error", err.Error()))
		
		if err.Error() == "user not found for character ID "+strconv.Itoa(characterID) {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
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

	response := map[string]interface{}{
		"success": true,
		"message": "User updated successfully",
		"user":    user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getUserStatsHandler returns user statistics
func (m *Module) getUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.stats",
		attribute.String("service", "users"),
		attribute.String("operation", "get_stats"),
	)
	defer span.End()

	stats, err := m.GetUserStats(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user statistics")
		slog.Error("Failed to get user statistics", slog.String("error", err.Error()))
		http.Error(w, "Failed to retrieve user statistics", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("stats.total_users", stats.TotalUsers),
		attribute.Int("stats.enabled_users", stats.EnabledUsers),
		attribute.Int("stats.banned_users", stats.BannedUsers),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// listCharactersHandler retrieves character summaries for a specific user
// Allows users to view their own characters or admins to view any user's characters
func (m *Module) listCharactersHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "users.characters",
		attribute.String("service", "users"),
		attribute.String("operation", "list_characters"),
	)
	defer span.End()

	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "Missing user_id parameter")
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	// Get authenticated user from context
	user, authenticated := auth.GetAuthenticatedUser(r)
	if !authenticated {
		span.SetStatus(codes.Error, "User not authenticated")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	span.SetAttributes(
		attribute.String("user.user_id", userID),
		attribute.String("auth.user_id", user.UserID),
		attribute.Int("auth.character_id", user.CharacterID),
	)

	// Check if user is requesting their own characters or if they have admin permissions
	if user.UserID != userID {
		// User is requesting someone else's characters - check admin permission
		allowed, err := m.groupsModule.CheckPermissionInHandler(r, "users", "read")
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Permission check failed")
			slog.Error("Permission check failed", slog.String("error", err.Error()))
			http.Error(w, "Permission check failed", http.StatusInternalServerError)
			return
		}
		
		if !allowed {
			span.SetStatus(codes.Error, "Insufficient permissions")
			slog.Warn("User attempted to access other user's characters",
				slog.String("requesting_user_id", user.UserID),
				slog.String("target_user_id", userID),
				slog.String("character_name", user.CharacterName))
			http.Error(w, "Access denied: can only view your own characters", http.StatusForbidden)
			return
		}
	}

	characters, err := m.ListCharacters(r.Context(), userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list characters")
		slog.Error("Failed to list characters", 
			slog.String("user_id", userID),
			slog.String("error", err.Error()))
		http.Error(w, "Failed to retrieve characters", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("response.character_count", len(characters)))

	response := map[string]interface{}{
		"user_id":    userID,
		"characters": characters,
		"count":      len(characters),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

