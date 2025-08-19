package casbin

import "time"

// ====================
// HUMA Input DTOs
// ====================

// RoleAssignmentInput represents the input for assigning a role to a user
type RoleAssignmentInput struct {
	Body RoleAssignmentRequest `json:"body"`
}

// RoleRemovalInput represents the input for removing a role from a user
type RoleRemovalInput struct {
	Body RoleRemovalRequest `json:"body"`
}

// PolicyAssignmentInput represents the input for assigning a permission to a role/user
type PolicyAssignmentInput struct {
	Body PolicyAssignmentRequest `json:"body"`
}

// PolicyRemovalInput represents the input for removing a permission from a role/user
type PolicyRemovalInput struct {
	Body PolicyRemovalRequest `json:"body"`
}

// PermissionCheckInput represents the input for checking user permissions
type PermissionCheckInput struct {
	Body PermissionCheckRequestWithChar `json:"body"`
}

// UserRolesInput represents the input for getting user roles
type UserRolesInput struct {
	UserID string `path:"user_id" validate:"required" doc:"User ID to get roles for"`
}

// RolePoliciesInput represents the input for getting role policies
type RolePoliciesInput struct {
	Role string `path:"role" validate:"required" doc:"Role name to get policies for"`
}

// ====================
// HUMA Output DTOs
// ====================

// RoleAssignmentOutput represents the output for role assignment operations
type RoleAssignmentOutput struct {
	Body RoleAssignmentResponse `json:"body"`
}

// PolicyAssignmentOutput represents the output for policy assignment operations
type PolicyAssignmentOutput struct {
	Body PolicyAssignmentResponse `json:"body"`
}

// PermissionCheckOutput represents the output for permission check operations
type PermissionCheckOutput struct {
	Body PermissionCheckResponseWithChar `json:"body"`
}

// UserRolesOutput represents the output for user roles operations
type UserRolesOutput struct {
	Body UserRolesResponse `json:"body"`
}

// RolePoliciesOutput represents the output for role policies operations
type RolePoliciesOutput struct {
	Body RolePoliciesResponse `json:"body"`
}

// PolicyListOutput represents the output for listing all policies
type PolicyListOutput struct {
	Body PolicyListResponse `json:"body"`
}

// RoleListOutput represents the output for listing all roles
type RoleListOutput struct {
	Body RoleListResponse `json:"body"`
}

// ====================
// Request/Response DTOs
// ====================

// RoleAssignmentRequest represents a role assignment request
type RoleAssignmentRequest struct {
	UserID      string `json:"user_id" validate:"required" minLength:"3" maxLength:"100" doc:"User ID to assign role to"`
	CharacterID *int64 `json:"character_id,omitempty" doc:"Optional character ID for character-specific role"`
	Role        string `json:"role" validate:"required" minLength:"3" maxLength:"50" doc:"Role name to assign"`
	Domain      string `json:"domain" validate:"omitempty" maxLength:"50" doc:"Domain for role assignment (default: global)"`
}

// RoleRemovalRequest represents a role removal request
type RoleRemovalRequest struct {
	UserID      string `json:"user_id" validate:"required" minLength:"3" maxLength:"100" doc:"User ID to remove role from"`
	CharacterID *int64 `json:"character_id,omitempty" doc:"Optional character ID for character-specific role"`
	Role        string `json:"role" validate:"required" minLength:"3" maxLength:"50" doc:"Role name to remove"`
	Domain      string `json:"domain" validate:"omitempty" maxLength:"50" doc:"Domain for role removal (default: global)"`
}

// PolicyAssignmentRequest represents a policy assignment request
type PolicyAssignmentRequest struct {
	Subject  string `json:"subject" validate:"required" minLength:"3" maxLength:"100" doc:"Subject (user:id, role:name, character:id, corporation:id, alliance:id)"`
	Resource string `json:"resource" validate:"required" minLength:"3" maxLength:"50" doc:"Resource name (e.g., scheduler, users, auth)"`
	Action   string `json:"action" validate:"required" minLength:"3" maxLength:"50" doc:"Action name (e.g., read, write, delete, admin)"`
	Domain   string `json:"domain" validate:"omitempty" maxLength:"50" doc:"Domain for permission (default: global)"`
	Effect   string `json:"effect" validate:"required,oneof=allow deny" doc:"Permission effect (allow or deny)"`
}

// PolicyRemovalRequest represents a policy removal request
type PolicyRemovalRequest struct {
	Subject  string `json:"subject" validate:"required" minLength:"3" maxLength:"100" doc:"Subject (user:id, role:name, character:id, corporation:id, alliance:id)"`
	Resource string `json:"resource" validate:"required" minLength:"3" maxLength:"50" doc:"Resource name (e.g., scheduler, users, auth)"`
	Action   string `json:"action" validate:"required" minLength:"3" maxLength:"50" doc:"Action name (e.g., read, write, delete, admin)"`
	Domain   string `json:"domain" validate:"omitempty" maxLength:"50" doc:"Domain for permission (default: global)"`
	Effect   string `json:"effect" validate:"required,oneof=allow deny" doc:"Permission effect (allow or deny)"`
}

// PermissionCheckRequestWithChar extends the existing PermissionCheckRequest with character support
type PermissionCheckRequestWithChar struct {
	UserID      string `json:"user_id" validate:"required" minLength:"3" maxLength:"100" doc:"User ID to check permissions for"`
	CharacterID *int64 `json:"character_id,omitempty" doc:"Optional character ID for character-specific check"`
	Resource    string `json:"resource" validate:"required" minLength:"3" maxLength:"50" doc:"Resource name to check"`
	Action      string `json:"action" validate:"required" minLength:"3" maxLength:"50" doc:"Action to check"`
	Domain      string `json:"domain" validate:"omitempty" maxLength:"50" doc:"Domain to check (default: global)"`
}

// RoleAssignmentResponse represents a role assignment response
type RoleAssignmentResponse struct {
	Success   bool      `json:"success" doc:"Whether the operation was successful"`
	Message   string    `json:"message" doc:"Success or error message"`
	UserID    string    `json:"user_id" doc:"User ID that was affected"`
	Role      string    `json:"role" doc:"Role that was assigned/removed"`
	Domain    string    `json:"domain" doc:"Domain of the role assignment"`
	Timestamp time.Time `json:"timestamp" doc:"When the operation occurred"`
}

// PolicyAssignmentResponse represents a policy assignment response
type PolicyAssignmentResponse struct {
	Success   bool      `json:"success" doc:"Whether the operation was successful"`
	Message   string    `json:"message" doc:"Success or error message"`
	Subject   string    `json:"subject" doc:"Subject that was affected"`
	Resource  string    `json:"resource" doc:"Resource that was affected"`
	Action    string    `json:"action" doc:"Action that was affected"`
	Effect    string    `json:"effect" doc:"Permission effect (allow/deny)"`
	Timestamp time.Time `json:"timestamp" doc:"When the operation occurred"`
}

// PermissionCheckResponseWithChar extends permission check response with additional info
type PermissionCheckResponseWithChar struct {
	HasPermission bool      `json:"has_permission" doc:"Whether the user has the requested permission"`
	UserID        string    `json:"user_id" doc:"User ID that was checked"`
	CharacterID   *int64    `json:"character_id,omitempty" doc:"Character ID that was checked"`
	Resource      string    `json:"resource" doc:"Resource that was checked"`
	Action        string    `json:"action" doc:"Action that was checked"`
	MatchedRules  []string  `json:"matched_rules" doc:"CASBIN rules that matched"`
	CheckedAt     time.Time `json:"checked_at" doc:"When the check was performed"`
}

// UserRolesResponse represents user roles response
type UserRolesResponse struct {
	UserID string     `json:"user_id" doc:"User ID"`
	Roles  []RoleInfo `json:"roles" doc:"List of roles assigned to the user"`
	Total  int        `json:"total" doc:"Total number of roles"`
}

// RolePoliciesResponse represents role policies response
type RolePoliciesResponse struct {
	Role     string       `json:"role" doc:"Role name"`
	Policies []PolicyInfo `json:"policies" doc:"List of policies assigned to the role"`
	Total    int          `json:"total" doc:"Total number of policies"`
}

// PolicyListResponse represents all policies response
type PolicyListResponse struct {
	Policies []PolicyInfo `json:"policies" doc:"List of all policies in the system"`
	Total    int          `json:"total" doc:"Total number of policies"`
}

// RoleListResponse represents all roles response
type RoleListResponse struct {
	Roles []RoleInfo `json:"roles" doc:"List of all roles in the system"`
	Total int        `json:"total" doc:"Total number of roles"`
}

// ====================
// Supporting Types
// ====================

// RoleInfo represents information about a role
type RoleInfo struct {
	Role      string    `json:"role" doc:"Role name"`
	Domain    string    `json:"domain" doc:"Role domain"`
	AssignedAt time.Time `json:"assigned_at,omitempty" doc:"When the role was assigned"`
}

// PolicyInfo represents information about a policy
type PolicyInfo struct {
	Subject  string `json:"subject" doc:"Subject of the policy"`
	Resource string `json:"resource" doc:"Resource of the policy"`
	Action   string `json:"action" doc:"Action of the policy"`
	Domain   string `json:"domain" doc:"Domain of the policy"`
	Effect   string `json:"effect" doc:"Effect of the policy (allow/deny)"`
}

// ====================
// Bulk Operation DTOs
// ====================

// BulkRoleAssignmentInput represents bulk role assignment input
type BulkRoleAssignmentInput struct {
	Body BulkRoleAssignmentRequest `json:"body"`
}

// BulkRoleAssignmentRequest represents bulk role assignment request
type BulkRoleAssignmentRequest struct {
	UserIDs []string `json:"user_ids" validate:"required,min=1,max=100" doc:"List of user IDs (max 100)"`
	Role    string   `json:"role" validate:"required" minLength:"3" maxLength:"50" doc:"Role to assign to all users"`
	Domain  string   `json:"domain" validate:"omitempty" maxLength:"50" doc:"Domain for role assignment (default: global)"`
}

// BulkRoleAssignmentOutput represents bulk role assignment output
type BulkRoleAssignmentOutput struct {
	Body BulkRoleAssignmentResponse `json:"body"`
}

// BulkRoleAssignmentResponse represents bulk role assignment response
type BulkRoleAssignmentResponse struct {
	Success       []string  `json:"success" doc:"User IDs that were successfully assigned the role"`
	Failed        []string  `json:"failed" doc:"User IDs that failed role assignment"`
	Total         int       `json:"total" doc:"Total number of users processed"`
	SuccessCount  int       `json:"success_count" doc:"Number of successful assignments"`
	FailureCount  int       `json:"failure_count" doc:"Number of failed assignments"`
	Role          string    `json:"role" doc:"Role that was assigned"`
	ProcessedAt   time.Time `json:"processed_at" doc:"When the bulk operation was processed"`
}