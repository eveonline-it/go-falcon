package dto

// AuthInput provides common authentication headers for secured endpoints
type AuthInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication" example:"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Cookie        string `header:"Cookie" doc:"Authentication cookie" example:"falcon_auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ReloadSDERequest represents a request to reload SDE data from files
type ReloadSDERequest struct {
	// DataTypes specifies which SDE data types to reload
	// If empty, all data types will be reloaded
	DataTypes []string `json:"data_types,omitempty" example:"[\"agents\",\"types\"]" doc:"List of SDE data types to reload. Leave empty to reload all types."`
}
