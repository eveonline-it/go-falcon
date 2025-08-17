package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go-falcon/internal/auth/dto"
	"go-falcon/internal/auth/middleware"
	"go-falcon/pkg/config"
	"go-falcon/pkg/handlers"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// eveBasicLoginHandler initiates EVE SSO login without scopes
func (r *Routes) eveBasicLoginHandler(w http.ResponseWriter, req *http.Request) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(req.Context(), "auth.routes.eve_basic_login")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_basic_login"),
	)

	// Check if user is already authenticated
	userID := ""
	if cookie, err := req.Cookie("falcon_auth_token"); err == nil {
		if user, err := r.authService.ValidateJWT(cookie.Value); err == nil {
			userID = user.UserID
		}
	}

	// Generate login URL without scopes
	loginResp, err := r.authService.InitiateEVELogin(ctx, false, userID)
	if err != nil {
		span.RecordError(err)
		handlers.InternalErrorResponse(w, "Failed to initiate login")
		return
	}

	handlers.JSONResponse(w, loginResp, http.StatusOK)
}

// eveFullLoginHandler initiates EVE SSO login with full scopes
func (r *Routes) eveFullLoginHandler(w http.ResponseWriter, req *http.Request) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(req.Context(), "auth.routes.eve_full_login")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_full_login"),
	)

	// Check if user is already authenticated
	userID := ""
	if cookie, err := req.Cookie("falcon_auth_token"); err == nil {
		if user, err := r.authService.ValidateJWT(cookie.Value); err == nil {
			userID = user.UserID
		}
	}

	// Generate login URL with full scopes
	loginResp, err := r.authService.InitiateEVELogin(ctx, true, userID)
	if err != nil {
		span.RecordError(err)
		handlers.InternalErrorResponse(w, "Failed to initiate registration")
		return
	}

	handlers.JSONResponse(w, loginResp, http.StatusOK)
}

// eveCallbackHandler handles EVE SSO OAuth callback
func (r *Routes) eveCallbackHandler(w http.ResponseWriter, req *http.Request) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(req.Context(), "auth.routes.eve_callback")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_callback"),
	)

	// Get code and state from query parameters
	code := req.URL.Query().Get("code")
	state := req.URL.Query().Get("state")

	if code == "" || state == "" {
		handlers.BadRequestResponse(w, "Missing code or state parameter")
		return
	}

	// Handle the callback
	jwtToken, userInfo, err := r.authService.HandleEVECallback(ctx, code, state)
	if err != nil {
		span.RecordError(err)
		handlers.InternalErrorResponse(w, "Authentication failed")
		return
	}

	// Set secure cookie
	r.setAuthCookie(w, jwtToken)

	// Redirect to frontend or return JSON based on Accept header
	frontendURL := config.GetFrontendURL()
	if frontendURL != "" && req.Header.Get("Accept") != "application/json" {
		http.Redirect(w, req, frontendURL, http.StatusFound)
		return
	}

	// Return JSON response
	handlers.JSONResponse(w, map[string]interface{}{
		"success": true,
		"user":    userInfo,
		"message": "Authentication successful",
	}, http.StatusOK)
}

// eveTokenExchangeHandler exchanges EVE token for JWT (mobile apps)
func (r *Routes) eveTokenExchangeHandler(w http.ResponseWriter, req *http.Request) {
	tracer := otel.Tracer("go-falcon/auth")
	ctx, span := tracer.Start(req.Context(), "auth.routes.eve_token_exchange")
	defer span.End()

	span.SetAttributes(
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_token_exchange"),
	)

	// Create a handler that validates and processes the request
	validationHandler := r.middleware.Validation.ValidateEVETokenRequest(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// This will only be called if validation passes
		var tokenReq dto.EVETokenExchangeRequest
		json.NewDecoder(req.Body).Decode(&tokenReq)

		// Exchange EVE token for JWT
		tokenResp, err := r.authService.ExchangeEVEToken(ctx, &tokenReq)
		if err != nil {
			span.RecordError(err)
			handlers.UnauthorizedResponse(w)
			return
		}

		handlers.JSONResponse(w, tokenResp, http.StatusOK)
	}))

	// Execute the validation handler
	validationHandler.ServeHTTP(w, req)
}

// authStatusHandler returns authentication status
func (r *Routes) authStatusHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	
	statusResp, err := r.authService.GetAuthStatus(ctx, req)
	if err != nil {
		handlers.InternalErrorResponse(w, "Failed to check auth status")
		return
	}

	handlers.JSONResponse(w, statusResp, http.StatusOK)
}

// getCurrentUserHandler returns current user information
func (r *Routes) getCurrentUserHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	
	userInfo, err := r.authService.GetCurrentUser(ctx, req)
	if err != nil {
		handlers.UnauthorizedResponse(w)
		return
	}

	handlers.JSONResponse(w, userInfo, http.StatusOK)
}

// profileHandler returns full user profile (authenticated)
func (r *Routes) profileHandler(w http.ResponseWriter, req *http.Request) {
	user := middleware.GetAuthenticatedUser(req)
	if user == nil {
		handlers.UnauthorizedResponse(w)
		return
	}

	profile, err := r.authService.GetUserProfile(req.Context(), user.CharacterID)
	if err != nil {
		handlers.InternalErrorResponse(w, "Failed to get profile")
		return
	}

	handlers.JSONResponse(w, profile, http.StatusOK)
}

// refreshProfileHandler refreshes user profile from ESI
func (r *Routes) refreshProfileHandler(w http.ResponseWriter, req *http.Request) {
	user := middleware.GetAuthenticatedUser(req)
	if user == nil {
		handlers.UnauthorizedResponse(w)
		return
	}

	profile, err := r.authService.RefreshUserProfile(req.Context(), user.CharacterID)
	if err != nil {
		handlers.InternalErrorResponse(w, "Failed to refresh profile")
		return
	}

	handlers.JSONResponse(w, profile, http.StatusOK)
}

// publicProfileHandler returns public character information
func (r *Routes) publicProfileHandler(w http.ResponseWriter, req *http.Request) {
	characterIDStr := req.URL.Query().Get("character_id")
	if characterIDStr == "" {
		handlers.BadRequestResponse(w, "character_id parameter required")
		return
	}

	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		handlers.BadRequestResponse(w, "Invalid character_id")
		return
	}

	profile, err := r.authService.GetPublicProfile(req.Context(), characterID)
	if err != nil {
		handlers.NotFoundResponse(w, "Character")
		return
	}

	handlers.JSONResponse(w, profile, http.StatusOK)
}

// tokenHandler returns bearer token for authenticated user
func (r *Routes) tokenHandler(w http.ResponseWriter, req *http.Request) {
	user := middleware.GetAuthenticatedUser(req)
	if user == nil {
		handlers.UnauthorizedResponse(w)
		return
	}

	tokenResp, err := r.authService.GetBearerToken(req.Context(), user.UserID, user.CharacterID, user.CharacterName, user.Scopes)
	if err != nil {
		handlers.InternalErrorResponse(w, "Failed to generate token")
		return
	}

	handlers.JSONResponse(w, tokenResp, http.StatusOK)
}

// logoutHandler clears authentication cookie
func (r *Routes) logoutHandler(w http.ResponseWriter, req *http.Request) {
	// Clear the auth cookie
	cookie := &http.Cookie{
		Name:     "falcon_auth_token",
		Value:    "",
		Path:     "/",
		Domain:   config.GetCookieDomain(),
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	handlers.JSONResponse(w, dto.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, http.StatusOK)
}

// eveRefreshHandler handles token refresh (not implemented yet)
func (r *Routes) eveRefreshHandler(w http.ResponseWriter, req *http.Request) {
	handlers.ErrorResponse(w, "Token refresh not implemented", http.StatusNotImplemented)
}

// eveVerifyHandler handles token verification (not implemented yet)
func (r *Routes) eveVerifyHandler(w http.ResponseWriter, req *http.Request) {
	handlers.ErrorResponse(w, "Token verification not implemented", http.StatusNotImplemented)
}

// setAuthCookie sets the authentication cookie
func (r *Routes) setAuthCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "falcon_auth_token",
		Value:    token,
		Path:     "/",
		Domain:   config.GetCookieDomain(),
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}