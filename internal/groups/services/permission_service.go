package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"go-falcon/internal/groups/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PermissionService handles legacy group-based permission checking
type PermissionService struct {
	repository   *Repository
	groupService *GroupService
	mongodb      *database.MongoDB
	redis        *database.Redis
}

// NewPermissionService creates a new legacy permission service
func NewPermissionService(mongodb *database.MongoDB, redis *database.Redis, groupService *GroupService) *PermissionService {
	return &PermissionService{
		repository:   NewRepository(mongodb),
		groupService: groupService,
		mongodb:      mongodb,
		redis:        redis,
	}
}

// CheckPermission checks if a user has specific legacy permissions
func (ps *PermissionService) CheckPermission(ctx context.Context, characterID int, resource string, actions ...string) (*models.PermissionResult, error) {
	result := &models.PermissionResult{
		CharacterID: characterID,
		Allowed:     false,
		Groups:      []string{},
		Reason:      "Access denied",
	}

	// Get user's groups
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return result, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Check if user is in super_admin group - they have all permissions
	for _, group := range groups {
		if group.Name == "super_admin" {
			result.Allowed = true
			result.Groups = []string{"super_admin"}
			result.Reason = "Super admin access"
			return result, nil
		}
	}

	// Check each group's permissions for the resource and actions
	var matchingGroups []string
	
	for _, group := range groups {
		// Check if group has permission for this resource
		if resourcePerms, exists := group.Permissions[resource]; exists {
			// Check if group has all required actions
			hasAllActions := true
			for _, requiredAction := range actions {
				hasAction := false
				for _, allowedAction := range resourcePerms {
					if allowedAction == requiredAction || allowedAction == "admin" {
						hasAction = true
						break
					}
				}
				if !hasAction {
					hasAllActions = false
					break
				}
			}
			
			if hasAllActions {
				matchingGroups = append(matchingGroups, group.Name)
			}
		}
	}

	// If we found matching groups, access is allowed
	if len(matchingGroups) > 0 {
		result.Allowed = true
		result.Groups = matchingGroups
		result.Reason = fmt.Sprintf("Access granted via groups: %v", matchingGroups)
	}

	return result, nil
}

// CheckPermissionFromRequest extracts character ID from request and checks permission
func (ps *PermissionService) CheckPermissionFromRequest(r *http.Request, resource string, actions ...string) (*models.PermissionResult, error) {
	// Get character ID from request context (set by auth middleware)
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		return &models.PermissionResult{
			Allowed: false,
			Reason:  "Authentication required",
		}, fmt.Errorf("failed to get character ID: %w", err)
	}

	return ps.CheckPermission(r.Context(), characterID, resource, actions...)
}

// HasPermission is a convenience method that returns only the boolean result
func (ps *PermissionService) HasPermission(ctx context.Context, characterID int, resource string, actions ...string) (bool, error) {
	result, err := ps.CheckPermission(ctx, characterID, resource, actions...)
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// GetUserPermissions returns all permissions for a user based on their group memberships
func (ps *PermissionService) GetUserPermissions(ctx context.Context, characterID int) (map[string][]string, error) {
	// Get user's groups
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Aggregate all permissions from all groups
	allPermissions := make(map[string][]string)
	
	for _, group := range groups {
		for resource, actions := range group.Permissions {
			// Add actions to resource, avoiding duplicates
			existing := allPermissions[resource]
			for _, action := range actions {
				found := false
				for _, existingAction := range existing {
					if existingAction == action {
						found = true
						break
					}
				}
				if !found {
					existing = append(existing, action)
				}
			}
			allPermissions[resource] = existing
		}
	}

	return allPermissions, nil
}

// GetUserPermissionSummary returns a formatted summary of user permissions
func (ps *PermissionService) GetUserPermissionSummary(ctx context.Context, characterID int) (*models.UserPermissionSummary, error) {
	// Get user's groups
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Get aggregated permissions
	permissions, err := ps.GetUserPermissions(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user is super admin
	isSuperAdmin := false
	isAdmin := false
	var groupNames []string
	
	for _, group := range groups {
		groupNames = append(groupNames, group.Name)
		if group.Name == "super_admin" {
			isSuperAdmin = true
		}
		if group.Name == "administrators" {
			isAdmin = true
		}
	}

	return &models.UserPermissionSummary{
		CharacterID:   characterID,
		Groups:        groupNames,
		Permissions:   permissions,
		IsSuperAdmin:  isSuperAdmin,
		IsAdmin:       isAdmin,
		GroupCount:    len(groups),
		ResourceCount: len(permissions),
	}, nil
}

// Legacy Group Permission Constants
const (
	// Resource types for legacy permissions
	ResourceGroups        = "groups"
	ResourceAuth          = "auth"
	ResourceSDE           = "sde"
	ResourceScheduler     = "scheduler"
	ResourceDev           = "dev"
	ResourceNotifications = "notifications"
	ResourceUsers         = "users"

	// Action types
	ActionRead   = "read"
	ActionWrite  = "write"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
)

// Convenience methods for common permission checks

// CanReadGroups checks if user can read group information
func (ps *PermissionService) CanReadGroups(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceGroups, ActionRead)
}

// CanManageGroups checks if user can manage groups (write/delete)
func (ps *PermissionService) CanManageGroups(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceGroups, ActionWrite)
}

// CanAdminGroups checks if user has admin access to groups
func (ps *PermissionService) CanAdminGroups(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceGroups, ActionAdmin)
}

// CanReadAuth checks if user can read auth information
func (ps *PermissionService) CanReadAuth(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceAuth, ActionRead)
}

// CanManageAuth checks if user can manage auth
func (ps *PermissionService) CanManageAuth(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceAuth, ActionWrite)
}

// CanReadSDE checks if user can read SDE data
func (ps *PermissionService) CanReadSDE(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceSDE, ActionRead)
}

// CanManageSDE checks if user can manage SDE
func (ps *PermissionService) CanManageSDE(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceSDE, ActionWrite)
}

// CanReadScheduler checks if user can read scheduler information
func (ps *PermissionService) CanReadScheduler(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceScheduler, ActionRead)
}

// CanManageScheduler checks if user can manage scheduler
func (ps *PermissionService) CanManageScheduler(ctx context.Context, characterID int) (bool, error) {
	return ps.HasPermission(ctx, characterID, ResourceScheduler, ActionWrite)
}

// IsSuperAdmin checks if user is a super admin
func (ps *PermissionService) IsSuperAdmin(ctx context.Context, characterID int) (bool, error) {
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return false, err
	}

	for _, group := range groups {
		if group.Name == "super_admin" {
			return true, nil
		}
	}

	return false, nil
}

// IsSuperAdminFromRequest checks if the request user is a super admin
func (ps *PermissionService) IsSuperAdminFromRequest(r *http.Request) (bool, error) {
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		return false, err
	}

	return ps.IsSuperAdmin(r.Context(), characterID)
}

// IsAdmin checks if user is an administrator
func (ps *PermissionService) IsAdmin(ctx context.Context, characterID int) (bool, error) {
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return false, err
	}

	for _, group := range groups {
		if group.Name == "super_admin" || group.Name == "administrators" {
			return true, nil
		}
	}

	return false, nil
}

// IsAdminFromRequest checks if the request user is an administrator
func (ps *PermissionService) IsAdminFromRequest(r *http.Request) (bool, error) {
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		return false, err
	}

	return ps.IsAdmin(r.Context(), characterID)
}

// AssignDefaultGroups assigns default groups to a new user
func (ps *PermissionService) AssignDefaultGroups(ctx context.Context, characterID int) error {
	// Get all default groups
	defaultGroups, err := ps.groupService.GetDefaultGroups(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default groups: %w", err)
	}

	// Add user to each default group
	for _, group := range defaultGroups {
		// Check if membership already exists
		_, err := ps.repository.GetMembership(ctx, characterID, group.ID)
		if err == nil {
			// Membership already exists, skip
			continue
		}

		// Create membership
		membership := &models.GroupMembership{
			CharacterID:      characterID,
			GroupID:          group.ID,
			AssignmentSource: "auto_default",
			AssignmentMetadata: map[string]interface{}{
				"assigned_reason": "Default group assignment",
			},
		}

		if err := ps.repository.CreateMembership(ctx, membership); err != nil {
			slog.Error("Failed to assign default group", 
				slog.Int("character_id", characterID),
				slog.String("group", group.Name),
				slog.String("error", err.Error()))
			continue
		}

		slog.Info("Assigned default group", 
			slog.Int("character_id", characterID),
			slog.String("group", group.Name))
	}

	return nil
}

// ValidateGroupMemberships validates a user's group memberships based on current criteria
func (ps *PermissionService) ValidateGroupMemberships(ctx context.Context, characterID int) error {
	// This would integrate with EVE ESI to validate corporation/alliance memberships
	// For now, we'll implement basic validation logic
	
	memberships, err := ps.repository.GetUserMemberships(ctx, characterID)
	if err != nil {
		return fmt.Errorf("failed to get user memberships: %w", err)
	}

	for _, membership := range memberships {
		// Update validation status
		validationStatus := models.ValidationStatusValid
		
		// TODO: Add ESI validation logic here
		// - Check corporation membership for corporate groups
		// - Check alliance membership for alliance groups
		// - Validate other group criteria
		
		if err := ps.repository.UpdateMembershipValidation(ctx, characterID, membership.GroupID, validationStatus); err != nil {
			slog.Error("Failed to update membership validation", 
				slog.Int("character_id", characterID),
				slog.String("group_id", membership.GroupID.Hex()),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

// GetPermissionMatrix returns a complete permission matrix for debugging/admin purposes
func (ps *PermissionService) GetPermissionMatrix(ctx context.Context) (map[string]map[string][]string, error) {
	// Get all groups
	allGroups, _, err := ps.repository.ListGroups(ctx, bson.M{}, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	matrix := make(map[string]map[string][]string)
	
	for _, group := range allGroups {
		matrix[group.Name] = group.Permissions
	}

	return matrix, nil
}

// Legacy middleware helpers for backward compatibility

// RequirePermissionMiddleware creates middleware that requires specific legacy permissions
func (ps *PermissionService) RequirePermissionMiddleware(resource string, actions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := ps.CheckPermissionFromRequest(r, resource, actions...)
			if err != nil {
				handlers.ErrorResponse(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				handlers.ForbiddenResponse(w, result.Reason)
				return
			}

			// Add permission result to context
			ctx := r.Context()
			ctx = handlers.WithPermissionResult(ctx, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalPermissionMiddleware creates middleware that adds permission info to context without blocking
func (ps *PermissionService) OptionalPermissionMiddleware(resource string, actions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result, err := ps.CheckPermissionFromRequest(r, resource, actions...)
			if err != nil {
				// For optional checks, we don't fail the request
				result = &models.PermissionResult{
					Allowed: false,
					Reason:  "Permission check error",
				}
			}

			// Add permission result to context
			ctx := r.Context()
			ctx = handlers.WithPermissionResult(ctx, result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Migration helpers for transitioning to granular permissions

// GetResourcePermissions returns all permissions for a specific resource across all groups
func (ps *PermissionService) GetResourcePermissions(ctx context.Context, resource string) (map[string][]string, error) {
	allGroups, _, err := ps.repository.ListGroups(ctx, bson.M{}, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	resourcePerms := make(map[string][]string)
	
	for _, group := range allGroups {
		if perms, exists := group.Permissions[resource]; exists {
			resourcePerms[group.Name] = perms
		}
	}

	return resourcePerms, nil
}

// GetUsersByPermission returns all users who have a specific permission
func (ps *PermissionService) GetUsersByPermission(ctx context.Context, resource string, action string) ([]int, error) {
	// This is a potentially expensive operation - use with caution
	allGroups, _, err := ps.repository.ListGroups(ctx, bson.M{}, 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	var relevantGroupIDs []primitive.ObjectID
	
	// Find groups that have the required permission
	for _, group := range allGroups {
		if perms, exists := group.Permissions[resource]; exists {
			for _, perm := range perms {
				if perm == action || perm == "admin" {
					relevantGroupIDs = append(relevantGroupIDs, group.ID)
					break
				}
			}
		}
	}

	// Get all members of relevant groups
	var userIDs []int
	seen := make(map[int]bool)
	
	for _, groupID := range relevantGroupIDs {
		memberships, _, err := ps.repository.GetGroupMemberships(ctx, groupID, 1, 10000) // Large limit
		if err != nil {
			slog.Error("Failed to get group memberships", 
				slog.String("group_id", groupID.Hex()),
				slog.String("error", err.Error()))
			continue
		}
		
		for _, membership := range memberships {
			if !seen[membership.CharacterID] {
				userIDs = append(userIDs, membership.CharacterID)
				seen[membership.CharacterID] = true
			}
		}
	}

	return userIDs, nil
}