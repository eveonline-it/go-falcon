package dto

// =============================================================================
// REQUEST DTOs (Legacy)
// =============================================================================

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
// HUMA INPUT DTOs
// =============================================================================

// EVELoginInput represents the input for EVE SSO login initiation (no body needed)
type EVELoginInput struct {
	Cookie string `header:"Cookie" doc:"Optional session cookie for authentication"`
}

// EVERegisterInput represents the input for EVE SSO registration initiation (no body needed)
type EVERegisterInput struct {
	Cookie string `header:"Cookie" doc:"Optional session cookie for authentication"`
}

// EVECallbackInput represents the input for EVE SSO callback
type EVECallbackInput struct {
	Code   string `query:"code" validate:"required" doc:"OAuth2 authorization code from EVE Online"`
	State  string `query:"state" validate:"required" doc:"CSRF protection state parameter"`
	Cookie string `header:"Cookie" doc:"Optional session cookie for authentication"`
}

// EVETokenExchangeInput represents the input for mobile token exchange
type EVETokenExchangeInput struct {
	Body EVETokenExchangeRequest `json:"body"`
}

// AuthStatusInput represents the input for authentication status (no body needed)
type AuthStatusInput struct {
	Authorization string `header:"Authorization" doc:"Optional Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Optional session cookie for authentication"`
}

// UserInfoInput represents the input for current user information (no body needed)
type UserInfoInput struct {
	Authorization string `header:"Authorization" doc:"Optional Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Optional session cookie for authentication"`
}

// ProfileInput represents the input for user profile (no body needed)
type ProfileInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Session cookie for authentication"`
}

// ProfileRefreshInput represents the input for profile refresh
type ProfileRefreshInput struct {
	Authorization string                `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string                `header:"Cookie" doc:"Session cookie for authentication"`
	Body          ProfileRefreshRequest `json:"body"`
}

// PublicProfileInput represents the input for public profile lookup
type PublicProfileInput struct {
	CharacterID int `query:"character_id" validate:"required" minimum:"90000000" maximum:"2147483647" doc:"EVE Online character ID"`
}

// TokenInput represents the input for bearer token retrieval (no body needed)
type TokenInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" doc:"Session cookie for authentication"`
}

// LogoutInput represents the input for logout (no body needed)
type LogoutInput struct {
	// No parameters needed - logout clears cookies
}

// RefreshTokenInput represents the input for token refresh
type RefreshTokenInput struct {
	Body RefreshTokenRequest `json:"body"`
}

// VerifyTokenInput represents the input for token verification
type VerifyTokenInput struct {
	Token string `query:"token" validate:"required" doc:"JWT token to verify"`
}
