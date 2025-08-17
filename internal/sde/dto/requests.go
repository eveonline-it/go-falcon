package dto

import (
	"time"
)

// UpdateRequest represents a request to start SDE update
type UpdateRequest struct {
	ForceUpdate bool `json:"force_update" validate:"boolean"`
}

// EntityRequest represents a request to get a specific entity
type EntityRequest struct {
	Type string `json:"type" validate:"required,min=1,max=50"`
	ID   string `json:"id" validate:"required,min=1,max=100"`
}

// EntitiesRequest represents a request to get entities by type
type EntitiesRequest struct {
	Type string `json:"type" validate:"required,min=1,max=50"`
}

// SearchSolarSystemRequest represents a request to search solar systems
type SearchSolarSystemRequest struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// RebuildIndexRequest represents a request to rebuild search indexes
type RebuildIndexRequest struct {
	IndexType string `json:"index_type" validate:"omitempty,oneof=solarsystems"`
}

// CheckUpdateRequest represents a request to check for SDE updates
type CheckUpdateRequest struct {
	AutoUpdate bool `json:"auto_update" validate:"boolean"`
	Notify     bool `json:"notify" validate:"boolean"`
}

// TestStoreSampleRequest represents a request for test data storage
type TestStoreSampleRequest struct {
	TestData map[string]interface{} `json:"test_data,omitempty"`
	Type     string                 `json:"type" validate:"omitempty,min=1,max=50"`
}

// ConfigUpdateRequest represents a request to update SDE configuration
type ConfigUpdateRequest struct {
	AutoCheckEnabled   *bool          `json:"auto_check_enabled,omitempty"`
	AutoUpdateEnabled  *bool          `json:"auto_update_enabled,omitempty"`
	CheckInterval      *time.Duration `json:"check_interval,omitempty" validate:"omitempty,min=1h"`
	NotifyOnUpdate     *bool          `json:"notify_on_update,omitempty"`
	RetainHistoryDays  *int           `json:"retain_history_days,omitempty" validate:"omitempty,min=1,max=365"`
	MaxRetries         *int           `json:"max_retries,omitempty" validate:"omitempty,min=1,max=10"`
	RetryDelay         *time.Duration `json:"retry_delay,omitempty" validate:"omitempty,min=1s"`
	DownloadTimeout    *time.Duration `json:"download_timeout,omitempty" validate:"omitempty,min=30s"`
	ProcessingTimeout  *time.Duration `json:"processing_timeout,omitempty" validate:"omitempty,min=1m"`
}

// HistoryQueryRequest represents a request to query SDE update history
type HistoryQueryRequest struct {
	Page      int       `json:"page" validate:"omitempty,min=1"`
	PageSize  int       `json:"page_size" validate:"omitempty,min=1,max=100"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Success   *bool     `json:"success,omitempty"`
}

// NotificationQueryRequest represents a request to query SDE notifications
type NotificationQueryRequest struct {
	Page       int    `json:"page" validate:"omitempty,min=1"`
	PageSize   int    `json:"page_size" validate:"omitempty,min=1,max=100"`
	Type       string `json:"type" validate:"omitempty,oneof=update_available update_started update_completed update_failed"`
	IsRead     *bool  `json:"is_read,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
}

// MarkNotificationReadRequest represents a request to mark notifications as read
type MarkNotificationReadRequest struct {
	NotificationIDs []string `json:"notification_ids" validate:"required,min=1,dive,required"`
}

// BulkEntityRequest represents a request to get multiple entities
type BulkEntityRequest struct {
	Entities []EntityIdentifier `json:"entities" validate:"required,min=1,dive"`
}

// EntityIdentifier represents an entity identifier
type EntityIdentifier struct {
	Type string `json:"type" validate:"required,min=1,max=50"`
	ID   string `json:"id" validate:"required,min=1,max=100"`
}