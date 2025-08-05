package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	TasksCollection      = "scheduler_tasks"
	ExecutionsCollection = "scheduler_executions"
)

// Repository handles database operations for scheduler
type Repository struct {
	mongodb *database.MongoDB
}

// NewRepository creates a new scheduler repository
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb: mongodb,
	}
}

// CreateTask creates a new task in the database
func (r *Repository) CreateTask(ctx context.Context, task *Task) error {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	_, err := collection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID
func (r *Repository) GetTask(ctx context.Context, taskID string) (*Task, error) {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	var task Task
	err := collection.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

// UpdateTask updates an existing task
func (r *Repository) UpdateTask(ctx context.Context, task *Task) error {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	update := bson.M{
		"$set": bson.M{
			"name":        task.Name,
			"description": task.Description,
			"type":        task.Type,
			"schedule":    task.Schedule,
			"status":      task.Status,
			"priority":    task.Priority,
			"enabled":     task.Enabled,
			"config":      task.Config,
			"metadata":    task.Metadata,
			"last_run":    task.LastRun,
			"next_run":    task.NextRun,
			"updated_at":  task.UpdatedAt,
			"updated_by":  task.UpdatedBy,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": task.ID}, update)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask deletes a task by ID
func (r *Repository) DeleteTask(ctx context.Context, taskID string) error {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	_, err := collection.DeleteOne(ctx, bson.M{"_id": taskID})
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// ListTasks retrieves tasks with pagination and filtering
func (r *Repository) ListTasks(ctx context.Context, filter bson.M, page, pageSize int) ([]Task, int64, error) {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	// Get total count
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Calculate skip
	skip := (page - 1) * pageSize

	// Find with pagination
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{"updated_at", -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, fmt.Errorf("failed to decode tasks: %w", err)
	}

	return tasks, total, nil
}

// GetEnabledTasks retrieves all enabled tasks
func (r *Repository) GetEnabledTasks(ctx context.Context) ([]*Task, error) {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	filter := bson.M{
		"enabled": true,
		"status": bson.M{"$nin": []TaskStatus{TaskStatusPaused, TaskStatusDisabled}},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find enabled tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []*Task
	for cursor.Next(ctx) {
		var task Task
		if err := cursor.Decode(&task); err != nil {
			slog.Error("Failed to decode task", slog.String("error", err.Error()))
			continue
		}
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// UpdateTaskStatus updates the status of a task
func (r *Repository) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus) error {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// UpdateTaskRun updates the last and next run times for a task
func (r *Repository) UpdateTaskRun(ctx context.Context, taskID string, lastRun, nextRun *time.Time) error {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	update := bson.M{
		"$set": bson.M{
			"last_run":   lastRun,
			"next_run":   nextRun,
			"updated_at": time.Now(),
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	if err != nil {
		return fmt.Errorf("failed to update task run times: %w", err)
	}

	return nil
}

// CreateExecution creates a new task execution record
func (r *Repository) CreateExecution(ctx context.Context, execution *TaskExecution) error {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	_, err := collection.InsertOne(ctx, execution)
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// UpdateExecution updates an existing execution record
func (r *Repository) UpdateExecution(ctx context.Context, execution *TaskExecution) error {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	update := bson.M{
		"$set": bson.M{
			"status":       execution.Status,
			"completed_at": execution.CompletedAt,
			"duration":     execution.Duration,
			"output":       execution.Output,
			"error":        execution.Error,
			"metadata":     execution.Metadata,
			"retry_count":  execution.RetryCount,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": execution.ID}, update)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	return nil
}

// GetExecution retrieves an execution by ID
func (r *Repository) GetExecution(ctx context.Context, executionID string) (*TaskExecution, error) {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	var execution TaskExecution
	err := collection.FindOne(ctx, bson.M{"_id": executionID}).Decode(&execution)
	if err != nil {
		return nil, err
	}

	return &execution, nil
}

// GetTaskExecutions retrieves executions for a specific task with pagination
func (r *Repository) GetTaskExecutions(ctx context.Context, taskID string, page, pageSize int) ([]*TaskExecution, error) {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	skip := (page - 1) * pageSize
	
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{"started_at", -1}})

	cursor, err := collection.Find(ctx, bson.M{"task_id": taskID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions: %w", err)
	}
	defer cursor.Close(ctx)

	var executions []*TaskExecution
	for cursor.Next(ctx) {
		var execution TaskExecution
		if err := cursor.Decode(&execution); err != nil {
			slog.Error("Failed to decode execution", slog.String("error", err.Error()))
			continue
		}
		executions = append(executions, &execution)
	}

	return executions, nil
}

// GetRunningExecutions retrieves all currently running executions
func (r *Repository) GetRunningExecutions(ctx context.Context) ([]*TaskExecution, error) {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	filter := bson.M{"status": TaskStatusRunning}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find running executions: %w", err)
	}
	defer cursor.Close(ctx)

	var executions []*TaskExecution
	for cursor.Next(ctx) {
		var execution TaskExecution
		if err := cursor.Decode(&execution); err != nil {
			slog.Error("Failed to decode running execution", slog.String("error", err.Error()))
			continue
		}
		executions = append(executions, &execution)
	}

	return executions, nil
}

// CleanupExecutions removes old execution records
func (r *Repository) CleanupExecutions(ctx context.Context, maxAge time.Duration) error {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	cutoff := time.Now().Add(-maxAge)
	filter := bson.M{
		"started_at": bson.M{"$lt": cutoff},
		"status": bson.M{"$in": []TaskStatus{TaskStatusCompleted, TaskStatusFailed}},
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup executions: %w", err)
	}

	if result.DeletedCount > 0 {
		slog.Info("Cleaned up old task executions", 
			slog.Int64("deleted_count", result.DeletedCount),
			slog.Duration("max_age", maxAge))
	}

	return nil
}

// UpdateTaskStatistics updates task execution statistics
func (r *Repository) UpdateTaskStatistics(ctx context.Context) error {
	collection := r.mongodb.Database.Collection(TasksCollection)
	
	// Aggregate execution statistics for each task
	pipeline := []bson.M{
		{
			"$lookup": bson.M{
				"from":         ExecutionsCollection,
				"localField":   "_id",
				"foreignField": "task_id",
				"as":           "executions",
			},
		},
		{
			"$addFields": bson.M{
				"metadata.total_runs": bson.M{"$size": "$executions"},
				"metadata.success_count": bson.M{
					"$size": bson.M{
						"$filter": bson.M{
							"input": "$executions",
							"cond":  bson.M{"$eq": []interface{}{"$$this.status", TaskStatusCompleted}},
						},
					},
				},
				"metadata.failure_count": bson.M{
					"$size": bson.M{
						"$filter": bson.M{
							"input": "$executions",
							"cond":  bson.M{"$eq": []interface{}{"$$this.status", TaskStatusFailed}},
						},
					},
				},
				"metadata.average_runtime": bson.M{
					"$avg": "$executions.duration",
				},
			},
		},
		{
			"$project": bson.M{
				"metadata.total_runs":      1,
				"metadata.success_count":   1,
				"metadata.failure_count":   1,
				"metadata.average_runtime": 1,
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("failed to aggregate task statistics: %w", err)
	}
	defer cursor.Close(ctx)

	// Update each task with its statistics
	for cursor.Next(ctx) {
		var result struct {
			ID       string `bson:"_id"`
			Metadata struct {
				TotalRuns      int64   `bson:"total_runs"`
				SuccessCount   int64   `bson:"success_count"`
				FailureCount   int64   `bson:"failure_count"`
				AverageRuntime float64 `bson:"average_runtime"` // Use float64 to handle existing data
			} `bson:"metadata"`
		}

		if err := cursor.Decode(&result); err != nil {
			slog.Error("Failed to decode task statistics", slog.String("error", err.Error()))
			continue
		}

		update := bson.M{
			"$set": bson.M{
				"metadata.total_runs":      result.Metadata.TotalRuns,
				"metadata.success_count":   result.Metadata.SuccessCount,
				"metadata.failure_count":   result.Metadata.FailureCount,
				"metadata.average_runtime": time.Duration(result.Metadata.AverageRuntime), // Convert float64 to Duration
				"updated_at":               time.Now(),
			},
		}

		_, err := collection.UpdateOne(ctx, bson.M{"_id": result.ID}, update)
		if err != nil {
			slog.Error("Failed to update task statistics",
				slog.String("task_id", result.ID),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

// HandleStaleRunningTasks finds and handles tasks that have been running too long
func (r *Repository) HandleStaleRunningTasks(ctx context.Context) error {
	collection := r.mongodb.Database.Collection(ExecutionsCollection)
	
	// Find executions that have been running for more than 2 hours (configurable)
	cutoff := time.Now().Add(-2 * time.Hour)
	filter := bson.M{
		"status":     TaskStatusRunning,
		"started_at": bson.M{"$lt": cutoff},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find stale running tasks: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var execution TaskExecution
		if err := cursor.Decode(&execution); err != nil {
			continue
		}

		// Mark as failed due to timeout
		now := time.Now()
		execution.Status = TaskStatusFailed
		execution.CompletedAt = &now
		execution.Duration = now.Sub(execution.StartedAt)
		execution.Error = "Task execution timed out (stale running task cleanup)"

		if err := r.UpdateExecution(ctx, &execution); err != nil {
			slog.Error("Failed to update stale execution",
				slog.String("execution_id", execution.ID),
				slog.String("error", err.Error()))
		} else {
			slog.Warn("Marked stale running task as failed",
				slog.String("execution_id", execution.ID),
				slog.String("task_id", execution.TaskID),
				slog.Duration("running_time", execution.Duration))
		}
	}

	return nil
}

// GetSchedulerStats retrieves scheduler statistics
func (r *Repository) GetSchedulerStats(ctx context.Context) (*SchedulerStats, error) {
	tasksCollection := r.mongodb.Database.Collection(TasksCollection)
	executionsCollection := r.mongodb.Database.Collection(ExecutionsCollection)

	var stats SchedulerStats

	// Count total tasks
	totalTasks, err := tasksCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total tasks: %w", err)
	}
	stats.TotalTasks = totalTasks

	// Count enabled tasks
	enabledTasks, err := tasksCollection.CountDocuments(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, fmt.Errorf("failed to count enabled tasks: %w", err)
	}
	stats.EnabledTasks = enabledTasks

	// Count running tasks
	runningTasks, err := executionsCollection.CountDocuments(ctx, bson.M{"status": TaskStatusRunning})
	if err != nil {
		return nil, fmt.Errorf("failed to count running tasks: %w", err)
	}
	stats.RunningTasks = runningTasks

	// Count completed today
	today := time.Now().Truncate(24 * time.Hour)
	completedToday, err := executionsCollection.CountDocuments(ctx, bson.M{
		"status":     TaskStatusCompleted,
		"started_at": bson.M{"$gte": today},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count completed tasks today: %w", err)
	}
	stats.CompletedToday = completedToday

	// Count failed today
	failedToday, err := executionsCollection.CountDocuments(ctx, bson.M{
		"status":     TaskStatusFailed,
		"started_at": bson.M{"$gte": today},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count failed tasks today: %w", err)
	}
	stats.FailedToday = failedToday

	// Get average runtime
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"status":     TaskStatusCompleted,
				"started_at": bson.M{"$gte": today},
			},
		},
		{
			"$group": bson.M{
				"_id":     nil,
				"avg_duration": bson.M{"$avg": "$duration"},
			},
		},
	}

	cursor, err := executionsCollection.Aggregate(ctx, pipeline)
	if err == nil {
		defer cursor.Close(ctx)
		if cursor.Next(ctx) {
			var result struct {
				AvgDuration time.Duration `bson:"avg_duration"`
			}
			if err := cursor.Decode(&result); err == nil {
				stats.AverageRuntime = result.AvgDuration.String()
			}
		}
	}

	// Get next scheduled run
	pipeline = []bson.M{
		{
			"$match": bson.M{
				"enabled":  true,
				"next_run": bson.M{"$ne": nil},
			},
		},
		{
			"$sort": bson.M{"next_run": 1},
		},
		{
			"$limit": 1,
		},
	}

	cursor, err = tasksCollection.Aggregate(ctx, pipeline)
	if err == nil {
		defer cursor.Close(ctx)
		if cursor.Next(ctx) {
			var result struct {
				NextRun *time.Time `bson:"next_run"`
			}
			if err := cursor.Decode(&result); err == nil && result.NextRun != nil {
				stats.NextScheduledRun = result.NextRun
			}
		}
	}

	return &stats, nil
}

// CreateIndexes creates necessary database indexes
func (r *Repository) CreateIndexes(ctx context.Context) error {
	tasksCollection := r.mongodb.Database.Collection(TasksCollection)
	executionsCollection := r.mongodb.Database.Collection(ExecutionsCollection)

	// Task indexes
	taskIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"enabled", 1}, {"status", 1}},
		},
		{
			Keys: bson.D{{"next_run", 1}},
		},
		{
			Keys: bson.D{{"metadata.is_system", 1}},
		},
		{
			Keys: bson.D{{"metadata.tags", 1}},
		},
		{
			Keys: bson.D{{"updated_at", -1}},
		},
	}

	if _, err := tasksCollection.Indexes().CreateMany(ctx, taskIndexes); err != nil {
		return fmt.Errorf("failed to create task indexes: %w", err)
	}

	// Execution indexes
	executionIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{"task_id", 1}, {"started_at", -1}},
		},
		{
			Keys: bson.D{{"status", 1}},
		},
		{
			Keys: bson.D{{"started_at", 1}},
		},
		{
			Keys: bson.D{{"started_at", 1}, {"status", 1}},
		},
	}

	if _, err := executionsCollection.Indexes().CreateMany(ctx, executionIndexes); err != nil {
		return fmt.Errorf("failed to create execution indexes: %w", err)
	}

	slog.Info("Created scheduler database indexes")
	return nil
}