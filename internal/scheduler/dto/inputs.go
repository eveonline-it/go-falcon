package dto

import (
	"go-falcon/internal/scheduler/models"
)

// =============================================================================
// HUMA INPUT DTOs (consolidated from huma_requests.go)
// =============================================================================

// TaskCreateRequest represents a request to create a new task
type TaskCreateRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	Description string                 `json:"description" validate:"max=500"`
	Type        models.TaskType        `json:"type" validate:"required,oneof=http function system custom"`
	Schedule    string                 `json:"schedule" validate:"required,cron"`
	Priority    models.TaskPriority    `json:"priority" validate:"oneof=low normal high critical"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config" validate:"required"`
	Metadata    *models.TaskMetadata   `json:"metadata,omitempty"`
	Tags        []string               `json:"tags"`
}

// TaskUpdateRequest represents a request to update a task
type TaskUpdateRequest struct {
	Name        *string                `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string                `json:"description,omitempty" validate:"omitempty,max=500"`
	Schedule    *string                `json:"schedule,omitempty" validate:"omitempty,cron"`
	Priority    *models.TaskPriority   `json:"priority,omitempty" validate:"omitempty,oneof=low normal high critical"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// TaskListQuery represents query parameters for listing tasks
type TaskListQuery struct {
	Page     int      `query:"page" validate:"min=1"`
	PageSize int      `query:"page_size" validate:"min=1,max=100"`
	Status   string   `query:"status" validate:"omitempty,oneof=pending running completed failed paused disabled"`
	Type     string   `query:"type" validate:"omitempty,oneof=http function system custom"`
	Enabled  string   `query:"enabled" validate:"omitempty,oneof=true false"`
	Tags     []string `query:"tags"`
}

// TaskExecutionQuery represents query parameters for task execution history
type TaskExecutionQuery struct {
	Page     int `query:"page" validate:"min=1"`
	PageSize int `query:"page_size" validate:"min=1,max=100"`
}

// ManualExecutionRequest represents a request to manually execute a task
type ManualExecutionRequest struct {
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Priority   *models.TaskPriority   `json:"priority,omitempty" validate:"omitempty,oneof=low normal high critical"`
}

// BulkTaskOperationRequest represents a request for bulk operations on tasks
type BulkTaskOperationRequest struct {
	TaskIDs   []string `json:"task_ids" validate:"required,min=1"`
	Operation string   `json:"operation" validate:"required,oneof=enable disable pause resume delete"`
}

// TaskImportRequest represents a request to import tasks
type TaskImportRequest struct {
	Tasks     []TaskCreateRequest `json:"tasks" validate:"required,min=1"`
	Overwrite bool               `json:"overwrite"`
}

// TaskCreateInput represents the input for creating a new task
type TaskCreateInput struct {
	Body TaskCreateRequest `json:"body"`
}

// TaskUpdateInput represents the input for updating a task
type TaskUpdateInput struct {
	TaskID string            `path:"task_id" validate:"required" doc:"Task ID to update"`
	Body   TaskUpdateRequest `json:"body"`
}

// TaskGetInput represents the input for getting a single task
type TaskGetInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to retrieve"`
}

// TaskDeleteInput represents the input for deleting a task
type TaskDeleteInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to delete"`
}

// TaskListInput represents the input for listing tasks
type TaskListInput struct {
	Page     int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
	Status   string `query:"status" validate:"omitempty,oneof=pending running completed failed paused disabled" doc:"Filter by task status"`
	Type     string `query:"type" validate:"omitempty,oneof=http function system custom" doc:"Filter by task type"`
	Enabled  string `query:"enabled" validate:"omitempty,oneof=true false" doc:"Filter by enabled status"`
	Tags     string `query:"tags" doc:"Comma-separated list of tags to filter by"`
}

// TaskExecuteInput represents the input for manually executing a task
type TaskExecuteInput struct {
	TaskID string                 `path:"task_id" validate:"required" doc:"Task ID to execute"`
	Body   ManualExecutionRequest `json:"body"`
}

// TaskExecutionHistoryInput represents the input for getting task execution history
type TaskExecutionHistoryInput struct {
	TaskID   string `path:"task_id" validate:"required" doc:"Task ID to get execution history for"`
	Page     int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
}

// BulkTaskOperationInput represents the input for bulk task operations
type BulkTaskOperationInput struct {
	Body BulkTaskOperationRequest `json:"body"`
}

// TaskImportInput represents the input for importing tasks
type TaskImportInput struct {
	Body TaskImportRequest `json:"body"`
}

// SchedulerStatsInput represents the input for getting scheduler statistics (no body needed)
type SchedulerStatsInput struct {
	// No parameters needed
}

// SchedulerStatusInput represents the input for getting scheduler status (no body needed)
type SchedulerStatusInput struct {
	// No parameters needed
}

// ExecutionGetInput represents the input for getting a single execution
type ExecutionGetInput struct {
	ExecutionID string `path:"execution_id" validate:"required" doc:"Execution ID to retrieve"`
}

// ExecutionListInput represents the input for listing executions across all tasks
type ExecutionListInput struct {
	Page     int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
	Status   string `query:"status" validate:"omitempty,oneof=pending running completed failed" doc:"Filter by execution status"`
	TaskID   string `query:"task_id" doc:"Filter by specific task ID"`
}

// TaskEnableInput represents the input for enabling a task
type TaskEnableInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to enable"`
}

// TaskDisableInput represents the input for disabling a task
type TaskDisableInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to disable"`
}

// TaskPauseInput represents the input for pausing a task
type TaskPauseInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to pause"`
}

// TaskResumeInput represents the input for resuming a task
type TaskResumeInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to resume"`
}