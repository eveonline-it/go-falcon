package dto

// NotificationListInput represents the input for listing notifications
type NotificationListInput struct {
	Page     int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Items per page"`
	Type     string `query:"type" validate:"omitempty" doc:"Filter by notification type"`
	IsRead   string `query:"is_read" validate:"omitempty,oneof=true false" doc:"Filter by read status"`
}

// NotificationListOutput represents the output for listing notifications
type NotificationListOutput struct {
	Body NotificationListResponse `json:"body"`
}

// NotificationGetInput represents the input for getting a specific notification
type NotificationGetInput struct {
	NotificationID string `path:"notification_id" validate:"required" doc:"Notification ID"`
}

// NotificationGetOutput represents the output for getting a specific notification
type NotificationGetOutput struct {
	Body NotificationResponse `json:"body"`
}

// NotificationMarkReadInput represents the input for marking a notification as read
type NotificationMarkReadInput struct {
	NotificationID string `path:"notification_id" validate:"required" doc:"Notification ID"`
}

// NotificationMarkReadOutput represents the output for marking a notification as read
type NotificationMarkReadOutput struct {
	Body map[string]interface{} `json:"body"`
}

// NotificationCreateInput represents the input for creating a notification
type NotificationCreateInput struct {
	Body NotificationRequest `json:"body"`
}

// NotificationCreateOutput represents the output for creating a notification
type NotificationCreateOutput struct {
	Body NotificationResponse `json:"body"`
}