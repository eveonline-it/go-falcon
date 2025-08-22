package dto

// CreateSiteSettingInput represents the input for creating a new site setting
type CreateSiteSettingInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Body          struct {
		Key         string      `json:"key" minLength:"1" maxLength:"100" required:"true" description:"Setting key (unique identifier)"`
		Value       interface{} `json:"value" required:"true" description:"Setting value (can be string, number, boolean, or object)"`
		Type        string      `json:"type" enum:"string,number,boolean,object" required:"true" description:"Data type of the setting value"`
		Category    string      `json:"category" maxLength:"50" description:"Setting category for organization"`
		Description string      `json:"description" maxLength:"500" description:"Description of what this setting controls"`
		IsPublic    bool        `json:"is_public" description:"Whether this setting can be read by non-admin users"`
	}
}

// UpdateSiteSettingInput represents the input for updating a site setting
type UpdateSiteSettingInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Key           string `path:"key" required:"true" description:"Setting key"`
	Body          struct {
		Value       interface{} `json:"value,omitempty" description:"Setting value (can be string, number, boolean, or object)"`
		Type        *string     `json:"type,omitempty" enum:"string,number,boolean,object" description:"Data type of the setting value"`
		Category    *string     `json:"category,omitempty" maxLength:"50" description:"Setting category for organization"`
		Description *string     `json:"description,omitempty" maxLength:"500" description:"Description of what this setting controls"`
		IsPublic    *bool       `json:"is_public,omitempty" description:"Whether this setting can be read by non-admin users"`
		IsActive    *bool       `json:"is_active,omitempty" description:"Whether this setting is active"`
	}
}

// GetSiteSettingInput represents the input for getting a specific site setting
type GetSiteSettingInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Key           string `path:"key" required:"true" description:"Setting key"`
}

// ListSiteSettingsInput represents the input for listing site settings
type ListSiteSettingsInput struct {
	Authorization  string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie         string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Category       string `query:"category" description:"Filter by setting category"`
	IsPublicFilter string `query:"is_public" description:"Filter by public/private settings: 'true', 'false', or empty for all"`
	IsActiveFilter string `query:"is_active" description:"Filter by active/inactive settings: 'true', 'false', or empty for all"`
	Page           int    `query:"page" minimum:"1" default:"1" description:"Page number"`
	Limit          int    `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Items per page"`
}

// DeleteSiteSettingInput represents the input for deleting a site setting
type DeleteSiteSettingInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Key           string `path:"key" required:"true" description:"Setting key"`
}

// GetPublicSiteSettingsInput represents the input for getting public site settings (no auth required)
type GetPublicSiteSettingsInput struct {
	Category string `query:"category" description:"Filter by setting category"`
	Page     int    `query:"page" minimum:"1" default:"1" description:"Page number"`
	Limit    int    `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Items per page"`
}

// Corporation Management DTOs

// AddCorporationInput represents the input for adding a new managed corporation
type AddCorporationInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Body          struct {
		CorporationID int64  `json:"corporation_id" required:"true" description:"EVE Online corporation ID"`
		Name          string `json:"name" minLength:"1" maxLength:"100" required:"true" description:"Corporation name"`
		Enabled       *bool  `json:"enabled" description:"Whether the corporation should be enabled (defaults to true)"`
		Position      *int   `json:"position" minimum:"1" description:"Display position (auto-assigned if not provided)"`
	}
}

// UpdateCorporationStatusInput represents the input for enabling/disabling a corporation
type UpdateCorporationStatusInput struct {
	Authorization   string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie          string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	CorporationID   int64  `path:"corp_id" required:"true" minimum:"1" description:"Corporation ID"`
	Body            struct {
		Enabled bool `json:"enabled" required:"true" description:"Whether the corporation should be enabled"`
	}
}

// RemoveCorporationInput represents the input for removing a managed corporation
type RemoveCorporationInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	CorporationID int64  `path:"corp_id" required:"true" minimum:"1" description:"Corporation ID"`
}

// ListManagedCorporationsInput represents the input for listing managed corporations
type ListManagedCorporationsInput struct {
	Authorization  string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie         string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	EnabledFilter  string `query:"enabled" description:"Filter by enabled status: 'true', 'false', or empty for all"`
	Page           int    `query:"page" minimum:"1" default:"1" description:"Page number"`
	Limit          int    `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Items per page"`
}

// GetManagedCorporationInput represents the input for getting a specific managed corporation
type GetManagedCorporationInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	CorporationID int64  `path:"corp_id" required:"true" minimum:"1" description:"Corporation ID"`
}

// BulkUpdateCorporationItem represents a corporation item in bulk update requests
type BulkUpdateCorporationItem struct {
	CorporationID int64  `json:"corporation_id" required:"true" description:"EVE Online corporation ID"`
	Name          string `json:"name" minLength:"1" maxLength:"100" required:"true" description:"Corporation name"`
	Enabled       bool   `json:"enabled" required:"true" description:"Whether the corporation should be enabled"`
	Position      *int   `json:"position" minimum:"1" description:"Display position (auto-assigned if not provided)"`
}

// ReorderCorporationItem represents a corporation item in reorder requests
type ReorderCorporationItem struct {
	CorporationID int64 `json:"corporation_id" required:"true" description:"Corporation ID"`
	Position      int   `json:"position" required:"true" minimum:"1" description:"New position"`
}

// BulkUpdateCorporationsInput represents the input for bulk updating corporations
type BulkUpdateCorporationsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Body          struct {
		Corporations []BulkUpdateCorporationItem `json:"corporations" required:"true" description:"List of corporations to update"`
	}
}

// ReorderCorporationsInput represents the input for reordering managed corporations
type ReorderCorporationsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Body          struct {
		CorporationOrders []ReorderCorporationItem `json:"corporation_orders" required:"true" description:"New ordering for corporations"`
	}
}

