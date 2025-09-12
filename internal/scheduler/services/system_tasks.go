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
				RetryInterval: models.Duration(2 * time.Minute),
				Timeout:       models.Duration(10 * time.Minute),
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
			ID:          "system-groups-sync",
			Name:        "Groups Synchronization",
			Description: "Synchronizes character group memberships and validates corp/alliance memberships via ESI",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 */6 * * *", // Every 6 hours
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "groups_sync",
				"parameters": map[string]interface{}{
					"batch_size":           50,
					"timeout":              "10m",
					"validate_memberships": true,
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(5 * time.Minute),
				Timeout:       models.Duration(15 * time.Minute),
				Tags:          []string{"system", "groups", "esi"},
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
				RetryInterval: models.Duration(5 * time.Minute),
				Timeout:       models.Duration(15 * time.Minute),
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
				RetryInterval: models.Duration(1 * time.Minute),
				Timeout:       models.Duration(1 * time.Minute),
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
				RetryInterval: models.Duration(30 * time.Minute),
				Timeout:       models.Duration(30 * time.Minute),
				Tags:          []string{"system", "cleanup", "maintenance"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-character-affiliation-update",
			Name:        "Character Affiliation Update",
			Description: "Updates character corporation and alliance affiliations from EVE ESI",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */30 * * * *", // Every 30 minutes
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     false, // DISABLED: Task was processing 1M+ characters and saturating MongoDB
			Config: map[string]interface{}{
				"task_name": "character_affiliation_update",
				"parameters": map[string]interface{}{
					"batch_size":       1000, // ESI max batch size
					"parallel_workers": 3,    // Concurrent ESI requests
					"timeout":          "5m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    3,
				RetryInterval: models.Duration(5 * time.Minute),
				Timeout:       models.Duration(10 * time.Minute),
				Tags:          []string{"system", "character", "esi", "affiliation"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-group-membership-validation",
			Name:        "Group Membership Validation",
			Description: "Validates corporate memberships and group integrity",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 */6 * * *", // Every 6 hours
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "group_validation",
				"parameters": map[string]interface{}{
					"type": "membership",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(10 * time.Minute),
				Timeout:       models.Duration(20 * time.Minute),
				Tags:          []string{"system", "groups", "validation"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-alliance-bulk-import",
			Name:        "Alliance Bulk Import",
			Description: "Retrieves all alliance IDs from ESI and imports detailed information for each alliance into the database",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 3 */7 * *", // Weekly at 3 AM on Sunday
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "alliance_bulk_import",
				"parameters": map[string]interface{}{
					"batch_size":             10,
					"delay_between_requests": "200ms",
					"timeout":                "30m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(15 * time.Minute),
				Timeout:       models.Duration(45 * time.Minute),
				Tags:          []string{"system", "alliance", "esi", "import"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-corporation-update",
			Name:        "Corporation Data Update",
			Description: "Updates all corporation information from EVE ESI for corporations in the database",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 4 * * *", // Daily at 4 AM
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "corporation_update",
				"parameters": map[string]interface{}{
					"concurrent_workers": 10,
					"timeout":            "60m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(15 * time.Minute),
				Timeout:       models.Duration(60 * time.Minute),
				Tags:          []string{"system", "corporation", "esi", "update"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-ceo-token-validation",
			Name:        "CEO Token Validation",
			Description: "Checks if CEOs from enabled corporations have valid EVE Online access tokens",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */15 * * * *", // Every 15 minutes
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "ceo_token_validation",
				"parameters": map[string]interface{}{
					"timeout": "5m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(5 * time.Minute),
				Timeout:       models.Duration(10 * time.Minute),
				Tags:          []string{"system", "corporation", "ceo", "token", "validation"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-discord-token-refresh",
			Name:        "Discord Token Refresh",
			Description: "Refreshes expired Discord access tokens for users with linked Discord accounts",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */30 * * * *", // Every 30 minutes
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "discord_token_refresh",
				"parameters": map[string]interface{}{
					"batch_size": 50,
					"timeout":    "10m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    3,
				RetryInterval: models.Duration(5 * time.Minute),
				Timeout:       models.Duration(15 * time.Minute),
				Tags:          []string{"system", "discord", "tokens", "refresh"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-discord-role-sync",
			Name:        "Discord Role Synchronization",
			Description: "Synchronizes Discord roles with Go Falcon group memberships for all configured guilds",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */15 * * * *", // Every 15 minutes
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "discord_role_sync",
				"parameters": map[string]interface{}{
					"timeout": "30m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(10 * time.Minute),
				Timeout:       models.Duration(45 * time.Minute),
				Tags:          []string{"system", "discord", "roles", "synchronization"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-market-data-fetch",
			Name:        "Regional Market Data Fetch",
			Description: "Fetches market orders from all EVE Online regions with adaptive pagination support",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 0 * * * *", // Every hour
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityNormal,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "market_data_fetch",
				"parameters": map[string]interface{}{
					"concurrent_workers":       8,
					"regions_per_batch":        20,
					"timeout":                  "45m",
					"pagination_mode":          "auto", // "auto", "offset", "token"
					"enable_incremental":       true,   // Use after tokens when available
					"max_duplicates_threshold": 1000,   // Alert threshold for token pagination
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    2,
				RetryInterval: models.Duration(15 * time.Minute),
				Timeout:       models.Duration(60 * time.Minute),
				Tags:          []string{"system", "market", "esi", "data_fetch"},
				IsSystem:      true,
				Source:        "system",
				Version:       1,
			},
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: "system",
		},
		{
			ID:          "system-market-pagination-monitor",
			Name:        "Market Pagination Migration Monitor",
			Description: "Monitors ESI market endpoints for token-based pagination availability and migration status",
			Type:        models.TaskTypeSystem,
			Schedule:    "0 */6 * * * *", // Every 6 hours
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityLow,
			Enabled:     true,
			Config: map[string]interface{}{
				"task_name": "pagination_migration_monitor",
				"parameters": map[string]interface{}{
					"test_regions": []int{10000002, 10000030}, // Jita, Heimatar
					"timeout":      "5m",
				},
			},
			Metadata: models.TaskMetadata{
				MaxRetries:    1,
				RetryInterval: models.Duration(30 * time.Minute),
				Timeout:       models.Duration(10 * time.Minute),
				Tags:          []string{"system", "market", "pagination", "monitoring"},
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
	"system-alliance-bulk-import": {
		Name:        "Alliance Bulk Import",
		Description: "Retrieves all alliance IDs from ESI and imports detailed information for each alliance",
		Schedule:    "Weekly on Sunday at 3:00 AM",
		Purpose:     "Maintains up-to-date alliance database with comprehensive EVE Online alliance information",
		Priority:    "Normal",
	},
	"system-corporation-update": {
		Name:        "Corporation Data Update",
		Description: "Updates all corporation information from EVE ESI for corporations in the database",
		Schedule:    "Daily at 4:00 AM",
		Purpose:     "Maintains up-to-date corporation database with fresh EVE Online corporation information",
		Priority:    "Normal",
	},
	"system-ceo-token-validation": {
		Name:        "CEO Token Validation",
		Description: "Checks if CEOs from enabled corporations have valid EVE Online access tokens",
		Schedule:    "Every 15 minutes",
		Purpose:     "Monitors CEO token validity for enabled corporations and identifies expired tokens",
		Priority:    "Normal",
	},
	"system-discord-token-refresh": {
		Name:        "Discord Token Refresh",
		Description: "Refreshes expired Discord access tokens for users with linked Discord accounts",
		Schedule:    "Every 30 minutes",
		Purpose:     "Maintains Discord authentication by refreshing tokens before they expire",
		Priority:    "Normal",
	},
	"system-discord-role-sync": {
		Name:        "Discord Role Synchronization",
		Description: "Synchronizes Discord roles with Go Falcon group memberships for all configured guilds",
		Schedule:    "Every 15 minutes",
		Purpose:     "Maintains Discord role consistency by syncing group memberships to Discord roles",
		Priority:    "Normal",
	},
	"system-market-data-fetch": {
		Name:        "Regional Market Data Fetch",
		Description: "Fetches market orders from all EVE Online regions with adaptive pagination support and atomic collection swapping",
		Schedule:    "Every hour",
		Purpose:     "Maintains up-to-date market data by fetching orders from all regions with parallel processing and ESI rate limiting compliance",
		Priority:    "Normal",
	},
	"system-market-pagination-monitor": {
		Name:        "Market Pagination Migration Monitor",
		Description: "Monitors ESI market endpoints for token-based pagination availability and migration status",
		Schedule:    "Every 6 hours",
		Purpose:     "Detects when EVE ESI transitions to token-based pagination and adjusts market fetching strategy accordingly",
		Priority:    "Low",
	},
}
