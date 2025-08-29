package dto

import "time"

// CharacterInfo represents basic character information
type CharacterInfo struct {
	CharacterID int    `json:"character_id" description:"Character ID"`
	Name        string `json:"name" description:"Character name"`
}

// StationInfo represents station information from SDE
type StationInfo struct {
	StationID                int     `json:"station_id" description:"Station ID"`
	ConstellationID          int     `json:"constellation_id" description:"Constellation ID where the station is located"`
	SolarSystemID            int     `json:"solar_system_id" description:"Solar system ID where the station is located"`
	RegionID                 int     `json:"region_id" description:"Region ID where the station is located"`
	CorporationID            int     `json:"corporation_id" description:"Corporation that owns the station"`
	DockingCostPerVolume     float64 `json:"docking_cost_per_volume" description:"Docking cost per volume"`
	MaxShipVolumeDockable    float64 `json:"max_ship_volume_dockable" description:"Maximum ship volume that can dock"`
	OfficeRentalCost         int     `json:"office_rental_cost" description:"Cost to rent an office"`
	ReprocessingEfficiency   float64 `json:"reprocessing_efficiency" description:"Reprocessing efficiency"`
	ReprocessingStationsTake float64 `json:"reprocessing_stations_take" description:"Station's take from reprocessing"`
	Security                 float64 `json:"security" description:"Security status"`
}

// CorporationInfo represents corporation information from EVE ESI
type CorporationInfo struct {
	AllianceID     *int           `json:"alliance_id,omitempty" description:"Alliance ID if corporation is in an alliance"`
	CEOCharacterID int            `json:"ceo_id" description:"Character ID of the corporation CEO" example:"661916654"`
	CEO            *CharacterInfo `json:"ceo,omitempty" description:"CEO character information"`
	CreatorID      int            `json:"creator_id" description:"Character ID who created the corporation" example:"661916654"`
	Creator        *CharacterInfo `json:"creator,omitempty" description:"Creator character information"`
	DateFounded    time.Time      `json:"date_founded" description:"Date the corporation was founded"`
	Description    string         `json:"description" description:"Corporation description"`
	FactionID      *int           `json:"faction_id,omitempty" description:"Faction ID if corporation belongs to a faction"`
	HomeStationID  *int           `json:"home_station_id,omitempty" description:"Home station ID"`
	HomeStation    *StationInfo   `json:"home_station,omitempty" description:"Home station information from SDE"`
	MemberCount    int            `json:"member_count" description:"Number of members in the corporation" example:"158"`
	Name           string         `json:"name" description:"Corporation name" example:"DO.IT"`
	Shares         *int64         `json:"shares,omitempty" description:"Number of shares the corporation has"`
	TaxRate        float64        `json:"tax_rate" description:"Tax rate for corporation members" example:"0.05"`
	Ticker         string         `json:"ticker" description:"Corporation ticker" example:".IT"`
	URL            *string        `json:"url,omitempty" description:"Corporation website URL"`
	WarEligible    *bool          `json:"war_eligible,omitempty" description:"Whether the corporation is eligible for wars"`
}

// CorporationInfoOutput represents a corporation info response (Huma wrapper)
type CorporationInfoOutput struct {
	Body CorporationInfo `json:"body"`
}

// CorporationErrorOutput represents error responses
type CorporationErrorOutput struct {
	Error   string `json:"error" description:"Error message"`
	Details string `json:"details,omitempty" description:"Additional error details"`
}

// CorporationSearchInfo represents a corporation in search results
type CorporationSearchInfo struct {
	CorporationID  int       `json:"corporation_id" description:"Corporation ID" example:"98701142"`
	Name           string    `json:"name" description:"Corporation name" example:"Dreddit"`
	Ticker         string    `json:"ticker" description:"Corporation ticker" example:"B0RT"`
	CEOCharacterID int       `json:"ceo_id" description:"Character ID of the corporation CEO" example:"661916654"`
	MemberCount    int       `json:"member_count" description:"Number of members" example:"3500"`
	AllianceID     *int      `json:"alliance_id,omitempty" description:"Alliance ID if in an alliance"`
	UpdatedAt      time.Time `json:"updated_at" description:"Last update timestamp"`
}

// SearchCorporationsResult represents search results for corporations
type SearchCorporationsResult struct {
	Corporations []CorporationSearchInfo `json:"corporations" description:"List of matching corporations"`
	Count        int                     `json:"count" description:"Number of corporations found"`
}

// SearchCorporationsByNameOutput represents the search response (Huma wrapper)
type SearchCorporationsByNameOutput struct {
	Body SearchCorporationsResult `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body CorporationStatusResponse `json:"body"`
}

// CorporationStatusResponse represents the actual status response data
type CorporationStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}

// MemberTrackingInfo represents member tracking information
type MemberTrackingInfo struct {
	BaseID       *int       `json:"base_id,omitempty" description:"Base ID where the member is located"`
	CharacterID  int        `json:"character_id" description:"Character ID of the member"`
	LocationID   *int64     `json:"location_id,omitempty" description:"Location ID where the member is"`
	LocationName *string    `json:"location_name,omitempty" description:"Name of the location where the member is"`
	LogoffDate   *time.Time `json:"logoff_date,omitempty" description:"Last logoff date"`
	LogonDate    *time.Time `json:"logon_date,omitempty" description:"Last logon date"`
	ShipTypeID   *int       `json:"ship_type_id,omitempty" description:"Type ID of the ship the member is flying"`
	StartDate    *time.Time `json:"start_date,omitempty" description:"Date when the member joined the corporation"`
}

// MemberTrackingResult represents member tracking results
type MemberTrackingResult struct {
	CorporationID int                  `json:"corporation_id" description:"Corporation ID"`
	Members       []MemberTrackingInfo `json:"members" description:"List of member tracking information"`
	Count         int                  `json:"count" description:"Number of members tracked"`
}

// CorporationMemberTrackingOutput represents the member tracking response (Huma wrapper)
type CorporationMemberTrackingOutput struct {
	Body MemberTrackingResult `json:"body"`
}

// CEOTokenInfo represents information about a CEO's token status
type CEOTokenInfo struct {
	CharacterID     int        `json:"character_id" description:"CEO character ID"`
	CharacterName   string     `json:"character_name" description:"CEO character name"`
	CorporationID   int        `json:"corporation_id" description:"Corporation ID"`
	CorporationName string     `json:"corporation_name,omitempty" description:"Corporation name"`
	Valid           bool       `json:"valid" description:"Whether the token is valid"`
	TokenExpiry     *time.Time `json:"token_expiry,omitempty" description:"Token expiration time"`
	LastLogin       *time.Time `json:"last_login,omitempty" description:"Last login time"`
}

// CEOTokenValidationResult represents the result of CEO token validation
type CEOTokenValidationResult struct {
	TotalCEOs     int            `json:"total_ceos" description:"Total number of CEOs found"`
	ValidTokens   int            `json:"valid_tokens" description:"Number of CEOs with valid tokens"`
	InvalidTokens int            `json:"invalid_tokens" description:"Number of CEOs with invalid tokens"`
	NoProfile     int            `json:"no_profile" description:"Number of CEOs with no user profile"`
	InvalidCEOs   []CEOTokenInfo `json:"invalid_ceos" description:"List of CEOs with invalid tokens"`
	MissingCEOs   []int          `json:"missing_ceos" description:"List of CEO character IDs with no profile"`
	ExecutedAt    time.Time      `json:"executed_at" description:"When the validation was executed"`
}

// ValidateCEOTokensOutput represents the CEO token validation response (Huma wrapper)
type ValidateCEOTokensOutput struct {
	Body CEOTokenValidationResult `json:"body"`
}

// AllianceHistoryEntry represents a single alliance history record
type AllianceHistoryEntry struct {
	AllianceID *int      `json:"alliance_id,omitempty" description:"Alliance ID (null if corporation left all alliances)"`
	IsDeleted  bool      `json:"is_deleted,omitempty" description:"True if the alliance has been deleted"`
	RecordID   int       `json:"record_id" description:"Unique record ID for this history entry"`
	StartDate  time.Time `json:"start_date" description:"Date when the corporation joined this alliance"`
}

// CorporationAllianceHistoryResult represents the alliance history for a corporation
type CorporationAllianceHistoryResult struct {
	CorporationID int                    `json:"corporation_id" description:"Corporation ID"`
	History       []AllianceHistoryEntry `json:"history" description:"Alliance history entries, ordered by date"`
	Count         int                    `json:"count" description:"Number of history entries"`
}

// CorporationAllianceHistoryOutput represents the alliance history response (Huma wrapper)
type CorporationAllianceHistoryOutput struct {
	Body CorporationAllianceHistoryResult `json:"body"`
}

// CorporationMemberInfo represents basic corporation member information
type CorporationMemberInfo struct {
	CharacterID int `json:"character_id" description:"Character ID of the corporation member"`
}

// CorporationMembersResult represents the members list for a corporation
type CorporationMembersResult struct {
	CorporationID int                     `json:"corporation_id" description:"Corporation ID"`
	Members       []CorporationMemberInfo `json:"members" description:"List of corporation members"`
	Count         int                     `json:"count" description:"Number of members in the corporation"`
}

// CorporationMembersOutput represents the corporation members response (Huma wrapper)
type CorporationMembersOutput struct {
	Body CorporationMembersResult `json:"body"`
}
