package dto

import (
	"go-falcon/internal/sitemap/models"
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
