package dto

import (
	"go-falcon/internal/scheduler/models"
)

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