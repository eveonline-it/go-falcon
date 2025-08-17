package routes

import (
	"encoding/json"
	"net/http"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/internal/scheduler/middleware"
	"go-falcon/internal/scheduler/services"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Routes handles HTTP routing for the scheduler module
type Routes struct {
	service    *services.SchedulerService
	middleware *middleware.Middleware
}

// New creates a new routes instance
func New(service *services.SchedulerService, middleware *middleware.Middleware) *Routes {
	return &Routes{
		service:    service,
		middleware: middleware,
	}
}

// RegisterRoutes registers all scheduler routes
func (rt *Routes) RegisterRoutes(r chi.Router) {
	// Apply common middleware
	r.Use(rt.middleware.RequestLogging)
	r.Use(rt.middleware.SecurityHeaders)

	// Public endpoints (read-only status)
	r.Get("/status", rt.getStatusHandler)
	r.Get("/stats", rt.getStatsHandler)

	// Protected task management routes - these need to be set by the main module
	// based on the groups module permissions
}

// RegisterProtectedRoutes registers protected routes with permission middleware
func (rt *Routes) RegisterProtectedRoutes(r chi.Router, permissionMiddleware func(service, resource, action string) func(http.Handler) http.Handler) {
	// Protected task management routes
	r.Group(func(r chi.Router) {
		r.Use(permissionMiddleware("scheduler", "tasks", "read"))
		r.With(rt.middleware.ValidateQueryParams).Get("/tasks", rt.listTasksHandler)
		r.Get("/tasks/{taskID}", rt.getTaskHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(permissionMiddleware("scheduler", "tasks", "write"))
		r.With(rt.middleware.GetValidationMiddleware().ValidateTaskCreateRequest).Post("/tasks", rt.createTaskHandler)
		r.With(rt.middleware.GetValidationMiddleware().ValidateTaskUpdateRequest).Put("/tasks/{taskID}", rt.updateTaskHandler)
		r.Post("/tasks/{taskID}/pause", rt.pauseTaskHandler)
		r.Post("/tasks/{taskID}/resume", rt.resumeTaskHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(permissionMiddleware("scheduler", "tasks", "delete"))
		r.Delete("/tasks/{taskID}", rt.deleteTaskHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(permissionMiddleware("scheduler", "tasks", "execute"))
		r.Post("/tasks/{taskID}/start", rt.startTaskHandler)
		r.Post("/tasks/{taskID}/stop", rt.stopTaskHandler)
	})

	// Protected task execution history
	r.Group(func(r chi.Router) {
		r.Use(permissionMiddleware("scheduler", "executions", "read"))
		r.With(rt.middleware.ValidateQueryParams).Get("/tasks/{taskID}/history", rt.getTaskHistoryHandler)
		r.Get("/tasks/{taskID}/executions/{executionID}", rt.getExecutionHandler)
	})

	// Protected scheduler management
	r.Group(func(r chi.Router) {
		r.Use(permissionMiddleware("scheduler", "tasks", "admin"))
		r.Post("/reload", rt.reloadTasksHandler)
	})
}

// Public Endpoints

func (rt *Routes) getStatusHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "scheduler.get_status",
		attribute.String("service", "scheduler"),
		attribute.String("operation", "get_status"),
	)
	defer span.End()

	status := rt.service.GetStatus()
	handlers.JSONResponse(w, status, http.StatusOK)
}

func (rt *Routes) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "scheduler.get_stats",
		attribute.String("service", "scheduler"),
		attribute.String("operation", "get_stats"),
	)
	defer span.End()

	stats, err := rt.service.GetStats(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get scheduler stats")
		handlers.ErrorResponse(w, "Failed to get scheduler stats", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, stats, http.StatusOK)
}

// Task Management Endpoints

func (rt *Routes) listTasksHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "scheduler.list_tasks",
		attribute.String("service", "scheduler"),
		attribute.String("operation", "list_tasks"),
	)
	defer span.End()

	// Get validated query from context
	query, ok := handlers.GetValidatedQuery(r.Context()).(*dto.TaskListQuery)
	if !ok {
		// Fallback to default if validation middleware wasn't used
		query = &dto.TaskListQuery{
			Page:     1,
			PageSize: 20,
		}
	}

	tasks, err := rt.service.ListTasks(r.Context(), query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list tasks")
		handlers.ErrorResponse(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int64("tasks.count", tasks.Total),
		attribute.Int("tasks.page", tasks.Page),
		attribute.Int("tasks.page_size", tasks.PageSize),
	)

	handlers.JSONResponse(w, tasks, http.StatusOK)
}

func (rt *Routes) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "scheduler.create_task",
		attribute.String("service", "scheduler"),
		attribute.String("operation", "create_task"),
	)
	defer span.End()

	// Get validated request from context
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.TaskCreateRequest)
	if !ok {
		span.SetStatus(codes.Error, "Invalid request")
		handlers.ErrorResponse(w, "Invalid request", http.StatusBadRequest)
		return
	}

	task, err := rt.service.CreateTask(r.Context(), req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create task")
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.String("task.id", task.ID),
		attribute.String("task.name", task.Name),
		attribute.String("task.type", string(task.Type)),
	)

	handlers.JSONResponse(w, task, http.StatusCreated)
}

func (rt *Routes) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	task, err := rt.service.GetTask(r.Context(), taskID)
	if err != nil {
		if err.Error() == "task not found" {
			handlers.ErrorResponse(w, "Task not found", http.StatusNotFound)
			return
		}
		handlers.ErrorResponse(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, task, http.StatusOK)
}

func (rt *Routes) updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	// Get validated request from context
	req, ok := handlers.GetValidatedRequest(r.Context()).(*dto.TaskUpdateRequest)
	if !ok {
		handlers.ErrorResponse(w, "Invalid request", http.StatusBadRequest)
		return
	}

	task, err := rt.service.UpdateTask(r.Context(), taskID, req)
	if err != nil {
		if err.Error() == "task not found" {
			handlers.ErrorResponse(w, "Task not found", http.StatusNotFound)
			return
		}
		if err.Error() == "cannot update system tasks" {
			handlers.ErrorResponse(w, "Cannot update system tasks", http.StatusForbidden)
			return
		}
		handlers.ErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	handlers.JSONResponse(w, task, http.StatusOK)
}

func (rt *Routes) deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	err := rt.service.DeleteTask(r.Context(), taskID)
	if err != nil {
		if err.Error() == "task not found" {
			handlers.ErrorResponse(w, "Task not found", http.StatusNotFound)
			return
		}
		if err.Error() == "cannot delete system tasks" {
			handlers.ErrorResponse(w, "Cannot delete system tasks", http.StatusForbidden)
			return
		}
		handlers.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	handlers.SuccessResponse(w, map[string]string{"message": "Task deleted successfully"}, http.StatusOK)
}

// Task Control Endpoints

func (rt *Routes) startTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	execution, err := rt.service.StartTask(r.Context(), taskID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, execution, http.StatusOK)
}

func (rt *Routes) stopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	err := rt.service.StopTask(r.Context(), taskID)
	if err != nil {
		handlers.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	handlers.SuccessResponse(w, map[string]string{"message": "Task stopped successfully"}, http.StatusOK)
}

func (rt *Routes) pauseTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	err := rt.service.PauseTask(r.Context(), taskID)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to pause task", http.StatusInternalServerError)
		return
	}

	handlers.SuccessResponse(w, map[string]string{"message": "Task paused successfully"}, http.StatusOK)
}

func (rt *Routes) resumeTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	err := rt.service.ResumeTask(r.Context(), taskID)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to resume task", http.StatusInternalServerError)
		return
	}

	handlers.SuccessResponse(w, map[string]string{"message": "Task resumed successfully"}, http.StatusOK)
}

// Execution History Endpoints

func (rt *Routes) getTaskHistoryHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		handlers.ErrorResponse(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	// Get validated query from context
	query, ok := handlers.GetValidatedQuery(r.Context()).(*dto.TaskExecutionQuery)
	if !ok {
		// Fallback to default
		query = &dto.TaskExecutionQuery{
			Page:     1,
			PageSize: 20,
		}
	}

	executions, err := rt.service.GetTaskExecutions(r.Context(), taskID, query)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to get task history", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, executions, http.StatusOK)
}

func (rt *Routes) getExecutionHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	executionID := chi.URLParam(r, "executionID")
	if taskID == "" || executionID == "" {
		handlers.ErrorResponse(w, "Task ID and Execution ID are required", http.StatusBadRequest)
		return
	}

	execution, err := rt.service.GetExecution(r.Context(), executionID)
	if err != nil {
		if err.Error() == "execution not found" {
			handlers.ErrorResponse(w, "Execution not found", http.StatusNotFound)
			return
		}
		handlers.ErrorResponse(w, "Failed to get execution", http.StatusInternalServerError)
		return
	}

	if execution.TaskID != taskID {
		handlers.ErrorResponse(w, "Execution does not belong to the specified task", http.StatusBadRequest)
		return
	}

	handlers.JSONResponse(w, execution, http.StatusOK)
}

// Management Endpoints

func (rt *Routes) reloadTasksHandler(w http.ResponseWriter, r *http.Request) {
	err := rt.service.ReloadTasks()
	if err != nil {
		handlers.ErrorResponse(w, "Failed to reload tasks", http.StatusInternalServerError)
		return
	}

	handlers.SuccessResponse(w, map[string]string{"message": "Tasks reloaded successfully"}, http.StatusOK)
}

// Health Check Handler (for base module compatibility)
func (rt *Routes) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"module":  "scheduler",
		"status":  "healthy",
		"version": "1.0.0",
	})
}