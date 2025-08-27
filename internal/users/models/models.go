package models

import (
	"time"
)

// User represents a user with character information and state control
// Uses the same database collection as auth module: user_profiles
type User struct {
	CharacterID   int       `json:"character_id" bson:"character_id"`     // EVE character ID (unique)
	UserID        string    `json:"user_id" bson:"user_id"`               // UUID for internal identification
	AccessToken   string    `json:"-" bson:"access_token"`                // EVE SSO access token (hidden from JSON)
	RefreshToken  string    `json:"-" bson:"refresh_token"`               // EVE SSO refresh token (hidden from JSON)
	Banned        bool      `json:"banned" bson:"banned"`                 // Ban status
	Scopes        string    `json:"scopes" bson:"scopes"`                 // EVE Online permissions
	Position      int       `json:"position" bson:"position"`             // User position/rank
	Notes         string    `json:"notes" bson:"notes"`                   // Administrative notes
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`         // Registration timestamp
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`         // Last update timestamp
	LastLogin     time.Time `json:"last_login" bson:"last_login"`         // Last login timestamp
	CharacterName string    `json:"character_name" bson:"character_name"` // EVE character name
	Valid         bool      `json:"valid" bson:"valid"`                   // Character profile validity status
}

// CharacterSummary represents basic character information for listing
type CharacterSummary struct {
	CharacterID   int        `json:"character_id" bson:"character_id"`
	CharacterName string     `json:"character_name" bson:"character_name"`
	UserID        string     `json:"user_id" bson:"user_id"`
	Banned        bool       `json:"banned" bson:"banned"`
	Position      int        `json:"position" bson:"position"`
	LastLogin     *time.Time `json:"last_login,omitempty" bson:"last_login,omitempty"`
	Valid         bool       `json:"valid" bson:"valid"`
}

// CollectionName returns the MongoDB collection name for users
func (User) CollectionName() string {
	return "user_profiles"
}

// CollectionName returns the MongoDB collection name for character summaries
func (CharacterSummary) CollectionName() string {
	return "user_profiles"
}
