package routes

import (
	"context"
	"fmt"
	"net/http"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/internal/scheduler/middleware"
	"go-falcon/internal/scheduler/services"
	casbinPkg "go-falcon/pkg/middleware/casbin"

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
	// Protected scheduler status endpoint - requires admin or super_admin role
	huma.Get(api, basePath+"/status", func(ctx context.Context, input *dto.SchedulerStatusInput) (*dto.SchedulerStatusOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		
		// Permission granted - return scheduler status
		status := service.GetStatus()
		return &dto.SchedulerStatusOutput{Body: *status}, nil
	})

	// Public stats endpoint (no authentication required)
	huma.Get(api, basePath+"/stats", func(ctx context.Context, input *dto.SchedulerStatsInput) (*dto.SchedulerStatsOutput, error) {
		stats, err := service.GetStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get scheduler stats", err)
		}
		return &dto.SchedulerStatsOutput{Body: *stats}, nil
	})

	// Protected task management endpoints - require admin or super_admin role
	huma.Get(api, basePath+"/tasks", func(ctx context.Context, input *dto.TaskListInput) (*dto.TaskListOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
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

	huma.Post(api, basePath+"/tasks", func(ctx context.Context, input *dto.TaskCreateInput) (*dto.TaskCreateOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		task, err := service.CreateTask(ctx, &input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to create task", err)
		}
		return &dto.TaskCreateOutput{Body: *task}, nil
	})

	huma.Get(api, basePath+"/tasks/{task_id}", func(ctx context.Context, input *dto.TaskGetInput) (*dto.TaskGetOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
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

	huma.Put(api, basePath+"/tasks/{task_id}", func(ctx context.Context, input *dto.TaskUpdateInput) (*dto.TaskUpdateOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
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

	huma.Delete(api, basePath+"/tasks/{task_id}", func(ctx context.Context, input *dto.TaskDeleteInput) (*dto.TaskDeleteOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
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

	// Task control endpoints - require admin or super_admin role
	huma.Post(api, basePath+"/tasks/{task_id}/execute", func(ctx context.Context, input *dto.TaskExecuteInput) (*dto.TaskExecuteOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		execution, err := service.StartTask(ctx, input.TaskID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to execute task", err)
		}
		return &dto.TaskExecuteOutput{Body: *execution}, nil
	})

	huma.Post(api, basePath+"/tasks/{task_id}/enable", func(ctx context.Context, input *dto.TaskEnableInput) (*dto.TaskEnableOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		// TODO: Implement enable task in service
		return nil, huma.Error501NotImplemented("Task enable not yet implemented")
	})

	huma.Post(api, basePath+"/tasks/{task_id}/disable", func(ctx context.Context, input *dto.TaskDisableInput) (*dto.TaskDisableOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		// TODO: Implement disable task in service
		return nil, huma.Error501NotImplemented("Task disable not yet implemented")
	})

	huma.Post(api, basePath+"/tasks/{task_id}/pause", func(ctx context.Context, input *dto.TaskPauseInput) (*dto.TaskPauseOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
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
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
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

	// Execution endpoints - require admin or super_admin role
	huma.Get(api, basePath+"/tasks/{task_id}/history", func(ctx context.Context, input *dto.TaskExecutionHistoryInput) (*dto.TaskExecutionHistoryOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
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

	huma.Get(api, basePath+"/executions", func(ctx context.Context, input *dto.ExecutionListInput) (*dto.ExecutionListOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		// TODO: Implement list all executions in service
		return nil, huma.Error501NotImplemented("List all executions not yet implemented")
	})

	huma.Get(api, basePath+"/executions/{execution_id}", func(ctx context.Context, input *dto.ExecutionGetInput) (*dto.ExecutionGetOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
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

	// Bulk operations - require admin or super_admin role
	huma.Post(api, basePath+"/tasks/bulk", func(ctx context.Context, input *dto.BulkTaskOperationInput) (*dto.BulkTaskOperationOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
		// TODO: Implement bulk operations in service
		return nil, huma.Error501NotImplemented("Bulk operations not yet implemented")
	})

	huma.Post(api, basePath+"/tasks/import", func(ctx context.Context, input *dto.TaskImportInput) (*dto.TaskImportOutput, error) {
		// Check admin or super_admin permission
		if err := checkSchedulerAdminPermission(casbinMiddleware, ctx, input.Authorization, input.Cookie); err != nil {
			return nil, err
		}
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

// checkSchedulerAdminPermission checks for admin or super_admin access to scheduler endpoints
func checkSchedulerAdminPermission(casbinMiddleware interface{}, ctx context.Context, authHeader, cookieHeader string) error {
	// Try scheduler.admin permission first (covers both admin and super_admin roles)
	if err := checkSchedulerPermission(casbinMiddleware, ctx, authHeader, cookieHeader, "scheduler", "admin"); err == nil {
		return nil // Permission granted via scheduler.admin
	}
	
	// If scheduler.admin fails, try system.super_admin as fallback
	if err := checkSchedulerPermission(casbinMiddleware, ctx, authHeader, cookieHeader, "system", "super_admin"); err != nil {
		// Both permission checks failed
		return huma.Error403Forbidden("Permission denied - requires admin or super_admin role")
	}
	
	return nil // Permission granted via system.super_admin
}

// checkSchedulerPermission manually checks CASBIN permissions for scheduler endpoints
func checkSchedulerPermission(casbinMiddleware interface{}, ctx context.Context, authHeader, cookieHeader, resource, action string) error {
	// Check if CASBIN is available
	if casbinMiddleware == nil {
		return huma.Error503ServiceUnavailable("Authentication system not available")
	}
	
	// Check for authentication headers
	if authHeader == "" && cookieHeader == "" {
		return huma.Error401Unauthorized("Authentication required - provide Authorization header or falcon_auth_token cookie")
	}
	
	// This will help identify what type we're actually getting
	fmt.Printf("[DEBUG] checkSchedulerPermission: casbinMiddleware type: %T\n", casbinMiddleware)
	fmt.Printf("[DEBUG] checkSchedulerPermission: authHeader present: %t\n", authHeader != "")
	fmt.Printf("[DEBUG] checkSchedulerPermission: cookieHeader present: %t\n", cookieHeader != "")
	fmt.Printf("[DEBUG] checkSchedulerPermission: checking %s.%s permission\n", resource, action)
	
	// Type cast the CASBIN factory
	factory, ok := casbinMiddleware.(*casbinPkg.CasbinMiddlewareFactory)
	if !ok {
		fmt.Printf("[DEBUG] checkSchedulerPermission: Failed to cast CASBIN factory\n")
		return huma.Error500InternalServerError("Failed to access authentication system")
	}
	
	fmt.Printf("[DEBUG] checkSchedulerPermission: Successfully cast CASBIN factory\n")
	
	// Create a mock HTTP request to trigger the CASBIN middleware debugging
	req, err := http.NewRequestWithContext(ctx, "GET", "/scheduler/status", nil)
	if err != nil {
		fmt.Printf("[DEBUG] checkSchedulerPermission: Failed to create mock request: %v\n", err)
		return huma.Error500InternalServerError("Internal error creating request")
	}
	
	// Add authentication headers to the mock request
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	
	fmt.Printf("[DEBUG] checkSchedulerPermission: Created mock request with headers\n")
	
	// Get the enhanced middleware that includes auth + character resolution + permissions
	enhancedMiddleware := factory.GetEnhanced()
	
	// Create a test response writer to capture the middleware behavior
	testResponseWriter := &testResponseWriter{statusCode: 200}
	
	// Create the complete middleware chain: Auth -> Character Resolution -> Permissions
	fullMiddleware := enhancedMiddleware.RequireAuthWithPermission(resource, action)
	
	fmt.Printf("[DEBUG] checkSchedulerPermission: About to call complete CASBIN middleware chain\n")
	
	// This will trigger authentication, character resolution, AND permission checking!
	fullMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[DEBUG] checkSchedulerPermission: Complete CASBIN middleware chain passed - access granted!\n")
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(testResponseWriter, req)
	
	fmt.Printf("[DEBUG] checkSchedulerPermission: CASBIN middleware completed with status: %d\n", testResponseWriter.statusCode)
	
	// Check the result
	if testResponseWriter.statusCode == http.StatusOK {
		return nil // Permission granted
	} else if testResponseWriter.statusCode == http.StatusUnauthorized {
		return huma.Error401Unauthorized("Authentication failed")
	} else if testResponseWriter.statusCode == http.StatusForbidden {
		return huma.Error403Forbidden("Permission denied - requires admin or super_admin role")
	} else {
		return huma.Error500InternalServerError("Authentication system error")
	}
}

// testResponseWriter captures the HTTP response for testing middleware
type testResponseWriter struct {
	statusCode int
	headers    http.Header
	body       []byte
}

func (w *testResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

