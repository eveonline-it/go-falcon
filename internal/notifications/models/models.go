package models

import (
	"time"
)

// Notification represents a notification document in MongoDB
type Notification struct {
	ID             string                 `json:"id" bson:"_id,omitempty"`
	NotificationID string                 `json:"notification_id" bson:"notification_id"` // UUID for external reference
	Type           string                 `json:"type" bson:"type"`                        // system, user, alert, event
	Title          string                 `json:"title" bson:"title"`
	Message        string                 `json:"message" bson:"message"`
	Priority       string                 `json:"priority" bson:"priority"`               // low, normal, high, critical
	SenderID       *int                   `json:"sender_id,omitempty" bson:"sender_id,omitempty"` // Character ID of sender
	Recipients     []RecipientStatus      `json:"recipients" bson:"recipients"`
	Channels       []string               `json:"channels" bson:"channels"`              // in_app, email, discord
	DeliveryStatus map[string]string      `json:"delivery_status" bson:"delivery_status"` // channel -> status
	Metadata       map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at" bson:"created_at"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
	DeletedAt      *time.Time             `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

// RecipientStatus represents the read status for a specific recipient
type RecipientStatus struct {
	CharacterID int        `json:"character_id" bson:"character_id"`
	Read        bool       `json:"read" bson:"read"`
	ReadAt      *time.Time `json:"read_at,omitempty" bson:"read_at,omitempty"`
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences struct {
	ID                   string                    `json:"id" bson:"_id,omitempty"`
	CharacterID          int                       `json:"character_id" bson:"character_id"`
	EmailNotifications   bool                      `json:"email_notifications" bson:"email_notifications"`
	DiscordNotifications bool                      `json:"discord_notifications" bson:"discord_notifications"`
	PushNotifications    bool                      `json:"push_notifications" bson:"push_notifications"`
	NotificationTypes    map[string]bool           `json:"notification_types" bson:"notification_types"`
	QuietHours           *QuietHours               `json:"quiet_hours,omitempty" bson:"quiet_hours,omitempty"`
	CreatedAt            time.Time                 `json:"created_at" bson:"created_at"`
	UpdatedAt            time.Time                 `json:"updated_at" bson:"updated_at"`
}

// QuietHours represents user's quiet hours preferences
type QuietHours struct {
	Enabled   bool   `json:"enabled" bson:"enabled"`
	StartTime string `json:"start_time" bson:"start_time"` // HH:MM format
	EndTime   string `json:"end_time" bson:"end_time"`     // HH:MM format
	Timezone  string `json:"timezone" bson:"timezone"`
}

// CollectionName returns the MongoDB collection name for notifications
func (Notification) CollectionName() string {
	return "notifications"
}

// CollectionName returns the MongoDB collection name for notification preferences
func (NotificationPreferences) CollectionName() string {
	return "notification_preferences"
}

// IsExpired checks if the notification has expired
func (n *Notification) IsExpired() bool {
	if n.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*n.ExpiresAt)
}

// IsDeleted checks if the notification has been soft deleted
func (n *Notification) IsDeleted() bool {
	return n.DeletedAt != nil
}

// GetRecipientStatus returns the recipient status for a specific character
func (n *Notification) GetRecipientStatus(characterID int) *RecipientStatus {
	for i := range n.Recipients {
		if n.Recipients[i].CharacterID == characterID {
			return &n.Recipients[i]
		}
	}
	return nil
}

// MarkAsReadForRecipient marks the notification as read for a specific recipient
func (n *Notification) MarkAsReadForRecipient(characterID int) bool {
	for i := range n.Recipients {
		if n.Recipients[i].CharacterID == characterID {
			if !n.Recipients[i].Read {
				now := time.Now()
				n.Recipients[i].Read = true
				n.Recipients[i].ReadAt = &now
				return true
			}
			return false
		}
	}
	return false
}

// MarkAsUnreadForRecipient marks the notification as unread for a specific recipient
func (n *Notification) MarkAsUnreadForRecipient(characterID int) bool {
	for i := range n.Recipients {
		if n.Recipients[i].CharacterID == characterID {
			if n.Recipients[i].Read {
				n.Recipients[i].Read = false
				n.Recipients[i].ReadAt = nil
				return true
			}
			return false
		}
	}
	return false
}

// IsReadByRecipient checks if the notification is read by a specific recipient
func (n *Notification) IsReadByRecipient(characterID int) bool {
	status := n.GetRecipientStatus(characterID)
	return status != nil && status.Read
}

// GetDefaultNotificationPreferences returns default notification preferences for a user
func GetDefaultNotificationPreferences(characterID int) *NotificationPreferences {
	now := time.Now()
	return &NotificationPreferences{
		CharacterID:          characterID,
		EmailNotifications:   true,
		DiscordNotifications: false,
		PushNotifications:    false,
		NotificationTypes: map[string]bool{
			"system": true,
			"user":   true,
			"alert":  true,
			"event":  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}