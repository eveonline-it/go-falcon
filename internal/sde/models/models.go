package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SDEStatus represents the current SDE status in the system
type SDEStatus struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CurrentHash    string             `bson:"current_hash" json:"current_hash"`
	LatestHash     string             `bson:"latest_hash" json:"latest_hash"`
	IsUpToDate     bool               `bson:"is_up_to_date" json:"is_up_to_date"`
	IsProcessing   bool               `bson:"is_processing" json:"is_processing"`
	Progress       float64            `bson:"progress" json:"progress"`
	LastError      string             `bson:"last_error,omitempty" json:"last_error,omitempty"`
	LastCheck      time.Time          `bson:"last_check" json:"last_check"`
	LastUpdate     time.Time          `bson:"last_update" json:"last_update"`
	FilesProcessed int                `bson:"files_processed" json:"files_processed"`
	TotalFiles     int                `bson:"total_files" json:"total_files"`
	CurrentStage   string             `bson:"current_stage,omitempty" json:"current_stage,omitempty"`
	CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
}

// SDEUpdateHistory represents the history of SDE updates
type SDEUpdateHistory struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Hash         string             `bson:"hash" json:"hash"`
	PreviousHash string             `bson:"previous_hash" json:"previous_hash"`
	StartTime    time.Time          `bson:"start_time" json:"start_time"`
	EndTime      time.Time          `bson:"end_time" json:"end_time"`
	Duration     time.Duration      `bson:"duration" json:"duration"`
	Success      bool               `bson:"success" json:"success"`
	Error        string             `bson:"error,omitempty" json:"error,omitempty"`
	FilesUpdated int                `bson:"files_updated" json:"files_updated"`
	SizeBytes    int64              `bson:"size_bytes" json:"size_bytes"`
	Stages       []UpdateStage      `bson:"stages" json:"stages"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
}

// UpdateStage represents a stage in the update process
type UpdateStage struct {
	Name      string        `bson:"name" json:"name"`
	StartTime time.Time     `bson:"start_time" json:"start_time"`
	EndTime   *time.Time    `bson:"end_time,omitempty" json:"end_time,omitempty"`
	Duration  time.Duration `bson:"duration" json:"duration"`
	Success   bool          `bson:"success" json:"success"`
	Error     string        `bson:"error,omitempty" json:"error,omitempty"`
	Progress  float64       `bson:"progress" json:"progress"`
	Message   string        `bson:"message,omitempty" json:"message,omitempty"`
}

// SDENotification represents a notification about SDE updates
type SDENotification struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Type      string                 `bson:"type" json:"type"` // "update_available", "update_started", "update_completed", "update_failed"
	Title     string                 `bson:"title" json:"title"`
	Message   string                 `bson:"message" json:"message"`
	Data      map[string]interface{} `bson:"data" json:"data"`
	IsRead    bool                   `bson:"is_read" json:"is_read"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	ReadAt    *time.Time             `bson:"read_at,omitempty" json:"read_at,omitempty"`
}

// SDEConfig represents the SDE module configuration
type SDEConfig struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AutoCheckEnabled  bool               `bson:"auto_check_enabled" json:"auto_check_enabled"`
	AutoUpdateEnabled bool               `bson:"auto_update_enabled" json:"auto_update_enabled"`
	CheckInterval     time.Duration      `bson:"check_interval" json:"check_interval"`
	NotifyOnUpdate    bool               `bson:"notify_on_update" json:"notify_on_update"`
	RetainHistoryDays int                `bson:"retain_history_days" json:"retain_history_days"`
	MaxRetries        int                `bson:"max_retries" json:"max_retries"`
	RetryDelay        time.Duration      `bson:"retry_delay" json:"retry_delay"`
	DownloadTimeout   time.Duration      `bson:"download_timeout" json:"download_timeout"`
	ProcessingTimeout time.Duration      `bson:"processing_timeout" json:"processing_timeout"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

// SDEEntity represents a generic SDE entity
type SDEEntity struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	RedisKey  string                 `json:"redis_key"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SolarSystemSearchResult represents a solar system search result
type SolarSystemSearchResult struct {
	SystemName        string  `json:"systemName"`
	Region            string  `json:"region"`
	Constellation     string  `json:"constellation"`
	UniverseType      string  `json:"universeType"`
	RedisKey          string  `json:"redisKey"`
	SolarSystemID     int     `json:"solarSystemID"`
	Security          float64 `json:"security"`
	SolarSystemNameID int     `json:"solarSystemNameID"`
}

// ProcessingStats represents statistics about SDE processing
type ProcessingStats struct {
	TotalUpdates      int           `bson:"total_updates" json:"total_updates"`
	SuccessfulUpdates int           `bson:"successful_updates" json:"successful_updates"`
	FailedUpdates     int           `bson:"failed_updates" json:"failed_updates"`
	AverageUpdateTime time.Duration `bson:"average_update_time" json:"average_update_time"`
	LastUpdateTime    time.Duration `bson:"last_update_time" json:"last_update_time"`
	TotalDownloadSize int64         `bson:"total_download_size" json:"total_download_size"`
	TotalProcessedFiles int         `bson:"total_processed_files" json:"total_processed_files"`
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	AverageSearchTime   time.Duration `bson:"average_search_time" json:"average_search_time"`
	AverageEntityAccess time.Duration `bson:"average_entity_access" json:"average_entity_access"`
	CacheHitRate        float64       `bson:"cache_hit_rate" json:"cache_hit_rate"`
	TotalRequests       int64         `bson:"total_requests" json:"total_requests"`
	RequestsPerSecond   float64       `bson:"requests_per_second" json:"requests_per_second"`
	ErrorRate           float64       `bson:"error_rate" json:"error_rate"`
}

// SDEStatistics represents comprehensive SDE statistics
type SDEStatistics struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TotalEntities    int                `bson:"total_entities" json:"total_entities"`
	EntitiesByType   map[string]int     `bson:"entities_by_type" json:"entities_by_type"`
	LastUpdate       time.Time          `bson:"last_update" json:"last_update"`
	DataSize         int64              `bson:"data_size_bytes" json:"data_size_bytes"`
	IndexSize        int64              `bson:"index_size_bytes" json:"index_size_bytes"`
	ProcessingStats  ProcessingStats    `bson:"processing_stats" json:"processing_stats"`
	PerformanceStats PerformanceStats   `bson:"performance_stats" json:"performance_stats"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

// SDEIndex represents a search index
type SDEIndex struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type       string             `bson:"type" json:"type"` // solarsystems, types, etc.
	Name       string             `bson:"name" json:"name"`
	RedisKey   string             `bson:"redis_key" json:"redis_key"`
	ItemCount  int                `bson:"item_count" json:"item_count"`
	LastBuilt  time.Time          `bson:"last_built" json:"last_built"`
	BuildTime  time.Duration      `bson:"build_time" json:"build_time"`
	IsActive   bool               `bson:"is_active" json:"is_active"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

// ProgressState represents the current processing state
type ProgressState struct {
	IsProcessing   bool      `json:"is_processing"`
	Progress       float64   `json:"progress"`
	Message        string    `json:"message"`
	Error          string    `json:"error,omitempty"`
	Stage          string    `json:"stage"`
	FilesProcessed int       `json:"files_processed"`
	TotalFiles     int       `json:"total_files"`
	StartTime      time.Time `json:"start_time"`
	EstimatedEnd   *time.Time `json:"estimated_end,omitempty"`
}

// Constants for notification types
const (
	NotificationTypeUpdateAvailable = "update_available"
	NotificationTypeUpdateStarted   = "update_started"
	NotificationTypeUpdateCompleted = "update_completed"
	NotificationTypeUpdateFailed    = "update_failed"
	NotificationTypeIndexRebuilt    = "index_rebuilt"
	NotificationTypeMaintenance     = "maintenance"
)

// Constants for update stages
const (
	StageDownload   = "download"
	StageExtract    = "extract"
	StageConvert    = "convert"
	StageStore      = "store"
	StageIndex      = "index"
	StageCleanup    = "cleanup"
	StageComplete   = "complete"
)

// Constants for index types
const (
	IndexTypeSolarSystems   = "solarsystems"
	IndexTypeRegions        = "regions"
	IndexTypeConstellations = "constellations"
	IndexTypeTypes          = "types"
	IndexTypeAgents         = "agents"
	IndexTypeCorporations   = "corporations"
)

// Constants for Redis keys
const (
	RedisKeyCurrentHash     = "sde:current_hash"
	RedisKeyStatus          = "sde:status"
	RedisKeyProgress        = "sde:progress"
	RedisKeySolarSystemIndex = "sde:index:solarsystems"
	RedisKeyConfig          = "sde:config"
	RedisKeyStatistics      = "sde:statistics"
)

// Constants for SDE URLs
const (
	SDEDownloadURL = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip"
	SDEHashURL     = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/checksum"
)

// Default configuration values
const (
	DefaultCheckInterval      = 6 * time.Hour
	DefaultRetainHistoryDays  = 30
	DefaultMaxRetries         = 3
	DefaultRetryDelay         = 5 * time.Minute
	DefaultDownloadTimeout    = 30 * time.Minute
	DefaultProcessingTimeout  = 60 * time.Minute
)