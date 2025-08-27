package routes

import (
	"context"
	"time"

	"go-falcon/internal/auth/dto"
	"go-falcon/internal/auth/middleware"
	"go-falcon/internal/auth/services"
	"go-falcon/pkg/config"
	humaMiddleware "go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles HTTP routing for the Auth module
type Routes struct {
	authService    *services.AuthService
	middleware     *middleware.Middleware
	authMiddleware *humaMiddleware.AuthMiddleware
	api            huma.API
}

// NewRoutes creates a new Auth routes handler
func NewRoutes(authService *services.AuthService, middleware *middleware.Middleware, router chi.Router) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Auth Module", "1.0.0")
	config.Info.Description = "EVE Online SSO authentication and user profile management"

	api := humachi.New(router, config)

	// Create authentication middleware using the auth service as JWT validator
	authMiddleware := humaMiddleware.NewAuthMiddleware(authService)

	hr := &Routes{
		authService:    authService,
		middleware:     middleware,
		authMiddleware: authMiddleware,
		api:            api,
	}

	// Register all routes
	hr.registerRoutes()

	return hr
}

// RegisterAuthRoutes registers auth routes on a shared API
func RegisterAuthRoutes(api huma.API, basePath string, authService *services.AuthService, middleware *middleware.Middleware) {
	// Create authentication middleware using the auth service as JWT validator
	authMiddleware := humaMiddleware.NewAuthMiddleware(authService)

	// EVE Online SSO endpoints (public)
	huma.Register(api, huma.Operation{
		OperationID: "auth-eve-login",
		Method:      "GET",
		Path:        basePath + "/eve/login",
		Summary:     "Initiate EVE SSO login (basic, no scopes)",
		Description: "Start EVE Online SSO authentication flow without additional scopes",
		Tags:        []string{"Auth / EVE"},
	}, func(ctx context.Context, input *dto.EVELoginInput) (*dto.EVELoginOutput, error) {
		// Extract user ID from cookie if present
		userID := ""
		if input.Cookie != "" {
			user, err := authService.GetCurrentUserFromHeaders(ctx, "", input.Cookie)
			if err == nil && user != nil {
				userID = user.UserID
			}
		}

		// Generate login URL without scopes (basic login)
		loginResp, err := authService.InitiateEVELogin(ctx, false, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to initiate login", err)
		}

		return &dto.EVELoginOutput{Body: *loginResp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-eve-register",
		Method:      "GET",
		Path:        basePath + "/eve/register",
		Summary:     "Initiate EVE SSO registration (full scopes)",
		Description: "Start EVE Online SSO authentication flow with all required scopes",
		Tags:        []string{"Auth / EVE"},
	}, func(ctx context.Context, input *dto.EVERegisterInput) (*dto.EVERegisterOutput, error) {
		// Extract user ID from cookie if present
		userID := ""
		if input.Cookie != "" {
			user, err := authService.GetCurrentUserFromHeaders(ctx, "", input.Cookie)
			if err == nil && user != nil {
				userID = user.UserID
			}
		}

		// Generate login URL with full scopes (registration)
		loginResp, err := authService.InitiateEVELogin(ctx, true, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to initiate registration", err)
		}

		return &dto.EVERegisterOutput{Body: *loginResp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-eve-callback",
		Method:      "GET",
		Path:        basePath + "/eve/callback",
		Summary:     "EVE SSO OAuth2 callback",
		Description: "Handle OAuth2 callback from EVE Online SSO",
		Tags:        []string{"Auth / EVE"},
	}, func(ctx context.Context, input *dto.EVECallbackInput) (*dto.EVECallbackOutput, error) {
		// Extract existing user_id from cookie if present
		var existingUserID string
		if input.Cookie != "" {
			// Try to validate the existing JWT from cookie
			user, err := authService.GetCurrentUserFromHeaders(ctx, "", input.Cookie)
			if err == nil && user != nil {
				existingUserID = user.UserID
			}
		}

		// Handle the OAuth callback with existing user ID if available
		jwtToken, _, err := authService.HandleEVECallbackWithUserID(ctx, input.Code, input.State, existingUserID)
		if err != nil {
			return nil, huma.Error400BadRequest("Authentication failed", err)
		}

		// Set authentication cookie using Huma header response
		cookieHeader := humaMiddleware.CreateAuthCookieHeader(jwtToken)

		// Get frontend URL from configuration
		frontendURL := config.GetFrontendURL()

		// Return HTTP 302 redirect with Location header and cookie
		// Huma will handle this as a proper redirect response
		return &dto.EVECallbackOutput{
			Status:    302,
			SetCookie: cookieHeader,
			Location:  frontendURL,
			Body:      nil, // Empty body for redirect
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-eve-token-exchange",
		Method:      "POST",
		Path:        basePath + "/eve/token",
		Summary:     "Exchange EVE token for JWT",
		Description: "Exchange EVE SSO access token for internal JWT (mobile apps)",
		Tags:        []string{"Auth / EVE"},
	}, func(ctx context.Context, input *dto.EVETokenExchangeInput) (*dto.EVETokenExchangeOutput, error) {
		// Exchange EVE token for JWT
		tokenResp, err := authService.ExchangeEVEToken(ctx, &input.Body)
		if err != nil {
			return nil, huma.Error401Unauthorized("Token exchange failed", err)
		}

		return &dto.EVETokenExchangeOutput{Body: *tokenResp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-eve-refresh",
		Method:      "POST",
		Path:        basePath + "/eve/refresh",
		Summary:     "Refresh access token",
		Description: "Refresh an expired EVE access token using refresh token",
		Tags:        []string{"Auth / EVE"},
	}, func(ctx context.Context, input *dto.RefreshTokenInput) (*dto.RefreshTokenOutput, error) {
		// Refresh the access token using the EVE service
		tokenResp, err := authService.RefreshAccessToken(ctx, input.Body.RefreshToken)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to refresh token", err)
		}

		// Calculate expiration time from expires_in seconds
		expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

		return &dto.RefreshTokenOutput{
			Body: dto.RefreshTokenResponse{
				AccessToken:  tokenResp.AccessToken,
				RefreshToken: tokenResp.RefreshToken,
				ExpiresIn:    tokenResp.ExpiresIn,
				ExpiresAt:    expiresAt,
			},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-eve-verify",
		Method:      "GET",
		Path:        basePath + "/eve/verify",
		Summary:     "Verify JWT token",
		Description: "Verify the validity of a JWT token",
		Tags:        []string{"Auth / EVE"},
	}, func(ctx context.Context, input *dto.VerifyTokenInput) (*dto.VerifyTokenOutput, error) {
		// Verify the JWT token using the auth service
		user, expiresAt, err := authService.VerifyJWT(input.Token)
		if err != nil {
			return &dto.VerifyTokenOutput{
				Body: dto.VerifyResponse{
					Valid: false,
				},
			}, nil
		}

		return &dto.VerifyTokenOutput{
			Body: dto.VerifyResponse{
				Valid:         true,
				CharacterID:   user.CharacterID,
				CharacterName: user.CharacterName,
				ExpiresAt:     expiresAt,
			},
		}, nil
	})

	// Status endpoint (public, no auth required)
	huma.Register(api, huma.Operation{
		OperationID: "auth-get-status",
		Method:      "GET",
		Path:        basePath + "/status",
		Summary:     "Get auth module status",
		Description: "Returns the health status of the auth module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
		status := authService.GetStatus(ctx)
		return &dto.StatusOutput{Body: *status}, nil
	})

	// Authentication status and user info (public with optional auth)
	huma.Register(api, huma.Operation{
		OperationID: "auth-auth-status",
		Method:      "GET",
		Path:        basePath + "/auth-status",
		Summary:     "Check authentication status",
		Description: "Quick check if user is authenticated",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *dto.AuthStatusInput) (*dto.AuthStatusOutput, error) {
		// Use the new method that accepts header strings
		statusResp, err := authService.GetAuthStatusFromHeaders(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to check auth status", err)
		}

		return &dto.AuthStatusOutput{Body: *statusResp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-user-info",
		Method:      "GET",
		Path:        basePath + "/user",
		Summary:     "Get current user info",
		Description: "Get information about the currently authenticated user",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *dto.UserInfoInput) (*dto.UserInfoOutput, error) {
		// Use the new method that accepts header strings
		userInfo, err := authService.GetCurrentUserFromHeaders(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, huma.Error401Unauthorized("User not authenticated", err)
		}

		return &dto.UserInfoOutput{Body: *userInfo}, nil
	})

	// Profile endpoints (require authentication)
	huma.Register(api, huma.Operation{
		OperationID: "auth-get-profile",
		Method:      "GET",
		Path:        basePath + "/profile",
		Summary:     "Get user profile",
		Description: "Get full user profile with character information (requires authentication)",
		Tags:        []string{"Auth / Profile"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.ProfileInput) (*dto.ProfileOutput, error) {
		// Validate authentication using Huma auth middleware
		user, err := authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
		if err != nil {
			return nil, err // Returns proper Huma error response
		}

		// Get user profile using authenticated character ID
		profile, err := authService.GetUserProfile(ctx, user.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user profile", err)
		}

		// Convert to DTO response
		response := &dto.ProfileResponse{
			UserID:          profile.UserID,
			CharacterID:     profile.CharacterID,
			CharacterName:   profile.CharacterName,
			CorporationID:   profile.CorporationID,
			CorporationName: profile.CorporationName,
			AllianceID:      profile.AllianceID,
			AllianceName:    profile.AllianceName,
			SecurityStatus:  profile.SecurityStatus,
			Birthday:        profile.Birthday,
			Scopes:          profile.Scopes,
			LastLogin:       profile.LastLogin,
		}

		return &dto.ProfileOutput{Body: *response}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-refresh-profile",
		Method:      "POST",
		Path:        basePath + "/profile/refresh",
		Summary:     "Refresh user profile",
		Description: "Refresh user profile data from EVE Online ESI (requires authentication)",
		Tags:        []string{"Auth / Profile"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.ProfileRefreshInput) (*dto.ProfileRefreshOutput, error) {
		// Validate authentication using Huma auth middleware
		user, err := authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
		if err != nil {
			return nil, err // Returns proper Huma error response
		}

		// Refresh user profile from EVE Online ESI
		profile, err := authService.RefreshUserProfile(ctx, user.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to refresh user profile", err)
		}

		// Convert to DTO response
		response := &dto.ProfileResponse{
			UserID:          profile.UserID,
			CharacterID:     profile.CharacterID,
			CharacterName:   profile.CharacterName,
			CorporationID:   profile.CorporationID,
			CorporationName: profile.CorporationName,
			AllianceID:      profile.AllianceID,
			AllianceName:    profile.AllianceName,
			SecurityStatus:  profile.SecurityStatus,
			Birthday:        profile.Birthday,
			Scopes:          profile.Scopes,
			LastLogin:       profile.LastLogin,
		}

		return &dto.ProfileRefreshOutput{Body: *response}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-get-token",
		Method:      "GET",
		Path:        basePath + "/token",
		Summary:     "Get bearer token",
		Description: "Get current JWT bearer token for API access (requires authentication)",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearerAuth": {}}, {"cookieAuth": {}}},
	}, func(ctx context.Context, input *dto.TokenInput) (*dto.TokenOutput, error) {
		// Validate authentication using Huma auth middleware
		user, err := authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
		if err != nil {
			return nil, err // Returns proper Huma error response
		}

		// Get user profile to obtain user ID for token generation
		profile, err := authService.GetUserProfile(ctx, user.CharacterID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get user profile", err)
		}

		// Generate JWT token for authenticated user
		tokenResp, err := authService.GetBearerToken(ctx, profile.UserID, user.CharacterID, user.CharacterName, user.Scopes)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to generate token", err)
		}

		// Convert to DTO response (TokenResponse already has the correct field names)
		response := &dto.TokenResponse{
			Token:     tokenResp.Token,
			ExpiresAt: tokenResp.ExpiresAt,
		}

		return &dto.TokenOutput{Body: *response}, nil
	})

	// Public endpoints
	huma.Register(api, huma.Operation{
		OperationID: "auth-public-profile",
		Method:      "GET",
		Path:        basePath + "/profile/public",
		Summary:     "Get public profile",
		Description: "Get public profile information by character ID",
		Tags:        []string{"Auth / Profile"},
	}, func(ctx context.Context, input *dto.PublicProfileInput) (*dto.PublicProfileOutput, error) {
		profile, err := authService.GetPublicProfile(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error404NotFound("Character not found", err)
		}

		return &dto.PublicProfileOutput{Body: *profile}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth-logout",
		Method:      "POST",
		Path:        basePath + "/logout",
		Summary:     "Logout user",
		Description: "Clear authentication cookie and logout the user",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *dto.LogoutInput) (*dto.LogoutOutput, error) {
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
	})
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

	// Status endpoint (public, no auth required)
	huma.Get(hr.api, "/status", hr.moduleStatus)

	// Authentication status and user info (public with optional auth)
	huma.Get(hr.api, "/auth-status", hr.authStatus)
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
	// Extract user ID from cookie if present
	userID := ""
	if input.Cookie != "" {
		user, err := hr.authService.GetCurrentUserFromHeaders(ctx, "", input.Cookie)
		if err == nil && user != nil {
			userID = user.UserID
		}
	}

	// Generate login URL without scopes (basic login)
	loginResp, err := hr.authService.InitiateEVELogin(ctx, false, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initiate login", err)
	}

	return &dto.EVELoginOutput{Body: *loginResp}, nil
}

func (hr *Routes) eveRegister(ctx context.Context, input *dto.EVERegisterInput) (*dto.EVERegisterOutput, error) {
	// Extract user ID from cookie if present
	userID := ""
	if input.Cookie != "" {
		user, err := hr.authService.GetCurrentUserFromHeaders(ctx, "", input.Cookie)
		if err == nil && user != nil {
			userID = user.UserID
		}
	}

	// Generate login URL with full scopes (registration)
	loginResp, err := hr.authService.InitiateEVELogin(ctx, true, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initiate registration", err)
	}

	return &dto.EVERegisterOutput{Body: *loginResp}, nil
}

func (hr *Routes) eveCallback(ctx context.Context, input *dto.EVECallbackInput) (*dto.EVECallbackOutput, error) {
	// Extract existing user_id from cookie if present
	var existingUserID string
	if input.Cookie != "" {
		// Try to validate the existing JWT from cookie
		user, err := hr.authService.GetCurrentUserFromHeaders(ctx, "", input.Cookie)
		if err == nil && user != nil {
			existingUserID = user.UserID
		}
	}

	// Handle the OAuth callback with existing user ID if available
	jwtToken, _, err := hr.authService.HandleEVECallbackWithUserID(ctx, input.Code, input.State, existingUserID)
	if err != nil {
		return nil, huma.Error400BadRequest("Authentication failed", err)
	}

	// Set authentication cookie using Huma header response
	cookieHeader := humaMiddleware.CreateAuthCookieHeader(jwtToken)

	// Get frontend URL from configuration
	frontendURL := config.GetFrontendURL()

	// Return HTTP 302 redirect with Location header and cookie
	// Huma will handle this as a proper redirect response
	return &dto.EVECallbackOutput{
		Status:    302,
		SetCookie: cookieHeader,
		Location:  frontendURL,
		Body:      nil, // Empty body for redirect
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

// Module status handler

func (hr *Routes) moduleStatus(ctx context.Context, input *struct{}) (*dto.StatusOutput, error) {
	status := hr.authService.GetStatus(ctx)
	return &dto.StatusOutput{Body: *status}, nil
}

// Authentication status and user info handlers

func (hr *Routes) authStatus(ctx context.Context, input *dto.AuthStatusInput) (*dto.AuthStatusOutput, error) {
	// Use the new method that accepts header strings
	statusResp, err := hr.authService.GetAuthStatusFromHeaders(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to check auth status", err)
	}

	return &dto.AuthStatusOutput{Body: *statusResp}, nil
}

func (hr *Routes) userInfo(ctx context.Context, input *dto.UserInfoInput) (*dto.UserInfoOutput, error) {
	// Use the new method that accepts header strings
	userInfo, err := hr.authService.GetCurrentUserFromHeaders(ctx, input.Authorization, input.Cookie)
	if err != nil {
		return nil, huma.Error401Unauthorized("User not authenticated", err)
	}

	return &dto.UserInfoOutput{Body: *userInfo}, nil
}

// Profile handlers (require authentication)

func (hr *Routes) profile(ctx context.Context, input *dto.ProfileInput) (*dto.ProfileOutput, error) {
	// Validate authentication using Huma auth middleware
	user, err := hr.authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
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
		UserID:          profile.UserID,
		CharacterID:     profile.CharacterID,
		CharacterName:   profile.CharacterName,
		CorporationID:   profile.CorporationID,
		CorporationName: profile.CorporationName,
		AllianceID:      profile.AllianceID,
		AllianceName:    profile.AllianceName,
		SecurityStatus:  profile.SecurityStatus,
		Birthday:        profile.Birthday,
		Scopes:          profile.Scopes,
		LastLogin:       profile.LastLogin,
	}

	return &dto.ProfileOutput{Body: *response}, nil
}

func (hr *Routes) profileRefresh(ctx context.Context, input *dto.ProfileRefreshInput) (*dto.ProfileRefreshOutput, error) {
	// Validate authentication using Huma auth middleware
	user, err := hr.authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
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
		UserID:          profile.UserID,
		CharacterID:     profile.CharacterID,
		CharacterName:   profile.CharacterName,
		CorporationID:   profile.CorporationID,
		CorporationName: profile.CorporationName,
		AllianceID:      profile.AllianceID,
		AllianceName:    profile.AllianceName,
		SecurityStatus:  profile.SecurityStatus,
		Birthday:        profile.Birthday,
		Scopes:          profile.Scopes,
		LastLogin:       profile.LastLogin,
	}

	return &dto.ProfileRefreshOutput{Body: *response}, nil
}

func (hr *Routes) token(ctx context.Context, input *dto.TokenInput) (*dto.TokenOutput, error) {
	// Validate authentication using Huma auth middleware
	user, err := hr.authMiddleware.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
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
