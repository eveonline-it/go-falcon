package dto

import (
	"github.com/go-playground/validator/v10"
)

// =============================================================================
// HUMA INPUT DTOs (consolidated from huma_requests.go)
// =============================================================================


// UserUpdateRequest represents user status update request
type UserUpdateRequest struct {
	Enabled  *bool   `json:"enabled,omitempty"`                                         // Enable/disable user
	Banned   *bool   `json:"banned,omitempty"`                                          // Ban/unban user  
	Invalid  *bool   `json:"invalid,omitempty"`                                         // Set validity status
	Position *int    `json:"position,omitempty" validate:"omitempty,min=0"`             // Update position/rank
	Notes    *string `json:"notes,omitempty" validate:"omitempty,max=1000"`             // Update administrative notes
}


// ValidateUserUpdateRequest validates the user update request
func ValidateUserUpdateRequest(req *UserUpdateRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// UserStatsInput represents the input for getting user statistics (no body needed)
type UserStatsInput struct {
	// No parameters needed
}

// UserGetInput represents the input for getting a specific user
type UserGetInput struct {
	CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}

// UserUpdateInput represents the input for updating a user
type UserUpdateInput struct {
	CharacterID int               `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Body        UserUpdateRequest `json:"body"`
}

// UserCharactersInput represents the input for getting user characters by user ID
type UserCharactersInput struct {
	UserID string `path:"user_id" validate:"required" doc:"User UUID"`
}

// UserListInput represents the input for listing users with pagination and filtering
type UserListInput struct {
	Page      int    `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize  int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
	Query     string `query:"query" doc:"Search by character name or ID"`
	Enabled   string `query:"enabled" doc:"Filter by enabled status (true/false)"`
	Banned    string `query:"banned" doc:"Filter by banned status (true/false)"`
	Invalid   string `query:"invalid" doc:"Filter by invalid status (true/false)"`
	Position  int    `query:"position" doc:"Filter by position value (0 means no filter)"`
	SortBy    string `query:"sort_by" enum:"character_name,created_at,last_login,position" default:"created_at" doc:"Sort field"`
	SortOrder string `query:"sort_order" enum:"asc,desc" default:"desc" doc:"Sort order"`
}