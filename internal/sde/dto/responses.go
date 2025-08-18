package dto

import (
	"time"
)

// SDEStatusResponse represents the current SDE status
type SDEStatusResponse struct {
	CurrentHash   string    `json:"current_hash"`
	LatestHash    string    `json:"latest_hash"`
	IsUpToDate    bool      `json:"is_up_to_date"`
	IsProcessing  bool      `json:"is_processing"`
	Progress      float64   `json:"progress"`
	LastError     string    `json:"last_error,omitempty"`
	LastCheck     time.Time `json:"last_check"`
	LastUpdate    time.Time `json:"last_update"`
	FilesProcessed int      `json:"files_processed"`
	TotalFiles    int       `json:"total_files"`
	CurrentStage  string    `json:"current_stage,omitempty"`
}

// ProgressResponse represents update progress information
type ProgressResponse struct {
	IsProcessing   bool    `json:"is_processing"`
	Progress       float64 `json:"progress"`
	Message        string  `json:"message"`
	Error          string  `json:"error,omitempty"`
	Stage          string  `json:"stage"`
	FilesProcessed int     `json:"files_processed"`
	TotalFiles     int     `json:"total_files"`
	StartTime      *time.Time `json:"start_time,omitempty"`
	EstimatedEnd   *time.Time `json:"estimated_end,omitempty"`
}

// EntityResponse represents a single SDE entity
type EntityResponse struct {
	Type string      `json:"type"`
	ID   string      `json:"id"`
	Data interface{} `json:"data"`
}

// EntitiesResponse represents multiple SDE entities of the same type
type EntitiesResponse struct {
	Type     string                 `json:"type"`
	Count    int                    `json:"count"`
	Entities map[string]interface{} `json:"entities"`
}

// SearchSolarSystemResponse represents solar system search results
type SearchSolarSystemResponse struct {
	Query   string                  `json:"query"`
	Count   int                     `json:"count"`
	Results []SolarSystemResult     `json:"results"`
}

// SolarSystemResult represents a single solar system search result
type SolarSystemResult struct {
	SystemName       string  `json:"systemName"`
	Region           string  `json:"region"`
	Constellation    string  `json:"constellation"`
	UniverseType     string  `json:"universeType"`
	RedisKey         string  `json:"redisKey"`
	SolarSystemID    int     `json:"solarSystemID"`
	Security         float64 `json:"security"`
	SolarSystemNameID int    `json:"solarSystemNameID"`
}

// IndexRebuildResponse represents the result of rebuilding an index
type IndexRebuildResponse struct {
	Message     string        `json:"message"`
	IndexType   string        `json:"index_type"`
	Duration    time.Duration `json:"duration_ms"`
	ItemsCount  int           `json:"items_count"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
}

// CheckUpdateResponse represents the result of checking for updates
type CheckUpdateResponse struct {
	UpdateAvailable bool      `json:"update_available"`
	CurrentHash     string    `json:"current_hash"`
	LatestHash      string    `json:"latest_hash"`
	LastCheck       time.Time `json:"last_check"`
	Message         string    `json:"message"`
}

// UpdateResponse represents the result of starting an update
type UpdateResponse struct {
	Started   bool      `json:"started"`
	Message   string    `json:"message"`
	StartTime time.Time `json:"start_time"`
	Error     string    `json:"error,omitempty"`
}

// TestVerifyResponse represents test verification results
type TestVerifyResponse struct {
	Success      bool                   `json:"success"`
	TestedKeys   []string               `json:"tested_keys"`
	Results      map[string]interface{} `json:"results"`
	FailedKeys   []string               `json:"failed_keys,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// ConfigResponse represents SDE configuration
type ConfigResponse struct {
	AutoCheckEnabled   bool          `json:"auto_check_enabled"`
	AutoUpdateEnabled  bool          `json:"auto_update_enabled"`
	CheckInterval      time.Duration `json:"check_interval"`
	NotifyOnUpdate     bool          `json:"notify_on_update"`
	RetainHistoryDays  int           `json:"retain_history_days"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	DownloadTimeout    time.Duration `json:"download_timeout"`
	ProcessingTimeout  time.Duration `json:"processing_timeout"`
	LastUpdated        time.Time     `json:"last_updated"`
}

// HistoryResponse represents SDE update history
type HistoryResponse struct {
	Updates    []UpdateHistoryEntry    `json:"updates"`
	Pagination SDEPaginationResponse  `json:"pagination"`
}

// UpdateHistoryEntry represents a single update history entry
type UpdateHistoryEntry struct {
	ID           string    `json:"id"`
	Hash         string    `json:"hash"`
	PreviousHash string    `json:"previous_hash"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     string    `json:"duration"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	FilesUpdated int       `json:"files_updated"`
	SizeBytes    int64     `json:"size_bytes"`
}

// NotificationResponse represents SDE notifications
type SDENotificationResponse struct {
	Notifications []NotificationEntry    `json:"notifications"`
	Pagination    SDEPaginationResponse `json:"pagination"`
}

// NotificationEntry represents a single notification
type NotificationEntry struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	IsRead    bool                   `json:"is_read"`
	CreatedAt time.Time              `json:"created_at"`
}

// PaginationResponse represents pagination information
type SDEPaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// BulkEntityResponse represents multiple entities from different types
type BulkEntityResponse struct {
	Entities []EntityResponse `json:"entities"`
	Found    int              `json:"found"`
	NotFound []EntityIdentifier `json:"not_found,omitempty"`
}


// StatisticsResponse represents SDE statistics
type StatisticsResponse struct {
	TotalEntities     int               `json:"total_entities"`
	EntitiesByType    map[string]int    `json:"entities_by_type"`
	LastUpdate        time.Time         `json:"last_update"`
	DataSize          int64             `json:"data_size_bytes"`
	IndexSize         int64             `json:"index_size_bytes"`
	ProcessingStats   ProcessingStats   `json:"processing_stats"`
	PerformanceStats  PerformanceStats  `json:"performance_stats"`
}

// ProcessingStats represents processing statistics
type ProcessingStats struct {
	TotalUpdates      int           `json:"total_updates"`
	SuccessfulUpdates int           `json:"successful_updates"`
	FailedUpdates     int           `json:"failed_updates"`
	AverageUpdateTime time.Duration `json:"average_update_time"`
	LastUpdateTime    time.Duration `json:"last_update_time"`
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	AverageSearchTime   time.Duration `json:"average_search_time"`
	AverageEntityAccess time.Duration `json:"average_entity_access"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	TotalRequests       int64         `json:"total_requests"`
}

// SDEHealthResponse represents module health information
type SDEHealthResponse struct {
	Status    string    `json:"status"`
	Module    string    `json:"module"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Checks    []SDEHealthCheck `json:"checks"`
}

// SDEHealthCheck represents an individual health check
type SDEHealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}