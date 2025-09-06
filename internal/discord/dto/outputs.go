package dto

import (
	"time"
)

// Standard Response Wrappers

// DiscordSuccessOutput represents a successful operation response
type DiscordSuccessOutput struct {
	Body DiscordSuccessResponse
}

// DiscordSuccessResponse represents the body of a success response
type DiscordSuccessResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// DiscordMessageOutput represents a message response
type DiscordMessageOutput struct {
	Body DiscordMessageResponse
}

// DiscordMessageResponse represents the body of a message response
type DiscordMessageResponse struct {
	Message string `json:"message" example:"Discord account linked successfully"`
}

// Authentication Flow Outputs

// DiscordAuthURLOutput represents the Discord OAuth URL response
type DiscordAuthURLOutput struct {
	Body DiscordAuthURLResponse
}

// DiscordAuthURLResponse contains the Discord OAuth URL and state
type DiscordAuthURLResponse struct {
	AuthURL string `json:"auth_url" example:"https://discord.com/api/oauth2/authorize?client_id=..." doc:"Discord OAuth authorization URL"`
	State   string `json:"state" example:"random_state_string" doc:"OAuth state parameter for CSRF protection"`
}

// DiscordCallbackOutput represents the output for Discord OAuth callback with redirect
type DiscordCallbackOutput struct {
	Status   int                    `json:"-" status:"302" doc:"HTTP status code for redirect"`
	Location string                 `header:"Location" doc:"Redirect location"`
	Body     map[string]interface{} `json:"body,omitempty"`
}

// DiscordAuthStatusOutput represents the Discord authentication status
type DiscordAuthStatusOutput struct {
	Body DiscordAuthStatusResponse
}

// DiscordAuthStatusResponse contains Discord authentication status
type DiscordAuthStatusResponse struct {
	IsLinked      bool                  `json:"is_linked" example:"true" doc:"Whether user has Discord account linked"`
	DiscordUsers  []DiscordUserResponse `json:"discord_users,omitempty" doc:"Linked Discord accounts"`
	Authenticated bool                  `json:"authenticated" example:"true" doc:"Whether user is authenticated"`
	UserID        string                `json:"user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000" doc:"Go Falcon user UUID"`
}

// DiscordUserOutput represents a single Discord user response
type DiscordUserOutput struct {
	Body DiscordUserResponse
}

// DiscordUserResponse represents Discord user information
type DiscordUserResponse struct {
	ID          string    `json:"id" example:"507f1f77bcf86cd799439011" doc:"Database record ID"`
	UserID      string    `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000" doc:"Go Falcon user UUID"`
	DiscordID   string    `json:"discord_id" example:"123456789012345678" doc:"Discord user ID"`
	Username    string    `json:"username" example:"discord_user" doc:"Discord username"`
	GlobalName  *string   `json:"global_name,omitempty" example:"Display Name" doc:"Discord global display name"`
	Avatar      *string   `json:"avatar,omitempty" example:"a_1234567890abcdef" doc:"Discord avatar hash"`
	IsActive    bool      `json:"is_active" example:"true" doc:"Whether the account link is active"`
	LinkedAt    time.Time `json:"linked_at" example:"2025-01-10T12:00:00Z" doc:"When the account was linked"`
	UpdatedAt   time.Time `json:"updated_at" example:"2025-01-10T12:00:00Z" doc:"Last update timestamp"`
	TokenExpiry time.Time `json:"token_expiry" example:"2025-01-17T12:00:00Z" doc:"When the OAuth token expires"`
}

// ListDiscordUsersOutput represents a list of Discord users response
type ListDiscordUsersOutput struct {
	Body ListDiscordUsersResponse
}

// ListDiscordUsersResponse contains paginated Discord users
type ListDiscordUsersResponse struct {
	Users []DiscordUserResponse `json:"users" doc:"List of Discord users"`
	Total int64                 `json:"total" example:"25" doc:"Total number of Discord users"`
	Page  int                   `json:"page" example:"1" doc:"Current page number"`
	Limit int                   `json:"limit" example:"20" doc:"Number of items per page"`
}

// Guild Management Outputs

// DiscordGuildConfigOutput represents a single Discord guild configuration
type DiscordGuildConfigOutput struct {
	Body DiscordGuildConfigResponse
}

// DiscordGuildConfigResponse represents Discord guild configuration
type DiscordGuildConfigResponse struct {
	ID           string                       `json:"id" example:"507f1f77bcf86cd799439011" doc:"Database record ID"`
	GuildID      string                       `json:"guild_id" example:"123456789012345678" doc:"Discord guild ID"`
	GuildName    string                       `json:"guild_name" example:"My Discord Server" doc:"Guild display name"`
	IsEnabled    bool                         `json:"is_enabled" example:"true" doc:"Whether role sync is enabled"`
	RoleMappings []DiscordRoleMappingResponse `json:"role_mappings" doc:"Role mappings for this guild"`
	CreatedBy    *int64                       `json:"created_by,omitempty" example:"123456789" doc:"Character ID who created the configuration"`
	CreatedAt    time.Time                    `json:"created_at" example:"2025-01-10T12:00:00Z" doc:"Creation timestamp"`
	UpdatedAt    time.Time                    `json:"updated_at" example:"2025-01-10T12:00:00Z" doc:"Last update timestamp"`
}

// ListDiscordGuildConfigsOutput represents a list of guild configurations
type ListDiscordGuildConfigsOutput struct {
	Body ListDiscordGuildConfigsResponse
}

// ListDiscordGuildConfigsResponse contains paginated guild configurations
type ListDiscordGuildConfigsResponse struct {
	Guilds []DiscordGuildConfigResponse `json:"guilds" doc:"List of Discord guild configurations"`
	Total  int64                        `json:"total" example:"5" doc:"Total number of configured guilds"`
	Page   int                          `json:"page" example:"1" doc:"Current page number"`
	Limit  int                          `json:"limit" example:"20" doc:"Number of items per page"`
}

// Role Mapping Outputs

// DiscordRoleMappingOutput represents a single role mapping response
type DiscordRoleMappingOutput struct {
	Body DiscordRoleMappingResponse
}

// DiscordRoleMappingResponse represents a role mapping between Go Falcon groups and Discord roles
type DiscordRoleMappingResponse struct {
	ID              string    `json:"id" example:"507f1f77bcf86cd799439012" doc:"Database record ID"`
	GuildID         string    `json:"guild_id" example:"123456789012345678" doc:"Discord guild ID"`
	GroupID         string    `json:"group_id" example:"507f1f77bcf86cd799439011" doc:"Go Falcon group ID"`
	GroupName       string    `json:"group_name" example:"Fleet Commanders" doc:"Go Falcon group name"`
	DiscordRoleID   string    `json:"discord_role_id" example:"987654321098765432" doc:"Discord role ID"`
	DiscordRoleName string    `json:"discord_role_name" example:"FC Role" doc:"Discord role name"`
	IsActive        bool      `json:"is_active" example:"true" doc:"Whether the mapping is active"`
	CreatedBy       *int64    `json:"created_by,omitempty" example:"123456789" doc:"Character ID who created the mapping"`
	CreatedAt       time.Time `json:"created_at" example:"2025-01-10T12:00:00Z" doc:"Creation timestamp"`
	UpdatedAt       time.Time `json:"updated_at" example:"2025-01-10T12:00:00Z" doc:"Last update timestamp"`
}

// ListDiscordRoleMappingsOutput represents a list of role mappings
type ListDiscordRoleMappingsOutput struct {
	Body ListDiscordRoleMappingsResponse
}

// ListDiscordRoleMappingsResponse contains paginated role mappings
type ListDiscordRoleMappingsResponse struct {
	GuildID  string                       `json:"guild_id" example:"123456789012345678" doc:"Discord guild ID"`
	Mappings []DiscordRoleMappingResponse `json:"mappings" doc:"List of role mappings"`
	Total    int64                        `json:"total" example:"15" doc:"Total number of role mappings"`
	Page     int                          `json:"page" example:"1" doc:"Current page number"`
	Limit    int                          `json:"limit" example:"20" doc:"Number of items per page"`
}

// Synchronization Outputs

// DiscordSyncStatusOutput represents sync status response
type DiscordSyncStatusOutput struct {
	Body DiscordSyncStatusResponse
}

// DiscordSyncStatusResponse contains synchronization status information
type DiscordSyncStatusResponse struct {
	CurrentStatus string                      `json:"current_status" example:"idle" enum:"idle,running,completed,failed" doc:"Current sync status"`
	RecentSyncs   []DiscordSyncResultResponse `json:"recent_syncs" doc:"Recent synchronization results"`
	TotalSyncs    int64                       `json:"total_syncs" example:"42" doc:"Total number of syncs performed"`
}

// DiscordSyncResultResponse represents the result of a synchronization operation
type DiscordSyncResultResponse struct {
	ID             string    `json:"id" example:"507f1f77bcf86cd799439013" doc:"Sync record ID"`
	GuildID        string    `json:"guild_id" example:"123456789012345678" doc:"Discord guild ID"`
	Status         string    `json:"status" example:"completed" enum:"running,completed,failed" doc:"Sync operation status"`
	UsersProcessed int64     `json:"users_processed" example:"150" doc:"Number of users processed"`
	UsersSucceeded int64     `json:"users_succeeded" example:"145" doc:"Number of users successfully synced"`
	UsersFailed    int64     `json:"users_failed" example:"5" doc:"Number of users that failed to sync"`
	RolesAdded     int64     `json:"roles_added" example:"23" doc:"Number of roles added"`
	RolesRemoved   int64     `json:"roles_removed" example:"12" doc:"Number of roles removed"`
	Duration       int64     `json:"duration" example:"5432" doc:"Sync duration in milliseconds"`
	Errors         []string  `json:"errors,omitempty" doc:"List of errors encountered during sync"`
	LastSyncAt     time.Time `json:"last_sync_at" example:"2025-01-10T12:00:00Z" doc:"When the sync was performed"`
	CreatedAt      time.Time `json:"created_at" example:"2025-01-10T12:00:00Z" doc:"Sync record creation timestamp"`
}

// Manual sync response
type ManualSyncOutput struct {
	Body ManualSyncResponse
}

// ManualSyncResponse represents the response from a manual sync trigger
type ManualSyncResponse struct {
	Message           string `json:"message" example:"Manual sync initiated successfully" doc:"Status message"`
	SyncID            string `json:"sync_id,omitempty" example:"507f1f77bcf86cd799439013" doc:"Sync operation ID"`
	IsAsync           bool   `json:"is_async" example:"true" doc:"Whether the sync is running asynchronously"`
	EstimatedDuration string `json:"estimated_duration,omitempty" example:"2-5 minutes" doc:"Estimated completion time"`
}

// Module Status Output

// DiscordStatusOutput represents the Discord module status response
type DiscordStatusOutput struct {
	Body DiscordStatusResponse
}

// DiscordStatusResponse contains Discord module health and status information
type DiscordStatusResponse struct {
	Module            string     `json:"module" example:"discord" doc:"Module name"`
	Status            string     `json:"status" example:"healthy" enum:"healthy,unhealthy,degraded" doc:"Module health status"`
	Message           string     `json:"message,omitempty" example:"All systems operational" doc:"Status message"`
	ConfiguredGuilds  int64      `json:"configured_guilds" example:"3" doc:"Number of configured Discord guilds"`
	ActiveUsers       int64      `json:"active_users" example:"125" doc:"Number of active Discord users"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty" example:"2025-01-10T12:00:00Z" doc:"Last synchronization timestamp"`
	DatabaseConnected bool       `json:"database_connected" example:"true" doc:"Database connectivity status"`
	DiscordAPIHealthy bool       `json:"discord_api_healthy" example:"true" doc:"Discord API connectivity status"`
}

// Error Responses

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"invalid_request" doc:"Error code"`
	Message string `json:"message" example:"The request was invalid" doc:"Human-readable error message"`
	Details string `json:"details,omitempty" example:"Additional error context" doc:"Additional error details"`
}
