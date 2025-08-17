package models

import (
	"time"
)

// TaskType defines the type of task to execute
type TaskType string

const (
	TaskTypeHTTP     TaskType = "http"
	TaskTypeFunction TaskType = "function"
	TaskTypeSystem   TaskType = "system"
	TaskTypeCustom   TaskType = "custom"
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusDisabled  TaskStatus = "disabled"
)

// TaskPriority defines task execution priority
type TaskPriority string

const (
	TaskPriorityLow      TaskPriority = "low"
	TaskPriorityNormal   TaskPriority = "normal"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityCritical TaskPriority = "critical"
)

// Task represents a scheduled task definition
type Task struct {
	ID           string                 `json:"id" bson:"_id"`
	Name         string                 `json:"name" bson:"name"`
	Description  string                 `json:"description" bson:"description"`
	Type         TaskType               `json:"type" bson:"type"`
	Schedule     string                 `json:"schedule" bson:"schedule"` // Cron format
	Status       TaskStatus             `json:"status" bson:"status"`
	Priority     TaskPriority           `json:"priority" bson:"priority"`
	Enabled      bool                   `json:"enabled" bson:"enabled"`
	Config       map[string]interface{} `json:"config" bson:"config"`
	Metadata     TaskMetadata           `json:"metadata" bson:"metadata"`
	LastRun      *time.Time             `json:"last_run,omitempty" bson:"last_run,omitempty"`
	NextRun      *time.Time             `json:"next_run,omitempty" bson:"next_run,omitempty"`
	CreatedAt    time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" bson:"updated_at"`
	CreatedBy    string                 `json:"created_by,omitempty" bson:"created_by,omitempty"`
	UpdatedBy    string                 `json:"updated_by,omitempty" bson:"updated_by,omitempty"`
}

// TaskMetadata contains additional task information
type TaskMetadata struct {
	MaxRetries     int           `json:"max_retries" bson:"max_retries"`
	RetryInterval  time.Duration `json:"retry_interval" bson:"retry_interval"`
	Timeout        time.Duration `json:"timeout" bson:"timeout"`
	Tags           []string      `json:"tags" bson:"tags"`
	IsSystem       bool          `json:"is_system" bson:"is_system"`
	Source         string        `json:"source" bson:"source"` // "system", "api", "import"
	Version        int           `json:"version" bson:"version"`
	LastError      string        `json:"last_error,omitempty" bson:"last_error,omitempty"`
	SuccessCount   int64         `json:"success_count" bson:"success_count"`
	FailureCount   int64         `json:"failure_count" bson:"failure_count"`
	TotalRuns      int64         `json:"total_runs" bson:"total_runs"`
	AverageRuntime time.Duration `json:"average_runtime" bson:"average_runtime"`
}

// TaskExecution represents a single task execution record
type TaskExecution struct {
	ID           string                 `json:"id" bson:"_id"`
	TaskID       string                 `json:"task_id" bson:"task_id"`
	Status       TaskStatus             `json:"status" bson:"status"`
	StartedAt    time.Time              `json:"started_at" bson:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	Duration     time.Duration          `json:"duration" bson:"duration"`
	Output       string                 `json:"output,omitempty" bson:"output,omitempty"`
	Error        string                 `json:"error,omitempty" bson:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata" bson:"metadata"`
	WorkerID     string                 `json:"worker_id" bson:"worker_id"`
	RetryCount   int                    `json:"retry_count" bson:"retry_count"`
}

// HTTPTaskConfig defines configuration for HTTP tasks
type HTTPTaskConfig struct {
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body,omitempty"`
	ExpectedCode   int               `json:"expected_code"`
	Timeout        time.Duration     `json:"timeout"`
	FollowRedirect bool              `json:"follow_redirect"`
	ValidateSSL    bool              `json:"validate_ssl"`
}

// FunctionTaskConfig defines configuration for function tasks
type FunctionTaskConfig struct {
	FunctionName string                 `json:"function_name"`
	Parameters   map[string]interface{} `json:"parameters"`
	Module       string                 `json:"module,omitempty"`
}

// SystemTaskConfig defines configuration for system tasks
type SystemTaskConfig struct {
	TaskName   string                 `json:"task_name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	Success   bool                   `json:"success"`
	Output    string                 `json:"output"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Retryable bool                   `json:"retryable"`
}

// SchedulerStats represents scheduler statistics
type SchedulerStats struct {
	TotalTasks       int64      `json:"total_tasks"`
	EnabledTasks     int64      `json:"enabled_tasks"`
	RunningTasks     int64      `json:"running_tasks"`
	CompletedToday   int64      `json:"completed_today"`
	FailedToday      int64      `json:"failed_today"`
	AverageRuntime   string     `json:"average_runtime"`
	NextScheduledRun *time.Time `json:"next_scheduled_run,omitempty"`
	WorkerCount      int        `json:"worker_count"`
	QueueSize        int        `json:"queue_size"`
}

// EngineStats represents engine statistics
type EngineStats struct {
	WorkerCount int `json:"worker_count"`
	QueueSize   int `json:"queue_size"`
	IsRunning   bool `json:"is_running"`
}

// SystemTaskDefinition represents metadata about a system task
type SystemTaskDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schedule    string `json:"schedule"`
	Purpose     string `json:"purpose"`
	Priority    string `json:"priority"`
}