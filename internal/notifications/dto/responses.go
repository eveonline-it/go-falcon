package dto

import (
	"time"
)

// NotificationResponse represents a notification in API responses
type NotificationResponse struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Title            string                 `json:"title"`
	Message          string                 `json:"message"`
	Priority         string                 `json:"priority"`
	SenderID         *int                   `json:"sender_id,omitempty"`
	SenderName       string                 `json:"sender_name,omitempty"`
	Read             bool                   `json:"read"`
	ReadAt           *time.Time             `json:"read_at,omitempty"`
	Channels         []string               `json:"channels"`
	DeliveryStatus   map[string]string      `json:"delivery_status"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	ActionURL        string                 `json:"action_url,omitempty"`
	ActionText       string                 `json:"action_text,omitempty"`
	Category         string                 `json:"category,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	ExpiresAt        *time.Time             `json:"expires_at,omitempty"`
}

// NotificationListResponse represents paginated notification list response
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int                    `json:"total"`
	UnreadCount   int                    `json:"unread_count"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	TotalPages    int                    `json:"total_pages"`
}

// NotificationStatsResponse represents notification statistics
type NotificationStatsResponse struct {
	TotalNotifications    int `json:"total_notifications"`
	UnreadNotifications   int `json:"unread_notifications"`
	SystemNotifications   int `json:"system_notifications"`
	UserNotifications     int `json:"user_notifications"`
	AlertNotifications    int `json:"alert_notifications"`
	EventNotifications    int `json:"event_notifications"`
	ExpiredNotifications  int `json:"expired_notifications"`
}

// NotificationCreateResponse represents the response after creating a notification
type NotificationCreateResponse struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	ID           string   `json:"id"`
	Recipients   []int    `json:"recipients"`
	Channels     []string `json:"channels"`
	DeliveryInfo map[string]interface{} `json:"delivery_info,omitempty"`
}

// NotificationUpdateResponse represents the response after updating a notification
type NotificationUpdateResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Notification NotificationResponse `json:"notification"`
}

// NotificationBulkResponse represents the response for bulk operations
type NotificationBulkResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	ProcessedIDs []string `json:"processed_ids"`
	FailedIDs   []string `json:"failed_ids"`
	TotalCount  int      `json:"total_count"`
	SuccessCount int     `json:"success_count"`
	FailureCount int     `json:"failure_count"`
}

// RecipientStatus represents the read status for a specific recipient
type RecipientStatus struct {
	CharacterID int        `json:"character_id"`
	Read        bool       `json:"read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}