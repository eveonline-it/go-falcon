package permissions

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Permission represents a specific permission that can be granted to groups
type Permission struct {
	ID          string    `json:"id" bson:"_id"`                  // e.g., "intel:reports:write"
	Service     string    `json:"service" bson:"service"`         // e.g., "intel", "scheduler"
	Resource    string    `json:"resource" bson:"resource"`       // e.g., "reports", "tasks"
	Action      string    `json:"action" bson:"action"`           // e.g., "write", "read", "create"
	IsStatic    bool      `json:"is_static" bson:"is_static"`     // true = hardcoded, false = configurable
	Name        string    `json:"name" bson:"name"`               // Human-readable name
	Description string    `json:"description" bson:"description"` // What this permission allows
	Category    string    `json:"category" bson:"category"`       // Grouping for UI (e.g., "System", "Fleet Management")
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

// GroupPermission represents the assignment of a permission to a group
type GroupPermission struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	GroupID      primitive.ObjectID `json:"group_id" bson:"group_id"`                         // Reference to groups collection
	PermissionID string             `json:"permission_id" bson:"permission_id"`               // Permission ID
	GrantedBy    *int64             `json:"granted_by,omitempty" bson:"granted_by,omitempty"` // Character ID who granted
	GrantedAt    time.Time          `json:"granted_at" bson:"granted_at"`
	IsActive     bool               `json:"is_active" bson:"is_active"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

// PermissionCheck represents the result of a permission check
type PermissionCheck struct {
	CharacterID  int64  `json:"character_id"`
	PermissionID string `json:"permission_id"`
	Granted      bool   `json:"granted"`
	GrantedVia   string `json:"granted_via,omitempty"` // Which group granted the permission
}

// PermissionCategory defines UI groupings for permissions
type PermissionCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Order       int    `json:"order"` // Display order in UI
}
