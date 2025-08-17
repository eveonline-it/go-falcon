package services

import (
	"context"
	"fmt"
	"log/slog"

	"go-falcon/internal/notifications/dto"
	"go-falcon/internal/notifications/models"
	"go-falcon/pkg/database"
)

// Service provides business logic for notification operations
type Service struct {
	repository *Repository
}

// NewService creates a new service instance
func NewService(mongodb *database.MongoDB) *Service {
	return &Service{
		repository: NewRepository(mongodb),
	}
}

// CreateNotification creates a new notification
func (s *Service) CreateNotification(ctx context.Context, req dto.NotificationRequest, senderID *int) (*dto.NotificationCreateResponse, error) {
	// Validate the request
	req.SetDefaults()
	if err := dto.ValidateNotificationRequest(&req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Create notification model
	notification := &models.Notification{
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		Priority:  req.Priority,
		SenderID:  senderID,
		Channels:  req.Channels,
		ExpiresAt: req.ExpiresAt,
		Metadata:  make(map[string]interface{}),
	}
	
	// Add metadata
	if req.ActionURL != "" {
		notification.Metadata["action_url"] = req.ActionURL
	}
	if req.ActionText != "" {
		notification.Metadata["action_text"] = req.ActionText
	}
	if req.Category != "" {
		notification.Metadata["category"] = req.Category
	}
	
	// Create recipient statuses
	notification.Recipients = make([]models.RecipientStatus, len(req.Recipients))
	for i, recipientID := range req.Recipients {
		notification.Recipients[i] = models.RecipientStatus{
			CharacterID: recipientID,
			Read:        false,
		}
	}
	
	// Create notification in database
	createdNotification, err := s.repository.CreateNotification(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}
	
	// Log notification creation
	slog.Info("Notification created",
		slog.String("notification_id", createdNotification.NotificationID),
		slog.String("type", createdNotification.Type),
		slog.String("priority", createdNotification.Priority),
		slog.Int("recipient_count", len(req.Recipients)),
		slog.Any("channels", req.Channels))
	
	return &dto.NotificationCreateResponse{
		Success:    true,
		Message:    "Notification created successfully",
		ID:         createdNotification.NotificationID,
		Recipients: req.Recipients,
		Channels:   req.Channels,
		DeliveryInfo: map[string]interface{}{
			"created_at":      createdNotification.CreatedAt,
			"delivery_status": createdNotification.DeliveryStatus,
		},
	}, nil
}

// GetUserNotifications retrieves notifications for a specific user with filtering and pagination
func (s *Service) GetUserNotifications(ctx context.Context, characterID int, req dto.NotificationSearchRequest) (*dto.NotificationListResponse, error) {
	// Set defaults and validate
	req.SetDefaults()
	if err := dto.ValidateNotificationSearchRequest(&req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	return s.repository.GetUserNotifications(ctx, characterID, req)
}

// GetNotification retrieves a specific notification
func (s *Service) GetNotification(ctx context.Context, notificationID string, characterID int) (*dto.NotificationResponse, error) {
	notification, err := s.repository.GetNotification(ctx, notificationID)
	if err != nil {
		return nil, err
	}
	
	// Check if user has access to this notification
	if !s.hasAccessToNotification(notification, characterID) {
		return nil, fmt.Errorf("access denied: notification not found or access denied")
	}
	
	// Convert to response DTO
	response := s.toNotificationResponse(notification, characterID)
	return &response, nil
}

// UpdateNotificationStatus updates the read status of a notification for a specific user
func (s *Service) UpdateNotificationStatus(ctx context.Context, notificationID string, characterID int, req dto.NotificationUpdateRequest) (*dto.NotificationUpdateResponse, error) {
	// Validate the request
	if err := dto.ValidateNotificationUpdateRequest(&req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	if req.Read == nil {
		return nil, fmt.Errorf("read status must be specified")
	}
	
	// Update notification status
	notification, err := s.repository.UpdateNotificationStatus(ctx, notificationID, characterID, *req.Read)
	if err != nil {
		return nil, err
	}
	
	// Log status update
	status := "unread"
	if *req.Read {
		status = "read"
	}
	slog.Info("Notification status updated",
		slog.String("notification_id", notificationID),
		slog.Int("character_id", characterID),
		slog.String("status", status))
	
	// Convert to response DTO
	response := s.toNotificationResponse(notification, characterID)
	
	return &dto.NotificationUpdateResponse{
		Success:      true,
		Message:      fmt.Sprintf("Notification marked as %s", status),
		Notification: response,
	}, nil
}

// DeleteNotification soft deletes a notification for a specific user
func (s *Service) DeleteNotification(ctx context.Context, notificationID string, characterID int) error {
	err := s.repository.DeleteNotification(ctx, notificationID, characterID)
	if err != nil {
		return err
	}
	
	// Log deletion
	slog.Info("Notification deleted",
		slog.String("notification_id", notificationID),
		slog.Int("character_id", characterID))
	
	return nil
}

// BulkUpdateNotifications performs bulk operations on notifications
func (s *Service) BulkUpdateNotifications(ctx context.Context, characterID int, req dto.NotificationBulkRequest) (*dto.NotificationBulkResponse, error) {
	// Validate the request
	if err := dto.ValidateNotificationBulkRequest(&req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Perform bulk operation
	response, err := s.repository.BulkUpdateNotifications(ctx, characterID, req.Action, req.NotificationIDs)
	if err != nil {
		return nil, err
	}
	
	// Log bulk operation
	slog.Info("Bulk notification operation completed",
		slog.String("action", req.Action),
		slog.Int("character_id", characterID),
		slog.Int("total_count", response.TotalCount),
		slog.Int("success_count", response.SuccessCount),
		slog.Int("failure_count", response.FailureCount))
	
	return response, nil
}

// GetNotificationStats returns notification statistics for a user
func (s *Service) GetNotificationStats(ctx context.Context, characterID int) (*dto.NotificationStatsResponse, error) {
	return s.repository.GetNotificationStats(ctx, characterID)
}

// CleanupExpiredNotifications removes expired notifications (background task)
func (s *Service) CleanupExpiredNotifications(ctx context.Context) (int64, error) {
	count, err := s.repository.CleanupExpiredNotifications(ctx)
	if err != nil {
		return 0, err
	}
	
	if count > 0 {
		slog.Info("Expired notifications cleaned up", slog.Int64("count", count))
	}
	
	return count, nil
}

// SendSystemNotification is a convenience method for sending system notifications
func (s *Service) SendSystemNotification(ctx context.Context, title, message string, recipients []int) (*dto.NotificationCreateResponse, error) {
	req := dto.NotificationRequest{
		Type:       "system",
		Title:      title,
		Message:    message,
		Priority:   "normal",
		Recipients: recipients,
		Channels:   []string{"in_app"},
		Category:   "system",
	}
	
	return s.CreateNotification(ctx, req, nil)
}

// SendAlertNotification is a convenience method for sending alert notifications
func (s *Service) SendAlertNotification(ctx context.Context, title, message string, recipients []int, priority string) (*dto.NotificationCreateResponse, error) {
	if priority == "" {
		priority = "high"
	}
	
	req := dto.NotificationRequest{
		Type:       "alert",
		Title:      title,
		Message:    message,
		Priority:   priority,
		Recipients: recipients,
		Channels:   []string{"in_app"},
		Category:   "alert",
	}
	
	return s.CreateNotification(ctx, req, nil)
}

// hasAccessToNotification checks if a user has access to a specific notification
func (s *Service) hasAccessToNotification(notification *models.Notification, characterID int) bool {
	if notification.IsDeleted() {
		return false
	}
	
	// Check if user is a recipient
	status := notification.GetRecipientStatus(characterID)
	return status != nil
}

// toNotificationResponse converts a notification model to response DTO
func (s *Service) toNotificationResponse(notification *models.Notification, characterID int) dto.NotificationResponse {
	status := notification.GetRecipientStatus(characterID)
	
	response := dto.NotificationResponse{
		ID:             notification.NotificationID,
		Type:           notification.Type,
		Title:          notification.Title,
		Message:        notification.Message,
		Priority:       notification.Priority,
		SenderID:       notification.SenderID,
		Read:           false,
		Channels:       notification.Channels,
		DeliveryStatus: notification.DeliveryStatus,
		Metadata:       notification.Metadata,
		CreatedAt:      notification.CreatedAt,
		ExpiresAt:      notification.ExpiresAt,
	}
	
	if status != nil {
		response.Read = status.Read
		response.ReadAt = status.ReadAt
	}
	
	// Extract metadata fields
	if notification.Metadata != nil {
		if actionURL, ok := notification.Metadata["action_url"].(string); ok {
			response.ActionURL = actionURL
		}
		if actionText, ok := notification.Metadata["action_text"].(string); ok {
			response.ActionText = actionText
		}
		if category, ok := notification.Metadata["category"].(string); ok {
			response.Category = category
		}
	}
	
	return response
}