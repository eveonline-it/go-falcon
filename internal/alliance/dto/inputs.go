package dto

// GetAllianceInput represents the input for getting alliance information
type GetAllianceInput struct {
	AllianceID int `path:"alliance_id" minimum:"99000000" maximum:"2147483647" description:"Alliance ID to retrieve information for" example:"99000001"`
}

// ListAlliancesInput represents the input for listing all alliances (no parameters needed)
type ListAlliancesInput struct {}

// GetAllianceCorporationsInput represents the input for getting alliance member corporations
type GetAllianceCorporationsInput struct {
	AllianceID int `path:"alliance_id" minimum:"99000000" maximum:"2147483647" description:"Alliance ID to retrieve member corporations for" example:"99000001"`
}