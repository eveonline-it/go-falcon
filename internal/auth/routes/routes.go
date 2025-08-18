package routes

import (
	"context"
	"fmt"

	"go-falcon/internal/auth/dto"
	"go-falcon/internal/auth/middleware"
	"go-falcon/internal/auth/services"
	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	humaMiddleware "go-falcon/pkg/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles HTTP routing for the Auth module
type Routes struct {
	authService *services.AuthService
	middleware  *middleware.Middleware
	authMiddleware *humaMiddleware.AuthMiddleware
	api         huma.API
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
		authService: authService,
		middleware:  middleware,
		authMiddleware: authMiddleware,
		api:         api,
	}

	// Register all routes
	hr.registerRoutes()

	return hr
}

// RegisterAuthRoutes registers auth routes on a shared Huma API
func RegisterAuthRoutes(api huma.API, basePath string, authService *services.AuthService, middleware *middleware.Middleware, mongodb *database.MongoDB) {
	// Create authentication middleware using the auth service as JWT validator
	authMiddleware := humaMiddleware.NewAuthMiddleware(authService)

	// EVE Online SSO endpoints (public)
	huma.Get(api, basePath+"/eve/login", func(ctx context.Context, input *dto.EVELoginInput) (*dto.EVELoginOutput, error) {
		// Extract user ID from context if authenticated
		userID := extractUserIDFromHeaders(authService, input.Authorization, input.Cookie)
		
		// Generate login URL without scopes (basic login)
		loginResp, err := authService.InitiateEVELogin(ctx, false, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to initiate login", err)
		}

		return &dto.EVELoginOutput{Body: *loginResp}, nil
	})
	
	huma.Get(api, basePath+"/eve/register", func(ctx context.Context, input *dto.EVERegisterInput) (*dto.EVERegisterOutput, error) {
		// Extract user ID from context if authenticated
		userID := extractUserIDFromHeaders(authService, input.Authorization, input.Cookie)
		
		// Generate login URL with full scopes (registration)
		loginResp, err := authService.InitiateEVELogin(ctx, true, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to initiate registration", err)
		}

		return &dto.EVERegisterOutput{Body: *loginResp}, nil
	})

	huma.Get(api, basePath+"/eve/callback", func(ctx context.Context, input *dto.EVECallbackInput) (*dto.EVECallbackOutput, error) {
		// Handle the OAuth callback
		jwtToken, _, err := authService.HandleEVECallback(ctx, input.Code, input.State)
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

	huma.Post(api, basePath+"/eve/token", func(ctx context.Context, input *dto.EVETokenExchangeInput) (*dto.EVETokenExchangeOutput, error) {
		// Exchange EVE token for JWT
		tokenResp, err := authService.ExchangeEVEToken(ctx, &input.Body)
		if err != nil {
			return nil, huma.Error401Unauthorized("Token exchange failed", err)
		}

		return &dto.EVETokenExchangeOutput{Body: *tokenResp}, nil
	})

	huma.Post(api, basePath+"/eve/refresh", func(ctx context.Context, input *dto.RefreshTokenInput) (*dto.RefreshTokenOutput, error) {
		// TODO: Implement token refresh
		return nil, huma.Error501NotImplemented("Token refresh not yet implemented")
	})

	huma.Get(api, basePath+"/eve/verify", func(ctx context.Context, input *dto.VerifyTokenInput) (*dto.VerifyTokenOutput, error) {
		// TODO: Implement token verification
		return nil, huma.Error501NotImplemented("Token verification not yet implemented")
	})

	// Authentication status and user info (public with optional auth)
	huma.Get(api, basePath+"/status", func(ctx context.Context, input *dto.AuthStatusInput) (*dto.AuthStatusOutput, error) {
		fmt.Printf("\n[DEBUG] ===== /auth/status HUMA HANDLER START =====\n")
		fmt.Printf("[DEBUG] AuthStatus Handler: Processing request\n")
		fmt.Printf("[DEBUG] AuthStatus Handler: Authorization header: %q\n", input.Authorization)
		if input.Cookie != "" {
			fmt.Printf("[DEBUG] AuthStatus Handler: Cookie present\n")
		} else {
			fmt.Printf("[DEBUG] AuthStatus Handler: No cookie present\n")
		}
		
		// Try to validate authentication (optional - don't fail if not authenticated)
		user := authMiddleware.ValidateOptionalAuthFromHeaders(input.Authorization, input.Cookie)
		if user == nil {
			fmt.Printf("[DEBUG] AuthStatus Handler: User not authenticated or validation failed\n")
			// Return unauthenticated status
			return &dto.AuthStatusOutput{
				Body: dto.AuthStatusResponse{
					Authenticated:  false,
					UserID:         nil,
					CharacterID:    nil,
					CharacterName:  nil,
					Characters:     []dto.CharacterInfo{},
					CharacterIDs:   []int64{},
					CorporationIDs: []int64{},
					AllianceIDs:    []int64{},
				},
			}, nil
		}
		
		fmt.Printf("[DEBUG] AuthStatus Handler: User authenticated: %s (Character: %d)\n", user.UserID, user.CharacterID)
		
		// Get user profile to obtain user ID for character resolution
		profile, err := authService.GetUserProfile(ctx, user.CharacterID)
		if err != nil {
			fmt.Printf("[DEBUG] AuthStatus Handler: Failed to get user profile: %v\n", err)
			// Return basic authenticated status without character details
			characterID := user.CharacterID
			characterName := user.CharacterName
			userID := user.UserID
			return &dto.AuthStatusOutput{
				Body: dto.AuthStatusResponse{
					Authenticated:  true,
					UserID:         &userID,
					CharacterID:    &characterID,
					CharacterName:  &characterName,
					Characters:     []dto.CharacterInfo{},
					CharacterIDs:   []int64{},
					CorporationIDs: []int64{},
					AllianceIDs:    []int64{},
				},
			}, nil
		}
		
		fmt.Printf("[DEBUG] AuthStatus Handler: Got user profile for UserID: %s\n", profile.UserID)
		
		// Create UserCharacterResolver and resolve all characters
		// Note: Redis not available in this context, using without caching
		resolver := humaMiddleware.NewUserCharacterResolver(mongodb)
		fmt.Printf("[DEBUG] AuthStatus Handler: Created UserCharacterResolver, now resolving characters...\n")
		
		userWithCharacters, err := resolver.GetUserWithCharacters(ctx, profile.UserID)
		if err != nil {
			fmt.Printf("[DEBUG] AuthStatus Handler: Character resolution failed: %v\n", err)
			// Return basic authenticated status without character details
			characterID := user.CharacterID
			characterName := user.CharacterName
			userID := user.UserID
			return &dto.AuthStatusOutput{
				Body: dto.AuthStatusResponse{
					Authenticated:  true,
					UserID:         &userID,
					CharacterID:    &characterID,
					CharacterName:  &characterName,
					Characters:     []dto.CharacterInfo{},
					CharacterIDs:   []int64{},
					CorporationIDs: []int64{},
					AllianceIDs:    []int64{},
				},
			}, nil
		}
		
		// Build character list and ID arrays for response
		var characters []dto.CharacterInfo
		var characterIDs []int64
		var corporationIDs []int64
		var allianceIDs []int64
		corporationMap := make(map[int64]bool)
		allianceMap := make(map[int64]bool)
		
		for _, char := range userWithCharacters.Characters {
			// Add to character list
			characters = append(characters, dto.CharacterInfo{
				CharacterID:   int(char.CharacterID),
				CharacterName: char.Name,
			})
			
			// Add to character IDs list
			characterIDs = append(characterIDs, char.CharacterID)
			
			// Add unique corporation IDs
			if char.CorporationID > 0 && !corporationMap[char.CorporationID] {
				corporationIDs = append(corporationIDs, char.CorporationID)
				corporationMap[char.CorporationID] = true
			}
			
			// Add unique alliance IDs
			if char.AllianceID > 0 && !allianceMap[char.AllianceID] {
				allianceIDs = append(allianceIDs, char.AllianceID)
				allianceMap[char.AllianceID] = true
			}
		}
		
		// Build final response with all character information
		userID := profile.UserID
		primaryCharID := user.CharacterID
		primaryCharName := user.CharacterName
		
		statusResp := dto.AuthStatusResponse{
			Authenticated:  true,
			UserID:         &userID,
			CharacterID:    &primaryCharID,
			CharacterName:  &primaryCharName,
			Characters:     characters,
			CharacterIDs:   characterIDs,
			CorporationIDs: corporationIDs,
			AllianceIDs:    allianceIDs,
		}
		
		fmt.Printf("[DEBUG] AuthStatus Handler: Successfully resolved %d characters, %d corporations, %d alliances for user %s\n", len(characters), len(corporationIDs), len(allianceIDs), profile.UserID)
		fmt.Printf("[DEBUG] ===== /auth/status HUMA HANDLER END =====\n\n")
		
		return &dto.AuthStatusOutput{Body: statusResp}, nil
	})

	huma.Get(api, basePath+"/user", func(ctx context.Context, input *dto.UserInfoInput) (*dto.UserInfoOutput, error) {
		// Use the new method that accepts header strings
		userInfo, err := authService.GetCurrentUserFromHeaders(ctx, input.Authorization, input.Cookie)
		if err != nil {
			return nil, huma.Error401Unauthorized("User not authenticated", err)
		}

		return &dto.UserInfoOutput{Body: *userInfo}, nil
	})

	// Profile endpoints (require authentication)
	huma.Get(api, basePath+"/profile", func(ctx context.Context, input *dto.ProfileInput) (*dto.ProfileOutput, error) {
		// Validate authentication using auth middleware
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
	})

	huma.Post(api, basePath+"/profile/refresh", func(ctx context.Context, input *dto.ProfileRefreshInput) (*dto.ProfileRefreshOutput, error) {
		// Validate authentication using auth middleware
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
	})

	huma.Get(api, basePath+"/token", func(ctx context.Context, input *dto.TokenInput) (*dto.TokenOutput, error) {
		// Validate authentication using auth middleware
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
	huma.Get(api, basePath+"/profile/public", func(ctx context.Context, input *dto.PublicProfileInput) (*dto.PublicProfileOutput, error) {
		profile, err := authService.GetPublicProfile(ctx, input.CharacterID)
		if err != nil {
			return nil, huma.Error404NotFound("Character not found", err)
		}

		return &dto.PublicProfileOutput{Body: *profile}, nil
	})

	huma.Post(api, basePath+"/logout", func(ctx context.Context, input *dto.LogoutInput) (*dto.LogoutOutput, error) {
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
	userID := extractUserIDFromHeaders(hr.authService, input.Authorization, input.Cookie)

	// Generate login URL without scopes (basic login)
	loginResp, err := hr.authService.InitiateEVELogin(ctx, false, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initiate login", err)
	}

	return &dto.EVELoginOutput{Body: *loginResp}, nil
}

func (hr *Routes) eveRegister(ctx context.Context, input *dto.EVERegisterInput) (*dto.EVERegisterOutput, error) {
	// Extract user ID from context if authenticated
	userID := extractUserIDFromHeaders(hr.authService, input.Authorization, input.Cookie)

	// Generate login URL with full scopes (registration)
	loginResp, err := hr.authService.InitiateEVELogin(ctx, true, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initiate registration", err)
	}

	return &dto.EVERegisterOutput{Body: *loginResp}, nil
}

func (hr *Routes) eveCallback(ctx context.Context, input *dto.EVECallbackInput) (*dto.EVECallbackOutput, error) {
	// Handle the OAuth callback
	jwtToken, _, err := hr.authService.HandleEVECallback(ctx, input.Code, input.State)
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
	// Validate authentication using auth middleware
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
	// Validate authentication using auth middleware
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
	// Validate authentication using auth middleware
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

// Helper methods for cookie handling and user context extraction

// extractUserIDFromHeaders extracts user ID from authentication headers if valid
func extractUserIDFromHeaders(authService *services.AuthService, authHeader, cookieHeader string) string {
	// Try to get current user from headers
	userInfo, err := authService.GetCurrentUserFromHeaders(context.Background(), authHeader, cookieHeader)
	if err != nil {
		// Not authenticated or invalid token - return empty string
		return ""
	}
	
	return userInfo.UserID
}