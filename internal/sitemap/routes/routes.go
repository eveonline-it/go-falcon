package routes

import (
	"context"

	"go-falcon/internal/sitemap/dto"
	"go-falcon/internal/sitemap/services"

	"github.com/danielgtaylor/huma/v2"
)

// Routes handles sitemap route definitions
type Routes struct {
	service *services.Service
}

// NewRoutes creates a new routes instance
func NewRoutes(service *services.Service) *Routes {
	return &Routes{
		service: service,
	}
}

// RegisterUnifiedRoutes registers all sitemap routes with Huma v2
func (r *Routes) RegisterUnifiedRoutes(api huma.API, basePath string) {
	// Public endpoints (no authentication required)
	r.registerPublicRoutes(api, basePath)

	// User endpoints (authentication required) - simplified for now
	r.registerUserRoutes(api, basePath)

	// Admin endpoints (admin permissions required) - simplified for now
	r.registerAdminRoutes(api, basePath)
}

// registerPublicRoutes registers public sitemap endpoints
func (r *Routes) registerPublicRoutes(api huma.API, basePath string) {
	// Get public sitemap (for unauthenticated users, SEO)
	huma.Register(api, huma.Operation{
		OperationID: "get-public-sitemap",
		Method:      "GET",
		Path:        basePath + "/public",
		Summary:     "Get public sitemap",
		Description: "Returns public routes and navigation for unauthenticated users",
		Tags:        []string{"Sitemap"},
	}, func(ctx context.Context, input *struct{}) (*dto.PublicSitemapOutput, error) {
		sitemap, err := r.service.GetPublicSitemap(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get public sitemap", err)
		}

		return &dto.PublicSitemapOutput{Body: *sitemap}, nil
	})

	// Module status endpoint (public)
	huma.Register(api, huma.Operation{
		OperationID: "get-sitemap-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get sitemap module status",
		Description: "Returns the health status of the sitemap module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := r.service.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})
}

// registerUserRoutes registers user-specific sitemap endpoints
func (r *Routes) registerUserRoutes(api huma.API, basePath string) {
	// Get user sitemap (personalized routes and navigation)
	// Note: For now, this returns all routes. Authentication integration will be added later.
	huma.Register(api, huma.Operation{
		OperationID: "get-user-sitemap",
		Method:      "GET",
		Path:        basePath,
		Summary:     "Get user sitemap",
		Description: "Returns personalized routes and navigation based on user permissions",
		Tags:        []string{"Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetUserRoutesInput) (*dto.SitemapOutput, error) {
		// TODO: Add proper authentication integration
		// For now, return a basic response with dummy user ID
		sitemap, err := r.service.GetUserSitemap(
			ctx,
			"dummy-user-id",
			123456789, // dummy character ID
			input.IncludeDisabled,
			input.IncludeHidden,
		)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user sitemap", err)
		}

		return &dto.SitemapOutput{Body: *sitemap}, nil
	})

	// Check route access for specific route
	huma.Register(api, huma.Operation{
		OperationID: "check-route-access",
		Method:      "GET",
		Path:        basePath + "/access/{route_id}",
		Summary:     "Check route access",
		Description: "Check if current user can access a specific route",
		Tags:        []string{"Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		RouteID string `path:"route_id" description:"Route ID to check"`
	}) (*dto.RouteAccessOutput, error) {
		// TODO: Add proper authentication integration
		access, err := r.service.CheckRouteAccess(ctx, input.RouteID, 123456789) // dummy character ID
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to check route access", err)
		}

		return &dto.RouteAccessOutput{Body: *access}, nil
	})
}

// registerAdminRoutes registers admin sitemap management endpoints
func (r *Routes) registerAdminRoutes(api huma.API, basePath string) {
	adminBasePath := "/admin" + basePath

	// List all routes (admin)
	huma.Register(api, huma.Operation{
		OperationID: "list-routes",
		Method:      "GET",
		Path:        adminBasePath,
		Summary:     "List all routes",
		Description: "Returns paginated list of all routes with filtering options",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.ListRoutesInput) (*dto.RoutesOutput, error) {
		// TODO: Add proper admin authentication check
		routes, total, err := r.service.GetRoutes(ctx, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get routes", err)
		}

		response := dto.RoutesResponse{
			Routes: routes,
			Total:  total,
			Page:   input.Page,
			Limit:  input.Limit,
		}

		return &dto.RoutesOutput{Body: response}, nil
	})

	// Get single route (admin)
	huma.Register(api, huma.Operation{
		OperationID: "get-route",
		Method:      "GET",
		Path:        adminBasePath + "/{id}",
		Summary:     "Get single route",
		Description: "Returns details of a specific route",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id" description:"Route ID or MongoDB ObjectID"`
	}) (*dto.RouteOutput, error) {
		// TODO: Add proper admin authentication check
		route, err := r.service.GetRouteByID(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Route not found", err)
		}

		return &dto.RouteOutput{Body: *route}, nil
	})

	// Create new route (admin)
	huma.Register(api, huma.Operation{
		OperationID: "create-route",
		Method:      "POST",
		Path:        adminBasePath,
		Summary:     "Create new route",
		Description: "Creates a new route configuration",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.CreateRouteInput) (*dto.CreateRouteOutput, error) {
		// TODO: Add proper admin authentication check
		route, err := r.service.CreateRoute(ctx, input)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to create route", err)
		}

		response := dto.CreateRouteResponse{
			Route:   *route,
			Message: "Route created successfully",
		}

		return &dto.CreateRouteOutput{Body: response}, nil
	})

	// Update route (admin)
	huma.Register(api, huma.Operation{
		OperationID: "update-route",
		Method:      "PUT",
		Path:        adminBasePath + "/{id}",
		Summary:     "Update route",
		Description: "Updates an existing route configuration",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		ID   string               `path:"id" description:"Route ID"`
		Body dto.UpdateRouteInput `json:"body"`
	}) (*dto.UpdateRouteOutput, error) {
		// TODO: Add proper admin authentication check
		route, err := r.service.UpdateRoute(ctx, input.ID, &input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to update route", err)
		}

		response := dto.UpdateRouteResponse{
			Route:   *route,
			Message: "Route updated successfully",
		}

		return &dto.UpdateRouteOutput{Body: response}, nil
	})

	// Delete route (admin)
	huma.Register(api, huma.Operation{
		OperationID: "delete-route",
		Method:      "DELETE",
		Path:        adminBasePath + "/{id}",
		Summary:     "Delete route",
		Description: "Deletes a route and all its children",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id" description:"Route ID"`
	}) (*dto.DeleteRouteOutput, error) {
		// TODO: Add proper admin authentication check
		deleted, err := r.service.DeleteRoute(ctx, input.ID)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to delete route", err)
		}

		response := dto.DeleteRouteResponse{
			Message: "Route deleted successfully",
			Deleted: deleted,
		}

		return &dto.DeleteRouteOutput{Body: response}, nil
	})

	// Bulk update navigation order (admin)
	huma.Register(api, huma.Operation{
		OperationID: "bulk-update-order",
		Method:      "POST",
		Path:        adminBasePath + "/reorder",
		Summary:     "Bulk update navigation order",
		Description: "Updates navigation order for multiple routes",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.BulkUpdateOrderInput) (*dto.BulkUpdateOutput, error) {
		// TODO: Add proper admin authentication check
		updated, failed, errors := r.service.BulkUpdateOrder(ctx, input.Updates)

		response := dto.BulkUpdateResponse{
			Updated: updated,
			Failed:  failed,
			Errors:  errors,
		}

		return &dto.BulkUpdateOutput{Body: response}, nil
	})

	// Get route statistics (admin)
	huma.Register(api, huma.Operation{
		OperationID: "get-route-stats",
		Method:      "GET",
		Path:        adminBasePath + "/stats",
		Summary:     "Get route statistics",
		Description: "Returns statistics about routes in the system",
		Tags:        []string{"Admin", "Sitemap"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct{}) (*dto.RouteStatsOutput, error) {
		// TODO: Add proper admin authentication check
		stats, err := r.service.GetRouteStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get route statistics", err)
		}

		return &dto.RouteStatsOutput{Body: *stats}, nil
	})
}
