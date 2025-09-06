package dto

// Authentication Flow Inputs

// GetDiscordAuthURLInput represents the request to get Discord OAuth URL
type GetDiscordAuthURLInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	LinkToUser    bool   `query:"link" example:"false" doc:"Whether to link to existing user or create new session"`
}

// DiscordCallbackInput represents the Discord OAuth callback
type DiscordCallbackInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication (optional)"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie (optional)"`
	Code          string `query:"code" required:"true" doc:"OAuth authorization code from Discord"`
	State         string `query:"state" required:"true" doc:"OAuth state parameter for CSRF protection"`
}

// LinkDiscordAccountInput represents a request to link Discord to existing user
type LinkDiscordAccountInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	Body          struct {
		AccessToken  string `json:"access_token" required:"true" example:"Discord_OAuth_Access_Token" doc:"Discord OAuth access token"`
		RefreshToken string `json:"refresh_token" example:"Discord_Refresh_Token" doc:"Discord OAuth refresh token"`
	}
}

// UnlinkDiscordAccountInput represents a request to unlink Discord account
type UnlinkDiscordAccountInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	DiscordID     string `path:"discord_id" required:"true" example:"123456789012345678" doc:"Discord user ID to unlink"`
}

// Guild Management Inputs

// CreateGuildConfigInput represents a request to add a Discord guild
type CreateGuildConfigInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	Body          struct {
		GuildID   string `json:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID"`
		GuildName string `json:"guild_name" required:"true" example:"My Discord Server" doc:"Display name for the guild"`
		BotToken  string `json:"bot_token" required:"true" example:"Bot YOUR_DISCORD_BOT_TOKEN_HERE" doc:"Discord bot token"`
	}
}

// UpdateGuildConfigInput represents a request to update guild configuration
type UpdateGuildConfigInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `path:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID"`
	Body          struct {
		GuildName *string `json:"guild_name" example:"Updated Server Name" doc:"Updated display name for the guild"`
		BotToken  *string `json:"bot_token" example:"Bot YOUR_NEW_BOT_TOKEN_HERE" doc:"Updated Discord bot token"`
		IsEnabled *bool   `json:"is_enabled" example:"true" doc:"Whether role sync is enabled for this guild"`
	}
}

// DeleteGuildConfigInput represents a request to remove a guild
type DeleteGuildConfigInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `path:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID to remove"`
}

// GetGuildConfigInput represents a request to get guild configuration
type GetGuildConfigInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `path:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID"`
}

// GetGuildRolesInput represents a request to get Discord guild roles
type GetGuildRolesInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `path:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID"`
}

// ListGuildConfigsInput represents a request to list guild configurations
type ListGuildConfigsInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	IsEnabled     string `query:"is_enabled" enum:"true,false" doc:"Filter by enabled status"`
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number for pagination"`
	Limit         int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Number of items per page"`
}

// Role Mapping Inputs

// CreateRoleMappingInput represents a request to create a role mapping
type CreateRoleMappingInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `path:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID"`
	Body          struct {
		GroupID         string `json:"group_id" required:"true" example:"507f1f77bcf86cd799439011" doc:"Go Falcon group ID (MongoDB ObjectID)"`
		DiscordRoleID   string `json:"discord_role_id" required:"true" example:"987654321098765432" doc:"Discord role ID"`
		DiscordRoleName string `json:"discord_role_name" required:"true" example:"Fleet Commander" doc:"Discord role name for display"`
	}
}

// UpdateRoleMappingInput represents a request to update a role mapping
type UpdateRoleMappingInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	MappingID     string `path:"mapping_id" required:"true" example:"507f1f77bcf86cd799439012" doc:"Role mapping ID"`
	Body          struct {
		DiscordRoleID   *string `json:"discord_role_id" example:"987654321098765433" doc:"Updated Discord role ID"`
		DiscordRoleName *string `json:"discord_role_name" example:"Updated Role Name" doc:"Updated Discord role name"`
		IsActive        *bool   `json:"is_active" example:"true" doc:"Whether the mapping is active"`
	}
}

// DeleteRoleMappingInput represents a request to delete a role mapping
type DeleteRoleMappingInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	MappingID     string `path:"mapping_id" required:"true" example:"507f1f77bcf86cd799439012" doc:"Role mapping ID to delete"`
}

// GetRoleMappingInput represents a request to get a role mapping
type GetRoleMappingInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	MappingID     string `path:"mapping_id" required:"true" example:"507f1f77bcf86cd799439012" doc:"Role mapping ID"`
}

// ListRoleMappingsInput represents a request to list role mappings
type ListRoleMappingsInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `path:"guild_id" required:"true" example:"123456789012345678" doc:"Discord guild ID"`
	IsActive      string `query:"is_active" enum:"true,false" doc:"Filter by active status"`
	GroupID       string `query:"group_id" example:"507f1f77bcf86cd799439011" doc:"Filter by Go Falcon group ID"`
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number for pagination"`
	Limit         int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Number of items per page"`
}

// Synchronization Inputs

// ManualSyncInput represents a request to trigger manual synchronization
type ManualSyncInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	Body          struct {
		GuildID string   `json:"guild_id" example:"123456789012345678" doc:"Optional guild ID to sync (empty for all guilds)"`
		UserIDs []string `json:"user_ids" example:"[\"user-uuid-1\", \"user-uuid-2\"]" doc:"Optional specific user IDs to sync"`
		DryRun  bool     `json:"dry_run" example:"false" doc:"Whether to perform a dry run (no actual role changes)"`
	}
}

// SyncUserInput represents a request to sync a specific user
type SyncUserInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	UserID        string `path:"user_id" required:"true" example:"550e8400-e29b-41d4-a716-446655440000" doc:"Go Falcon user UUID"`
	Body          struct {
		GuildID string `json:"guild_id" example:"123456789012345678" doc:"Optional guild ID to sync (empty for all guilds)"`
		DryRun  bool   `json:"dry_run" example:"false" doc:"Whether to perform a dry run"`
	}
}

// GetSyncStatusInput represents a request to get sync status
type GetSyncStatusInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	GuildID       string `query:"guild_id" example:"123456789012345678" doc:"Optional guild ID to filter status"`
	Limit         int    `query:"limit" minimum:"1" maximum:"50" default:"10" doc:"Number of recent sync records to return"`
}

// User Management Inputs

// GetDiscordUserInput represents a request to get Discord user info
type GetDiscordUserInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	UserID        string `path:"user_id" required:"true" example:"550e8400-e29b-41d4-a716-446655440000" doc:"Go Falcon user UUID"`
}

// ListDiscordUsersInput represents a request to list Discord users
type ListDiscordUsersInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie"`
	IsActive      string `query:"is_active" enum:"true,false" doc:"Filter by active status"`
	DiscordID     string `query:"discord_id" example:"123456789012345678" doc:"Filter by Discord user ID"`
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number for pagination"`
	Limit         int    `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Number of items per page"`
}

// Utility Inputs

// DiscordStatusInput represents a request for Discord module status
type DiscordStatusInput struct {
	Authorization string `header:"Authorization" example:"Bearer your-jwt-token" doc:"Bearer token for authentication (optional for enhanced status)"`
	Cookie        string `header:"Cookie" example:"falcon_auth_token=your-token" doc:"Authentication cookie (optional for enhanced status)"`
}
