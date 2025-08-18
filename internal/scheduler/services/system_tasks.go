package services

import (
	"time"

	"go-falcon/internal/scheduler/models"
)

// GetSystemTasks returns predefined system tasks
func GetSystemTasks() []*models.Task {
	now := time.Now()

	return []*models.Task{
		{
			ID:          "system-token-refresh",
			Name:        "EVE Token Refresh",
			Description: "Refreshes expired EVE Online access tokens for users",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */15 * * * *", // Every 15 minutes
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityHigh,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "token_refresh",
				"parameters": map[string]interface{}{
					"batch_size": 100,
					"timeout":    "5m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    3,
				RetryInterval: 2 * time.Minute,
				Timeout:       10 * time.Minute,
				Tags:          []string{"system", "auth", "tokens"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-state-cleanup",
			Name:        "State Cleanup",
			Description: "Cleans up expired states and temporary data",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 */2 * * *", // Every 2 hours
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "state_cleanup",
				"parameters": map[string]interface{}{
					"retention_hours": 24,
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 5 * time.Minute,
				Timeout:       15 * time.Minute,
				Tags:          []string{"system", "cleanup"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-health-check",
			Name:        "Health Check",
			Description: "Monitors system health (MongoDB, Redis, ESI)",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */5 * * * *", // Every 5 minutes
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "health_check",
				"parameters": map[string]interface{}{
					"timeout": "30s",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    1,
				RetryInterval: 1 * time.Minute,
				Timeout:       1 * time.Minute,
				Tags:          []string{"system", "monitoring", "health"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-task-cleanup",
			Name:        "Task History Cleanup",
			Description: "Removes old task execution records",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 2 * * *", // Daily at 2 AM
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityLow,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "task_cleanup",
				"parameters": map[string]interface{}{
					"retention_days": 30,
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 30 * time.Minute,
				Timeout:       30 * time.Minute,
				Tags:          []string{"system", "cleanup", "maintenance"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
	}
}

// SystemTaskDefinitions provides metadata about system tasks
var SystemTaskDefinitions = map[string]models.SystemTaskDefinition{
	"system-token-refresh": {
		Name:        "EVE Token Refresh",
		Description: "Automatically refreshes expired EVE Online access tokens for authenticated users",
		Schedule:    "Every 15 minutes",
		Purpose:     "Maintains user authentication by refreshing tokens before they expire",
		Priority:    "High",
	},
	"system-state-cleanup": {
		Name:        "State Cleanup",
		Description: "Removes expired OAuth states and temporary authentication data",
		Schedule:    "Every 2 hours",
		Purpose:     "Prevents memory leaks and maintains security by cleaning expired states",
		Priority:    "Normal",
	},
	"system-health-check": {
		Name:        "Health Check",
		Description: "Monitors the health of critical system components",
		Schedule:    "Every 5 minutes",
		Purpose:     "Early detection of system issues and service degradation",
		Priority:    "Normal",
	},
	"system-task-cleanup": {
		Name:        "Task History Cleanup",
		Description: "Removes old task execution records to manage database size",
		Schedule:    "Daily at 2:00 AM",
		Purpose:     "Maintains database performance by removing old execution history",
		Priority:    "Low",
	},
}