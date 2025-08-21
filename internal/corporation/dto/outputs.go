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