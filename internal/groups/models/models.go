package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GroupType represents the type of group
type GroupType string

const (
	GroupTypeSystem      GroupType = "system"      // System groups (super_admin, authenticated, guest)
	GroupTypeCorporation GroupType = "corporation" // EVE Corporation groups
	GroupTypeAlliance    GroupType = "alliance"    // EVE Alliance groups
	GroupTypeCustom      GroupType = "custom"      // User-created custom groups
)

// Group represents a group in the system
type Group struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Description  string             `bson:"description,omitempty" json:"description"`
	Type         GroupType          `bson:"type" json:"type"`
	SystemName   *string            `bson:"system_name,omitempty" json:"system_name"` // For system groups: "super_admin", "authenticated", "guest"
	EVEEntityID  *int64             `bson:"eve_entity_id,omitempty" json:"eve_entity_id"` // Corporation/Alliance ID
	IsActive     bool               `bson:"is_active" json:"is_active"`
	CreatedBy    *int64             `bson:"created_by,omitempty" json:"created_by"` // Character ID
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// GroupMembership represents a character's membership in a group
type GroupMembership struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GroupID     primitive.ObjectID `bson:"group_id" json:"group_id"`
	CharacterID int64              `bson:"character_id" json:"character_id"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	AddedBy     *int64             `bson:"added_by,omitempty" json:"added_by"` // Character ID who added this membership
	AddedAt     time.Time          `bson:"added_at" json:"added_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// SystemGroups contains the predefined system group names
var SystemGroups = map[string]string{
	"super_admin":   "Super Administrator",
	"authenticated": "Authenticated Users",
	"guest":         "Guest Users",
}

// Collection names
const (
	GroupsCollection      = "groups"
	MembershipsCollection = "group_memberships"
)