package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Group represents a permission group in the system
type Group struct {
	ID                  primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Name                string                `bson:"name" json:"name"`
	Description         string                `bson:"description" json:"description"`
	IsDefault           bool                  `bson:"is_default" json:"is_default"`
	Permissions         map[string][]string   `bson:"permissions" json:"-"` // Hidden from API responses - use granular permission system
	DiscordRoles        []DiscordRole         `bson:"discord_roles" json:"discord_roles"`
	AutoAssignmentRules *AutoAssignmentRules  `bson:"auto_assignment_rules,omitempty" json:"auto_assignment_rules,omitempty"`
	CreatedAt           time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time             `bson:"updated_at" json:"updated_at"`
	CreatedBy           int                   `bson:"created_by" json:"created_by"`
	MemberCount         int                   `bson:"member_count" json:"member_count"`
	IsMember            bool                  `bson:"-" json:"is_member"` // Runtime field, not stored
}

// DiscordRole represents a Discord role assignment for a group
type DiscordRole struct {
	ServerID   string `bson:"server_id" json:"server_id"`
	ServerName string `bson:"server_name,omitempty" json:"server_name,omitempty"`
	RoleName   string `bson:"role_name" json:"role_name"`
}

// AutoAssignmentRules defines rules for automatic group assignment
type AutoAssignmentRules struct {
	CorporationIDs      []int     `bson:"corporation_ids,omitempty" json:"corporation_ids,omitempty"`
	AllianceIDs         []int     `bson:"alliance_ids,omitempty" json:"alliance_ids,omitempty"`
	MinSecurityStatus   *float64  `bson:"min_security_status,omitempty" json:"min_security_status,omitempty"`
}

// GroupMembership represents a user's membership in a group
type GroupMembership struct {
	ID                 primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	CharacterID        int                    `bson:"character_id" json:"character_id"`
	GroupID            primitive.ObjectID     `bson:"group_id" json:"group_id"`
	AssignedAt         time.Time              `bson:"assigned_at" json:"assigned_at"`
	AssignedBy         int                    `bson:"assigned_by" json:"assigned_by"`
	ExpiresAt          *time.Time             `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	LastValidated      *time.Time             `bson:"last_validated,omitempty" json:"last_validated,omitempty"`
	ValidationStatus   string                 `bson:"validation_status" json:"validation_status"` // valid, invalid, pending
	AssignmentSource   string                 `bson:"assignment_source" json:"assignment_source"` // manual, auto_default, auto_corporation, auto_alliance
	AssignmentMetadata map[string]interface{} `bson:"assignment_metadata,omitempty" json:"assignment_metadata,omitempty"`
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	CharacterID int      `json:"character_id"`
	Allowed     bool     `json:"allowed"`
	Reason      string   `json:"reason,omitempty"`
	Groups      []string `json:"groups,omitempty"`
}

// Service represents a service in the granular permission system
type Service struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	DisplayName string             `bson:"display_name" json:"display_name"`
	Description string             `bson:"description" json:"description"`
	Resources   []ResourceConfig   `bson:"resources" json:"resources"`
	Enabled     bool               `bson:"enabled" json:"enabled"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ResourceConfig represents a resource configuration within a service
type ResourceConfig struct {
	Name        string   `bson:"name" json:"name" validate:"required"`
	DisplayName string   `bson:"display_name" json:"display_name" validate:"required"`
	Description string   `bson:"description" json:"description"`
	Actions     []string `bson:"actions" json:"actions" validate:"required,min=1"`
	Enabled     bool     `bson:"enabled" json:"enabled"`
}

// PermissionAssignment represents a granular permission assignment
type PermissionAssignment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Service     string             `bson:"service" json:"service"`
	Resource    string             `bson:"resource" json:"resource"`
	Action      string             `bson:"action" json:"action"`
	SubjectType string             `bson:"subject_type" json:"subject_type"` // group, member, corporation, alliance
	SubjectID   string             `bson:"subject_id" json:"subject_id"`
	GrantedBy   int                `bson:"granted_by" json:"granted_by"`
	GrantedAt   time.Time          `bson:"granted_at" json:"granted_at"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Reason      string             `bson:"reason" json:"reason"`
	Enabled     bool               `bson:"enabled" json:"enabled"`
}

// AuditLog represents an audit log entry for group operations
type AuditLog struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Action      string                 `bson:"action" json:"action"`
	GroupID     *primitive.ObjectID    `bson:"group_id,omitempty" json:"group_id,omitempty"`
	CharacterID *int                   `bson:"character_id,omitempty" json:"character_id,omitempty"`
	PerformedBy int                    `bson:"performed_by" json:"performed_by"`
	Details     map[string]interface{} `bson:"details,omitempty" json:"details,omitempty"`
	Reason      string                 `bson:"reason,omitempty" json:"reason,omitempty"`
	Timestamp   time.Time              `bson:"timestamp" json:"timestamp"`
	IPAddress   string                 `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent   string                 `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
}

// DefaultGroup represents a default group configuration
type DefaultGroup struct {
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	IsDefault           bool                 `json:"is_default"`
	Permissions         map[string][]string  `json:"permissions"`
	DiscordRoles        []DiscordRole        `json:"discord_roles"`
	AutoAssignmentRules *AutoAssignmentRules `json:"auto_assignment_rules,omitempty"`
}

// Permission represents a legacy permission
type Permission struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

// Subject represents a subject that can receive permissions
type Subject struct {
	Type string `json:"type"` // group, member, corporation, alliance
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ValidationError represents a validation error for group membership
type ValidationError struct {
	CharacterID int    `json:"character_id"`
	GroupID     string `json:"group_id"`
	Error       string `json:"error"`
	Details     string `json:"details,omitempty"`
}

// MembershipStats represents statistics about group membership
type MembershipStats struct {
	TotalMembers     int                    `json:"total_members"`
	ActiveMembers    int                    `json:"active_members"`
	ExpiredMembers   int                    `json:"expired_members"`
	GroupBreakdown   map[string]int         `json:"group_breakdown"`
	RecentJoins      int                    `json:"recent_joins"`
	RecentExpiries   int                    `json:"recent_expiries"`
	LastUpdated      time.Time              `json:"last_updated"`
}

// DiscordSyncStatus represents the status of Discord synchronization
type DiscordSyncStatus struct {
	LastSync      time.Time `json:"last_sync"`
	Status        string    `json:"status"` // success, failed, in_progress
	SyncedUsers   int       `json:"synced_users"`
	FailedUsers   int       `json:"failed_users"`
	SyncedServers int       `json:"synced_servers"`
	FailedServers int       `json:"failed_servers"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	NextSync      time.Time `json:"next_sync"`
}

// UserContext represents user context for permission checks
type UserContext struct {
	CharacterID       int      `json:"character_id"`
	CharacterName     string   `json:"character_name"`
	CorporationID     int      `json:"corporation_id"`
	AllianceID        int      `json:"alliance_id"`
	Groups            []string `json:"groups"`
	SecurityStatus    float64  `json:"security_status"`
	LastLogin         time.Time `json:"last_login"`
	IsAuthenticated   bool     `json:"is_authenticated"`
}

// PermissionContext represents context for permission evaluation
type PermissionContext struct {
	User        *UserContext `json:"user"`
	Service     string       `json:"service"`
	Resource    string       `json:"resource"`
	Action      string       `json:"action"`
	RequestTime time.Time    `json:"request_time"`
	IPAddress   string       `json:"ip_address,omitempty"`
	UserAgent   string       `json:"user_agent,omitempty"`
}

// GroupStatistics represents comprehensive group statistics
type GroupStatistics struct {
	TotalGroups       int                    `json:"total_groups"`
	DefaultGroups     int                    `json:"default_groups"`
	CustomGroups      int                    `json:"custom_groups"`
	TotalMembers      int                    `json:"total_members"`
	ActiveMembers     int                    `json:"active_members"`
	GroupDistribution map[string]int         `json:"group_distribution"`
	MembershipTrends  map[string]int         `json:"membership_trends"`
	LastCalculated    time.Time              `json:"last_calculated"`
}

// PermissionMatrix represents a complete permission matrix for a user
type PermissionMatrix struct {
	CharacterID         int                         `json:"character_id"`
	Groups              []string                    `json:"groups"`
	LegacyPermissions   map[string][]string         `json:"legacy_permissions"`   // resource -> actions
	GranularPermissions map[string][]string         `json:"granular_permissions"` // service.resource.action -> reasons
	ComputedAt          time.Time                   `json:"computed_at"`
	ExpiresAt           time.Time                   `json:"expires_at"`
}

// Constants for validation statuses
const (
	ValidationStatusValid   = "valid"
	ValidationStatusInvalid = "invalid"
	ValidationStatusPending = "pending"
)

// Constants for permission actions
const (
	ActionRead   = "read"
	ActionWrite  = "write"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
)

// Constants for subject types
const (
	SubjectTypeGroup       = "group"
	SubjectTypeMember      = "member"
	SubjectTypeCorporation = "corporation"
	SubjectTypeAlliance    = "alliance"
)

// Constants for audit actions
const (
	AuditActionGrantPermission  = "grant_permission"
	AuditActionRevokePermission = "revoke_permission"
	AuditActionCreateGroup      = "create_group"
	AuditActionUpdateGroup      = "update_group"
	AuditActionDeleteGroup      = "delete_group"
	AuditActionAddMember        = "add_member"
	AuditActionRemoveMember     = "remove_member"
	AuditActionCreateService    = "create_service"
	AuditActionUpdateService    = "update_service"
	AuditActionDeleteService    = "delete_service"
)

// Additional models needed by services

// UserPermissionSummary represents a summary of a user's permissions
type UserPermissionSummary struct {
	CharacterID   int                 `json:"character_id"`
	Groups        []string            `json:"groups"`
	Permissions   map[string][]string `json:"permissions"`
	IsSuperAdmin  bool                `json:"is_super_admin"`
	IsAdmin       bool                `json:"is_admin"`
	GroupCount    int                 `json:"group_count"`
	ResourceCount int                 `json:"resource_count"`
}

// GranularPermissionCheck represents a granular permission check request
type GranularPermissionCheck struct {
	CharacterID int    `json:"character_id"`
	Service     string `json:"service"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// PermissionAuditLog represents an audit log for permission operations
type PermissionAuditLog struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Action      string                 `bson:"action" json:"action"`
	Service     string                 `bson:"service" json:"service"`
	Resource    string                 `bson:"resource" json:"resource"`
	Permission  string                 `bson:"permission" json:"permission"`
	SubjectType string                 `bson:"subject_type" json:"subject_type"`
	SubjectID   string                 `bson:"subject_id" json:"subject_id"`
	PerformedBy int                    `bson:"performed_by" json:"performed_by"`
	PerformedAt time.Time              `bson:"performed_at" json:"performed_at"`
	Reason      string                 `bson:"reason" json:"reason"`
	OldValues   map[string]interface{} `bson:"old_values,omitempty" json:"old_values,omitempty"`
	NewValues   map[string]interface{} `bson:"new_values,omitempty" json:"new_values,omitempty"`
}