package dto

import (
	"time"

	"go-falcon/internal/killmails/models"
)

// KillmailResponse represents a complete killmail response
type KillmailResponse struct {
	KillmailID   int64     `json:"killmail_id" doc:"Unique killmail identifier"`
	KillmailHash string    `json:"killmail_hash" doc:"Killmail hash for verification"`
	KillmailTime time.Time `json:"killmail_time" doc:"Time when the kill occurred"`

	// Location
	SolarSystemID int64  `json:"solar_system_id" doc:"Solar system where the kill occurred"`
	MoonID        *int64 `json:"moon_id,omitempty" doc:"Moon ID if kill occurred at a moon"`
	WarID         *int64 `json:"war_id,omitempty" doc:"War ID if kill was part of a declared war"`

	// Participants
	Victim    VictimResponse     `json:"victim" doc:"Victim information"`
	Attackers []AttackerResponse `json:"attackers" doc:"List of attackers involved"`
}

// VictimResponse represents the victim in a killmail
type VictimResponse struct {
	CharacterID   *int64            `json:"character_id,omitempty" doc:"Character ID of the victim (if applicable)"`
	CorporationID *int64            `json:"corporation_id,omitempty" doc:"Corporation ID of the victim"`
	AllianceID    *int64            `json:"alliance_id,omitempty" doc:"Alliance ID of the victim (if applicable)"`
	FactionID     *int64            `json:"faction_id,omitempty" doc:"Faction ID of the victim (if applicable)"`
	ShipTypeID    int64             `json:"ship_type_id" doc:"Ship type ID that was destroyed"`
	DamageTaken   int64             `json:"damage_taken" doc:"Total damage taken by the victim"`
	Position      *PositionResponse `json:"position,omitempty" doc:"3D coordinates of the victim"`
	Items         []ItemResponse    `json:"items,omitempty" doc:"Items that were on the victim's ship"`
}

// AttackerResponse represents an attacker in a killmail
type AttackerResponse struct {
	CharacterID    *int64  `json:"character_id,omitempty" doc:"Character ID of the attacker (if applicable)"`
	CorporationID  *int64  `json:"corporation_id,omitempty" doc:"Corporation ID of the attacker"`
	AllianceID     *int64  `json:"alliance_id,omitempty" doc:"Alliance ID of the attacker (if applicable)"`
	FactionID      *int64  `json:"faction_id,omitempty" doc:"Faction ID of the attacker (if applicable)"`
	ShipTypeID     *int64  `json:"ship_type_id,omitempty" doc:"Ship type ID used by the attacker"`
	WeaponTypeID   *int64  `json:"weapon_type_id,omitempty" doc:"Weapon type ID used for the kill"`
	DamageDone     int64   `json:"damage_done" doc:"Damage dealt by this attacker"`
	FinalBlow      bool    `json:"final_blow" doc:"Whether this attacker achieved the final blow"`
	SecurityStatus float64 `json:"security_status" doc:"Security status of the attacker"`
}

// PositionResponse represents 3D coordinates in space
type PositionResponse struct {
	X float64 `json:"x" doc:"X coordinate"`
	Y float64 `json:"y" doc:"Y coordinate"`
	Z float64 `json:"z" doc:"Z coordinate"`
}

// ItemResponse represents an item from the victim's ship
type ItemResponse struct {
	ItemTypeID        int64          `json:"item_type_id" doc:"Type ID of the item"`
	Flag              int64          `json:"flag" doc:"Flag indicating the location of the item on the ship"`
	Singleton         int64          `json:"singleton" doc:"Singleton value for the item"`
	QuantityDestroyed *int64         `json:"quantity_destroyed,omitempty" doc:"Quantity of this item that was destroyed"`
	QuantityDropped   *int64         `json:"quantity_dropped,omitempty" doc:"Quantity of this item that was dropped"`
	Items             []ItemResponse `json:"items,omitempty" doc:"Nested items (for containers)"`
}

// KillmailRefResponse represents a reference to a killmail (used in lists)
type KillmailRefResponse struct {
	KillmailID   int64  `json:"killmail_id" doc:"Unique killmail identifier"`
	KillmailHash string `json:"killmail_hash" doc:"Killmail hash for verification"`
}

// KillmailListResponse represents a list of killmail references
type KillmailListResponse struct {
	Killmails []KillmailRefResponse `json:"killmails" doc:"List of killmail references"`
	Count     int                   `json:"count" doc:"Number of killmails returned"`
	Total     *int64                `json:"total,omitempty" doc:"Total number of killmails available (if known)"`
}

// KillmailStatsResponse represents statistics about killmails
type KillmailStatsResponse struct {
	TotalKillmails int64  `json:"total_killmails" doc:"Total number of killmails stored"`
	Collection     string `json:"collection" doc:"Database collection name"`
}

// StatusOutput represents the module status response (Huma v2 wrapper)
type StatusOutput struct {
	Body ModuleStatusResponse
}

// ModuleStatusResponse represents the health status of the killmails module
type ModuleStatusResponse struct {
	Module  string `json:"module" doc:"Module name"`
	Status  string `json:"status" doc:"Module status (healthy/unhealthy)"`
	Message string `json:"message,omitempty" doc:"Additional status message"`
}

// KillmailOutput wraps KillmailResponse for Huma v2
type KillmailOutput struct {
	Body KillmailResponse
}

// KillmailListOutput wraps KillmailListResponse for Huma v2
type KillmailListOutput struct {
	Body KillmailListResponse
}

// KillmailStatsOutput wraps KillmailStatsResponse for Huma v2
type KillmailStatsOutput struct {
	Body KillmailStatsResponse
}

// Helper functions to convert models to responses

// ConvertKillmailToResponse converts a killmail model to response DTO
func ConvertKillmailToResponse(killmail *models.Killmail) *KillmailOutput {
	if killmail == nil {
		return nil
	}

	response := KillmailResponse{
		KillmailID:    killmail.KillmailID,
		KillmailHash:  killmail.KillmailHash,
		KillmailTime:  killmail.KillmailTime,
		SolarSystemID: killmail.SolarSystemID,
		MoonID:        killmail.MoonID,
		WarID:         killmail.WarID,
		Victim:        ConvertVictimToResponse(killmail.Victim),
		Attackers:     ConvertAttackersToResponse(killmail.Attackers),
	}

	return &KillmailOutput{Body: response}
}

// ConvertVictimToResponse converts a victim model to response DTO
func ConvertVictimToResponse(victim models.Victim) VictimResponse {
	response := VictimResponse{
		CharacterID:   victim.CharacterID,
		CorporationID: victim.CorporationID,
		AllianceID:    victim.AllianceID,
		FactionID:     victim.FactionID,
		ShipTypeID:    victim.ShipTypeID,
		DamageTaken:   victim.DamageTaken,
	}

	if victim.Position != nil {
		response.Position = &PositionResponse{
			X: victim.Position.X,
			Y: victim.Position.Y,
			Z: victim.Position.Z,
		}
	}

	if victim.Items != nil {
		response.Items = ConvertItemsToResponse(victim.Items)
	}

	return response
}

// ConvertAttackersToResponse converts attackers models to response DTOs
func ConvertAttackersToResponse(attackers []models.Attacker) []AttackerResponse {
	if attackers == nil {
		return nil
	}

	responses := make([]AttackerResponse, len(attackers))
	for i, attacker := range attackers {
		responses[i] = AttackerResponse{
			CharacterID:    attacker.CharacterID,
			CorporationID:  attacker.CorporationID,
			AllianceID:     attacker.AllianceID,
			FactionID:      attacker.FactionID,
			ShipTypeID:     attacker.ShipTypeID,
			WeaponTypeID:   attacker.WeaponTypeID,
			DamageDone:     attacker.DamageDone,
			FinalBlow:      attacker.FinalBlow,
			SecurityStatus: attacker.SecurityStatus,
		}
	}

	return responses
}

// ConvertItemsToResponse converts items models to response DTOs
func ConvertItemsToResponse(items []models.Item) []ItemResponse {
	if items == nil {
		return nil
	}

	responses := make([]ItemResponse, len(items))
	for i, item := range items {
		responses[i] = ItemResponse{
			ItemTypeID:        item.ItemTypeID,
			Flag:              item.Flag,
			Singleton:         item.Singleton,
			QuantityDestroyed: item.QuantityDestroyed,
			QuantityDropped:   item.QuantityDropped,
		}

		if item.Items != nil {
			responses[i].Items = ConvertItemsToResponse(item.Items)
		}
	}

	return responses
}

// ConvertKillmailRefsToResponse converts killmail references to response DTOs
func ConvertKillmailRefsToResponse(refs []models.KillmailRef) []KillmailRefResponse {
	if refs == nil {
		return nil
	}

	responses := make([]KillmailRefResponse, len(refs))
	for i, ref := range refs {
		responses[i] = KillmailRefResponse{
			KillmailID:   ref.KillmailID,
			KillmailHash: ref.KillmailHash,
		}
	}

	return responses
}

// ConvertKillmailsToList converts killmails to a list response
func ConvertKillmailsToList(killmails []models.Killmail, total *int64) *KillmailListOutput {
	refs := make([]KillmailRefResponse, len(killmails))
	for i, km := range killmails {
		refs[i] = KillmailRefResponse{
			KillmailID:   km.KillmailID,
			KillmailHash: km.KillmailHash,
		}
	}

	response := KillmailListResponse{
		Killmails: refs,
		Count:     len(refs),
		Total:     total,
	}

	return &KillmailListOutput{Body: response}
}
