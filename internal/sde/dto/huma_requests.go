package dto

// SDEStatusInput represents the input for getting SDE status (no body needed)
type SDEStatusInput struct {
	// No parameters needed
}

// SDEStatusOutput represents the output for getting SDE status
type SDEStatusOutput struct {
	Body SDEStatusResponse `json:"body"`
}

// SDEHealthInput represents the input for SDE health check (no body needed)
type SDEHealthInput struct {
	// No parameters needed
}

// SDEHealthOutput represents the output for SDE health check
type SDEHealthOutput struct {
	Body SDEHealthResponse `json:"body"`
}

// EntityGetInput represents the input for getting a specific entity
type EntityGetInput struct {
	Type string `path:"type" validate:"required" minLength:"1" maxLength:"50" doc:"SDE entity type (e.g., agents, types, categories)"`
	ID   string `path:"id" validate:"required" minLength:"1" maxLength:"100" doc:"Entity ID"`
}

// EntityGetOutput represents the output for getting a specific entity
type EntityGetOutput struct {
	Body EntityResponse `json:"body"`
}

// EntitiesGetInput represents the input for getting entities by type
type EntitiesGetInput struct {
	Type string `path:"type" validate:"required" minLength:"1" maxLength:"50" doc:"SDE entity type"`
}

// EntitiesGetOutput represents the output for getting entities by type
type EntitiesGetOutput struct {
	Body EntitiesResponse `json:"body"`
}

// SearchSolarSystemInput represents the input for searching solar systems
type SearchSolarSystemInput struct {
	Name string `query:"name" validate:"required" minLength:"1" maxLength:"100" doc:"Solar system name to search for"`
}

// SearchSolarSystemOutput represents the output for searching solar systems
type SearchSolarSystemOutput struct {
	Body SearchSolarSystemResponse `json:"body"`
}

// CheckUpdateInput represents the input for checking SDE updates
type CheckUpdateInput struct {
	Body CheckUpdateRequest `json:"body"`
}

// CheckUpdateOutput represents the output for checking SDE updates
type CheckUpdateOutput struct {
	Body CheckUpdateResponse `json:"body"`
}

// UpdateInput represents the input for starting SDE update
type UpdateInput struct {
	Body UpdateRequest `json:"body"`
}

// UpdateOutput represents the output for starting SDE update
type UpdateOutput struct {
	Body UpdateResponse `json:"body"`
}

// ProgressInput represents the input for getting update progress (no body needed)
type ProgressInput struct {
	// No parameters needed
}

// ProgressOutput represents the output for getting update progress
type ProgressOutput struct {
	Body ProgressResponse `json:"body"`
}

// RebuildIndexInput represents the input for rebuilding search indexes
type RebuildIndexInput struct {
	Body RebuildIndexRequest `json:"body"`
}

// RebuildIndexOutput represents the output for rebuilding search indexes
type RebuildIndexOutput struct {
	Body IndexRebuildResponse `json:"body"`
}

// TestStoreSampleInput represents the input for storing test sample data
type TestStoreSampleInput struct {
	Body TestStoreSampleRequest `json:"body"`
}

// TestStoreSampleOutput represents the output for storing test sample data
type TestStoreSampleOutput struct {
	Body map[string]interface{} `json:"body"`
}

// TestVerifyInput represents the input for verifying test functionality (no body needed)
type TestVerifyInput struct {
	// No parameters needed
}

// TestVerifyOutput represents the output for verifying test functionality
type TestVerifyOutput struct {
	Body TestVerifyResponse `json:"body"`
}

// ConfigGetInput represents the input for getting SDE configuration (no body needed)
type ConfigGetInput struct {
	// No parameters needed
}

// ConfigGetOutput represents the output for getting SDE configuration
type ConfigGetOutput struct {
	Body ConfigResponse `json:"body"`
}

// ConfigUpdateInput represents the input for updating SDE configuration
type ConfigUpdateInput struct {
	Body ConfigUpdateRequest `json:"body"`
}

// ConfigUpdateOutput represents the output for updating SDE configuration
type ConfigUpdateOutput struct {
	Body ConfigResponse `json:"body"`
}

// HistoryGetInput represents the input for getting SDE update history
type HistoryGetInput struct {
	Page      int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize  int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
	StartTime string `query:"start_time" validate:"omitempty" doc:"Start time filter (RFC3339 format)"`
	EndTime   string `query:"end_time" validate:"omitempty" doc:"End time filter (RFC3339 format)"`
	Success   string `query:"success" validate:"omitempty,oneof=true false" doc:"Filter by success status"`
}

// HistoryGetOutput represents the output for getting SDE update history
type HistoryGetOutput struct {
	Body HistoryResponse `json:"body"`
}

// NotificationsGetInput represents the input for getting SDE notifications
type NotificationsGetInput struct {
	Page      int    `query:"page" validate:"omitempty" minimum:"1" doc:"Page number"`
	PageSize  int    `query:"page_size" validate:"omitempty" minimum:"1" maximum:"100" doc:"Number of items per page"`
	Type      string `query:"type" validate:"omitempty,oneof=update_available update_started update_completed update_failed" doc:"Filter by notification type"`
	IsRead    string `query:"is_read" validate:"omitempty,oneof=true false" doc:"Filter by read status"`
	StartTime string `query:"start_time" validate:"omitempty" doc:"Start time filter (RFC3339 format)"`
	EndTime   string `query:"end_time" validate:"omitempty" doc:"End time filter (RFC3339 format)"`
}

// NotificationsGetOutput represents the output for getting SDE notifications
type NotificationsGetOutput struct {
	Body NotificationResponse `json:"body"`
}

// NotificationsMarkReadInput represents the input for marking notifications as read
type NotificationsMarkReadInput struct {
	Body MarkNotificationReadRequest `json:"body"`
}

// NotificationsMarkReadOutput represents the output for marking notifications as read
type NotificationsMarkReadOutput struct {
	Body map[string]interface{} `json:"body"`
}

// BulkEntityInput represents the input for getting multiple entities
type BulkEntityInput struct {
	Body BulkEntityRequest `json:"body"`
}

// BulkEntityOutput represents the output for getting multiple entities
type BulkEntityOutput struct {
	Body BulkEntityResponse `json:"body"`
}

// StatisticsInput represents the input for getting SDE statistics (no body needed)
type StatisticsInput struct {
	// No parameters needed
}

// StatisticsOutput represents the output for getting SDE statistics
type StatisticsOutput struct {
	Body StatisticsResponse `json:"body"`
}