package dto

// AuthInput provides common authentication headers for secured endpoints
type AuthInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication" example:"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Cookie        string `header:"Cookie" doc:"Authentication cookie" example:"falcon_auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ImportSDERequest represents a request to import SDE data to Redis
type ImportSDERequest struct {
	// DataTypes specifies which SDE data types to import
	// If empty, all data types will be imported
	DataTypes []string `json:"data_types,omitempty" example:"[\"agents\",\"types\"]" doc:"List of SDE data types to import. Leave empty to import all types."`

	// Force indicates whether to overwrite existing data in Redis
	Force bool `json:"force,omitempty" default:"false" doc:"Force import even if data already exists in Redis"`

	// BatchSize controls how many items to process in each batch
	BatchSize int `json:"batch_size,omitempty" default:"1000" minimum:"100" maximum:"10000" doc:"Number of items to process in each batch"`
}

// GetImportStatusRequest represents a request to get import status
type GetImportStatusRequest struct {
	// ImportID is the ID of the import operation to check
	ImportID string `path:"import_id" doc:"The ID of the import operation to check"`
}
