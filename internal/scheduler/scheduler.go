package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Module struct {
	*module.BaseModule
	engine     *Engine
	repository *Repository
}

// New creates a new scheduler module
func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	repository := NewRepository(mongodb)
	engine := NewEngine(repository, redis)

	return &Module{
		BaseModule: module.NewBaseModule("scheduler", mongodb, redis, sdeService),
		engine:     engine,
		repository: repository,
	}
}

// Routes sets up the HTTP routes for the scheduler module
func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler

	// Task management routes
	r.Get("/tasks", m.listTasksHandler)
	r.Post("/tasks", m.createTaskHandler)
	r.Get("/tasks/{taskID}", m.getTaskHandler)
	r.Put("/tasks/{taskID}", m.updateTaskHandler)
	r.Delete("/tasks/{taskID}", m.deleteTaskHandler)

	// Task control routes
	r.Post("/tasks/{taskID}/start", m.startTaskHandler)
	r.Post("/tasks/{taskID}/stop", m.stopTaskHandler)
	r.Post("/tasks/{taskID}/pause", m.pauseTaskHandler)
	r.Post("/tasks/{taskID}/resume", m.resumeTaskHandler)

	// Task execution history
	r.Get("/tasks/{taskID}/history", m.getTaskHistoryHandler)
	r.Get("/tasks/{taskID}/executions/{executionID}", m.getExecutionHandler)

	// Scheduler management
	r.Get("/stats", m.getStatsHandler)
	r.Post("/reload", m.reloadTasksHandler)
	r.Get("/status", m.getStatusHandler)
}

// StartBackgroundTasks starts the scheduler engine and background tasks
func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting scheduler background tasks", slog.String("module", m.Name()))

	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)

	// Initialize hardcoded system tasks
	go m.initializeSystemTasks(ctx)

	// Start the scheduler engine
	go m.engine.Start(ctx)

	// Start task cleanup routine
	go m.runTaskCleanup(ctx)

	// Monitor scheduler health
	for {
		select {
		case <-ctx.Done():
			slog.Info("Scheduler background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Scheduler background tasks stopped")
			return
		default:
			// Main scheduler loop - the engine handles the actual scheduling
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			case <-time.After(1 * time.Minute):
				// Periodic health check and maintenance
				m.performMaintenance()
			}
		}
	}
}

// initializeSystemTasks creates hardcoded system tasks if they don't exist
func (m *Module) initializeSystemTasks(ctx context.Context) {
	systemTasks := m.getSystemTasks()

	for _, task := range systemTasks {
		existing, err := m.repository.GetTask(ctx, task.ID)
		if err != nil && err != mongo.ErrNoDocuments {
			slog.Error("Failed to check existing system task",
				slog.String("task_id", task.ID),
				slog.String("error", err.Error()))
			continue
		}

		if existing == nil {
			// Task doesn't exist, create it
			if err := m.repository.CreateTask(ctx, task); err != nil {
				slog.Error("Failed to create system task",
					slog.String("task_id", task.ID),
					slog.String("task_name", task.Name),
					slog.String("error", err.Error()))
			} else {
				slog.Info("Created system task",
					slog.String("task_id", task.ID),
					slog.String("task_name", task.Name))
			}
		} else {
			// Task exists, update if needed (maintain system task integrity)
			if existing.Schedule != task.Schedule || existing.Type != task.Type {
				existing.Schedule = task.Schedule
				existing.Type = task.Type
				existing.Config = task.Config
				existing.UpdatedAt = time.Now()

				if err := m.repository.UpdateTask(ctx, existing); err != nil {
					slog.Error("Failed to update system task",
						slog.String("task_id", task.ID),
						slog.String("error", err.Error()))
				} else {
					slog.Info("Updated system task",
						slog.String("task_id", task.ID),
						slog.String("task_name", task.Name))
				}
			}
		}
	}
}

// getSystemTasks returns predefined system tasks
func (m *Module) getSystemTasks() []*Task {
	return getSystemTasks()
}

// runTaskCleanup performs periodic cleanup of task execution history
func (m *Module) runTaskCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Task cleanup stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Task cleanup stopped")
			return
		case <-ticker.C:
			if err := m.repository.CleanupExecutions(ctx, 30*24*time.Hour); err != nil {
				slog.Error("Failed to cleanup task executions", slog.String("error", err.Error()))
			}
		}
	}
}

// performMaintenance performs periodic maintenance tasks
func (m *Module) performMaintenance() {
	// Update task statistics
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := m.repository.UpdateTaskStatistics(ctx); err != nil {
		slog.Error("Failed to update task statistics", slog.String("error", err.Error()))
	}

	// Check for stale running tasks (running for more than their timeout)
	if err := m.repository.HandleStaleRunningTasks(ctx); err != nil {
		slog.Error("Failed to handle stale running tasks", slog.String("error", err.Error()))
	}
}

// HTTP Handlers

func (m *Module) listTasksHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "scheduler.list_tasks",
		attribute.String("service", "scheduler"),
		attribute.String("operation", "list_tasks"),
	)
	defer span.End()

	// Parse query parameters
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := r.URL.Query().Get("status")
	taskType := r.URL.Query().Get("type")
	enabled := r.URL.Query().Get("enabled")
	tags := strings.Split(r.URL.Query().Get("tags"), ",")
	if len(tags) == 1 && tags[0] == "" {
		tags = nil
	}

	// Build filter
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	if taskType != "" {
		filter["type"] = taskType
	}
	if enabled != "" {
		if enabled == "true" {
			filter["enabled"] = true
		} else if enabled == "false" {
			filter["enabled"] = false
		}
	}
	if len(tags) > 0 {
		filter["metadata.tags"] = bson.M{"$in": tags}
	}

	tasks, total, err := m.repository.ListTasks(r.Context(), filter, page, pageSize)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list tasks")
		slog.Error("Failed to list tasks", slog.String("error", err.Error()))
		http.Error(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}

	response := TaskListResponse{
		Tasks:      tasks,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}

	span.SetAttributes(
		attribute.Int64("tasks.count", total),
		attribute.Int("tasks.page", page),
		attribute.Int("tasks.page_size", pageSize),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) createTaskHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "scheduler.create_task",
		attribute.String("service", "scheduler"),
		attribute.String("operation", "create_task"),
	)
	defer span.End()

	var request TaskCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := m.validateTaskCreateRequest(&request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Validation failed")
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Create task
	now := time.Now()
	task := &Task{
		ID:          uuid.New().String(),
		Name:        request.Name,
		Description: request.Description,
		Type:        request.Type,
		Schedule:    request.Schedule,
		Status:      TaskStatusPending,
		Priority:    request.Priority,
		Enabled:     request.Enabled,
		Config:      request.Config,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "api", // TODO: Get from authenticated user
	}

	// Set metadata
	if request.Metadata != nil {
		task.Metadata = *request.Metadata
	} else {
		task.Metadata = TaskMetadata{
			MaxRetries:    3,
			RetryInterval: 1 * time.Minute,
			Timeout:       5 * time.Minute,
			Tags:          request.Tags,
			IsSystem:      false,
			Source:        "api",
			Version:       1,
		}
	}

	if err := m.repository.CreateTask(r.Context(), task); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create task")
		slog.Error("Failed to create task", slog.String("error", err.Error()))
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	// Reload tasks in engine
	m.engine.ReloadTasks()

	span.SetAttributes(
		attribute.String("task.id", task.ID),
		attribute.String("task.name", task.Name),
		attribute.String("task.type", string(task.Type)),
	)

	slog.Info("Task created successfully",
		slog.String("task_id", task.ID),
		slog.String("task_name", task.Name))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (m *Module) getTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	task, err := m.repository.GetTask(r.Context(), taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		slog.Error("Failed to get task", slog.String("error", err.Error()))
		http.Error(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (m *Module) updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	var request TaskUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	task, err := m.repository.GetTask(r.Context(), taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		slog.Error("Failed to get task for update", slog.String("error", err.Error()))
		http.Error(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	// Prevent updating system tasks
	if task.Metadata.IsSystem {
		http.Error(w, "Cannot update system tasks", http.StatusForbidden)
		return
	}

	// Apply updates
	if request.Name != nil {
		task.Name = *request.Name
	}
	if request.Description != nil {
		task.Description = *request.Description
	}
	if request.Schedule != nil {
		task.Schedule = *request.Schedule
	}
	if request.Priority != nil {
		task.Priority = *request.Priority
	}
	if request.Enabled != nil {
		task.Enabled = *request.Enabled
	}
	if request.Config != nil {
		task.Config = request.Config
	}
	if request.Tags != nil {
		task.Metadata.Tags = request.Tags
	}

	task.UpdatedAt = time.Now()
	task.UpdatedBy = "api" // TODO: Get from authenticated user

	if err := m.repository.UpdateTask(r.Context(), task); err != nil {
		slog.Error("Failed to update task", slog.String("error", err.Error()))
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	// Reload tasks in engine
	m.engine.ReloadTasks()

	slog.Info("Task updated successfully",
		slog.String("task_id", task.ID),
		slog.String("task_name", task.Name))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (m *Module) deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	task, err := m.repository.GetTask(r.Context(), taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		slog.Error("Failed to get task for deletion", slog.String("error", err.Error()))
		http.Error(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	// Prevent deleting system tasks
	if task.Metadata.IsSystem {
		http.Error(w, "Cannot delete system tasks", http.StatusForbidden)
		return
	}

	if err := m.repository.DeleteTask(r.Context(), taskID); err != nil {
		slog.Error("Failed to delete task", slog.String("error", err.Error()))
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	// Reload tasks in engine
	m.engine.ReloadTasks()

	slog.Info("Task deleted successfully", slog.String("task_id", taskID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Task deleted successfully",
	})
}

func (m *Module) startTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	execution, err := m.engine.ExecuteTaskNow(r.Context(), taskID)
	if err != nil {
		slog.Error("Failed to start task", slog.String("task_id", taskID), slog.String("error", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to start task: %v", err), http.StatusInternalServerError)
		return
	}

	response := TaskExecutionResponse{
		ExecutionID: execution.ID,
		Status:      string(execution.Status),
		Message:     "Task started successfully",
		StartedAt:   execution.StartedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) stopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	if err := m.engine.StopTask(taskID); err != nil {
		slog.Error("Failed to stop task", slog.String("task_id", taskID), slog.String("error", err.Error()))
		http.Error(w, fmt.Sprintf("Failed to stop task: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Task stopped successfully",
	})
}

func (m *Module) pauseTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	if err := m.repository.UpdateTaskStatus(r.Context(), taskID, TaskStatusPaused); err != nil {
		slog.Error("Failed to pause task", slog.String("task_id", taskID), slog.String("error", err.Error()))
		http.Error(w, "Failed to pause task", http.StatusInternalServerError)
		return
	}

	// Reload tasks in engine
	m.engine.ReloadTasks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Task paused successfully",
	})
}

func (m *Module) resumeTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	if err := m.repository.UpdateTaskStatus(r.Context(), taskID, TaskStatusPending); err != nil {
		slog.Error("Failed to resume task", slog.String("task_id", taskID), slog.String("error", err.Error()))
		http.Error(w, "Failed to resume task", http.StatusInternalServerError)
		return
	}

	// Reload tasks in engine
	m.engine.ReloadTasks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Task resumed successfully",
	})
}

func (m *Module) getTaskHistoryHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	executions, err := m.repository.GetTaskExecutions(r.Context(), taskID, page, pageSize)
	if err != nil {
		slog.Error("Failed to get task history", slog.String("error", err.Error()))
		http.Error(w, "Failed to get task history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

func (m *Module) getExecutionHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	executionID := chi.URLParam(r, "executionID")
	if taskID == "" || executionID == "" {
		http.Error(w, "Task ID and Execution ID are required", http.StatusBadRequest)
		return
	}

	execution, err := m.repository.GetExecution(r.Context(), executionID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		slog.Error("Failed to get execution", slog.String("error", err.Error()))
		http.Error(w, "Failed to get execution", http.StatusInternalServerError)
		return
	}

	if execution.TaskID != taskID {
		http.Error(w, "Execution does not belong to the specified task", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

func (m *Module) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := m.repository.GetSchedulerStats(r.Context())
	if err != nil {
		slog.Error("Failed to get scheduler stats", slog.String("error", err.Error()))
		http.Error(w, "Failed to get scheduler stats", http.StatusInternalServerError)
		return
	}

	// Add engine stats
	engineStats := m.engine.GetStats()
	stats.WorkerCount = engineStats.WorkerCount
	stats.QueueSize = engineStats.QueueSize

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (m *Module) reloadTasksHandler(w http.ResponseWriter, r *http.Request) {
	m.engine.ReloadTasks()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Tasks reloaded successfully",
	})
}

func (m *Module) getStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"module":  "scheduler",
		"status":  "running",
		"version": "1.0.0",
		"engine":  m.engine.IsRunning(),
	})
}

// validateTaskCreateRequest validates a task creation request
func (m *Module) validateTaskCreateRequest(request *TaskCreateRequest) error {
	if request.Name == "" {
		return fmt.Errorf("name is required")
	}
	if request.Type == "" {
		return fmt.Errorf("type is required")
	}
	if request.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}
	if request.Config == nil {
		return fmt.Errorf("config is required")
	}

	// Validate cron schedule
	if err := validateCronSchedule(request.Schedule); err != nil {
		return fmt.Errorf("invalid schedule: %v", err)
	}

	// Validate config based on task type
	switch request.Type {
	case TaskTypeHTTP:
		if err := validateHTTPConfig(request.Config); err != nil {
			return fmt.Errorf("invalid HTTP config: %v", err)
		}
	case TaskTypeFunction:
		if err := validateFunctionConfig(request.Config); err != nil {
			return fmt.Errorf("invalid function config: %v", err)
		}
	case TaskTypeSystem:
		if err := validateSystemConfig(request.Config); err != nil {
			return fmt.Errorf("invalid system config: %v", err)
		}
	}

	return nil
}

// validateCronSchedule validates a cron schedule expression
func validateCronSchedule(schedule string) error {
	// Validate 6-field cron expression (with seconds)
	parts := strings.Fields(schedule)
	if len(parts) != 6 {
		return fmt.Errorf("cron expression must have 6 fields (seconds minute hour day month dow)")
	}
	
	// Use cron library for proper validation
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	
	return nil
}

// validateHTTPConfig validates HTTP task configuration
func validateHTTPConfig(config map[string]interface{}) error {
	url, ok := config["url"].(string)
	if !ok || url == "" {
		return fmt.Errorf("url is required for HTTP tasks")
	}

	method, ok := config["method"].(string)
	if !ok || method == "" {
		return fmt.Errorf("method is required for HTTP tasks")
	}

	return nil
}

// validateFunctionConfig validates function task configuration
func validateFunctionConfig(config map[string]interface{}) error {
	functionName, ok := config["function_name"].(string)
	if !ok || functionName == "" {
		return fmt.Errorf("function_name is required for function tasks")
	}

	return nil
}

// validateSystemConfig validates system task configuration
func validateSystemConfig(config map[string]interface{}) error {
	taskName, ok := config["task_name"].(string)
	if !ok || taskName == "" {
		return fmt.Errorf("task_name is required for system tasks")
	}

	return nil
}