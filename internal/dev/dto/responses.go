package dto

import (
	"time"
)

// DevResponse represents the standard development response wrapper
type DevResponse struct {
	Source         string      `json:"source"`
	Endpoint       string      `json:"endpoint"`
	ResponseTimeMS int64       `json:"response_time_ms"`
	Status         string      `json:"status"`
	Data           interface{} `json:"data,omitempty"`
	Module         string      `json:"module"`
	Timestamp      time.Time   `json:"timestamp"`
	Cache          *CacheInfo  `json:"cache,omitempty"`
	Error          string      `json:"error,omitempty"`
	Details        string      `json:"details,omitempty"`
}

// CacheInfo represents cache-related information
type CacheInfo struct {
	Cached    bool      `json:"cached"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	ExpiresIn int       `json:"expires_in,omitempty"`
	CacheHit  bool      `json:"cache_hit,omitempty"`
	CacheKey  string    `json:"cache_key,omitempty"`
}

// ESIStatusResponse represents the EVE Online server status
type ESIStatusResponse struct {
	DevResponse
	ServerVersion string `json:"server_version,omitempty"`
	Players       int    `json:"players,omitempty"`
	StartTime     string `json:"start_time,omitempty"`
	VIP           bool   `json:"vip,omitempty"`
}

// CharacterResponse represents character information
type CharacterResponse struct {
	DevResponse
	Name            string    `json:"name,omitempty"`
	CorporationID   int       `json:"corporation_id,omitempty"`
	AllianceID      int       `json:"alliance_id,omitempty"`
	FactionID       int       `json:"faction_id,omitempty"`
	SecurityStatus  float64   `json:"security_status,omitempty"`
	Birthday        time.Time `json:"birthday,omitempty"`
	Gender          string    `json:"gender,omitempty"`
	RaceID          int       `json:"race_id,omitempty"`
	BloodlineID     int       `json:"bloodline_id,omitempty"`
	AncestryID      int       `json:"ancestry_id,omitempty"`
	Title           string    `json:"title,omitempty"`
}

// AllianceResponse represents alliance information
type AllianceResponse struct {
	DevResponse
	Name             string    `json:"name,omitempty"`
	Ticker           string    `json:"ticker,omitempty"`
	ExecutorCorp     int       `json:"executor_corporation_id,omitempty"`
	DateFounded      time.Time `json:"date_founded,omitempty"`
	CreatorID        int       `json:"creator_id,omitempty"`
	CreatorCorpID    int       `json:"creator_corporation_id,omitempty"`
	FactionID        int       `json:"faction_id,omitempty"`
}

// CorporationResponse represents corporation information
type CorporationResponse struct {
	DevResponse
	Name           string    `json:"name,omitempty"`
	Ticker         string    `json:"ticker,omitempty"`
	MemberCount    int       `json:"member_count,omitempty"`
	AllianceID     int       `json:"alliance_id,omitempty"`
	FactionID      int       `json:"faction_id,omitempty"`
	DateFounded    time.Time `json:"date_founded,omitempty"`
	CreatorID      int       `json:"creator_id,omitempty"`
	CEOID          int       `json:"ceo_id,omitempty"`
	URL            string    `json:"url,omitempty"`
	Description    string    `json:"description,omitempty"`
	TaxRate        float64   `json:"tax_rate,omitempty"`
	WarEligible    bool      `json:"war_eligible,omitempty"`
}

// SystemResponse represents solar system information
type SystemResponse struct {
	DevResponse
	Name            string         `json:"name,omitempty"`
	SystemID        int            `json:"system_id,omitempty"`
	ConstellationID int            `json:"constellation_id,omitempty"`
	StarID          int            `json:"star_id,omitempty"`
	SecurityStatus  float64        `json:"security_status,omitempty"`
	SecurityClass   string         `json:"security_class,omitempty"`
	Planets         []PlanetInfo   `json:"planets,omitempty"`
	Stargates       []StargateInfo `json:"stargates,omitempty"`
	Stations        []StationInfo  `json:"stations,omitempty"`
	Position        Position       `json:"position,omitempty"`
}

// PlanetInfo represents planet information
type PlanetInfo struct {
	PlanetID int      `json:"planet_id"`
	Moons    []int    `json:"moons,omitempty"`
	Position Position `json:"position"`
}

// StargateInfo represents stargate information
type StargateInfo struct {
	StargateID  int      `json:"stargate_id"`
	Destination int      `json:"destination"`
	Position    Position `json:"position"`
}

// StationInfo represents station information
type StationInfo struct {
	StationID int      `json:"station_id"`
	Name      string   `json:"name"`
	OwnerID   int      `json:"owner_id"`
	TypeID    int      `json:"type_id"`
	Position  Position `json:"position"`
}

// Position represents 3D coordinates
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// DevSDEStatusResponse represents SDE service status from dev module perspective
type DevSDEStatusResponse struct {
	DevResponse
	Loaded        bool              `json:"loaded"`
	EntitiesCount map[string]int    `json:"entities_count,omitempty"`
	LoadTime      time.Duration     `json:"load_time,omitempty"`
	MemoryUsage   int64             `json:"memory_usage_bytes,omitempty"`
	LastUpdate    time.Time         `json:"last_update,omitempty"`
	Version       string            `json:"version,omitempty"`
	Statistics    *SDEStatistics    `json:"statistics,omitempty"`
}

// SDEStatistics represents SDE statistics
type SDEStatistics struct {
	TotalEntities    int            `json:"total_entities"`
	EntitiesByType   map[string]int `json:"entities_by_type"`
	DataSizeBytes    int64          `json:"data_size_bytes"`
	IndexSizeBytes   int64          `json:"index_size_bytes"`
	LastUpdate       time.Time      `json:"last_update"`
}

// SDEEntityResponse represents SDE entity data
type SDEEntityResponse struct {
	DevResponse
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	EntityData interface{} `json:"entity_data"`
}

// SDETypesResponse represents SDE types collection
type SDETypesResponse struct {
	DevResponse
	PublishedOnly bool                   `json:"published_only"`
	Count         int                    `json:"count"`
	Types         map[string]interface{} `json:"types"`
}

// UniverseResponse represents universe data
type UniverseResponse struct {
	DevResponse
	Type          string      `json:"universe_type"`
	Region        string      `json:"region,omitempty"`
	Constellation string      `json:"constellation,omitempty"`
	System        string      `json:"system,omitempty"`
	Data          interface{} `json:"data"`
}

// UniverseSystemsResponse represents systems in a region/constellation
type UniverseSystemsResponse struct {
	DevResponse
	Type          string   `json:"universe_type"`
	Region        string   `json:"region"`
	Constellation string   `json:"constellation,omitempty"`
	Systems       []string `json:"systems"`
	Count         int      `json:"count"`
}

// ServiceDiscoveryResponse represents available services
type ServiceDiscoveryResponse struct {
	Services  []ServiceInfo `json:"services"`
	Count     int           `json:"count"`
	Timestamp time.Time     `json:"timestamp"`
}

// ServiceInfo represents information about a service
type ServiceInfo struct {
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Status      string              `json:"status"`
	Endpoints   []EndpointInfo      `json:"endpoints,omitempty"`
	Health      *HealthInfo         `json:"health,omitempty"`
	Metadata    map[string]string   `json:"metadata,omitempty"`
}

// EndpointInfo represents endpoint information
type EndpointInfo struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Description string `json:"description,omitempty"`
	Permission  string `json:"permission,omitempty"`
}

// HealthInfo represents health information
type HealthInfo struct {
	Status    string            `json:"status"`
	Checks    []DevHealthCheck     `json:"checks,omitempty"`
	Uptime    time.Duration     `json:"uptime,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// DevHealthCheck represents an individual health check
type DevHealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CacheTestResponse represents cache operation results
type CacheTestResponse struct {
	DevResponse
	Operation string      `json:"operation"`
	Key       string      `json:"key,omitempty"`
	Hit       bool        `json:"hit,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	TTL       int         `json:"ttl,omitempty"`
	Stats     *CacheStats `json:"stats,omitempty"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalKeys   int     `json:"total_keys"`
	HitRate     float64 `json:"hit_rate"`
	MissRate    float64 `json:"miss_rate"`
	TotalHits   int64   `json:"total_hits"`
	TotalMisses int64   `json:"total_misses"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
}

// ValidationTestResponse represents validation test results
type ValidationTestResponse struct {
	DevResponse
	TestType  string      `json:"test_type"`
	TestValue interface{} `json:"test_value"`
	Valid     bool        `json:"valid"`
	Message   string      `json:"message,omitempty"`
	Errors    []string    `json:"errors,omitempty"`
}

// PerformanceTestResponse represents performance test results
type PerformanceTestResponse struct {
	DevResponse
	TestType      string            `json:"test_type"`
	Iterations    int               `json:"iterations"`
	TotalTime     time.Duration     `json:"total_time"`
	AverageTime   time.Duration     `json:"average_time"`
	MinTime       time.Duration     `json:"min_time"`
	MaxTime       time.Duration     `json:"max_time"`
	Concurrent    bool              `json:"concurrent"`
	SuccessRate   float64           `json:"success_rate"`
	ErrorRate     float64           `json:"error_rate"`
	ThroughputRPS float64           `json:"throughput_rps"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// DebugResponse represents debugging information
type DebugResponse struct {
	DevResponse
	Component   string                 `json:"component"`
	Action      string                 `json:"action"`
	Information map[string]interface{} `json:"information"`
	Logs        []LogEntry             `json:"logs,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// MockDataResponse represents generated mock data
type MockDataResponse struct {
	DevResponse
	DataType string        `json:"data_type"`
	Count    int           `json:"count"`
	Seed     int64         `json:"seed"`
	Data     []interface{} `json:"data"`
}

// BulkTestResponse represents bulk test operation results
type BulkTestResponse struct {
	DevResponse
	TotalOperations     int                  `json:"total_operations"`
	SuccessfulOperations int                 `json:"successful_operations"`
	FailedOperations    int                  `json:"failed_operations"`
	ExecutionTime       time.Duration        `json:"execution_time"`
	Results             []BulkTestResult     `json:"results"`
	Summary             *BulkTestSummary     `json:"summary,omitempty"`
}

// BulkTestResult represents a single bulk test result
type BulkTestResult struct {
	Operation     TestOperation `json:"operation"`
	Success       bool          `json:"success"`
	ExecutionTime time.Duration `json:"execution_time"`
	Response      interface{}   `json:"response,omitempty"`
	Error         string        `json:"error,omitempty"`
}

// BulkTestSummary represents bulk test summary statistics
type BulkTestSummary struct {
	SuccessRate      float64       `json:"success_rate"`
	AverageTime      time.Duration `json:"average_time"`
	TotalTime        time.Duration `json:"total_time"`
	ErrorsByType     map[string]int `json:"errors_by_type,omitempty"`
	PerformanceStats *PerformanceStats `json:"performance_stats,omitempty"`
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	Min    time.Duration `json:"min"`
	Max    time.Duration `json:"max"`
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"`
	P95    time.Duration `json:"p95"`
	P99    time.Duration `json:"p99"`
}

// DevHealthResponse represents module health information
type DevHealthResponse struct {
	Status     string        `json:"status"`
	Module     string        `json:"module"`
	Version    string        `json:"version"`
	Timestamp  time.Time     `json:"timestamp"`
	Checks     []DevHealthCheck `json:"checks"`
	Uptime     time.Duration `json:"uptime,omitempty"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
}

// ComponentHealth represents health of individual components
type ComponentHealth struct {
	Status      string            `json:"status"`
	LastCheck   time.Time         `json:"last_check"`
	ResponseTime time.Duration    `json:"response_time,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}