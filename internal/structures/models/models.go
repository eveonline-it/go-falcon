package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Structure represents a station or player structure in EVE Online
type Structure struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StructureID       int64              `bson:"structure_id" json:"structure_id"`
	CharacterID       int32              `bson:"character_id,omitempty" json:"character_id,omitempty"` // Character with access (for player structures)
	Name              string             `bson:"name" json:"name"`
	OwnerID           int32              `bson:"owner_id" json:"owner_id"`
	Position          *Position          `bson:"position,omitempty" json:"position,omitempty"`
	SolarSystemID     int32              `bson:"solar_system_id" json:"solar_system_id"`
	SolarSystemName   string             `bson:"solar_system_name,omitempty" json:"solar_system_name,omitempty"`
	RegionID          int32              `bson:"region_id,omitempty" json:"region_id,omitempty"`
	RegionName        string             `bson:"region_name,omitempty" json:"region_name,omitempty"`
	ConstellationID   int32              `bson:"constellation_id,omitempty" json:"constellation_id,omitempty"`
	ConstellationName string             `bson:"constellation_name,omitempty" json:"constellation_name,omitempty"`
	TypeID            int32              `bson:"type_id" json:"type_id"`
	TypeName          string             `bson:"type_name,omitempty" json:"type_name,omitempty"`
	IsNPCStation      bool               `bson:"is_npc_station" json:"is_npc_station"`
	Services          []string           `bson:"services,omitempty" json:"services,omitempty"`
	StateTimerStart   *time.Time         `bson:"state_timer_start,omitempty" json:"state_timer_start,omitempty"`
	StateTimerEnd     *time.Time         `bson:"state_timer_end,omitempty" json:"state_timer_end,omitempty"`
	FuelExpires       *time.Time         `bson:"fuel_expires,omitempty" json:"fuel_expires,omitempty"`
	UnanchorsAt       *time.Time         `bson:"unanchors_at,omitempty" json:"unanchors_at,omitempty"`
	State             string             `bson:"state,omitempty" json:"state,omitempty"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

// Position represents 3D coordinates in space
type Position struct {
	X float64 `bson:"x" json:"x"`
	Y float64 `bson:"y" json:"y"`
	Z float64 `bson:"z" json:"z"`
}

// StructureAccess tracks which characters have access to which structures
type StructureAccess struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StructureID int64              `bson:"structure_id" json:"structure_id"`
	CharacterID int32              `bson:"character_id" json:"character_id"`
	HasAccess   bool               `bson:"has_access" json:"has_access"`
	LastChecked time.Time          `bson:"last_checked" json:"last_checked"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionNames
const (
	StructuresCollection      = "structures"
	StructureAccessCollection = "structure_access"
)
