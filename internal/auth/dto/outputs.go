package dto

import "time"

// =============================================================================
// RESPONSE DTOs (Legacy)
// =============================================================================

// AuthStatusResponse represents authentication status
type AuthStatusResponse struct {
	Authenticated bool     `json:"authenticated"`
	UserID        *string  `json:"user_id"`
	CharacterID   *int     `json:"character_id"`
	CharacterName *string  `json:"character_name"`
	Characters    []string `json:"characters"`
	Permissions   []string `json:"permissions"`
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
	UserID          string            `json:"user_id"`
	CharacterID     int               `json:"character_id"`
	CharacterName   string            `json:"character_name"`
	CorporationID   int               `json:"corporation_id,omitempty"`
	CorporationName string            `json:"corporation_name,omitempty"`
	AllianceID      int               `json:"alliance_id,omitempty"`
	AllianceName    string            `json:"alliance_name,omitempty"`
	SecurityStatus  float64           `json:"security_status,omitempty"`
	Birthday        time.Time         `json:"birthday,omitempty"`
	Scopes          string            `json:"scopes"`
	TokenExpiry     time.Time         `json:"token_expiry,omitempty"`
	LastLogin       time.Time         `json:"last_login"`
	ProfileUpdated  time.Time         `json:"profile_updated"`
	Valid           bool              `json:"valid"`
	Metadata        map[string]string `json:"metadata,omitempty"`
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

// =============================================================================
// HUMA OUTPUT DTOs
// =============================================================================

// EVELoginOutput represents the output for EVE SSO login initiation
type EVELoginOutput struct {
	Body EVELoginResponse `json:"body"`
}

// EVERegisterOutput represents the output for EVE SSO registration initiation
type EVERegisterOutput struct {
	Body EVELoginResponse `json:"body"`
}

// EVECallbackOutput represents the output for EVE SSO callback
type EVECallbackOutput struct {
	Status    int                    `json:"-" status:"302" doc:"HTTP status code for redirect"`
	SetCookie string                 `header:"Set-Cookie" doc:"Authentication cookie"`
	Location  string                 `header:"Location" doc:"Redirect location"`
	Body      map[string]interface{} `json:"body,omitempty"`
}

// EVETokenExchangeOutput represents the output for mobile token exchange
type EVETokenExchangeOutput struct {
	Body TokenResponse `json:"body"`
}

// AuthStatusOutput represents the output for authentication status
type AuthStatusOutput struct {
	Body AuthStatusResponse `json:"body"`
}

// UserInfoOutput represents the output for current user information
type UserInfoOutput struct {
	Body UserInfoResponse `json:"body"`
}

// ProfileOutput represents the output for user profile
type ProfileOutput struct {
	Body ProfileResponse `json:"body"`
}

// ProfileRefreshOutput represents the output for profile refresh
type ProfileRefreshOutput struct {
	Body ProfileResponse `json:"body"`
}

// PublicProfileOutput represents the output for public profile lookup
type PublicProfileOutput struct {
	Body PublicProfileResponse `json:"body"`
}

// TokenOutput represents the output for bearer token retrieval
type TokenOutput struct {
	Body TokenResponse `json:"body"`
}

// LogoutOutput represents the output for logout
type LogoutOutput struct {
	SetCookie string         `header:"Set-Cookie" doc:"Clear authentication cookie"`
	Body      LogoutResponse `json:"body"`
}

// RefreshTokenOutput represents the output for token refresh
type RefreshTokenOutput struct {
	Body RefreshTokenResponse `json:"body"`
}

// VerifyTokenOutput represents the output for token verification
type VerifyTokenOutput struct {
	Body VerifyResponse `json:"body"`
}

// StatusOutput represents the module status response
type StatusOutput struct {
	Body AuthModuleStatusResponse `json:"body"`
}

// AuthModuleStatusResponse represents the actual status response data
type AuthModuleStatusResponse struct {
	Module       string                `json:"module" description:"Module name"`
	Status       string                `json:"status" enum:"healthy,degraded,unhealthy" description:"Module health status"`
	Message      string                `json:"message,omitempty" description:"Optional status message or error details"`
	Dependencies *AuthDependencyStatus `json:"dependencies,omitempty" description:"Status of module dependencies"`
	Metrics      *AuthMetrics          `json:"metrics,omitempty" description:"Performance and operational metrics"`
	LastChecked  string                `json:"last_checked" description:"Timestamp of last health check"`
}

// AuthDependencyStatus represents the status of auth module dependencies
type AuthDependencyStatus struct {
	Database        string `json:"database" description:"MongoDB connection status"`
	DatabaseLatency string `json:"database_latency,omitempty" description:"Database response time"`
	Redis           string `json:"redis,omitempty" description:"Redis connection status (if available)"`
	RedisLatency    string `json:"redis_latency,omitempty" description:"Redis response time"`
	EVEOnlineESI    string `json:"eve_online_esi,omitempty" description:"EVE Online ESI availability"`
	ESILatency      string `json:"esi_latency,omitempty" description:"ESI response time"`
}

// AuthMetrics represents performance metrics for the auth module
type AuthMetrics struct {
	ActiveSessions   int     `json:"active_sessions" description:"Number of active user sessions"`
	TotalUsers       int     `json:"total_users" description:"Total registered users"`
	TokenRefreshes   int     `json:"token_refreshes_24h" description:"Token refreshes in last 24 hours"`
	FailedLogins     int     `json:"failed_logins_1h" description:"Failed login attempts in last hour"`
	AverageLoginTime string  `json:"avg_login_time" description:"Average login processing time"`
	MemoryUsage      float64 `json:"memory_usage_mb" description:"Memory usage in MB"`
}
