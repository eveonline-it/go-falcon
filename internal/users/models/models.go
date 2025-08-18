package models

import (
	"time"
)

// User represents a user with character information and state control
// Uses the same database collection as auth module: user_profiles
type User struct {
	CharacterID   int       `json:"character_id" bson:"character_id"`   // EVE character ID (unique)
	UserID        string    `json:"user_id" bson:"user_id"`             // UUID for internal identification
	AccessToken   string    `json:"-" bson:"access_token"`              // EVE SSO access token (hidden from JSON)
	RefreshToken  string    `json:"-" bson:"refresh_token"`             // EVE SSO refresh token (hidden from JSON)
	Enabled       bool      `json:"enabled" bson:"enabled"`             // User account status
	Banned        bool      `json:"banned" bson:"banned"`               // Ban status
	Invalid       bool      `json:"invalid" bson:"invalid"`             // Token/account validity
	Scopes        string    `json:"scopes" bson:"scopes"`               // EVE Online permissions
	Position      int       `json:"position" bson:"position"`           // User position/rank
	Notes         string    `json:"notes" bson:"notes"`                 // Administrative notes
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`       // Registration timestamp
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`       // Last update timestamp
	LastLogin     time.Time `json:"last_login" bson:"last_login"`       // Last login timestamp
	CharacterName string    `json:"character_name" bson:"character_name"` // EVE character name
	Valid         bool      `json:"valid" bson:"valid"`                 // Character profile validity status
}

// CharacterSummary represents basic character information for listing
type CharacterSummary struct {
	CharacterID   int        `json:"character_id" bson:"character_id"`
	CharacterName string     `json:"character_name" bson:"character_name"`
	UserID        string     `json:"user_id" bson:"user_id"`
	Enabled       bool       `json:"enabled" bson:"enabled"`
	Banned        bool       `json:"banned" bson:"banned"`
	Position      int        `json:"position" bson:"position"`
	LastLogin     *time.Time `json:"last_login,omitempty" bson:"last_login,omitempty"`
}

// CollectionName returns the MongoDB collection name for users
func (User) CollectionName() string {
	return "user_profiles"
}

// CollectionName returns the MongoDB collection name for character summaries
func (CharacterSummary) CollectionName() string {
	return "user_profiles"
}

// Character Management Models (Phase 2: Character Resolution System)

// UserWithCharacters represents a user with all their associated characters
type UserWithCharacters struct {
	ID         string           `json:"id"`
	Characters []UserCharacter  `json:"characters"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// UserCharacter represents a character associated with a user account
type UserCharacter struct {
	CharacterID   int64     `json:"character_id" bson:"character_id"`
	Name          string    `json:"name" bson:"name"`
	CorporationID int64     `json:"corporation_id" bson:"corporation_id"`
	AllianceID    int64     `json:"alliance_id,omitempty" bson:"alliance_id,omitempty"`
	IsPrimary     bool      `json:"is_primary" bson:"is_primary"`
	AddedAt       time.Time `json:"added_at" bson:"added_at"`
	LastActive    time.Time `json:"last_active" bson:"last_active"`
}

// CachedCharacter represents cached character information for performance
type CachedCharacter struct {
	CharacterID   int64     `json:"character_id" bson:"_id"`
	Name          string    `json:"name" bson:"name"`
	CorporationID int64     `json:"corporation_id" bson:"corporation_id"`
	AllianceID    int64     `json:"alliance_id,omitempty" bson:"alliance_id,omitempty"`
	LastUpdated   time.Time `json:"last_updated" bson:"last_updated"`
	ExpiresAt     time.Time `json:"expires_at" bson:"expires_at"`
}