package dto

import (
	"time"

	"go-falcon/internal/groups/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GroupCreateRequest represents a request to create a new group
type GroupCreateRequest struct {
	Name                string                       `json:"name" validate:"required,min=3,max=50,alphanum"`
	Description         string                       `json:"description" validate:"required,min=10,max=500"`
	IsDefault           bool                         `json:"is_default"`
	DiscordRoles        []models.DiscordRole         `json:"discord_roles,omitempty"`
	AutoAssignmentRules *models.AutoAssignmentRules  `json:"auto_assignment_rules,omitempty"`
}

// GroupUpdateRequest represents a request to update an existing group
type GroupUpdateRequest struct {
	Name                *string                      `json:"name,omitempty" validate:"omitempty,min=3,max=50,alphanum"`
	Description         *string                      `json:"description,omitempty" validate:"omitempty,min=10,max=500"`
	IsDefault           *bool                        `json:"is_default,omitempty"`
	DiscordRoles        []models.DiscordRole         `json:"discord_roles,omitempty"`
	AutoAssignmentRules *models.AutoAssignmentRules  `json:"auto_assignment_rules,omitempty"`
}

// MembershipRequest represents a request to add/remove group membership
type MembershipRequest struct {
	CharacterID        int                    `json:"character_id" validate:"required,min=1"`
	ExpiresAt          *time.Time             `json:"expires_at,omitempty"`
	Reason             string                 `json:"reason,omitempty"`
	AssignmentMetadata map[string]interface{} `json:"assignment_metadata,omitempty"`
}

// BulkMembershipRequest represents a request for bulk membership operations
type BulkMembershipRequest struct {
	CharacterIDs []int      `json:"character_ids" validate:"required,min=1,dive,min=1"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// PermissionCheckRequest represents a request to check permissions
type PermissionCheckRequest struct {
	Resource string   `json:"resource" validate:"required"`
	Actions  []string `json:"actions" validate:"required,min=1"`
}

// ServiceCreateRequest represents a request to create a new service in granular permission system
type ServiceCreateRequest struct {
	Name        string                  `json:"name" validate:"required,min=2,max=50,alphanum"`
	DisplayName string                  `json:"display_name" validate:"required,min=3,max=100"`
	Description string                  `json:"description" validate:"required,min=10,max=500"`
	Resources   []models.ResourceConfig `json:"resources" validate:"required,min=1,dive"`
}

// ServiceUpdateRequest represents a request to update an existing service
type ServiceUpdateRequest struct {
	DisplayName *string                 `json:"display_name,omitempty" validate:"omitempty,min=3,max=100"`
	Description *string                 `json:"description,omitempty" validate:"omitempty,min=10,max=500"`
	Resources   []models.ResourceConfig `json:"resources,omitempty" validate:"omitempty,min=1,dive"`
	Enabled     *bool                   `json:"enabled,omitempty"`
}

// PermissionAssignmentRequest represents a request to assign granular permissions
type PermissionAssignmentRequest struct {
	Service     string    `json:"service" validate:"required"`
	Resource    string    `json:"resource" validate:"required"`
	Action      string    `json:"action" validate:"required,oneof=read write delete admin"`
	SubjectType string    `json:"subject_type" validate:"required,oneof=group member corporation alliance"`
	SubjectID   string    `json:"subject_id" validate:"required"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Reason      string    `json:"reason" validate:"required,min=10,max=500"`
}

// PermissionCheckGranularRequest represents a request to check granular permissions
type PermissionCheckGranularRequest struct {
	Service     string `json:"service" validate:"required"`
	Resource    string `json:"resource" validate:"required"`
	Action      string `json:"action" validate:"required"`
	CharacterID int    `json:"character_id" validate:"required,min=1"`
}

// GroupListQuery represents query parameters for listing groups
type GroupListQuery struct {
	Page        int    `query:"page" validate:"min=1"`
	PageSize    int    `query:"page_size" validate:"min=1,max=100"`
	IsDefault   *bool  `query:"is_default"`
	Search      string `query:"search"`
	ShowMembers bool   `query:"show_members"`
}

// MemberListQuery represents query parameters for listing group members
type MemberListQuery struct {
	Page     int `query:"page" validate:"min=1"`
	PageSize int `query:"page_size" validate:"min=1,max=100"`
}

// AuditLogQuery represents query parameters for audit logs
type AuditLogQuery struct {
	Page      int        `query:"page" validate:"min=1"`
	PageSize  int        `query:"page_size" validate:"min=1,max=100"`
	Service   string     `query:"service"`
	Action    string     `query:"action"`
	SubjectID string     `query:"subject_id"`
	StartDate *time.Time `query:"start_date"`
	EndDate   *time.Time `query:"end_date"`
}

// SubjectValidationRequest represents a request to validate a subject
type SubjectValidationRequest struct {
	Type string `json:"type" validate:"required,oneof=group member corporation alliance"`
	ID   string `json:"id" validate:"required"`
}

// DiscordRoleCreateRequest represents a request to create/update Discord roles for a group
type DiscordRoleCreateRequest struct {
	ServerID   string `json:"server_id" validate:"required"`
	ServerName string `json:"server_name,omitempty"`
	RoleName   string `json:"role_name" validate:"required"`
}

// BulkPermissionRequest represents a request for bulk permission operations
type BulkPermissionRequest struct {
	Assignments []PermissionAssignmentRequest `json:"assignments" validate:"required,min=1,dive"`
}

// PermissionRevocationRequest represents a request to revoke permissions
type PermissionRevocationRequest struct {
	Service     string `query:"service" validate:"required"`
	Resource    string `query:"resource" validate:"required"`
	Action      string `query:"action" validate:"required"`
	SubjectType string `query:"subject_type" validate:"required"`
	SubjectID   string `query:"subject_id" validate:"required"`
}

// GroupMemberSearchQuery represents query parameters for searching group members
type GroupMemberSearchQuery struct {
	GroupID      primitive.ObjectID `query:"group_id"`
	CharacterID  *int              `query:"character_id"`
	IsActive     *bool             `query:"is_active"`
	HasExpired   *bool             `query:"has_expired"`
	Page         int               `query:"page" validate:"min=1"`
	PageSize     int               `query:"page_size" validate:"min=1,max=100"`
}