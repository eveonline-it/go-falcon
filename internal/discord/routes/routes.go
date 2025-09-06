package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"go-falcon/internal/discord/dto"
	"go-falcon/internal/discord/services"
	"go-falcon/pkg/config"
	"go-falcon/pkg/handlers"
)

// Module represents the Discord routes module
type Module struct {
	service    *services.Service
	middleware MiddlewareInterface
}

// MiddlewareInterface defines the middleware interface for authentication
type MiddlewareInterface interface {
	RequireAuth() func(http.Handler) http.Handler
	OptionalAuth() func(http.Handler) http.Handler
	GetAuthenticatedUser(r *http.Request) (map[string]interface{}, bool)
}

// NewModule creates a new Discord routes module
func NewModule(service *services.Service, middleware MiddlewareInterface) *Module {
	return &Module{
		service:    service,
		middleware: middleware,
	}
}

// RegisterUnifiedRoutes registers Discord routes on the unified API
func (m *Module) RegisterUnifiedRoutes(api huma.API) {
	// Authentication routes
	huma.Register(api, huma.Operation{
		OperationID: "getDiscordAuthURL",
		Method:      http.MethodGet,
		Path:        "/discord/auth/login",
		Summary:     "Get Discord OAuth authorization URL",
		Description: "Generate Discord OAuth authorization URL for user authentication or account linking",
		Tags:        []string{"Discord Authentication"},
	}, m.getDiscordAuthURL)

	huma.Register(api, huma.Operation{
		OperationID: "discordCallback",
		Method:      http.MethodGet,
		Path:        "/discord/auth/callback",
		Summary:     "Handle Discord OAuth callback",
		Description: "Process Discord OAuth callback and complete authentication or account linking",
		Tags:        []string{"Discord Authentication"},
	}, m.discordCallback)

	huma.Register(api, huma.Operation{
		OperationID: "linkDiscordAccount",
		Method:      http.MethodPost,
		Path:        "/discord/auth/link",
		Summary:     "Link Discord account to existing user",
		Description: "Link a Discord account to an existing Go Falcon user using OAuth tokens",
		Tags:        []string{"Discord Authentication"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.linkDiscordAccount)

	huma.Register(api, huma.Operation{
		OperationID: "unlinkDiscordAccount",
		Method:      http.MethodDelete,
		Path:        "/discord/auth/unlink/{discord_id}",
		Summary:     "Unlink Discord account",
		Description: "Unlink a Discord account from the current Go Falcon user",
		Tags:        []string{"Discord Authentication"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.unlinkDiscordAccount)

	huma.Register(api, huma.Operation{
		OperationID: "getDiscordAuthStatus",
		Method:      http.MethodGet,
		Path:        "/discord/auth/status",
		Summary:     "Get Discord authentication status",
		Description: "Check if user has Discord accounts linked and get status information",
		Tags:        []string{"Discord Authentication"},
	}, m.getDiscordAuthStatus)

	// User management routes
	huma.Register(api, huma.Operation{
		OperationID: "getDiscordUser",
		Method:      http.MethodGet,
		Path:        "/discord/users/{user_id}",
		Summary:     "Get Discord user information",
		Description: "Get Discord account information for a specific Go Falcon user",
		Tags:        []string{"Discord Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.getDiscordUser)

	huma.Register(api, huma.Operation{
		OperationID: "listDiscordUsers",
		Method:      http.MethodGet,
		Path:        "/discord/users",
		Summary:     "List Discord users",
		Description: "List all Discord users with filtering and pagination",
		Tags:        []string{"Discord Users"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.listDiscordUsers)

	// Guild management routes
	huma.Register(api, huma.Operation{
		OperationID: "createGuildConfig",
		Method:      http.MethodPost,
		Path:        "/discord/guilds",
		Summary:     "Create Discord guild configuration",
		Description: "Add a new Discord guild configuration with bot token for role management",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.createGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "getGuildConfig",
		Method:      http.MethodGet,
		Path:        "/discord/guilds/{guild_id}",
		Summary:     "Get Discord guild configuration",
		Description: "Get configuration details for a specific Discord guild",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.getGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "updateGuildConfig",
		Method:      http.MethodPut,
		Path:        "/discord/guilds/{guild_id}",
		Summary:     "Update Discord guild configuration",
		Description: "Update configuration for an existing Discord guild",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.updateGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "deleteGuildConfig",
		Method:      http.MethodDelete,
		Path:        "/discord/guilds/{guild_id}",
		Summary:     "Delete Discord guild configuration",
		Description: "Remove a Discord guild configuration and all associated role mappings",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.deleteGuildConfig)

	huma.Register(api, huma.Operation{
		OperationID: "listGuildConfigs",
		Method:      http.MethodGet,
		Path:        "/discord/guilds",
		Summary:     "List Discord guild configurations",
		Description: "List all Discord guild configurations with filtering and pagination",
		Tags:        []string{"Discord Guilds"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.listGuildConfigs)

	// Synchronization routes
	huma.Register(api, huma.Operation{
		OperationID: "triggerManualSync",
		Method:      http.MethodPost,
		Path:        "/discord/sync/manual",
		Summary:     "Trigger manual role synchronization",
		Description: "Manually trigger Discord role synchronization for all guilds or specific targets",
		Tags:        []string{"Discord Sync"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.triggerManualSync)

	huma.Register(api, huma.Operation{
		OperationID: "syncUser",
		Method:      http.MethodPost,
		Path:        "/discord/sync/user/{user_id}",
		Summary:     "Synchronize specific user roles",
		Description: "Synchronize Discord roles for a specific Go Falcon user",
		Tags:        []string{"Discord Sync"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.syncUser)

	huma.Register(api, huma.Operation{
		OperationID: "getSyncStatus",
		Method:      http.MethodGet,
		Path:        "/discord/sync/status",
		Summary:     "Get synchronization status",
		Description: "Get current and recent Discord role synchronization status",
		Tags:        []string{"Discord Sync"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, m.getSyncStatus)

	// Module status route
	huma.Register(api, huma.Operation{
		OperationID: "getDiscordStatus",
		Method:      http.MethodGet,
		Path:        "/discord/status",
		Summary:     "Get Discord module status",
		Description: "Check Discord module health and operational status",
		Tags:        []string{"Module Status"},
	}, m.getDiscordStatus)
}

// Authentication handlers

func (m *Module) getDiscordAuthURL(ctx context.Context, input *dto.GetDiscordAuthURLInput) (*dto.DiscordAuthURLOutput, error) {
	// Get current user ID if available (for linking)
	var userID *string
	if m.middleware != nil {
		if r, ok := ctx.Value("request").(*http.Request); ok {
			if user, authenticated := m.middleware.GetAuthenticatedUser(r); authenticated {
				if uid, ok := user["user_id"].(string); ok {
					userID = &uid
				}
			}
		}
	}

	return m.service.GetAuthURL(ctx, input, userID)
}

func (m *Module) discordCallback(ctx context.Context, input *dto.DiscordCallbackInput) (*dto.DiscordMessageOutput, error) {
	// Get current user ID if available
	var userID *string
	if m.middleware != nil {
		if r, ok := ctx.Value("request").(*http.Request); ok {
			if user, authenticated := m.middleware.GetAuthenticatedUser(r); authenticated {
				if uid, ok := user["user_id"].(string); ok {
					userID = &uid
				}
			}
		}
	}

	result, linkedUserID, err := m.service.HandleCallback(ctx, input, userID)
	if err != nil {
		return nil, err
	}

	// If this was a new user authentication (not linking), redirect to frontend
	if userID == nil && linkedUserID != "" {
		// In a real implementation, you might want to set a session cookie here
		// or redirect to a success page with the user ID
	}

	return result, nil
}

func (m *Module) linkDiscordAccount(ctx context.Context, input *dto.LinkDiscordAccountInput) (*dto.DiscordMessageOutput, error) {
	userID, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.LinkAccount(ctx, input, userID)
}

func (m *Module) unlinkDiscordAccount(ctx context.Context, input *dto.UnlinkDiscordAccountInput) (*dto.DiscordMessageOutput, error) {
	userID, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.UnlinkAccount(ctx, input, userID)
}

func (m *Module) getDiscordAuthStatus(ctx context.Context, input *dto.DiscordStatusInput) (*dto.DiscordAuthStatusOutput, error) {
	// Get current user ID if available (optional auth)
	var userID *string
	if m.middleware != nil {
		if r, ok := ctx.Value("request").(*http.Request); ok {
			if user, authenticated := m.middleware.GetAuthenticatedUser(r); authenticated {
				if uid, ok := user["user_id"].(string); ok {
					userID = &uid
				}
			}
		}
	}

	return m.service.GetAuthStatus(ctx, userID)
}

// User management handlers

func (m *Module) getDiscordUser(ctx context.Context, input *dto.GetDiscordUserInput) (*dto.DiscordUserOutput, error) {
	// Require authentication for user management
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.GetDiscordUser(ctx, input)
}

func (m *Module) listDiscordUsers(ctx context.Context, input *dto.ListDiscordUsersInput) (*dto.ListDiscordUsersOutput, error) {
	// Require authentication for user management
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.ListDiscordUsers(ctx, input)
}

// Guild management handlers

func (m *Module) createGuildConfig(ctx context.Context, input *dto.CreateGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	// Convert userID to character ID (simplified for now)
	characterID := int64(0) // TODO: Get character ID from user ID

	return m.service.CreateGuildConfig(ctx, input, characterID)
}

func (m *Module) getGuildConfig(ctx context.Context, input *dto.GetGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.GetGuildConfig(ctx, input)
}

func (m *Module) updateGuildConfig(ctx context.Context, input *dto.UpdateGuildConfigInput) (*dto.DiscordGuildConfigOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.UpdateGuildConfig(ctx, input)
}

func (m *Module) deleteGuildConfig(ctx context.Context, input *dto.DeleteGuildConfigInput) (*dto.DiscordSuccessOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.DeleteGuildConfig(ctx, input)
}

func (m *Module) listGuildConfigs(ctx context.Context, input *dto.ListGuildConfigsInput) (*dto.ListDiscordGuildConfigsOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.ListGuildConfigs(ctx, input)
}

// Synchronization handlers

func (m *Module) triggerManualSync(ctx context.Context, input *dto.ManualSyncInput) (*dto.ManualSyncOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.TriggerManualSync(ctx, input)
}

func (m *Module) syncUser(ctx context.Context, input *dto.SyncUserInput) (*dto.ManualSyncOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.SyncUser(ctx, input)
}

func (m *Module) getSyncStatus(ctx context.Context, input *dto.GetSyncStatusInput) (*dto.DiscordSyncStatusOutput, error) {
	_, err := m.getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Authentication required")
	}

	return m.service.GetSyncStatus(ctx, input)
}

// Status handler

func (m *Module) getDiscordStatus(ctx context.Context, input *dto.DiscordStatusInput) (*dto.DiscordStatusOutput, error) {
	status := m.service.GetStatus(ctx)
	return &dto.DiscordStatusOutput{
		Body: *status,
	}, nil
}

// Helper methods

// getAuthenticatedUserID extracts the user ID from the authenticated request
func (m *Module) getAuthenticatedUserID(ctx context.Context) (string, error) {
	if m.middleware == nil {
		return "", fmt.Errorf("middleware not available")
	}

	r, ok := ctx.Value("request").(*http.Request)
	if !ok {
		return "", fmt.Errorf("request not available in context")
	}

	user, authenticated := m.middleware.GetAuthenticatedUser(r)
	if !authenticated {
		return "", fmt.Errorf("user not authenticated")
	}

	userID, ok := user["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user ID not found in authentication context")
	}

	return userID, nil
}

// RegisterRoutes registers Discord routes using Chi router (legacy support)
func (m *Module) RegisterRoutes(r chi.Router) {
	// Add basic health check
	r.Get("/discord/status", func(w http.ResponseWriter, r *http.Request) {
		status := m.service.GetStatus(r.Context())
		handlers.JSONResponse(w, status, http.StatusOK)
	})

	// OAuth redirect endpoint (for frontend integration)
	r.Get("/discord/auth/redirect", func(w http.ResponseWriter, r *http.Request) {
		// Redirect to frontend after successful OAuth
		frontendURL := config.GetEnv("FRONTEND_URL", "https://go.eveonline.it")
		if frontendURL == "" {
			frontendURL = "http://localhost:3000"
		}

		http.Redirect(w, r, frontendURL+"/discord/success", http.StatusTemporaryRedirect)
	})
}
