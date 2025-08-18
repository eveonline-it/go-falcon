package dto

// Import removed as DTOs are defined in this package

// TaskCreateInput represents the input for creating a new task
type TaskCreateInput struct {
	Body TaskCreateRequest `json:"body"`
}

// TaskCreateOutput represents the output for creating a new task
type TaskCreateOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskUpdateInput represents the input for updating a task
type TaskUpdateInput struct {
	TaskID string            `path:"task_id" validate:"required" doc:"Task ID to update"`
	Body   TaskUpdateRequest `json:"body"`
}

// TaskUpdateOutput represents the output for updating a task
type TaskUpdateOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskGetInput represents the input for getting a single task
type TaskGetInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to retrieve"`
}

// TaskGetOutput represents the output for getting a single task
type TaskGetOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskDeleteInput represents the input for deleting a task
type TaskDeleteInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to delete"`
}

// TaskDeleteOutput represents the output for deleting a task
type TaskDeleteOutput struct {
	Body map[string]interface{} `json:"body"`
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

// TaskListOutput represents the output for listing tasks
type TaskListOutput struct {
	Body TaskListResponse `json:"body"`
}

// TaskExecuteInput represents the input for manually executing a task
type TaskExecuteInput struct {
	TaskID string                 `path:"task_id" validate:"required" doc:"Task ID to execute"`
	Body   ManualExecutionRequest `json:"body"`
}

// TaskExecuteOutput represents the output for manually executing a task
type TaskExecuteOutput struct {
	Body TaskExecutionResponse `json:"body"`
}

// TaskExecutionHistoryInput represents the input for getting task execution history
type TaskExecutionHistoryInput struct {
	TaskID   string `path:"task_id" validate:"required" doc:"Task ID to get execution history for"`
	Page     int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
}

// TaskExecutionHistoryOutput represents the output for getting task execution history
type TaskExecutionHistoryOutput struct {
	Body ExecutionListResponse `json:"body"`
}

// BulkTaskOperationInput represents the input for bulk task operations
type BulkTaskOperationInput struct {
	Body BulkTaskOperationRequest `json:"body"`
}

// BulkTaskOperationOutput represents the output for bulk task operations
type BulkTaskOperationOutput struct {
	Body BulkOperationResponse `json:"body"`
}

// TaskImportInput represents the input for importing tasks
type TaskImportInput struct {
	Body TaskImportRequest `json:"body"`
}

// TaskImportOutput represents the output for importing tasks
type TaskImportOutput struct {
	Body TaskImportResponse `json:"body"`
}

// SchedulerStatsInput represents the input for getting scheduler statistics (no body needed)
type SchedulerStatsInput struct {
	// No parameters needed
}

// SchedulerStatsOutput represents the output for getting scheduler statistics
type SchedulerStatsOutput struct {
	Body SchedulerStatsResponse `json:"body"`
}

// SchedulerStatusInput represents the input for getting scheduler status (no body needed)
type SchedulerStatusInput struct {
	// No parameters needed
}

// SchedulerStatusOutput represents the output for getting scheduler status
type SchedulerStatusOutput struct {
	Body SchedulerStatusResponse `json:"body"`
}

// ExecutionGetInput represents the input for getting a single execution
type ExecutionGetInput struct {
	ExecutionID string `path:"execution_id" validate:"required" doc:"Execution ID to retrieve"`
}

// ExecutionGetOutput represents the output for getting a single execution
type ExecutionGetOutput struct {
	Body ExecutionResponse `json:"body"`
}

// ExecutionListInput represents the input for listing executions across all tasks
type ExecutionListInput struct {
	Page     int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
	Status   string `query:"status" validate:"omitempty,oneof=pending running completed failed" doc:"Filter by execution status"`
	TaskID   string `query:"task_id" doc:"Filter by specific task ID"`
}

// ExecutionListOutput represents the output for listing executions
type ExecutionListOutput struct {
	Body ExecutionListResponse `json:"body"`
}

// TaskEnableInput represents the input for enabling a task
type TaskEnableInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to enable"`
}

// TaskEnableOutput represents the output for enabling a task
type TaskEnableOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskDisableInput represents the input for disabling a task
type TaskDisableInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to disable"`
}

// TaskDisableOutput represents the output for disabling a task
type TaskDisableOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskPauseInput represents the input for pausing a task
type TaskPauseInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to pause"`
}

// TaskPauseOutput represents the output for pausing a task
type TaskPauseOutput struct {
	Body TaskResponse `json:"body"`
}

// TaskResumeInput represents the input for resuming a task
type TaskResumeInput struct {
	TaskID string `path:"task_id" validate:"required" doc:"Task ID to resume"`
}

// TaskResumeOutput represents the output for resuming a task
type TaskResumeOutput struct {
	Body TaskResponse `json:"body"`
}