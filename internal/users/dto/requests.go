package dto

import (
	"github.com/go-playground/validator/v10"
)

// UserSearchRequest represents user search and filter parameters
type UserSearchRequest struct {
	Query      string `json:"query" form:"query" validate:"omitempty,max=100"`           // Search by character name or ID
	Enabled    *bool  `json:"enabled" form:"enabled"`                                    // Filter by enabled status
	Banned     *bool  `json:"banned" form:"banned"`                                      // Filter by banned status
	Invalid    *bool  `json:"invalid" form:"invalid"`                                    // Filter by validity status
	Position   *int   `json:"position" form:"position" validate:"omitempty,min=0"`       // Filter by position
	Page       int    `json:"page" form:"page" validate:"omitempty,min=1"`              // Page number (1-based)
	PageSize   int    `json:"page_size" form:"page_size" validate:"omitempty,min=1,max=100"` // Items per page (max 100)
	SortBy     string `json:"sort_by" form:"sort_by" validate:"omitempty,oneof=character_name created_at last_login position"` // Sort field
	SortOrder  string `json:"sort_order" form:"sort_order" validate:"omitempty,oneof=asc desc"` // Sort order
}

// UserUpdateRequest represents user status update request
type UserUpdateRequest struct {
	Enabled  *bool   `json:"enabled,omitempty"`                                         // Enable/disable user
	Banned   *bool   `json:"banned,omitempty"`                                          // Ban/unban user  
	Invalid  *bool   `json:"invalid,omitempty"`                                         // Set validity status
	Position *int    `json:"position,omitempty" validate:"omitempty,min=0"`             // Update position/rank
	Notes    *string `json:"notes,omitempty" validate:"omitempty,max=1000"`             // Update administrative notes
}

// ValidateUserSearchRequest validates the user search request
func ValidateUserSearchRequest(req *UserSearchRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// ValidateUserUpdateRequest validates the user update request
func ValidateUserUpdateRequest(req *UserUpdateRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// SetDefaults sets default values for UserSearchRequest
func (r *UserSearchRequest) SetDefaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	if r.SortBy == "" {
		r.SortBy = "character_name"
	}
	if r.SortOrder == "" {
		r.SortOrder = "asc"
	}
}