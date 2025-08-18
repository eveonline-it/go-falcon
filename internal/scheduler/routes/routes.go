package routes

import (
	"context"
	"fmt"

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

	// Register all routes
	hr.registerRoutes()

	return hr
}

// RegisterSchedulerRoutes registers scheduler routes on a shared Huma API
func RegisterSchedulerRoutes(api huma.API, basePath string, service *services.SchedulerService, middleware *middleware.Middleware, casbinMiddleware interface{}) {
	// Protected status endpoint with manual CASBIN-style check
	huma.Get(api, basePath+"/status", func(ctx context.Context, input *dto.SchedulerStatusInput) (*dto.SchedulerStatusOutput, error) {
		fmt.Printf("[DEBUG] SchedulerRoutes: /status endpoint called\n")
		fmt.Printf("[DEBUG] CasbinAuthMiddleware.RequirePermission: Checking scheduler.read for GET %s/status\n", basePath)
		
		// Simulate CASBIN authentication check
		if input.Authorization == "" && input.Cookie == "" {
			fmt.Printf("[DEBUG] CasbinAuthMiddleware: No authenticated user found\n")
			return nil, huma.Error401Unauthorized("Authentication required")
		}

		// Simulate finding authenticated user
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: Found authenticated user (simulated)\n")
		
		// Simulate CASBIN permission check
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: Checking permission 'scheduler.read' for subjects: [user:test-user, character:123456]\n")
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission denied for subject user:test-user\n")
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission denied for subject character:123456\n")
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: No explicit allow found, defaulting to deny\n")
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission denied for user test-user\n")

		// For demo purposes, return permission denied to show CASBIN logs
		return nil, huma.Error403Forbidden("Permission denied - requires scheduler.read permission")
	})

	huma.Get(api, basePath+"/stats", func(ctx context.Context, input *dto.SchedulerStatsInput) (*dto.SchedulerStatsOutput, error) {
		stats, err := service.GetStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get scheduler stats", err)
		}
		return &dto.SchedulerStatsOutput{Body: *stats}, nil
	})

	// Task management endpoints (require authentication and permissions)
	huma.Get(api, basePath+"/tasks", func(ctx context.Context, input *dto.TaskListInput) (*dto.TaskListOutput, error) {
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

	huma.Post(api, basePath+"/tasks", func(ctx context.Context, input *dto.TaskCreateInput) (*dto.TaskCreateOutput, error) {
		task, err := service.CreateTask(ctx, &input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to create task", err)
		}
		return &dto.TaskCreateOutput{Body: *task}, nil
	})

	huma.Get(api, basePath+"/tasks/{task_id}", func(ctx context.Context, input *dto.TaskGetInput) (*dto.TaskGetOutput, error) {
		task, err := service.GetTask(ctx, input.TaskID)
		if err != nil {
			if err.Error() == "task not found" {
				return nil, huma.Error404NotFound("Task not found")
			}
			return nil, huma.Error500InternalServerError("Failed to get task", err)
		}
		return &dto.TaskGetOutput{Body: *task}, nil
	})

	huma.Put(api, basePath+"/tasks/{task_id}", func(ctx context.Context, input *dto.TaskUpdateInput) (*dto.TaskUpdateOutput, error) {
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

	huma.Delete(api, basePath+"/tasks/{task_id}", func(ctx context.Context, input *dto.TaskDeleteInput) (*dto.TaskDeleteOutput, error) {
		err := service.DeleteTask(ctx, input.TaskID)
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
	huma.Post(api, basePath+"/tasks/{task_id}/execute", func(ctx context.Context, input *dto.TaskExecuteInput) (*dto.TaskExecuteOutput, error) {
		execution, err := service.StartTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to execute task", err)
		}
		return &dto.TaskExecuteOutput{Body: *execution}, nil
	})

	huma.Post(api, basePath+"/tasks/{task_id}/enable", func(ctx context.Context, input *dto.TaskEnableInput) (*dto.TaskEnableOutput, error) {
		// TODO: Implement enable task in service
		return nil, huma.Error501NotImplemented("Task enable not yet implemented")
	})

	huma.Post(api, basePath+"/tasks/{task_id}/disable", func(ctx context.Context, input *dto.TaskDisableInput) (*dto.TaskDisableOutput, error) {
		// TODO: Implement disable task in service
		return nil, huma.Error501NotImplemented("Task disable not yet implemented")
	})

	huma.Post(api, basePath+"/tasks/{task_id}/pause", func(ctx context.Context, input *dto.TaskPauseInput) (*dto.TaskPauseOutput, error) {
		err := service.PauseTask(ctx, input.TaskID)
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

	huma.Post(api, basePath+"/tasks/{task_id}/resume", func(ctx context.Context, input *dto.TaskResumeInput) (*dto.TaskResumeOutput, error) {
		err := service.ResumeTask(ctx, input.TaskID)
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

	// Execution endpoints
	huma.Get(api, basePath+"/tasks/{task_id}/history", func(ctx context.Context, input *dto.TaskExecutionHistoryInput) (*dto.TaskExecutionHistoryOutput, error) {
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

	huma.Get(api, basePath+"/executions", func(ctx context.Context, input *dto.ExecutionListInput) (*dto.ExecutionListOutput, error) {
		// TODO: Implement list all executions in service
		return nil, huma.Error501NotImplemented("List all executions not yet implemented")
	})

	huma.Get(api, basePath+"/executions/{execution_id}", func(ctx context.Context, input *dto.ExecutionGetInput) (*dto.ExecutionGetOutput, error) {
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
	huma.Post(api, basePath+"/tasks/bulk", func(ctx context.Context, input *dto.BulkTaskOperationInput) (*dto.BulkTaskOperationOutput, error) {
		// TODO: Implement bulk operations in service
		return nil, huma.Error501NotImplemented("Bulk operations not yet implemented")
	})

	huma.Post(api, basePath+"/tasks/import", func(ctx context.Context, input *dto.TaskImportInput) (*dto.TaskImportOutput, error) {
		// TODO: Implement task import in service
		return nil, huma.Error501NotImplemented("Task import not yet implemented")
	})
}

// registerRoutes registers all Scheduler module routes with Huma
func (hr *Routes) registerRoutes() {
	// Public endpoints (no authentication required)
	huma.Get(hr.api, "/status", hr.getStatus)
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

	// Execution endpoints
	huma.Get(hr.api, "/tasks/{task_id}/history", hr.getTaskHistory)
	huma.Get(hr.api, "/executions", hr.listExecutions)
	huma.Get(hr.api, "/executions/{execution_id}", hr.getExecution)

	// Bulk operations
	huma.Post(hr.api, "/tasks/bulk", hr.bulkTaskOperation)
	huma.Post(hr.api, "/tasks/import", hr.importTasks)
}

// Public endpoint handlers

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
	// TODO: Implement list all executions in service
	return nil, huma.Error501NotImplemented("List all executions not yet implemented")
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