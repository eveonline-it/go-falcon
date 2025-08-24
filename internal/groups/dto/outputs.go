package dto

import (
	"time"
)

// GroupOutput represents a group API response
type GroupOutput struct {
	Body GroupResponse `json:"body"`
}

// GroupResponse represents the actual group data
type GroupResponse struct {
	ID          string    `json:"id" description:"Group ID"`
	Name        string    `json:"name" description:"Group name"`
	Description string    `json:"description" description:"Group description"`
	Type        string    `json:"type" description:"Group type"`
	SystemName  *string   `json:"system_name,omitempty" description:"System group identifier"`
	EVEEntityID *int64    `json:"eve_entity_id,omitempty" description:"EVE Corporation/Alliance ID"`
	IsActive    bool      `json:"is_active" description:"Whether the group is active"`
	MemberCount *int64    `json:"member_count,omitempty" description:"Number of active members"`
	CreatedBy   *int64    `json:"created_by,omitempty" description:"Character ID who created this group"`
	CreatedAt   time.Time `json:"created_at" description:"Creation timestamp"`
	UpdatedAt   time.Time `json:"updated_at" description:"Last update timestamp"`
}

// GroupMembershipOutput represents a group membership API response
type GroupMembershipOutput struct {
	Body GroupMembershipResponse `json:"body"`
}

// GroupMembershipResponse represents the actual membership data
type GroupMembershipResponse struct {
	ID            string    `json:"id" description:"Membership ID"`
	GroupID       string    `json:"group_id" description:"Group ID"`
	CharacterID   int64     `json:"character_id" description:"Character ID"`
	CharacterName string    `json:"character_name" description:"Character name"`
	IsActive      bool      `json:"is_active" description:"Whether the membership is active"`
	AddedBy       *int64    `json:"added_by,omitempty" description:"Character ID who added this membership"`
	AddedAt       time.Time `json:"added_at" description:"When the membership was added"`
	UpdatedAt     time.Time `json:"updated_at" description:"Last update timestamp"`
}

// ListGroupsOutput represents the response for listing groups
type ListGroupsOutput struct {
	Body ListGroupsResponse `json:"body"`
}

// ListGroupsResponse represents the actual response data for listing groups
type ListGroupsResponse struct {
	Groups []GroupResponse `json:"groups" description:"List of groups"`
	Total  int64           `json:"total" description:"Total number of groups matching the criteria"`
	Page   int             `json:"page" description:"Current page number"`
	Limit  int             `json:"limit" description:"Items per page"`
}

// ListMembersOutput represents the response for listing group members
type ListMembersOutput struct {
	Body ListMembersResponse `json:"body"`
}

// ListMembersResponse represents the actual response data for listing group members
type ListMembersResponse struct {
	Members []GroupMembershipResponse `json:"members" description:"List of group members"`
	Total   int64                     `json:"total" description:"Total number of members matching the criteria"`
	Page    int                       `json:"page" description:"Current page number"`
	Limit   int                       `json:"limit" description:"Items per page"`
}

// MembershipCheckOutput represents the response for checking membership
type MembershipCheckOutput struct {
	Body MembershipCheckResponse `json:"body"`
}

// MembershipCheckResponse represents the actual membership check data
type MembershipCheckResponse struct {
	IsMember bool       `json:"is_member" description:"Whether the character is a member of the group"`
	IsActive bool       `json:"is_active" description:"Whether the membership is active (only relevant if is_member is true)"`
	AddedAt  *time.Time `json:"added_at,omitempty" description:"When the membership was added (only relevant if is_member is true)"`
}

// CharacterGroupsOutput represents the response for getting character groups
type CharacterGroupsOutput struct {
	Body CharacterGroupsResponse `json:"body"`
}

// CharacterGroupsResponse represents the actual character groups data
type CharacterGroupsResponse struct {
	Groups []GroupResponse `json:"groups" description:"List of groups the character belongs to"`
	Total  int64           `json:"total" description:"Total number of groups"`
}

// SuccessOutput represents a simple success response
type SuccessOutput struct {
	Body SuccessResponse `json:"body"`
}

// SuccessResponse represents the actual success data
type SuccessResponse struct {
	Message string `json:"message" description:"Success message"`
}

// HealthOutput represents the health check response
type HealthOutput struct {
	Body HealthResponse `json:"body"`
}

// HealthResponse represents the actual health response data
type HealthResponse struct {
	Health string `json:"health" description:"Health status"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body GroupsStatusResponse `json:"body"`
}

// GroupsStatusResponse represents the actual status response data
type GroupsStatusResponse struct {
	Module  string `json:"module" description:"Module name"`
	Status  string `json:"status" enum:"healthy,unhealthy" description:"Module health status"`
	Message string `json:"message,omitempty" description:"Optional status message or error details"`
}

// UserGroupsOutput represents the response for getting user groups
type UserGroupsOutput struct {
	Body UserGroupsResponse `json:"body"`
}

// UserGroupsResponse represents the actual user groups data
type UserGroupsResponse struct {
	UserID     string          `json:"user_id" description:"User ID"`
	Characters []int64         `json:"characters" description:"List of character IDs belonging to this user"`
	Groups     []GroupResponse `json:"groups" description:"List of unique groups across all user's characters"`
	Total      int64           `json:"total" description:"Total number of unique groups"`
}

// PermissionOutput represents a single permission response
type PermissionOutput struct {
	Body PermissionResponse `json:"body"`
}

// PermissionResponse represents the actual permission data
type PermissionResponse struct {
	ID          string    `json:"id" description:"Permission ID (e.g., 'intel:reports:write')"`
	Service     string    `json:"service" description:"Service name (e.g., 'intel', 'scheduler')"`
	Resource    string    `json:"resource" description:"Resource name (e.g., 'reports', 'tasks')"`
	Action      string    `json:"action" description:"Action name (e.g., 'write', 'read', 'create')"`
	IsStatic    bool      `json:"is_static" description:"Whether permission is hardcoded (true) or configurable (false)"`
	Name        string    `json:"name" description:"Human-readable permission name"`
	Description string    `json:"description" description:"What this permission allows"`
	Category    string    `json:"category" description:"Permission category for UI grouping"`
	CreatedAt   time.Time `json:"created_at" description:"Creation timestamp"`
}

// ListPermissionsOutput represents the response for listing permissions
type ListPermissionsOutput struct {
	Body ListPermissionsResponse `json:"body"`
}

// ListPermissionsResponse represents the actual permissions list data
type ListPermissionsResponse struct {
	Permissions []PermissionResponse `json:"permissions" description:"List of permissions"`
	Categories  []PermissionCategory `json:"categories" description:"Permission categories for UI organization"`
	Total       int64                `json:"total" description:"Total number of permissions"`
}

// PermissionCategory represents a permission category for UI organization
type PermissionCategory struct {
	Name        string `json:"name" description:"Category name"`
	Description string `json:"description" description:"Category description"`
	Order       int    `json:"order" description:"Display order in UI"`
}

// GroupPermissionOutput represents a group permission assignment response
type GroupPermissionOutput struct {
	Body GroupPermissionResponse `json:"body"`
}

// GroupPermissionResponse represents the actual group permission data
type GroupPermissionResponse struct {
	ID           string             `json:"id" description:"Assignment ID"`
	GroupID      string             `json:"group_id" description:"Group ID"`
	GroupName    string             `json:"group_name" description:"Group name"`
	PermissionID string             `json:"permission_id" description:"Permission ID"`
	Permission   PermissionResponse `json:"permission" description:"Permission details"`
	GrantedBy    *int64             `json:"granted_by,omitempty" description:"Character ID who granted the permission"`
	GrantedAt    time.Time          `json:"granted_at" description:"When permission was granted"`
	IsActive     bool               `json:"is_active" description:"Whether the assignment is active"`
	UpdatedAt    time.Time          `json:"updated_at" description:"Last update timestamp"`
}

// ListGroupPermissionsOutput represents the response for listing group permissions
type ListGroupPermissionsOutput struct {
	Body ListGroupPermissionsResponse `json:"body"`
}

// ListGroupPermissionsResponse represents the actual group permissions list data
type ListGroupPermissionsResponse struct {
	GroupID     string                    `json:"group_id" description:"Group ID"`
	GroupName   string                    `json:"group_name" description:"Group name"`
	Permissions []GroupPermissionResponse `json:"permissions" description:"List of permissions assigned to this group"`
	Total       int64                     `json:"total" description:"Total number of permissions"`
}

// PermissionCheckOutput represents the response for permission checking
type PermissionCheckOutput struct {
	Body PermissionCheckResponse `json:"body"`
}

// PermissionCheckResponse represents the actual permission check data
type PermissionCheckResponse struct {
	CharacterID  int64  `json:"character_id" description:"Character ID that was checked"`
	PermissionID string `json:"permission_id" description:"Permission ID that was checked"`
	Granted      bool   `json:"granted" description:"Whether permission is granted"`
	GrantedVia   string `json:"granted_via,omitempty" description:"Which group granted the permission"`
}

// MessageOutput represents a simple message response
type MessageOutput struct {
	Body MessageResponse `json:"body"`
}

// MessageResponse represents a simple message
type MessageResponse struct {
	Message string `json:"message" description:"Response message"`
}
