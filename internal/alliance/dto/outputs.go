package dto

import "time"

// AllianceInfo represents alliance information from EVE ESI according to official specification
// https://esi.evetech.net/meta/openapi.json - AlliancesAllianceIdGet schema
type AllianceInfo struct {
	Name                  string    `json:"name" description:"the full name of the alliance" example:"Test Alliance Please Ignore"`
	CreatorID             int       `json:"creator_id" description:"ID of the character that created the alliance" example:"12345"`
	CreatorCorporationID  int       `json:"creator_corporation_id" description:"ID of the corporation that created the alliance" example:"45678"`
	Ticker                string    `json:"ticker" description:"the short name of the alliance" example:"TEST"`
	DateFounded           time.Time `json:"date_founded" description:"Date the alliance was founded" format:"date-time"`
	ExecutorCorporationID *int      `json:"executor_corporation_id,omitempty" description:"the executor corporation ID, if this alliance is not closed" example:"98356193"`
	FactionID             *int      `json:"faction_id,omitempty" description:"Faction ID this alliance is fighting for, if enlisted in factional warfare" example:"500001"`
}

// AllianceInfoOutput represents an alliance info response (Huma wrapper)
type AllianceInfoOutput struct {
	Body AllianceInfo `json:"body"`
}

// AllianceListOutput represents the response for listing all alliances according to ESI specification
// https://esi.evetech.net/meta/openapi.json - AlliancesGet schema
type AllianceListOutput struct {
	Body []int64 `json:"body" description:"List of alliance IDs"`
}

// AllianceCorporationsOutput represents the response for listing alliance member corporations according to ESI specification
// https://esi.evetech.net/meta/openapi.json - AlliancesAllianceIdCorporationsGet schema
type AllianceCorporationsOutput struct {
	Body []int64 `json:"body" description:"List of corporation IDs that are members of the alliance"`
}