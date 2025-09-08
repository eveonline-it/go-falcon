package dto

// GetStructureRequest represents a request to get structure information
type GetStructureRequest struct {
	StructureID   int64  `path:"structure_id" json:"structure_id" minimum:"1" doc:"EVE structure ID"`
	Authorization string `header:"authorization" json:"-" doc:"Bearer token for authentication"`
	Cookie        string `header:"cookie" json:"-" doc:"Cookie header for authentication"`
}

// GetStructuresBySystemRequest represents a request to get structures in a system
type GetStructuresBySystemRequest struct {
	SolarSystemID int32  `path:"solar_system_id" json:"solar_system_id" minimum:"1" doc:"Solar system ID"`
	Authorization string `header:"authorization" json:"-" doc:"Bearer token for authentication"`
	Cookie        string `header:"cookie" json:"-" doc:"Cookie header for authentication"`
}

// GetStructuresByOwnerRequest represents a request to get structures by owner
type GetStructuresByOwnerRequest struct {
	OwnerID int32 `path:"owner_id" json:"owner_id" minimum:"1" doc:"Owner corporation/alliance ID"`
}

// RefreshStructureRequest represents a request to refresh structure data
type RefreshStructureRequest struct {
	StructureID int64 `path:"structure_id" json:"structure_id" minimum:"1" doc:"EVE structure ID"`
	CharacterID int32 `json:"character_id,omitempty" doc:"Character ID with access (for player structures)"`
}

// BulkRefreshStructuresRequest represents a request to refresh multiple structures
type BulkRefreshStructuresRequest struct {
	StructureIDs []int64 `json:"structure_ids" minItems:"1" maxItems:"100" doc:"List of structure IDs to refresh"`
	CharacterID  int32   `json:"character_id,omitempty" doc:"Character ID with access (for player structures)"`
}
