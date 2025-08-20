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