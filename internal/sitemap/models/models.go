package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RouteType represents the type of route
type RouteType string

const (
	RouteTypePublic    RouteType = "public"    // No auth required
	RouteTypeAuth      RouteType = "auth"      // Authentication required
	RouteTypeProtected RouteType = "protected" // Specific permissions required
	RouteTypeAdmin     RouteType = "admin"     // Admin-only routes
)

// NavigationPosition represents where the route appears in navigation
type NavigationPosition string

const (
	NavMain   NavigationPosition = "main"   // Main navigation
	NavUser   NavigationPosition = "user"   // User dropdown
	NavAdmin  NavigationPosition = "admin"  // Admin menu
	NavFooter NavigationPosition = "footer" // Footer links
	NavHidden NavigationPosition = "hidden" // Not in navigation
)

// Route represents a frontend route configuration
type Route struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RouteID   string             `bson:"route_id" json:"route_id"`   // Unique identifier for frontend
	Path      string             `bson:"path" json:"path"`           // React Router path
	Component string             `bson:"component" json:"component"` // Component name to render
	Name      string             `bson:"name" json:"name"`           // Display name
	Icon      *string            `bson:"icon,omitempty" json:"icon"` // Icon identifier (FontAwesome)
	Type      RouteType          `bson:"type" json:"type"`
	ParentID  *string            `bson:"parent_id,omitempty" json:"parent_id"` // For nested routes

	// Navigation
	NavPosition NavigationPosition `bson:"nav_position" json:"nav_position"`
	NavOrder    int                `bson:"nav_order" json:"nav_order"` // Sort order in navigation
	ShowInNav   bool               `bson:"show_in_nav" json:"show_in_nav"`

	// Permissions (uses existing permission system)
	RequiredPermissions []string `bson:"required_permissions" json:"required_permissions"` // AND logic
	RequiredGroups      []string `bson:"required_groups,omitempty" json:"required_groups"` // OR logic

	// Metadata
	Title       string   `bson:"title" json:"title"` // Page title
	Description *string  `bson:"description,omitempty" json:"description"`
	Keywords    []string `bson:"keywords,omitempty" json:"keywords"` // For search
	Group       *string  `bson:"group,omitempty" json:"group"`       // Grouping (dashboard, app, etc.)

	// Feature flags
	FeatureFlags []string `bson:"feature_flags,omitempty" json:"feature_flags"`
	IsEnabled    bool     `bson:"is_enabled" json:"is_enabled"`

	// React-specific
	Props    map[string]interface{} `bson:"props,omitempty" json:"props"`   // Default props for component
	LazyLoad bool                   `bson:"lazy_load" json:"lazy_load"`     // Code splitting
	Exact    bool                   `bson:"exact,omitempty" json:"exact"`   // Exact path matching
	NewTab   bool                   `bson:"newtab,omitempty" json:"newtab"` // Open in new tab

	// Badge (for navigation items)
	BadgeType *string `bson:"badge_type,omitempty" json:"badge_type,omitempty"` // Badge type (success, warning, etc.)
	BadgeText *string `bson:"badge_text,omitempty" json:"badge_text,omitempty"` // Badge text

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// NavigationGroup represents a group of navigation items
type NavigationGroup struct {
	Label        string    `json:"label"`
	LabelDisable bool      `json:"labelDisable,omitempty"`
	Icon         *string   `json:"icon,omitempty"`
	Items        []NavItem `json:"children"` // Using "children" to match frontend
}

// NavItem represents a navigation item for frontend consumption
type NavItem struct {
	RouteID  string      `json:"routeId,omitempty"`
	Name     string      `json:"name"`
	To       string      `json:"to,omitempty"`
	Icon     interface{} `json:"icon,omitempty"` // Can be string or array of strings
	Active   bool        `json:"active,omitempty"`
	Exact    bool        `json:"exact,omitempty"`
	NewTab   bool        `json:"newtab,omitempty"`
	Badge    *Badge      `json:"badge,omitempty"`
	Children []NavItem   `json:"children,omitempty"`
}

// Badge represents a badge on a navigation item
type Badge struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SitemapResponse for frontend consumption
type SitemapResponse struct {
	Routes          []RouteConfig     `json:"routes"`
	Navigation      []NavigationGroup `json:"navigation"`
	UserPermissions []string          `json:"userPermissions"`
	UserGroups      []string          `json:"userGroups"`
	Features        map[string]bool   `json:"features,omitempty"`
}

// RouteConfig for frontend consumption
type RouteConfig struct {
	ID          string                 `json:"id"`
	Path        string                 `json:"path"`
	Component   string                 `json:"component"`
	Name        string                 `json:"name"`
	Icon        *string                `json:"icon,omitempty"`
	Title       string                 `json:"title"`
	Permissions []string               `json:"permissions,omitempty"`
	Meta        *RouteMeta             `json:"meta,omitempty"`
	Children    []RouteConfig          `json:"children,omitempty"`
	Props       map[string]interface{} `json:"props,omitempty"`
	LazyLoad    bool                   `json:"lazyLoad"`
	Accessible  bool                   `json:"accessible"` // Based on user permissions
}

// RouteMeta contains metadata for a route
type RouteMeta struct {
	Title       string  `json:"title"`
	Icon        *string `json:"icon,omitempty"`
	Group       *string `json:"group,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Collection names
const (
	RoutesCollection = "routes"
)

// Default route groups for organization
var RouteGroups = []string{
	"dashboard",
	"app",
	"pages",
	"modules",
	"user",
	"admin",
	"documentation",
}
