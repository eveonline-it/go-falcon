package dto

// GetAllianceInput represents the input for getting alliance information
type GetAllianceInput struct {
	AllianceID int `path:"alliance_id" minimum:"99000000" maximum:"2147483647" description:"Alliance ID to retrieve information for" example:"99000001"`
}

// ListAlliancesInput represents the input for listing all alliances (no parameters needed)
type ListAlliancesInput struct{}

// GetAllianceCorporationsInput represents the input for getting alliance member corporations
type GetAllianceCorporationsInput struct {
	AllianceID int `path:"alliance_id" minimum:"99000000" maximum:"2147483647" description:"Alliance ID to retrieve member corporations for" example:"99000001"`
}

// SearchAlliancesByNameInput represents the input for searching alliances by name
type SearchAlliancesByNameInput struct {
	Name string `query:"name" validate:"required" minLength:"3" maxLength:"100" description:"Alliance name to search for (minimum 3 characters)" example:"Goonswarm"`
}

// BulkImportAlliancesInput represents the input for bulk importing alliances
type BulkImportAlliancesInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Authentication cookie"`
}
