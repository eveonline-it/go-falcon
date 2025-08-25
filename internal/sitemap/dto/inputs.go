package dto

import (
	"go-falcon/internal/sitemap/models"
)

// CreateRouteInput represents the input for creating a new route
type CreateRouteInput struct {
	RouteID   string           `json:"route_id" required:"true" minLength:"3" maxLength:"100" description:"Unique route identifier"`
	Path      string           `json:"path" required:"true" minLength:"1" maxLength:"200" description:"Frontend path"`
	Component string           `json:"component" required:"true" minLength:"1" maxLength:"100" description:"React component name"`
	Name      string           `json:"name" required:"true" minLength:"1" maxLength:"100" description:"Display name"`
	Icon      *string          `json:"icon,omitempty" maxLength:"50" description:"Icon identifier"`
	Type      models.RouteType `json:"type" required:"true" enum:"public,auth,protected,admin" description:"Route type"`
	ParentID  *string          `json:"parent_id,omitempty" description:"Parent route ID for nested routes"`

	// Navigation
	NavPosition models.NavigationPosition `json:"nav_position" enum:"main,user,admin,footer,hidden" description:"Navigation position"`
	NavOrder    int                       `json:"nav_order" minimum:"0" maximum:"999" description:"Sort order in navigation"`
	ShowInNav   bool                      `json:"show_in_nav" description:"Show in navigation"`

	// Permissions
	RequiredPermissions []string `json:"required_permissions,omitempty" description:"Required permissions (AND logic)"`
	RequiredGroups      []string `json:"required_groups,omitempty" description:"Required groups (OR logic)"`

	// Metadata
	Title       string   `json:"title" required:"true" minLength:"1" maxLength:"100" description:"Page title"`
	Description *string  `json:"description,omitempty" maxLength:"500" description:"Page description"`
	Keywords    []string `json:"keywords,omitempty" description:"Search keywords"`
	Group       *string  `json:"group,omitempty" description:"Route group"`

	// Feature flags
	FeatureFlags []string `json:"feature_flags,omitempty" description:"Required feature flags"`
	IsEnabled    bool     `json:"is_enabled" default:"true" description:"Route enabled status"`

	// React-specific
	Props    map[string]interface{} `json:"props,omitempty" description:"Default props for component"`
	LazyLoad bool                   `json:"lazy_load" default:"true" description:"Enable code splitting"`
	Exact    bool                   `json:"exact,omitempty" description:"Exact path matching"`
	NewTab   bool                   `json:"newtab,omitempty" description:"Open in new tab"`

	// Badge
	BadgeType *string `json:"badge_type,omitempty" enum:"success,warning,danger,info,primary,secondary" description:"Badge type"`
	BadgeText *string `json:"badge_text,omitempty" maxLength:"20" description:"Badge text"`
}

// UpdateRouteInput represents the input for updating a route
type UpdateRouteInput struct {
	Path      *string           `json:"path,omitempty" minLength:"1" maxLength:"200" description:"Frontend path"`
	Component *string           `json:"component,omitempty" minLength:"1" maxLength:"100" description:"React component name"`
	Name      *string           `json:"name,omitempty" minLength:"1" maxLength:"100" description:"Display name"`
	Icon      *string           `json:"icon,omitempty" maxLength:"50" description:"Icon identifier"`
	Type      *models.RouteType `json:"type,omitempty" enum:"public,auth,protected,admin" description:"Route type"`
	ParentID  *string           `json:"parent_id,omitempty" description:"Parent route ID"`

	// Navigation
	NavPosition *models.NavigationPosition `json:"nav_position,omitempty" enum:"main,user,admin,footer,hidden" description:"Navigation position"`
	NavOrder    *int                       `json:"nav_order,omitempty" minimum:"0" maximum:"999" description:"Sort order"`
	ShowInNav   *bool                      `json:"show_in_nav,omitempty" description:"Show in navigation"`

	// Permissions
	RequiredPermissions []string `json:"required_permissions,omitempty" description:"Required permissions"`
	RequiredGroups      []string `json:"required_groups,omitempty" description:"Required groups"`

	// Metadata
	Title       *string  `json:"title,omitempty" minLength:"1" maxLength:"100" description:"Page title"`
	Description *string  `json:"description,omitempty" maxLength:"500" description:"Page description"`
	Keywords    []string `json:"keywords,omitempty" description:"Search keywords"`
	Group       *string  `json:"group,omitempty" description:"Route group"`

	// Feature flags
	FeatureFlags []string `json:"feature_flags,omitempty" description:"Required feature flags"`
	IsEnabled    *bool    `json:"is_enabled,omitempty" description:"Route enabled status"`

	// React-specific
	Props    map[string]interface{} `json:"props,omitempty" description:"Default props"`
	LazyLoad *bool                  `json:"lazy_load,omitempty" description:"Code splitting"`
	Exact    *bool                  `json:"exact,omitempty" description:"Exact matching"`
	NewTab   *bool                  `json:"newtab,omitempty" description:"Open in new tab"`

	// Badge
	BadgeType *string `json:"badge_type,omitempty" enum:"success,warning,danger,info,primary,secondary" description:"Badge type"`
	BadgeText *string `json:"badge_text,omitempty" maxLength:"20" description:"Badge text"`
}

// ListRoutesInput represents the input for listing routes
type ListRoutesInput struct {
	Type        string `query:"type" enum:"public,auth,protected,admin,all" default:"all" description:"Filter by route type"`
	Group       string `query:"group" description:"Filter by group"`
	IsEnabled   string `query:"is_enabled" enum:"true,false,all" default:"all" description:"Filter by enabled status"`
	ShowInNav   string `query:"show_in_nav" enum:"true,false,all" default:"all" description:"Filter by navigation visibility"`
	NavPosition string `query:"nav_position" enum:"main,user,admin,footer,hidden,all" default:"all" description:"Filter by navigation position"`
	Page        int    `query:"page" minimum:"1" default:"1" description:"Page number"`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Items per page"`
}

// BulkUpdateOrderInput represents the input for updating navigation order
type BulkUpdateOrderInput struct {
	Updates []OrderUpdate `json:"updates" required:"true" minItems:"1" description:"Order updates"`
}

// OrderUpdate represents a single order update
type OrderUpdate struct {
	RouteID  string `json:"route_id" required:"true" description:"Route ID"`
	NavOrder int    `json:"nav_order" required:"true" minimum:"0" maximum:"999" description:"New order"`
}

// GetUserRoutesInput represents the input for getting user-specific routes
type GetUserRoutesInput struct {
	IncludeDisabled bool `query:"include_disabled" default:"false" description:"Include disabled routes"`
	IncludeHidden   bool `query:"include_hidden" default:"false" description:"Include hidden navigation items"`
}
