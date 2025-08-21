package dto

// GetCorporationInput represents the input for getting corporation information
type GetCorporationInput struct {
	CorporationID int `path:"corporation_id" minimum:"1" description:"Corporation ID to retrieve information for" example:"98000001"`
}