package routes

import (
	"context"
	"net/http"

	"go-falcon/internal/auth/dto"
	"go-falcon/internal/auth/middleware"
	"go-falcon/internal/auth/services"
	humaMiddleware "go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles HTTP routing for the Auth module
type Routes struct {
	authService *services.AuthService
	middleware  *middleware.Middleware
	humaAuth    *humaMiddleware.HumaAuthMiddleware
	api         huma.API
}

// NewRoutes creates a new Huma Auth routes handler
func NewRoutes(authService *services.AuthService, middleware *middleware.Middleware, router chi.Router) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Auth Module", "1.0.0")
	config.Info.Description = "EVE Online SSO authentication and user profile management"
	
	api := humachi.New(router, config)

	// Create Huma authentication middleware using the auth service as JWT validator
	humaAuth := humaMiddleware.NewHumaAuthMiddleware(authService)

	hr := &Routes{
		authService: authService,
		middleware:  middleware,
		humaAuth:    humaAuth,
		api:         api,
	}

	// Register all routes
	hr.registerRoutes()

	return hr
}

// registerRoutes registers all Auth module routes with Huma
func (hr *Routes) registerRoutes() {
	// EVE Online SSO endpoints (public)
	huma.Get(hr.api, "/eve/login", hr.eveLogin)
	huma.Get(hr.api, "/eve/register", hr.eveRegister) 
	huma.Get(hr.api, "/eve/callback", hr.eveCallback)
	huma.Post(hr.api, "/eve/token", hr.eveTokenExchange)
	huma.Post(hr.api, "/eve/refresh", hr.eveRefresh)
	huma.Get(hr.api, "/eve/verify", hr.eveVerify)

	// Authentication status and user info (public with optional auth)
	huma.Get(hr.api, "/status", hr.authStatus)
	huma.Get(hr.api, "/user", hr.userInfo)

	// Profile endpoints (require authentication)
	huma.Get(hr.api, "/profile", hr.profile)
	huma.Post(hr.api, "/profile/refresh", hr.profileRefresh)
	huma.Get(hr.api, "/token", hr.token)

	// Public profile endpoint (no auth required)
	huma.Get(hr.api, "/profile/public", hr.publicProfile)

	// Logout endpoint (public)
	huma.Post(hr.api, "/logout", hr.logout)
}

// EVE SSO endpoint handlers

func (hr *Routes) eveLogin(ctx context.Context, input *dto.EVELoginInput) (*dto.EVELoginOutput, error) {
	// Extract user ID from context if authenticated (similar to original handler)
	userID := ""
	// TODO: Extract from context/cookies in Huma middleware

	// Generate login URL without scopes (basic login)
	loginResp, err := hr.authService.InitiateEVELogin(ctx, false, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initiate login", err)
	}

	return &dto.EVELoginOutput{Body: *loginResp}, nil
}

func (hr *Routes) eveRegister(ctx context.Context, input *dto.EVERegisterInput) (*dto.EVERegisterOutput, error) {
	// Extract user ID from context if authenticated
	userID := ""
	// TODO: Extract from context/cookies in Huma middleware

	// Generate login URL with full scopes (registration)
	loginResp, err := hr.authService.InitiateEVELogin(ctx, true, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initiate registration", err)
	}

	return &dto.EVERegisterOutput{Body: *loginResp}, nil
}

func (hr *Routes) eveCallback(ctx context.Context, input *dto.EVECallbackInput) (*dto.EVECallbackOutput, error) {
	// Handle the OAuth callback
	jwtToken, userInfo, err := hr.authService.HandleEVECallback(ctx, input.Code, input.State)
	if err != nil {
		return nil, huma.Error400BadRequest("Authentication failed", err)
	}

	// Create response with authentication cookie
	response := map[string]interface{}{
		"success": true,
		"user":    userInfo,
		"message": "Authentication successful",
	}

	// Set authentication cookie using Huma header response
	cookieHeader := humaMiddleware.CreateAuthCookieHeader(jwtToken)
	
	// TODO: Add proper redirect URL from frontend configuration
	redirectURL := "https://react.eveonline.it"

	return &dto.EVECallbackOutput{
		SetCookie: cookieHeader,
		Location:  redirectURL,
		Body:      response,
	}, nil
}

func (hr *Routes) eveTokenExchange(ctx context.Context, input *dto.EVETokenExchangeInput) (*dto.EVETokenExchangeOutput, error) {
	// Exchange EVE token for JWT
	tokenResp, err := hr.authService.ExchangeEVEToken(ctx, &input.Body)
	if err != nil {
		return nil, huma.Error401Unauthorized("Token exchange failed", err)
	}

	return &dto.EVETokenExchangeOutput{Body: *tokenResp}, nil
}

func (hr *Routes) eveRefresh(ctx context.Context, input *dto.RefreshTokenInput) (*dto.RefreshTokenOutput, error) {
	// TODO: Implement token refresh
	return nil, huma.Error501NotImplemented("Token refresh not yet implemented")
}

func (hr *Routes) eveVerify(ctx context.Context, input *dto.VerifyTokenInput) (*dto.VerifyTokenOutput, error) {
	// TODO: Implement token verification
	return nil, huma.Error501NotImplemented("Token verification not yet implemented")
}

// Authentication status and user info handlers

func (hr *Routes) authStatus(ctx context.Context, input *dto.AuthStatusInput) (*dto.AuthStatusOutput, error) {
	// TODO: Extract HTTP request from context for cookie checking
	// For now, create a minimal request object
	req := &http.Request{}
	
	statusResp, err := hr.authService.GetAuthStatus(ctx, req)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check auth status", err)
	}

	return &dto.AuthStatusOutput{Body: *statusResp}, nil
}

func (hr *Routes) userInfo(ctx context.Context, input *dto.UserInfoInput) (*dto.UserInfoOutput, error) {
	// TODO: Extract HTTP request from context
	req := &http.Request{}
	
	userInfo, err := hr.authService.GetCurrentUser(ctx, req)
	if err != nil {
		return nil, huma.Error401Unauthorized("User not authenticated", err)
	}

	return &dto.UserInfoOutput{Body: *userInfo}, nil
}

// Profile handlers (require authentication)

func (hr *Routes) profile(ctx context.Context, input *dto.ProfileInput) (*dto.ProfileOutput, error) {
	// Validate authentication using Huma auth middleware
	user, err := hr.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err // Returns proper Huma error response
	}

	// Get user profile using authenticated character ID
	profile, err := hr.authService.GetUserProfile(ctx, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user profile", err)
	}

	// Convert to DTO response
	response := &dto.ProfileResponse{
		UserID:            profile.UserID,
		CharacterID:       profile.CharacterID,
		CharacterName:     profile.CharacterName,
		CorporationID:     profile.CorporationID,
		CorporationName:   profile.CorporationName,
		AllianceID:        profile.AllianceID,
		AllianceName:      profile.AllianceName,
		SecurityStatus:    profile.SecurityStatus,
		Birthday:          profile.Birthday,
		Scopes:            profile.Scopes,
		LastLogin:         profile.LastLogin,
	}

	return &dto.ProfileOutput{Body: *response}, nil
}

func (hr *Routes) profileRefresh(ctx context.Context, input *dto.ProfileRefreshInput) (*dto.ProfileRefreshOutput, error) {
	// Validate authentication using Huma auth middleware
	user, err := hr.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err // Returns proper Huma error response
	}

	// Refresh user profile from EVE Online ESI
	profile, err := hr.authService.RefreshUserProfile(ctx, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to refresh user profile", err)
	}

	// Convert to DTO response
	response := &dto.ProfileResponse{
		UserID:            profile.UserID,
		CharacterID:       profile.CharacterID,
		CharacterName:     profile.CharacterName,
		CorporationID:     profile.CorporationID,
		CorporationName:   profile.CorporationName,
		AllianceID:        profile.AllianceID,
		AllianceName:      profile.AllianceName,
		SecurityStatus:    profile.SecurityStatus,
		Birthday:          profile.Birthday,
		Scopes:            profile.Scopes,
		LastLogin:         profile.LastLogin,
	}

	return &dto.ProfileRefreshOutput{Body: *response}, nil
}

func (hr *Routes) token(ctx context.Context, input *dto.TokenInput) (*dto.TokenOutput, error) {
	// Validate authentication using Huma auth middleware
	user, err := hr.humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	if err != nil {
		return nil, err // Returns proper Huma error response
	}

	// Get user profile to obtain user ID for token generation
	profile, err := hr.authService.GetUserProfile(ctx, user.CharacterID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get user profile", err)
	}

	// Generate JWT token for authenticated user
	tokenResp, err := hr.authService.GetBearerToken(ctx, profile.UserID, user.CharacterID, user.CharacterName, user.Scopes)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to generate token", err)
	}

	// Convert to DTO response (TokenResponse already has the correct field names)
	response := &dto.TokenResponse{
		Token:     tokenResp.Token,
		ExpiresAt: tokenResp.ExpiresAt,
	}

	return &dto.TokenOutput{Body: *response}, nil
}

// Public endpoints

func (hr *Routes) publicProfile(ctx context.Context, input *dto.PublicProfileInput) (*dto.PublicProfileOutput, error) {
	profile, err := hr.authService.GetPublicProfile(ctx, input.CharacterID)
	if err != nil {
		return nil, huma.Error404NotFound("Character not found", err)
	}

	return &dto.PublicProfileOutput{Body: *profile}, nil
}

func (hr *Routes) logout(ctx context.Context, input *dto.LogoutInput) (*dto.LogoutOutput, error) {
	// Clear authentication cookie
	response := dto.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}

	// Clear authentication cookie using Huma header response
	cookieHeader := humaMiddleware.CreateClearCookieHeader()

	return &dto.LogoutOutput{
		SetCookie: cookieHeader,
		Body:      response,
	}, nil
}

// Helper methods for cookie handling (to be implemented in future iterations)

// TODO: Implement cookie handling and user context extraction
// These will be added when Huma middleware integration is complete