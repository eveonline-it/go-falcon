package groups

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service represents a system service/module that can have permissions
type Service struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`                               // e.g., "sde", "auth", "scheduler"
	DisplayName string             `bson:"display_name" json:"display_name"`               // e.g., "Static Data Export"
	Description string             `bson:"description" json:"description"`                 // Human-readable description
	Resources   []Resource         `bson:"resources" json:"resources"`                     // Available resources in this service
	Enabled     bool               `bson:"enabled" json:"enabled"`                         // Whether this service accepts permissions
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Resource represents a specific resource within a service
type Resource struct {
	Name        string   `bson:"name" json:"name"`                 // e.g., "entities", "users", "tasks"
	DisplayName string   `bson:"display_name" json:"display_name"` // e.g., "SDE Entities", "User Profiles"
	Description string   `bson:"description" json:"description"`   // What this resource represents
	Actions     []string `bson:"actions" json:"actions"`           // Available actions: ["read", "write", "delete", "admin"]
	Enabled     bool     `bson:"enabled" json:"enabled"`           // Whether this resource accepts permissions
}

// PermissionAssignment represents a specific permission granted to a subject
type PermissionAssignment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Service     string             `bson:"service" json:"service"`         // Service name (e.g., "sde")
	Resource    string             `bson:"resource" json:"resource"`       // Resource name (e.g., "entities")
	Action      string             `bson:"action" json:"action"`           // Action (e.g., "read", "write", "delete", "admin")
	SubjectType string             `bson:"subject_type" json:"subject_type"` // "group", "member", "corporation", "alliance"
	SubjectID   string             `bson:"subject_id" json:"subject_id"`   // ID of the subject (group ObjectID, character ID, corp ID, alliance ID)
	GrantedBy   int                `bson:"granted_by" json:"granted_by"`   // Character ID of admin who granted this
	GrantedAt   time.Time          `bson:"granted_at" json:"granted_at"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"` // Optional expiration
	Reason      string             `bson:"reason,omitempty" json:"reason,omitempty"`         // Optional reason for granting
	Enabled     bool               `bson:"enabled" json:"enabled"`                           // Whether this assignment is active
}

// GranularPermissionCheck represents a permission check request for the new system
type GranularPermissionCheck struct {
	Service     string `json:"service"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	CharacterID int    `json:"character_id"`
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Allowed       bool     `json:"allowed"`
	Service       string   `json:"service"`
	Resource      string   `json:"resource"`
	Action        string   `json:"action"`
	CharacterID   int      `json:"character_id"`
	GrantedThrough []string `json:"granted_through"` // How permission was granted: ["group:corporate", "member:direct"]
	CheckedAt     time.Time `json:"checked_at"`
}

// SubjectInfo represents detailed information about a permission subject
type SubjectInfo struct {
	Type        string `json:"type"`         // "group", "member", "corporation", "alliance"
	ID          string `json:"id"`           // The subject ID
	Name        string `json:"name"`         // Display name
	Description string `json:"description"`  // Additional info
}

// Request/Response types for API

// CreateServiceRequest represents a request to create a new service
type CreateServiceRequest struct {
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	Resources   []Resource `json:"resources"`
}

func (r *CreateServiceRequest) Validate() error {
	if r.Name == "" {
		return NewValidationError("service name is required")
	}
	if r.DisplayName == "" {
		return NewValidationError("service display name is required")
	}
	if len(r.Resources) == 0 {
		return NewValidationError("at least one resource is required")
	}
	
	// Validate resources
	for i, resource := range r.Resources {
		if resource.Name == "" {
			return NewValidationError("resource name is required for resource %d", i)
		}
		if resource.DisplayName == "" {
			return NewValidationError("resource display name is required for resource %d", i)
		}
		if len(resource.Actions) == 0 {
			return NewValidationError("at least one action is required for resource %d", i)
		}
		
		// Validate actions
		validActions := map[string]bool{"read": true, "write": true, "delete": true, "admin": true}
		for _, action := range resource.Actions {
			if !validActions[action] {
				return NewValidationError("invalid action '%s' for resource %d", action, i)
			}
		}
	}
	
	return nil
}

// UpdateServiceRequest represents a request to update a service
type UpdateServiceRequest struct {
	DisplayName *string     `json:"display_name,omitempty"`
	Description *string     `json:"description,omitempty"`
	Resources   []Resource  `json:"resources,omitempty"`
	Enabled     *bool       `json:"enabled,omitempty"`
}

// CreatePermissionRequest represents a request to grant a permission
type CreatePermissionRequest struct {
	Service     string     `json:"service"`
	Resource    string     `json:"resource"`
	Action      string     `json:"action"`
	SubjectType string     `json:"subject_type"` // "group", "member", "corporation", "alliance"
	SubjectID   string     `json:"subject_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

func (r *CreatePermissionRequest) Validate() error {
	if r.Service == "" {
		return NewValidationError("service is required")
	}
	if r.Resource == "" {
		return NewValidationError("resource is required")
	}
	if r.Action == "" {
		return NewValidationError("action is required")
	}
	if r.SubjectType == "" {
		return NewValidationError("subject_type is required")
	}
	if r.SubjectID == "" {
		return NewValidationError("subject_id is required")
	}
	
	// Validate subject type
	validSubjectTypes := map[string]bool{"group": true, "member": true, "corporation": true, "alliance": true}
	if !validSubjectTypes[r.SubjectType] {
		return NewValidationError("invalid subject_type '%s'", r.SubjectType)
	}
	
	// Validate action
	validActions := map[string]bool{"read": true, "write": true, "delete": true, "admin": true}
	if !validActions[r.Action] {
		return NewValidationError("invalid action '%s'", r.Action)
	}
	
	return nil
}

// BulkPermissionRequest represents a request to grant multiple permissions
type BulkPermissionRequest struct {
	Permissions []CreatePermissionRequest `json:"permissions"`
	Reason      string                    `json:"reason,omitempty"`
}

func (r *BulkPermissionRequest) Validate() error {
	if len(r.Permissions) == 0 {
		return NewValidationError("at least one permission is required")
	}
	
	for i, perm := range r.Permissions {
		if err := perm.Validate(); err != nil {
			return NewValidationError("permission %d: %s", i, err.Error())
		}
	}
	
	return nil
}

// PermissionQuery represents a query for permissions
type PermissionQuery struct {
	Service     string `json:"service,omitempty"`
	Resource    string `json:"resource,omitempty"`
	Action      string `json:"action,omitempty"`
	SubjectType string `json:"subject_type,omitempty"`
	SubjectID   string `json:"subject_id,omitempty"`
	GrantedBy   int    `json:"granted_by,omitempty"`
	IncludeExpired bool `json:"include_expired,omitempty"`
}

// PermissionSummary represents a summary of permissions for a subject
type PermissionSummary struct {
	SubjectType   string                          `json:"subject_type"`
	SubjectID     string                          `json:"subject_id"`
	SubjectName   string                          `json:"subject_name"`
	Permissions   map[string]ServicePermissions   `json:"permissions"` // service -> ServicePermissions
	TotalCount    int                             `json:"total_count"`
	LastUpdated   time.Time                       `json:"last_updated"`
}

// ServicePermissions represents permissions for a specific service
type ServicePermissions struct {
	Service   string                        `json:"service"`
	Resources map[string]ResourcePermissions `json:"resources"` // resource -> ResourcePermissions
}

// ResourcePermissions represents permissions for a specific resource
type ResourcePermissions struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(format string, args ...interface{}) *ValidationError {
	return &ValidationError{
		Message: fmt.Sprintf(format, args...),
	}
}

// PermissionAuditLog represents an audit entry for permission changes
type PermissionAuditLog struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Action       string             `bson:"action" json:"action"`             // "grant", "revoke", "update"
	Service      string             `bson:"service" json:"service"`
	Resource     string             `bson:"resource" json:"resource"`
	Permission   string             `bson:"permission" json:"permission"`     // The action that was granted/revoked
	SubjectType  string             `bson:"subject_type" json:"subject_type"`
	SubjectID    string             `bson:"subject_id" json:"subject_id"`
	PerformedBy  int                `bson:"performed_by" json:"performed_by"` // Character ID
	PerformedAt  time.Time          `bson:"performed_at" json:"performed_at"`
	Reason       string             `bson:"reason,omitempty" json:"reason,omitempty"`
	OldValues    map[string]any     `bson:"old_values,omitempty" json:"old_values,omitempty"`
	NewValues    map[string]any     `bson:"new_values,omitempty" json:"new_values,omitempty"`
}