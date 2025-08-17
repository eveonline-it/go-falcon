package dto

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// NotificationRequest represents a request to create a new notification
type NotificationRequest struct {
	Type       string    `json:"type" validate:"required,oneof=system user alert event"`
	Title      string    `json:"title" validate:"required,min=3,max=200"`
	Message    string    `json:"message" validate:"required,min=10,max=2000"`
	Priority   string    `json:"priority" validate:"required,oneof=low normal high critical"`
	Recipients []int     `json:"recipients" validate:"required,min=1,max=100,dive,min=1"`
	Channels   []string  `json:"channels" validate:"omitempty,dive,oneof=in_app email discord"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" validate:"omitempty,gtfield=CreatedAt"`
	ActionURL  string    `json:"action_url,omitempty" validate:"omitempty,url"`
	ActionText string    `json:"action_text,omitempty" validate:"omitempty,max=50"`
	Category   string    `json:"category,omitempty" validate:"omitempty,max=100"`
}

// NotificationUpdateRequest represents a request to update notification status
type NotificationUpdateRequest struct {
	Read *bool `json:"read,omitempty"`
}

// NotificationBulkRequest represents a bulk operation request
type NotificationBulkRequest struct {
	Action          string   `json:"action" validate:"required,oneof=mark_read mark_unread delete"`
	NotificationIDs []string `json:"notification_ids" validate:"required,min=1,max=100,dive,min=1"`
}

// NotificationSearchRequest represents search and filter parameters
type NotificationSearchRequest struct {
	UnreadOnly *bool     `json:"unread_only" form:"unread_only"`
	Type       string    `json:"type" form:"type" validate:"omitempty,oneof=system user alert event"`
	Priority   string    `json:"priority" form:"priority" validate:"omitempty,oneof=low normal high critical"`
	Category   string    `json:"category" form:"category" validate:"omitempty,max=100"`
	FromDate   *time.Time `json:"from_date" form:"from_date"`
	ToDate     *time.Time `json:"to_date" form:"to_date"`
	Page       int       `json:"page" form:"page" validate:"omitempty,min=1"`
	PageSize   int       `json:"page_size" form:"page_size" validate:"omitempty,min=1,max=100"`
	SortBy     string    `json:"sort_by" form:"sort_by" validate:"omitempty,oneof=created_at title priority type"`
	SortOrder  string    `json:"sort_order" form:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ValidateNotificationRequest validates the notification creation request
func ValidateNotificationRequest(req *NotificationRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// ValidateNotificationUpdateRequest validates the notification update request
func ValidateNotificationUpdateRequest(req *NotificationUpdateRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// ValidateNotificationBulkRequest validates the bulk operation request
func ValidateNotificationBulkRequest(req *NotificationBulkRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// ValidateNotificationSearchRequest validates the search request
func ValidateNotificationSearchRequest(req *NotificationSearchRequest) error {
	validate := validator.New()
	return validate.Struct(req)
}

// SetDefaults sets default values for NotificationSearchRequest
func (r *NotificationSearchRequest) SetDefaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	if r.SortBy == "" {
		r.SortBy = "created_at"
	}
	if r.SortOrder == "" {
		r.SortOrder = "desc"
	}
}

// SetDefaults sets default values for NotificationRequest
func (r *NotificationRequest) SetDefaults() {
	if len(r.Channels) == 0 {
		r.Channels = []string{"in_app"}
	}
	if r.Priority == "" {
		r.Priority = "normal"
	}
}