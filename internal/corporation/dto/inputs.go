package dto

// GetCorporationInput represents the input for getting corporation information
type GetCorporationInput struct {
	CorporationID int `path:"corporation_id" minimum:"1" description:"Corporation ID to retrieve information for" example:"98000001"`
}

// GetCorporationAuthInput represents the authenticated input for getting corporation information
type GetCorporationAuthInput struct {
	CorporationID int    `path:"corporation_id" minimum:"1" description:"Corporation ID to retrieve information for" example:"98000001"`
	Authorization string `header:"Authorization" description:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Authentication cookie"`
}

// SearchCorporationsByNameInput represents the input for searching corporations by name
type SearchCorporationsByNameInput struct {
	Name string `query:"name" validate:"required" minLength:"3" maxLength:"100" description:"Corporation name to search for (minimum 3 characters)" example:"Dreddit"`
}

// SearchCorporationsByNameAuthInput represents the authenticated input for searching corporations by name
type SearchCorporationsByNameAuthInput struct {
	Name          string `query:"name" validate:"required" minLength:"3" maxLength:"100" description:"Corporation name to search for (minimum 3 characters)" example:"Dreddit"`
	Authorization string `header:"Authorization" description:"JWT Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Authentication cookie"`
}
