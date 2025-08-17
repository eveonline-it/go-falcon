package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"go-falcon/internal/notifications/dto"
	"go-falcon/internal/notifications/models"
	"go-falcon/pkg/database"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for notifications
type Repository struct {
	mongodb *database.MongoDB
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb: mongodb,
	}
}

// CreateNotification creates a new notification in the database
func (r *Repository) CreateNotification(ctx context.Context, notification *models.Notification) (*models.Notification, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	// Generate unique notification ID
	notification.NotificationID = uuid.New().String()
	notification.CreatedAt = time.Now()
	
	// Initialize delivery status for all channels
	notification.DeliveryStatus = make(map[string]string)
	for _, channel := range notification.Channels {
		notification.DeliveryStatus[channel] = "pending"
	}
	
	result, err := collection.InsertOne(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}
	
	notification.ID = result.InsertedID.(string)
	return notification, nil
}

// GetNotification retrieves a notification by ID
func (r *Repository) GetNotification(ctx context.Context, notificationID string) (*models.Notification, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	var notification models.Notification
	filter := bson.M{
		"notification_id": notificationID,
		"deleted_at":      bson.M{"$exists": false},
	}
	
	err := collection.FindOne(ctx, filter).Decode(&notification)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	
	return &notification, nil
}

// GetUserNotifications retrieves notifications for a specific user with filtering and pagination
func (r *Repository) GetUserNotifications(ctx context.Context, characterID int, req dto.NotificationSearchRequest) (*dto.NotificationListResponse, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	// Build filter
	filter := bson.M{
		"recipients.character_id": characterID,
		"deleted_at":              bson.M{"$exists": false},
	}
	
	// Add optional filters
	if req.UnreadOnly != nil && *req.UnreadOnly {
		filter["recipients"] = bson.M{
			"$elemMatch": bson.M{
				"character_id": characterID,
				"read":         false,
			},
		}
	}
	
	if req.Type != "" {
		filter["type"] = req.Type
	}
	
	if req.Priority != "" {
		filter["priority"] = req.Priority
	}
	
	if req.Category != "" {
		filter["metadata.category"] = req.Category
	}
	
	if req.FromDate != nil {
		filter["created_at"] = bson.M{"$gte": *req.FromDate}
	}
	
	if req.ToDate != nil {
		if fromDate, exists := filter["created_at"]; exists {
			filter["created_at"] = bson.M{
				"$gte": fromDate,
				"$lte": *req.ToDate,
			}
		} else {
			filter["created_at"] = bson.M{"$lte": *req.ToDate}
		}
	}
	
	// Exclude expired notifications
	filter["$or"] = []bson.M{
		{"expires_at": bson.M{"$exists": false}},
		{"expires_at": bson.M{"$gt": time.Now()}},
	}
	
	// Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count notifications: %w", err)
	}
	
	// Count unread notifications
	unreadFilter := bson.M{
		"recipients": bson.M{
			"$elemMatch": bson.M{
				"character_id": characterID,
				"read":         false,
			},
		},
		"deleted_at": bson.M{"$exists": false},
		"$or": []bson.M{
			{"expires_at": bson.M{"$exists": false}},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}
	unreadCount, err := collection.CountDocuments(ctx, unreadFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count unread notifications: %w", err)
	}
	
	// Calculate pagination
	skip := (req.Page - 1) * req.PageSize
	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize)))
	
	// Build sort options
	sortOrder := 1
	if req.SortOrder == "desc" {
		sortOrder = -1
	}
	sortOptions := bson.D{{req.SortBy, sortOrder}}
	
	// Find notifications with pagination
	findOptions := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(req.PageSize)).
		SetSort(sortOptions)
	
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find notifications: %w", err)
	}
	defer cursor.Close(ctx)
	
	var notifications []models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, fmt.Errorf("failed to decode notifications: %w", err)
	}
	
	// Convert to response DTOs
	notificationResponses := make([]dto.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		notificationResponses[i] = r.toNotificationResponse(&notification, characterID)
	}
	
	return &dto.NotificationListResponse{
		Notifications: notificationResponses,
		Total:         int(total),
		UnreadCount:   int(unreadCount),
		Page:          req.Page,
		PageSize:      req.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// UpdateNotificationStatus updates the read status for a specific recipient
func (r *Repository) UpdateNotificationStatus(ctx context.Context, notificationID string, characterID int, read bool) (*models.Notification, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	filter := bson.M{
		"notification_id":         notificationID,
		"recipients.character_id": characterID,
		"deleted_at":              bson.M{"$exists": false},
	}
	
	update := bson.M{
		"$set": bson.M{
			"recipients.$.read": read,
		},
	}
	
	if read {
		update["$set"].(bson.M)["recipients.$.read_at"] = time.Now()
	} else {
		update["$unset"] = bson.M{"recipients.$.read_at": ""}
	}
	
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update notification status: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("notification not found or access denied")
	}
	
	return r.GetNotification(ctx, notificationID)
}

// DeleteNotification soft deletes a notification
func (r *Repository) DeleteNotification(ctx context.Context, notificationID string, characterID int) error {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	filter := bson.M{
		"notification_id":         notificationID,
		"recipients.character_id": characterID,
		"deleted_at":              bson.M{"$exists": false},
	}
	
	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}
	
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("notification not found or access denied")
	}
	
	return nil
}

// BulkUpdateNotifications performs bulk operations on notifications
func (r *Repository) BulkUpdateNotifications(ctx context.Context, characterID int, action string, notificationIDs []string) (*dto.NotificationBulkResponse, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	filter := bson.M{
		"notification_id":         bson.M{"$in": notificationIDs},
		"recipients.character_id": characterID,
		"deleted_at":              bson.M{"$exists": false},
	}
	
	var update bson.M
	switch action {
	case "mark_read":
		update = bson.M{
			"$set": bson.M{
				"recipients.$.read":    true,
				"recipients.$.read_at": time.Now(),
			},
		}
	case "mark_unread":
		update = bson.M{
			"$set": bson.M{
				"recipients.$.read": false,
			},
			"$unset": bson.M{
				"recipients.$.read_at": "",
			},
		}
	case "delete":
		update = bson.M{
			"$set": bson.M{
				"deleted_at": time.Now(),
			},
		}
	default:
		return nil, fmt.Errorf("invalid bulk action: %s", action)
	}
	
	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to perform bulk update: %w", err)
	}
	
	successCount := int(result.ModifiedCount)
	failureCount := len(notificationIDs) - successCount
	
	return &dto.NotificationBulkResponse{
		Success:      successCount > 0,
		Message:      fmt.Sprintf("Bulk %s completed", action),
		ProcessedIDs: notificationIDs[:successCount],
		FailedIDs:    notificationIDs[successCount:],
		TotalCount:   len(notificationIDs),
		SuccessCount: successCount,
		FailureCount: failureCount,
	}, nil
}

// GetNotificationStats returns notification statistics for a user
func (r *Repository) GetNotificationStats(ctx context.Context, characterID int) (*dto.NotificationStatsResponse, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	// Aggregation pipeline for statistics
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"recipients.character_id": characterID,
				"deleted_at":              bson.M{"$exists": false},
			},
		},
		{
			"$addFields": bson.M{
				"recipient_status": bson.M{
					"$arrayElemAt": []interface{}{
						bson.M{
							"$filter": bson.M{
								"input": "$recipients",
								"cond":  bson.M{"$eq": []interface{}{"$$this.character_id", characterID}},
							},
						},
						0,
					},
				},
				"is_expired": bson.M{
					"$and": []interface{}{
						bson.M{"$ne": []interface{}{"$expires_at", nil}},
						bson.M{"$lt": []interface{}{"$expires_at", time.Now()}},
					},
				},
			},
		},
		{
			"$group": bson.M{
				"_id": nil,
				"total_notifications": bson.M{"$sum": 1},
				"unread_notifications": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$recipient_status.read", false}},
							1,
							0,
						},
					},
				},
				"system_notifications": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$type", "system"}},
							1,
							0,
						},
					},
				},
				"user_notifications": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$type", "user"}},
							1,
							0,
						},
					},
				},
				"alert_notifications": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$type", "alert"}},
							1,
							0,
						},
					},
				},
				"event_notifications": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$type", "event"}},
							1,
							0,
						},
					},
				},
				"expired_notifications": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							"$is_expired",
							1,
							0,
						},
					},
				},
			},
		},
	}
	
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification statistics: %w", err)
	}
	defer cursor.Close(ctx)
	
	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode statistics: %w", err)
	}
	
	stats := &dto.NotificationStatsResponse{}
	if len(results) > 0 {
		result := results[0]
		if val, ok := result["total_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.TotalNotifications = int(count)
			}
		}
		if val, ok := result["unread_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.UnreadNotifications = int(count)
			}
		}
		if val, ok := result["system_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.SystemNotifications = int(count)
			}
		}
		if val, ok := result["user_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.UserNotifications = int(count)
			}
		}
		if val, ok := result["alert_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.AlertNotifications = int(count)
			}
		}
		if val, ok := result["event_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.EventNotifications = int(count)
			}
		}
		if val, ok := result["expired_notifications"]; ok {
			if count, ok := val.(int32); ok {
				stats.ExpiredNotifications = int(count)
			}
		}
	}
	
	return stats, nil
}

// CleanupExpiredNotifications removes expired notifications from the database
func (r *Repository) CleanupExpiredNotifications(ctx context.Context) (int64, error) {
	collection := r.mongodb.Collection(models.Notification{}.CollectionName())
	
	filter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lt":     time.Now(),
		},
		"deleted_at": bson.M{"$exists": false},
	}
	
	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}
	
	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired notifications: %w", err)
	}
	
	return result.ModifiedCount, nil
}

// toNotificationResponse converts a notification model to response DTO
func (r *Repository) toNotificationResponse(notification *models.Notification, characterID int) dto.NotificationResponse {
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