package dto

import (
	"encoding/json"
	"time"
)

// RedisQResponse represents the response from ZKillboard RedisQ
type RedisQResponse struct {
	Package *RedisQPackage `json:"package"`
}

// RedisQPackage represents a killmail package from RedisQ
type RedisQPackage struct {
	KillID   int64           `json:"killID"`
	Killmail json.RawMessage `json:"killmail"`
	ZKB      ZKBData         `json:"zkb"`
}

// ZKBData represents ZKillboard-specific metadata in the RedisQ response
type ZKBData struct {
	LocationID     int64   `json:"locationID"`
	Hash           string  `json:"hash"`
	FittedValue    float64 `json:"fittedValue"`
	DroppedValue   float64 `json:"droppedValue"`
	DestroyedValue float64 `json:"destroyedValue"`
	TotalValue     float64 `json:"totalValue"`
	Points         int     `json:"points"`
	NPC            bool    `json:"npc"`
	Solo           bool    `json:"solo"`
	Awox           bool    `json:"awox"`
	Href           string  `json:"href"`
}

// ESIKillmail represents the killmail data from ESI
type ESIKillmail struct {
	KillmailID    int64         `json:"killmail_id"`
	KillmailTime  time.Time     `json:"killmail_time"`
	SolarSystemID int32         `json:"solar_system_id"`
	Victim        ESIVictim     `json:"victim"`
	Attackers     []ESIAttacker `json:"attackers"`
}

// ESIVictim represents victim information in ESI killmail
type ESIVictim struct {
	CharacterID   *int32    `json:"character_id,omitempty"`
	CorporationID int32     `json:"corporation_id"`
	AllianceID    *int32    `json:"alliance_id,omitempty"`
	ShipTypeID    int32     `json:"ship_type_id"`
	DamageTaken   int       `json:"damage_taken"`
	Position      *Position `json:"position,omitempty"`
	Items         []ESIItem `json:"items,omitempty"`
}

// ESIAttacker represents attacker information in ESI killmail
type ESIAttacker struct {
	CharacterID    *int32  `json:"character_id,omitempty"`
	CorporationID  *int32  `json:"corporation_id,omitempty"`
	AllianceID     *int32  `json:"alliance_id,omitempty"`
	ShipTypeID     *int32  `json:"ship_type_id,omitempty"`
	WeaponTypeID   *int32  `json:"weapon_type_id,omitempty"`
	DamageDone     int     `json:"damage_done"`
	FinalBlow      bool    `json:"final_blow"`
	SecurityStatus float32 `json:"security_status"`
}

// ESIItem represents an item in the killmail
type ESIItem struct {
	ItemTypeID        int32     `json:"item_type_id"`
	SingletonID       int64     `json:"singleton"`
	Flag              int32     `json:"flag"`
	QuantityDropped   *int64    `json:"quantity_dropped,omitempty"`
	QuantityDestroyed *int64    `json:"quantity_destroyed,omitempty"`
	Items             []ESIItem `json:"items,omitempty"`
}

// Position represents a position in space
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
