package dto

import (
	"time"
)

// UserResponse represents a user in API responses
type UserResponse struct {
	CharacterID   int       `json:"character_id"`
	UserID        string    `json:"user_id"`
	Enabled       bool      `json:"enabled"`
	Banned        bool      `json:"banned"`
	Invalid       bool      `json:"invalid"`
	Scopes        string    `json:"scopes"`
	Position      int       `json:"position"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastLogin     time.Time `json:"last_login"`
	CharacterName string    `json:"character_name"`
	Valid         bool      `json:"valid"`
}


// CharacterSummaryResponse represents basic character information for listing
type CharacterSummaryResponse struct {
	CharacterID   int        `json:"character_id"`
	CharacterName string     `json:"character_name"`
	UserID        string     `json:"user_id"`
	Enabled       bool       `json:"enabled"`
	Banned        bool       `json:"banned"`
	Position      int        `json:"position"`
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

// UserUpdateResponse represents the response after updating a user
type UserUpdateResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	User    UserResponse `json:"user"`
}

// CharacterListResponse represents the response for character listing
type CharacterListResponse struct {
	UserID     string                     `json:"user_id"`
	Characters []CharacterSummaryResponse `json:"characters"`
	Count      int                        `json:"count"`
}

// UserListResponse represents paginated user listing response
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// =============================================================================
// HUMA OUTPUT DTOs (consolidated from huma_requests.go)
// =============================================================================

// UserStatsOutput represents the output for getting user statistics
type UserStatsOutput struct {
	Body UserStatsResponse `json:"body"`
}

// UserGetOutput represents the output for getting a specific user
type UserGetOutput struct {
	Body UserResponse `json:"body"`
}

// UserUpdateOutput represents the output for updating a user
type UserUpdateOutput struct {
	Body UserResponse `json:"body"`
}

// UserCharactersOutput represents the output for getting user characters
type UserCharactersOutput struct {
	Body CharacterListResponse `json:"body"`
}

// UserListOutput represents the output for listing users with pagination
type UserListOutput struct {
	Body UserListResponse `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body UsersStatusResponse `json:"body"`
}

// UserDeleteOutput represents the output for deleting a user character
type UserDeleteOutput struct {
	Body UserDeleteResponse `json:"body"`
}

// UserDeleteResponse represents the response after deleting a user character
type UserDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UsersStatusResponse represents the actual status response data
type UsersStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}