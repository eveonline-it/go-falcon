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

// AllianceSearchInfo represents an alliance in search results
type AllianceSearchInfo struct {
	AllianceID            int       `json:"alliance_id" description:"Alliance ID" example:"99000001"`
	Name                  string    `json:"name" description:"Alliance name" example:"Goonswarm Federation"`
	Ticker                string    `json:"ticker" description:"Alliance ticker" example:"CONDI"`
	ExecutorCorporationID *int      `json:"executor_corporation_id,omitempty" description:"Executor corporation ID if not closed"`
	DateFounded           time.Time `json:"date_founded" description:"Date the alliance was founded"`
	UpdatedAt             time.Time `json:"updated_at" description:"Last update timestamp"`
}

// SearchAlliancesResult represents search results for alliances
type SearchAlliancesResult struct {
	Alliances []AllianceSearchInfo `json:"alliances" description:"List of matching alliances"`
	Count     int                  `json:"count" description:"Number of alliances found"`
}

// SearchAlliancesByNameOutput represents the search response (Huma wrapper)
type SearchAlliancesByNameOutput struct {
	Body SearchAlliancesResult `json:"body"`
}

// BulkImportStats represents statistics for bulk alliance import operation
type BulkImportStats struct {
	TotalAlliances int `json:"total_alliances" description:"Total number of alliance IDs retrieved from ESI"`
	Processed      int `json:"processed" description:"Number of alliances processed"`
	Updated        int `json:"updated" description:"Number of existing alliances updated"`
	Created        int `json:"created" description:"Number of new alliances created"`
	Failed         int `json:"failed" description:"Number of alliances that failed to import"`
	Skipped        int `json:"skipped" description:"Number of alliances skipped"`
}

// BulkImportAlliancesOutput represents the bulk import response (Huma wrapper)
type BulkImportAlliancesOutput struct {
	Body BulkImportStats `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body AllianceStatusResponse `json:"body"`
}

// AllianceStatusResponse represents the actual status response data
type AllianceStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}