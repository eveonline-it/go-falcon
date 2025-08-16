package scheduler

import (
	"time"
)

// getSystemTasks returns the hardcoded system tasks that are automatically
// created and managed by the scheduler. These tasks handle critical system
// operations and maintenance functions.
func getSystemTasks() []*Task {
	now := time.Now()

	return []*Task{
		// EVE Token Refresh - High priority task to keep authentication tokens fresh
		{
			ID:          "system-token-refresh",
			Name:        "EVE Token Refresh",
			Description: "Refresh expired EVE Online access tokens",
			Type:        TaskTypeSystem,
			Schedule:    "0 */15 * * * *", // Every 15 minutes
			Status:      TaskStatusPending,
			Priority:    TaskPriorityHigh,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "token_refresh",
				"parameters": map[string]interface{}{
					"batch_size": 100,
					"timeout":    "5m",
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    3,
				RetryInterval: 2 * time.Minute,
				Timeout:       10 * time.Minute,
				Tags:          []string{"system", "auth", "eve"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// State Cleanup - Regular cleanup of expired states and temporary data
		{
			ID:          "system-state-cleanup",
			Name:        "State Cleanup",
			Description: "Clean up expired states and temporary data",
			Type:        TaskTypeSystem,
			Schedule:    "0 0 */2 * * *", // Every 2 hours
			Status:      TaskStatusPending,
			Priority:    TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "state_cleanup",
				"parameters": map[string]interface{}{
					"max_age": "24h",
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 5 * time.Minute,
				Timeout:       30 * time.Minute,
				Tags:          []string{"system", "cleanup"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// Health Check - Regular monitoring of system components
		{
			ID:          "system-health-check",
			Name:        "Health Check",
			Description: "Perform system health checks and monitoring",
			Type:        TaskTypeSystem,
			Schedule:    "0 */5 * * * *", // Every 5 minutes
			Status:      TaskStatusPending,
			Priority:    TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "health_check",
				"parameters": map[string]interface{}{
					"check_services": []string{"mongodb", "redis", "esi"},
					"timeout":        "30s",
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 1 * time.Minute,
				Timeout:       2 * time.Minute,
				Tags:          []string{"system", "monitoring", "health"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// Task History Cleanup - Daily maintenance to remove old execution records
		{
			ID:          "system-task-cleanup",
			Name:        "Task History Cleanup",
			Description: "Clean up old task execution records",
			Type:        TaskTypeSystem,
			Schedule:    "0 0 2 * * *", // Daily at 2 AM
			Status:      TaskStatusPending,
			Priority:    TaskPriorityLow,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "task_cleanup",
				"parameters": map[string]interface{}{
					"retention_days": 30,
					"batch_size":     1000,
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 10 * time.Minute,
				Timeout:       1 * time.Hour,
				Tags:          []string{"system", "cleanup", "maintenance"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// SDE Update Check - Regular check for new EVE Online SDE versions
		{
			ID:          "system-sde-check",
			Name:        "SDE Update Check",
			Description: "Check for new EVE Online SDE versions and notify if updates are available",
			Type:        TaskTypeSystem,
			Schedule:    "0 0 */6 * * *", // Every 6 hours
			Status:      TaskStatusPending,
			Priority:    TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "sde_check",
				"parameters": map[string]interface{}{
					"auto_update": false, // Only check, don't auto-update
					"notify":      true,
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    3,
				RetryInterval: 30 * time.Minute,
				Timeout:       15 * time.Minute,
				Tags:          []string{"system", "sde", "update", "eve"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// Corporate Membership Validation - Validate corporation and alliance memberships
		{
			ID:          "system-corporate-validation",
			Name:        "Corporate Membership Validation",
			Description: "Validate corporate and alliance group memberships against ESI data",
			Type:        TaskTypeSystem,
			Schedule:    "0 0 */1 * * *", // Every hour
			Status:      TaskStatusPending,
			Priority:    TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "corporate_validation",
				"parameters": map[string]interface{}{
					"validate_corporations": true,
					"validate_alliances":    true,
					"remove_invalid":        true,
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 15 * time.Minute,
				Timeout:       30 * time.Minute,
				Tags:          []string{"system", "groups", "validation", "eve"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// Group Membership Cleanup - Clean up expired group memberships
		{
			ID:          "system-group-cleanup",
			Name:        "Group Membership Cleanup",
			Description: "Clean up expired group memberships and invalidate permission cache",
			Type:        TaskTypeSystem,
			Schedule:    "0 0 3 * * *", // Daily at 3 AM
			Status:      TaskStatusPending,
			Priority:    TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "group_cleanup",
				"parameters": map[string]interface{}{
					"clear_cache": true,
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    2,
				RetryInterval: 10 * time.Minute,
				Timeout:       1 * time.Hour,
				Tags:          []string{"system", "groups", "cleanup"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// Discord Role Sync - Synchronize group memberships with Discord roles
		{
			ID:          "system-discord-sync",
			Name:        "Discord Role Sync",
			Description: "Synchronize group memberships with Discord roles across all servers",
			Type:        TaskTypeSystem,
			Schedule:    "0 */30 * * * *", // Every 30 minutes
			Status:      TaskStatusPending,
			Priority:    TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "discord_sync",
				"parameters": map[string]interface{}{
					"batch_process": true,
					"retry_failed":  true,
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    3,
				RetryInterval: 5 * time.Minute,
				Timeout:       45 * time.Minute,
				Tags:          []string{"system", "groups", "discord", "sync"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},

		// Group Integrity Check - Validate group data integrity
		{
			ID:          "system-group-integrity",
			Name:        "Group Integrity Check",
			Description: "Validate group data integrity and fix inconsistencies",
			Type:        TaskTypeSystem,
			Schedule:    "0 0 4 * * *", // Daily at 4 AM
			Status:      TaskStatusPending,
			Priority:    TaskPriorityLow,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "group_integrity",
				"parameters": map[string]interface{}{
					"check_orphaned":    true,
					"check_duplicates":  true,
					"check_permissions": true,
					"auto_fix":          true,
				},
			},
			Metadata: TaskMetadata{
				MaxRetries:    1,
				RetryInterval: 30 * time.Minute,
				Timeout:       2 * time.Hour,
				Tags:          []string{"system", "groups", "integrity", "maintenance"},
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

// SystemTaskDefinitions provides metadata about each system task for documentation
// and administrative purposes
var SystemTaskDefinitions = map[string]struct {
	Name        string
	Description string
	Schedule    string
	Purpose     string
	Priority    string
}{
	"system-token-refresh": {
		Name:        "EVE Token Refresh",
		Description: "Refresh expired EVE Online access tokens",
		Schedule:    "Every 15 minutes",
		Purpose:     "Maintains authentication for EVE Online API access",
		Priority:    "High",
	},
	"system-state-cleanup": {
		Name:        "State Cleanup",
		Description: "Clean up expired states and temporary data",
		Schedule:    "Every 2 hours",
		Purpose:     "Prevents memory/storage bloat from temporary data",
		Priority:    "Normal",
	},
	"system-health-check": {
		Name:        "Health Check",
		Description: "Perform system health checks and monitoring",
		Schedule:    "Every 5 minutes",
		Purpose:     "Early detection of system issues and service failures",
		Priority:    "Normal",
	},
	"system-task-cleanup": {
		Name:        "Task History Cleanup",
		Description: "Clean up old task execution records",
		Schedule:    "Daily at 2 AM",
		Purpose:     "Maintains database performance by removing old execution logs",
		Priority:    "Low",
	},
	"system-sde-check": {
		Name:        "SDE Update Check",
		Description: "Check for new EVE Online SDE versions and notify if updates are available",
		Schedule:    "Every 6 hours",
		Purpose:     "Keeps EVE Online static data current for game features",
		Priority:    "Normal",
	},
	"system-corporate-validation": {
		Name:        "Corporate Membership Validation",
		Description: "Validate corporate and alliance group memberships against ESI data",
		Schedule:    "Every hour",
		Purpose:     "Ensures group memberships stay current with EVE Online corporation/alliance changes",
		Priority:    "Normal",
	},
	"system-group-cleanup": {
		Name:        "Group Membership Cleanup",
		Description: "Clean up expired group memberships and invalidate permission cache",
		Schedule:    "Daily at 3 AM",
		Purpose:     "Removes expired memberships and maintains permission system performance",
		Priority:    "Normal",
	},
	"system-discord-sync": {
		Name:        "Discord Role Sync",
		Description: "Synchronize group memberships with Discord roles across all servers",
		Schedule:    "Every 30 minutes",
		Purpose:     "Keeps Discord roles synchronized with application group memberships",
		Priority:    "Normal",
	},
	"system-group-integrity": {
		Name:        "Group Integrity Check",
		Description: "Validate group data integrity and fix inconsistencies",
		Schedule:    "Daily at 4 AM",
		Purpose:     "Maintains data consistency and fixes orphaned or duplicate memberships",
		Priority:    "Low",
	},
}