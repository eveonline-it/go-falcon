package dto

// CharacterAffiliation represents a character's corporation and alliance affiliations
type CharacterAffiliation struct {
	CharacterID   int `json:"character_id" bson:"character_id"`
	CorporationID int `json:"corporation_id" bson:"corporation_id"`
	AllianceID    int `json:"alliance_id,omitempty" bson:"alliance_id,omitempty"`
	FactionID     int `json:"faction_id,omitempty" bson:"faction_id,omitempty"`
}

// AffiliationUpdateStats tracks statistics for affiliation update operations
type AffiliationUpdateStats struct {
	TotalCharacters   int `json:"total_characters"`
	UpdatedCharacters int `json:"updated_characters"`
	FailedCharacters  int `json:"failed_characters"`
	SkippedCharacters int `json:"skipped_characters"`
	BatchesProcessed  int `json:"batches_processed"`
	Duration          int `json:"duration_seconds"`
}

// BatchAffiliationUpdate represents a batch update operation result
type BatchAffiliationUpdate struct {
	CharacterIDs []int                  `json:"character_ids"`
	Affiliations []CharacterAffiliation `json:"affiliations"`
	Errors       []string               `json:"errors,omitempty"`
}