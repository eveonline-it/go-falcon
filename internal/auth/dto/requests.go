package dto

// LoginRequest represents a basic login request
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// EVETokenExchangeRequest represents a mobile app token exchange request
type EVETokenExchangeRequest struct {
	AccessToken  string `json:"access_token" validate:"required"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ProfileRefreshRequest represents a profile refresh request
type ProfileRefreshRequest struct {
	ForceRefresh bool `json:"force_refresh,omitempty"`
}

// =============================================================================
// HUMA INPUT/OUTPUT WRAPPERS (from huma_requests.go)
// =============================================================================

// EVELoginInput represents the input for EVE SSO login initiation (no body needed)
type EVELoginInput struct {
	// No body parameters needed - this is a simple GET endpoint
}

// EVELoginOutput represents the output for EVE SSO login initiation
type EVELoginOutput struct {
	Body EVELoginResponse `json:"body"`
}

// EVERegisterInput represents the input for EVE SSO registration initiation (no body needed)
type EVERegisterInput struct {
	// No body parameters needed - this is a simple GET endpoint
}

// EVERegisterOutput represents the output for EVE SSO registration initiation
type EVERegisterOutput struct {
	Body EVELoginResponse `json:"body"`
}

// EVECallbackInput represents the input for EVE SSO callback
type EVECallbackInput struct {
	Code  string `query:"code" validate:"required" doc:"OAuth2 authorization code from EVE Online"`
	State string `query:"state" validate:"required" doc:"CSRF protection state parameter"`
}

// EVECallbackOutput represents the output for EVE SSO callback
type EVECallbackOutput struct {
	Status    int                    `json:"-" status:"302" doc:"HTTP status code for redirect"`
	SetCookie string                 `header:"Set-Cookie" doc:"Authentication cookie"`
	Location  string                 `header:"Location" doc:"Redirect location"`
	Body      map[string]interface{} `json:"body,omitempty"`
}

// EVETokenExchangeInput represents the input for mobile token exchange
type EVETokenExchangeInput struct {
	Body EVETokenExchangeRequest `json:"body"`
}

// EVETokenExchangeOutput represents the output for mobile token exchange
type EVETokenExchangeOutput struct {
	Body TokenResponse `json:"body"`
}

// AuthStatusInput represents the input for authentication status (no body needed)
type AuthStatusInput struct {
	// No parameters needed - status is determined from context/cookies
}

// AuthStatusOutput represents the output for authentication status
type AuthStatusOutput struct {
	Body AuthStatusResponse `json:"body"`
}

// UserInfoInput represents the input for current user information (no body needed)
type UserInfoInput struct {
	// No parameters needed - user info comes from authenticated context
}

// UserInfoOutput represents the output for current user information
type UserInfoOutput struct {
	Body UserInfoResponse `json:"body"`
}

// ProfileInput represents the input for user profile (no body needed)
type ProfileInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Session cookie for authentication"`
}

// ProfileOutput represents the output for user profile
type ProfileOutput struct {
	Body ProfileResponse `json:"body"`
}

// ProfileRefreshInput represents the input for profile refresh
type ProfileRefreshInput struct {
	Authorization string                `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string                `header:"Cookie" doc:"Session cookie for authentication"`
	Body          ProfileRefreshRequest `json:"body"`
}

// ProfileRefreshOutput represents the output for profile refresh
type ProfileRefreshOutput struct {
	Body ProfileResponse `json:"body"`
}

// PublicProfileInput represents the input for public profile lookup
type PublicProfileInput struct {
	CharacterID int `query:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}

// PublicProfileOutput represents the output for public profile lookup
type PublicProfileOutput struct {
	Body PublicProfileResponse `json:"body"`
}

// TokenInput represents the input for bearer token retrieval (no body needed)
type TokenInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Session cookie for authentication"`
}

// TokenOutput represents the output for bearer token retrieval
type TokenOutput struct {
	Body TokenResponse `json:"body"`
}

// LogoutInput represents the input for logout (no body needed)
type LogoutInput struct {
	// No parameters needed - logout clears cookies
}

// LogoutOutput represents the output for logout
type LogoutOutput struct {
	SetCookie string         `header:"Set-Cookie" doc:"Clear authentication cookie"`
	Body      LogoutResponse `json:"body"`
}

// RefreshTokenInput represents the input for token refresh
type RefreshTokenInput struct {
	Body RefreshTokenRequest `json:"body"`
}

// RefreshTokenOutput represents the output for token refresh
type RefreshTokenOutput struct {
	Body RefreshTokenResponse `json:"body"`
}

// VerifyTokenInput represents the input for token verification
type VerifyTokenInput struct {
	Token string `query:"token" validate:"required" doc:"JWT token to verify"`
}

// VerifyTokenOutput represents the output for token verification
type VerifyTokenOutput struct {
	Body VerifyResponse `json:"body"`
}