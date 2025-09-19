package dto

import (
	"time"

	"go-falcon/internal/mapservice/models"
)

// SignatureOutput represents a signature response
type SignatureOutput struct {
	ID            string     `json:"id" doc:"Signature database ID"`
	SystemID      int32      `json:"system_id" doc:"EVE System ID"`
	SystemName    string     `json:"system_name" doc:"System name"`
	SignatureID   string     `json:"signature_id" doc:"In-game signature ID"`
	Type          string     `json:"type" doc:"Signature type"`
	Name          string     `json:"name,omitempty" doc:"Signature name"`
	Description   string     `json:"description,omitempty" doc:"Description"`
	Strength      float32    `json:"strength,omitempty" doc:"Signal strength"`
	CreatedBy     string     `json:"created_by" doc:"Creator user ID"`
	CreatedByName string     `json:"created_by_name" doc:"Creator name"`
	UpdatedBy     string     `json:"updated_by,omitempty" doc:"Last updater ID"`
	UpdatedByName string     `json:"updated_by_name,omitempty" doc:"Last updater name"`
	SharingLevel  string     `json:"sharing_level" doc:"Visibility level"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" doc:"Expiration time"`
	CreatedAt     time.Time  `json:"created_at" doc:"Creation time"`
	UpdatedAt     time.Time  `json:"updated_at" doc:"Last update time"`
}

// WormholeOutput represents a wormhole connection response
type WormholeOutput struct {
	ID              string              `json:"id" doc:"Wormhole database ID"`
	FromSystemID    int32               `json:"from_system_id" doc:"Origin system ID"`
	FromSystemName  string              `json:"from_system_name" doc:"Origin system name"`
	ToSystemID      int32               `json:"to_system_id" doc:"Destination system ID"`
	ToSystemName    string              `json:"to_system_name" doc:"Destination system name"`
	FromSignatureID string              `json:"from_signature_id" doc:"Origin signature ID"`
	ToSignatureID   string              `json:"to_signature_id,omitempty" doc:"Destination signature ID"`
	WormholeType    string              `json:"wormhole_type,omitempty" doc:"Wormhole type code"`
	WormholeInfo    *WormholeStaticInfo `json:"wormhole_info,omitempty" doc:"Static wormhole information"`
	MassStatus      string              `json:"mass_status" doc:"Mass status"`
	TimeStatus      string              `json:"time_status" doc:"Time status"`
	MaxMass         int64               `json:"max_mass,omitempty" doc:"Maximum total mass"`
	JumpMass        int64               `json:"jump_mass,omitempty" doc:"Maximum ship mass"`
	RemainingMass   int64               `json:"remaining_mass,omitempty" doc:"Estimated remaining mass"`
	CreatedBy       string              `json:"created_by" doc:"Creator user ID"`
	CreatedByName   string              `json:"created_by_name" doc:"Creator name"`
	UpdatedBy       string              `json:"updated_by,omitempty" doc:"Last updater ID"`
	UpdatedByName   string              `json:"updated_by_name,omitempty" doc:"Last updater name"`
	SharingLevel    string              `json:"sharing_level" doc:"Visibility level"`
	ExpiresAt       *time.Time          `json:"expires_at,omitempty" doc:"Expiration time"`
	CreatedAt       time.Time           `json:"created_at" doc:"Creation time"`
	UpdatedAt       time.Time           `json:"updated_at" doc:"Last update time"`
}

// WormholeStaticInfo represents static wormhole type information
type WormholeStaticInfo struct {
	Code          string `json:"code" doc:"Wormhole type code"`
	LeadsTo       string `json:"leads_to" doc:"Destination class"`
	MaxMass       int64  `json:"max_mass" doc:"Total mass capacity"`
	JumpMass      int64  `json:"jump_mass" doc:"Maximum ship mass"`
	MassRegenRate int64  `json:"mass_regen_rate" doc:"Mass regeneration rate"`
	Lifetime      int    `json:"lifetime" doc:"Lifetime in hours"`
	Description   string `json:"description" doc:"Description"`
}

// NoteOutput represents a map note response
type NoteOutput struct {
	ID            string     `json:"id" doc:"Note database ID"`
	SystemID      int32      `json:"system_id" doc:"EVE System ID"`
	SystemName    string     `json:"system_name" doc:"System name"`
	Text          string     `json:"text" doc:"Note text"`
	Size          string     `json:"size" doc:"Note size"`
	Color         string     `json:"color,omitempty" doc:"Note color"`
	PosX          float32    `json:"pos_x,omitempty" doc:"X position"`
	PosY          float32    `json:"pos_y,omitempty" doc:"Y position"`
	CreatedBy     string     `json:"created_by" doc:"Creator user ID"`
	CreatedByName string     `json:"created_by_name" doc:"Creator name"`
	SharingLevel  string     `json:"sharing_level" doc:"Visibility level"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" doc:"Expiration time"`
	CreatedAt     time.Time  `json:"created_at" doc:"Creation time"`
	UpdatedAt     time.Time  `json:"updated_at" doc:"Last update time"`
}

// RouteOutput represents a calculated route response
type RouteOutput struct {
	FromSystemID      int32         `json:"from_system_id" doc:"Origin system ID"`
	FromSystemName    string        `json:"from_system_name" doc:"Origin system name"`
	ToSystemID        int32         `json:"to_system_id" doc:"Destination system ID"`
	ToSystemName      string        `json:"to_system_name" doc:"Destination system name"`
	RouteType         string        `json:"route_type" doc:"Route calculation type"`
	Route             []RouteSystem `json:"route" doc:"List of systems in route"`
	Jumps             int           `json:"jumps" doc:"Total number of jumps"`
	IncludesWH        bool          `json:"includes_wh" doc:"Route includes wormholes"`
	IncludesThera     bool          `json:"includes_thera" doc:"Route includes Thera"`
	SecurityBreakdown SecurityStats `json:"security_breakdown" doc:"Security status breakdown"`
}

// RouteSystem represents a system in a route
type RouteSystem struct {
	SystemID   int32   `json:"system_id" doc:"EVE System ID"`
	SystemName string  `json:"system_name" doc:"System name"`
	Security   float32 `json:"security" doc:"Security status"`
	RegionID   int32   `json:"region_id" doc:"Region ID"`
	RegionName string  `json:"region_name" doc:"Region name"`
	IsWormhole bool    `json:"is_wormhole,omitempty" doc:"Is a wormhole connection"`
}

// SecurityStats represents security breakdown of a route
type SecurityStats struct {
	HighSec  int `json:"high_sec" doc:"Number of high-sec systems"`
	LowSec   int `json:"low_sec" doc:"Number of low-sec systems"`
	NullSec  int `json:"null_sec" doc:"Number of null-sec systems"`
	Wormhole int `json:"wormhole" doc:"Number of wormhole systems"`
}

// SystemActivityOutput represents system activity data
type SystemActivityOutput struct {
	SystemID   int32     `json:"system_id" doc:"EVE System ID"`
	SystemName string    `json:"system_name" doc:"System name"`
	ShipKills  int       `json:"ship_kills" doc:"Ship kills in last hour"`
	NPCKills   int       `json:"npc_kills" doc:"NPC kills in last hour"`
	PodKills   int       `json:"pod_kills" doc:"Pod kills in last hour"`
	Jumps      int       `json:"jumps" doc:"Jumps in last hour"`
	UpdatedAt  time.Time `json:"updated_at" doc:"Last update time"`
}

// MapRegionOutput represents a region data response containing map elements
type MapRegionOutput struct {
	Elements []MapElement `json:"elements" doc:"Array of map elements (nodes and edges)"`
}

// MapElement represents either a node (system) or edge (connection) on the map
type MapElement struct {
	Group    string       `json:"group" doc:"Element type: nodes or edges"`
	Data     interface{}  `json:"data" doc:"Element data"`
	Classes  string       `json:"classes,omitempty" doc:"CSS classes for styling"`
	Position *MapPosition `json:"position,omitempty" doc:"Position on map (nodes only)"`
}

// MapNodeData represents system data in the map format
type MapNodeData struct {
	ID             int32         `json:"id" doc:"System ID"`
	Name           string        `json:"name" doc:"System name"`
	Label          string        `json:"label" doc:"System label"`
	RegionID       int32         `json:"region_id" doc:"Region ID"`
	RegionName     string        `json:"region_name" doc:"Region name"`
	SecStatus      float32       `json:"secstatus" doc:"Security status"`
	Regional       bool          `json:"regional,omitempty" doc:"Is regional system"`
	AllianceID     *int32        `json:"alliance_id,omitempty" doc:"Alliance ID"`
	FactionID      *int32        `json:"faction_id,omitempty" doc:"Faction ID"`
	SunPower       float32       `json:"sunPower,omitempty" doc:"Sun power for Equinox"`
	EquinoxPlanets []interface{} `json:"equinoxPlanets,omitempty" doc:"Equinox planets data"`
	Planets        []PlanetData  `json:"planets,omitempty" doc:"Planets in system"`
	Temperate      int           `json:"temperate,omitempty" doc:"Number of temperate planets"`
}

// MapEdgeData represents connection data in the map format
type MapEdgeData struct {
	ID     int32 `json:"id" doc:"Edge ID"`
	Source int32 `json:"source" doc:"Source system ID"`
	Target int32 `json:"target" doc:"Target system ID"`
}

// MapPosition represents position coordinates
type MapPosition struct {
	X float64 `json:"x" doc:"X coordinate"`
	Y float64 `json:"y" doc:"Y coordinate"`
}

// PlanetData represents planet information
type PlanetData struct {
	PlanetID int32 `json:"planetId" doc:"Planet ID"`
	TypeID   int32 `json:"typeId" doc:"Planet type ID"`
}

// BatchSignatureOutput represents the result of batch signature operations
type BatchSignatureOutput struct {
	Created []SignatureOutput `json:"created" doc:"Successfully created signatures"`
	Updated []SignatureOutput `json:"updated" doc:"Updated existing signatures"`
	Deleted []string          `json:"deleted" doc:"IDs of deleted signatures"`
	Errors  []BatchError      `json:"errors,omitempty" doc:"Errors encountered"`
}

// BatchWormholeOutput represents the result of batch wormhole operations
type BatchWormholeOutput struct {
	Created []WormholeOutput `json:"created" doc:"Successfully created wormholes"`
	Updated []WormholeOutput `json:"updated" doc:"Updated existing wormholes"`
	Deleted []string         `json:"deleted" doc:"IDs of deleted wormholes"`
	Errors  []BatchError     `json:"errors,omitempty" doc:"Errors encountered"`
}

// BatchError represents an error in batch operations
type BatchError struct {
	SignatureID string `json:"signature_id" doc:"Signature ID that failed"`
	Error       string `json:"error" doc:"Error message"`
}

// SearchSystemOutput represents system search results
type SearchSystemOutput struct {
	SystemID          int32   `json:"system_id" doc:"EVE System ID"`
	SystemName        string  `json:"system_name" doc:"System name"`
	RegionName        string  `json:"region_name" doc:"Region name"`
	ConstellationName string  `json:"constellation_name" doc:"Constellation name"`
	Security          float32 `json:"security" doc:"Security status"`
	MatchType         string  `json:"match_type" doc:"How the result matched (exact, starts_with, contains)"`
}

// MapStatusOutput represents the map module status response
type MapStatusOutput struct {
	Body MapStatusResponse `json:"body"`
}

// MapStatusResponse represents the actual map status data
type MapStatusResponse struct {
	Module  string `json:"module" doc:"Module name"`
	Status  string `json:"status" doc:"Module status"`
	Message string `json:"message,omitempty" doc:"Status message"`
	Stats   struct {
		Signatures   int `json:"signatures" doc:"Total signatures tracked"`
		Wormholes    int `json:"wormholes" doc:"Total wormhole connections"`
		Notes        int `json:"notes" doc:"Total map notes"`
		CachedRoutes int `json:"cached_routes" doc:"Number of cached routes"`
	} `json:"stats,omitempty" doc:"Module statistics"`
}

// Convert models to DTOs
func SignatureToOutput(sig *models.MapSignature, systemName string) SignatureOutput {
	return SignatureOutput{
		ID:            sig.ID.Hex(),
		SystemID:      sig.SystemID,
		SystemName:    systemName,
		SignatureID:   sig.SignatureID,
		Type:          sig.Type,
		Name:          sig.Name,
		Description:   sig.Description,
		Strength:      sig.Strength,
		CreatedBy:     sig.CreatedBy.Hex(),
		CreatedByName: sig.CreatedByName,
		UpdatedBy:     sig.UpdatedBy.Hex(),
		UpdatedByName: sig.UpdatedByName,
		SharingLevel:  sig.SharingLevel,
		ExpiresAt:     sig.ExpiresAt,
		CreatedAt:     sig.CreatedAt,
		UpdatedAt:     sig.UpdatedAt,
	}
}

func WormholeToOutput(wh *models.MapWormhole, fromSystemName, toSystemName string, staticInfo *models.WormholeStatic) WormholeOutput {
	output := WormholeOutput{
		ID:              wh.ID.Hex(),
		FromSystemID:    wh.FromSystemID,
		FromSystemName:  fromSystemName,
		ToSystemID:      wh.ToSystemID,
		ToSystemName:    toSystemName,
		FromSignatureID: wh.FromSignatureID,
		ToSignatureID:   wh.ToSignatureID,
		WormholeType:    wh.WormholeType,
		MassStatus:      wh.MassStatus,
		TimeStatus:      wh.TimeStatus,
		MaxMass:         wh.MaxMass,
		JumpMass:        wh.JumpMass,
		RemainingMass:   wh.RemainingMass,
		CreatedBy:       wh.CreatedBy.Hex(),
		CreatedByName:   wh.CreatedByName,
		UpdatedBy:       wh.UpdatedBy.Hex(),
		UpdatedByName:   wh.UpdatedByName,
		SharingLevel:    wh.SharingLevel,
		ExpiresAt:       wh.ExpiresAt,
		CreatedAt:       wh.CreatedAt,
		UpdatedAt:       wh.UpdatedAt,
	}

	if staticInfo != nil {
		output.WormholeInfo = &WormholeStaticInfo{
			Code:          staticInfo.ID,
			LeadsTo:       staticInfo.LeadsTo,
			MaxMass:       staticInfo.MaxMass,
			JumpMass:      staticInfo.JumpMass,
			MassRegenRate: staticInfo.MassRegenRate,
			Lifetime:      staticInfo.Lifetime,
			Description:   staticInfo.Description,
		}
	}

	return output
}

// Additional response wrappers for protected endpoints

// SignatureResponseOutput wraps a single signature for protected endpoint responses
type SignatureResponseOutput struct {
	Body SignatureOutput `json:"body"`
}

// SignatureListResponseOutput wraps a list of signatures for protected endpoint responses
type SignatureListResponseOutput struct {
	Body []SignatureOutput `json:"body"`
}

// DeleteSignatureResponseOutput wraps a signature deletion response
type DeleteSignatureResponseOutput struct {
	Body struct {
		Success bool   `json:"success" doc:"Operation success"`
		Message string `json:"message" doc:"Response message"`
	} `json:"body"`
}

// Wormhole response wrappers for protected endpoints

// WormholeResponseOutput wraps a single wormhole for protected endpoint responses
type WormholeResponseOutput struct {
	Body WormholeOutput `json:"body"`
}

// WormholeListResponseOutput wraps a list of wormholes for protected endpoint responses
type WormholeListResponseOutput struct {
	Body []WormholeOutput `json:"body"`
}

// DeleteWormholeResponseOutput wraps a wormhole deletion response
type DeleteWormholeResponseOutput struct {
	Body struct {
		Success bool   `json:"success" doc:"Operation success"`
		Message string `json:"message" doc:"Response message"`
	} `json:"body"`
}

func NoteToOutput(note *models.MapNote, systemName string) NoteOutput {
	return NoteOutput{
		ID:            note.ID.Hex(),
		SystemID:      note.SystemID,
		SystemName:    systemName,
		Text:          note.Text,
		Size:          note.Size,
		Color:         note.Color,
		PosX:          note.PosX,
		PosY:          note.PosY,
		CreatedBy:     note.CreatedBy.Hex(),
		CreatedByName: note.CreatedByName,
		SharingLevel:  note.SharingLevel,
		ExpiresAt:     note.ExpiresAt,
		CreatedAt:     note.CreatedAt,
		UpdatedAt:     note.UpdatedAt,
	}
}
