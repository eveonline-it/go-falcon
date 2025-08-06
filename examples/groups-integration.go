package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"go-falcon/internal/auth"
	"go-falcon/internal/groups"
	"go-falcon/pkg/app"

	"github.com/go-chi/chi/v5"
)

// Example of integrating the groups module with a custom service
func main() {
	// Initialize application context
	appCtx, err := app.InitializeApp("groups-example")
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer appCtx.Shutdown(context.Background())

	// Initialize modules
	authModule := auth.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService)
	groupsModule := groups.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, authModule)

	// Initialize router
	r := chi.NewRouter()

	// Example: Public endpoint that shows different content based on permissions
	r.Get("/content", groupsModule.OptionalPermissionMiddleware(publicContentHandler))

	// Example: Admin-only endpoint
	r.With(authModule.JWTMiddleware, groupsModule.RequirePermission("groups", "admin")).
		Post("/admin/cleanup", adminCleanupHandler(groupsModule))

	// Example: Corporate member endpoint
	r.With(authModule.JWTMiddleware, groupsModule.RequireGroup("corporate")).
		Get("/corporate/data", corporateDataHandler)

	// Example: Multi-group access endpoint
	r.With(authModule.JWTMiddleware, groupsModule.RequireAnyGroup("administrators", "moderators")).
		Get("/moderation/queue", moderationQueueHandler)

	// Example: Resource owner or admin access
	ownerExtractor := func(r *http.Request) int {
		// Extract user ID from URL parameter
		userIDStr := chi.URLParam(r, "userID")
		// In real implementation, convert to int
		return 12345 // placeholder
	}

	r.With(authModule.JWTMiddleware, groupsModule.ResourceOwnerOrPermission(ownerExtractor, "user", "admin")).
		Put("/users/{userID}/profile", updateProfileHandler)

	// Example: Conditional permissions based on request
	conditionalCheck := func(r *http.Request) bool {
		// Apply permission check only for certain operations
		return r.Header.Get("X-Dangerous-Operation") == "true"
	}

	r.With(authModule.JWTMiddleware, groupsModule.ConditionalPermission(conditionalCheck, "system", "admin")).
		Post("/system/operations", dangerousOperationHandler)

	// Start server
	fmt.Println("Groups integration example running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Handler that shows different content based on user permissions
func publicContentHandler(w http.ResponseWriter, r *http.Request) {
	// Get user permissions from context (set by OptionalPermissionMiddleware)
	permissions, hasPermissions := groups.GetUserPermissionsFromContext(r)
	
	response := map[string]interface{}{
		"message": "Welcome to our service!",
		"content": "public",
	}

	if hasPermissions && !permissions.IsGuest {
		response["content"] = "authenticated"
		response["character_id"] = permissions.CharacterID
		response["groups"] = permissions.Groups

		// Show additional content for specific groups
		if containsGroup(permissions.Groups, "corporate") {
			response["corporate_news"] = "Latest corporate updates..."
		}
		
		if containsGroup(permissions.Groups, "administrators") {
			response["admin_panel"] = "/admin"
		}
	} else {
		response["message"] = "Public content only - please authenticate for more features"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Admin cleanup handler
func adminCleanupHandler(groupsModule *groups.Module) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := auth.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Perform cleanup operations
		groupService := groupsModule.GetGroupService()
		cleaned, err := groupService.CleanupExpiredMemberships(r.Context())
		if err != nil {
			http.Error(w, "Cleanup failed", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"message":         "Cleanup completed successfully",
			"expired_cleaned": cleaned,
			"performed_by":    user.CharacterName,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// Corporate data handler
func corporateDataHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"message":        "Corporate data access granted",
		"character_name": user.CharacterName,
		"data": map[string]interface{}{
			"corporate_structures": "Structure list...",
			"corp_wallet_balance":  "Balance info...",
			"alliance_standings":   "Standings data...",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Moderation queue handler
func moderationQueueHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get group membership info from context
	groupInfo := r.Context().Value(groups.GroupPermissionContextKeyPermissions)

	response := map[string]interface{}{
		"message":       "Moderation queue access",
		"moderator":     user.CharacterName,
		"queue_items":   []string{"Item 1", "Item 2", "Item 3"},
		"access_level":  groupInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Update profile handler
func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	userID := chi.URLParam(r, "userID")
	
	response := map[string]interface{}{
		"message":    "Profile update authorized",
		"target_user": userID,
		"updated_by": user.CharacterName,
		"timestamp":  "2024-01-01T10:00:00Z",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Dangerous operation handler
func dangerousOperationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"message":     "Dangerous operation authorized",
		"operator":    user.CharacterName,
		"operation":   "system maintenance",
		"warning":     "This operation requires administrator privileges",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function
func containsGroup(groups []string, target string) bool {
	for _, group := range groups {
		if group == target {
			return true
		}
	}
	return false
}