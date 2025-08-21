package dto

// GetAllianceInput represents the input for getting alliance information
type GetAllianceInput struct {
	AllianceID int `path:"alliance_id" minimum:"99000000" maximum:"2147483647" description:"Alliance ID to retrieve information for" example:"99000001"`
}