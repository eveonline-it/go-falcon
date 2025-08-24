package routes

import (
	"context"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/internal/scheduler/middleware"
	"go-falcon/internal/scheduler/services"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles Huma-based HTTP routing for the Scheduler module
type Routes struct {
	service    *services.SchedulerService
	middleware *middleware.Middleware
	api        huma.API
}

// NewRoutes creates a new Huma Scheduler routes handler
func NewRoutes(service *services.SchedulerService, middleware *middleware.Middleware, router chi.Router) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Scheduler Module", "1.0.0")
	config.Info.Description = "Task scheduling and management system with cron support and distributed execution"

	api := humachi.New(router, config)

	hr := &Routes{
		service:    service,
		middleware: middleware,
		api:        api,
	}

	// Note: registerRoutes() call removed for security - it registered endpoints without authentication
	// All secure routes are now registered via RegisterSchedulerRoutes() called by RegisterUnifiedRoutes()
	// hr.registerRoutes() // REMOVED: This registered unprotected endpoints

	return hr
}

// RegisterSchedulerRoutes registers scheduler routes on a shared Huma API
func RegisterSchedulerRoutes(api huma.API, basePath string, service *services.SchedulerService, middleware *middleware.Middleware) {
	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "scheduler-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get scheduler module status",
		Description: "Returns the health status of the scheduler module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := service.GetModuleStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Legacy scheduler status endpoint
	huma.Register(api, huma.Operation{
		OperationID: "scheduler-get-scheduler-status",
		Method:      "GET",
		Path:        basePath + "/scheduler-status",
		Summary:     "Get scheduler status",
		Description: "Get current scheduler status including worker count and running tasks",
		Tags:        []string{"Scheduler / Status"},
	}, func(ctx context.Context, input *dto.SchedulerStatusInput) (*dto.SchedulerStatusOutput, error) {
		status := service.GetStatus()
		return &dto.SchedulerStatusOutput{Body: *status}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-get-stats",
		Method:      "GET",
		Path:        basePath + "/stats",
		Summary:     "Get scheduler statistics",
		Description: "Get comprehensive scheduler statistics including task counts and execution metrics",
		Tags:        []string{"Scheduler / Status"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.SchedulerStatsInput) (*dto.SchedulerStatsOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		stats, err := service.GetStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get scheduler stats", err)
		}
		return &dto.SchedulerStatsOutput{Body: *stats}, nil
	})

	// Task management endpoints (require authentication and permissions)
	huma.Register(api, huma.Operation{
		OperationID: "scheduler-list-tasks",
		Method:      "GET",
		Path:        basePath + "/tasks",
		Summary:     "List scheduled tasks",
		Description: "List all scheduled tasks with filtering and pagination support",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskListInput) (*dto.TaskListOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		// Convert Huma input to service query format
		query := &dto.TaskListQuery{
			Page:     input.Page,
			PageSize: input.PageSize,
			Status:   input.Status,
			Type:     input.Type,
			Enabled:  input.Enabled,
		}

		// Set defaults if not provided
		if query.Page == 0 {
			query.Page = 1
		}
		if query.PageSize == 0 {
			query.PageSize = 20
		}

		tasks, err := service.ListTasks(ctx, query)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list tasks", err)
		}
		return &dto.TaskListOutput{Body: *tasks}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-create-task",
		Method:      "POST",
		Path:        basePath + "/tasks",
		Summary:     "Create new task",
		Description: "Create a new scheduled task with cron-like scheduling",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskCreateInput) (*dto.TaskCreateOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		task, err := service.CreateTask(ctx, &input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to create task", err)
		}
		return &dto.TaskCreateOutput{Body: *task}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-get-task",
		Method:      "GET",
		Path:        basePath + "/tasks/{task_id}",
		Summary:     "Get task details",
		Description: "Get detailed information about a specific scheduled task",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskGetInput) (*dto.TaskGetOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		task, err := service.GetTask(ctx, input.TaskID)
		if err != nil {
			if err.Error() == "task not found" {
				return nil, huma.Error404NotFound("Task not found")
			}
			return nil, huma.Error500InternalServerError("Failed to get task", err)
		}
		return &dto.TaskGetOutput{Body: *task}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-update-task",
		Method:      "PUT",
		Path:        basePath + "/tasks/{task_id}",
		Summary:     "Update task",
		Description: "Update an existing scheduled task configuration",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskUpdateInput) (*dto.TaskUpdateOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		task, err := service.UpdateTask(ctx, input.TaskID, &input.Body)
		if err != nil {
			if err.Error() == "task not found" {
				return nil, huma.Error404NotFound("Task not found")
			}
			if err.Error() == "cannot update system tasks" {
				return nil, huma.Error403Forbidden("Cannot update system tasks")
			}
			return nil, huma.Error400BadRequest("Failed to update task", err)
		}
		return &dto.TaskUpdateOutput{Body: *task}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-delete-task",
		Method:      "DELETE",
		Path:        basePath + "/tasks/{task_id}",
		Summary:     "Delete task",
		Description: "Delete a scheduled task (system tasks cannot be deleted)",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskDeleteInput) (*dto.TaskDeleteOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		err = service.DeleteTask(ctx, input.TaskID)
		if err != nil {
			if err.Error() == "task not found" {
				return nil, huma.Error404NotFound("Task not found")
			}
			if err.Error() == "cannot delete system tasks" {
				return nil, huma.Error403Forbidden("Cannot delete system tasks")
			}
			return nil, huma.Error500InternalServerError("Failed to delete task", err)
		}

		response := map[string]interface{}{
			"message": "Task deleted successfully",
			"task_id": input.TaskID,
		}
		return &dto.TaskDeleteOutput{Body: response}, nil
	})

	// Task control endpoints
	huma.Register(api, huma.Operation{
		OperationID: "scheduler-execute-task",
		Method:      "POST",
		Path:        basePath + "/tasks/{task_id}/execute",
		Summary:     "Execute task immediately",
		Description: "Manually trigger immediate execution of a scheduled task",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskExecuteInput) (*dto.TaskExecuteOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		execution, err := service.StartTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to execute task", err)
		}
		return &dto.TaskExecuteOutput{Body: *execution}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-enable-task",
		Method:      "POST",
		Path:        basePath + "/tasks/{task_id}/enable",
		Summary:     "Enable task",
		Description: "Enable a disabled scheduled task",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskEnableInput) (*dto.TaskEnableOutput, error) {
		// TODO: Implement enable task in service
		return nil, huma.Error501NotImplemented("Task enable not yet implemented")
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-disable-task",
		Method:      "POST",
		Path:        basePath + "/tasks/{task_id}/disable",
		Summary:     "Disable task",
		Description: "Disable a scheduled task without deleting it",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskDisableInput) (*dto.TaskDisableOutput, error) {
		// TODO: Implement disable task in service
		return nil, huma.Error501NotImplemented("Task disable not yet implemented")
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-pause-task",
		Method:      "POST",
		Path:        basePath + "/tasks/{task_id}/pause",
		Summary:     "Pause task",
		Description: "Pause execution of a scheduled task",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskPauseInput) (*dto.TaskPauseOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		err = service.PauseTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to pause task", err)
		}

		// Get updated task to return
		task, err := service.GetTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get updated task", err)
		}
		return &dto.TaskPauseOutput{Body: *task}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-resume-task",
		Method:      "POST",
		Path:        basePath + "/tasks/{task_id}/resume",
		Summary:     "Resume task",
		Description: "Resume execution of a paused scheduled task",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskResumeInput) (*dto.TaskResumeOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		err = service.ResumeTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to resume task", err)
		}

		// Get updated task to return
		task, err := service.GetTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get updated task", err)
		}
		return &dto.TaskResumeOutput{Body: *task}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-stop-task",
		Method:      "POST",
		Path:        basePath + "/tasks/{task_id}/stop",
		Summary:     "Stop running task",
		Description: "Stop a currently running scheduled task execution",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskStopInput) (*dto.TaskStopOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		err = service.StopTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to stop task", err)
		}

		response := map[string]interface{}{
			"message": "Task stopped successfully",
			"task_id": input.TaskID,
		}
		return &dto.TaskStopOutput{Body: response}, nil
	})

	// Execution endpoints
	huma.Register(api, huma.Operation{
		OperationID: "scheduler-task-history",
		Method:      "GET",
		Path:        basePath + "/tasks/{task_id}/history",
		Summary:     "Get task execution history",
		Description: "Get execution history for a specific scheduled task",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskExecutionHistoryInput) (*dto.TaskExecutionHistoryOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		query := &dto.TaskExecutionQuery{
			Page:     input.Page,
			PageSize: input.PageSize,
		}

		// Set defaults
		if query.Page == 0 {
			query.Page = 1
		}
		if query.PageSize == 0 {
			query.PageSize = 20
		}

		executions, err := service.GetTaskExecutions(ctx, input.TaskID, query)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get task history", err)
		}
		return &dto.TaskExecutionHistoryOutput{Body: *executions}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-list-executions",
		Method:      "GET",
		Path:        basePath + "/executions",
		Summary:     "List all executions",
		Description: "List all task executions across all tasks with filtering and pagination support",
		Tags:        []string{"Scheduler / Executions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.ExecutionListInput) (*dto.ExecutionListOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		executions, err := service.ListExecutions(ctx, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list executions", err)
		}
		return &dto.ExecutionListOutput{Body: *executions}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-get-execution",
		Method:      "GET",
		Path:        basePath + "/executions/{execution_id}",
		Summary:     "Get execution details",
		Description: "Get detailed information about a specific task execution",
		Tags:        []string{"Scheduler / Executions"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.ExecutionGetInput) (*dto.ExecutionGetOutput, error) {
		// Validate authentication and task management permission
		if middleware == nil || middleware.GetAuthMiddleware() == nil {
			return nil, huma.Error500InternalServerError("Authentication system not available")
		}
		_, err := middleware.GetAuthMiddleware().RequireTaskManagement(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		execution, err := service.GetExecution(ctx, input.ExecutionID)
		if err != nil {
			if err.Error() == "execution not found" {
				return nil, huma.Error404NotFound("Execution not found")
			}
			return nil, huma.Error500InternalServerError("Failed to get execution", err)
		}
		return &dto.ExecutionGetOutput{Body: *execution}, nil
	})

	// Bulk operations
	huma.Register(api, huma.Operation{
		OperationID: "scheduler-bulk-operations",
		Method:      "POST",
		Path:        basePath + "/tasks/bulk",
		Summary:     "Bulk task operations",
		Description: "Perform bulk operations on multiple tasks",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.BulkTaskOperationInput) (*dto.BulkTaskOperationOutput, error) {
		// TODO: Implement bulk operations in service
		return nil, huma.Error501NotImplemented("Bulk operations not yet implemented")
	})

	huma.Register(api, huma.Operation{
		OperationID: "scheduler-import-tasks",
		Method:      "POST",
		Path:        basePath + "/tasks/import",
		Summary:     "Import tasks",
		Description: "Import task configurations from JSON",
		Tags:        []string{"Scheduler / Tasks"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TaskImportInput) (*dto.TaskImportOutput, error) {
		// TODO: Implement task import in service
		return nil, huma.Error501NotImplemented("Task import not yet implemented")
	})
}

// registerRoutes registers all Scheduler module routes with Huma
func (hr *Routes) registerRoutes() {
	// Public endpoints (no authentication required)
	huma.Get(hr.api, "/status", hr.getModuleStatus)
	huma.Get(hr.api, "/scheduler-status", hr.getStatus)
	huma.Get(hr.api, "/stats", hr.getStats)

	// Task management endpoints (require authentication and permissions)
	huma.Get(hr.api, "/tasks", hr.listTasks)
	huma.Post(hr.api, "/tasks", hr.createTask)
	huma.Get(hr.api, "/tasks/{task_id}", hr.getTask)
	huma.Put(hr.api, "/tasks/{task_id}", hr.updateTask)
	huma.Delete(hr.api, "/tasks/{task_id}", hr.deleteTask)

	// Task control endpoints
	huma.Post(hr.api, "/tasks/{task_id}/execute", hr.executeTask)
	huma.Post(hr.api, "/tasks/{task_id}/enable", hr.enableTask)
	huma.Post(hr.api, "/tasks/{task_id}/disable", hr.disableTask)
	huma.Post(hr.api, "/tasks/{task_id}/pause", hr.pauseTask)
	huma.Post(hr.api, "/tasks/{task_id}/resume", hr.resumeTask)
	huma.Post(hr.api, "/tasks/{task_id}/stop", hr.stopTask)

	// Execution endpoints
	huma.Get(hr.api, "/tasks/{task_id}/history", hr.getTaskHistory)
	huma.Get(hr.api, "/executions", hr.listExecutions)
	huma.Get(hr.api, "/executions/{execution_id}", hr.getExecution)

	// Bulk operations
	huma.Post(hr.api, "/tasks/bulk", hr.bulkTaskOperation)
	huma.Post(hr.api, "/tasks/import", hr.importTasks)
}

// Public endpoint handlers

func (hr *Routes) getModuleStatus(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
	status := hr.service.GetModuleStatus(ctx)
	return &dto.StatusOutput{Body: *status}, nil
}

func (hr *Routes) getStatus(ctx context.Context, input *dto.SchedulerStatusInput) (*dto.SchedulerStatusOutput, error) {
	status := hr.service.GetStatus()
	return &dto.SchedulerStatusOutput{Body: *status}, nil
}

func (hr *Routes) getStats(ctx context.Context, input *dto.SchedulerStatsInput) (*dto.SchedulerStatsOutput, error) {
	stats, err := hr.service.GetStats(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get scheduler stats", err)
	}

	return &dto.SchedulerStatsOutput{Body: *stats}, nil
}

// Task management handlers

func (hr *Routes) listTasks(ctx context.Context, input *dto.TaskListInput) (*dto.TaskListOutput, error) {
	// Convert Huma input to service query format
	query := &dto.TaskListQuery{
		Page:     input.Page,
		PageSize: input.PageSize,
		Status:   input.Status,
		Type:     input.Type,
		Enabled:  input.Enabled,
	}

	// Set defaults if not provided
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}

	// TODO: Parse tags from comma-separated string
	// query.Tags = strings.Split(input.Tags, ",")

	tasks, err := hr.service.ListTasks(ctx, query)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list tasks", err)
	}

	return &dto.TaskListOutput{Body: *tasks}, nil
}

func (hr *Routes) createTask(ctx context.Context, input *dto.TaskCreateInput) (*dto.TaskCreateOutput, error) {
	// TODO: Add permission checking middleware
	// For now, create the task directly
	task, err := hr.service.CreateTask(ctx, &input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create task", err)
	}

	return &dto.TaskCreateOutput{Body: *task}, nil
}

func (hr *Routes) getTask(ctx context.Context, input *dto.TaskGetInput) (*dto.TaskGetOutput, error) {
	task, err := hr.service.GetTask(ctx, input.TaskID)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, huma.Error404NotFound("Task not found")
		}
		return nil, huma.Error500InternalServerError("Failed to get task", err)
	}

	return &dto.TaskGetOutput{Body: *task}, nil
}

func (hr *Routes) updateTask(ctx context.Context, input *dto.TaskUpdateInput) (*dto.TaskUpdateOutput, error) {
	task, err := hr.service.UpdateTask(ctx, input.TaskID, &input.Body)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, huma.Error404NotFound("Task not found")
		}
		if err.Error() == "cannot update system tasks" {
			return nil, huma.Error403Forbidden("Cannot update system tasks")
		}
		return nil, huma.Error400BadRequest("Failed to update task", err)
	}

	return &dto.TaskUpdateOutput{Body: *task}, nil
}

func (hr *Routes) deleteTask(ctx context.Context, input *dto.TaskDeleteInput) (*dto.TaskDeleteOutput, error) {
	err := hr.service.DeleteTask(ctx, input.TaskID)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, huma.Error404NotFound("Task not found")
		}
		if err.Error() == "cannot delete system tasks" {
			return nil, huma.Error403Forbidden("Cannot delete system tasks")
		}
		return nil, huma.Error500InternalServerError("Failed to delete task", err)
	}

	response := map[string]interface{}{
		"message": "Task deleted successfully",
		"task_id": input.TaskID,
	}

	return &dto.TaskDeleteOutput{Body: response}, nil
}

// Task control handlers

func (hr *Routes) executeTask(ctx context.Context, input *dto.TaskExecuteInput) (*dto.TaskExecuteOutput, error) {
	execution, err := hr.service.StartTask(ctx, input.TaskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to execute task", err)
	}

	return &dto.TaskExecuteOutput{Body: *execution}, nil
}

func (hr *Routes) enableTask(ctx context.Context, input *dto.TaskEnableInput) (*dto.TaskEnableOutput, error) {
	// TODO: Implement enable task in service
	return nil, huma.Error501NotImplemented("Task enable not yet implemented")
}

func (hr *Routes) disableTask(ctx context.Context, input *dto.TaskDisableInput) (*dto.TaskDisableOutput, error) {
	// TODO: Implement disable task in service
	return nil, huma.Error501NotImplemented("Task disable not yet implemented")
}

func (hr *Routes) pauseTask(ctx context.Context, input *dto.TaskPauseInput) (*dto.TaskPauseOutput, error) {
	err := hr.service.PauseTask(ctx, input.TaskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to pause task", err)
	}

	// Get updated task to return
	task, err := hr.service.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get updated task", err)
	}

	return &dto.TaskPauseOutput{Body: *task}, nil
}

func (hr *Routes) resumeTask(ctx context.Context, input *dto.TaskResumeInput) (*dto.TaskResumeOutput, error) {
	err := hr.service.ResumeTask(ctx, input.TaskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to resume task", err)
	}

	// Get updated task to return
	task, err := hr.service.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get updated task", err)
	}

	return &dto.TaskResumeOutput{Body: *task}, nil
}

func (hr *Routes) stopTask(ctx context.Context, input *dto.TaskStopInput) (*dto.TaskStopOutput, error) {
	err := hr.service.StopTask(ctx, input.TaskID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to stop task", err)
	}

	response := map[string]interface{}{
		"message": "Task stopped successfully",
		"task_id": input.TaskID,
	}

	return &dto.TaskStopOutput{Body: response}, nil
}

// Execution handlers

func (hr *Routes) getTaskHistory(ctx context.Context, input *dto.TaskExecutionHistoryInput) (*dto.TaskExecutionHistoryOutput, error) {
	query := &dto.TaskExecutionQuery{
		Page:     input.Page,
		PageSize: input.PageSize,
	}

	// Set defaults
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}

	executions, err := hr.service.GetTaskExecutions(ctx, input.TaskID, query)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get task history", err)
	}

	return &dto.TaskExecutionHistoryOutput{Body: *executions}, nil
}

func (hr *Routes) listExecutions(ctx context.Context, input *dto.ExecutionListInput) (*dto.ExecutionListOutput, error) {
	executions, err := hr.service.ListExecutions(ctx, input)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to list executions", err)
	}
	return &dto.ExecutionListOutput{Body: *executions}, nil
}

func (hr *Routes) getExecution(ctx context.Context, input *dto.ExecutionGetInput) (*dto.ExecutionGetOutput, error) {
	execution, err := hr.service.GetExecution(ctx, input.ExecutionID)
	if err != nil {
		if err.Error() == "execution not found" {
			return nil, huma.Error404NotFound("Execution not found")
		}
		return nil, huma.Error500InternalServerError("Failed to get execution", err)
	}

	return &dto.ExecutionGetOutput{Body: *execution}, nil
}

// Bulk operation handlers

func (hr *Routes) bulkTaskOperation(ctx context.Context, input *dto.BulkTaskOperationInput) (*dto.BulkTaskOperationOutput, error) {
	// TODO: Implement bulk operations in service
	return nil, huma.Error501NotImplemented("Bulk operations not yet implemented")
}

func (hr *Routes) importTasks(ctx context.Context, input *dto.TaskImportInput) (*dto.TaskImportOutput, error) {
	// TODO: Implement task import in service
	return nil, huma.Error501NotImplemented("Task import not yet implemented")
}
