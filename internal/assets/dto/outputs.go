package dto

import (
	"time"

	"go-falcon/internal/assets/models"
)

// AssetResponse represents an asset in the response
type AssetResponse struct {
	ItemID          int64   `json:"item_id" doc:"Unique item instance ID"`
	TypeID          int32   `json:"type_id" doc:"EVE type ID"`
	TypeName        string  `json:"type_name,omitempty" doc:"Item type name"`
	LocationID      int64   `json:"location_id" doc:"Location ID"`
	LocationType    string  `json:"location_type" doc:"Location type (station/structure/other)"`
	LocationFlag    string  `json:"location_flag" doc:"Location flag (Hangar/Cargo/etc)"`
	LocationName    string  `json:"location_name,omitempty" doc:"Location name"`
	Quantity        int32   `json:"quantity" doc:"Item quantity"`
	IsSingleton     bool    `json:"is_singleton" doc:"Whether item is singleton"`
	IsBlueprintCopy bool    `json:"is_blueprint_copy,omitempty" doc:"Whether blueprint is copy"`
	Name            string  `json:"name,omitempty" doc:"Custom item name"`
	MarketPrice     float64 `json:"market_price,omitempty" doc:"Market price per unit"`
	TotalValue      float64 `json:"total_value,omitempty" doc:"Total value"`
	SolarSystemID   int32   `json:"solar_system_id,omitempty" doc:"Solar system ID"`
	SolarSystemName string  `json:"solar_system_name,omitempty" doc:"Solar system name"`
	RegionID        int32   `json:"region_id,omitempty" doc:"Region ID"`
	RegionName      string  `json:"region_name,omitempty" doc:"Region name"`
	ParentItemID    *int64  `json:"parent_item_id,omitempty" doc:"Parent container/ship ID"`
	IsContainer     bool    `json:"is_container" doc:"Whether item is a container"`
}

// AssetListResponse represents a list of assets
type AssetListResponse struct {
	Assets      []AssetResponse `json:"assets" doc:"List of assets"`
	Total       int             `json:"total" doc:"Total number of assets"`
	TotalValue  float64         `json:"total_value,omitempty" doc:"Total value of all assets"`
	Page        int             `json:"page,omitempty" doc:"Current page"`
	PageSize    int             `json:"page_size,omitempty" doc:"Page size"`
	LastUpdated time.Time       `json:"last_updated" doc:"Last update time"`
}

// AssetLocationSummary represents a summary of assets at a location
type AssetLocationSummary struct {
	LocationID      int64   `json:"location_id" doc:"Location ID"`
	LocationName    string  `json:"location_name" doc:"Location name"`
	LocationType    string  `json:"location_type" doc:"Location type"`
	SolarSystemName string  `json:"solar_system_name,omitempty" doc:"Solar system name"`
	RegionName      string  `json:"region_name,omitempty" doc:"Region name"`
	ItemCount       int32   `json:"item_count" doc:"Number of items"`
	UniqueTypes     int32   `json:"unique_types" doc:"Number of unique item types"`
	TotalValue      float64 `json:"total_value" doc:"Total value"`
}

// AssetSummaryResponse represents a summary of all assets
type AssetSummaryResponse struct {
	CharacterID   int32                  `json:"character_id,omitempty" doc:"Character ID"`
	CorporationID int32                  `json:"corporation_id,omitempty" doc:"Corporation ID"`
	TotalItems    int32                  `json:"total_items" doc:"Total number of items"`
	UniqueTypes   int32                  `json:"unique_types" doc:"Number of unique item types"`
	TotalValue    float64                `json:"total_value" doc:"Total value of all assets"`
	LocationCount int                    `json:"location_count" doc:"Number of locations with assets"`
	Locations     []AssetLocationSummary `json:"locations" doc:"Asset summaries by location"`
	LastUpdated   time.Time              `json:"last_updated" doc:"Last update time"`
}

// AssetTrackingResponse represents an asset tracking configuration
type AssetTrackingResponse struct {
	ID              string    `json:"id" doc:"Tracking configuration ID"`
	CharacterID     int32     `json:"character_id" doc:"Character ID"`
	CorporationID   int32     `json:"corporation_id,omitempty" doc:"Corporation ID"`
	Name            string    `json:"name" doc:"Tracking name"`
	Description     string    `json:"description,omitempty" doc:"Description"`
	LocationIDs     []int64   `json:"location_ids" doc:"Tracked location IDs"`
	TypeIDs         []int32   `json:"type_ids,omitempty" doc:"Tracked type IDs"`
	Enabled         bool      `json:"enabled" doc:"Whether tracking is enabled"`
	NotifyThreshold float64   `json:"notify_threshold,omitempty" doc:"Notification threshold"`
	LastChecked     time.Time `json:"last_checked,omitempty" doc:"Last check time"`
	LastValue       float64   `json:"last_value,omitempty" doc:"Last tracked value"`
	CreatedAt       time.Time `json:"created_at" doc:"Creation time"`
	UpdatedAt       time.Time `json:"updated_at" doc:"Last update time"`
}

// AssetSnapshotResponse represents an asset snapshot
type AssetSnapshotResponse struct {
	ID            string    `json:"id" doc:"Snapshot ID"`
	CharacterID   int32     `json:"character_id" doc:"Character ID"`
	CorporationID int32     `json:"corporation_id,omitempty" doc:"Corporation ID"`
	LocationID    int64     `json:"location_id" doc:"Location ID"`
	TotalValue    float64   `json:"total_value" doc:"Total value"`
	ItemCount     int32     `json:"item_count" doc:"Number of items"`
	UniqueTypes   int32     `json:"unique_types" doc:"Number of unique types"`
	SnapshotTime  time.Time `json:"snapshot_time" doc:"Snapshot timestamp"`
}

// RefreshAssetsResponse represents the result of an asset refresh
type RefreshAssetsResponse struct {
	CharacterID   int32     `json:"character_id,omitempty" doc:"Character ID"`
	CorporationID int32     `json:"corporation_id,omitempty" doc:"Corporation ID"`
	ItemsUpdated  int       `json:"items_updated" doc:"Number of items updated"`
	NewItems      int       `json:"new_items" doc:"Number of new items"`
	RemovedItems  int       `json:"removed_items" doc:"Number of removed items"`
	TotalValue    float64   `json:"total_value" doc:"Total value of assets"`
	UpdatedAt     time.Time `json:"updated_at" doc:"Update timestamp"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body AssetModuleStatusResponse `json:"body"`
}

// AssetListOutput represents the asset list response
type AssetListOutput struct {
	Body AssetListResponse `json:"body"`
}

// AssetSummaryOutput represents the asset summary response
type AssetSummaryOutput struct {
	Body AssetSummaryResponse `json:"body"`
}

// RefreshAssetsOutput represents the refresh assets response
type RefreshAssetsOutput struct {
	Body RefreshAssetsResponse `json:"body"`
}

// AssetModuleStatusResponse represents the actual status response data
type AssetModuleStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,degraded,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}

// ToAssetResponse converts a model to a response DTO
func ToAssetResponse(asset *models.Asset) AssetResponse {
	return AssetResponse{
		ItemID:          asset.ItemID,
		TypeID:          asset.TypeID,
		TypeName:        asset.TypeName,
		LocationID:      asset.LocationID,
		LocationType:    asset.LocationType,
		LocationFlag:    asset.LocationFlag,
		LocationName:    asset.LocationName,
		Quantity:        asset.Quantity,
		IsSingleton:     asset.IsSingleton,
		IsBlueprintCopy: asset.IsBlueprintCopy,
		Name:            asset.Name,
		MarketPrice:     asset.MarketPrice,
		TotalValue:      asset.TotalValue,
		SolarSystemID:   asset.SolarSystemID,
		SolarSystemName: asset.SolarSystemName,
		RegionID:        asset.RegionID,
		RegionName:      asset.RegionName,
		ParentItemID:    asset.ParentItemID,
		IsContainer:     asset.IsContainer,
	}
}

// ToAssetTrackingResponse converts a model to a response DTO
func ToAssetTrackingResponse(tracking *models.AssetTracking) AssetTrackingResponse {
	return AssetTrackingResponse{
		ID:              tracking.ID.Hex(),
		CharacterID:     tracking.CharacterID,
		CorporationID:   tracking.CorporationID,
		Name:            tracking.Name,
		Description:     tracking.Description,
		LocationIDs:     tracking.LocationIDs,
		TypeIDs:         tracking.TypeIDs,
		Enabled:         tracking.Enabled,
		NotifyThreshold: tracking.NotifyThreshold,
		LastChecked:     tracking.LastChecked,
		LastValue:       tracking.LastValue,
		CreatedAt:       tracking.CreatedAt,
		UpdatedAt:       tracking.UpdatedAt,
	}
}

// ToAssetSnapshotResponse converts a model to a response DTO
func ToAssetSnapshotResponse(snapshot *models.AssetSnapshot) AssetSnapshotResponse {
	return AssetSnapshotResponse{
		ID:            snapshot.ID.Hex(),
		CharacterID:   snapshot.CharacterID,
		CorporationID: snapshot.CorporationID,
		LocationID:    snapshot.LocationID,
		TotalValue:    snapshot.TotalValue,
		ItemCount:     snapshot.ItemCount,
		UniqueTypes:   snapshot.UniqueTypes,
		SnapshotTime:  snapshot.SnapshotTime,
	}
}
