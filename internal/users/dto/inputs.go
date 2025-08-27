package dto

import (
	"github.com/go-playground/validator/v10"
)

// =============================================================================
// HUMA INPUT DTOs (consolidated from huma_requests.go)
// =============================================================================

// UserUpdateRequest represents user status update request
type UserUpdateRequest struct {
	Enabled  *bool   `json:"enabled,omitempty"`                             // Enable/disable user
	Banned   *bool   `json:"banned,omitempty"`                              // Ban/unban user
	Position *int    `json:"position,omitempty" validate:"omitempty,min=0"` // Update position/rank
	Notes    *string `json:"notes,omitempty" validate:"omitempty,max=1000"` // Update administrative notes
}

// ValidateUserUpdateRequest validates the user update request
func ValidateUserUpdateRequest(req *UserUpdateRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// UserStatsInput represents the input for getting user statistics (no body needed)
type UserStatsInput struct {
	// No parameters needed
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// UserGetInput represents the input for getting a specific user
type UserGetInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// UserUpdateInput represents the input for updating a user
type UserUpdateInput struct {
	CharacterID   int               `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Body          UserUpdateRequest `json:"body"`
	Authorization string            `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string            `header:"Cookie" doc:"Authentication cookie"`
}

// UserCharactersInput represents the input for getting user characters by user ID
type UserCharactersInput struct {
	UserID        string `path:"user_id" validate:"required" doc:"User UUID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// UserDeleteInput represents the input for deleting a user character
type UserDeleteInput struct {
	CharacterID   int    `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// UserListInput represents the input for listing users with pagination and filtering
type UserListInput struct {
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
	Query         string `query:"query" doc:"Search by character name or ID"`
	Enabled       string `query:"enabled" doc:"Filter by enabled status (true/false)"`
	Banned        string `query:"banned" doc:"Filter by banned status (true/false)"`
	Position      int    `query:"position" doc:"Filter by position value (0 means no filter)"`
	SortBy        string `query:"sort_by" enum:"character_name,created_at,last_login,position" default:"created_at" doc:"Sort field"`
	SortOrder     string `query:"sort_order" enum:"asc,desc" default:"desc" doc:"Sort order"`
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}

// CharacterReorderRequest represents a character position change request
type CharacterReorderRequest struct {
	CharacterID int `json:"character_id" validate:"required,min=90000000" doc:"Character ID to reorder"`
	Position    int `json:"position" validate:"min=0" doc:"New position for the character"`
}

// UserReorderCharactersRequest represents the request body for reordering user characters
type UserReorderCharactersRequest struct {
	Characters []CharacterReorderRequest `json:"characters" validate:"required,dive" doc:"Array of character position updates"`
}

// ValidateUserReorderCharactersRequest validates the reorder request
func ValidateUserReorderCharactersRequest(req *UserReorderCharactersRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// UserReorderCharactersInput represents the input for reordering user characters
type UserReorderCharactersInput struct {
	UserID        string                       `path:"user_id" validate:"required" doc:"User UUID"`
	Body          UserReorderCharactersRequest `json:"body"`
	Authorization string                       `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string                       `header:"Cookie" doc:"Authentication cookie"`
}
