package dto

import (
	"time"

	"go-falcon/internal/scheduler/models"
)

// TaskResponse represents a task in API responses
type TaskResponse struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Type            models.TaskType        `json:"type"`
	Schedule        string                 `json:"schedule"`
	Status          models.TaskStatus      `json:"status"`
	Priority        models.TaskPriority    `json:"priority"`
	Enabled         bool                   `json:"enabled"`
	Config          map[string]interface{} `json:"config"`
	Metadata        models.TaskMetadata    `json:"metadata"`
	LastRun         *time.Time             `json:"last_run,omitempty"`
	LastRunDuration *models.Duration       `json:"last_run_duration,omitempty"`
	NextRun         *time.Time             `json:"next_run,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by,omitempty"`
	UpdatedBy       string                 `json:"updated_by,omitempty"`
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
	ID          string                 `json:"id"`
	TaskID      string                 `json:"task_id"`
	Status      models.TaskStatus      `json:"status"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    models.Duration        `json:"duration"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	WorkerID    string                 `json:"worker_id"`
	RetryCount  int                    `json:"retry_count"`
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
	Success      []string `json:"success"`
	Failed       []string `json:"failed"`
	Total        int      `json:"total"`
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
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

// =============================================================================
// HUMA OUTPUT DTOs (consolidated from huma_requests.go)
// =============================================================================

// TaskCreateOutput represents the output for creating a new task
type TaskCreateOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskUpdateOutput represents the output for updating a task
type TaskUpdateOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskGetOutput represents the output for getting a single task
type TaskGetOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskDeleteOutput represents the output for deleting a task
type TaskDeleteOutput struct {
	Body map[string]interface{} `json:"body"`
}

// TaskListOutput represents the output for listing tasks
type TaskListOutput struct {
	Body TaskListResponse `json:"body"`
}

// TaskExecuteOutput represents the output for manually executing a task
type TaskExecuteOutput struct {
	Body TaskExecutionResponse `json:"body"`
}

// TaskExecutionHistoryOutput represents the output for getting task execution history
type TaskExecutionHistoryOutput struct {
	Body ExecutionListResponse `json:"body"`
}

// BulkTaskOperationOutput represents the output for bulk task operations
type BulkTaskOperationOutput struct {
	Body BulkOperationResponse `json:"body"`
}

// TaskImportOutput represents the output for importing tasks
type TaskImportOutput struct {
	Body TaskImportResponse `json:"body"`
}

// SchedulerStatsOutput represents the output for getting scheduler statistics
type SchedulerStatsOutput struct {
	Body SchedulerStatsResponse `json:"body"`
}

// SchedulerStatusOutput represents the output for getting scheduler status
type SchedulerStatusOutput struct {
	Body SchedulerStatusResponse `json:"body"`
}

// ExecutionGetOutput represents the output for getting a single execution
type ExecutionGetOutput struct {
	Body ExecutionResponse `json:"body"`
}

// ExecutionListOutput represents the output for listing executions
type ExecutionListOutput struct {
	Body ExecutionListResponse `json:"body"`
}

// TaskEnableOutput represents the output for enabling a task
type TaskEnableOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskDisableOutput represents the output for disabling a task
type TaskDisableOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskPauseOutput represents the output for pausing a task
type TaskPauseOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskResumeOutput represents the output for resuming a task
type TaskResumeOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskStopOutput represents the output for stopping a task
type TaskStopOutput struct {
	Body map[string]interface{} `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body SchedulerModuleStatusResponse `json:"body"`
}

// SchedulerModuleStatusResponse represents the actual status response data
type SchedulerModuleStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}
