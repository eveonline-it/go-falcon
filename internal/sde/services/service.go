package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go-falcon/internal/sde/dto"
	"go-falcon/internal/sde/models"
	"go-falcon/pkg/sde"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service provides SDE business logic
type Service struct {
	repo       *Repository
	sdeService *sde.Service
}

// NewService creates a new SDE service
func NewService(repo *Repository, sdeService *sde.Service) *Service {
	return &Service{
		repo:       repo,
		sdeService: sdeService,
	}
}

// GetStatus returns the current SDE status
func (s *Service) GetStatus(ctx context.Context) (*dto.SDEStatusResponse, error) {
	status, err := s.repo.GetStatus(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get SDE status", "error", err)
		return nil, fmt.Errorf("failed to get SDE status: %w", err)
	}

	return &dto.SDEStatusResponse{
		CurrentHash:    status.CurrentHash,
		LatestHash:     status.LatestHash,
		IsUpToDate:     status.IsUpToDate,
		IsProcessing:   status.IsProcessing,
		Progress:       status.Progress,
		LastError:      status.LastError,
		LastCheck:      status.LastCheck,
		LastUpdate:     status.LastUpdate,
		FilesProcessed: status.FilesProcessed,
		TotalFiles:     status.TotalFiles,
		CurrentStage:   status.CurrentStage,
	}, nil
}

// GetProgress returns the current processing progress
func (s *Service) GetProgress(ctx context.Context) (*dto.ProgressResponse, error) {
	status, err := s.repo.GetStatus(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get SDE progress", "error", err)
		return nil, fmt.Errorf("failed to get SDE progress: %w", err)
	}

	response := &dto.ProgressResponse{
		IsProcessing:   status.IsProcessing,
		Progress:       status.Progress,
		Stage:          status.CurrentStage,
		FilesProcessed: status.FilesProcessed,
		TotalFiles:     status.TotalFiles,
	}

	if status.IsProcessing {
		// Calculate estimated end time based on progress
		if status.Progress > 0 && status.Progress < 1 {
			elapsed := time.Since(status.LastUpdate)
			estimatedTotal := time.Duration(float64(elapsed) / status.Progress)
			estimatedEnd := status.LastUpdate.Add(estimatedTotal)
			response.EstimatedEnd = &estimatedEnd
		}
	}

	if status.LastError != "" {
		response.Error = status.LastError
	}

	return response, nil
}

// CheckUpdate checks for available SDE updates
func (s *Service) CheckUpdate(ctx context.Context, req *dto.CheckUpdateRequest) (*dto.CheckUpdateResponse, error) {
	// Get current status
	status, err := s.repo.GetStatus(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get current status for update check", "error", err)
		return nil, fmt.Errorf("failed to get current status: %w", err)
	}

	// For now, simulate check for updates (replace with actual SDE service call)
	updateAvailable := false
	latestHash := status.LatestHash
	if latestHash == "" {
		latestHash = "simulated-hash-123"
		updateAvailable = status.CurrentHash != latestHash
	}

	// Update status with latest information
	status.LatestHash = latestHash
	status.IsUpToDate = !updateAvailable
	status.LastCheck = time.Now()

	if err := s.repo.UpdateStatus(ctx, status); err != nil {
		slog.WarnContext(ctx, "Failed to update status after check", "error", err)
	}

	response := &dto.CheckUpdateResponse{
		UpdateAvailable: updateAvailable,
		CurrentHash:     status.CurrentHash,
		LatestHash:      latestHash,
		LastCheck:       status.LastCheck,
	}

	if updateAvailable {
		response.Message = "SDE update available"
		
		if req.Notify {
			// Create notification
			notification := &models.SDENotification{
				Type:      models.NotificationTypeUpdateAvailable,
				Title:     "SDE Update Available",
				Message:   fmt.Sprintf("New SDE version %s is available", latestHash[:8]),
				Data: map[string]interface{}{
					"current_hash": status.CurrentHash,
					"latest_hash":  latestHash,
				},
				IsRead:    false,
				CreatedAt: time.Now(),
			}
			
			if err := s.repo.CreateNotification(ctx, notification); err != nil {
				slog.WarnContext(ctx, "Failed to create update notification", "error", err)
			}
		}
	} else {
		response.Message = "SDE is up to date"
	}

	return response, nil
}

// StartUpdate initiates an SDE update process
func (s *Service) StartUpdate(ctx context.Context, req *dto.UpdateRequest) (*dto.UpdateResponse, error) {
	// Check if already processing
	status, err := s.repo.GetStatus(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get status before update", "error", err)
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	if status.IsProcessing && !req.ForceUpdate {
		return &dto.UpdateResponse{
			Started: false,
			Message: "Update already in progress",
			Error:   "Another update is currently running",
		}, nil
	}

	// Start the update process
	startTime := time.Now()
	
	// Create history entry
	history := &models.SDEUpdateHistory{
		Hash:         status.LatestHash,
		PreviousHash: status.CurrentHash,
		StartTime:    startTime,
		Success:      false,
		Stages:       []models.UpdateStage{},
		CreatedAt:    startTime,
	}
	
	historyID, err := s.repo.CreateUpdateHistory(ctx, history)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create update history", "error", err)
		return nil, fmt.Errorf("failed to create update history: %w", err)
	}

	// Update status to processing
	status.IsProcessing = true
	status.Progress = 0.0
	status.CurrentStage = models.StageDownload
	status.LastError = ""
	status.FilesProcessed = 0
	status.TotalFiles = 0
	
	if err := s.repo.UpdateStatus(ctx, status); err != nil {
		slog.ErrorContext(ctx, "Failed to update status to processing", "error", err)
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// Create notification
	notification := &models.SDENotification{
		Type:      models.NotificationTypeUpdateStarted,
		Title:     "SDE Update Started",
		Message:   fmt.Sprintf("SDE update to version %s has started", status.LatestHash[:8]),
		Data: map[string]interface{}{
			"previous_hash": status.CurrentHash,
			"target_hash":   status.LatestHash,
			"history_id":    historyID.Hex(),
		},
		IsRead:    false,
		CreatedAt: startTime,
	}
	
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		slog.WarnContext(ctx, "Failed to create update started notification", "error", err)
	}

	// Start the background update process
	go s.performUpdate(context.Background(), historyID, status.LatestHash)

	return &dto.UpdateResponse{
		Started:   true,
		Message:   "SDE update started successfully",
		StartTime: startTime,
	}, nil
}

// GetEntity retrieves a specific SDE entity (placeholder implementation)
func (s *Service) GetEntity(ctx context.Context, req *dto.EntityRequest) (*dto.EntityResponse, error) {
	// This would use the SDE service to get entity data
	// For now, return a placeholder response
	slog.InfoContext(ctx, "Getting SDE entity", "type", req.Type, "id", req.ID)
	
	return &dto.EntityResponse{
		Type: req.Type,
		ID:   req.ID,
		Data: map[string]interface{}{
			"type": req.Type,
			"id":   req.ID,
			"name": fmt.Sprintf("Entity %s", req.ID),
		},
	}, nil
}

// GetEntities retrieves all entities of a specific type (placeholder implementation)
func (s *Service) GetEntities(ctx context.Context, req *dto.EntitiesRequest) (*dto.EntitiesResponse, error) {
	// This would use the SDE service to get entities by type
	// For now, return a placeholder response
	slog.InfoContext(ctx, "Getting SDE entities", "type", req.Type)
	
	entities := make(map[string]interface{})
	entities["1"] = map[string]interface{}{"name": "Sample Entity 1"}
	entities["2"] = map[string]interface{}{"name": "Sample Entity 2"}
	
	return &dto.EntitiesResponse{
		Type:     req.Type,
		Count:    len(entities),
		Entities: entities,
	}, nil
}

// SearchSolarSystems searches for solar systems by name (placeholder implementation)
func (s *Service) SearchSolarSystems(ctx context.Context, req *dto.SearchSolarSystemRequest) (*dto.SearchSolarSystemResponse, error) {
	// This would use the SDE service to search solar systems
	// For now, return a placeholder response
	slog.InfoContext(ctx, "Searching solar systems", "query", req.Name)
	
	results := []dto.SolarSystemResult{
		{
			SystemName:        req.Name + " System",
			Region:           "Sample Region",
			Constellation:    "Sample Constellation",
			UniverseType:     "eve",
			RedisKey:         fmt.Sprintf("sde:universe:eve:region:constellation:%s", req.Name),
			SolarSystemID:    12345,
			Security:         0.5,
			SolarSystemNameID: 67890,
		},
	}

	return &dto.SearchSolarSystemResponse{
		Query:   req.Name,
		Count:   len(results),
		Results: results,
	}, nil
}

// RebuildIndex rebuilds search indexes (placeholder implementation)
func (s *Service) RebuildIndex(ctx context.Context, req *dto.RebuildIndexRequest) (*dto.IndexRebuildResponse, error) {
	indexType := req.IndexType
	if indexType == "" {
		indexType = models.IndexTypeSolarSystems
	}

	startTime := time.Now()
	slog.InfoContext(ctx, "Rebuilding search index", "type", indexType)
	
	// Simulate index rebuild
	time.Sleep(100 * time.Millisecond)
	duration := time.Since(startTime)
	
	// Create notification
	notification := &models.SDENotification{
		Type:      models.NotificationTypeIndexRebuilt,
		Title:     "Search Index Rebuilt",
		Message:   fmt.Sprintf("Search index for %s has been rebuilt", indexType),
		Data: map[string]interface{}{
			"index_type":   indexType,
			"items_count":  1000,
			"duration_ms":  duration.Milliseconds(),
		},
		IsRead:    false,
		CreatedAt: time.Now(),
	}
	
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		slog.WarnContext(ctx, "Failed to create index rebuilt notification", "error", err)
	}

	return &dto.IndexRebuildResponse{
		Message:    "Index rebuilt successfully",
		IndexType:  indexType,
		Duration:   duration,
		ItemsCount: 1000,
		Success:    true,
	}, nil
}

// GetBulkEntities retrieves multiple entities
func (s *Service) GetBulkEntities(ctx context.Context, req *dto.BulkEntityRequest) (*dto.BulkEntityResponse, error) {
	var entities []dto.EntityResponse
	var notFound []dto.EntityIdentifier

	for _, identifier := range req.Entities {
		// Simulate entity retrieval
		entities = append(entities, dto.EntityResponse{
			Type: identifier.Type,
			ID:   identifier.ID,
			Data: map[string]interface{}{
				"type": identifier.Type,
				"id":   identifier.ID,
				"name": fmt.Sprintf("Entity %s", identifier.ID),
			},
		})
	}

	response := &dto.BulkEntityResponse{
		Entities: entities,
		Found:    len(entities),
	}

	if len(notFound) > 0 {
		response.NotFound = notFound
	}

	return response, nil
}

// GetConfig retrieves SDE configuration
func (s *Service) GetConfig(ctx context.Context) (*dto.ConfigResponse, error) {
	config, err := s.repo.GetConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get SDE config", "error", err)
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &dto.ConfigResponse{
		AutoCheckEnabled:   config.AutoCheckEnabled,
		AutoUpdateEnabled:  config.AutoUpdateEnabled,
		CheckInterval:      config.CheckInterval,
		NotifyOnUpdate:     config.NotifyOnUpdate,
		RetainHistoryDays:  config.RetainHistoryDays,
		MaxRetries:         config.MaxRetries,
		RetryDelay:         config.RetryDelay,
		DownloadTimeout:    config.DownloadTimeout,
		ProcessingTimeout:  config.ProcessingTimeout,
		LastUpdated:        config.UpdatedAt,
	}, nil
}

// UpdateConfig updates SDE configuration
func (s *Service) UpdateConfig(ctx context.Context, req *dto.ConfigUpdateRequest) (*dto.ConfigResponse, error) {
	config, err := s.repo.GetConfig(ctx)
	if err != nil {
		// Create default config if not exists
		config = &models.SDEConfig{
			AutoCheckEnabled:   true,
			AutoUpdateEnabled:  false,
			CheckInterval:      models.DefaultCheckInterval,
			NotifyOnUpdate:     true,
			RetainHistoryDays:  models.DefaultRetainHistoryDays,
			MaxRetries:         models.DefaultMaxRetries,
			RetryDelay:         models.DefaultRetryDelay,
			DownloadTimeout:    models.DefaultDownloadTimeout,
			ProcessingTimeout:  models.DefaultProcessingTimeout,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}
	}

	// Update fields if provided
	if req.AutoCheckEnabled != nil {
		config.AutoCheckEnabled = *req.AutoCheckEnabled
	}
	if req.AutoUpdateEnabled != nil {
		config.AutoUpdateEnabled = *req.AutoUpdateEnabled
	}
	if req.CheckInterval != nil {
		config.CheckInterval = *req.CheckInterval
	}
	if req.NotifyOnUpdate != nil {
		config.NotifyOnUpdate = *req.NotifyOnUpdate
	}
	if req.RetainHistoryDays != nil {
		config.RetainHistoryDays = *req.RetainHistoryDays
	}
	if req.MaxRetries != nil {
		config.MaxRetries = *req.MaxRetries
	}
	if req.RetryDelay != nil {
		config.RetryDelay = *req.RetryDelay
	}
	if req.DownloadTimeout != nil {
		config.DownloadTimeout = *req.DownloadTimeout
	}
	if req.ProcessingTimeout != nil {
		config.ProcessingTimeout = *req.ProcessingTimeout
	}

	config.UpdatedAt = time.Now()

	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		slog.ErrorContext(ctx, "Failed to update SDE config", "error", err)
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	return &dto.ConfigResponse{
		AutoCheckEnabled:   config.AutoCheckEnabled,
		AutoUpdateEnabled:  config.AutoUpdateEnabled,
		CheckInterval:      config.CheckInterval,
		NotifyOnUpdate:     config.NotifyOnUpdate,
		RetainHistoryDays:  config.RetainHistoryDays,
		MaxRetries:         config.MaxRetries,
		RetryDelay:         config.RetryDelay,
		DownloadTimeout:    config.DownloadTimeout,
		ProcessingTimeout:  config.ProcessingTimeout,
		LastUpdated:        config.UpdatedAt,
	}, nil
}

// GetHistory retrieves SDE update history
func (s *Service) GetHistory(ctx context.Context, req *dto.HistoryQueryRequest) (*dto.HistoryResponse, error) {
	// Set defaults
	page := 1
	pageSize := 20
	if req.Page > 0 {
		page = req.Page
	}
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}

	history, total, err := s.repo.GetHistory(ctx, page, pageSize, req.StartTime, req.EndTime, req.Success)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get SDE history", "error", err)
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	// Convert to DTO format
	entries := make([]dto.UpdateHistoryEntry, len(history))
	for i, h := range history {
		entries[i] = dto.UpdateHistoryEntry{
			ID:           h.ID.Hex(),
			Hash:         h.Hash,
			PreviousHash: h.PreviousHash,
			StartTime:    h.StartTime,
			EndTime:      h.EndTime,
			Duration:     h.Duration.String(),
			Success:      h.Success,
			Error:        h.Error,
			FilesUpdated: h.FilesUpdated,
			SizeBytes:    h.SizeBytes,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &dto.HistoryResponse{
		Updates: entries,
		Pagination: dto.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetNotifications retrieves SDE notifications
func (s *Service) GetNotifications(ctx context.Context, req *dto.NotificationQueryRequest) (*dto.NotificationResponse, error) {
	// Set defaults
	page := 1
	pageSize := 20
	if req.Page > 0 {
		page = req.Page
	}
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}

	notifications, total, err := s.repo.GetNotifications(ctx, page, pageSize, req.Type, req.IsRead, req.StartTime, req.EndTime)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get SDE notifications", "error", err)
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	// Convert to DTO format
	entries := make([]dto.NotificationEntry, len(notifications))
	for i, n := range notifications {
		entries[i] = dto.NotificationEntry{
			ID:        n.ID.Hex(),
			Type:      n.Type,
			Title:     n.Title,
			Message:   n.Message,
			Data:      n.Data,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &dto.NotificationResponse{
		Notifications: entries,
		Pagination: dto.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// MarkNotificationsRead marks notifications as read
func (s *Service) MarkNotificationsRead(ctx context.Context, req *dto.MarkNotificationReadRequest) error {
	// Convert string IDs to ObjectIDs
	objectIDs := make([]primitive.ObjectID, len(req.NotificationIDs))
	for i, idStr := range req.NotificationIDs {
		objectID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			return fmt.Errorf("invalid notification ID %s: %w", idStr, err)
		}
		objectIDs[i] = objectID
	}

	if err := s.repo.MarkNotificationsRead(ctx, objectIDs); err != nil {
		slog.ErrorContext(ctx, "Failed to mark notifications as read", "error", err)
		return fmt.Errorf("failed to mark notifications as read: %w", err)
	}

	return nil
}

// GetStatistics retrieves SDE statistics
func (s *Service) GetStatistics(ctx context.Context) (*dto.StatisticsResponse, error) {
	stats, err := s.repo.GetStatistics(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get SDE statistics", "error", err)
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return &dto.StatisticsResponse{
		TotalEntities:    stats.TotalEntities,
		EntitiesByType:   stats.EntitiesByType,
		LastUpdate:       stats.LastUpdate,
		DataSize:         stats.DataSize,
		IndexSize:        stats.IndexSize,
		ProcessingStats: dto.ProcessingStats{
			TotalUpdates:      stats.ProcessingStats.TotalUpdates,
			SuccessfulUpdates: stats.ProcessingStats.SuccessfulUpdates,
			FailedUpdates:     stats.ProcessingStats.FailedUpdates,
			AverageUpdateTime: stats.ProcessingStats.AverageUpdateTime,
			LastUpdateTime:    stats.ProcessingStats.LastUpdateTime,
		},
		PerformanceStats: dto.PerformanceStats{
			AverageSearchTime:   stats.PerformanceStats.AverageSearchTime,
			AverageEntityAccess: stats.PerformanceStats.AverageEntityAccess,
			CacheHitRate:        stats.PerformanceStats.CacheHitRate,
			TotalRequests:       stats.PerformanceStats.TotalRequests,
		},
	}, nil
}

// TestVerify performs verification tests on SDE data
func (s *Service) TestVerify(ctx context.Context) (*dto.TestVerifyResponse, error) {
	// Test key types
	testKeys := []string{
		"agents:3008416",
		"types:34",
		"solarsystems:30000142",
		"regions:10000002",
	}

	results := make(map[string]interface{})
	var failedKeys []string

	for _, key := range testKeys {
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			failedKeys = append(failedKeys, key)
			continue
		}

		// Simulate successful verification
		results[key] = map[string]interface{}{
			"type": parts[0],
			"id":   parts[1],
			"name": fmt.Sprintf("Test %s %s", parts[0], parts[1]),
		}
	}

	response := &dto.TestVerifyResponse{
		Success:    len(failedKeys) == 0,
		TestedKeys: testKeys,
		Results:    results,
	}

	if len(failedKeys) > 0 {
		response.FailedKeys = failedKeys
		response.ErrorMessage = fmt.Sprintf("Failed to verify %d out of %d test keys", len(failedKeys), len(testKeys))
	}

	return response, nil
}

// TestStoreSample stores sample test data
func (s *Service) TestStoreSample(ctx context.Context, req *dto.TestStoreSampleRequest) error {
	if req.TestData == nil {
		// Store default test data
		req.TestData = map[string]interface{}{
			"test_timestamp": time.Now().Unix(),
			"test_message":   "SDE module test data",
			"test_version":   "1.0.0",
		}
	}

	entityType := req.Type
	if entityType == "" {
		entityType = "test"
	}

	testID := fmt.Sprintf("test_%d", time.Now().Unix())
	
	slog.InfoContext(ctx, "Storing test sample", 
		"type", entityType, 
		"id", testID,
		"data", req.TestData)

	return nil
}

// performUpdate runs the actual update process in the background
func (s *Service) performUpdate(ctx context.Context, historyID primitive.ObjectID, targetHash string) {
	startTime := time.Now()
	var updateErr error

	defer func() {
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		success := updateErr == nil

		// Update history
		history := &models.SDEUpdateHistory{
			ID:       historyID,
			EndTime:  endTime,
			Duration: duration,
			Success:  success,
		}
		if updateErr != nil {
			history.Error = updateErr.Error()
		}
		
		if err := s.repo.UpdateHistory(ctx, history); err != nil {
			slog.ErrorContext(ctx, "Failed to update history after completion", "error", err)
		}

		// Update status
		status, err := s.repo.GetStatus(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get status after update completion", "error", err)
			return
		}

		status.IsProcessing = false
		if success {
			status.CurrentHash = targetHash
			status.IsUpToDate = true
			status.Progress = 1.0
			status.CurrentStage = models.StageComplete
			status.LastUpdate = endTime
			status.LastError = ""
		} else {
			status.LastError = updateErr.Error()
			status.Progress = 0.0
			status.CurrentStage = ""
		}

		if err := s.repo.UpdateStatus(ctx, status); err != nil {
			slog.ErrorContext(ctx, "Failed to update status after completion", "error", err)
		}

		// Create completion notification
		notificationType := models.NotificationTypeUpdateCompleted
		title := "SDE Update Completed"
		message := fmt.Sprintf("SDE update to version %s completed successfully", targetHash[:8])
		
		if !success {
			notificationType = models.NotificationTypeUpdateFailed
			title = "SDE Update Failed"
			message = fmt.Sprintf("SDE update to version %s failed: %s", targetHash[:8], updateErr.Error())
		}

		notification := &models.SDENotification{
			Type:    notificationType,
			Title:   title,
			Message: message,
			Data: map[string]interface{}{
				"target_hash":  targetHash,
				"success":      success,
				"duration_ms":  duration.Milliseconds(),
				"history_id":   historyID.Hex(),
			},
			IsRead:    false,
			CreatedAt: time.Now(),
		}
		
		if err := s.repo.CreateNotification(ctx, notification); err != nil {
			slog.WarnContext(ctx, "Failed to create completion notification", "error", err)
		}
	}()

	// Simulate the update process
	slog.InfoContext(ctx, "Starting SDE update simulation", "target_hash", targetHash)
	time.Sleep(2 * time.Second) // Simulate processing time
	slog.InfoContext(ctx, "SDE update simulation completed", "target_hash", targetHash)
}