package dto

import "time"

// CorporationInfo represents corporation information from EVE ESI
type CorporationInfo struct {
	AllianceID      *int      `json:"alliance_id,omitempty" description:"Alliance ID if corporation is in an alliance"`
	CEOCharacterID  int       `json:"ceo_id" description:"Character ID of the corporation CEO" example:"661916654"`
	CreatorID       int       `json:"creator_id" description:"Character ID who created the corporation" example:"661916654"`
	DateFounded     time.Time `json:"date_founded" description:"Date the corporation was founded"`
	Description     string    `json:"description" description:"Corporation description"`
	FactionID       *int      `json:"faction_id,omitempty" description:"Faction ID if corporation belongs to a faction"`
	HomeStationID   *int      `json:"home_station_id,omitempty" description:"Home station ID"`
	MemberCount     int       `json:"member_count" description:"Number of members in the corporation" example:"158"`
	Name            string    `json:"name" description:"Corporation name" example:"DO.IT"`
	Shares          *int64    `json:"shares,omitempty" description:"Number of shares the corporation has"`
	TaxRate         float64   `json:"tax_rate" description:"Tax rate for corporation members" example:"0.05"`
	Ticker          string    `json:"ticker" description:"Corporation ticker" example:".IT"`
	URL             *string   `json:"url,omitempty" description:"Corporation website URL"`
	WarEligible     *bool     `json:"war_eligible,omitempty" description:"Whether the corporation is eligible for wars"`
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
	CorporationID int       `json:"corporation_id" description:"Corporation ID" example:"98000001"`
	Name          string    `json:"name" description:"Corporation name" example:"Dreddit"`
	Ticker        string    `json:"ticker" description:"Corporation ticker" example:"B0RT"`
	MemberCount   int       `json:"member_count" description:"Number of members" example:"3500"`
	AllianceID    *int      `json:"alliance_id,omitempty" description:"Alliance ID if in an alliance"`
	UpdatedAt     time.Time `json:"updated_at" description:"Last update timestamp"`
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