package dto

import "time"

// CharacterProfile represents a character profile data
type CharacterProfile struct {
	CharacterID    int       `json:"character_id" doc:"EVE Online character ID"`
	Name           string    `json:"name" doc:"Character name"`
	CorporationID  int       `json:"corporation_id" doc:"Corporation ID"`
	AllianceID     int       `json:"alliance_id,omitempty" doc:"Alliance ID"`
	Birthday       time.Time `json:"birthday" doc:"Character birthday"`
	SecurityStatus float64   `json:"security_status" doc:"Security status"`
	Description    string    `json:"description,omitempty" doc:"Character description"`
	Gender         string    `json:"gender" doc:"Character gender"`
	RaceID         int       `json:"race_id" doc:"Race ID"`
	BloodlineID    int       `json:"bloodline_id" doc:"Bloodline ID"`
	AncestryID     int       `json:"ancestry_id,omitempty" doc:"Ancestry ID"`
	FactionID      int       `json:"faction_id,omitempty" doc:"Faction ID"`
	CreatedAt      time.Time `json:"created_at" doc:"Profile created timestamp"`
	UpdatedAt      time.Time `json:"updated_at" doc:"Profile updated timestamp"`
}

// CharacterProfileOutput represents a character profile response (Huma wrapper)
type CharacterProfileOutput struct {
	Body CharacterProfile `json:"body"`
}

// SearchCharactersResult represents search results for characters
type SearchCharactersResult struct {
	Characters []CharacterProfile `json:"characters" doc:"List of matching characters"`
	Count      int                `json:"count" doc:"Number of characters found"`
}

// SearchCharactersByNameOutput represents the search response (Huma wrapper)
type SearchCharactersByNameOutput struct {
	Body SearchCharactersResult `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body CharacterStatusResponse `json:"body"`
}

// CharacterStatusResponse represents the actual status response data
type CharacterStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}
