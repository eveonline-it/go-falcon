package dto

import "time"

// AuthStatusResponse represents authentication status
type AuthStatusResponse struct {
	Authenticated bool `json:"authenticated"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

// EVELoginResponse represents EVE SSO login initiation response
type EVELoginResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// UserInfoResponse represents current user information
type UserInfoResponse struct {
	UserID        string `json:"user_id"`
	CharacterID   int    `json:"character_id"`
	CharacterName string `json:"character_name"`
	Scopes        string `json:"scopes"`
	ExpiresAt     string `json:"expires_at,omitempty"`
}

// ProfileResponse represents a user profile
type ProfileResponse struct {
	UserID            string            `json:"user_id"`
	CharacterID       int               `json:"character_id"`
	CharacterName     string            `json:"character_name"`
	CorporationID     int               `json:"corporation_id,omitempty"`
	CorporationName   string            `json:"corporation_name,omitempty"`
	AllianceID        int               `json:"alliance_id,omitempty"`
	AllianceName      string            `json:"alliance_name,omitempty"`
	SecurityStatus    float64           `json:"security_status,omitempty"`
	Birthday          time.Time         `json:"birthday,omitempty"`
	Scopes            string            `json:"scopes"`
	TokenExpiry       time.Time         `json:"token_expiry,omitempty"`
	LastLogin         time.Time         `json:"last_login"`
	ProfileUpdated    time.Time         `json:"profile_updated"`
	Valid             bool              `json:"valid"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// PublicProfileResponse represents public character information
type PublicProfileResponse struct {
	CharacterID     int     `json:"character_id"`
	CharacterName   string  `json:"character_name"`
	CorporationID   int     `json:"corporation_id,omitempty"`
	CorporationName string  `json:"corporation_name,omitempty"`
	AllianceID      int     `json:"alliance_id,omitempty"`
	AllianceName    string  `json:"alliance_name,omitempty"`
	SecurityStatus  float64 `json:"security_status,omitempty"`
}

// TokenResponse represents a JWT token response
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RefreshTokenResponse represents a successful token refresh
type RefreshTokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// LogoutResponse represents a successful logout
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VerifyResponse represents JWT verification response
type VerifyResponse struct {
	Valid         bool      `json:"valid"`
	CharacterID   int       `json:"character_id,omitempty"`
	CharacterName string    `json:"character_name,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}