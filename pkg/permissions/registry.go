package permissions

import "time"

// Static permission categories for UI organization
var PermissionCategories = []PermissionCategory{
	{Name: "System Administration", Description: "Core system management permissions", Order: 1},
	{Name: "User Management", Description: "User and authentication management", Order: 2},
	{Name: "Group Management", Description: "Group and role management", Order: 3},
	{Name: "Content Management", Description: "Application content and data management", Order: 4},
	{Name: "Fleet Operations", Description: "Fleet command and coordination", Order: 5},
	{Name: "Intelligence", Description: "Intelligence gathering and reporting", Order: 6},
	{Name: "Corporation Management", Description: "Corporation administration", Order: 7},
	{Name: "Alliance Operations", Description: "Alliance-level operations", Order: 8},
}

// StaticPermissions defines hardcoded system permissions that cannot be modified
var StaticPermissions = map[string]Permission{
	// System Administration (Super Admin only)
	"system:admin:full": {
		ID:          "system:admin:full",
		Service:     "system",
		Resource:    "admin",
		Action:      "full",
		IsStatic:    true,
		Name:        "Full System Administration",
		Description: "Complete access to all system functions and data",
		Category:    "System Administration",
		CreatedAt:   time.Now(),
	},
	"system:config:manage": {
		ID:          "system:config:manage",
		Service:     "system",
		Resource:    "config",
		Action:      "manage",
		IsStatic:    true,
		Name:        "System Configuration",
		Description: "Modify system configuration and settings",
		Category:    "System Administration",
		CreatedAt:   time.Now(),
	},

	// User Management
	"users:management:full": {
		ID:          "users:management:full",
		Service:     "users",
		Resource:    "management",
		Action:      "full",
		IsStatic:    true,
		Name:        "User Management",
		Description: "Create, modify, and delete user accounts",
		Category:    "User Management",
		CreatedAt:   time.Now(),
	},
	"users:profiles:view": {
		ID:          "users:profiles:view",
		Service:     "users",
		Resource:    "profiles",
		Action:      "view",
		IsStatic:    true,
		Name:        "View User Profiles",
		Description: "View user profile information",
		Category:    "User Management",
		CreatedAt:   time.Now(),
	},

	// Group Management
	"groups:management:full": {
		ID:          "groups:management:full",
		Service:     "groups",
		Resource:    "management",
		Action:      "full",
		IsStatic:    true,
		Name:        "Group Management",
		Description: "Create, modify, and delete groups",
		Category:    "Group Management",
		CreatedAt:   time.Now(),
	},
	"groups:memberships:manage": {
		ID:          "groups:memberships:manage",
		Service:     "groups",
		Resource:    "memberships",
		Action:      "manage",
		IsStatic:    true,
		Name:        "Group Membership Management",
		Description: "Add and remove members from groups",
		Category:    "Group Management",
		CreatedAt:   time.Now(),
	},
	"groups:permissions:manage": {
		ID:          "groups:permissions:manage",
		Service:     "groups",
		Resource:    "permissions",
		Action:      "manage",
		IsStatic:    true,
		Name:        "Permission Management",
		Description: "Assign and revoke permissions to/from groups",
		Category:    "Group Management",
		CreatedAt:   time.Now(),
	},
	"groups:view:all": {
		ID:          "groups:view:all",
		Service:     "groups",
		Resource:    "view",
		Action:      "all",
		IsStatic:    true,
		Name:        "View All Groups",
		Description: "View group information and memberships",
		Category:    "Group Management",
		CreatedAt:   time.Now(),
	},

	// Authentication System
	"auth:tokens:manage": {
		ID:          "auth:tokens:manage",
		Service:     "auth",
		Resource:    "tokens",
		Action:      "manage",
		IsStatic:    true,
		Name:        "Token Management",
		Description: "Manage authentication tokens and sessions",
		Category:    "User Management",
		CreatedAt:   time.Now(),
	},

	// Scheduler System
	"scheduler:tasks:full": {
		ID:          "scheduler:tasks:full",
		Service:     "scheduler",
		Resource:    "tasks",
		Action:      "full",
		IsStatic:    true,
		Name:        "Task Scheduler Management",
		Description: "Create, modify, delete, and execute scheduled tasks",
		Category:    "System Administration",
		CreatedAt:   time.Now(),
	},
}

// GetStaticPermission retrieves a static permission by ID
func GetStaticPermission(id string) (Permission, bool) {
	perm, exists := StaticPermissions[id]
	return perm, exists
}

// GetAllStaticPermissions returns all static permissions
func GetAllStaticPermissions() map[string]Permission {
	return StaticPermissions
}

// IsStaticPermission checks if a permission ID is static
func IsStaticPermission(id string) bool {
	_, exists := StaticPermissions[id]
	return exists
}