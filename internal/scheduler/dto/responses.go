package dto

import (
	"time"

	"go-falcon/internal/scheduler/models"
)

// TaskResponse represents a task in API responses
type TaskResponse struct {
	ID           string                    `json:"id"`
	Name         string                    `json:"name"`
	Description  string                    `json:"description"`
	Type         models.TaskType           `json:"type"`
	Schedule     string                    `json:"schedule"`
	Status       models.TaskStatus         `json:"status"`
	Priority     models.TaskPriority       `json:"priority"`
	Enabled      bool                      `json:"enabled"`
	Config       map[string]interface{}    `json:"config"`
	Metadata     models.TaskMetadata       `json:"metadata"`
	LastRun      *time.Time                `json:"last_run,omitempty"`
	NextRun      *time.Time                `json:"next_run,omitempty"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	CreatedBy    string                    `json:"created_by,omitempty"`
	UpdatedBy    string                    `json:"updated_by,omitempty"`
}

// TaskListResponse represents a paginated list of tasks
type TaskListResponse struct {
	Tasks      []TaskResponse `json:"tasks"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// TaskExecutionResponse represents task execution result
type TaskExecutionResponse struct {
	ExecutionID string    `json:"execution_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	StartedAt   time.Time `json:"started_at"`
}

// ExecutionResponse represents a task execution in API responses
type ExecutionResponse struct {
	ID           string                 `json:"id"`
	TaskID       string                 `json:"task_id"`
	Status       models.TaskStatus      `json:"status"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Output       string                 `json:"output,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	WorkerID     string                 `json:"worker_id"`
	RetryCount   int                    `json:"retry_count"`
}

// ExecutionListResponse represents a paginated list of task executions
type ExecutionListResponse struct {
	Executions []ExecutionResponse `json:"executions"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// SchedulerStatsResponse represents scheduler statistics
type SchedulerStatsResponse struct {
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

// SchedulerStatusResponse represents scheduler status
type SchedulerStatusResponse struct {
	Module  string `json:"module"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Engine  bool   `json:"engine"`
}

// BulkOperationResponse represents the result of a bulk operation
type BulkOperationResponse struct {
	Success     []string `json:"success"`
	Failed      []string `json:"failed"`
	Total       int      `json:"total"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
}

// TaskImportResponse represents the result of a task import operation
type TaskImportResponse struct {
	ImportedTasks []string `json:"imported_tasks"`
	SkippedTasks  []string `json:"skipped_tasks"`
	FailedTasks   []string `json:"failed_tasks"`
	Total         int      `json:"total"`
	Imported      int      `json:"imported"`
	Skipped       int      `json:"skipped"`
	Failed        int      `json:"failed"`
}