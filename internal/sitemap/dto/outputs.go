package dto

import (
	"go-falcon/internal/sitemap/models"
	"time"
)

// RouteOutput represents the output for a single route
type RouteOutput struct {
	Body models.Route `json:"body" description:"Route details"`
}

// RoutesOutput represents the output for multiple routes
type RoutesOutput struct {
	Body RoutesResponse `json:"body" description:"Routes list with pagination"`
}

// RoutesResponse contains the paginated routes response
type RoutesResponse struct {
	Routes []models.Route `json:"routes" description:"List of routes"`
	Total  int64          `json:"total" description:"Total number of routes"`
	Page   int            `json:"page" description:"Current page"`
	Limit  int            `json:"limit" description:"Items per page"`
}

// SitemapOutput represents the user-specific sitemap response
type SitemapOutput struct {
	Body models.SitemapResponse `json:"body" description:"User sitemap with routes and navigation"`
}

// PublicSitemapOutput represents the public sitemap (for SEO/unauthenticated users)
type PublicSitemapOutput struct {
	Body PublicSitemapResponse `json:"body" description:"Public routes and navigation"`
}

// PublicSitemapResponse contains public routes only
type PublicSitemapResponse struct {
	Routes     []models.RouteConfig     `json:"routes" description:"Public routes"`
	Navigation []models.NavigationGroup `json:"navigation" description:"Public navigation"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body StatusResponse `json:"body" description:"Module status"`
}

// StatusResponse contains the module health status
type StatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message"`
}

// BulkUpdateOutput represents the response for bulk operations
type BulkUpdateOutput struct {
	Body BulkUpdateResponse `json:"body" description:"Bulk update results"`
}

// BulkUpdateResponse contains bulk update results
type BulkUpdateResponse struct {
	Updated int      `json:"updated" description:"Number of routes updated"`
	Failed  int      `json:"failed" description:"Number of routes failed to update"`
	Errors  []string `json:"errors,omitempty" description:"Error messages for failed updates"`
}

// RouteStatsOutput represents route usage statistics
type RouteStatsOutput struct {
	Body RouteStatsResponse `json:"body" description:"Route statistics"`
}

// RouteStatsResponse contains route usage statistics
type RouteStatsResponse struct {
	TotalRoutes      int64            `json:"total_routes" description:"Total number of routes"`
	EnabledRoutes    int64            `json:"enabled_routes" description:"Number of enabled routes"`
	DisabledRoutes   int64            `json:"disabled_routes" description:"Number of disabled routes"`
	PublicRoutes     int64            `json:"public_routes" description:"Number of public routes"`
	ProtectedRoutes  int64            `json:"protected_routes" description:"Number of protected routes"`
	RoutesByType     map[string]int64 `json:"routes_by_type" description:"Routes grouped by type"`
	RoutesByGroup    map[string]int64 `json:"routes_by_group" description:"Routes grouped by group"`
	RoutesByPosition map[string]int64 `json:"routes_by_position" description:"Routes grouped by navigation position"`
	LastUpdated      time.Time        `json:"last_updated" description:"Last route update time"`
}

// RouteTreeOutput represents hierarchical route structure
type RouteTreeOutput struct {
	Body RouteTreeResponse `json:"body" description:"Hierarchical route tree"`
}

// RouteTreeResponse contains the route tree structure
type RouteTreeResponse struct {
	Tree []RouteNode `json:"tree" description:"Route tree structure"`
}

// RouteNode represents a node in the route tree
type RouteNode struct {
	Route    models.Route `json:"route" description:"Route details"`
	Children []RouteNode  `json:"children,omitempty" description:"Child routes"`
}

// RouteAccessOutput represents route access check result
type RouteAccessOutput struct {
	Body RouteAccessResponse `json:"body" description:"Route access check result"`
}

// RouteAccessResponse contains route access information
type RouteAccessResponse struct {
	RouteID    string   `json:"route_id" description:"Route identifier"`
	Path       string   `json:"path" description:"Route path"`
	Accessible bool     `json:"accessible" description:"Whether user can access this route"`
	Reason     string   `json:"reason,omitempty" description:"Reason for access denial"`
	Missing    []string `json:"missing,omitempty" description:"Missing permissions or groups"`
}

// NavigationOutput represents the navigation structure
type NavigationOutput struct {
	Body NavigationResponse `json:"body" description:"Navigation structure"`
}

// NavigationResponse contains the navigation groups
type NavigationResponse struct {
	Navigation []models.NavigationGroup `json:"navigation" description:"Navigation groups"`
}

// CreateRouteOutput represents the response for route creation
type CreateRouteOutput struct {
	Body CreateRouteResponse `json:"body" description:"Created route details"`
}

// CreateRouteResponse contains the created route
type CreateRouteResponse struct {
	Route   models.Route `json:"route" description:"Created route"`
	Message string       `json:"message" description:"Success message"`
}

// UpdateRouteOutput represents the response for route update
type UpdateRouteOutput struct {
	Body UpdateRouteResponse `json:"body" description:"Updated route details"`
}

// UpdateRouteResponse contains the updated route
type UpdateRouteResponse struct {
	Route   models.Route `json:"route" description:"Updated route"`
	Message string       `json:"message" description:"Success message"`
}

// DeleteRouteOutput represents the response for route deletion
type DeleteRouteOutput struct {
	Body DeleteRouteResponse `json:"body" description:"Deletion result"`
}

// DeleteRouteResponse contains the deletion result
type DeleteRouteResponse struct {
	Message string `json:"message" description:"Success message"`
	Deleted int    `json:"deleted" description:"Number of routes deleted (including children)"`
}
