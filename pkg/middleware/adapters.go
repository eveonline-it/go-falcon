package middleware

import (
	"context"

	"go-falcon/internal/auth/models"
)

// ModuleAdapter provides module-specific permission checking patterns
type ModuleAdapter struct {
	permissionMiddleware *PermissionMiddleware
}

// NewModuleAdapter creates a new module adapter
func NewModuleAdapter(permissionMiddleware *PermissionMiddleware) *ModuleAdapter {
	return &ModuleAdapter{
		permissionMiddleware: permissionMiddleware,
	}
}

// SitemapAdapter provides sitemap-specific permission methods
type SitemapAdapter struct {
	*ModuleAdapter
}

// NewSitemapAdapter creates a new sitemap adapter
func NewSitemapAdapter(permissionMiddleware *PermissionMiddleware) *SitemapAdapter {
	return &SitemapAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireSitemapView checks for sitemap view permissions
func (sa *SitemapAdapter) RequireSitemapView(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return sa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "sitemap:routes:view")
}

// RequireSitemapAdmin checks for sitemap admin permissions
func (sa *SitemapAdapter) RequireSitemapAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return sa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "sitemap:admin:full")
}

// RequireSitemapNavigation checks for sitemap navigation customization permissions
func (sa *SitemapAdapter) RequireSitemapNavigation(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return sa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "sitemap:navigation:customize")
}

// RequireAuth provides basic authentication for sitemap routes
func (sa *SitemapAdapter) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return sa.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// SchedulerAdapter provides scheduler-specific permission methods
type SchedulerAdapter struct {
	*ModuleAdapter
}

// NewSchedulerAdapter creates a new scheduler adapter
func NewSchedulerAdapter(permissionMiddleware *PermissionMiddleware) *SchedulerAdapter {
	return &SchedulerAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireSchedulerManagement checks for scheduler management permissions (any of the scheduler permissions)
func (sa *SchedulerAdapter) RequireSchedulerManagement(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	requiredPermissions := []string{
		"scheduler:tasks:read",
		"scheduler:tasks:create",
		"scheduler:tasks:update",
		"scheduler:tasks:delete",
		"scheduler:tasks:execute",
		"scheduler:tasks:control",
		"scheduler:system:manage",
	}

	return sa.permissionMiddleware.RequireAnyPermission(ctx, authHeader, cookieHeader, requiredPermissions)
}

// RequireTaskManagement checks for task management permissions (create, update, delete, execute, control)
func (sa *SchedulerAdapter) RequireTaskManagement(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	taskManagementPermissions := []string{
		"scheduler:tasks:create",
		"scheduler:tasks:update",
		"scheduler:tasks:delete",
		"scheduler:tasks:execute",
		"scheduler:tasks:control",
		"scheduler:system:manage",
	}

	return sa.permissionMiddleware.RequireAnyPermission(ctx, authHeader, cookieHeader, taskManagementPermissions)
}

// RequireSpecificPermission checks for a specific scheduler permission
func (sa *SchedulerAdapter) RequireSpecificPermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	return sa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, permissionID)
}

// GroupsAdapter provides groups-specific permission methods
type GroupsAdapter struct {
	*ModuleAdapter
}

// NewGroupsAdapter creates a new groups adapter
func NewGroupsAdapter(permissionMiddleware *PermissionMiddleware) *GroupsAdapter {
	return &GroupsAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireGroupManagement checks for group management permissions
func (ga *GroupsAdapter) RequireGroupManagement(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ga.permissionMiddleware.RequireAnyPermission(ctx, authHeader, cookieHeader, []string{
		"groups:management:full",
		"groups:memberships:manage",
	})
}

// RequireGroupPermissions checks for group permission management
func (ga *GroupsAdapter) RequireGroupPermissions(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ga.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "groups:permissions:manage")
}

// RequireGroupAccess checks for basic group access (fallback method for compatibility)
func (ga *GroupsAdapter) RequireGroupAccess(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// This is a compatibility method - just requires authentication
	return ga.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// UsersAdapter provides users-specific permission methods
type UsersAdapter struct {
	*ModuleAdapter
}

// NewUsersAdapter creates a new users adapter
func NewUsersAdapter(permissionMiddleware *PermissionMiddleware) *UsersAdapter {
	return &UsersAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireUserManagement checks for user management permissions
func (ua *UsersAdapter) RequireUserManagement(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ua.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "users:management:full")
}

// RequireProfileAccess checks for profile view permissions
func (ua *UsersAdapter) RequireProfileAccess(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ua.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "users:profiles:view")
}

// RequireUserAccess ensures the user can access user information (self or admin)
func (ua *UsersAdapter) RequireUserAccess(ctx context.Context, authHeader, cookieHeader, targetUserID string) (*models.AuthenticatedUser, error) {
	user, err := ua.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Users can always access their own data
	if user.UserID == targetUserID {
		return user, nil
	}

	// Check if user has admin permissions for accessing other users
	return ua.RequireUserManagement(ctx, authHeader, cookieHeader)
}

// AllianceAdapter provides alliance-specific permission methods
type AllianceAdapter struct {
	*ModuleAdapter
}

// NewAllianceAdapter creates a new alliance adapter
func NewAllianceAdapter(permissionMiddleware *PermissionMiddleware) *AllianceAdapter {
	return &AllianceAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireAllianceAdmin checks for alliance administration permissions
func (aa *AllianceAdapter) RequireAllianceAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return aa.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader) // Simple auth for now
}

// MigrationHelper provides utilities for migrating from module-specific middleware
type MigrationHelper struct {
	permissionMiddleware *PermissionMiddleware
}

// NewMigrationHelper creates a new migration helper
func NewMigrationHelper(permissionMiddleware *PermissionMiddleware) *MigrationHelper {
	return &MigrationHelper{
		permissionMiddleware: permissionMiddleware,
	}
}

// CreateSitemapAdapter creates a sitemap adapter for migration
func (mh *MigrationHelper) CreateSitemapAdapter() *SitemapAdapter {
	return NewSitemapAdapter(mh.permissionMiddleware)
}

// CreateSchedulerAdapter creates a scheduler adapter for migration
func (mh *MigrationHelper) CreateSchedulerAdapter() *SchedulerAdapter {
	return NewSchedulerAdapter(mh.permissionMiddleware)
}

// CreateGroupsAdapter creates a groups adapter for migration
func (mh *MigrationHelper) CreateGroupsAdapter() *GroupsAdapter {
	return NewGroupsAdapter(mh.permissionMiddleware)
}

// CreateUsersAdapter creates a users adapter for migration
func (mh *MigrationHelper) CreateUsersAdapter() *UsersAdapter {
	return NewUsersAdapter(mh.permissionMiddleware)
}

// CreateAllianceAdapter creates an alliance adapter for migration
func (mh *MigrationHelper) CreateAllianceAdapter() *AllianceAdapter {
	return NewAllianceAdapter(mh.permissionMiddleware)
}

// CreateCharacterAdapter creates a character adapter for migration
func (mh *MigrationHelper) CreateCharacterAdapter() *CharacterAdapter {
	return NewCharacterAdapter(mh.permissionMiddleware)
}

// CreateCorporationAdapter creates a corporation adapter for migration
func (mh *MigrationHelper) CreateCorporationAdapter() *CorporationAdapter {
	return NewCorporationAdapter(mh.permissionMiddleware)
}

// CreateSiteSettingsAdapter creates a site settings adapter for migration
func (mh *MigrationHelper) CreateSiteSettingsAdapter() *SiteSettingsAdapter {
	return NewSiteSettingsAdapter(mh.permissionMiddleware)
}

// CreateSDEAdminAdapter creates a SDE admin adapter for migration
func (mh *MigrationHelper) CreateSDEAdminAdapter() *SDEAdminAdapter {
	return NewSDEAdminAdapter(mh.permissionMiddleware)
}

// CharacterAdapter provides character-specific permission methods
type CharacterAdapter struct {
	*ModuleAdapter
}

// NewCharacterAdapter creates a new character adapter
func NewCharacterAdapter(permissionMiddleware *PermissionMiddleware) *CharacterAdapter {
	return &CharacterAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireAuth ensures the user is authenticated
func (ca *CharacterAdapter) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ca.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RequireCharacterAccess ensures the user has access to character data
func (ca *CharacterAdapter) RequireCharacterAccess(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// For now, any authenticated user can access character data
	return ca.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RequirePermission checks if the authenticated user has a specific permission
func (ca *CharacterAdapter) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	return ca.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, permissionID)
}

// CorporationAdapter provides corporation-specific permission methods
type CorporationAdapter struct {
	*ModuleAdapter
}

// NewCorporationAdapter creates a new corporation adapter
func NewCorporationAdapter(permissionMiddleware *PermissionMiddleware) *CorporationAdapter {
	return &CorporationAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireAuth ensures the user is authenticated
func (ca *CorporationAdapter) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ca.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RequireCorporationAccess ensures the user has access to corporation data
func (ca *CorporationAdapter) RequireCorporationAccess(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	// For now, any authenticated user can access corporation data
	return ca.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RequirePermission checks if the authenticated user has a specific permission
func (ca *CorporationAdapter) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	return ca.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, permissionID)
}

// SiteSettingsAdapter provides site settings-specific permission methods
type SiteSettingsAdapter struct {
	*ModuleAdapter
}

// NewSiteSettingsAdapter creates a new site settings adapter
func NewSiteSettingsAdapter(permissionMiddleware *PermissionMiddleware) *SiteSettingsAdapter {
	return &SiteSettingsAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireAuth ensures the user is authenticated
func (ssa *SiteSettingsAdapter) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ssa.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RequireSiteSettingsView checks for site settings view permissions
func (ssa *SiteSettingsAdapter) RequireSiteSettingsView(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ssa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "site_settings:settings:view")
}

// RequireSiteSettingsAdmin checks for site settings admin permissions
func (ssa *SiteSettingsAdapter) RequireSiteSettingsAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return ssa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, "site_settings:settings:manage")
}

// RequirePermission checks if the authenticated user has a specific permission
func (ssa *SiteSettingsAdapter) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	return ssa.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, permissionID)
}

// SDEAdminAdapter provides SDE admin-specific permission methods
type SDEAdminAdapter struct {
	*ModuleAdapter
}

// NewSDEAdminAdapter creates a new SDE admin adapter
func NewSDEAdminAdapter(permissionMiddleware *PermissionMiddleware) *SDEAdminAdapter {
	return &SDEAdminAdapter{
		ModuleAdapter: NewModuleAdapter(permissionMiddleware),
	}
}

// RequireSuperAdmin requires super administrator access for all SDE admin operations
func (saa *SDEAdminAdapter) RequireSuperAdmin(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return saa.permissionMiddleware.RequireSuperAdmin(ctx, authHeader, cookieHeader)
}

// RequireAuth ensures the user is authenticated (for status endpoints)
func (saa *SDEAdminAdapter) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return saa.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}
