package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestExecution represents a test execution record
type TestExecution struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	TestType      string                 `bson:"test_type" json:"test_type"`
	TestName      string                 `bson:"test_name" json:"test_name"`
	Status        string                 `bson:"status" json:"status"` // "running", "completed", "failed"
	StartTime     time.Time              `bson:"start_time" json:"start_time"`
	EndTime       *time.Time             `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Duration      time.Duration          `bson:"duration" json:"duration"`
	UserID        int                    `bson:"user_id" json:"user_id"`
	Parameters    map[string]interface{} `bson:"parameters" json:"parameters"`
	Results       map[string]interface{} `bson:"results" json:"results"`
	Error         string                 `bson:"error,omitempty" json:"error,omitempty"`
	Metadata      map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt     time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
}

// CacheMetrics represents cache performance metrics
type CacheMetrics struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CacheType       string             `bson:"cache_type" json:"cache_type"` // "esi", "sde", "general"
	TotalRequests   int64              `bson:"total_requests" json:"total_requests"`
	CacheHits       int64              `bson:"cache_hits" json:"cache_hits"`
	CacheMisses     int64              `bson:"cache_misses" json:"cache_misses"`
	HitRate         float64            `bson:"hit_rate" json:"hit_rate"`
	AverageLatency  time.Duration      `bson:"average_latency" json:"average_latency"`
	MemoryUsage     int64              `bson:"memory_usage_bytes" json:"memory_usage_bytes"`
	KeyCount        int                `bson:"key_count" json:"key_count"`
	ExpiredKeys     int64              `bson:"expired_keys" json:"expired_keys"`
	EvictedKeys     int64              `bson:"evicted_keys" json:"evicted_keys"`
	LastReset       time.Time          `bson:"last_reset" json:"last_reset"`
	Timestamp       time.Time          `bson:"timestamp" json:"timestamp"`
}

// ESIMetrics represents ESI API performance metrics
type ESIMetrics struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Endpoint          string                 `bson:"endpoint" json:"endpoint"`
	TotalRequests     int64                  `bson:"total_requests" json:"total_requests"`
	SuccessfulRequests int64                 `bson:"successful_requests" json:"successful_requests"`
	FailedRequests    int64                  `bson:"failed_requests" json:"failed_requests"`
	ErrorRate         float64                `bson:"error_rate" json:"error_rate"`
	AverageLatency    time.Duration          `bson:"average_latency" json:"average_latency"`
	MinLatency        time.Duration          `bson:"min_latency" json:"min_latency"`
	MaxLatency        time.Duration          `bson:"max_latency" json:"max_latency"`
	ErrorLimitHits    int64                  `bson:"error_limit_hits" json:"error_limit_hits"`
	LastErrorLimit    time.Time              `bson:"last_error_limit" json:"last_error_limit"`
	StatusCodes       map[string]int64       `bson:"status_codes" json:"status_codes"`
	ErrorTypes        map[string]int64       `bson:"error_types" json:"error_types"`
	CacheEfficiency   *CacheEfficiencyMetrics `bson:"cache_efficiency,omitempty" json:"cache_efficiency,omitempty"`
	Timestamp         time.Time              `bson:"timestamp" json:"timestamp"`
	LastReset         time.Time              `bson:"last_reset" json:"last_reset"`
}

// CacheEfficiencyMetrics represents cache efficiency metrics
type CacheEfficiencyMetrics struct {
	HitRate         float64       `bson:"hit_rate" json:"hit_rate"`
	MissRate        float64       `bson:"miss_rate" json:"miss_rate"`
	AverageHitTime  time.Duration `bson:"average_hit_time" json:"average_hit_time"`
	AverageMissTime time.Duration `bson:"average_miss_time" json:"average_miss_time"`
	TotalHits       int64         `bson:"total_hits" json:"total_hits"`
	TotalMisses     int64         `bson:"total_misses" json:"total_misses"`
}

// PerformanceTest represents a performance test configuration
type PerformanceTest struct {
	ID             primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name           string                 `bson:"name" json:"name"`
	TestType       string                 `bson:"test_type" json:"test_type"`
	Description    string                 `bson:"description" json:"description"`
	Configuration  map[string]interface{} `bson:"configuration" json:"configuration"`
	Iterations     int                    `bson:"iterations" json:"iterations"`
	Concurrent     bool                   `bson:"concurrent" json:"concurrent"`
	WarmupRuns     int                    `bson:"warmup_runs" json:"warmup_runs"`
	ExpectedResult *ExpectedResult        `bson:"expected_result,omitempty" json:"expected_result,omitempty"`
	IsActive       bool                   `bson:"is_active" json:"is_active"`
	CreatedBy      int                    `bson:"created_by" json:"created_by"`
	CreatedAt      time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time              `bson:"updated_at" json:"updated_at"`
}

// ExpectedResult represents expected performance test results
type ExpectedResult struct {
	MaxLatency      time.Duration `bson:"max_latency" json:"max_latency"`
	MinSuccessRate  float64       `bson:"min_success_rate" json:"min_success_rate"`
	MaxErrorRate    float64       `bson:"max_error_rate" json:"max_error_rate"`
	MinThroughput   float64       `bson:"min_throughput" json:"min_throughput"`
}

// TestResult represents performance test results
type TestResult struct {
	ID                primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	TestID            primitive.ObjectID     `bson:"test_id" json:"test_id"`
	TestName          string                 `bson:"test_name" json:"test_name"`
	ExecutionID       primitive.ObjectID     `bson:"execution_id" json:"execution_id"`
	Status            string                 `bson:"status" json:"status"`
	StartTime         time.Time              `bson:"start_time" json:"start_time"`
	EndTime           time.Time              `bson:"end_time" json:"end_time"`
	TotalDuration     time.Duration          `bson:"total_duration" json:"total_duration"`
	Iterations        int                    `bson:"iterations" json:"iterations"`
	SuccessfulRuns    int                    `bson:"successful_runs" json:"successful_runs"`
	FailedRuns        int                    `bson:"failed_runs" json:"failed_runs"`
	SuccessRate       float64                `bson:"success_rate" json:"success_rate"`
	ErrorRate         float64                `bson:"error_rate" json:"error_rate"`
	AverageLatency    time.Duration          `bson:"average_latency" json:"average_latency"`
	MinLatency        time.Duration          `bson:"min_latency" json:"min_latency"`
	MaxLatency        time.Duration          `bson:"max_latency" json:"max_latency"`
	MedianLatency     time.Duration          `bson:"median_latency" json:"median_latency"`
	P95Latency        time.Duration          `bson:"p95_latency" json:"p95_latency"`
	P99Latency        time.Duration          `bson:"p99_latency" json:"p99_latency"`
	ThroughputRPS     float64                `bson:"throughput_rps" json:"throughput_rps"`
	MemoryUsage       int64                  `bson:"memory_usage_bytes" json:"memory_usage_bytes"`
	CPUUsage          float64                `bson:"cpu_usage_percent" json:"cpu_usage_percent"`
	ErrorDetails      map[string]int         `bson:"error_details" json:"error_details"`
	ResultsData       map[string]interface{} `bson:"results_data" json:"results_data"`
	Passed            bool                   `bson:"passed" json:"passed"`
	PassedChecks      []string               `bson:"passed_checks" json:"passed_checks"`
	FailedChecks      []string               `bson:"failed_checks" json:"failed_checks"`
	CreatedAt         time.Time              `bson:"created_at" json:"created_at"`
}

// DebugSession represents a debugging session
type DebugSession struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	SessionName string                 `bson:"session_name" json:"session_name"`
	Component   string                 `bson:"component" json:"component"`
	UserID      int                    `bson:"user_id" json:"user_id"`
	Status      string                 `bson:"status" json:"status"` // "active", "completed", "expired"
	StartTime   time.Time              `bson:"start_time" json:"start_time"`
	EndTime     *time.Time             `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Actions     []DebugAction          `bson:"actions" json:"actions"`
	Logs        []DebugLog             `bson:"logs" json:"logs"`
	Metadata    map[string]interface{} `bson:"metadata" json:"metadata"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

// DebugAction represents an action performed during debugging
type DebugAction struct {
	Action      string                 `bson:"action" json:"action"`
	Timestamp   time.Time              `bson:"timestamp" json:"timestamp"`
	Parameters  map[string]interface{} `bson:"parameters" json:"parameters"`
	Result      map[string]interface{} `bson:"result" json:"result"`
	Error       string                 `bson:"error,omitempty" json:"error,omitempty"`
	Duration    time.Duration          `bson:"duration" json:"duration"`
}

// DebugLog represents a debug log entry
type DebugLog struct {
	Level     string                 `bson:"level" json:"level"`
	Message   string                 `bson:"message" json:"message"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	Context   map[string]interface{} `bson:"context" json:"context"`
	Source    string                 `bson:"source" json:"source"`
}

// MockDataTemplate represents a template for generating mock data
type MockDataTemplate struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	DataType    string                 `bson:"data_type" json:"data_type"`
	Description string                 `bson:"description" json:"description"`
	Template    map[string]interface{} `bson:"template" json:"template"`
	Rules       []GenerationRule       `bson:"rules" json:"rules"`
	IsActive    bool                   `bson:"is_active" json:"is_active"`
	CreatedBy   int                    `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

// GenerationRule represents a rule for generating mock data
type GenerationRule struct {
	Field     string      `bson:"field" json:"field"`
	Type      string      `bson:"type" json:"type"` // "range", "list", "pattern", "function"
	Value     interface{} `bson:"value" json:"value"`
	Optional  bool        `bson:"optional" json:"optional"`
	Condition string      `bson:"condition,omitempty" json:"condition,omitempty"`
}

// ComponentStatus represents the status of individual components
type ComponentStatus struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Component    string             `bson:"component" json:"component"`
	Status       string             `bson:"status" json:"status"` // "healthy", "degraded", "unhealthy"
	LastCheck    time.Time          `bson:"last_check" json:"last_check"`
	ResponseTime time.Duration      `bson:"response_time" json:"response_time"`
	Error        string             `bson:"error,omitempty" json:"error,omitempty"`
	Version      string             `bson:"version,omitempty" json:"version,omitempty"`
	Metadata     map[string]string  `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Uptime       time.Duration      `bson:"uptime" json:"uptime"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// Constants for test execution status
const (
	TestStatusPending   = "pending"
	TestStatusRunning   = "running"
	TestStatusCompleted = "completed"
	TestStatusFailed    = "failed"
	TestStatusCancelled = "cancelled"
)

// Constants for test types
const (
	TestTypeESI         = "esi"
	TestTypeSDE         = "sde"
	TestTypeCache       = "cache"
	TestTypeValidation  = "validation"
	TestTypePerformance = "performance"
	TestTypeHealth      = "health"
	TestTypeMock        = "mock"
	TestTypeDebug       = "debug"
	TestTypeBulk        = "bulk"
)

// Constants for component health status
const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
	StatusUnknown   = "unknown"
)

// Constants for cache types
const (
	CacheTypeESI     = "esi"
	CacheTypeSDE     = "sde"
	CacheTypeGeneral = "general"
	CacheTypeSession = "session"
)

// Constants for debug session status
const (
	DebugStatusActive    = "active"
	DebugStatusCompleted = "completed"
	DebugStatusExpired   = "expired"
	DebugStatusCancelled = "cancelled"
)

// Constants for ESI endpoints categorization
const (
	ESIEndpointTypePublic      = "public"
	ESIEndpointTypePrivate     = "private"
	ESIEndpointTypeApplication = "application"
)

// Constants for universe types
const (
	UniverseTypeEve      = "eve"
	UniverseTypeAbyssal  = "abyssal"
	UniverseTypeWormhole = "wormhole"
	UniverseTypeVoid     = "void"
	UniverseTypeHidden   = "hidden"
)

// Constants for performance test thresholds
const (
	DefaultMaxLatency      = 5 * time.Second
	DefaultMinSuccessRate  = 0.95
	DefaultMaxErrorRate    = 0.05
	DefaultMinThroughput   = 10.0
)

// Constants for data generation
const (
	MockDataTypeCharacter   = "character"
	MockDataTypeAlliance    = "alliance"
	MockDataTypeCorporation = "corporation"
	MockDataTypeSystem      = "system"
	MockDataTypeType        = "type"
)

// MongoDB collection names
const (
	TestExecutionsCollection = "dev_test_executions"
	CacheMetricsCollection   = "dev_cache_metrics"
	ESIMetricsCollection     = "dev_esi_metrics"
	PerformanceTestsCollection = "dev_performance_tests"
	TestResultsCollection    = "dev_test_results"
	DebugSessionsCollection  = "dev_debug_sessions"
	MockDataTemplatesCollection = "dev_mock_data_templates"
	ComponentStatusCollection = "dev_component_status"
)

// Default configuration values
const (
	DefaultTestTimeout     = 30 * time.Second
	DefaultMaxIterations   = 1000
	DefaultMaxConcurrency  = 100
	DefaultCacheExpiration = 5 * time.Minute
	DefaultDebugSessionTTL = 1 * time.Hour
)

// Common error types
const (
	ErrorTypeValidation = "validation"
	ErrorTypeNetwork    = "network"
	ErrorTypeTimeout    = "timeout"
	ErrorTypeAuth       = "authentication"
	ErrorTypePermission = "permission"
	ErrorTypeRateLimit  = "rate_limit"
	ErrorTypeESI        = "esi"
	ErrorTypeSDE        = "sde"
	ErrorTypeCache      = "cache"
	ErrorTypeInternal   = "internal"
)