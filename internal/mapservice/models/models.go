package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MapSignature represents a cosmic signature in EVE Online
type MapSignature struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	SystemID      int32               `bson:"system_id" json:"system_id"`
	SignatureID   string              `bson:"signature_id" json:"signature_id"`
	Type          string              `bson:"type" json:"type"` // Combat, Data, Relic, Gas, Wormhole, etc.
	Name          string              `bson:"name,omitempty" json:"name,omitempty"`
	Description   string              `bson:"description,omitempty" json:"description,omitempty"`
	Strength      float32             `bson:"strength,omitempty" json:"strength,omitempty"`
	CreatedBy     primitive.ObjectID  `bson:"created_by" json:"created_by"`
	CreatedByName string              `bson:"created_by_name" json:"created_by_name"`
	UpdatedBy     primitive.ObjectID  `bson:"updated_by,omitempty" json:"updated_by,omitempty"`
	UpdatedByName string              `bson:"updated_by_name,omitempty" json:"updated_by_name,omitempty"`
	SharingLevel  string              `bson:"sharing_level" json:"sharing_level"` // private, corporation, alliance
	GroupID       *primitive.ObjectID `bson:"group_id,omitempty" json:"group_id,omitempty"`
	ExpiresAt     *time.Time          `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt     time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time           `bson:"updated_at" json:"updated_at"`
}

// MapWormhole represents a wormhole connection between systems
type MapWormhole struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	FromSystemID    int32               `bson:"from_system_id" json:"from_system_id"`
	ToSystemID      int32               `bson:"to_system_id" json:"to_system_id"`
	FromSignatureID string              `bson:"from_signature_id" json:"from_signature_id"`
	ToSignatureID   string              `bson:"to_signature_id,omitempty" json:"to_signature_id,omitempty"`
	WormholeType    string              `bson:"wormhole_type,omitempty" json:"wormhole_type,omitempty"` // K162, B274, etc.
	MassStatus      string              `bson:"mass_status" json:"mass_status"`                         // stable, destabilized, critical
	TimeStatus      string              `bson:"time_status" json:"time_status"`                         // stable, eol
	MaxMass         int64               `bson:"max_mass,omitempty" json:"max_mass,omitempty"`           // Maximum mass in kg
	MassRegenRate   int64               `bson:"mass_regen_rate,omitempty" json:"mass_regen_rate,omitempty"`
	RemainingMass   int64               `bson:"remaining_mass,omitempty" json:"remaining_mass,omitempty"`
	JumpMass        int64               `bson:"jump_mass,omitempty" json:"jump_mass,omitempty"` // Maximum ship mass
	CreatedBy       primitive.ObjectID  `bson:"created_by" json:"created_by"`
	CreatedByName   string              `bson:"created_by_name" json:"created_by_name"`
	UpdatedBy       primitive.ObjectID  `bson:"updated_by,omitempty" json:"updated_by,omitempty"`
	UpdatedByName   string              `bson:"updated_by_name,omitempty" json:"updated_by_name,omitempty"`
	SharingLevel    string              `bson:"sharing_level" json:"sharing_level"`
	GroupID         *primitive.ObjectID `bson:"group_id,omitempty" json:"group_id,omitempty"`
	ExpiresAt       *time.Time          `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt       time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time           `bson:"updated_at" json:"updated_at"`
}

// MapNote represents a user note on the map
type MapNote struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	SystemID      int32               `bson:"system_id" json:"system_id"`
	Text          string              `bson:"text" json:"text"`
	Size          string              `bson:"size" json:"size"` // small, medium, large
	Color         string              `bson:"color,omitempty" json:"color,omitempty"`
	PosX          float32             `bson:"pos_x,omitempty" json:"pos_x,omitempty"`
	PosY          float32             `bson:"pos_y,omitempty" json:"pos_y,omitempty"`
	CreatedBy     primitive.ObjectID  `bson:"created_by" json:"created_by"`
	CreatedByName string              `bson:"created_by_name" json:"created_by_name"`
	SharingLevel  string              `bson:"sharing_level" json:"sharing_level"`
	GroupID       *primitive.ObjectID `bson:"group_id,omitempty" json:"group_id,omitempty"`
	ExpiresAt     *time.Time          `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt     time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time           `bson:"updated_at" json:"updated_at"`
}

// MapRoute represents a calculated route between systems
type MapRoute struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FromSystemID  int32              `bson:"from_system_id" json:"from_system_id"`
	ToSystemID    int32              `bson:"to_system_id" json:"to_system_id"`
	RouteType     string             `bson:"route_type" json:"route_type"` // shortest, safest, avoid_null
	Route         []int32            `bson:"route" json:"route"`           // Array of system IDs
	Jumps         int                `bson:"jumps" json:"jumps"`
	IncludesWH    bool               `bson:"includes_wh" json:"includes_wh"`
	IncludesThera bool               `bson:"includes_thera" json:"includes_thera"`
	CachedAt      time.Time          `bson:"cached_at" json:"cached_at"`
	ExpiresAt     time.Time          `bson:"expires_at" json:"expires_at"`
}

// MapSystemActivity represents system activity data from ESI
type MapSystemActivity struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SystemID  int32              `bson:"system_id" json:"system_id"`
	ShipKills int                `bson:"ship_kills" json:"ship_kills"`
	NPCKills  int                `bson:"npc_kills" json:"npc_kills"`
	PodKills  int                `bson:"pod_kills" json:"pod_kills"`
	Jumps     int                `bson:"jumps" json:"jumps"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// WormholeStatic represents static wormhole information
type WormholeStatic struct {
	ID            string `bson:"_id" json:"id"`              // Wormhole type code (e.g., "B274")
	LeadsTo       string `bson:"leads_to" json:"leads_to"`   // Destination class (HS, LS, NS, C1-C6, etc.)
	MaxMass       int64  `bson:"max_mass" json:"max_mass"`   // Total mass in kg
	JumpMass      int64  `bson:"jump_mass" json:"jump_mass"` // Max ship mass in kg
	MassRegenRate int64  `bson:"mass_regen_rate" json:"mass_regen_rate"`
	Lifetime      int    `bson:"lifetime" json:"lifetime"` // Lifetime in hours
	Description   string `bson:"description" json:"description"`
}
