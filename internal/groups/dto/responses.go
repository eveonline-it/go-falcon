package dto

import (
	"time"

	"go-falcon/internal/groups/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GroupResponse represents a group in API responses
type GroupResponse struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	Description         string                       `json:"description"`
	IsDefault           bool                         `json:"is_default"`
	DiscordRoles        []models.DiscordRole         `json:"discord_roles"`
	AutoAssignmentRules *models.AutoAssignmentRules  `json:"auto_assignment_rules,omitempty"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	CreatedBy           int                          `json:"created_by"`
	IsMember            bool                         `json:"is_member"`
	MemberCount         int                          `json:"member_count,omitempty"`
	Members             []MembershipResponse         `json:"members,omitempty"`
}

// GroupListResponse represents a paginated list of groups
type GroupListResponse struct {
	Groups     []GroupResponse   `json:"groups"`
	Pagination PaginationResponse `json:"pagination"`
}

// GroupMemberResponse represents a group member in API responses
type GroupMemberResponse struct {
	CharacterID        int        `json:"character_id"`
	CharacterName      string     `json:"character_name"`
	AssignedAt         time.Time  `json:"assigned_at"`
	AssignedBy         int        `json:"assigned_by"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	LastValidated      time.Time  `json:"last_validated"`
	ValidationStatus   string     `json:"validation_status"`
	IsActive           bool       `json:"is_active"`
}

// GroupMemberListResponse represents a paginated list of group members
type GroupMemberListResponse struct {
	Members    []GroupMemberResponse `json:"members"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalPages int                   `json:"total_pages"`
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Allowed bool     `json:"allowed"`
	Reason  string   `json:"reason,omitempty"`
	Groups  []string `json:"groups,omitempty"`
}

// UserPermissionsResponse represents a user's complete permission matrix
type UserPermissionsResponse struct {
	CharacterID int                            `json:"character_id"`
	Groups      []string                       `json:"groups"`
	Permissions map[string]map[string][]string `json:"permissions"` // resource -> action -> groups
	LastUpdated time.Time                      `json:"last_updated"`
}

// ServiceResponse represents a service in the granular permission system
type ServiceResponse struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	DisplayName string                  `json:"display_name"`
	Description string                  `json:"description"`
	Resources   []models.ResourceConfig `json:"resources"`
	Enabled     bool                    `json:"enabled"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// ServiceListResponse represents a paginated list of services
type ServiceListResponse struct {
	Services   []ServiceResponse `json:"services"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// PermissionAssignmentResponse represents a permission assignment
type PermissionAssignmentResponse struct {
	ID          string     `json:"id"`
	Service     string     `json:"service"`
	Resource    string     `json:"resource"`
	Action      string     `json:"action"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	SubjectName string     `json:"subject_name,omitempty"`
	GrantedBy   int        `json:"granted_by"`
	GrantedAt   time.Time  `json:"granted_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Reason      string     `json:"reason"`
	Enabled     bool       `json:"enabled"`
}

// PermissionAssignmentListResponse represents a paginated list of permission assignments
type PermissionAssignmentListResponse struct {
	Assignments []PermissionAssignmentResponse `json:"assignments"`
	Total       int64                          `json:"total"`
	Page        int                            `json:"page"`
	PageSize    int                            `json:"page_size"`
	TotalPages  int                            `json:"total_pages"`
}

// BulkOperationResponse represents the result of a bulk operation
type BulkOperationResponse struct {
	Success      []string `json:"success"`
	Failed       []string `json:"failed"`
	Total        int      `json:"total"`
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
}

// SubjectResponse represents a subject that can receive permissions
type SubjectResponse struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubjectListResponse represents a list of available subjects
type SubjectListResponse struct {
	Subjects []SubjectResponse `json:"subjects"`
	Total    int64             `json:"total"`
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID          primitive.ObjectID `json:"id"`
	Action      string             `json:"action"`
	Service     string             `json:"service,omitempty"`
	Resource    string             `json:"resource,omitempty"`
	SubjectType string             `json:"subject_type,omitempty"`
	SubjectID   string             `json:"subject_id,omitempty"`
	ActorID     int                `json:"actor_id"`
	ActorName   string             `json:"actor_name,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
	IPAddress   string             `json:"ip_address,omitempty"`
	UserAgent   string             `json:"user_agent,omitempty"`
}

// AuditLogResponse represents a paginated list of audit log entries
type AuditLogResponse struct {
	Entries    []AuditLogEntry `json:"entries"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// UserPermissionSummaryResponse represents a summary of user permissions
type UserPermissionSummaryResponse struct {
	CharacterID         int                         `json:"character_id"`
	CharacterName       string                      `json:"character_name"`
	Groups              []string                    `json:"groups"`
	GranularPermissions map[string][]string         `json:"granular_permissions"` // service.resource.action -> reasons
	LastUpdated         time.Time                   `json:"last_updated"`
	TotalPermissions    int                         `json:"total_permissions"`
}

// ServicePermissionSummaryResponse represents permissions for a specific service
type ServicePermissionSummaryResponse struct {
	Service     string                           `json:"service"`
	Resources   map[string][]string              `json:"resources"` // resource -> actions
	Assignments []PermissionAssignmentResponse   `json:"assignments"`
	Total       int                              `json:"total"`
}

// DiscordRoleResponse represents Discord role information
type DiscordRoleResponse struct {
	ServerID   string `json:"server_id"`
	ServerName string `json:"server_name"`
	RoleName   string `json:"role_name"`
	RoleID     string `json:"role_id,omitempty"`
	Color      string `json:"color,omitempty"`
	Position   int    `json:"position,omitempty"`
}

// DiscordServerResponse represents Discord server information
type DiscordServerResponse struct {
	ServerID   string                `json:"server_id"`
	ServerName string                `json:"server_name"`
	Roles      []DiscordRoleResponse `json:"roles"`
}

// DiscordSyncStatusResponse represents Discord synchronization status
type DiscordSyncStatusResponse struct {
	LastSync      time.Time `json:"last_sync"`
	Status        string    `json:"status"`
	SyncedUsers   int       `json:"synced_users"`
	FailedUsers   int       `json:"failed_users"`
	SyncedServers int       `json:"synced_servers"`
	FailedServers int       `json:"failed_servers"`
	NextSync      time.Time `json:"next_sync"`
}

// ValidationStatusResponse represents the result of membership validation
type ValidationStatusResponse struct {
	CharacterID      int       `json:"character_id"`
	CharacterName    string    `json:"character_name"`
	TotalGroups      int       `json:"total_groups"`
	ValidGroups      int       `json:"valid_groups"`
	InvalidGroups    int       `json:"invalid_groups"`
	LastValidated    time.Time `json:"last_validated"`
	ValidationErrors []string  `json:"validation_errors,omitempty"`
}

// GroupStatsResponse represents statistics about groups
type GroupStatsResponse struct {
	TotalGroups        int                    `json:"total_groups"`
	DefaultGroups      int                    `json:"default_groups"`
	CustomGroups       int                    `json:"custom_groups"`
	TotalMembers       int                    `json:"total_members"`
	ActiveMembers      int                    `json:"active_members"`
	GroupDistribution  map[string]int         `json:"group_distribution"`
	MembershipTrends   map[string]int         `json:"membership_trends"`
	LastUpdated        time.Time              `json:"last_updated"`
}

// Additional DTOs needed by services

// PaginationResponse represents pagination information
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// MembershipResponse represents a group membership
type MembershipResponse struct {
	CharacterID        int                    `json:"character_id"`
	AssignedAt         time.Time              `json:"assigned_at"`
	ExpiresAt          *time.Time             `json:"expires_at,omitempty"`
	ValidationStatus   string                 `json:"validation_status"`
	LastValidated      *time.Time             `json:"last_validated,omitempty"`
	AssignmentSource   string                 `json:"assignment_source"`
	AssignmentMetadata map[string]interface{} `json:"assignment_metadata,omitempty"`
}

// MembershipListResponse represents a paginated list of memberships
type MembershipListResponse struct {
	Members    []MembershipResponse `json:"members"`
	Pagination PaginationResponse   `json:"pagination"`
}