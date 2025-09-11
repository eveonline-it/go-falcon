package dto

import "time"

// CharacterProfile represents a character profile data
type CharacterProfile struct {
	CharacterID    int       `json:"character_id" doc:"EVE Online character ID"`
	Name           string    `json:"name" doc:"Character name"`
	CorporationID  int       `json:"corporation_id" doc:"Corporation ID"`
	AllianceID     int       `json:"alliance_id,omitempty" doc:"Alliance ID"`
	Birthday       time.Time `json:"birthday" doc:"Character birthday"`
	SecurityStatus float64   `json:"security_status" doc:"Security status"`
	Description    string    `json:"description,omitempty" doc:"Character description"`
	Gender         string    `json:"gender" doc:"Character gender"`
	RaceID         int       `json:"race_id" doc:"Race ID"`
	BloodlineID    int       `json:"bloodline_id" doc:"Bloodline ID"`
	AncestryID     int       `json:"ancestry_id,omitempty" doc:"Ancestry ID"`
	FactionID      int       `json:"faction_id,omitempty" doc:"Faction ID"`
	CreatedAt      time.Time `json:"created_at" doc:"Profile created timestamp"`
	UpdatedAt      time.Time `json:"updated_at" doc:"Profile updated timestamp"`
}

// CharacterProfileOutput represents a character profile response (Huma wrapper)
type CharacterProfileOutput struct {
	Body CharacterProfile `json:"body"`
}

// SearchCharactersResult represents search results for characters
type SearchCharactersResult struct {
	Characters []CharacterProfile `json:"characters" doc:"List of matching characters"`
	Count      int                `json:"count" doc:"Number of characters found"`
}

// SearchCharactersByNameOutput represents the search response (Huma wrapper)
type SearchCharactersByNameOutput struct {
	Body SearchCharactersResult `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body CharacterStatusResponse `json:"body"`
}

// CharacterStatusResponse represents the actual status response data
type CharacterStatusResponse struct {
	Module       string                     `json:"module" description:"Module name"`
	Status       string                     `json:"status" enum:"healthy,degraded,unhealthy" description:"Module health status"`
	Message      string                     `json:"message,omitempty" description:"Optional status message or error details"`
	Dependencies *CharacterDependencyStatus `json:"dependencies,omitempty" description:"Status of module dependencies"`
	Metrics      *CharacterMetrics          `json:"metrics,omitempty" description:"Performance and operational metrics"`
	LastChecked  string                     `json:"last_checked" description:"Timestamp of last health check"`
}

// CharacterDependencyStatus represents the status of character module dependencies
type CharacterDependencyStatus struct {
	Database        string `json:"database" description:"MongoDB connection status"`
	DatabaseLatency string `json:"database_latency,omitempty" description:"Database response time"`
	EVEOnlineESI    string `json:"eve_online_esi" description:"EVE Online ESI availability"`
	ESILatency      string `json:"esi_latency,omitempty" description:"ESI response time"`
	ESIErrorLimits  string `json:"esi_error_limits,omitempty" description:"ESI error limit status"`
}

// CharacterMetrics represents performance metrics for the character module
type CharacterMetrics struct {
	TotalCharacters       int     `json:"total_characters" description:"Total characters in database"`
	RecentlyUpdated       int     `json:"recently_updated" description:"Characters updated in last 24 hours"`
	AffiliationUpdates    int     `json:"affiliation_updates_24h" description:"Affiliation updates in last 24 hours"`
	ESIRequests           int     `json:"esi_requests_1h" description:"ESI requests in last hour"`
	CacheHitRate          float64 `json:"cache_hit_rate" description:"Database cache hit rate percentage"`
	AverageResponseTime   string  `json:"average_response_time" description:"Average API response time"`
	MemoryUsage           float64 `json:"memory_usage_mb" description:"Memory usage in MB"`
	LastAffiliationUpdate string  `json:"last_affiliation_update,omitempty" description:"Last background affiliation update"`
}

// CharacterAttributes represents character attribute data
type CharacterAttributes struct {
	Charisma                 int        `json:"charisma" doc:"Charisma attribute value"`
	Intelligence             int        `json:"intelligence" doc:"Intelligence attribute value"`
	Memory                   int        `json:"memory" doc:"Memory attribute value"`
	Perception               int        `json:"perception" doc:"Perception attribute value"`
	Willpower                int        `json:"willpower" doc:"Willpower attribute value"`
	AccruedRemapCooldownDate *time.Time `json:"accrued_remap_cooldown_date,omitempty" doc:"Date when remap cooldown ends"`
	BonusRemaps              *int       `json:"bonus_remaps,omitempty" doc:"Number of bonus remaps available"`
	LastRemapDate            *time.Time `json:"last_remap_date,omitempty" doc:"Date of last attribute remap"`
}

// CharacterAttributesOutput represents the character attributes response (Huma wrapper)
type CharacterAttributesOutput struct {
	Body CharacterAttributes `json:"body"`
}

// SkillQueueItem represents a single skill in the character's skill queue
type SkillQueueItem struct {
	SkillID         int        `json:"skill_id" doc:"Skill type ID"`
	FinishedLevel   int        `json:"finished_level" doc:"Level this skill will complete to"`
	QueuePosition   int        `json:"queue_position" doc:"Position in the skill queue"`
	StartDate       *time.Time `json:"start_date,omitempty" doc:"Start date of training"`
	FinishDate      *time.Time `json:"finish_date,omitempty" doc:"Completion date of training"`
	TrainingStartSP *int       `json:"training_start_sp,omitempty" doc:"Skill points at training start"`
	LevelEndSP      *int       `json:"level_end_sp,omitempty" doc:"Skill points at level completion"`
	LevelStartSP    *int       `json:"level_start_sp,omitempty" doc:"Skill points at level start"`
}

// CharacterSkillQueue represents the character's complete skill queue
type CharacterSkillQueue struct {
	CharacterID int              `json:"character_id" doc:"EVE Online character ID"`
	Skills      []SkillQueueItem `json:"skills" doc:"List of skills in the queue"`
	UpdatedAt   time.Time        `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterSkillQueueOutput represents the skill queue response (Huma wrapper)
type CharacterSkillQueueOutput struct {
	Body CharacterSkillQueue `json:"body"`
}

// Skill represents a single trained skill
type Skill struct {
	SkillID            int `json:"skill_id" doc:"Skill type ID"`
	SkillpointsInSkill int `json:"skillpoints_in_skill" doc:"Total skill points in this skill"`
	TrainedSkillLevel  int `json:"trained_skill_level" doc:"Trained skill level"`
	ActiveSkillLevel   int `json:"active_skill_level" doc:"Active skill level"`
}

// CharacterSkills represents the character's complete skills
type CharacterSkills struct {
	CharacterID   int       `json:"character_id" doc:"EVE Online character ID"`
	Skills        []Skill   `json:"skills" doc:"List of trained skills"`
	TotalSP       int64     `json:"total_sp" doc:"Total skill points"`
	UnallocatedSP *int      `json:"unallocated_sp,omitempty" doc:"Unallocated skill points"`
	UpdatedAt     time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterSkillsOutput represents the skills response (Huma wrapper)
type CharacterSkillsOutput struct {
	Body CharacterSkills `json:"body"`
}

// CorporationHistoryEntry represents a single corporation history entry
type CorporationHistoryEntry struct {
	CorporationID int       `json:"corporation_id" doc:"Corporation ID"`
	IsDeleted     bool      `json:"is_deleted,omitempty" doc:"True if the corporation is deleted"`
	RecordID      int       `json:"record_id" doc:"Unique record ID"`
	StartDate     time.Time `json:"start_date" doc:"Date the character joined the corporation"`
}

// CharacterCorporationHistory represents the character's complete corporation history
type CharacterCorporationHistory struct {
	CharacterID int                       `json:"character_id" doc:"EVE Online character ID"`
	History     []CorporationHistoryEntry `json:"history" doc:"List of corporation history entries"`
	UpdatedAt   time.Time                 `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterCorporationHistoryOutput represents the corporation history response (Huma wrapper)
type CharacterCorporationHistoryOutput struct {
	Body CharacterCorporationHistory `json:"body"`
}

// HomeLocation represents the character's home location
type HomeLocation struct {
	LocationID   int64  `json:"location_id" doc:"Location ID of the home location"`
	LocationType string `json:"location_type" doc:"Type of location (station or structure)"`
}

// JumpClone represents a single jump clone
type JumpClone struct {
	Implants     []int  `json:"implants" doc:"List of implant type IDs"`
	JumpCloneID  int    `json:"jump_clone_id" doc:"Unique jump clone ID"`
	LocationID   int64  `json:"location_id" doc:"Location ID of the jump clone"`
	LocationType string `json:"location_type" doc:"Type of location (station or structure)"`
	Name         string `json:"name,omitempty" doc:"Optional name for the jump clone"`
}

// CharacterClones represents the character's clone information
type CharacterClones struct {
	CharacterID           int           `json:"character_id" doc:"EVE Online character ID"`
	HomeLocation          *HomeLocation `json:"home_location,omitempty" doc:"Home location details"`
	JumpClones            []JumpClone   `json:"jump_clones" doc:"List of jump clones"`
	LastCloneJumpDate     *time.Time    `json:"last_clone_jump_date,omitempty" doc:"Last clone jump timestamp"`
	LastStationChangeDate *time.Time    `json:"last_station_change_date,omitempty" doc:"Last station change timestamp"`
	UpdatedAt             time.Time     `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterClonesOutput represents the clones response (Huma wrapper)
type CharacterClonesOutput struct {
	Body CharacterClones `json:"body"`
}

// CharacterImplants represents the character's active implants
type CharacterImplants struct {
	CharacterID int       `json:"character_id" doc:"EVE Online character ID"`
	Implants    []int     `json:"implants" doc:"List of implant type IDs"`
	UpdatedAt   time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterImplantsOutput represents the implants response (Huma wrapper)
type CharacterImplantsOutput struct {
	Body CharacterImplants `json:"body"`
}

// CharacterLocation represents the character's current location
type CharacterLocation struct {
	CharacterID   int       `json:"character_id" doc:"EVE Online character ID"`
	SolarSystemID int       `json:"solar_system_id" doc:"Current solar system ID"`
	StationID     *int      `json:"station_id,omitempty" doc:"Current station ID if docked at station"`
	StructureID   *int64    `json:"structure_id,omitempty" doc:"Current structure ID if docked at structure"`
	UpdatedAt     time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterLocationOutput represents the location response (Huma wrapper)
type CharacterLocationOutput struct {
	Body CharacterLocation `json:"body"`
}

// CharacterFatigue represents the character's jump fatigue information
type CharacterFatigue struct {
	CharacterID           int        `json:"character_id" doc:"EVE Online character ID"`
	JumpFatigueExpireDate *time.Time `json:"jump_fatigue_expire_date,omitempty" doc:"Date when jump fatigue expires"`
	LastJumpDate          *time.Time `json:"last_jump_date,omitempty" doc:"Date of last jump"`
	LastUpdateDate        *time.Time `json:"last_update_date,omitempty" doc:"Date when this information was last updated"`
	UpdatedAt             time.Time  `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterFatigueOutput represents the fatigue response (Huma wrapper)
type CharacterFatigueOutput struct {
	Body CharacterFatigue `json:"body"`
}

// CharacterOnline represents the character's online status information
type CharacterOnline struct {
	CharacterID int        `json:"character_id" doc:"EVE Online character ID"`
	Online      bool       `json:"online" doc:"True if character is currently online"`
	LastLogin   *time.Time `json:"last_login,omitempty" doc:"Date and time when character last logged in"`
	LastLogout  *time.Time `json:"last_logout,omitempty" doc:"Date and time when character last logged out"`
	LoginsToday *int       `json:"logins" doc:"Total number of times the character has logged in today"`
	UpdatedAt   time.Time  `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterOnlineOutput represents the online status response (Huma wrapper)
type CharacterOnlineOutput struct {
	Body CharacterOnline `json:"body"`
}

// CharacterShip represents the character's current ship information
type CharacterShip struct {
	CharacterID int       `json:"character_id" doc:"EVE Online character ID"`
	ShipItemID  int64     `json:"ship_item_id" doc:"Item ID of the current ship"`
	ShipName    string    `json:"ship_name" doc:"Name of the current ship"`
	ShipTypeID  int       `json:"ship_type_id" doc:"Type ID of the current ship"`
	UpdatedAt   time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterShipOutput represents the ship response (Huma wrapper)
type CharacterShipOutput struct {
	Body CharacterShip `json:"body"`
}

// CharacterWallet represents the character's wallet balance information
type CharacterWallet struct {
	CharacterID int       `json:"character_id" doc:"EVE Online character ID"`
	Balance     float64   `json:"balance" doc:"Current wallet balance in ISK"`
	UpdatedAt   time.Time `json:"updated_at" doc:"Last update timestamp"`
}

// CharacterWalletOutput represents the wallet response (Huma wrapper)
type CharacterWalletOutput struct {
	Body CharacterWallet `json:"body"`
}
