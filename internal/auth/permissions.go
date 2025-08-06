package auth

import (
	"fmt"
	"strings"
)

// System permissions define what actions users can perform
const (
	// User management permissions
	PermissionUsersRead   = "users:read"
	PermissionUsersWrite  = "users:write"
	PermissionUsersDelete = "users:delete"
	PermissionUsersAdmin  = "users:admin"
	
	// Scheduler permissions
	PermissionSchedulerRead   = "scheduler:read"
	PermissionSchedulerWrite  = "scheduler:write"
	PermissionSchedulerDelete = "scheduler:delete"
	PermissionSchedulerAdmin  = "scheduler:admin"
	
	// Notifications permissions
	PermissionNotificationsRead   = "notifications:read"
	PermissionNotificationsWrite  = "notifications:write"
	PermissionNotificationsDelete = "notifications:delete"
	PermissionNotificationsAdmin  = "notifications:admin"
	
	// Development/debugging permissions (for dev module)
	PermissionDevRead  = "dev:read"
	PermissionDevWrite = "dev:write"
	PermissionDevAdmin = "dev:admin"
	
	// System administration permissions
	PermissionSystemAdmin     = "system:admin"
	PermissionSystemRead      = "system:read"
	PermissionPermissionsRead = "permissions:read"
	PermissionPermissionsWrite = "permissions:write"
)

// systemPermissions contains all available system permissions
var systemPermissions = []string{
	// User management
	PermissionUsersRead,
	PermissionUsersWrite,
	PermissionUsersDelete,
	PermissionUsersAdmin,
	
	// Scheduler
	PermissionSchedulerRead,
	PermissionSchedulerWrite,
	PermissionSchedulerDelete,
	PermissionSchedulerAdmin,
	
	// Notifications
	PermissionNotificationsRead,
	PermissionNotificationsWrite,
	PermissionNotificationsDelete,
	PermissionNotificationsAdmin,
	
	// Development
	PermissionDevRead,
	PermissionDevWrite,
	PermissionDevAdmin,
	
	// System administration
	PermissionSystemAdmin,
	PermissionSystemRead,
	PermissionPermissionsRead,
	PermissionPermissionsWrite,
}

// Admin permissions that grant permission management capabilities
var adminPermissions = []string{
	PermissionSystemAdmin,
	PermissionUsersAdmin,
	PermissionPermissionsWrite,
}

// GetSystemPermissions returns all available system permissions
func GetSystemPermissions() []string {
	return append([]string{}, systemPermissions...) // Return a copy
}

// ValidatePermission validates if a permission exists in the system
func ValidatePermission(permission string) error {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return fmt.Errorf("permission cannot be empty")
	}
	
	// Check if permission follows resource:action format
	parts := strings.Split(permission, ":")
	if len(parts) != 2 {
		return fmt.Errorf("permission must follow 'resource:action' format, got: %s", permission)
	}
	
	resource := strings.TrimSpace(parts[0])
	action := strings.TrimSpace(parts[1])
	
	if resource == "" || action == "" {
		return fmt.Errorf("permission resource and action cannot be empty: %s", permission)
	}
	
	// Check if it's a known system permission
	for _, systemPerm := range systemPermissions {
		if systemPerm == permission {
			return nil
		}
	}
	
	return fmt.Errorf("unknown permission: %s", permission)
}

// IsValidResourceAction validates resource:action format
func IsValidResourceAction(resource, action string) bool {
	if resource == "" || action == "" {
		return false
	}
	
	// Check valid resources
	validResources := []string{"users", "scheduler", "notifications", "dev", "system", "permissions"}
	validActions := []string{"read", "write", "delete", "admin"}
	
	resourceValid := false
	for _, r := range validResources {
		if r == resource {
			resourceValid = true
			break
		}
	}
	
	actionValid := false
	for _, a := range validActions {
		if a == action {
			actionValid = true
			break
		}
	}
	
	return resourceValid && actionValid
}

// IsAdminPermission checks if a permission grants admin capabilities
func IsAdminPermission(permission string) bool {
	for _, adminPerm := range adminPermissions {
		if adminPerm == permission {
			return true
		}
	}
	return false
}

// HasAdminPermission checks if a user has admin permissions
func HasAdminPermission(userPermissions []string) bool {
	for _, userPerm := range userPermissions {
		if IsAdminPermission(userPerm) {
			return true
		}
	}
	return false
}

// GetPermissionsByResource returns all permissions for a specific resource
func GetPermissionsByResource(resource string) []string {
	var permissions []string
	prefix := resource + ":"
	
	for _, perm := range systemPermissions {
		if strings.HasPrefix(perm, prefix) {
			permissions = append(permissions, perm)
		}
	}
	
	return permissions
}

// GetResourceFromPermission extracts the resource name from a permission
func GetResourceFromPermission(permission string) string {
	parts := strings.Split(permission, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

// GetActionFromPermission extracts the action from a permission
func GetActionFromPermission(permission string) string {
	parts := strings.Split(permission, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// BuildPermission constructs a permission string from resource and action
func BuildPermission(resource, action string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(resource), strings.TrimSpace(action))
}

// PermissionInfo represents information about a permission
type PermissionInfo struct {
	Permission  string `json:"permission"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
	IsAdmin     bool   `json:"is_admin"`
}

// GetPermissionInfo returns detailed information about all system permissions
func GetPermissionInfo() []PermissionInfo {
	var permissions []PermissionInfo
	
	descriptions := map[string]string{
		// User management
		PermissionUsersRead:   "View user information and profiles",
		PermissionUsersWrite:  "Create and update user information",
		PermissionUsersDelete: "Delete user accounts",
		PermissionUsersAdmin:  "Full user management including permissions",
		
		// Scheduler
		PermissionSchedulerRead:   "View scheduled tasks and execution history",
		PermissionSchedulerWrite:  "Create and modify scheduled tasks",
		PermissionSchedulerDelete: "Delete scheduled tasks",
		PermissionSchedulerAdmin:  "Full scheduler management and configuration",
		
		// Notifications
		PermissionNotificationsRead:   "View notifications and templates",
		PermissionNotificationsWrite:  "Create and send notifications",
		PermissionNotificationsDelete: "Delete notifications and templates",
		PermissionNotificationsAdmin:  "Full notification system management",
		
		// Development
		PermissionDevRead:  "Access development tools and debug information",
		PermissionDevWrite: "Execute development operations and testing",
		PermissionDevAdmin: "Full development environment access",
		
		// System
		PermissionSystemAdmin:      "Full system administration access",
		PermissionSystemRead:       "View system information and status",
		PermissionPermissionsRead:  "View user permissions",
		PermissionPermissionsWrite: "Grant and revoke user permissions",
	}
	
	for _, perm := range systemPermissions {
		resource := GetResourceFromPermission(perm)
		action := GetActionFromPermission(perm)
		description := descriptions[perm]
		if description == "" {
			description = fmt.Sprintf("Permission to %s %s resources", action, resource)
		}
		
		permissions = append(permissions, PermissionInfo{
			Permission:  perm,
			Resource:    resource,
			Action:      action,
			Description: description,
			IsAdmin:     IsAdminPermission(perm),
		})
	}
	
	return permissions
}