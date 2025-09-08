package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Asset represents an item in EVE Online owned by a character or corporation
type Asset struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CharacterID     int32              `bson:"character_id" json:"character_id"`
	CorporationID   int32              `bson:"corporation_id,omitempty" json:"corporation_id,omitempty"`
	ItemID          int64              `bson:"item_id" json:"item_id"`                                         // Unique item instance ID
	TypeID          int32              `bson:"type_id" json:"type_id"`                                         // EVE type ID
	TypeName        string             `bson:"type_name,omitempty" json:"type_name,omitempty"`                 // Resolved from SDE
	LocationID      int64              `bson:"location_id" json:"location_id"`                                 // Structure/Station/Container ID
	LocationType    string             `bson:"location_type" json:"location_type"`                             // station/structure/other
	LocationFlag    string             `bson:"location_flag" json:"location_flag"`                             // Hangar/Slot designation
	LocationName    string             `bson:"location_name,omitempty" json:"location_name,omitempty"`         // Resolved location name
	Quantity        int32              `bson:"quantity" json:"quantity"`                                       // Item stack size
	IsSingleton     bool               `bson:"is_singleton" json:"is_singleton"`                               // Stackable vs unique items
	IsBlueprintCopy bool               `bson:"is_blueprint_copy,omitempty" json:"is_blueprint_copy,omitempty"` // BPC vs BPO
	Name            string             `bson:"name,omitempty" json:"name,omitempty"`                           // Custom name for ships/containers

	// Market data
	MarketPrice float64 `bson:"market_price,omitempty" json:"market_price,omitempty"`
	TotalValue  float64 `bson:"total_value,omitempty" json:"total_value,omitempty"`

	// Location hierarchy
	SolarSystemID   int32  `bson:"solar_system_id,omitempty" json:"solar_system_id,omitempty"`
	SolarSystemName string `bson:"solar_system_name,omitempty" json:"solar_system_name,omitempty"`
	RegionID        int32  `bson:"region_id,omitempty" json:"region_id,omitempty"`
	RegionName      string `bson:"region_name,omitempty" json:"region_name,omitempty"`

	// Container hierarchy (for items inside containers/ships)
	ParentItemID *int64 `bson:"parent_item_id" json:"parent_item_id,omitempty"`
	IsContainer  bool   `bson:"is_container" json:"is_container"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// AssetSnapshot represents a point-in-time snapshot of assets for tracking
type AssetSnapshot struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CharacterID   int32              `bson:"character_id" json:"character_id"`
	CorporationID int32              `bson:"corporation_id,omitempty" json:"corporation_id,omitempty"`
	LocationID    int64              `bson:"location_id" json:"location_id"`
	TotalValue    float64            `bson:"total_value" json:"total_value"`
	ItemCount     int32              `bson:"item_count" json:"item_count"`
	UniqueTypes   int32              `bson:"unique_types" json:"unique_types"`
	SnapshotTime  time.Time          `bson:"snapshot_time" json:"snapshot_time"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}

// AssetTracking represents user-defined asset monitoring configuration
type AssetTracking struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID `bson:"user_id" json:"user_id"`
	CharacterID     int32              `bson:"character_id" json:"character_id"`
	CorporationID   int32              `bson:"corporation_id,omitempty" json:"corporation_id,omitempty"`
	Name            string             `bson:"name" json:"name"`
	Description     string             `bson:"description,omitempty" json:"description,omitempty"`
	LocationIDs     []int64            `bson:"location_ids" json:"location_ids"`
	TypeIDs         []int32            `bson:"type_ids,omitempty" json:"type_ids,omitempty"` // Optional: specific types to track
	Enabled         bool               `bson:"enabled" json:"enabled"`
	NotifyThreshold float64            `bson:"notify_threshold,omitempty" json:"notify_threshold,omitempty"` // Value change threshold for notifications
	LastChecked     time.Time          `bson:"last_checked,omitempty" json:"last_checked,omitempty"`
	LastValue       float64            `bson:"last_value,omitempty" json:"last_value,omitempty"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// LocationFlag constants
const (
	LocationFlagAssetSafety            = "AssetSafety"
	LocationFlagAutoFit                = "AutoFit"
	LocationFlagCargo                  = "Cargo"
	LocationFlagCorpDeliveries         = "CorpDeliveries"
	LocationFlagCorpSAG1               = "CorpSAG1" // Division 1
	LocationFlagCorpSAG2               = "CorpSAG2" // Division 2
	LocationFlagCorpSAG3               = "CorpSAG3" // Division 3
	LocationFlagCorpSAG4               = "CorpSAG4" // Division 4
	LocationFlagCorpSAG5               = "CorpSAG5" // Division 5
	LocationFlagCorpSAG6               = "CorpSAG6" // Division 6
	LocationFlagCorpSAG7               = "CorpSAG7" // Division 7
	LocationFlagDroneBay               = "DroneBay"
	LocationFlagFighterBay             = "FighterBay"
	LocationFlagFighterTube0           = "FighterTube0"
	LocationFlagFleetHangar            = "FleetHangar"
	LocationFlagHangar                 = "Hangar"
	LocationFlagHangarAll              = "HangarAll"
	LocationFlagHiSlot0                = "HiSlot0"
	LocationFlagHiddenModifiers        = "HiddenModifiers"
	LocationFlagImplant                = "Implant"
	LocationFlagLoSlot0                = "LoSlot0"
	LocationFlagMedSlot0               = "MedSlot0"
	LocationFlagOfficeFolder           = "OfficeFolder"
	LocationFlagRigSlot0               = "RigSlot0"
	LocationFlagShipHangar             = "ShipHangar"
	LocationFlagSpecializedAmmoHold    = "SpecializedAmmoHold"
	LocationFlagSpecializedFuelBay     = "SpecializedFuelBay"
	LocationFlagSpecializedGasHold     = "SpecializedGasHold"
	LocationFlagSpecializedMineralHold = "SpecializedMineralHold"
	LocationFlagSpecializedOreHold     = "SpecializedOreHold"
	LocationFlagStructureFuel          = "StructureFuel"
	LocationFlagSubSystemBay           = "SubSystemBay"
	LocationFlagSubSystemSlot0         = "SubSystemSlot0"
	LocationFlagWardrobe               = "Wardrobe"
)

// LocationType constants
const (
	LocationTypeStation     = "station"
	LocationTypeStructure   = "structure"
	LocationTypeSolarSystem = "solar_system"
	LocationTypeOther       = "other"
)

// CollectionNames
const (
	AssetsCollection         = "assets"
	AssetSnapshotsCollection = "asset_snapshots"
	AssetTrackingCollection  = "asset_tracking"
)

// Container type IDs (hardcoded from existing implementation)
var ContainerTypeIDs = []int32{
	3465, 3466, 3467, 11488, 11489, 17363, 17364, 17365,
	24445, 33003, 33005, 33007, 33009, 33011,
	3293, 3296, 3297, 17366, 17367, 17368,
}
