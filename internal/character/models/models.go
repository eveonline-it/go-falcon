package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Character represents a character document in the database
type Character struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID    int                `bson:"character_id" json:"character_id"`
	Name           string             `bson:"name" json:"name"`
	CorporationID  int                `bson:"corporation_id" json:"corporation_id"`
	AllianceID     int                `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	Birthday       time.Time          `bson:"birthday" json:"birthday"`
	SecurityStatus float64            `bson:"security_status" json:"security_status"`
	Description    string             `bson:"description,omitempty" json:"description,omitempty"`
	Gender         string             `bson:"gender" json:"gender"`
	RaceID         int                `bson:"race_id" json:"race_id"`
	BloodlineID    int                `bson:"bloodline_id" json:"bloodline_id"`
	AncestryID     int                `bson:"ancestry_id,omitempty" json:"ancestry_id,omitempty"`
	FactionID      int                `bson:"faction_id,omitempty" json:"faction_id,omitempty"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for characters
func (c *Character) CollectionName() string {
	return "characters"
}
