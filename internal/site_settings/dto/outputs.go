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
	Body ListSiteSettingsResponseBody `json:"body"`
}

// ListSiteSettingsResponseBody represents the body of the list site settings response
type ListSiteSettingsResponseBody struct {
	Settings   []SiteSettingOutput `json:"settings" description:"List of site settings"`
	Total      int                 `json:"total" description:"Total number of settings"`
	Page       int                 `json:"page" description:"Current page number"`
	Limit      int                 `json:"limit" description:"Items per page"`
	TotalPages int                 `json:"total_pages" description:"Total number of pages"`
}

// GetSiteSettingOutput represents the response for getting a single site setting
type GetSiteSettingOutput struct {
	Body GetSiteSettingResponseBody `json:"body"`
}

// GetSiteSettingResponseBody represents the body of the get site setting response
type GetSiteSettingResponseBody struct {
	Setting SiteSettingOutput `json:"setting" description:"Site setting details"`
}

// CreateSiteSettingOutput represents the response for creating a site setting
type CreateSiteSettingOutput struct {
	Body CreateSiteSettingResponseBody `json:"body"`
}

// CreateSiteSettingResponseBody represents the body of the create site setting response
type CreateSiteSettingResponseBody struct {
	Setting SiteSettingOutput `json:"setting" description:"Created site setting"`
	Message string            `json:"message" description:"Success message"`
}

// UpdateSiteSettingOutput represents the response for updating a site setting
type UpdateSiteSettingOutput struct {
	Body UpdateSiteSettingResponseBody `json:"body"`
}

// UpdateSiteSettingResponseBody represents the body of the update site setting response
type UpdateSiteSettingResponseBody struct {
	Setting SiteSettingOutput `json:"setting" description:"Updated site setting"`
	Message string            `json:"message" description:"Success message"`
}

// DeleteSiteSettingOutput represents the response for deleting a site setting
type DeleteSiteSettingOutput struct {
	Body DeleteSiteSettingResponseBody `json:"body"`
}

// DeleteSiteSettingResponseBody represents the body of the delete site setting response
type DeleteSiteSettingResponseBody struct {
	Message string `json:"message" description:"Success message"`
}

// HealthOutput represents the health check response
type HealthOutput struct {
	Body SiteSettingsHealthResponse `json:"body"`
}

// SiteSettingsHealthResponse represents the health check response body
type SiteSettingsHealthResponse struct {
	Health      string `json:"health" description:"Health status"`
	TotalCount  int    `json:"total_count" description:"Total number of settings"`
	PublicCount int    `json:"public_count" description:"Number of public settings"`
}

// Corporation Management DTOs

// ManagedCorporation represents a managed corporation in API responses
type ManagedCorporation struct {
	CorporationID int64     `json:"corporation_id" description:"EVE Online corporation ID"`
	Name          string    `json:"name" description:"Corporation name"`
	Enabled       bool      `json:"enabled" description:"Whether the corporation is enabled"`
	Position      int       `json:"position" description:"Display order position"`
	AddedAt       time.Time `json:"added_at" description:"When the corporation was added"`
	AddedBy       *int64    `json:"added_by,omitempty" description:"Character ID who added the corporation"`
	UpdatedAt     time.Time `json:"updated_at" description:"When the corporation was last updated"`
	UpdatedBy     *int64    `json:"updated_by,omitempty" description:"Character ID who last updated the corporation"`
}

// AddCorporationOutput represents the response for adding a managed corporation
type AddCorporationOutput struct {
	Body AddCorporationResponseBody `json:"body"`
}

// AddCorporationResponseBody represents the body of the add corporation response
type AddCorporationResponseBody struct {
	Corporation ManagedCorporation `json:"corporation" description:"Added corporation details"`
	Message     string             `json:"message" description:"Success message"`
}

// UpdateCorporationStatusOutput represents the response for updating corporation status
type UpdateCorporationStatusOutput struct {
	Body UpdateCorporationStatusResponseBody `json:"body"`
}

// UpdateCorporationStatusResponseBody represents the body of the update status response
type UpdateCorporationStatusResponseBody struct {
	Corporation ManagedCorporation `json:"corporation" description:"Updated corporation details"`
	Message     string             `json:"message" description:"Success message"`
}

// RemoveCorporationOutput represents the response for removing a managed corporation
type RemoveCorporationOutput struct {
	Body RemoveCorporationResponseBody `json:"body"`
}

// RemoveCorporationResponseBody represents the body of the remove corporation response
type RemoveCorporationResponseBody struct {
	Message string `json:"message" description:"Success message"`
}

// ListManagedCorporationsOutput represents the response for listing managed corporations
type ListManagedCorporationsOutput struct {
	Body ListManagedCorporationsResponseBody `json:"body"`
}

// ListManagedCorporationsResponseBody represents the body of the list corporations response
type ListManagedCorporationsResponseBody struct {
	Corporations []ManagedCorporation `json:"corporations" description:"List of managed corporations"`
	Total        int                  `json:"total" description:"Total number of corporations"`
	Page         int                  `json:"page" description:"Current page number"`
	Limit        int                  `json:"limit" description:"Items per page"`
	TotalPages   int                  `json:"total_pages" description:"Total number of pages"`
}

// GetManagedCorporationOutput represents the response for getting a specific managed corporation
type GetManagedCorporationOutput struct {
	Body GetManagedCorporationResponseBody `json:"body"`
}

// GetManagedCorporationResponseBody represents the body of the get corporation response
type GetManagedCorporationResponseBody struct {
	Corporation ManagedCorporation `json:"corporation" description:"Corporation details"`
}

// BulkUpdateCorporationsOutput represents the response for bulk updating corporations
type BulkUpdateCorporationsOutput struct {
	Body BulkUpdateCorporationsResponseBody `json:"body"`
}

// BulkUpdateCorporationsResponseBody represents the body of the bulk update response
type BulkUpdateCorporationsResponseBody struct {
	Corporations []ManagedCorporation `json:"corporations" description:"Updated corporations"`
	Message      string               `json:"message" description:"Success message"`
	Updated      int                  `json:"updated" description:"Number of corporations updated"`
	Added        int                  `json:"added" description:"Number of corporations added"`
}

// ReorderCorporationsOutput represents the response for reordering corporations
type ReorderCorporationsOutput struct {
	Body ReorderCorporationsResponseBody `json:"body"`
}

// ReorderCorporationsResponseBody represents the body of the reorder response
type ReorderCorporationsResponseBody struct {
	Corporations []ManagedCorporation `json:"corporations" description:"Reordered corporations"`
	Message      string               `json:"message" description:"Success message"`
}

