package sde

import "time"

// SDEUpdateTask represents a scheduled SDE update task
type SDEUpdateTask struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Type        string    `bson:"type" json:"type"`
	Schedule    string    `bson:"schedule" json:"schedule"`
	LastRun     time.Time `bson:"last_run" json:"last_run"`
	NextRun     time.Time `bson:"next_run" json:"next_run"`
	IsEnabled   bool      `bson:"is_enabled" json:"is_enabled"`
	AutoUpdate  bool      `bson:"auto_update" json:"auto_update"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

// SDEUpdateHistory represents the history of SDE updates
type SDEUpdateHistory struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Hash        string    `bson:"hash" json:"hash"`
	PreviousHash string   `bson:"previous_hash" json:"previous_hash"`
	StartTime   time.Time `bson:"start_time" json:"start_time"`
	EndTime     time.Time `bson:"end_time" json:"end_time"`
	Duration    string    `bson:"duration" json:"duration"`
	Success     bool      `bson:"success" json:"success"`
	Error       string    `bson:"error,omitempty" json:"error,omitempty"`
	FilesUpdated int      `bson:"files_updated" json:"files_updated"`
	SizeBytes   int64     `bson:"size_bytes" json:"size_bytes"`
}

// SDENotification represents a notification about SDE updates
type SDENotification struct {
	ID          string                 `bson:"_id,omitempty" json:"id"`
	Type        string                 `bson:"type" json:"type"` // "update_available", "update_started", "update_completed", "update_failed"
	Title       string                 `bson:"title" json:"title"`
	Message     string                 `bson:"message" json:"message"`
	Data        map[string]interface{} `bson:"data" json:"data"`
	IsRead      bool                   `bson:"is_read" json:"is_read"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
}

// SDEConfig represents the SDE module configuration
type SDEConfig struct {
	AutoCheckEnabled   bool          `json:"auto_check_enabled"`
	AutoUpdateEnabled  bool          `json:"auto_update_enabled"`
	CheckInterval      time.Duration `json:"check_interval"`
	NotifyOnUpdate     bool          `json:"notify_on_update"`
	RetainHistoryDays  int           `json:"retain_history_days"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	DownloadTimeout    time.Duration `json:"download_timeout"`
	ProcessingTimeout  time.Duration `json:"processing_timeout"`
}