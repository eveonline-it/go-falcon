package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DiscordUser represents a Discord user linked to a Go Falcon user
type DiscordUser struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	UserID       string             `bson:"user_id"`       // Go Falcon user UUID
	DiscordID    string             `bson:"discord_id"`    // Discord user ID
	Username     string             `bson:"username"`      // Discord username
	GlobalName   *string            `bson:"global_name"`   // Discord global name
	Avatar       *string            `bson:"avatar"`        // Avatar hash
	AccessToken  string             `bson:"access_token"`  // Discord OAuth token (encrypted)
	RefreshToken string             `bson:"refresh_token"` // Discord refresh token (encrypted)
	TokenExpiry  time.Time          `bson:"token_expiry"`
	IsActive     bool               `bson:"is_active"`
	LinkedAt     time.Time          `bson:"linked_at"`
	UpdatedAt    time.Time          `bson:"updated_at"`
}

// DiscordGuildConfig represents configuration for a Discord guild
type DiscordGuildConfig struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty"`
	GuildID      string               `bson:"guild_id"`      // Discord guild ID
	GuildName    string               `bson:"guild_name"`    // Guild name for display
	BotToken     string               `bson:"bot_token"`     // Encrypted bot token
	IsEnabled    bool                 `bson:"is_enabled"`    // Whether sync is enabled
	RoleMappings []DiscordRoleMapping `bson:"role_mappings"` // Embedded role mappings
	CreatedBy    *int64               `bson:"created_by"`    // Character ID who created
	CreatedAt    time.Time            `bson:"created_at"`
	UpdatedAt    time.Time            `bson:"updated_at"`
}

// DiscordRoleMapping represents a mapping between Go Falcon groups and Discord roles
type DiscordRoleMapping struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	GuildID         string             `bson:"guild_id"`          // Discord guild ID
	GroupID         primitive.ObjectID `bson:"group_id"`          // Go Falcon group ID
	GroupName       string             `bson:"group_name"`        // Cached group name
	DiscordRoleID   string             `bson:"discord_role_id"`   // Discord role ID
	DiscordRoleName string             `bson:"discord_role_name"` // Discord role name
	IsActive        bool               `bson:"is_active"`         // Whether mapping is active
	CreatedBy       *int64             `bson:"created_by"`        // Character ID who created
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
}

// DiscordSyncStatus represents the status of the last sync operation
type DiscordSyncStatus struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	GuildID        string             `bson:"guild_id"`
	LastSyncAt     time.Time          `bson:"last_sync_at"`
	UsersProcessed int64              `bson:"users_processed"`
	UsersSucceeded int64              `bson:"users_succeeded"`
	UsersFailed    int64              `bson:"users_failed"`
	RolesAdded     int64              `bson:"roles_added"`
	RolesRemoved   int64              `bson:"roles_removed"`
	Errors         []string           `bson:"errors,omitempty"`
	Status         string             `bson:"status"`   // "running", "completed", "failed"
	Duration       int64              `bson:"duration"` // Duration in milliseconds
	CreatedAt      time.Time          `bson:"created_at"`
}

// DiscordOAuthState represents temporary OAuth state for CSRF protection
type DiscordOAuthState struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	State     string             `bson:"state"`      // Random state parameter
	UserID    *string            `bson:"user_id"`    // Optional user ID for linking
	ExpiresAt time.Time          `bson:"expires_at"` // State expiration
	CreatedAt time.Time          `bson:"created_at"`
}

// MongoDB Collection Names
const (
	DiscordUsersCollection        = "discord_users"
	DiscordGuildConfigsCollection = "discord_guild_configs"
	DiscordRoleMappingsCollection = "discord_role_mappings"
	DiscordSyncStatusCollection   = "discord_sync_status"
	DiscordOAuthStatesCollection  = "discord_oauth_states"
)
