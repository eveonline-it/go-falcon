package services

import (
	"context"
	"fmt"

	schedulerDTO "go-falcon/internal/scheduler/dto"
	schedulerModels "go-falcon/internal/scheduler/models"
	schedulerServices "go-falcon/internal/scheduler/services"
)

// RegisterScheduledTasks registers all asset-related scheduled tasks
func (s *AssetService) RegisterScheduledTasks(ctx context.Context, schedulerService *schedulerServices.SchedulerService) error {
	// Process asset tracking every 30 minutes
	_, err := schedulerService.CreateTask(ctx, &schedulerDTO.TaskCreateRequest{
		Name:        "Process Asset Tracking",
		Description: "Processes all active asset tracking configurations",
		Type:        schedulerModels.TaskTypeSystem,
		Schedule:    "0 */30 * * * *", // Every 30 minutes (6-field format with seconds)
		Priority:    schedulerModels.TaskPriorityNormal,
		Enabled:     true,
		Config: map[string]interface{}{
			"task_name": "asset_tracking_processor",
		},
		Tags: []string{"assets", "tracking", "system"},
	})
	if err != nil {
		return fmt.Errorf("failed to register asset tracking processor: %w", err)
	}

	// Create daily asset snapshots at 4 AM
	_, err = schedulerService.CreateTask(ctx, &schedulerDTO.TaskCreateRequest{
		Name:        "Create Asset Snapshots",
		Description: "Creates daily snapshots of all tracked assets",
		Type:        schedulerModels.TaskTypeSystem,
		Schedule:    "0 0 4 * * *", // Daily at 4 AM (6-field format)
		Priority:    schedulerModels.TaskPriorityNormal,
		Enabled:     true,
		Config: map[string]interface{}{
			"task_name": "asset_snapshot_creator",
		},
		Tags: []string{"assets", "snapshots", "system"},
	})
	if err != nil {
		return fmt.Errorf("failed to register asset snapshot creator: %w", err)
	}

	// Refresh stale character assets every 2 hours
	_, err = schedulerService.CreateTask(ctx, &schedulerDTO.TaskCreateRequest{
		Name:        "Refresh Stale Assets",
		Description: "Refreshes assets that haven't been updated recently",
		Type:        schedulerModels.TaskTypeSystem,
		Schedule:    "0 0 */2 * * *", // Every 2 hours (6-field format)
		Priority:    schedulerModels.TaskPriorityNormal,
		Enabled:     true,
		Config: map[string]interface{}{
			"task_name": "stale_asset_refresher",
		},
		Tags: []string{"assets", "refresh", "system"},
	})
	if err != nil {
		return fmt.Errorf("failed to register stale asset refresher: %w", err)
	}

	// Retry failed structure access every 6 hours
	_, err = schedulerService.CreateTask(ctx, &schedulerDTO.TaskCreateRequest{
		Name:        "Retry Failed Structure Access",
		Description: "Intelligently retries structures that previously returned 403 errors",
		Type:        schedulerModels.TaskTypeSystem,
		Schedule:    "0 0 */6 * * *", // Every 6 hours (6-field format)
		Priority:    schedulerModels.TaskPriorityLow,
		Enabled:     true,
		Config: map[string]interface{}{
			"task_name": "structure_access_retry",
		},
		Tags: []string{"assets", "structures", "retry", "system"},
	})
	if err != nil {
		return fmt.Errorf("failed to register structure access retry task: %w", err)
	}

	return nil
}

// NOTE: The actual task execution logic is handled by the scheduler's system executor
// based on the task_name specified in the Config field. The AssetService methods
// would be called from the system executor when these tasks are scheduled to run.

// These methods would be called by the scheduler's system executor:
// - ProcessAssetTracking() for "asset_tracking_processor"
// - GetAssetSummary() and snapshot creation for "asset_snapshot_creator"
// - RefreshCharacterAssets() for "stale_asset_refresher"
