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
	// No public endpoints needed - main /sitemap endpoint handles both authenticated and public access
}

// registerUserRoutes registers user-specific sitemap endpoints
func (r *Routes) registerUserRoutes(api huma.API, basePath string) {
	// Get sitemap (handles both authenticated and public access)
	huma.Register(api, huma.Operation{
		OperationID: "get-sitemap",
		Method:      "GET",
		Path:        basePath,
		Summary:     "Get sitemap",
		Description: "Returns routes and navigation. Shows personalized routes for authenticated users or public routes for unauthenticated users",
		Tags:        []string{"Sitemap"},
		// No security requirement - endpoint handles both authenticated and unauthenticated access
	}, func(ctx context.Context, input *dto.GetUserRoutesInput) (*dto.SitemapOutput, error) {
		// TODO: Implement group-based filtering
		// Check which groups the user belongs to and filter sitemap accordingly
		// For now, return ALL enabled routes for testing (regardless of type)

		sitemap, err := r.service.GetAllEnabledRoutes(ctx, input.IncludeDisabled, input.IncludeHidden)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get sitemap", err)
		}

		return &dto.SitemapOutput{Body: *sitemap}, nil
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

}
