package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-falcon/pkg/database"

	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// AuthModule interface defines the methods needed from the auth module
type AuthModule interface {
	RefreshExpiringTokens(ctx context.Context, batchSize int) (successCount, failureCount int, err error)
}

// Engine handles task scheduling and execution
type Engine struct {
	repository *Repository
	redis      *database.Redis
	cron       *cron.Cron
	
	// Worker pool
	workers    int
	taskQueue  chan *TaskExecution
	workerWg   sync.WaitGroup
	
	// Task management
	activeTasks   map[string]*Task
	tasksMutex    sync.RWMutex
	
	// Executors
	executors map[TaskType]TaskExecutor
	
	// Control
	running   bool
	runMutex  sync.RWMutex
	stopCh    chan struct{}
	stats     *EngineStats
}

// EngineStats holds engine statistics
type EngineStats struct {
	WorkerCount   int    `json:"worker_count"`
	QueueSize     int    `json:"queue_size"`
	ActiveTasks   int    `json:"active_tasks"`
	TotalExecuted int64  `json:"total_executed"`
	IsRunning     bool   `json:"is_running"`
}

// TaskExecutor defines the interface for task executors
type TaskExecutor interface {
	Execute(ctx context.Context, task *Task) (*TaskResult, error)
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	Success  bool                   `json:"success"`
	Output   string                 `json:"output"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewEngine creates a new scheduler engine
func NewEngine(repository *Repository, redis *database.Redis, authModule AuthModule, sdeModule SDEModule) *Engine {
	// Create cron scheduler with second precision
	cronScheduler := cron.New(cron.WithSeconds())
	
	engine := &Engine{
		repository:  repository,
		redis:       redis,
		cron:        cronScheduler,
		workers:     10, // Default worker count
		taskQueue:   make(chan *TaskExecution, 1000),
		activeTasks: make(map[string]*Task),
		executors:   make(map[TaskType]TaskExecutor),
		stopCh:      make(chan struct{}),
		stats: &EngineStats{
			WorkerCount: 10,
		},
	}
	
	// Register default executors
	engine.registerDefaultExecutors(authModule, sdeModule)
	
	return engine
}

// Start starts the scheduler engine
func (e *Engine) Start(ctx context.Context) {
	e.runMutex.Lock()
	if e.running {
		e.runMutex.Unlock()
		return
	}
	e.running = true
	e.runMutex.Unlock()
	
	slog.Info("Starting scheduler engine", 
		slog.Int("workers", e.workers),
		slog.Int("queue_size", cap(e.taskQueue)))
	
	// Start worker goroutines
	for i := 0; i < e.workers; i++ {
		e.workerWg.Add(1)
		go e.worker(ctx, fmt.Sprintf("worker-%d", i))
	}
	
	// Load and schedule tasks
	e.LoadTasks(ctx)
	
	// Start cron scheduler
	e.cron.Start()
	
	// Start monitoring goroutine
	go e.monitor(ctx)
	
	e.stats.IsRunning = true
	slog.Info("Scheduler engine started successfully")
}

// Stop stops the scheduler engine
func (e *Engine) Stop() {
	e.runMutex.Lock()
	if !e.running {
		e.runMutex.Unlock()
		return
	}
	e.running = false
	e.runMutex.Unlock()
	
	slog.Info("Stopping scheduler engine")
	
	// Stop accepting new tasks
	close(e.stopCh)
	
	// Stop cron scheduler
	cronCtx := e.cron.Stop()
	<-cronCtx.Done()
	
	// Close task queue
	close(e.taskQueue)
	
	// Wait for workers to finish
	e.workerWg.Wait()
	
	e.stats.IsRunning = false
	slog.Info("Scheduler engine stopped")
}

// IsRunning returns whether the engine is running
func (e *Engine) IsRunning() bool {
	e.runMutex.RLock()
	defer e.runMutex.RUnlock()
	return e.running
}

// LoadTasks loads tasks from the database and schedules them
func (e *Engine) LoadTasks(ctx context.Context) {
	slog.Info("Loading tasks from database")
	
	tasks, err := e.repository.GetEnabledTasks(ctx)
	if err != nil {
		slog.Error("Failed to load tasks", slog.String("error", err.Error()))
		return
	}
	
	e.tasksMutex.Lock()
	defer e.tasksMutex.Unlock()
	
	// Clear existing tasks
	e.activeTasks = make(map[string]*Task)
	
	// Schedule each task
	for _, task := range tasks {
		if err := e.scheduleTask(task); err != nil {
			slog.Error("Failed to schedule task",
				slog.String("task_id", task.ID),
				slog.String("task_name", task.Name),
				slog.String("error", err.Error()))
		} else {
			e.activeTasks[task.ID] = task
		}
	}
	
	e.stats.ActiveTasks = len(e.activeTasks)
	slog.Info("Tasks loaded", slog.Int("count", len(e.activeTasks)))
}

// ReloadTasks reloads tasks from the database
func (e *Engine) ReloadTasks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	e.LoadTasks(ctx)
}

// scheduleTask schedules a single task with the cron scheduler
func (e *Engine) scheduleTask(task *Task) error {
	if task.Schedule == "" {
		return fmt.Errorf("task has no schedule")
	}
	
	// Parse and validate schedule (6-field format with seconds)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(task.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}
	
	// Add to cron scheduler
	_, err = e.cron.AddFunc(task.Schedule, func() {
		e.executeTask(context.Background(), task)
	})
	
	if err != nil {
		return fmt.Errorf("failed to add task to cron: %w", err)
	}
	
	// Calculate next run time
	schedule, _ := parser.Parse(task.Schedule)
	nextRun := schedule.Next(time.Now())
	
	// Update next run time in database
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := e.repository.UpdateTaskRun(ctx, task.ID, nil, &nextRun); err != nil {
			slog.Error("Failed to update next run time",
				slog.String("task_id", task.ID),
				slog.String("error", err.Error()))
		}
	}()
	
	slog.Debug("Task scheduled",
		slog.String("task_id", task.ID),
		slog.String("task_name", task.Name),
		slog.String("schedule", task.Schedule),
		slog.Time("next_run", nextRun))
	
	return nil
}

// executeTask executes a task (called by cron scheduler)
func (e *Engine) executeTask(ctx context.Context, task *Task) {
	// Check if engine is still running
	e.runMutex.RLock()
	running := e.running
	e.runMutex.RUnlock()
	
	if !running {
		return
	}
	
	// Try to acquire distributed lock
	lockKey := fmt.Sprintf("scheduler:lock:%s", task.ID)
	lockValue := uuid.New().String()
	
	// Try to acquire lock with task timeout
	timeout := task.Metadata.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute // Default timeout
	}
	
	acquired, err := e.acquireLock(ctx, lockKey, lockValue, timeout)
	if err != nil {
		slog.Error("Failed to acquire lock for task",
			slog.String("task_id", task.ID),
			slog.String("error", err.Error()))
		return
	}
	
	if !acquired {
		slog.Debug("Task already running, skipping",
			slog.String("task_id", task.ID),
			slog.String("task_name", task.Name))
		return
	}
	
	// Create execution record
	execution := &TaskExecution{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
		Metadata: map[string]interface{}{
			"lock_key":   lockKey,
			"lock_value": lockValue,
		},
	}
	
	// Create execution in database
	if err := e.repository.CreateExecution(ctx, execution); err != nil {
		slog.Error("Failed to create execution record",
			slog.String("task_id", task.ID),
			slog.String("error", err.Error()))
		e.releaseLock(ctx, lockKey, lockValue)
		return
	}
	
	// Add to worker queue
	select {
	case e.taskQueue <- execution:
		slog.Debug("Task queued for execution",
			slog.String("task_id", task.ID),
			slog.String("execution_id", execution.ID))
	default:
		// Queue is full
		slog.Warn("Task queue is full, dropping task",
			slog.String("task_id", task.ID),
			slog.String("execution_id", execution.ID))
		
		// Mark execution as failed
		now := time.Now()
		execution.Status = TaskStatusFailed
		execution.CompletedAt = &now
		execution.Duration = time.Since(execution.StartedAt)
		execution.Error = "Task queue is full"
		
		e.repository.UpdateExecution(ctx, execution)
		e.releaseLock(ctx, lockKey, lockValue)
	}
}

// ExecuteTaskNow executes a task immediately (manual trigger)
func (e *Engine) ExecuteTaskNow(ctx context.Context, taskID string) (*TaskExecution, error) {
	// Get task from database
	task, err := e.repository.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	
	if !task.Enabled {
		return nil, fmt.Errorf("task is disabled")
	}
	
	// Try to acquire distributed lock
	lockKey := fmt.Sprintf("scheduler:lock:%s", task.ID)
	lockValue := uuid.New().String()
	
	timeout := task.Metadata.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	
	acquired, err := e.acquireLock(ctx, lockKey, lockValue, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	if !acquired {
		return nil, fmt.Errorf("task is already running")
	}
	
	// Create execution record
	execution := &TaskExecution{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Status:    TaskStatusRunning,
		StartedAt: time.Now(),
		Metadata: map[string]interface{}{
			"lock_key":   lockKey,
			"lock_value": lockValue,
			"manual":     true,
		},
	}
	
	// Create execution in database
	if err := e.repository.CreateExecution(ctx, execution); err != nil {
		e.releaseLock(ctx, lockKey, lockValue)
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}
	
	// Add to worker queue
	select {
	case e.taskQueue <- execution:
		slog.Info("Task manually triggered",
			slog.String("task_id", task.ID),
			slog.String("execution_id", execution.ID))
		return execution, nil
	default:
		// Queue is full
		now := time.Now()
		execution.Status = TaskStatusFailed
		execution.CompletedAt = &now
		execution.Duration = time.Since(execution.StartedAt)
		execution.Error = "Task queue is full"
		
		e.repository.UpdateExecution(ctx, execution)
		e.releaseLock(ctx, lockKey, lockValue)
		
		return nil, fmt.Errorf("task queue is full")
	}
}

// StopTask stops a running task
func (e *Engine) StopTask(taskID string) error {
	// This would require more complex worker coordination
	// For now, we'll just update the task status
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	return e.repository.UpdateTaskStatus(ctx, taskID, TaskStatusPaused)
}

// worker processes tasks from the queue
func (e *Engine) worker(ctx context.Context, workerID string) {
	defer e.workerWg.Done()
	
	slog.Debug("Worker started", slog.String("worker_id", workerID))
	
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Worker stopped due to context cancellation", slog.String("worker_id", workerID))
			return
		case <-e.stopCh:
			slog.Debug("Worker stopped", slog.String("worker_id", workerID))
			return
		case execution, ok := <-e.taskQueue:
			if !ok {
				slog.Debug("Worker stopped - queue closed", slog.String("worker_id", workerID))
				return
			}
			
			execution.WorkerID = workerID
			e.processExecution(ctx, execution)
		}
	}
}

// processExecution processes a single task execution
func (e *Engine) processExecution(ctx context.Context, execution *TaskExecution) {
	startTime := time.Now()
	
	slog.Info("Processing task execution",
		slog.String("execution_id", execution.ID),
		slog.String("task_id", execution.TaskID),
		slog.String("worker_id", execution.WorkerID))
	
	// Get task details
	task, err := e.repository.GetTask(ctx, execution.TaskID)
	if err != nil {
		e.completeExecution(ctx, execution, &TaskResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get task: %v", err),
		})
		return
	}
	
	// Get executor for task type
	executor, exists := e.executors[task.Type]
	if !exists {
		e.completeExecution(ctx, execution, &TaskResult{
			Success: false,
			Error:   fmt.Sprintf("No executor found for task type: %s", task.Type),
		})
		return
	}
	
	// Create timeout context
	timeout := task.Metadata.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Execute task
	result, err := executor.Execute(execCtx, task)
	if err != nil {
		result = &TaskResult{
			Success: false,
			Error:   err.Error(),
		}
	}
	
	// Update execution times
	execution.Duration = time.Since(startTime)
	
	// Complete execution
	e.completeExecution(ctx, execution, result)
	
	e.stats.TotalExecuted++
	
	slog.Info("Task execution completed",
		slog.String("execution_id", execution.ID),
		slog.String("task_id", execution.TaskID),
		slog.Bool("success", result.Success),
		slog.Duration("duration", execution.Duration))
}

// completeExecution completes a task execution and updates the database
func (e *Engine) completeExecution(ctx context.Context, execution *TaskExecution, result *TaskResult) {
	now := time.Now()
	execution.CompletedAt = &now
	
	if result.Success {
		execution.Status = TaskStatusCompleted
		execution.Output = result.Output
	} else {
		execution.Status = TaskStatusFailed
		execution.Error = result.Error
	}
	
	// Update execution in database
	if err := e.repository.UpdateExecution(ctx, execution); err != nil {
		slog.Error("Failed to update execution",
			slog.String("execution_id", execution.ID),
			slog.String("error", err.Error()))
	}
	
	// Update task last run time
	if err := e.repository.UpdateTaskRun(ctx, execution.TaskID, &now, nil); err != nil {
		slog.Error("Failed to update task last run time",
			slog.String("task_id", execution.TaskID),
			slog.String("error", err.Error()))
	}
	
	// Release distributed lock
	if lockKey, ok := execution.Metadata["lock_key"].(string); ok {
		if lockValue, ok := execution.Metadata["lock_value"].(string); ok {
			e.releaseLock(ctx, lockKey, lockValue)
		}
	}
}

// monitor monitors the engine health and stats
func (e *Engine) monitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.updateStats()
		}
	}
}

// updateStats updates engine statistics
func (e *Engine) updateStats() {
	e.tasksMutex.RLock()
	activeCount := len(e.activeTasks)
	e.tasksMutex.RUnlock()
	
	e.stats.ActiveTasks = activeCount
	e.stats.QueueSize = len(e.taskQueue)
	e.stats.WorkerCount = e.workers
}

// GetStats returns current engine statistics
func (e *Engine) GetStats() *EngineStats {
	e.updateStats()
	return e.stats
}

// acquireLock acquires a distributed lock using Redis
func (e *Engine) acquireLock(ctx context.Context, key, value string, timeout time.Duration) (bool, error) {
	client := e.redis.Client
	
	result, err := client.SetNX(ctx, key, value, timeout).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	return result, nil
}

// releaseLock releases a distributed lock using Redis
func (e *Engine) releaseLock(ctx context.Context, key, value string) error {
	client := e.redis.Client
	
	// Use Lua script to ensure we only release our own lock
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	
	_, err := script.Run(ctx, client, []string{key}, value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	
	return nil
}

// registerDefaultExecutors registers the default task executors
func (e *Engine) registerDefaultExecutors(authModule AuthModule, sdeModule SDEModule) {
	e.executors[TaskTypeHTTP] = &HTTPExecutor{}
	e.executors[TaskTypeFunction] = &FunctionExecutor{}
	e.executors[TaskTypeSystem] = NewSystemExecutor(authModule, sdeModule)
}

// RegisterExecutor registers a custom task executor
func (e *Engine) RegisterExecutor(taskType TaskType, executor TaskExecutor) {
	e.executors[taskType] = executor
}