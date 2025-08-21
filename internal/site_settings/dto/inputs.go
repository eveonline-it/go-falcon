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
		Value       interface{} `json:"value" description:"Setting value (can be string, number, boolean, or object)"`
		Type        *string     `json:"type" enum:"string,number,boolean,object" description:"Data type of the setting value"`
		Category    *string     `json:"category" maxLength:"50" description:"Setting category for organization"`
		Description *string     `json:"description" maxLength:"500" description:"Description of what this setting controls"`
		IsPublic    *bool       `json:"is_public" description:"Whether this setting can be read by non-admin users"`
		IsActive    *bool       `json:"is_active" description:"Whether this setting is active"`
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