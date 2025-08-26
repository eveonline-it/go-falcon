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
	Body RoutesResponse `json:"body" description:"Routes list"`
}

// RoutesResponse contains the routes response
type RoutesResponse struct {
	Routes []models.Route `json:"routes" description:"List of routes"`
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

// FolderOutput represents the output for a single folder
type FolderOutput struct {
	Body models.Route `json:"body" description:"Folder details"`
}

// CreateFolderOutput represents the response for folder creation
type CreateFolderOutput struct {
	Body CreateFolderResponse `json:"body" description:"Created folder details"`
}

// CreateFolderResponse contains the created folder
type CreateFolderResponse struct {
	Folder  models.Route `json:"folder" description:"Created folder"`
	Message string       `json:"message" description:"Success message"`
}

// UpdateFolderOutput represents the response for folder update
type UpdateFolderOutput struct {
	Body UpdateFolderResponse `json:"body" description:"Updated folder details"`
}

// UpdateFolderResponse contains the updated folder
type UpdateFolderResponse struct {
	Folder  models.Route `json:"folder" description:"Updated folder"`
	Message string       `json:"message" description:"Success message"`
}

// FolderChildrenOutput represents the output for folder children
type FolderChildrenOutput struct {
	Body FolderChildrenResponse `json:"body" description:"Folder children"`
}

// FolderChildrenResponse contains folder children with metadata
type FolderChildrenResponse struct {
	FolderID      string         `json:"folder_id" description:"Parent folder ID"`
	FolderName    string         `json:"folder_name" description:"Parent folder name"`
	FolderPath    string         `json:"folder_path" description:"Full folder path"`
	Children      []models.Route `json:"children" description:"Direct children"`
	TotalChildren int            `json:"total_children" description:"Total number of children"`
	Depth         int            `json:"depth" description:"Folder depth level"`
	HasSubfolders bool           `json:"has_subfolders" description:"Contains subfolders"`
}

// FolderStatsOutput represents the output for folder statistics
type FolderStatsOutput struct {
	Body models.FolderStats `json:"body" description:"Folder statistics"`
}

// MoveFolderOutput represents the response for folder move operations
type MoveFolderOutput struct {
	Body MoveFolderResponse `json:"body" description:"Move operation result"`
}

// MoveFolderResponse contains move operation results
type MoveFolderResponse struct {
	Message   string `json:"message" description:"Success message"`
	ItemMoved string `json:"item_moved" description:"Moved item ID"`
	OldParent string `json:"old_parent,omitempty" description:"Previous parent ID"`
	NewParent string `json:"new_parent,omitempty" description:"New parent ID"`
	NewPath   string `json:"new_path" description:"New folder path"`
}

// BulkMoveOutput represents the response for bulk move operations
type BulkMoveOutput struct {
	Body BulkMoveResponse `json:"body" description:"Bulk move results"`
}

// BulkMoveResponse contains bulk move operation results
type BulkMoveResponse struct {
	TargetFolder string   `json:"target_folder,omitempty" description:"Target folder ID"`
	ItemsMoved   []string `json:"items_moved" description:"Successfully moved item IDs"`
	ItemsFailed  []string `json:"items_failed,omitempty" description:"Failed item IDs"`
	TotalMoved   int      `json:"total_moved" description:"Number of items moved"`
	TotalFailed  int      `json:"total_failed" description:"Number of items failed"`
	Errors       []string `json:"errors,omitempty" description:"Error messages"`
	Message      string   `json:"message" description:"Operation summary"`
}
