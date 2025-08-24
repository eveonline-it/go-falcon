package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SettingType represents the data type of a site setting value
type SettingType string

const (
	SettingTypeString  SettingType = "string"
	SettingTypeNumber  SettingType = "number"
	SettingTypeBoolean SettingType = "boolean"
	SettingTypeObject  SettingType = "object"
)

// SiteSetting represents a site configuration setting
type SiteSetting struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key         string             `bson:"key" json:"key"`                           // Unique setting identifier
	Value       interface{}        `bson:"value" json:"value"`                       // Setting value (can be any type)
	Type        SettingType        `bson:"type" json:"type"`                         // Data type for validation
	Category    string             `bson:"category,omitempty" json:"category"`       // Organization category
	Description string             `bson:"description,omitempty" json:"description"` // Human-readable description
	IsPublic    bool               `bson:"is_public" json:"is_public"`               // Whether non-admins can read this
	IsActive    bool               `bson:"is_active" json:"is_active"`               // Whether this setting is active
	CreatedBy   *int64             `bson:"created_by,omitempty" json:"created_by"`   // Character ID who created
	UpdatedBy   *int64             `bson:"updated_by,omitempty" json:"updated_by"`   // Character ID who last updated
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// DefaultSiteSettings contains predefined settings that should be created on initialization
var DefaultSiteSettings = []SiteSetting{
	{
		Key:         "site_name",
		Value:       "Go Falcon API Gateway",
		Type:        SettingTypeString,
		Category:    "general",
		Description: "The name of the site displayed in the UI",
		IsPublic:    true,
		IsActive:    true,
	},
	{
		Key:         "maintenance_mode",
		Value:       false,
		Type:        SettingTypeBoolean,
		Category:    "system",
		Description: "Whether the site is in maintenance mode",
		IsPublic:    true,
		IsActive:    true,
	},
	{
		Key:         "max_users",
		Value:       1000,
		Type:        SettingTypeNumber,
		Category:    "system",
		Description: "Maximum number of registered users allowed",
		IsPublic:    false,
		IsActive:    true,
	},
	{
		Key:         "api_rate_limit",
		Value:       100,
		Type:        SettingTypeNumber,
		Category:    "api",
		Description: "API rate limit per minute per user",
		IsPublic:    false,
		IsActive:    true,
	},
	{
		Key:         "registration_enabled",
		Value:       true,
		Type:        SettingTypeBoolean,
		Category:    "auth",
		Description: "Whether new user registration is enabled",
		IsPublic:    true,
		IsActive:    true,
	},
	{
		Key: "contact_info",
		Value: map[string]interface{}{
			"email":   "admin@example.com",
			"discord": "https://discord.gg/example",
			"website": "https://example.com",
		},
		Type:        SettingTypeObject,
		Category:    "general",
		Description: "Contact information for site administrators",
		IsPublic:    true,
		IsActive:    true,
	},
}

// SettingCategories contains valid categories for organization
var SettingCategories = []string{
	"general",
	"system",
	"auth",
	"api",
	"eve",
	"notifications",
	"security",
	"ui",
}

// ManagedCorporation represents a managed corporation in the database
type ManagedCorporation struct {
	CorporationID int64     `bson:"corporation_id" json:"corporation_id"`
	Name          string    `bson:"name" json:"name"`
	Ticker        string    `bson:"ticker" json:"ticker"` // NEW: Corporation ticker
	Enabled       bool      `bson:"enabled" json:"enabled"`
	Position      int       `bson:"position" json:"position"`
	AddedAt       time.Time `bson:"added_at" json:"added_at"`
	AddedBy       *int64    `bson:"added_by,omitempty" json:"added_by,omitempty"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
	UpdatedBy     *int64    `bson:"updated_by,omitempty" json:"updated_by,omitempty"`
}

// ManagedCorporationsValue represents the value structure for managed_corporations setting
type ManagedCorporationsValue struct {
	Corporations []ManagedCorporation `bson:"corporations" json:"corporations"`
}

// ManagedAlliance represents a managed alliance in the database
type ManagedAlliance struct {
	AllianceID int64     `bson:"alliance_id" json:"alliance_id"`
	Name       string    `bson:"name" json:"name"`
	Ticker     string    `bson:"ticker" json:"ticker"` // NEW: Alliance ticker
	Enabled    bool      `bson:"enabled" json:"enabled"`
	Position   int       `bson:"position" json:"position"`
	AddedAt    time.Time `bson:"added_at" json:"added_at"`
	AddedBy    *int64    `bson:"added_by,omitempty" json:"added_by,omitempty"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
	UpdatedBy  *int64    `bson:"updated_by,omitempty" json:"updated_by,omitempty"`
}

// ManagedAlliancesValue represents the value structure for managed_alliances setting
type ManagedAlliancesValue struct {
	Alliances []ManagedAlliance `bson:"alliances" json:"alliances"`
}

// Collection name
const (
	SiteSettingsCollection = "site_settings"
)
