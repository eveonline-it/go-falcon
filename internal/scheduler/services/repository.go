package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/scheduler/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for scheduler
type Repository struct {
	mongodb *database.MongoDB
	tasks   *mongo.Collection
	executions *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:    mongodb,
		tasks:      mongodb.Database.Collection("scheduler_tasks"),
		executions: mongodb.Database.Collection("scheduler_executions"),
	}
}

// Task Operations

// CreateTask creates a new task
func (r *Repository) CreateTask(ctx context.Context, task *models.Task) error {
	_, err := r.tasks.InsertOne(ctx, task)
	return err
}

// GetTask retrieves a task by ID
func (r *Repository) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	var task models.Task
	err := r.tasks.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTask updates an existing task
func (r *Repository) UpdateTask(ctx context.Context, task *models.Task) error {
	_, err := r.tasks.ReplaceOne(ctx, bson.M{"_id": task.ID}, task)
	return err
}

// DeleteTask deletes a task
func (r *Repository) DeleteTask(ctx context.Context, taskID string) error {
	_, err := r.tasks.DeleteOne(ctx, bson.M{"_id": taskID})
	return err
}

// ListTasks lists tasks with filtering and pagination
func (r *Repository) ListTasks(ctx context.Context, filter bson.M, page, pageSize int) ([]models.Task, int64, error) {
	// Count total documents
	total, err := r.tasks.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := r.tasks.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetActiveTasks retrieves all enabled tasks
func (r *Repository) GetActiveTasks(ctx context.Context) ([]models.Task, error) {
	filter := bson.M{
		"enabled": true,
		"status": bson.M{"$nin": []string{"paused", "disabled"}},
	}

	cursor, err := r.tasks.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// UpdateTaskStatus updates a task's status
func (r *Repository) UpdateTaskStatus(ctx context.Context, taskID string, status models.TaskStatus) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	_, err := r.tasks.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	return err
}

// UpdateTaskRun updates task run information
func (r *Repository) UpdateTaskRun(ctx context.Context, taskID string, lastRun, nextRun *time.Time) error {
	update := bson.M{
		"$set": bson.M{
			"last_run":   lastRun,
			"next_run":   nextRun,
			"updated_at": time.Now(),
		},
	}

	_, err := r.tasks.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	return err
}

// UpdateTaskRunWithDuration updates task run information including execution duration and calculates average runtime
func (r *Repository) UpdateTaskRunWithDuration(ctx context.Context, taskID string, lastRun, nextRun *time.Time, duration *time.Duration, success bool) error {
	updateFields := bson.M{
		"last_run":   lastRun,
		"next_run":   nextRun,
		"updated_at": time.Now(),
	}
	
	if duration != nil {
		updateFields["last_run_duration"] = *duration
	}
	
	update := bson.M{"$set": updateFields}
	
	// Update statistics
	incFields := bson.M{
		"metadata.total_runs": 1,
	}
	
	if success {
		incFields["metadata.success_count"] = 1
	} else {
		incFields["metadata.failure_count"] = 1
	}
	
	update["$inc"] = incFields
	
	// Update average runtime if we have a duration
	if duration != nil && success {
		// Get current task to calculate new average
		task, err := r.GetTask(ctx, taskID)
		if err == nil {
			newAverage := r.calculateNewAverage(time.Duration(task.Metadata.AverageRuntime), task.Metadata.SuccessCount, *duration)
			updateFields["metadata.average_runtime"] = newAverage
		}
	}
	
	_, err := r.tasks.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	return err
}

// calculateNewAverage calculates a new running average runtime
func (r *Repository) calculateNewAverage(currentAverage time.Duration, currentCount int64, newDuration time.Duration) models.Duration {
	if currentCount == 0 {
		return models.Duration(newDuration)
	}
	
	// Calculate new average: ((current_avg * current_count) + new_duration) / (current_count + 1)
	totalTime := time.Duration(int64(currentAverage) * currentCount) + newDuration
	newCount := currentCount + 1
	return models.Duration(int64(totalTime) / newCount)
}

// UpdateTaskStatistics updates task statistics
func (r *Repository) UpdateTaskStatistics(ctx context.Context) error {
	// This could be implemented to update aggregated statistics
	// For now, we'll update statistics per task when executions complete
	return nil
}

// HandleStaleRunningTasks finds and handles tasks that have been running too long
func (r *Repository) HandleStaleRunningTasks(ctx context.Context) error {
	// Find tasks that have been running for more than their timeout
	staleThreshold := time.Now().Add(-2 * time.Hour) // Default 2 hours

	filter := bson.M{
		"status":     models.TaskStatusRunning,
		"updated_at": bson.M{"$lt": staleThreshold},
	}

	update := bson.M{
		"$set": bson.M{
			"status":     models.TaskStatusFailed,
			"updated_at": time.Now(),
			"metadata.last_error": "Task marked as stale after timeout",
		},
	}

	result, err := r.tasks.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.ModifiedCount > 0 {
		// Log the number of stale tasks handled
		fmt.Printf("Handled %d stale running tasks\n", result.ModifiedCount)
	}

	return nil
}

// Execution Operations

// CreateExecution creates a new task execution record
func (r *Repository) CreateExecution(ctx context.Context, execution *models.TaskExecution) error {
	_, err := r.executions.InsertOne(ctx, execution)
	return err
}

// GetExecution retrieves an execution by ID
func (r *Repository) GetExecution(ctx context.Context, executionID string) (*models.TaskExecution, error) {
	var execution models.TaskExecution
	err := r.executions.FindOne(ctx, bson.M{"_id": executionID}).Decode(&execution)
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// UpdateExecution updates an execution record
func (r *Repository) UpdateExecution(ctx context.Context, execution *models.TaskExecution) error {
	_, err := r.executions.ReplaceOne(ctx, bson.M{"_id": execution.ID}, execution)
	return err
}

// GetTaskExecutions retrieves executions for a specific task with pagination
func (r *Repository) GetTaskExecutions(ctx context.Context, taskID string, page, pageSize int) ([]models.TaskExecution, error) {
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "started_at", Value: -1}})

	cursor, err := r.executions.Find(ctx, bson.M{"task_id": taskID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var executions []models.TaskExecution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, err
	}

	return executions, nil
}

// ListExecutions retrieves all executions with filtering and pagination
func (r *Repository) ListExecutions(ctx context.Context, filter bson.M, page, pageSize int) ([]models.TaskExecution, int64, error) {
	// Count total documents
	total, err := r.executions.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "started_at", Value: -1}})

	cursor, err := r.executions.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var executions []models.TaskExecution
	if err := cursor.All(ctx, &executions); err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// CleanupExecutions removes old execution records
func (r *Repository) CleanupExecutions(ctx context.Context, retentionPeriod time.Duration) error {
	cutoff := time.Now().Add(-retentionPeriod)
	
	filter := bson.M{
		"started_at": bson.M{"$lt": cutoff},
	}

	result, err := r.executions.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount > 0 {
		fmt.Printf("Cleaned up %d old execution records\n", result.DeletedCount)
	}

	return nil
}

// Statistics Operations

// GetSchedulerStats retrieves scheduler statistics
func (r *Repository) GetSchedulerStats(ctx context.Context) (*models.SchedulerStats, error) {
	stats := &models.SchedulerStats{}

	// Count total tasks
	total, err := r.tasks.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats.TotalTasks = total

	// Count enabled tasks
	enabled, err := r.tasks.CountDocuments(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, err
	}
	stats.EnabledTasks = enabled

	// Count running tasks
	running, err := r.tasks.CountDocuments(ctx, bson.M{"status": models.TaskStatusRunning})
	if err != nil {
		return nil, err
	}
	stats.RunningTasks = running

	// Count completions today
	today := time.Now().Truncate(24 * time.Hour)
	completedToday, err := r.executions.CountDocuments(ctx, bson.M{
		"status":     models.TaskStatusCompleted,
		"started_at": bson.M{"$gte": today},
	})
	if err != nil {
		return nil, err
	}
	stats.CompletedToday = completedToday

	// Count failures today
	failedToday, err := r.executions.CountDocuments(ctx, bson.M{
		"status":     models.TaskStatusFailed,
		"started_at": bson.M{"$gte": today},
	})
	if err != nil {
		return nil, err
	}
	stats.FailedToday = failedToday

	// Get average runtime (simplified calculation)
	// In a real implementation, you might use aggregation pipeline for better performance
	pipeline := []bson.M{
		{"$match": bson.M{"status": models.TaskStatusCompleted}},
		{"$group": bson.M{
			"_id": nil,
			"avg_duration": bson.M{"$avg": "$duration"},
		}},
	}

	cursor, err := r.executions.Aggregate(ctx, pipeline)
	if err == nil {
		defer cursor.Close(ctx)
		var result []bson.M
		if err := cursor.All(ctx, &result); err == nil && len(result) > 0 {
			// MongoDB stores time.Duration as int64 nanoseconds
			// The aggregation can return float64 when averaging
			switch v := result[0]["avg_duration"].(type) {
			case float64:
				stats.AverageRuntime = time.Duration(int64(v)).String()
			case int64:
				stats.AverageRuntime = time.Duration(v).String()
			case int32:
				stats.AverageRuntime = time.Duration(int64(v)).String()
			}
		}
	}

	if stats.AverageRuntime == "" {
		stats.AverageRuntime = "0s"
	}

	// Find next scheduled run
	opts := options.FindOne().SetSort(bson.D{{Key: "next_run", Value: 1}})
	var nextTask models.Task
	err = r.tasks.FindOne(ctx, bson.M{
		"enabled": true,
		"next_run": bson.M{"$ne": nil},
		"status": bson.M{"$nin": []string{"paused", "disabled"}},
	}, opts).Decode(&nextTask)
	
	if err == nil && nextTask.NextRun != nil {
		stats.NextScheduledRun = nextTask.NextRun
	}

	return stats, nil
}

// Bulk Operations

// BulkUpdateTaskStatus updates multiple tasks' status
func (r *Repository) BulkUpdateTaskStatus(ctx context.Context, taskIDs []string, status models.TaskStatus) (int64, error) {
	filter := bson.M{"_id": bson.M{"$in": taskIDs}}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.tasks.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// BulkDeleteTasks deletes multiple tasks (excludes system tasks)
func (r *Repository) BulkDeleteTasks(ctx context.Context, taskIDs []string) (int64, error) {
	filter := bson.M{
		"_id": bson.M{"$in": taskIDs},
		"metadata.is_system": bson.M{"$ne": true}, // Protect system tasks
	}

	result, err := r.tasks.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// CheckHealth verifies database connectivity
func (r *Repository) CheckHealth(ctx context.Context) error {
	// Perform a simple ping to check database connectivity
	return r.mongodb.Client.Ping(ctx, nil)
}