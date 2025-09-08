package dto

// GetCharacterAssetsRequest represents a request to get character assets
type GetCharacterAssetsRequest struct {
	CharacterID int32 `path:"character_id" json:"character_id" minimum:"1" doc:"EVE character ID"`
	LocationID  int64 `query:"location_id" json:"location_id,omitempty" doc:"Optional: filter by location ID (0 means no filter)"`
	Page        int   `query:"page" json:"page,omitempty" minimum:"1" default:"1" doc:"Page number"`
	PageSize    int   `query:"page_size" json:"page_size,omitempty" minimum:"1" maximum:"1000" default:"100" doc:"Items per page"`
}

// GetCorporationAssetsRequest represents a request to get corporation assets
type GetCorporationAssetsRequest struct {
	CorporationID int32 `path:"corporation_id" json:"corporation_id" minimum:"1" doc:"EVE corporation ID"`
	LocationID    int64 `query:"location_id" json:"location_id,omitempty" doc:"Optional: filter by location ID (0 means no filter)"`
	Division      int   `query:"division" json:"division,omitempty" minimum:"0" maximum:"7" doc:"Optional: filter by division (1-7, 0 means no filter)"`
	Page          int   `query:"page" json:"page,omitempty" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int   `query:"page_size" json:"page_size,omitempty" minimum:"1" maximum:"1000" default:"100" doc:"Items per page"`
}

// GetAssetsByLocationRequest represents a request to get assets at a specific location
type GetAssetsByLocationRequest struct {
	LocationID int64 `path:"location_id" json:"location_id" minimum:"1" doc:"Location ID (station/structure)"`
	Page       int   `query:"page" json:"page,omitempty" minimum:"1" default:"1" doc:"Page number"`
	PageSize   int   `query:"page_size" json:"page_size,omitempty" minimum:"1" maximum:"1000" default:"100" doc:"Items per page"`
}

// RefreshCharacterAssetsRequest represents a request to refresh character assets
type RefreshCharacterAssetsRequest struct {
	CharacterID int32 `path:"character_id" json:"character_id" minimum:"1" doc:"EVE character ID"`
}

// RefreshCorporationAssetsRequest represents a request to refresh corporation assets
type RefreshCorporationAssetsRequest struct {
	CorporationID int32 `path:"corporation_id" json:"corporation_id" minimum:"1" doc:"EVE corporation ID"`
	CharacterID   int32 `json:"character_id" doc:"Character ID with director/accountant roles"`
}

// CreateAssetTrackingRequest represents a request to create asset tracking
type CreateAssetTrackingRequest struct {
	CharacterID     int32   `json:"character_id" minimum:"1" doc:"EVE character ID"`
	CorporationID   *int32  `json:"corporation_id,omitempty" doc:"Optional: corporation ID for corp assets"`
	Name            string  `json:"name" minLength:"1" maxLength:"100" doc:"Tracking configuration name"`
	Description     string  `json:"description,omitempty" maxLength:"500" doc:"Optional description"`
	LocationIDs     []int64 `json:"location_ids" minItems:"1" maxItems:"50" doc:"Location IDs to track"`
	TypeIDs         []int32 `json:"type_ids,omitempty" maxItems:"100" doc:"Optional: specific type IDs to track"`
	NotifyThreshold float64 `json:"notify_threshold,omitempty" minimum:"0" doc:"Value change threshold for notifications"`
	Enabled         bool    `json:"enabled" default:"true" doc:"Whether tracking is enabled"`
}

// UpdateAssetTrackingRequest represents a request to update asset tracking
type UpdateAssetTrackingRequest struct {
	TrackingID      string   `path:"tracking_id" json:"tracking_id" doc:"Tracking configuration ID"`
	Name            *string  `json:"name,omitempty" minLength:"1" maxLength:"100" doc:"Tracking configuration name"`
	Description     *string  `json:"description,omitempty" maxLength:"500" doc:"Optional description"`
	LocationIDs     []int64  `json:"location_ids,omitempty" minItems:"1" maxItems:"50" doc:"Location IDs to track"`
	TypeIDs         []int32  `json:"type_ids,omitempty" maxItems:"100" doc:"Optional: specific type IDs to track"`
	NotifyThreshold *float64 `json:"notify_threshold,omitempty" minimum:"0" doc:"Value change threshold for notifications"`
	Enabled         *bool    `json:"enabled,omitempty" doc:"Whether tracking is enabled"`
}

// DeleteAssetTrackingRequest represents a request to delete asset tracking
type DeleteAssetTrackingRequest struct {
	TrackingID string `path:"tracking_id" json:"tracking_id" doc:"Tracking configuration ID"`
}

// GetAssetTrackingRequest represents a request to get asset tracking configurations
type GetAssetTrackingRequest struct {
	CharacterID   int32 `query:"character_id" json:"character_id,omitempty" doc:"Filter by character ID (0 means no filter)"`
	CorporationID int32 `query:"corporation_id" json:"corporation_id,omitempty" doc:"Filter by corporation ID (0 means no filter)"`
	Enabled       bool  `query:"enabled" json:"enabled,omitempty" doc:"Filter by enabled status (only used if explicitly set)"`
}

// GetAssetSnapshotsRequest represents a request to get asset snapshots
type GetAssetSnapshotsRequest struct {
	CharacterID   int32  `query:"character_id" json:"character_id,omitempty" doc:"Filter by character ID (0 means no filter)"`
	CorporationID int32  `query:"corporation_id" json:"corporation_id,omitempty" doc:"Filter by corporation ID (0 means no filter)"`
	LocationID    int64  `query:"location_id" json:"location_id,omitempty" doc:"Filter by location ID (0 means no filter)"`
	StartDate     string `query:"start_date" json:"start_date,omitempty" doc:"Start date (ISO 8601)"`
	EndDate       string `query:"end_date" json:"end_date,omitempty" doc:"End date (ISO 8601)"`
	Limit         int    `query:"limit" json:"limit,omitempty" minimum:"1" maximum:"1000" default:"100" doc:"Maximum results"`
}
