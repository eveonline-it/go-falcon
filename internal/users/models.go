package users

import (
	"time"
)

// User represents a user with character information and state control
// Uses the same database collection as auth module: user_profiles
type User struct {
	CharacterID  int       `json:"character_id" bson:"character_id"`   // EVE character ID (unique)
	UserID       string    `json:"user_id" bson:"user_id"`             // UUID for internal identification
	AccessToken  string    `json:"-" bson:"access_token"`              // EVE SSO access token (hidden from JSON)
	RefreshToken string    `json:"-" bson:"refresh_token"`             // EVE SSO refresh token (hidden from JSON)
	Enabled      bool      `json:"enabled" bson:"enabled"`             // User account status
	Banned       bool      `json:"banned" bson:"banned"`               // Ban status
	Invalid      bool      `json:"invalid" bson:"invalid"`             // Token/account validity
	Scopes       string    `json:"scopes" bson:"scopes"`               // EVE Online permissions
	Position     int       `json:"position" bson:"position"`           // User position/rank
	Notes        string    `json:"notes" bson:"notes"`                 // Administrative notes
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`       // Registration timestamp
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`       // Last update timestamp
	LastLogin    time.Time `json:"last_login" bson:"last_login"`       // Last login timestamp

	// Character information (from EVE SSO)
	CharacterName string `json:"character_name" bson:"character_name"` // EVE character name
	
	// Legacy field mapping (for compatibility with auth module)
	Valid bool `json:"valid" bson:"valid"` // Character profile validity status
}

// UserListResponse represents paginated user list response
type UserListResponse struct {
	Users      []User `json:"users"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

// UserSearchRequest represents user search and filter parameters
type UserSearchRequest struct {
	Query      string `json:"query" form:"query"`           // Search by character name or ID
	Enabled    *bool  `json:"enabled" form:"enabled"`       // Filter by enabled status
	Banned     *bool  `json:"banned" form:"banned"`         // Filter by banned status
	Invalid    *bool  `json:"invalid" form:"invalid"`       // Filter by validity status
	Position   *int   `json:"position" form:"position"`     // Filter by position
	Page       int    `json:"page" form:"page"`             // Page number (1-based)
	PageSize   int    `json:"page_size" form:"page_size"`   // Items per page (max 100)
	SortBy     string `json:"sort_by" form:"sort_by"`       // Sort field: name, created_at, last_login, position
	SortOrder  string `json:"sort_order" form:"sort_order"` // Sort order: asc, desc
}

// UserUpdateRequest represents user status update request
type UserUpdateRequest struct {
	Enabled  *bool   `json:"enabled,omitempty"`  // Enable/disable user
	Banned   *bool   `json:"banned,omitempty"`   // Ban/unban user  
	Invalid  *bool   `json:"invalid,omitempty"`  // Set validity status
	Position *int    `json:"position,omitempty"` // Update position/rank
	Notes    *string `json:"notes,omitempty"`    // Update administrative notes
}

// CharacterSummary represents basic character information for listing
type CharacterSummary struct {
	CharacterID   int    `json:"character_id"`
	CharacterName string `json:"character_name"`
	UserID        string `json:"user_id"`
	Enabled       bool   `json:"enabled"`
	Banned        bool   `json:"banned"`
	Position      int    `json:"position"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// UserStatsResponse represents user statistics
type UserStatsResponse struct {
	TotalUsers    int `json:"total_users"`
	EnabledUsers  int `json:"enabled_users"`
	DisabledUsers int `json:"disabled_users"`
	BannedUsers   int `json:"banned_users"`
	InvalidUsers  int `json:"invalid_users"`
}