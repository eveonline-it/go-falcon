package middleware

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CasbinRule represents a policy rule in the Casbin system
type CasbinRule struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	PType string             `bson:"ptype" json:"ptype"` // Policy type: "p" for policy, "g" for role
	V0    string             `bson:"v0" json:"v0"`       // Subject (user, role, etc.)
	V1    string             `bson:"v1" json:"v1"`       // Object/Resource or Role (for role assignments)
	V2    string             `bson:"v2" json:"v2"`       // Action or Domain (for role assignments)
	V3    string             `bson:"v3" json:"v3"`       // Domain or empty
	V4    string             `bson:"v4" json:"v4"`       // Effect (allow/deny) or empty
	V5    string             `bson:"v5" json:"v5"`       // Reserved for future use
}

// PermissionHierarchy tracks EVE entity relationships for permission inheritance
type PermissionHierarchy struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CharacterID   int64              `bson:"character_id" json:"character_id"`
	CharacterName string             `bson:"character_name" json:"character_name"`
	CorporationID int64              `bson:"corporation_id" json:"corporation_id"`
	CorporationName string           `bson:"corporation_name" json:"corporation_name"`
	AllianceID    int64              `bson:"alliance_id,omitempty" json:"alliance_id,omitempty"`
	AllianceName  string             `bson:"alliance_name,omitempty" json:"alliance_name,omitempty"`
	UserID        string             `bson:"user_id" json:"user_id"`
	IsPrimary     bool               `bson:"is_primary" json:"is_primary"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
	LastSyncAt    time.Time          `bson:"last_sync_at" json:"last_sync_at"`
}

// RoleAssignment tracks role assignments with metadata
type RoleAssignment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	RoleName    string             `bson:"role_name" json:"role_name"`
	SubjectType string             `bson:"subject_type" json:"subject_type"` // user, character, corporation, alliance
	SubjectID   string             `bson:"subject_id" json:"subject_id"`     // actual ID as string
	SubjectName string             `bson:"subject_name" json:"subject_name"` // display name
	Domain      string             `bson:"domain" json:"domain"`             // global, or specific domain
	GrantedBy   int64              `bson:"granted_by" json:"granted_by"`     // character_id who granted
	GrantedByName string           `bson:"granted_by_name" json:"granted_by_name"`
	GrantedAt   time.Time          `bson:"granted_at" json:"granted_at"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Reason      string             `bson:"reason,omitempty" json:"reason,omitempty"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
}

// PermissionPolicy represents a permission policy with metadata
type PermissionPolicy struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	SubjectType string             `bson:"subject_type" json:"subject_type"` // user, character, corporation, alliance, role
	SubjectID   string             `bson:"subject_id" json:"subject_id"`     // actual ID as string
	SubjectName string             `bson:"subject_name" json:"subject_name"` // display name
	Resource    string             `bson:"resource" json:"resource"`         // e.g., "scheduler.tasks"
	Action      string             `bson:"action" json:"action"`             // read, write, delete, admin
	Domain      string             `bson:"domain" json:"domain"`             // global, or specific domain
	Effect      string             `bson:"effect" json:"effect"`             // allow, deny
	CreatedBy   int64              `bson:"created_by" json:"created_by"`     // character_id who created
	CreatedByName string           `bson:"created_by_name" json:"created_by_name"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	Reason      string             `bson:"reason,omitempty" json:"reason,omitempty"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
}

// PermissionAuditLog represents an audit log entry for permission operations
type PermissionAuditLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Operation   string             `bson:"operation" json:"operation"`       // grant, revoke, check, sync
	OperationType string           `bson:"operation_type" json:"operation_type"` // policy, role, hierarchy
	SubjectType string             `bson:"subject_type" json:"subject_type"` // user, character, corporation, alliance, role
	SubjectID   string             `bson:"subject_id" json:"subject_id"`     // actual ID as string
	SubjectName string             `bson:"subject_name" json:"subject_name"` // display name
	TargetType  string             `bson:"target_type,omitempty" json:"target_type,omitempty"` // For role grants: user, character, etc.
	TargetID    string             `bson:"target_id,omitempty" json:"target_id,omitempty"`     // For role grants: target ID
	TargetName  string             `bson:"target_name,omitempty" json:"target_name,omitempty"` // For role grants: target name
	Resource    string             `bson:"resource,omitempty" json:"resource,omitempty"`       // For permission checks
	Action      string             `bson:"action,omitempty" json:"action,omitempty"`           // For permission checks
	Domain      string             `bson:"domain" json:"domain"`                               // global, or specific domain
	Effect      string             `bson:"effect,omitempty" json:"effect,omitempty"`           // allow, deny (for policies)
	Result      *bool              `bson:"result,omitempty" json:"result,omitempty"`           // true/false for permission checks
	PerformedBy int64              `bson:"performed_by" json:"performed_by"`                   // character_id who performed operation
	PerformedByName string         `bson:"performed_by_name" json:"performed_by_name"`
	IPAddress   string             `bson:"ip_address" json:"ip_address"`
	UserAgent   string             `bson:"user_agent" json:"user_agent"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Details     map[string]interface{} `bson:"details,omitempty" json:"details,omitempty"` // Additional context
	Error       string             `bson:"error,omitempty" json:"error,omitempty"`         // Error message if operation failed
}

// PermissionCheckRequest represents a request to check permissions
type PermissionCheckRequest struct {
	UserID   string `json:"user_id"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Domain   string `json:"domain,omitempty"`
}

// PermissionCheckResponse represents the response of a permission check
type PermissionCheckResponse struct {
	Allowed    bool     `json:"allowed"`
	Reason     string   `json:"reason,omitempty"`
	MatchedBy  []string `json:"matched_by,omitempty"` // Which subjects granted the permission
	DeniedBy   []string `json:"denied_by,omitempty"`  // Which subjects denied the permission
	CheckedAt  time.Time `json:"checked_at"`
}

// BatchPermissionCheckRequest represents a batch permission check request
type BatchPermissionCheckRequest struct {
	UserID      string                        `json:"user_id"`
	Permissions []PermissionCheckRequest      `json:"permissions"`
}

// BatchPermissionCheckResponse represents the response of a batch permission check
type BatchPermissionCheckResponse struct {
	Results   map[string]PermissionCheckResponse `json:"results"` // Keyed by "resource.action"
	CheckedAt time.Time                          `json:"checked_at"`
}

// PolicyCreateRequest represents a request to create a new policy
type PolicyCreateRequest struct {
	SubjectType string     `json:"subject_type" validate:"required,oneof=user character corporation alliance role"`
	SubjectID   string     `json:"subject_id" validate:"required"`
	Resource    string     `json:"resource" validate:"required"`
	Action      string     `json:"action" validate:"required,oneof=read write delete admin"`
	Effect      string     `json:"effect" validate:"required,oneof=allow deny"`
	Domain      string     `json:"domain,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

// RoleCreateRequest represents a request to create a role assignment
type RoleCreateRequest struct {
	RoleName    string     `json:"role_name" validate:"required"`
	SubjectType string     `json:"subject_type" validate:"required,oneof=user character corporation alliance"`
	SubjectID   string     `json:"subject_id" validate:"required"`
	Domain      string     `json:"domain,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

// GetEffectivePermissionsResponse represents effective permissions for a user
type GetEffectivePermissionsResponse struct {
	UserID      string                    `json:"user_id"`
	Characters  []PermissionSubjectInfo   `json:"characters"`
	Corporations []PermissionSubjectInfo  `json:"corporations"`
	Alliances   []PermissionSubjectInfo   `json:"alliances"`
	Roles       []string                  `json:"roles"`
	Policies    []PermissionPolicyInfo    `json:"policies"`
	CheckedAt   time.Time                 `json:"checked_at"`
}

// PermissionSubjectInfo represents information about a permission subject
type PermissionSubjectInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Roles      []string `json:"roles"`
	DirectPolicies []PermissionPolicyInfo `json:"direct_policies"`
}

// PermissionPolicyInfo represents information about a specific policy
type PermissionPolicyInfo struct {
	Resource  string `json:"resource"`
	Action    string `json:"action"`
	Effect    string `json:"effect"`
	Domain    string `json:"domain"`
	Source    string `json:"source"`    // Which subject granted this
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}