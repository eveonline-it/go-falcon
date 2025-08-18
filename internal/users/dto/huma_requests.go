package dto

// UserStatsInput represents the input for getting user statistics (no body needed)
type UserStatsInput struct {
	// No parameters needed
}

// UserStatsOutput represents the output for getting user statistics
type UserStatsOutput struct {
	Body UserStatsResponse `json:"body"`
}

// UserListInput represents the input for listing users
type UserListInput struct {
	Query     string `query:"query" validate:"omitempty" maxLength:"100" doc:"Search by character name or ID"`
	Enabled   string `query:"enabled" validate:"omitempty,oneof=true false" doc:"Filter by enabled status"`
	Banned    string `query:"banned" validate:"omitempty,oneof=true false" doc:"Filter by banned status"`
	Invalid   string `query:"invalid" validate:"omitempty,oneof=true false" doc:"Filter by invalid status"`
	Position  int    `query:"position" validate:"omitempty" minimum:"0" doc:"Filter by position value"`
	Page      int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize  int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Items per page"`
	SortBy    string `query:"sort_by" validate:"omitempty,oneof=character_name created_at last_login position" doc:"Sort field"`
	SortOrder string `query:"sort_order" validate:"omitempty,oneof=asc desc" doc:"Sort order"`
}

// UserListOutput represents the output for listing users
type UserListOutput struct {
	Body UserListResponse `json:"body"`
}

// UserGetInput represents the input for getting a specific user
type UserGetInput struct {
	CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}

// UserGetOutput represents the output for getting a specific user
type UserGetOutput struct {
	Body UserResponse `json:"body"`
}

// UserUpdateInput represents the input for updating a user
type UserUpdateInput struct {
	CharacterID int               `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Body        UserUpdateRequest `json:"body"`
}

// UserUpdateOutput represents the output for updating a user
type UserUpdateOutput struct {
	Body UserResponse `json:"body"`
}

// UserCharactersInput represents the input for getting user characters by user ID
type UserCharactersInput struct {
	UserID string `path:"user_id" validate:"required" doc:"User UUID"`
}

// UserCharactersOutput represents the output for getting user characters
type UserCharactersOutput struct {
	Body CharacterListResponse `json:"body"`
}