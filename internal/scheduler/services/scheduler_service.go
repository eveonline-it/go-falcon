package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/internal/scheduler/models"
	"go-falcon/pkg/database"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// SchedulerService is the main service that orchestrates scheduler operations
type SchedulerService struct {
	repository        *Repository
	engineService     *EngineService
	authModule        AuthModule
	characterModule   CharacterModule
	allianceModule    AllianceModule
	corporationModule CorporationModule
}

// NewSchedulerService creates a new scheduler service with all dependencies
func NewSchedulerService(mongodb *database.MongoDB, redis *database.Redis, authModule AuthModule, characterModule CharacterModule, allianceModule AllianceModule, corporationModule CorporationModule) *SchedulerService {
	repository := NewRepository(mongodb)
	engineService := NewEngineService(repository, redis, authModule, characterModule, allianceModule, corporationModule)

	return &SchedulerService{
		repository:        repository,
		engineService:     engineService,
		authModule:        authModule,
		characterModule:   characterModule,
		allianceModule:    allianceModule,
		corporationModule: corporationModule,
	}
}

// Task Management

// CreateTask creates a new task
func (s *SchedulerService) CreateTask(ctx context.Context, req *dto.TaskCreateRequest) (*dto.TaskResponse, error) {
	// Validate request
	if err := s.validateTaskCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create task model
	now := time.Now()
	task := &models.Task{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Schedule:    req.Schedule,
		Status:      models.TaskStatusPending,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		Config:      req.Config,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "api", // TODO: Get from authenticated user
	}

	// Set metadata
	if req.Metadata != nil {
		task.Metadata = *req.Metadata
	} else {
		task.Metadata = models.TaskMetadata{
			MaxRetries:    3,
			RetryInterval: 1 * time.Minute,
			Timeout:       5 * time.Minute,
			Tags:          req.Tags,
			IsSystem:      false,
			Source:        "api",
			Version:       1,
		}
	}

	if err := s.repository.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Reload tasks in engine
	s.engineService.ReloadTasks()

	slog.Info("Task created successfully",
		slog.String("task_id", task.ID),
		slog.String("task_name", task.Name))

	return s.taskToDTO(task), nil
}

// GetTask retrieves a task by ID
func (s *SchedulerService) GetTask(ctx context.Context, taskID string) (*dto.TaskResponse, error) {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return s.taskToDTO(task), nil
}

// UpdateTask updates an existing task
func (s *SchedulerService) UpdateTask(ctx context.Context, taskID string, req *dto.TaskUpdateRequest) (*dto.TaskResponse, error) {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Prevent updating system tasks
	if task.Metadata.IsSystem {
		return nil, fmt.Errorf("cannot update system tasks")
	}

	// Apply updates
	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Schedule != nil {
		task.Schedule = *req.Schedule
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}
	if req.Config != nil {
		task.Config = req.Config
	}
	if req.Tags != nil {
		task.Metadata.Tags = req.Tags
	}

	task.UpdatedAt = time.Now()
	task.UpdatedBy = "api" // TODO: Get from authenticated user

	if err := s.repository.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Reload tasks in engine
	s.engineService.ReloadTasks()

	slog.Info("Task updated successfully",
		slog.String("task_id", task.ID),
		slog.String("task_name", task.Name))

	return s.taskToDTO(task), nil
}

// DeleteTask deletes a task
func (s *SchedulerService) DeleteTask(ctx context.Context, taskID string) error {
	task, err := s.repository.GetTask(ctx, taskID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("task not found")
		}
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Prevent deleting system tasks
	if task.Metadata.IsSystem {
		return fmt.Errorf("cannot delete system tasks")
	}

	if err := s.repository.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Reload tasks in engine
	s.engineService.ReloadTasks()

	slog.Info("Task deleted successfully", slog.String("task_id", taskID))
	return nil
}

// ListTasks lists tasks with filtering and pagination
func (s *SchedulerService) ListTasks(ctx context.Context, query *dto.TaskListQuery) (*dto.TaskListResponse, error) {
	// Build filter
	filter := bson.M{}
	if query.Status != "" {
		filter["status"] = query.Status
	}
	if query.Type != "" {
		filter["type"] = query.Type
	}
	if query.Enabled != "" {
		if query.Enabled == "true" {
			filter["enabled"] = true
		} else if query.Enabled == "false" {
			filter["enabled"] = false
		}
	}
	if len(query.Tags) > 0 {
		filter["metadata.tags"] = bson.M{"$in": query.Tags}
	}

	tasks, total, err := s.repository.ListTasks(ctx, filter, query.Page, query.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Convert to DTOs
	taskDTOs := make([]dto.TaskResponse, len(tasks))
	for i, task := range tasks {
		taskDTOs[i] = *s.taskToDTO(&task)
	}

	response := &dto.TaskListResponse{
		Tasks:      taskDTOs,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: int((total + int64(query.PageSize) - 1) / int64(query.PageSize)),
	}

	return response, nil
}

// Task Control

// StartTask manually executes a task immediately
func (s *SchedulerService) StartTask(ctx context.Context, taskID string) (*dto.TaskExecutionResponse, error) {
	execution, err := s.engineService.ExecuteTaskNow(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	return &dto.TaskExecutionResponse{
		ExecutionID: execution.ID,
		Status:      string(execution.Status),
		Message:     "Task started successfully",
		StartedAt:   execution.StartedAt,
	}, nil
}

// StopTask stops a currently running task
func (s *SchedulerService) StopTask(ctx context.Context, taskID string) error {
	if err := s.engineService.StopTask(taskID); err != nil {
		return fmt.Errorf("failed to stop task: %w", err)
	}
	return nil
}

// PauseTask pauses a task
func (s *SchedulerService) PauseTask(ctx context.Context, taskID string) error {
	if err := s.repository.UpdateTaskStatus(ctx, taskID, models.TaskStatusPaused); err != nil {
		return fmt.Errorf("failed to pause task: %w", err)
	}

	// Reload tasks in engine
	s.engineService.ReloadTasks()
	return nil
}

// ResumeTask resumes a paused task
func (s *SchedulerService) ResumeTask(ctx context.Context, taskID string) error {
	if err := s.repository.UpdateTaskStatus(ctx, taskID, models.TaskStatusPending); err != nil {
		return fmt.Errorf("failed to resume task: %w", err)
	}

	// Reload tasks in engine
	s.engineService.ReloadTasks()
	return nil
}

// Execution History

// GetTaskExecutions retrieves execution history for a task
func (s *SchedulerService) GetTaskExecutions(ctx context.Context, taskID string, query *dto.TaskExecutionQuery) (*dto.ExecutionListResponse, error) {
	executions, err := s.repository.GetTaskExecutions(ctx, taskID, query.Page, query.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get task executions: %w", err)
	}

	// Convert to DTOs
	executionDTOs := make([]dto.ExecutionResponse, len(executions))
	for i, execution := range executions {
		executionDTOs[i] = s.executionToDTO(&execution)
	}

	// For simplicity, we're not getting total count here
	// In a real implementation, you might want to add a count query
	response := &dto.ExecutionListResponse{
		Executions: executionDTOs,
		Total:      int64(len(executions)), // This is not accurate, but good enough for now
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: 1, // This should be calculated properly
	}

	return response, nil
}

// GetExecution retrieves a specific execution
func (s *SchedulerService) GetExecution(ctx context.Context, executionID string) (*dto.ExecutionResponse, error) {
	execution, err := s.repository.GetExecution(ctx, executionID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("execution not found")
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	dto := s.executionToDTO(execution)
	return &dto, nil
}

// ListExecutions retrieves all executions with filtering and pagination
func (s *SchedulerService) ListExecutions(ctx context.Context, query *dto.ExecutionListInput) (*dto.ExecutionListResponse, error) {
	// Build filter
	filter := bson.M{}
	
	if query.Status != "" {
		filter["status"] = query.Status
	}
	
	if query.TaskID != "" {
		filter["task_id"] = query.TaskID
	}

	// Set defaults
	page := query.Page
	pageSize := query.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}

	executions, total, err := s.repository.ListExecutions(ctx, filter, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	// Convert to DTOs
	executionDTOs := make([]dto.ExecutionResponse, len(executions))
	for i, execution := range executions {
		executionDTOs[i] = s.executionToDTO(&execution)
	}

	response := &dto.ExecutionListResponse{
		Executions: executionDTOs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: int((total + int64(pageSize) - 1) / int64(pageSize)),
	}

	return response, nil
}

// Statistics

// GetStats retrieves scheduler statistics
func (s *SchedulerService) GetStats(ctx context.Context) (*dto.SchedulerStatsResponse, error) {
	stats, err := s.repository.GetSchedulerStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler stats: %w", err)
	}

	// Add engine stats
	engineStats := s.engineService.GetStats()
	stats.WorkerCount = engineStats.WorkerCount
	stats.QueueSize = engineStats.QueueSize

	return &dto.SchedulerStatsResponse{
		TotalTasks:       stats.TotalTasks,
		EnabledTasks:     stats.EnabledTasks,
		RunningTasks:     stats.RunningTasks,
		CompletedToday:   stats.CompletedToday,
		FailedToday:      stats.FailedToday,
		AverageRuntime:   stats.AverageRuntime,
		NextScheduledRun: stats.NextScheduledRun,
		WorkerCount:      stats.WorkerCount,
		QueueSize:        stats.QueueSize,
	}, nil
}

// GetStatus returns scheduler status
func (s *SchedulerService) GetStatus() *dto.SchedulerStatusResponse {
	return &dto.SchedulerStatusResponse{
		Module:  "scheduler",
		Status:  "running",
		Version: "1.0.0",
		Engine:  s.engineService.IsRunning(),
	}
}

// ReloadTasks reloads tasks from database
func (s *SchedulerService) ReloadTasks() error {
	return s.engineService.ReloadTasks()
}

// Engine Management

// StartEngine starts the scheduler engine
func (s *SchedulerService) StartEngine(ctx context.Context) error {
	return s.engineService.Start(ctx)
}

// StopEngine stops the scheduler engine
func (s *SchedulerService) StopEngine() error {
	return s.engineService.Stop()
}

// System Tasks

// InitializeSystemTasks creates hardcoded system tasks if they don't exist
func (s *SchedulerService) InitializeSystemTasks(ctx context.Context) error {
	systemTasks := GetSystemTasks()

	for _, task := range systemTasks {
		existing, err := s.repository.GetTask(ctx, task.ID)
		if err != nil && err != mongo.ErrNoDocuments {
			slog.Error("Failed to check existing system task",
				slog.String("task_id", task.ID),
				slog.String("error", err.Error()))
			continue
		}

		if existing == nil {
			// Task doesn't exist, create it
			if err := s.repository.CreateTask(ctx, task); err != nil {
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

				if err := s.repository.UpdateTask(ctx, existing); err != nil {
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

	return nil
}

// Validation

// validateTaskCreateRequest validates a task creation request
func (s *SchedulerService) validateTaskCreateRequest(request *dto.TaskCreateRequest) error {
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

	// Validate config based on task type
	switch request.Type {
	case models.TaskTypeHTTP:
		if err := s.validateHTTPConfig(request.Config); err != nil {
			return fmt.Errorf("invalid HTTP config: %v", err)
		}
	case models.TaskTypeFunction:
		if err := s.validateFunctionConfig(request.Config); err != nil {
			return fmt.Errorf("invalid function config: %v", err)
		}
	case models.TaskTypeSystem:
		if err := s.validateSystemConfig(request.Config); err != nil {
			return fmt.Errorf("invalid system config: %v", err)
		}
	}

	return nil
}

// validateHTTPConfig validates HTTP task configuration
func (s *SchedulerService) validateHTTPConfig(config map[string]interface{}) error {
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
func (s *SchedulerService) validateFunctionConfig(config map[string]interface{}) error {
	functionName, ok := config["function_name"].(string)
	if !ok || functionName == "" {
		return fmt.Errorf("function_name is required for function tasks")
	}

	return nil
}

// validateSystemConfig validates system task configuration
func (s *SchedulerService) validateSystemConfig(config map[string]interface{}) error {
	taskName, ok := config["task_name"].(string)
	if !ok || taskName == "" {
		return fmt.Errorf("task_name is required for system tasks")
	}

	return nil
}

// DTO Conversion

// taskToDTO converts a task model to DTO
func (s *SchedulerService) taskToDTO(task *models.Task) *dto.TaskResponse {
	return &dto.TaskResponse{
		ID:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		Type:        task.Type,
		Schedule:    task.Schedule,
		Status:      task.Status,
		Priority:    task.Priority,
		Enabled:     task.Enabled,
		Config:      task.Config,
		Metadata:    task.Metadata,
		LastRun:     task.LastRun,
		NextRun:     task.NextRun,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		CreatedBy:   task.CreatedBy,
		UpdatedBy:   task.UpdatedBy,
	}
}

// executionToDTO converts an execution model to DTO
func (s *SchedulerService) executionToDTO(execution *models.TaskExecution) dto.ExecutionResponse {
	return dto.ExecutionResponse{
		ID:          execution.ID,
		TaskID:      execution.TaskID,
		Status:      execution.Status,
		StartedAt:   execution.StartedAt,
		CompletedAt: execution.CompletedAt,
		Duration:    execution.Duration,
		Output:      execution.Output,
		Error:       execution.Error,
		Metadata:    execution.Metadata,
		WorkerID:    execution.WorkerID,
		RetryCount:  execution.RetryCount,
	}
}

// Utility functions

// parseQueryInt parses an integer from query string with default value
func parseQueryInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return defaultValue
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}