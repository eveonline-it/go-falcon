package routes

import (
	"context"

	"go-falcon/internal/sitemap/dto"
	"go-falcon/internal/sitemap/services"
	"go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// Routes handles sitemap route definitions
type Routes struct {
	service        *services.Service
	sitemapAdapter *middleware.SitemapAdapter
}

// NewRoutes creates a new routes instance
func NewRoutes(service *services.Service, sitemapAdapter *middleware.SitemapAdapter) *Routes {
	return &Routes{
		service:        service,
		sitemapAdapter: sitemapAdapter,
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
		Tags:        []string{"Sitemap / User"},
		// Optional security - endpoint handles both authenticated and unauthenticated access
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetUserRoutesInput) (*dto.SitemapOutput, error) {
		// Try to authenticate user (optional) - for future personalization
		if input.Authorization != "" || input.Cookie != "" {
			// Only attempt authentication if auth headers are provided
			_, err := r.sitemapAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
			if err == nil {
				// User is authenticated - in future this can be used for personalization
				// For now, just continue with normal processing
			}
			// If authentication fails, continue as unauthenticated user (don't return error)
		}

		// Get routes with folder support
		sitemap, err := r.service.GetUserRoutesWithFolders(ctx, input)
		if err != nil {
			// Fallback to old method if new method not available yet
			sitemap, err = r.service.GetAllEnabledRoutes(ctx, input.IncludeDisabled, input.IncludeHidden)
			if err != nil {
				return nil, huma.Error500InternalServerError("Failed to get sitemap", err)
			}
		}

		// TODO: Filter routes based on user permissions and groups if authenticated
		// For now, return all enabled routes

		return &dto.SitemapOutput{Body: *sitemap}, nil
	})

	// TODO: Implement route access check endpoint when CheckRouteAccess service method is available
	/*
		// Check route access for specific route (authenticated users only)
		huma.Register(api, huma.Operation{
			OperationID: "check-route-access",
			Method:      "GET",
			Path:        basePath + "/access/{route_id}",
			Summary:     "Check route access",
			Description: "Check if current user can access a specific route. Requires authentication.",
			Tags:        []string{"Sitemap / User"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
				{"cookieAuth": {}},
			},
		}, func(ctx context.Context, input *struct {
			Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
			Cookie        string `header:"Cookie" doc:"falcon_auth_token cookie for authentication"`
			RouteID       string `path:"route_id" description:"Route ID to check access for"`
		}) (*dto.RouteAccessOutput, error) {
			// Require authentication for this endpoint
			_, err := r.sitemapAdapter.RequireAuth(ctx, input.Authorization, input.Cookie)
			if err != nil {
				return nil, err
			}

			// Check route access
			access, err := r.service.CheckRouteAccess(ctx, input.RouteID)
			if err != nil {
				return nil, huma.Error404NotFound("Route not found", err)
			}

			return &dto.RouteAccessOutput{Body: *access}, nil
		})
	*/

}

// registerAdminRoutes registers admin sitemap management endpoints
func (r *Routes) registerAdminRoutes(api huma.API, basePath string) {
	adminBasePath := "/admin" + basePath

	// List all routes (admin) - requires sitemap:routes:view permission
	huma.Register(api, huma.Operation{
		OperationID: "list-routes",
		Method:      "GET",
		Path:        adminBasePath,
		Summary:     "List all routes",
		Description: "Returns list of all routes with filtering options. Requires sitemap:routes:view permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.ListRoutesInput) (*dto.RoutesOutput, error) {
		// Check permission for viewing routes
		_, err := r.sitemapAdapter.RequireSitemapView(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		routes, err := r.service.GetRoutes(ctx, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get routes", err)
		}

		response := dto.RoutesResponse{
			Routes: routes,
		}

		return &dto.RoutesOutput{Body: response}, nil
	})

	// Get single route (admin) - requires sitemap:routes:view permission
	huma.Register(api, huma.Operation{
		OperationID: "get-route",
		Method:      "GET",
		Path:        adminBasePath + "/{id}",
		Summary:     "Get single route",
		Description: "Returns details of a specific route. Requires sitemap:routes:view permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetRouteInput) (*dto.RouteOutput, error) {
		// Check permission for viewing routes
		_, err := r.sitemapAdapter.RequireSitemapView(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		route, err := r.service.GetRouteByID(ctx, input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Route not found", err)
		}

		return &dto.RouteOutput{Body: *route}, nil
	})

	// Create new route (admin) - requires sitemap:admin:manage permission
	huma.Register(api, huma.Operation{
		OperationID: "create-route",
		Method:      "POST",
		Path:        adminBasePath,
		Summary:     "Create new route",
		Description: "Creates a new route configuration. Requires sitemap:admin:manage permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.CreateRouteInput) (*dto.CreateRouteOutput, error) {
		// Check permission for managing routes
		_, err := r.sitemapAdapter.RequireSitemapAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

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

	// Update route (admin) - requires sitemap:admin:manage permission
	huma.Register(api, huma.Operation{
		OperationID: "update-route",
		Method:      "PUT",
		Path:        adminBasePath + "/{id}",
		Summary:     "Update route",
		Description: "Updates an existing route configuration. Requires sitemap:admin:manage permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.UpdateRouteInput) (*dto.UpdateRouteOutput, error) {
		// Check permission for managing routes
		_, err := r.sitemapAdapter.RequireSitemapAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		route, err := r.service.UpdateRoute(ctx, input.ID, input)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to update route", err)
		}

		response := dto.UpdateRouteResponse{
			Route:   *route,
			Message: "Route updated successfully",
		}

		return &dto.UpdateRouteOutput{Body: response}, nil
	})

	// Delete route (admin) - requires sitemap:admin:manage permission
	huma.Register(api, huma.Operation{
		OperationID: "delete-route",
		Method:      "DELETE",
		Path:        adminBasePath + "/{id}",
		Summary:     "Delete route",
		Description: "Deletes a route and all its children. Requires sitemap:admin:manage permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.DeleteRouteInput) (*dto.DeleteRouteOutput, error) {
		// Check permission for managing routes
		_, err := r.sitemapAdapter.RequireSitemapAdmin(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

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

	// Bulk update navigation order (admin) - requires sitemap:navigation:customize permission
	huma.Register(api, huma.Operation{
		OperationID: "bulk-update-order",
		Method:      "POST",
		Path:        adminBasePath + "/reorder",
		Summary:     "Bulk update navigation order",
		Description: "Updates navigation order for multiple routes. Requires sitemap:navigation:customize permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.BulkUpdateOrderInput) (*dto.BulkUpdateOutput, error) {
		// Check permission for customizing navigation
		_, err := r.sitemapAdapter.RequireSitemapNavigation(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		updated, failed, errors := r.service.BulkUpdateOrder(ctx, input.Body.Updates)

		response := dto.BulkUpdateResponse{
			Updated: updated,
			Failed:  failed,
			Errors:  errors,
		}

		return &dto.BulkUpdateOutput{Body: response}, nil
	})

	// Status endpoint (public, no auth required)
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

	// Get sitemap statistics (admin) - requires sitemap:routes:view permission
	huma.Register(api, huma.Operation{
		OperationID: "get-sitemap-stats",
		Method:      "GET",
		Path:        adminBasePath + "/stats",
		Summary:     "Get sitemap statistics",
		Description: "Returns comprehensive sitemap usage statistics. Requires sitemap:routes:view permission.",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"bearerAuth": {}},
			{"cookieAuth": {}},
		},
	}, func(ctx context.Context, input *dto.GetStatsInput) (*dto.FolderStatsOutput, error) {
		// Check permission for viewing routes
		_, err := r.sitemapAdapter.RequireSitemapView(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, err
		}

		stats, err := r.service.GetFolderStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get stats", err)
		}

		return &dto.FolderStatsOutput{Body: *stats}, nil
	})

	// Folder management endpoints
	r.registerFolderRoutes(api, adminBasePath)
}

// registerFolderRoutes registers folder-specific endpoints
func (r *Routes) registerFolderRoutes(api huma.API, basePath string) {
	// Create folder
	huma.Register(api, huma.Operation{
		OperationID: "create-folder",
		Method:      "POST",
		Path:        basePath + "/folders",
		Summary:     "Create new folder",
		Description: "Creates a new folder container for organizing routes",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.CreateFolderInput) (*dto.CreateFolderOutput, error) {
		// TODO: Add proper admin authentication check
		folder, err := r.service.CreateFolder(ctx, input)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to create folder", err)
		}

		response := dto.CreateFolderResponse{
			Folder:  *folder,
			Message: "Folder created successfully",
		}

		return &dto.CreateFolderOutput{Body: response}, nil
	})

	// Update folder
	huma.Register(api, huma.Operation{
		OperationID: "update-folder",
		Method:      "PUT",
		Path:        basePath + "/folders/{folder_id}",
		Summary:     "Update folder",
		Description: "Updates an existing folder configuration",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		FolderID string                `path:"folder_id" description:"Folder ID"`
		Body     dto.UpdateFolderInput `json:"body"`
	}) (*dto.UpdateFolderOutput, error) {
		// TODO: Add proper admin authentication check
		folder, err := r.service.UpdateFolder(ctx, input.FolderID, &input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to update folder", err)
		}

		response := dto.UpdateFolderResponse{
			Folder:  *folder,
			Message: "Folder updated successfully",
		}

		return &dto.UpdateFolderOutput{Body: response}, nil
	})

	// Move item to folder
	huma.Register(api, huma.Operation{
		OperationID: "move-to-folder",
		Method:      "POST",
		Path:        basePath + "/move/{item_id}",
		Summary:     "Move item to folder",
		Description: "Moves a route or folder to a different parent folder",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct {
		ItemID string              `path:"item_id" description:"Route or folder ID to move"`
		Body   dto.MoveFolderInput `json:"body"`
	}) (*dto.MoveFolderOutput, error) {
		// TODO: Add proper admin authentication check
		result, err := r.service.MoveToFolder(ctx, input.ItemID, &input.Body)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to move item", err)
		}

		return &dto.MoveFolderOutput{Body: *result}, nil
	})

	// Get folder children
	huma.Register(api, huma.Operation{
		OperationID: "get-folder-children",
		Method:      "GET",
		Path:        basePath + "/folders/{folder_id}/children",
		Summary:     "Get folder children",
		Description: "Returns the children of a specific folder",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.FolderChildrenInput) (*dto.FolderChildrenOutput, error) {
		// TODO: Add proper admin authentication check
		children, err := r.service.GetFolderChildren(ctx, input)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to get folder children", err)
		}

		return &dto.FolderChildrenOutput{Body: *children}, nil
	})

	// Bulk move items
	huma.Register(api, huma.Operation{
		OperationID: "bulk-move-items",
		Method:      "POST",
		Path:        basePath + "/bulk-move",
		Summary:     "Bulk move items",
		Description: "Moves multiple routes/folders to a target folder",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *dto.BulkMoveInput) (*dto.BulkMoveOutput, error) {
		// TODO: Add proper admin authentication check
		result, err := r.service.BulkMove(ctx, input)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to bulk move items", err)
		}

		return &dto.BulkMoveOutput{Body: *result}, nil
	})

	// Get folder statistics
	huma.Register(api, huma.Operation{
		OperationID: "get-folder-stats",
		Method:      "GET",
		Path:        basePath + "/folders/stats",
		Summary:     "Get folder statistics",
		Description: "Returns folder usage statistics and metrics",
		Tags:        []string{"Sitemap / Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *struct{}) (*dto.FolderStatsOutput, error) {
		// TODO: Add proper admin authentication check
		stats, err := r.service.GetFolderStats(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get folder stats", err)
		}

		return &dto.FolderStatsOutput{Body: *stats}, nil
	})
}
