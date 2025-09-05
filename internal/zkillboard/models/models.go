package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ZKBMetadata represents ZKillboard-specific metadata for a killmail
type ZKBMetadata struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	KillmailID     int64              `bson:"killmail_id" json:"killmail_id"`
	LocationID     int64              `bson:"location_id" json:"location_id"`
	Hash           string             `bson:"hash" json:"hash"`
	FittedValue    float64            `bson:"fitted_value" json:"fitted_value"`
	DroppedValue   float64            `bson:"dropped_value" json:"dropped_value"`
	DestroyedValue float64            `bson:"destroyed_value" json:"destroyed_value"`
	TotalValue     float64            `bson:"total_value" json:"total_value"`
	Points         int                `bson:"points" json:"points"`
	NPC            bool               `bson:"npc" json:"npc"`
	Solo           bool               `bson:"solo" json:"solo"`
	Awox           bool               `bson:"awox" json:"awox"`
	Labels         []string           `bson:"labels" json:"labels"`
	Href           string             `bson:"href" json:"href"`
	ProcessedAt    time.Time          `bson:"processed_at" json:"processed_at"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// KillmailTimeseries represents aggregated killmail statistics over time
type KillmailTimeseries struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Period          string             `bson:"period" json:"period"` // "hour", "day", "month"
	Timestamp       time.Time          `bson:"timestamp" json:"timestamp"`
	SolarSystemID   int32              `bson:"solar_system_id,omitempty" json:"solar_system_id,omitempty"`
	ConstellationID int32              `bson:"constellation_id,omitempty" json:"constellation_id,omitempty"`
	RegionID        int32              `bson:"region_id,omitempty" json:"region_id,omitempty"`
	AllianceID      int32              `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	CorporationID   int32              `bson:"corporation_id,omitempty" json:"corporation_id,omitempty"`
	ShipTypeID      int32              `bson:"ship_type_id,omitempty" json:"ship_type_id,omitempty"`
	KillCount       int                `bson:"kill_count" json:"kill_count"`
	TotalValue      float64            `bson:"total_value" json:"total_value"`
	NPCKills        int                `bson:"npc_kills" json:"npc_kills"`
	SoloKills       int                `bson:"solo_kills" json:"solo_kills"`
	ShipTypes       map[int32]int      `bson:"ship_types" json:"ship_types"`
	TopVictims      []CharacterStats   `bson:"top_victims,omitempty" json:"top_victims,omitempty"`
	TopAttackers    []CharacterStats   `bson:"top_attackers,omitempty" json:"top_attackers,omitempty"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// CharacterStats represents character statistics in timeseries
type CharacterStats struct {
	CharacterID   int32   `bson:"character_id" json:"character_id"`
	CharacterName string  `bson:"character_name" json:"character_name"`
	Count         int     `bson:"count" json:"count"`
	Value         float64 `bson:"value" json:"value"`
}

// ConsumerState represents the state of the RedisQ consumer
type ConsumerState struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	QueueID        string             `bson:"queue_id" json:"queue_id"`
	State          string             `bson:"state" json:"state"` // stopped, running, throttled, draining
	LastPollTime   time.Time          `bson:"last_poll_time" json:"last_poll_time"`
	LastKillmailID int64              `bson:"last_killmail_id" json:"last_killmail_id"`
	TotalPolls     int64              `bson:"total_polls" json:"total_polls"`
	NullResponses  int64              `bson:"null_responses" json:"null_responses"`
	KillmailsFound int64              `bson:"killmails_found" json:"killmails_found"`
	HTTPErrors     int64              `bson:"http_errors" json:"http_errors"`
	ParseErrors    int64              `bson:"parse_errors" json:"parse_errors"`
	StoreErrors    int64              `bson:"store_errors" json:"store_errors"`
	RateLimitHits  int64              `bson:"rate_limit_hits" json:"rate_limit_hits"`
	CurrentTTW     int                `bson:"current_ttw" json:"current_ttw"`
	NullStreak     int                `bson:"null_streak" json:"null_streak"`
	StartedAt      time.Time          `bson:"started_at" json:"started_at"`
	StoppedAt      *time.Time         `bson:"stopped_at,omitempty" json:"stopped_at,omitempty"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}
