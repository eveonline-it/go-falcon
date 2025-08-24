package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Corporation represents a corporation entity stored in the database
type Corporation struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CorporationID  int                `bson:"corporation_id" json:"corporation_id"`
	Name           string             `bson:"name" json:"name"`
	Ticker         string             `bson:"ticker" json:"ticker"`
	Description    string             `bson:"description" json:"description"`
	URL            *string            `bson:"url,omitempty" json:"url,omitempty"`
	AllianceID     *int               `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	CEOCharacterID int                `bson:"ceo_character_id" json:"ceo_character_id"`
	CreatorID      int                `bson:"creator_id" json:"creator_id"`
	DateFounded    time.Time          `bson:"date_founded" json:"date_founded"`
	FactionID      *int               `bson:"faction_id,omitempty" json:"faction_id,omitempty"`
	HomeStationID  *int               `bson:"home_station_id,omitempty" json:"home_station_id,omitempty"`
	MemberCount    int                `bson:"member_count" json:"member_count"`
	Shares         *int64             `bson:"shares,omitempty" json:"shares,omitempty"`
	TaxRate        float64            `bson:"tax_rate" json:"tax_rate"`
	WarEligible    *bool              `bson:"war_eligible,omitempty" json:"war_eligible,omitempty"`

	// Metadata
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Constants for collection names
const (
	CorporationCollection = "corporations"
)
