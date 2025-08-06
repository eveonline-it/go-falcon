package auth

import (
	"strings"
	"testing"
)

// Test ValidatePermission function with valid permissions
func TestValidatePermission_ValidPermissions(t *testing.T) {
	validPermissions := []string{
		"users:read",
		"users:write", 
		"users:delete",
		"users:admin",
		"scheduler:read",
		"scheduler:write",
		"notifications:read",
		"system:admin",
	}
	
	for _, permission := range validPermissions {
		err := ValidatePermission(permission)
		if err != nil {
			t.Errorf("Expected permission '%s' to be valid, got error: %s", permission, err.Error())
		}
	}
}

// Test ValidatePermission function with invalid permissions
func TestValidatePermission_InvalidPermissions(t *testing.T) {
	invalidPermissions := []string{
		"",                    // Empty permission
		"invalid",            // Missing colon
		"users:",            // Missing action
		":read",             // Missing resource
		"users:read:extra",  // Too many parts
		" users : read ",    // Spaces around parts
		"unknown:action",    // Unknown permission
	}
	
	for _, permission := range invalidPermissions {
		err := ValidatePermission(permission)
		if err == nil {
			t.Errorf("Expected permission '%s' to be invalid, but got no error", permission)
		}
	}
}

// Test IsValidResourceAction function
func TestIsValidResourceAction(t *testing.T) {
	testCases := []struct {
		resource string
		action   string
		expected bool
	}{
		{"users", "read", true},
		{"users", "write", true},
		{"users", "delete", true},
		{"users", "admin", true},
		{"scheduler", "read", true},
		{"notifications", "write", true},
		{"system", "admin", true},
		{"permissions", "read", true},
		{"", "read", false},           // Empty resource
		{"users", "", false},          // Empty action
		{"invalid", "read", false},    // Invalid resource
		{"users", "invalid", false},   // Invalid action
	}
	
	for _, tc := range testCases {
		result := IsValidResourceAction(tc.resource, tc.action)
		if result != tc.expected {
			t.Errorf("IsValidResourceAction(%s, %s) = %v, expected %v", 
				tc.resource, tc.action, result, tc.expected)
		}
	}
}

// Test GetSystemPermissions function
func TestGetSystemPermissions(t *testing.T) {
	permissions := GetSystemPermissions()
	
	// Should have permissions
	if len(permissions) == 0 {
		t.Error("Expected system permissions to be non-empty")
	}
	
	// Should contain expected permissions
	expectedPermissions := []string{
		"users:read",
		"users:write", 
		"users:admin",
		"scheduler:read",
		"scheduler:write",
		"system:admin",
		"permissions:read",
		"permissions:write",
	}
	
	for _, expected := range expectedPermissions {
		found := false
		for _, perm := range permissions {
			if perm == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find permission '%s' in system permissions", expected)
		}
	}
	
	// Should return a copy (modifying returned slice shouldn't affect original)
	originalLen := len(permissions)
	permissions = append(permissions, "test:permission")
	newPermissions := GetSystemPermissions()
	if len(newPermissions) != originalLen {
		t.Error("GetSystemPermissions should return a copy, not the original slice")
	}
}

// Test IsAdminPermission function
func TestIsAdminPermission(t *testing.T) {
	testCases := []struct {
		permission string
		expected   bool
	}{
		{"system:admin", true},
		{"users:admin", true},
		{"permissions:write", true},
		{"users:read", false},
		{"users:write", false},
		{"scheduler:read", false},
		{"scheduler:admin", false}, // Not in admin permissions list
		{"invalid:permission", false},
	}
	
	for _, tc := range testCases {
		result := IsAdminPermission(tc.permission)
		if result != tc.expected {
			t.Errorf("IsAdminPermission(%s) = %v, expected %v", 
				tc.permission, result, tc.expected)
		}
	}
}

// Test HasAdminPermission function
func TestHasAdminPermission(t *testing.T) {
	testCases := []struct {
		permissions []string
		expected    bool
		description string
	}{
		{
			[]string{"system:admin"},
			true,
			"should detect system:admin",
		},
		{
			[]string{"users:admin"},
			true,
			"should detect users:admin",
		},
		{
			[]string{"permissions:write"},
			true,
			"should detect permissions:write",
		},
		{
			[]string{"users:read", "system:admin", "scheduler:read"},
			true,
			"should detect admin permission among others",
		},
		{
			[]string{"users:read", "users:write", "scheduler:read"},
			false,
			"should not detect admin without admin permissions",
		},
		{
			[]string{},
			false,
			"should handle empty permissions list",
		},
	}
	
	for _, tc := range testCases {
		result := HasAdminPermission(tc.permissions)
		if result != tc.expected {
			t.Errorf("%s: HasAdminPermission(%v) = %v, expected %v", 
				tc.description, tc.permissions, result, tc.expected)
		}
	}
}

// Test GetPermissionsByResource function
func TestGetPermissionsByResource(t *testing.T) {
	testCases := []struct {
		resource        string
		expectedCount   int
		shouldContain   []string
		shouldNotContain []string
	}{
		{
			resource:      "users",
			expectedCount: 4, // read, write, delete, admin
			shouldContain: []string{"users:read", "users:write", "users:admin"},
			shouldNotContain: []string{"scheduler:read", "system:admin"},
		},
		{
			resource:      "system", 
			expectedCount: 2, // admin, read
			shouldContain: []string{"system:admin", "system:read"},
			shouldNotContain: []string{"users:admin", "permissions:read"},
		},
		{
			resource:      "nonexistent",
			expectedCount: 0,
			shouldContain: []string{},
			shouldNotContain: []string{"users:read", "system:admin"},
		},
	}
	
	for _, tc := range testCases {
		permissions := GetPermissionsByResource(tc.resource)
		
		if len(permissions) != tc.expectedCount {
			t.Errorf("GetPermissionsByResource(%s) returned %d permissions, expected %d",
				tc.resource, len(permissions), tc.expectedCount)
		}
		
		for _, expected := range tc.shouldContain {
			found := false
			for _, perm := range permissions {
				if perm == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected GetPermissionsByResource(%s) to contain '%s'",
					tc.resource, expected)
			}
		}
		
		for _, notExpected := range tc.shouldNotContain {
			for _, perm := range permissions {
				if perm == notExpected {
					t.Errorf("Expected GetPermissionsByResource(%s) to NOT contain '%s'",
						tc.resource, notExpected)
				}
			}
		}
	}
}

// Test GetResourceFromPermission function
func TestGetResourceFromPermission(t *testing.T) {
	testCases := []struct {
		permission string
		expected   string
	}{
		{"users:read", "users"},
		{"scheduler:write", "scheduler"},
		{"system:admin", "system"},
		{"permissions:read", "permissions"},
		{"invalid", ""},           // No colon
		{"users:read:extra", ""}, // Too many parts (returns empty for invalid format)
		{"", ""},                 // Empty string
	}
	
	for _, tc := range testCases {
		result := GetResourceFromPermission(tc.permission)
		if result != tc.expected {
			t.Errorf("GetResourceFromPermission(%s) = '%s', expected '%s'",
				tc.permission, result, tc.expected)
		}
	}
}

// Test GetActionFromPermission function
func TestGetActionFromPermission(t *testing.T) {
	testCases := []struct {
		permission string
		expected   string
	}{
		{"users:read", "read"},
		{"scheduler:write", "write"},
		{"system:admin", "admin"},
		{"permissions:delete", "delete"},
		{"invalid", ""},           // No colon
		{"users:read:extra", ""}, // Too many parts (returns empty for invalid format)
		{"", ""},                 // Empty string
	}
	
	for _, tc := range testCases {
		result := GetActionFromPermission(tc.permission)
		if result != tc.expected {
			t.Errorf("GetActionFromPermission(%s) = '%s', expected '%s'",
				tc.permission, result, tc.expected)
		}
	}
}

// Test BuildPermission function
func TestBuildPermission(t *testing.T) {
	testCases := []struct {
		resource string
		action   string
		expected string
	}{
		{"users", "read", "users:read"},
		{"scheduler", "write", "scheduler:write"},
		{"system", "admin", "system:admin"},
		{" users ", " read ", "users:read"}, // Should trim spaces
		{"", "read", ":read"},               // Empty resource
		{"users", "", "users:"},             // Empty action
	}
	
	for _, tc := range testCases {
		result := BuildPermission(tc.resource, tc.action)
		if result != tc.expected {
			t.Errorf("BuildPermission(%s, %s) = '%s', expected '%s'",
				tc.resource, tc.action, result, tc.expected)
		}
	}
}

// Test GetPermissionInfo function
func TestGetPermissionInfo(t *testing.T) {
	permissionInfos := GetPermissionInfo()
	
	// Should have permission info for all system permissions
	systemPermissions := GetSystemPermissions()
	if len(permissionInfos) != len(systemPermissions) {
		t.Errorf("Expected %d permission infos, got %d", 
			len(systemPermissions), len(permissionInfos))
	}
	
	// Check that each permission info has required fields
	for _, info := range permissionInfos {
		if info.Permission == "" {
			t.Error("Permission info should have non-empty Permission field")
		}
		
		if info.Resource == "" {
			t.Errorf("Permission info for '%s' should have non-empty Resource field", info.Permission)
		}
		
		if info.Action == "" {
			t.Errorf("Permission info for '%s' should have non-empty Action field", info.Permission)
		}
		
		if info.Description == "" {
			t.Errorf("Permission info for '%s' should have non-empty Description field", info.Permission)
		}
		
		// Check that resource and action match the permission
		expectedPermission := BuildPermission(info.Resource, info.Action)
		if info.Permission != expectedPermission {
			t.Errorf("Permission info mismatch: permission='%s', but resource:action='%s'",
				info.Permission, expectedPermission)
		}
		
		// Check that IsAdmin flag matches IsAdminPermission function
		if info.IsAdmin != IsAdminPermission(info.Permission) {
			t.Errorf("Permission info IsAdmin flag mismatch for '%s': got %v, expected %v",
				info.Permission, info.IsAdmin, IsAdminPermission(info.Permission))
		}
	}
	
	// Test specific permission info
	var usersReadInfo *PermissionInfo
	for i := range permissionInfos {
		if permissionInfos[i].Permission == "users:read" {
			usersReadInfo = &permissionInfos[i]
			break
		}
	}
	
	if usersReadInfo == nil {
		t.Error("Expected to find permission info for 'users:read'")
	} else {
		if usersReadInfo.Resource != "users" {
			t.Errorf("Expected resource 'users', got '%s'", usersReadInfo.Resource)
		}
		if usersReadInfo.Action != "read" {
			t.Errorf("Expected action 'read', got '%s'", usersReadInfo.Action)
		}
		if usersReadInfo.IsAdmin {
			t.Error("Expected users:read to not be an admin permission")
		}
		if !strings.Contains(strings.ToLower(usersReadInfo.Description), "view") {
			t.Errorf("Expected users:read description to mention 'view', got: %s", usersReadInfo.Description)
		}
	}
}