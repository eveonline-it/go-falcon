package dto

// GetKillmailInput represents the input for fetching a specific killmail
type GetKillmailInput struct {
	KillmailID int64  `path:"killmail_id" validate:"required" minimum:"1" doc:"EVE Online killmail ID"`
	Hash       string `path:"hash" validate:"required" minLength:"40" maxLength:"40" doc:"Killmail hash (40 character string)"`
}

// GetCharacterRecentKillmailsInput represents input for fetching character's recent killmails
type GetCharacterRecentKillmailsInput struct {
	CharacterID int `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Limit       int `query:"limit" validate:"min:1,max:200" default:"50" doc:"Maximum number of killmails to return (1-200, default 50)"`
}

// GetCorporationRecentKillmailsInput represents input for fetching corporation's recent killmails
type GetCorporationRecentKillmailsInput struct {
	CorporationID int `path:"corporation_id" validate:"required" minimum:"98000000" maximum:"2147483647" doc:"EVE Online corporation ID"`
	Limit         int `query:"limit" validate:"min:1,max:200" default:"50" doc:"Maximum number of killmails to return (1-200, default 50)"`
}

// ImportKillmailBody represents the request body for importing a killmail
type ImportKillmailBody struct {
	KillmailID int64  `json:"killmail_id" validate:"required" minimum:"1" doc:"EVE Online killmail ID"`
	Hash       string `json:"hash" validate:"required" minLength:"40" maxLength:"40" doc:"Killmail hash (40 character string)"`
}

// ImportKillmailInput represents input for importing a killmail by ID and hash
type ImportKillmailInput struct {
	Body ImportKillmailBody `doc:"Killmail import request body"`
}

// GetRecentKillmailsInput represents input for fetching recent killmails from database
type GetRecentKillmailsInput struct {
	CharacterID   int64  `query:"character_id" validate:"omitempty,min:90000000" doc:"Filter by character ID (optional)"`
	CorporationID int64  `query:"corporation_id" validate:"omitempty,min:98000000" doc:"Filter by corporation ID (optional)"`
	AllianceID    int64  `query:"alliance_id" validate:"omitempty,min:99000000" doc:"Filter by alliance ID (optional)"`
	SystemID      int64  `query:"system_id" validate:"omitempty,min:30000000" doc:"Filter by solar system ID (optional)"`
	Since         string `query:"since" validate:"omitempty" doc:"Filter killmails since this timestamp (RFC3339 format, optional)"`
	Limit         int    `query:"limit" validate:"min:1,max:100" default:"20" doc:"Maximum number of killmails to return (1-100, default 20)"`
}

// Character Stats Input DTOs

// GetCharacterStatsInput represents input for fetching character killmail stats
type GetCharacterStatsInput struct {
	CharacterID int32 `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}

// GetCharacterLastShipByCategoryInput represents input for fetching character's last ship by category
type GetCharacterLastShipByCategoryInput struct {
	CharacterID int32  `path:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
	Category    string `path:"category" validate:"required" doc:"Ship category (interdictor, forcerecon, strategic, hic, monitor, blackops, marauders, fax, dread, carrier, super, titan, lancer)"`
}

// GetCharactersByShipCategoryInput represents input for fetching characters by ship category
type GetCharactersByShipCategoryInput struct {
	Category string `path:"category" validate:"required" doc:"Ship category (interdictor, forcerecon, strategic, hic, monitor, blackops, marauders, fax, dread, carrier, super, titan, lancer)"`
	Limit    int    `query:"limit" validate:"min:1,max:100" default:"50" doc:"Maximum number of characters to return (1-100, default 50)"`
}

// GetCharactersByShipTypeInput represents input for fetching characters by ship type
type GetCharactersByShipTypeInput struct {
	ShipTypeID int64 `path:"ship_type_id" validate:"required" minimum:"1" doc:"EVE Online ship type ID"`
	Limit      int   `query:"limit" validate:"min:1,max:100" default:"50" doc:"Maximum number of characters to return (1-100, default 50)"`
}

// GetRecentCharacterActivityInput represents input for fetching recent character activity
type GetRecentCharacterActivityInput struct {
	Hours int `query:"hours" validate:"min:1,max:720" default:"24" doc:"Number of hours to look back for activity (1-720, default 24)"`
	Limit int `query:"limit" validate:"min:1,max:100" default:"50" doc:"Maximum number of characters to return (1-100, default 50)"`
}
