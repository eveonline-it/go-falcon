package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Alliance represents an alliance entity stored in the database
type Alliance struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AllianceID        int                `bson:"alliance_id" json:"alliance_id"`
	Name              string             `bson:"name" json:"name"`
	Ticker            string             `bson:"ticker" json:"ticker"`
	DateFounded       time.Time          `bson:"date_founded" json:"date_founded"`
	CreatorCorporationID int            `bson:"creator_corporation_id" json:"creator_corporation_id"`
	CreatorCharacterID int              `bson:"creator_character_id" json:"creator_character_id"`
	ExecutorCorporationID *int           `bson:"executor_corporation_id,omitempty" json:"executor_corporation_id,omitempty"`
	FactionID         *int               `bson:"faction_id,omitempty" json:"faction_id,omitempty"`
	
	// Metadata
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Constants for collection names
const (
	AllianceCollection = "alliances"
)