package dto

import (
	"time"
)

// ESITestRequest represents a request to test ESI endpoints
type ESITestRequest struct {
	Endpoint   string                 `json:"endpoint" validate:"required,min=1,max=255"`
	Method     string                 `json:"method" validate:"omitempty,oneof=GET POST PUT DELETE"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Headers    map[string]string      `json:"headers,omitempty"`
}

// CharacterRequest represents a request for character information
type CharacterRequest struct {
	CharacterID int `json:"character_id" validate:"required,min=1,max=2147483647"`
}

// AllianceRequest represents a request for alliance information
type AllianceRequest struct {
	AllianceID int `json:"alliance_id" validate:"required,min=1,max=2147483647"`
}

// CorporationRequest represents a request for corporation information
type CorporationRequest struct {
	CorporationID int `json:"corporation_id" validate:"required,min=1,max=2147483647"`
}

// SystemRequest represents a request for solar system information
type SystemRequest struct {
	SystemID int `json:"system_id" validate:"required,min=1,max=2147483647"`
}

// UniverseRequest represents a request for universe data
type UniverseRequest struct {
	Type          string `json:"type" validate:"required,oneof=eve abyssal wormhole void hidden"`
	Region        string `json:"region" validate:"omitempty,min=1,max=100"`
	Constellation string `json:"constellation" validate:"omitempty,min=1,max=100"`
	System        string `json:"system" validate:"omitempty,min=1,max=100"`
}

// SDEEntityRequest represents a request for SDE entity data
type SDEEntityRequest struct {
	Type string `json:"type" validate:"required,min=1,max=50"`
	ID   string `json:"id" validate:"required,min=1,max=100"`
}

// SDETypeRequest represents a request for SDE type data
type SDETypeRequest struct {
	TypeID    int  `json:"type_id" validate:"required,min=1,max=2147483647"`
	Published *bool `json:"published,omitempty"`
}

// SDEAgentRequest represents a request for SDE agent data
type SDEAgentRequest struct {
	AgentID int `json:"agent_id" validate:"required,min=1,max=2147483647"`
}

// SDECategoryRequest represents a request for SDE category data
type SDECategoryRequest struct {
	CategoryID int `json:"category_id" validate:"required,min=1,max=2147483647"`
}

// SDEBlueprintRequest represents a request for SDE blueprint data
type SDEBlueprintRequest struct {
	BlueprintID int `json:"blueprint_id" validate:"required,min=1,max=2147483647"`
}

// RedisSDERequest represents a request for Redis-based SDE data
type RedisSDERequest struct {
	Type string `json:"type" validate:"required,min=1,max=50"`
	ID   string `json:"id" validate:"omitempty,min=1,max=100"`
}

// CacheTestRequest represents a request for cache testing
type CacheTestRequest struct {
	CacheKey   string        `json:"cache_key" validate:"required,min=1,max=255"`
	Value      interface{}   `json:"value,omitempty"`
	Expiration time.Duration `json:"expiration,omitempty" validate:"omitempty,min=1s,max=24h"`
}

// ServiceDiscoveryRequest represents a request for service discovery
type ServiceDiscoveryRequest struct {
	ServiceName string `json:"service_name" validate:"omitempty,min=1,max=100"`
	Detailed    bool   `json:"detailed" validate:"boolean"`
}

// CacheOperationRequest represents a request for cache operations
type CacheOperationRequest struct {
	Operation string      `json:"operation" validate:"required,oneof=get set delete clear stats"`
	Key       string      `json:"key" validate:"omitempty,min=1,max=255"`
	Value     interface{} `json:"value,omitempty"`
	TTL       int         `json:"ttl" validate:"omitempty,min=1,max=86400"`
}

// ESITokenTestRequest represents a request to test ESI with tokens
type ESITokenTestRequest struct {
	AccessToken string   `json:"access_token" validate:"required,min=1"`
	Scopes      []string `json:"scopes,omitempty"`
	Endpoint    string   `json:"endpoint" validate:"required,min=1,max=255"`
}

// ValidationTestRequest represents a request for validation testing
type ValidationTestRequest struct {
	TestType   string                 `json:"test_type" validate:"required,oneof=character_id alliance_id corporation_id system_id type_id"`
	TestValue  interface{}            `json:"test_value" validate:"required"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// PerformanceTestRequest represents a request for performance testing
type PerformanceTestRequest struct {
	TestType    string `json:"test_type" validate:"required,oneof=esi_latency sde_speed cache_performance"`
	Iterations  int    `json:"iterations" validate:"omitempty,min=1,max=1000"`
	Concurrent  bool   `json:"concurrent" validate:"boolean"`
	WarmupRuns  int    `json:"warmup_runs" validate:"omitempty,min=0,max=100"`
}

// DebugRequest represents a request for debugging information
type DebugRequest struct {
	Component string                 `json:"component" validate:"required,oneof=esi sde cache auth permissions"`
	Action    string                 `json:"action" validate:"required,oneof=status info logs test"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// MockDataRequest represents a request for mock data generation
type MockDataRequest struct {
	DataType string `json:"data_type" validate:"required,oneof=character alliance corporation system type"`
	Count    int    `json:"count" validate:"omitempty,min=1,max=100"`
	Seed     int64  `json:"seed" validate:"omitempty"`
}

// BulkTestRequest represents a request for bulk testing operations
type BulkTestRequest struct {
	Operations []TestOperation `json:"operations" validate:"required,min=1,max=50,dive"`
	Parallel   bool            `json:"parallel" validate:"boolean"`
	StopOnError bool           `json:"stop_on_error" validate:"boolean"`
}

// TestOperation represents a single test operation
type TestOperation struct {
	Type       string                 `json:"type" validate:"required,oneof=esi sde cache validation"`
	Endpoint   string                 `json:"endpoint" validate:"required,min=1,max=255"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Expected   interface{}            `json:"expected,omitempty"`
}

// HealthCheckRequest represents a request for detailed health checks
type HealthCheckRequest struct {
	Components []string `json:"components" validate:"omitempty,dive,oneof=esi sde cache database redis auth"`
	Deep       bool     `json:"deep" validate:"boolean"`
	Timeout    int      `json:"timeout" validate:"omitempty,min=1,max=60"`
}