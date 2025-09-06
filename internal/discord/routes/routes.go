package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"go-falcon/internal/discord/dto"
	"go-falcon/internal/discord/services"
	"go-falcon/pkg/config"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/middleware"
)

// Routes represents the Discord routes module
type Routes struct {
	service        *services.Service
	discordAdapter *middleware.DiscordAdapter
}

// NewRoutes creates a new Discord routes module
func NewRoutes(service *services.Service, discordAdapter *middleware.DiscordAdapter) *Routes {
	return &Routes{
		service:        service,
		discordAdapter: discordAdapter,
	}
}

// RegisterUnifiedRoutes registers Discord routes on the unified API
func (r *Routes) RegisterUnifiedRoutes(api huma.API) {
	// Authentication routes
	huma.Register(api, huma.Operation{
		OperationID: "getDiscordAuthURL",
		Method:      http.MethodGet,
		Path:        "/discord/auth/login",
		Summary:     "Get Discord OAuth authorization URL",
		Description: "Generate Discord OAuth authorization URL for user authentication or account linking",
		Tags:        []string{"Discord Authentication"},
	}, r.getDiscordAuthURL)

	huma.Register(api, huma.Operation{
		OperationID: "discordCallback",
		Method:      http.MethodGet,
		Path:        "/discord/auth/callback",
		Summary:     "Handle Discord OAuth callback",
		Description: "Process Discord OAuth callback and complete authentication or account linking",
		Tags:        []string{"Discord Authentication"},
	}, r.discordCallback)

	huma.Register(api, huma.Operation{
		OperationID: "linkDiscordAccount",
		Method:      http.MethodPost,
		Path:        "/discord/auth/link",
		Summary:     "Link Discord account to existing user",
		Description: "Link a Discord account to an existing Go Falcon user using OAuth tokens",
		Tags:        []string{"Discord Authentication"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.linkDiscordAccount)

	huma.Register(api, huma.Operation{
		OperationID: "unlinkDiscordAccount",
		Method:      http.MethodDelete,
		Path:        "/discord/auth/unlink/{discord_id}",
		Summary:     "Unlink Discord account",
		Description: "Unlink a Discord account from the current Go Falcon user",
		Tags:        []string{"Discord Authentication"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.unlinkDiscordAccount)

	huma.Register(api, huma.Operation{
		OperationID: "getDiscordAuthStatus",
		Method:      http.MethodGet,
		Path:        "/discord/auth/status",
		Summary:     "Get Discord authentication status",
		Description: "Check if user has Discord accounts linked and get status information",
		Tags:        []string{"Discord Authentication"},
	}, r.getDiscordAuthStatus)

	// User management routes
	huma.Register(api, huma.Operation{
		OperationID: "getDiscordUser",
		Method:      http.MethodGet,
		Path:        "/discord/users/{user_id}",
		Summary:     "Get Discord user information",
		Description: "Get Discord account information for a specific Go Falcon user",
		Tags:        []string{"Discord Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.getDiscordUser)

	huma.Register(api, huma.Operation{
		OperationID: "listDiscordUsers",
		Method:      http.MethodGet,
		Path:        "/discord/users",
		Summary:     "List Discord users",
		Description: "List all Discord users with filtering and pagination",
		Tags:        []string{"Discord Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.listDiscordUsers)

	// Guild management routes
	huma.Register(api, huma.Operation{
		OperationID: "createGuildConfig",
		Method:      http.MethodPost,
		Path:        "/discord/guilds",
		Summary:     "Create Discord guild configuration",
		Description: "Add a new Discord guild configuration with bot token for role management",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.createGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "getGuildConfig",
		Method:      http.MethodGet,
		Path:        "/discord/guilds/{guild_id}",
		Summary:     "Get Discord guild configuration",
		Description: "Get configuration details for a specific Discord guild",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.getGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "updateGuildConfig",
		Method:      http.MethodPut,
		Path:        "/discord/guilds/{guild_id}",
		Summary:     "Update Discord guild configuration",
		Description: "Update configuration for an existing Discord guild",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.updateGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "deleteGuildConfig",
		Method:      http.MethodDelete,
		Path:        "/discord/guilds/{guild_id}",
		Summary:     "Delete Discord guild configuration",
		Description: "Remove a Discord guild configuration and all associated role mappings",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.deleteGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "listGuildConfigs",
		Method:      http.MethodGet,
		Path:        "/discord/guilds",
		Summary:     "List Discord guild configurations",
		Description: "List all Discord guild configurations with filtering and pagination",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.listGuildConfigs)

	// Synchronization routes
	huma.Register(api, huma.Operation{
		OperationID: "triggerManualSync",
		Method:      http.MethodPost,
		Path:        "/discord/sync/manual",
		Summary:     "Trigger manual role synchronization",
		Description: "Manually trigger Discord role synchronization for all guilds or specific targets",
		Tags:        []string{"Discord Sync"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.triggerManualSync)

	huma.Register(api, huma.Operation{
		OperationID: "syncUser",
		Method:      http.MethodPost,
		Path:        "/discord/sync/user/{user_id}",
		Summary:     "Synchronize specific user roles",
		Description: "Synchronize Discord roles for a specific Go Falcon user",
		Tags:        []string{"Discord Sync"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.syncUser)

	huma.Register(api, huma.Operation{
		OperationID: "getSyncStatus",
		Method:      http.MethodGet,
		Path:        "/discord/sync/status",
		Summary:     "Get synchronization status",
		Description: "Get current and recent Discord role synchronization status",
		Tags:        []string{"Discord Sync"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.getSyncStatus)

	// Role mapping routes
	huma.Register(api, huma.Operation{
		OperationID: "createRoleMapping",
		Method:      http.MethodPost,
		Path:        "/discord/guilds/{guild_id}/role-mappings",
		Summary:     "Create Discord role mapping",
		Description: "Create a new mapping between a Go Falcon group and Discord role",
		Tags:        []string{"Discord Role Mappings"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.createRoleMapping)

	huma.Register(api, huma.Operation{
		OperationID: "listRoleMappings",
		Method:      http.MethodGet,
		Path:        "/discord/guilds/{guild_id}/role-mappings",
		Summary:     "List Discord role mappings",
		Description: "List role mappings for a specific Discord guild with filtering",
		Tags:        []string{"Discord Role Mappings"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.listRoleMappings)

	huma.Register(api, huma.Operation{
		OperationID: "getRoleMapping",
		Method:      http.MethodGet,
		Path:        "/discord/role-mappings/{mapping_id}",
		Summary:     "Get Discord role mapping",
		Description: "Get details for a specific Discord role mapping",
		Tags:        []string{"Discord Role Mappings"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.getRoleMapping)

	huma.Register(api, huma.Operation{
		OperationID: "updateRoleMapping",
		Method:      http.MethodPut,
		Path:        "/discord/role-mappings/{mapping_id}",
		Summary:     "Update Discord role mapping",
		Description: "Update an existing Discord role mapping",
		Tags:        []string{"Discord Role Mappings"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.updateRoleMapping)

	huma.Register(api, huma.Operation{
		OperationID: "deleteRoleMapping",
		Method:      http.MethodDelete,
		Path:        "/discord/role-mappings/{mapping_id}",
		Summary:     "Delete Discord role mapping",
		Description: "Delete a Discord role mapping",
		Tags:        []string{"Discord Role Mappings"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, r.deleteRoleMapping)

	// Module status route
	huma.Register(api, huma.Operation{
		OperationID: "getDiscordStatus",
		Method:      http.MethodGet,
		Path:        "/discord/status",
		Summary:     "Get Discord module status",
		Description: "Check Discord module health and operational status",
		Tags:        []string{"Module Status"},
	}, r.getDiscordStatus)
}

// Authentication handlers

func (r *Routes) getDiscordAuthURL(ctx context.Context, input *dto.GetDiscordAuthURLInput) (*dto.DiscordAuthURLOutput, error) {
	// Get current user ID if available (for linking)
	var userID *string

	// If this is for account linking, require authentication
	if input.LinkToUser {
		// For account linking, we need to know which user to link to
		// This requires authentication
		if input.Authorization == "" && input.Cookie == "" {
			return nil, fmt.Errorf("authentication required for account linking")
		}

		user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}
		userID = &user.UserID
	}

	return r.service.GetAuthURL(ctx, input, userID)
}

func (r *Routes) discordCallback(ctx context.Context, input *dto.DiscordCallbackInput) (*dto.DiscordCallbackOutput, error) {
	// Get current user ID if available (optional authentication)
	var userID *string

	// Try to extract authentication if headers are present
	if input.Authorization != "" || input.Cookie != "" {
		if user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie); err == nil {
			userID = &user.UserID
		}
		// Note: We don't return the error here as authentication is optional for the callback
	}

	_, linkedUserID, err := r.service.HandleCallback(ctx, input, userID)

	// Get frontend URL from configuration
	frontendURL := config.GetFrontendURL()

	if err != nil {
		// Redirect to frontend with error
		errorMessage := url.QueryEscape("Discord linking failed. Please try again.")
		errorURL := fmt.Sprintf("%s/discord/error?message=%s", frontendURL, errorMessage)

		return &dto.DiscordCallbackOutput{
			Status:   302,
			Location: errorURL,
			Body:     nil,
		}, nil
	}

	// Redirect to frontend with success
	var successURL string
	if userID != nil {
		// Account linking successful
		successURL = fmt.Sprintf("%s/discord/success?action=linked", frontendURL)
	} else if linkedUserID != "" {
		// New user authentication (future implementation)
		successURL = fmt.Sprintf("%s/discord/success?action=authenticated&user_id=%s", frontendURL, linkedUserID)
	} else {
		// Fallback success
		successURL = fmt.Sprintf("%s/discord/success", frontendURL)
	}

	return &dto.DiscordCallbackOutput{
		Status:   302,
		Location: successURL,
		Body:     nil,
	}, nil
}

func (r *Routes) linkDiscordAccount(ctx context.Context, input *dto.LinkDiscordAccountInput) (*dto.DiscordMessageOutput, error) {
	user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.LinkAccount(ctx, input, user.UserID)
}

func (r *Routes) unlinkDiscordAccount(ctx context.Context, input *dto.UnlinkDiscordAccountInput) (*dto.DiscordMessageOutput, error) {
	user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.UnlinkAccount(ctx, input, user.UserID)
}

func (r *Routes) getDiscordAuthStatus(ctx context.Context, input *dto.DiscordStatusInput) (*dto.DiscordAuthStatusOutput, error) {
	// Get current user ID if available (optional auth)
	var userID *string

	// Try to authenticate user for enhanced response (but don't require it)
	if input.Authorization != "" || input.Cookie != "" {
		if user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie); err == nil {
			userID = &user.UserID
		}
	}

	return r.service.GetAuthStatus(ctx, userID)
}

// User management handlers

func (r *Routes) getDiscordUser(ctx context.Context, input *dto.GetDiscordUserInput) (*dto.DiscordUserOutput, error) {
	// Require authentication for user management
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.GetDiscordUser(ctx, input)
}

func (r *Routes) listDiscordUsers(ctx context.Context, input *dto.ListDiscordUsersInput) (*dto.ListDiscordUsersOutput, error) {
	// Require authentication for user management
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.ListDiscordUsers(ctx, input)
}

// Guild management handlers

func (r *Routes) createGuildConfig(ctx context.Context, input *dto.CreateGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	// Use character ID from authenticated user
	characterID := int64(user.CharacterID)

	return r.service.CreateGuildConfig(ctx, input, characterID)
}

func (r *Routes) getGuildConfig(ctx context.Context, input *dto.GetGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.GetGuildConfig(ctx, input)
}

func (r *Routes) updateGuildConfig(ctx context.Context, input *dto.UpdateGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.UpdateGuildConfig(ctx, input)
}

func (r *Routes) deleteGuildConfig(ctx context.Context, input *dto.DeleteGuildConfigInput) (*dto.DiscordSuccessOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.DeleteGuildConfig(ctx, input)
}

func (r *Routes) listGuildConfigs(ctx context.Context, input *dto.ListGuildConfigsInput) (*dto.ListDiscordGuildConfigsOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.ListGuildConfigs(ctx, input)
}

// Synchronization handlers

func (r *Routes) triggerManualSync(ctx context.Context, input *dto.ManualSyncInput) (*dto.ManualSyncOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.TriggerManualSync(ctx, input)
}

func (r *Routes) syncUser(ctx context.Context, input *dto.SyncUserInput) (*dto.ManualSyncOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.SyncUser(ctx, input)
}

func (r *Routes) getSyncStatus(ctx context.Context, input *dto.GetSyncStatusInput) (*dto.DiscordSyncStatusOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.GetSyncStatus(ctx, input)
}

// Role mapping handlers

func (r *Routes) createRoleMapping(ctx context.Context, input *dto.CreateRoleMappingInput) (*dto.DiscordRoleMappingOutput, error) {
	user, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	// Use character ID from authenticated user
	characterID := int64(user.CharacterID)

	return r.service.CreateRoleMapping(ctx, input, characterID)
}

func (r *Routes) getRoleMapping(ctx context.Context, input *dto.GetRoleMappingInput) (*dto.DiscordRoleMappingOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.GetRoleMapping(ctx, input)
}

func (r *Routes) updateRoleMapping(ctx context.Context, input *dto.UpdateRoleMappingInput) (*dto.DiscordRoleMappingOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.UpdateRoleMapping(ctx, input)
}

func (r *Routes) deleteRoleMapping(ctx context.Context, input *dto.DeleteRoleMappingInput) (*dto.DiscordSuccessOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.DeleteRoleMapping(ctx, input)
}

func (r *Routes) listRoleMappings(ctx context.Context, input *dto.ListRoleMappingsInput) (*dto.ListDiscordRoleMappingsOutput, error) {
	_, err := r.discordAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, err
	}

	return r.service.ListRoleMappings(ctx, input)
}

// Status handler

func (r *Routes) getDiscordStatus(ctx context.Context, input *dto.DiscordStatusInput) (*dto.DiscordStatusOutput, error) {
	status := r.service.GetStatus(ctx)
	return &dto.DiscordStatusOutput{
		Body: *status,
	}, nil
}

// RegisterRoutes registers Discord routes using Chi router (legacy support)
func (r *Routes) RegisterRoutes(router chi.Router) {
	// Add basic health check
	router.Get("/discord/status", func(w http.ResponseWriter, req *http.Request) {
		status := r.service.GetStatus(req.Context())
		handlers.JSONResponse(w, status, http.StatusOK)
	})

	// OAuth redirect endpoint (for frontend integration)
	router.Get("/discord/auth/redirect", func(w http.ResponseWriter, req *http.Request) {
		// Redirect to frontend after successful OAuth
		frontendURL := config.GetEnv("FRONTEND_URL", "https://go.eveonline.it")
		if frontendURL == "" {
			frontendURL = "http://localhost:3000"
		}

		http.Redirect(w, req, frontendURL+"/discord/success", http.StatusTemporaryRedirect)
	})
}
