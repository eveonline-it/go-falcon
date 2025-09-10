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
