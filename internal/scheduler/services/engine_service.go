package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-falcon/internal/scheduler/models"
	"go-falcon/pkg/database"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// ExecutionContext holds cancellation context for a running execution
type ExecutionContext struct {
	Execution *models.TaskExecution
	Cancel    context.CancelFunc
	Context   context.Context
}

// EngineService handles task scheduling and execution
type EngineService struct {
	repository *Repository
	redis      *database.Redis
	cron       *cron.Cron
	
	// Worker pool
	workers    int
	taskQueue  chan *models.TaskExecution
	workerWg   sync.WaitGroup
	
	// Task management
	activeTasks   map[string]*models.Task
	tasksMutex    sync.RWMutex
	
	// Execution tracking with cancellation
	runningExecutions map[string]*ExecutionContext
	executionsMutex   sync.RWMutex
	
	// Executors
	executors map[models.TaskType]TaskExecutor
	
	// Engine state
	running   bool
	runMutex  sync.RWMutex
	stopChan  chan struct{}
	
	// Module dependencies
	authModule      AuthModule
	characterModule CharacterModule
}

// AuthModule interface defines the methods needed from the auth module
type AuthModule interface {
	RefreshExpiringTokens(ctx context.Context, batchSize int) (successCount, failureCount int, err error)
}

// TaskExecutor interface for different task types
type TaskExecutor interface {
	Execute(ctx context.Context, task *models.Task) (*models.TaskResult, error)
}

// NewEngineService creates a new scheduler engine
func NewEngineService(repository *Repository, redis *database.Redis, authModule AuthModule, characterModule CharacterModule) *EngineService {
	engine := &EngineService{
		repository:        repository,
		redis:             redis,
		workers:           10, // Default worker count
		taskQueue:         make(chan *models.TaskExecution, 1000),
		activeTasks:       make(map[string]*models.Task),
		runningExecutions: make(map[string]*ExecutionContext),
		executors:         make(map[models.TaskType]TaskExecutor),
		stopChan:          make(chan struct{}),
		authModule:        authModule,
		characterModule:   characterModule,
	}

	// Initialize cron scheduler
	engine.cron = cron.New(cron.WithSeconds())

	// Register built-in executors
	engine.registerBuiltinExecutors()

	return engine
}

// Start starts the scheduler engine
func (e *EngineService) Start(ctx context.Context) error {
	e.runMutex.Lock()
	defer e.runMutex.Unlock()

	if e.running {
		return fmt.Errorf("engine is already running")
	}

	slog.Info("Starting scheduler engine", slog.Int("workers", e.workers))

	// Start worker pool
	for i := 0; i < e.workers; i++ {
		e.workerWg.Add(1)
		go e.worker(ctx, fmt.Sprintf("worker-%d", i))
	}

	// Load and schedule tasks
	if err := e.loadTasks(ctx); err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Start cron scheduler
	e.cron.Start()

	e.running = true
	slog.Info("Scheduler engine started successfully")

	return nil
}

// Stop stops the scheduler engine
func (e *EngineService) Stop() error {
	e.runMutex.Lock()
	defer e.runMutex.Unlock()

	if !e.running {
		return nil
	}

	slog.Info("Stopping scheduler engine")

	// Stop cron scheduler
	cronCtx := e.cron.Stop()
	<-cronCtx.Done()

	// Signal workers to stop
	close(e.stopChan)

	// Close task queue
	close(e.taskQueue)

	// Wait for workers to finish
	e.workerWg.Wait()

	e.running = false
	slog.Info("Scheduler engine stopped")

	return nil
}

// IsRunning returns whether the engine is running
func (e *EngineService) IsRunning() bool {
	e.runMutex.RLock()
	defer e.runMutex.RUnlock()
	return e.running
}

// ReloadTasks reloads tasks from the database
func (e *EngineService) ReloadTasks() error {
	ctx := context.Background()
	return e.loadTasks(ctx)
}

// ExecuteTaskNow manually executes a task immediately
func (e *EngineService) ExecuteTaskNow(ctx context.Context, taskID string) (*models.TaskExecution, error) {
	// Get task from database
	task, err := e.repository.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if !task.Enabled {
		return nil, fmt.Errorf("task is disabled")
	}

	// Create execution record
	execution := &models.TaskExecution{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Status:    models.TaskStatusPending,
		StartedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Queue for execution
	select {
	case e.taskQueue <- execution:
		slog.Info("Task queued for immediate execution", slog.String("task_id", taskID))
		return execution, nil
	default:
		return nil, fmt.Errorf("task queue is full")
	}
}

// StopTask stops a currently running task and cancels any running executions
func (e *EngineService) StopTask(taskID string) error {
	slog.Info("Stopping task", slog.String("task_id", taskID))
	
	// Find and cancel all running executions for this task
	var cancelledExecutions []string
	
	e.executionsMutex.Lock()
	for executionID, execContext := range e.runningExecutions {
		if execContext.Execution.TaskID == taskID {
			slog.Info("Cancelling running execution",
				slog.String("task_id", taskID),
				slog.String("execution_id", executionID))
			
			// Cancel the execution context
			execContext.Cancel()
			cancelledExecutions = append(cancelledExecutions, executionID)
		}
	}
	e.executionsMutex.Unlock()
	
	// Update task status to paused to prevent future scheduling
	ctx := context.Background()
	if err := e.repository.UpdateTaskStatus(ctx, taskID, models.TaskStatusPaused); err != nil {
		return fmt.Errorf("failed to pause task: %w", err)
	}
	
	slog.Info("Task stopped successfully",
		slog.String("task_id", taskID),
		slog.Int("cancelled_executions", len(cancelledExecutions)))
	
	return nil
}

// GetRunningExecutions returns information about currently running executions
func (e *EngineService) GetRunningExecutions() map[string]*ExecutionContext {
	e.executionsMutex.RLock()
	defer e.executionsMutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]*ExecutionContext)
	for id, execContext := range e.runningExecutions {
		result[id] = execContext
	}
	return result
}

// GetStats returns engine statistics
func (e *EngineService) GetStats() models.EngineStats {
	e.runMutex.RLock()
	defer e.runMutex.RUnlock()

	return models.EngineStats{
		WorkerCount: e.workers,
		QueueSize:   len(e.taskQueue),
		IsRunning:   e.running,
	}
}

// loadTasks loads active tasks from database and schedules them
func (e *EngineService) loadTasks(ctx context.Context) error {
	tasks, err := e.repository.GetActiveTasks(ctx)
	if err != nil {
		return err
	}

	e.tasksMutex.Lock()
	defer e.tasksMutex.Unlock()

	// Clear existing cron entries
	for _, entry := range e.cron.Entries() {
		e.cron.Remove(entry.ID)
	}

	// Clear active tasks
	e.activeTasks = make(map[string]*models.Task)

	// Schedule each task
	for _, task := range tasks {
		if err := e.scheduleTask(&task); err != nil {
			slog.Error("Failed to schedule task", 
				slog.String("task_id", task.ID),
				slog.String("error", err.Error()))
			continue
		}
		e.activeTasks[task.ID] = &task
	}

	slog.Info("Tasks loaded and scheduled", slog.Int("count", len(tasks)))
	return nil
}

// scheduleTask schedules a single task with cron
func (e *EngineService) scheduleTask(task *models.Task) error {
	if task.Schedule == "" {
		return fmt.Errorf("task has no schedule")
	}

	// Create task execution function
	taskFunc := func() {
		e.executeScheduledTask(task.ID)
	}

	// Add to cron scheduler
	_, err := e.cron.AddFunc(task.Schedule, taskFunc)
	if err != nil {
		return fmt.Errorf("invalid cron schedule '%s': %w", task.Schedule, err)
	}

	// Calculate next run time
	schedule, err := cron.ParseStandard(task.Schedule)
	if err == nil {
		nextRun := schedule.Next(time.Now())
		e.repository.UpdateTaskRun(context.Background(), task.ID, nil, &nextRun)
	}

	return nil
}

// executeScheduledTask executes a task from the cron scheduler
func (e *EngineService) executeScheduledTask(taskID string) {
	// Create execution record
	execution := &models.TaskExecution{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Status:    models.TaskStatusPending,
		StartedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Queue for execution
	select {
	case e.taskQueue <- execution:
		// Successfully queued
	default:
		slog.Warn("Task queue full, skipping execution", slog.String("task_id", taskID))
	}
}

// worker processes tasks from the queue
func (e *EngineService) worker(ctx context.Context, workerID string) {
	defer e.workerWg.Done()

	slog.Info("Worker started", slog.String("worker_id", workerID))

	for {
		select {
		case <-e.stopChan:
			slog.Info("Worker stopping", slog.String("worker_id", workerID))
			return
		case <-ctx.Done():
			slog.Info("Worker stopping due to context cancellation", slog.String("worker_id", workerID))
			return
		case execution, ok := <-e.taskQueue:
			if !ok {
				slog.Info("Task queue closed, worker stopping", slog.String("worker_id", workerID))
				return
			}

			// Process the execution
			e.processExecution(ctx, execution, workerID)
		}
	}
}

// processExecution processes a single task execution
func (e *EngineService) processExecution(parentCtx context.Context, execution *models.TaskExecution, workerID string) {
	execution.WorkerID = workerID
	execution.Status = models.TaskStatusRunning

	// Create cancellable context for this execution
	executionCtx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// Track this execution for cancellation
	execContext := &ExecutionContext{
		Execution: execution,
		Cancel:    cancel,
		Context:   executionCtx,
	}

	e.executionsMutex.Lock()
	e.runningExecutions[execution.ID] = execContext
	e.executionsMutex.Unlock()

	// Defer cleanup of running execution tracking
	defer func() {
		e.executionsMutex.Lock()
		delete(e.runningExecutions, execution.ID)
		e.executionsMutex.Unlock()
	}()

	// Save execution start
	if err := e.repository.CreateExecution(parentCtx, execution); err != nil {
		slog.Error("Failed to create execution record", 
			slog.String("execution_id", execution.ID),
			slog.String("error", err.Error()))
		return
	}

	// Get task details
	task, err := e.repository.GetTask(parentCtx, execution.TaskID)
	if err != nil {
		execution.Status = models.TaskStatusFailed
		execution.Error = fmt.Sprintf("Failed to get task: %v", err)
		e.finishExecution(parentCtx, execution)
		return
	}

	// Update task status
	e.repository.UpdateTaskStatus(parentCtx, task.ID, models.TaskStatusRunning)

	// Execute the task with cancellable context
	result := e.executeTask(executionCtx, task)

	// Check if execution was cancelled
	if executionCtx.Err() == context.Canceled {
		execution.Status = models.TaskStatusFailed
		execution.Error = "Task execution was cancelled"
		execution.Output = "Execution stopped by user request"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Duration = now.Sub(execution.StartedAt)
		
		slog.Info("Task execution cancelled",
			slog.String("task_id", task.ID),
			slog.String("execution_id", execution.ID))
	} else {
		// Update execution with result
		execution.CompletedAt = &result.CompletedAt
		execution.Duration = result.Duration
		execution.Output = result.Output
		execution.Error = result.Error

		if result.Success {
			execution.Status = models.TaskStatusCompleted
		} else {
			execution.Status = models.TaskStatusFailed
		}
	}

	// Finish execution
	e.finishExecution(parentCtx, execution)

	// Update task status
	e.repository.UpdateTaskStatus(parentCtx, task.ID, models.TaskStatusPending)

	// Update task run time
	now := time.Now()
	var nextRun *time.Time
	if schedule, err := cron.ParseStandard(task.Schedule); err == nil {
		next := schedule.Next(now)
		nextRun = &next
	}
	e.repository.UpdateTaskRun(parentCtx, task.ID, &now, nextRun)
}

// executeTask executes a task and returns the result
func (e *EngineService) executeTask(ctx context.Context, task *models.Task) *ExecutionResult {
	start := time.Now()

	// Get executor for task type
	executor, exists := e.executors[task.Type]
	if !exists {
		return &ExecutionResult{
			Success:     false,
			Error:       fmt.Sprintf("No executor found for task type: %s", task.Type),
			Duration:    time.Since(start),
			CompletedAt: time.Now(),
		}
	}

	// Execute with timeout
	taskCtx, cancel := context.WithTimeout(ctx, task.Metadata.Timeout)
	defer cancel()

	result, err := executor.Execute(taskCtx, task)
	if err != nil {
		return &ExecutionResult{
			Success:     false,
			Error:       err.Error(),
			Output:      "",
			Duration:    time.Since(start),
			CompletedAt: time.Now(),
		}
	}

	return &ExecutionResult{
		Success:     result.Success,
		Error:       result.Error,
		Output:      result.Output,
		Duration:    time.Since(start),
		CompletedAt: time.Now(),
	}
}

// finishExecution finalizes an execution record
func (e *EngineService) finishExecution(ctx context.Context, execution *models.TaskExecution) {
	if err := e.repository.UpdateExecution(ctx, execution); err != nil {
		slog.Error("Failed to update execution record",
			slog.String("execution_id", execution.ID),
			slog.String("error", err.Error()))
	}
}

// registerBuiltinExecutors registers the built-in task executors
func (e *EngineService) registerBuiltinExecutors() {
	e.executors[models.TaskTypeHTTP] = NewHTTPExecutor()
	e.executors[models.TaskTypeSystem] = NewSystemExecutor(e.authModule, e.characterModule)
	e.executors[models.TaskTypeFunction] = NewFunctionExecutor()
}

// ExecutionResult represents the result of a task execution
type ExecutionResult struct {
	Success     bool
	Error       string
	Output      string
	Duration    time.Duration
	CompletedAt time.Time
}