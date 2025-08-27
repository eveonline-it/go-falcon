package dto

import (
	"time"
)

// EnrichedCorporationInfo represents corporation information for enriched character responses
type EnrichedCorporationInfo struct {
	CorporationID  int       `json:"corporation_id"`
	Name           string    `json:"name"`
	Ticker         string    `json:"ticker"`
	MemberCount    int       `json:"member_count"`
	AllianceID     *int      `json:"alliance_id,omitempty"`
	CEOCharacterID int       `json:"ceo_character_id"`
	DateFounded    time.Time `json:"date_founded"`
	Description    string    `json:"description"`
	TaxRate        float64   `json:"tax_rate"`
	WarEligible    *bool     `json:"war_eligible,omitempty"`
}

// EnrichedAllianceInfo represents alliance information for enriched character responses
type EnrichedAllianceInfo struct {
	AllianceID            int       `json:"alliance_id"`
	Name                  string    `json:"name"`
	Ticker                string    `json:"ticker"`
	DateFounded           time.Time `json:"date_founded"`
	CreatorID             int       `json:"creator_id"`
	CreatorCorporationID  int       `json:"creator_corporation_id"`
	ExecutorCorporationID *int      `json:"executor_corporation_id,omitempty"`
	FactionID             *int      `json:"faction_id,omitempty"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	CharacterID   int       `json:"character_id"`
	UserID        string    `json:"user_id"`
	Banned        bool      `json:"banned"`
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
	Banned        bool       `json:"banned"`
	Position      int        `json:"position"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
	Valid         bool       `json:"valid"`
}

// EnrichedCharacterSummaryResponse represents character information enriched with profile data
type EnrichedCharacterSummaryResponse struct {
	// User management fields
	CharacterID   int        `json:"character_id"`
	CharacterName string     `json:"character_name"`
	UserID        string     `json:"user_id"`
	Banned        bool       `json:"banned"`
	Position      int        `json:"position"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
	Valid         bool       `json:"valid"`

	// Rich character profile data (optional, may not be available for all characters)
	CorporationID  *int       `json:"corporation_id,omitempty"`
	AllianceID     *int       `json:"alliance_id,omitempty"`
	SecurityStatus *float64   `json:"security_status,omitempty"`
	Birthday       *time.Time `json:"birthday,omitempty"`
	Gender         *string    `json:"gender,omitempty"`
	RaceID         *int       `json:"race_id,omitempty"`
	BloodlineID    *int       `json:"bloodline_id,omitempty"`
	AncestryID     *int       `json:"ancestry_id,omitempty"`
	FactionID      *int       `json:"faction_id,omitempty"`
	Description    *string    `json:"description,omitempty"`

	// Full corporation and alliance information (optional)
	Corporation *EnrichedCorporationInfo `json:"corporation,omitempty"`
	Alliance    *EnrichedAllianceInfo    `json:"alliance,omitempty"`

	// Portrait URLs (optional)
	Portraits *CharacterPortraits `json:"portraits,omitempty"`
}

// CharacterPortraits represents character portrait URLs in different sizes
type CharacterPortraits struct {
	Px64x64   string `json:"px64x64"`
	Px128x128 string `json:"px128x128"`
	Px256x256 string `json:"px256x256"`
	Px512x512 string `json:"px512x512"`
}

// UserStatsResponse represents user statistics
type UserStatsResponse struct {
	TotalUsers    int `json:"total_users"`
	DisabledUsers int `json:"disabled_users"`
	BannedUsers   int `json:"banned_users"`
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

// EnrichedCharacterListResponse represents the response for enriched character listing
type EnrichedCharacterListResponse struct {
	UserID     string                             `json:"user_id"`
	Characters []EnrichedCharacterSummaryResponse `json:"characters"`
	Count      int                                `json:"count"`
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

// EnrichedUserCharactersOutput represents the output for getting enriched user characters
type EnrichedUserCharactersOutput struct {
	Body EnrichedCharacterListResponse `json:"body"`
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

// UserReorderCharactersResponse represents the response after reordering user characters
type UserReorderCharactersResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"` // Number of characters reordered
}

// UserReorderCharactersOutput represents the output for reordering user characters
type UserReorderCharactersOutput struct {
	Body UserReorderCharactersResponse `json:"body"`
}
