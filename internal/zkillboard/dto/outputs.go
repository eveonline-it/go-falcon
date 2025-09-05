package dto

import (
	"time"
)

// ServiceStatusOutput represents the status of the ZKillboard consumer service
type ServiceStatusOutput struct {
	Body ServiceStatusResponse `json:"body" doc:"ZKillboard service status"`
}

// ServiceStatusResponse represents the actual status data
type ServiceStatusResponse struct {
	Status       string         `json:"status" doc:"Service status (stopped, running, throttled, draining)"`
	QueueID      string         `json:"queue_id" doc:"Unique queue identifier"`
	LastPoll     *time.Time     `json:"last_poll,omitempty" doc:"Last successful poll time"`
	LastKillmail *int64         `json:"last_killmail_id,omitempty" doc:"Last processed killmail ID"`
	Metrics      ServiceMetrics `json:"metrics" doc:"Service performance metrics"`
	Config       ServiceConfig  `json:"config" doc:"Service configuration"`
	Message      string         `json:"message,omitempty" doc:"Status message"`
}

// ServiceMetrics represents performance metrics for the consumer
type ServiceMetrics struct {
	TotalPolls     int64         `json:"total_polls" doc:"Total number of polls made"`
	NullResponses  int64         `json:"null_responses" doc:"Number of null responses received"`
	KillmailsFound int64         `json:"killmails_found" doc:"Number of killmails processed"`
	HTTPErrors     int64         `json:"http_errors" doc:"Number of HTTP errors encountered"`
	ParseErrors    int64         `json:"parse_errors" doc:"Number of parse errors"`
	StoreErrors    int64         `json:"store_errors" doc:"Number of storage errors"`
	RateLimitHits  int64         `json:"rate_limit_hits" doc:"Number of rate limit hits"`
	CurrentTTW     int           `json:"current_ttw" doc:"Current time-to-wait value (seconds)"`
	NullStreak     int           `json:"null_streak" doc:"Consecutive null responses"`
	Uptime         time.Duration `json:"uptime" doc:"Service uptime duration"`
}

// ServiceConfig represents the current service configuration
type ServiceConfig struct {
	Endpoint      string `json:"endpoint" doc:"RedisQ endpoint URL"`
	TTWMin        int    `json:"ttw_min" doc:"Minimum time-to-wait (seconds)"`
	TTWMax        int    `json:"ttw_max" doc:"Maximum time-to-wait (seconds)"`
	NullThreshold int    `json:"null_threshold" doc:"Null responses before increasing TTW"`
	BatchSize     int    `json:"batch_size" doc:"Database batch insert size"`
}

// ServiceControlInput represents input for service control operations
type ServiceControlInput struct {
	Action  string `json:"action" required:"true" enum:"start,stop,restart" doc:"Control action to perform"`
	QueueID string `json:"queue_id,omitempty" doc:"Optional queue ID override"`
}

// ServiceControlOutput represents the result of a service control operation
type ServiceControlOutput struct {
	Body ServiceControlResponse `json:"body" doc:"Service control operation result"`
}

// ServiceControlResponse represents the actual control operation result
type ServiceControlResponse struct {
	Success bool   `json:"success" doc:"Whether the operation succeeded"`
	Message string `json:"message" doc:"Operation result message"`
	Status  string `json:"status" doc:"Current service status"`
}

// ServiceStatsOutput represents statistical data about processed killmails
type ServiceStatsOutput struct {
	Body ServiceStatsResponse `json:"body" doc:"Killmail statistics data"`
}

// ServiceStatsResponse represents the actual statistics data
type ServiceStatsResponse struct {
	Period         string          `json:"period" doc:"Statistics period"`
	TotalKillmails int64           `json:"total_killmails" doc:"Total killmails in period"`
	TotalValue     float64         `json:"total_value" doc:"Total ISK destroyed"`
	NPCKills       int64           `json:"npc_kills" doc:"Number of NPC kills"`
	SoloKills      int64           `json:"solo_kills" doc:"Number of solo kills"`
	TopSystems     []SystemStats   `json:"top_systems" doc:"Most active systems"`
	TopAlliances   []AllianceStats `json:"top_alliances" doc:"Most active alliances"`
	TopShipTypes   []ShipTypeStats `json:"top_ship_types" doc:"Most destroyed ship types"`
}

// SystemStats represents killmail statistics for a solar system
type SystemStats struct {
	SystemID   int32   `json:"system_id" doc:"Solar system ID"`
	SystemName string  `json:"system_name" doc:"Solar system name"`
	Kills      int64   `json:"kills" doc:"Number of kills"`
	Value      float64 `json:"value" doc:"Total ISK destroyed"`
}

// AllianceStats represents killmail statistics for an alliance
type AllianceStats struct {
	AllianceID   int32   `json:"alliance_id" doc:"Alliance ID"`
	AllianceName string  `json:"alliance_name" doc:"Alliance name"`
	Kills        int64   `json:"kills" doc:"Number of kills involved"`
	Losses       int64   `json:"losses" doc:"Number of losses"`
	Value        float64 `json:"value" doc:"Total ISK value"`
}

// ShipTypeStats represents statistics for a ship type
type ShipTypeStats struct {
	ShipTypeID   int32   `json:"ship_type_id" doc:"Ship type ID"`
	ShipTypeName string  `json:"ship_type_name" doc:"Ship type name"`
	Destroyed    int64   `json:"destroyed" doc:"Number destroyed"`
	Value        float64 `json:"value" doc:"Total ISK value"`
}

// RecentKillmailsOutput represents recently processed killmails
type RecentKillmailsOutput struct {
	Body RecentKillmailsResponse `json:"body" doc:"Recent killmails data"`
}

// RecentKillmailsResponse represents the actual recent killmails data
type RecentKillmailsResponse struct {
	Killmails []KillmailSummary `json:"killmails" doc:"Recent killmails"`
	Count     int               `json:"count" doc:"Number of killmails returned"`
}

// KillmailSummary represents a summary of a processed killmail
type KillmailSummary struct {
	KillmailID    int64     `json:"killmail_id" doc:"Killmail ID"`
	Timestamp     time.Time `json:"timestamp" doc:"Kill time"`
	SolarSystemID int32     `json:"solar_system_id" doc:"Solar system ID"`
	SystemName    string    `json:"system_name" doc:"Solar system name"`
	VictimID      *int32    `json:"victim_id,omitempty" doc:"Victim character ID"`
	VictimName    string    `json:"victim_name" doc:"Victim name"`
	ShipTypeID    int32     `json:"ship_type_id" doc:"Destroyed ship type"`
	ShipTypeName  string    `json:"ship_type_name" doc:"Ship type name"`
	TotalValue    float64   `json:"total_value" doc:"Total ISK value"`
	Points        int       `json:"points" doc:"ZKillboard points"`
	Solo          bool      `json:"solo" doc:"Solo kill flag"`
	NPC           bool      `json:"npc" doc:"NPC kill flag"`
	Href          string    `json:"href" doc:"ZKillboard URL"`
}
