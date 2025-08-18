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
}

// CharacterResponse represents character information
type CharacterResponse struct {
	DevResponse
}

// AllianceResponse represents alliance information
type AllianceResponse struct {
	DevResponse
}

// CorporationResponse represents corporation information
type CorporationResponse struct {
	DevResponse
}

// SystemResponse represents solar system information
type SystemResponse struct {
	DevResponse
}


// DevSDEStatusResponse represents SDE service status from dev module perspective
type DevSDEStatusResponse struct {
	DevResponse
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
}

// SDETypesResponse represents SDE types collection
type SDETypesResponse struct {
	DevResponse
}

// UniverseResponse represents universe data
type UniverseResponse struct {
	DevResponse
}

// UniverseSystemsResponse represents systems in a region/constellation
type UniverseSystemsResponse struct {
	DevResponse
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