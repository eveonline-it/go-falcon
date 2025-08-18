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

// UserListResponse represents paginated user list response
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
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

// FullCharacterResponse represents complete character information for middleware resolution
type FullCharacterResponse struct {
	CharacterID     int        `json:"character_id"`
	CharacterName   string     `json:"character_name"`
	UserID          string     `json:"user_id"`
	CorporationID   int        `json:"corporation_id"`
	CorporationName string     `json:"corporation_name"`
	AllianceID      int        `json:"alliance_id"`
	AllianceName    string     `json:"alliance_name"`
	Enabled         bool       `json:"enabled"`
	Banned          bool       `json:"banned"`
	Position        int        `json:"position"`
	LastLogin       *time.Time `json:"last_login,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}