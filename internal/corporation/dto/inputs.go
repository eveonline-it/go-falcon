package dto

// GetCorporationInput represents the input for getting corporation information
type GetCorporationInput struct {
	CorporationID int `path:"corporation_id" minimum:"1" description:"Corporation ID to retrieve information for" example:"98000001"`
}

// SearchCorporationsByNameInput represents the input for searching corporations by name
type SearchCorporationsByNameInput struct {
	Name string `query:"name" validate:"required" minLength:"3" maxLength:"100" description:"Corporation name to search for (minimum 3 characters)" example:"Dreddit"`
}