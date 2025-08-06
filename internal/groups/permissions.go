package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
)

// PermissionService handles permission checking and caching
type PermissionService struct {
	mongodb      *database.MongoDB
	redis        *database.Redis
	groupService *GroupService
	cacheTTL     time.Duration
}

func NewPermissionService(mongodb *database.MongoDB, redis *database.Redis, groupService *GroupService) *PermissionService {
	cacheTTL := time.Duration(config.GetEnvInt("GROUPS_CACHE_TTL", 300)) * time.Second
	
	return &PermissionService{
		mongodb:      mongodb,
		redis:        redis,
		groupService: groupService,
		cacheTTL:     cacheTTL,
	}
}

// CheckPermission checks if a user has permission to perform an action on a resource
func (ps *PermissionService) CheckPermission(ctx context.Context, characterID int, resource, action string) (bool, []string, error) {
	// Get user's permission matrix
	permissions, err := ps.GetUserPermissions(ctx, characterID)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user has permission
	allowed := ps.hasPermission(permissions, resource, action)
	
	return allowed, permissions.Groups, nil
}

// GetUserPermissions retrieves the complete permission matrix for a user
func (ps *PermissionService) GetUserPermissions(ctx context.Context, characterID int) (*UserPermissionMatrix, error) {
	// Try to get from cache first
	if ps.redis != nil {
		cacheKey := fmt.Sprintf("user_permissions:%d", characterID)
		cached, err := ps.redis.Client.Get(ctx, cacheKey).Result()
		if err == nil {
			var permissions UserPermissionMatrix
			if err := json.Unmarshal([]byte(cached), &permissions); err == nil {
				return &permissions, nil
			}
		}
	}

	// Get user's groups
	var groups []Group
	var err error
	
	if characterID == 0 {
		// Guest user - only guest permissions
		guestGroup, err := ps.groupService.GetGroupByName(ctx, "guest")
		if err != nil {
			// If guest group doesn't exist, create minimal permissions
			return &UserPermissionMatrix{
				CharacterID: 0,
				Groups:      []string{"guest"},
				Permissions: map[string]map[string]bool{
					"public": {"read": true},
				},
				IsGuest: true,
			}, nil
		}
		groups = []Group{*guestGroup}
	} else {
		groups, err = ps.groupService.GetUserGroups(ctx, characterID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user groups: %w", err)
		}
	}

	// Build permission matrix
	permissions := ps.buildPermissionMatrix(characterID, groups)

	// Cache the result
	if ps.redis != nil {
		cacheKey := fmt.Sprintf("user_permissions:%d", characterID)
		data, err := json.Marshal(permissions)
		if err == nil {
			ps.redis.Client.Set(ctx, cacheKey, data, ps.cacheTTL)
		}
	}

	return permissions, nil
}

// buildPermissionMatrix creates a comprehensive permission matrix from user groups
func (ps *PermissionService) buildPermissionMatrix(characterID int, groups []Group) *UserPermissionMatrix {
	matrix := &UserPermissionMatrix{
		CharacterID: characterID,
		Groups:      make([]string, len(groups)),
		Permissions: make(map[string]map[string]bool),
		IsGuest:     characterID == 0,
	}

	// Collect group names
	for i, group := range groups {
		matrix.Groups[i] = group.Name
	}

	// Aggregate permissions from all groups
	for _, group := range groups {
		for resource, actions := range group.Permissions {
			if matrix.Permissions[resource] == nil {
				matrix.Permissions[resource] = make(map[string]bool)
			}

			for _, action := range actions {
				// Handle wildcard permissions
				if resource == "*" && action == "*" {
					// Super admin - grant all permissions
					ps.grantAllPermissions(matrix)
					return matrix
				}
				
				if action == "*" {
					// Grant all actions on this resource
					ps.grantAllActionsOnResource(matrix, resource)
				} else {
					matrix.Permissions[resource][action] = true
				}
			}
		}
	}

	return matrix
}

// grantAllPermissions grants all possible permissions (super admin)
func (ps *PermissionService) grantAllPermissions(matrix *UserPermissionMatrix) {
	resources := []string{"public", "user", "profile", "corporation", "alliance", "groups", "system", "admin", "scheduler", "notifications", "dev", "*"}
	actions := []string{"read", "write", "delete", "admin", "*"}

	for _, resource := range resources {
		if matrix.Permissions[resource] == nil {
			matrix.Permissions[resource] = make(map[string]bool)
		}
		for _, action := range actions {
			matrix.Permissions[resource][action] = true
		}
	}
	
	// Also add the wildcard permission for ultimate access
	if matrix.Permissions["*"] == nil {
		matrix.Permissions["*"] = make(map[string]bool)
	}
	matrix.Permissions["*"]["*"] = true
}

// grantAllActionsOnResource grants all actions on a specific resource
func (ps *PermissionService) grantAllActionsOnResource(matrix *UserPermissionMatrix, resource string) {
	actions := []string{"read", "write", "delete", "admin"}
	
	if matrix.Permissions[resource] == nil {
		matrix.Permissions[resource] = make(map[string]bool)
	}
	
	for _, action := range actions {
		matrix.Permissions[resource][action] = true
	}
}

// hasPermission checks if a permission matrix allows a specific action on a resource
func (ps *PermissionService) hasPermission(permissions *UserPermissionMatrix, resource, action string) bool {
	// Check exact resource match
	if resourcePerms, exists := permissions.Permissions[resource]; exists {
		if allowed, exists := resourcePerms[action]; exists && allowed {
			return true
		}
	}

	// Check wildcard resource (super admin)
	if resourcePerms, exists := permissions.Permissions["*"]; exists {
		if allowed, exists := resourcePerms["*"]; exists && allowed {
			return true
		}
		if allowed, exists := resourcePerms[action]; exists && allowed {
			return true
		}
	}

	return false
}

// InvalidateUserPermissions removes cached permissions for a user
func (ps *PermissionService) InvalidateUserPermissions(ctx context.Context, characterID int) error {
	if ps.redis == nil {
		return nil // No caching enabled
	}

	cacheKey := fmt.Sprintf("user_permissions:%d", characterID)
	err := ps.redis.Client.Del(ctx, cacheKey).Err()
	if err != nil {
		slog.Warn("Failed to invalidate user permissions cache", 
			slog.String("error", err.Error()),
			slog.Int("character_id", characterID))
	}
	
	return err
}

// InvalidateAllUserPermissions clears all cached user permissions
func (ps *PermissionService) InvalidateAllUserPermissions(ctx context.Context) error {
	if ps.redis == nil {
		return nil
	}

	pattern := "user_permissions:*"
	keys, err := ps.redis.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get permission cache keys: %w", err)
	}

	if len(keys) > 0 {
		err = ps.redis.Client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to clear permission cache: %w", err)
		}
		
		slog.Info("Cleared user permissions cache", slog.Int("keys", len(keys)))
	}

	return nil
}

// CheckGroupPermission checks if a user has a specific group permission
func (ps *PermissionService) CheckGroupPermission(ctx context.Context, characterID int, groupName, resource, action string) (bool, error) {
	// Get user's groups
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return false, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Find the specific group
	var targetGroup *Group
	for _, group := range groups {
		if group.Name == groupName {
			targetGroup = &group
			break
		}
	}

	if targetGroup == nil {
		return false, nil // User is not a member of the group
	}

	// Check if the group has the required permission
	if actions, exists := targetGroup.Permissions[resource]; exists {
		for _, a := range actions {
			if a == action || a == "*" {
				return true, nil
			}
		}
	}

	// Check wildcard resource
	if actions, exists := targetGroup.Permissions["*"]; exists {
		for _, a := range actions {
			if a == action || a == "*" {
				return true, nil
			}
		}
	}

	return false, nil
}

// IsUserInGroup checks if a user is a member of a specific group
func (ps *PermissionService) IsUserInGroup(ctx context.Context, characterID int, groupName string) (bool, error) {
	groups, err := ps.groupService.GetUserGroups(ctx, characterID)
	if err != nil {
		return false, fmt.Errorf("failed to get user groups: %w", err)
	}

	for _, group := range groups {
		if group.Name == groupName {
			return true, nil
		}
	}

	return false, nil
}

// GetUsersWithPermission returns all users who have a specific permission
func (ps *PermissionService) GetUsersWithPermission(ctx context.Context, resource, action string) ([]int, error) {
	// This is a more expensive operation - we'll query groups that have the permission
	// then find all members of those groups
	
	collection := ps.mongodb.Database.Collection("groups")
	
	// Find groups with the required permission
	filter := bson.M{
		"$or": []bson.M{
			{fmt.Sprintf("permissions.%s", resource): action},
			{fmt.Sprintf("permissions.%s", resource): "*"},
			{"permissions.*": "*"}, // Super admin wildcard
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups with permission: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	// Get all members of these groups
	var allUsers []int
	userSet := make(map[int]bool)

	for _, group := range groups {
		members, err := ps.groupService.ListGroupMembers(ctx, group.ID.Hex())
		if err != nil {
			slog.Warn("Failed to get members for group", 
				slog.String("group", group.Name),
				slog.String("error", err.Error()))
			continue
		}

		for _, member := range members {
			if !userSet[member.CharacterID] {
				userSet[member.CharacterID] = true
				allUsers = append(allUsers, member.CharacterID)
			}
		}
	}

	return allUsers, nil
}

// BulkCheckPermissions checks multiple permissions for a user at once
func (ps *PermissionService) BulkCheckPermissions(ctx context.Context, characterID int, checks []PermissionCheck) (map[string]bool, error) {
	permissions, err := ps.GetUserPermissions(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	results := make(map[string]bool, len(checks))
	for _, check := range checks {
		key := fmt.Sprintf("%s:%s", check.Resource, check.Action)
		results[key] = ps.hasPermission(permissions, check.Resource, check.Action)
	}

	return results, nil
}

// PermissionCheck represents a permission check request
type PermissionCheck struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// RefreshUserPermissions forces a refresh of user permissions from the database
func (ps *PermissionService) RefreshUserPermissions(ctx context.Context, characterID int) (*UserPermissionMatrix, error) {
	// Invalidate cache first
	if err := ps.InvalidateUserPermissions(ctx, characterID); err != nil {
		slog.Warn("Failed to invalidate permissions cache during refresh", 
			slog.String("error", err.Error()))
	}

	// Get fresh permissions
	return ps.GetUserPermissions(ctx, characterID)
}

// AnalyzeGroupPermissions provides analytics about group permissions
func (ps *PermissionService) AnalyzeGroupPermissions(ctx context.Context) (*PermissionAnalysis, error) {
	groups, err := ps.groupService.ListGroups(ctx, true) // Include all groups
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	analysis := &PermissionAnalysis{
		TotalGroups:     len(groups),
		DefaultGroups:   0,
		CustomGroups:    0,
		ResourceActions: make(map[string][]string),
		GroupSummaries:  make([]GroupPermissionSummary, 0, len(groups)),
	}

	resourceActionsSet := make(map[string]map[string]bool)

	for _, group := range groups {
		if group.IsDefault {
			analysis.DefaultGroups++
		} else {
			analysis.CustomGroups++
		}

		summary := GroupPermissionSummary{
			GroupName:   group.Name,
			IsDefault:   group.IsDefault,
			Permissions: group.Permissions,
		}

		// Count members
		members, err := ps.groupService.ListGroupMembers(ctx, group.ID.Hex())
		if err == nil {
			summary.MemberCount = len(members)
		}

		analysis.GroupSummaries = append(analysis.GroupSummaries, summary)

		// Collect all resource-action combinations
		for resource, actions := range group.Permissions {
			if resourceActionsSet[resource] == nil {
				resourceActionsSet[resource] = make(map[string]bool)
			}
			for _, action := range actions {
				resourceActionsSet[resource][action] = true
			}
		}
	}

	// Convert set to slice
	for resource, actions := range resourceActionsSet {
		actionList := make([]string, 0, len(actions))
		for action := range actions {
			actionList = append(actionList, action)
		}
		analysis.ResourceActions[resource] = actionList
	}

	return analysis, nil
}

// PermissionAnalysis provides analytics about the permission system
type PermissionAnalysis struct {
	TotalGroups     int                            `json:"total_groups"`
	DefaultGroups   int                            `json:"default_groups"`
	CustomGroups    int                            `json:"custom_groups"`
	ResourceActions map[string][]string            `json:"resource_actions"`
	GroupSummaries  []GroupPermissionSummary       `json:"group_summaries"`
}

// GroupPermissionSummary provides a summary of a group's permissions
type GroupPermissionSummary struct {
	GroupName   string                 `json:"group_name"`
	IsDefault   bool                   `json:"is_default"`
	MemberCount int                    `json:"member_count"`
	Permissions map[string][]string    `json:"permissions"`
}

// ValidatePermissionStructure validates that permission configurations are valid
func (ps *PermissionService) ValidatePermissionStructure(permissions map[string][]string) []string {
	var issues []string

	validResources := map[string]bool{
		"public":        true,
		"user":          true,
		"profile":       true,
		"corporation":   true,
		"alliance":      true,
		"groups":        true,
		"system":        true,
		"admin":         true,
		"scheduler":     true,
		"notifications": true,
		"dev":           true,
		"*":             true, // Wildcard resource
	}

	validActions := map[string]bool{
		"read":   true,
		"write":  true,
		"delete": true,
		"admin":  true,
		"*":      true, // Wildcard action
	}

	for resource, actions := range permissions {
		if !validResources[resource] {
			issues = append(issues, fmt.Sprintf("Invalid resource: %s", resource))
		}

		for _, action := range actions {
			if !validActions[action] {
				issues = append(issues, fmt.Sprintf("Invalid action '%s' for resource '%s'", action, resource))
			}
		}

		if len(actions) == 0 {
			issues = append(issues, fmt.Sprintf("Resource '%s' has no actions defined", resource))
		}
	}

	return issues
}