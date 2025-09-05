package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	KillmailsCollection          = "killmails"
	KillmailsCharStatsCollection = "killmails_char_stats"
)

type Killmail struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	KillmailID   int64              `bson:"killmail_id" json:"killmail_id"`
	KillmailHash string             `bson:"killmail_hash" json:"killmail_hash"`
	KillmailTime time.Time          `bson:"killmail_time" json:"killmail_time"`

	// Location
	SolarSystemID int64  `bson:"solar_system_id" json:"solar_system_id"`
	MoonID        *int64 `bson:"moon_id,omitempty" json:"moon_id,omitempty"`
	WarID         *int64 `bson:"war_id,omitempty" json:"war_id,omitempty"`

	// Victim
	Victim Victim `bson:"victim" json:"victim"`

	// Attackers
	Attackers []Attacker `bson:"attackers" json:"attackers"`
}

type Victim struct {
	CharacterID   *int64    `bson:"character_id,omitempty" json:"character_id,omitempty"`
	CorporationID *int64    `bson:"corporation_id,omitempty" json:"corporation_id,omitempty"`
	AllianceID    *int64    `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	FactionID     *int64    `bson:"faction_id,omitempty" json:"faction_id,omitempty"`
	ShipTypeID    int64     `bson:"ship_type_id" json:"ship_type_id"`
	DamageTaken   int64     `bson:"damage_taken" json:"damage_taken"`
	Position      *Position `bson:"position,omitempty" json:"position,omitempty"`
	Items         []Item    `bson:"items,omitempty" json:"items,omitempty"`
}

type Attacker struct {
	CharacterID    *int64  `bson:"character_id,omitempty" json:"character_id,omitempty"`
	CorporationID  *int64  `bson:"corporation_id,omitempty" json:"corporation_id,omitempty"`
	AllianceID     *int64  `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	FactionID      *int64  `bson:"faction_id,omitempty" json:"faction_id,omitempty"`
	ShipTypeID     *int64  `bson:"ship_type_id,omitempty" json:"ship_type_id,omitempty"`
	WeaponTypeID   *int64  `bson:"weapon_type_id,omitempty" json:"weapon_type_id,omitempty"`
	DamageDone     int64   `bson:"damage_done" json:"damage_done"`
	FinalBlow      bool    `bson:"final_blow" json:"final_blow"`
	SecurityStatus float64 `bson:"security_status" json:"security_status"`
}

type Position struct {
	X float64 `bson:"x" json:"x"`
	Y float64 `bson:"y" json:"y"`
	Z float64 `bson:"z" json:"z"`
}

type Item struct {
	ItemTypeID        int64  `bson:"item_type_id" json:"item_type_id"`
	Flag              int64  `bson:"flag" json:"flag"`
	Singleton         int64  `bson:"singleton" json:"singleton"`
	QuantityDestroyed *int64 `bson:"quantity_destroyed,omitempty" json:"quantity_destroyed,omitempty"`
	QuantityDropped   *int64 `bson:"quantity_dropped,omitempty" json:"quantity_dropped,omitempty"`
	Items             []Item `bson:"items,omitempty" json:"items,omitempty"` // Nested items (cargo containers, etc)
}

// KillmailRef represents a reference to a killmail (used in recent killmails endpoints)
type KillmailRef struct {
	KillmailID   int64  `bson:"killmail_id" json:"killmail_id"`
	KillmailHash string `bson:"killmail_hash" json:"killmail_hash"`
}

// CharacterKillmailStats represents character statistics for killmails
type CharacterKillmailStats struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID int32              `bson:"character_id" json:"character_id"`

	// Notable ships used per category - maps category name to ship type ID
	// Example: { "interdictor": 22452, "forcerecon": 11965 }
	NotableShips map[string]int64 `bson:"notable_ships" json:"notable_ships"`

	// Tracking metadata
	LastUpdated time.Time `bson:"last_updated" json:"last_updated"`
}
