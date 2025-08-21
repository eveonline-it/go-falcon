package dto

import "time"

// SiteSettingOutput represents a site setting in API responses
type SiteSettingOutput struct {
	Key         string      `json:"key" description:"Setting key (unique identifier)"`
	Value       interface{} `json:"value" description:"Setting value"`
	Type        string      `json:"type" description:"Data type of the setting value"`
	Category    string      `json:"category" description:"Setting category"`
	Description string      `json:"description" description:"Description of what this setting controls"`
	IsPublic    bool        `json:"is_public" description:"Whether this setting can be read by non-admin users"`
	IsActive    bool        `json:"is_active" description:"Whether this setting is active"`
	CreatedBy   *int64      `json:"created_by,omitempty" description:"Character ID who created this setting"`
	UpdatedBy   *int64      `json:"updated_by,omitempty" description:"Character ID who last updated this setting"`
	CreatedAt   time.Time   `json:"created_at" description:"Creation timestamp"`
	UpdatedAt   time.Time   `json:"updated_at" description:"Last update timestamp"`
}

// ListSiteSettingsOutput represents the response for listing site settings
type ListSiteSettingsOutput struct {
	Settings   []SiteSettingOutput `json:"settings" description:"List of site settings"`
	Total      int                 `json:"total" description:"Total number of settings"`
	Page       int                 `json:"page" description:"Current page number"`
	Limit      int                 `json:"limit" description:"Items per page"`
	TotalPages int                 `json:"total_pages" description:"Total number of pages"`
}

// GetSiteSettingOutput represents the response for getting a single site setting
type GetSiteSettingOutput struct {
	Setting SiteSettingOutput `json:"setting" description:"Site setting details"`
}

// CreateSiteSettingOutput represents the response for creating a site setting
type CreateSiteSettingOutput struct {
	Setting SiteSettingOutput `json:"setting" description:"Created site setting"`
	Message string            `json:"message" description:"Success message"`
}

// UpdateSiteSettingOutput represents the response for updating a site setting
type UpdateSiteSettingOutput struct {
	Setting SiteSettingOutput `json:"setting" description:"Updated site setting"`
	Message string            `json:"message" description:"Success message"`
}

// DeleteSiteSettingOutput represents the response for deleting a site setting
type DeleteSiteSettingOutput struct {
	Message string `json:"message" description:"Success message"`
}

// HealthOutput represents the health check response
type HealthOutput struct {
	Body SiteSettingsHealthResponse `json:"body"`
}

// SiteSettingsHealthResponse represents the health check response body
type SiteSettingsHealthResponse struct {
	Health string `json:"health" description:"Health status"`
}