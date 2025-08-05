package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
	eveSSOHandler *EVESSOHandler
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	return &Module{
		BaseModule:    module.NewBaseModule("auth", mongodb, redis, sdeService),
		eveSSOHandler: NewEVESSOHandler(),
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Post("/login", m.loginHandler)
	r.Post("/register", m.registerHandler)
	r.Get("/status", m.statusHandler)
	
	// EVE Online SSO routes
	r.Get("/eve/login", m.eveLoginHandler)
	r.Get("/eve/callback", m.eveCallbackHandler)
	r.Post("/eve/refresh", m.eveRefreshHandler)
	r.Get("/eve/verify", m.eveVerifyHandler)
	
	// Profile routes (require authentication)
	r.With(m.JWTMiddleware).Get("/profile", m.profileHandler)
	r.With(m.JWTMiddleware).Post("/profile/refresh", m.refreshProfileHandler)
	r.Get("/profile/public", m.publicProfileHandler)
	
	// User info endpoint (checks JWT cookie and returns user data)
	r.Get("/user", m.getCurrentUserHandler)
	
	// Authentication status and logout endpoints
	r.Get("/status", m.authStatusHandler)
	r.Post("/logout", m.logoutHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting auth-specific background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Start cleanup routine for expired EVE SSO states
	go m.runStateCleanup(ctx)
	
	// Add auth-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Auth background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Auth background tasks stopped")
			return
		default:
			// Auth-specific background work would go here
			// For now, just wait
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}

func (m *Module) runStateCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("EVE SSO state cleanup stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("EVE SSO state cleanup stopped")
			return
		case <-ticker.C:
			m.eveSSOHandler.CleanupExpiredStates()
		}
	}
}

func (m *Module) loginHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Login attempt", slog.String("remote_addr", r.RemoteAddr), slog.String("module", m.Name()))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Auth module - login endpoint","status":"not_implemented"}`))
}

func (m *Module) registerHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Registration attempt", slog.String("remote_addr", r.RemoteAddr), slog.String("module", m.Name()))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Auth module - register endpoint","status":"not_implemented"}`))
}

func (m *Module) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"module":"auth","status":"running","version":"1.0.0"}`))
}

// EVE Online SSO Handlers

func (m *Module) eveLoginHandler(w http.ResponseWriter, r *http.Request) {
	authURL, state, err := m.eveSSOHandler.GenerateAuthURL()
	if err != nil {
		slog.Error("Failed to generate EVE auth URL", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}

	// Store state in session/cookie for additional security if needed
	http.SetCookie(w, &http.Cookie{
		Name:     "eve_auth_state",
		Value:    state,
		Domain:   ".eveonline.it", // Allow access from all subdomains
		Path:     "/",
		MaxAge:   900, // 15 minutes
		HttpOnly: true,
		Secure:   true, // Always secure for production
		SameSite: http.SameSiteLaxMode,
	})

	response := map[string]string{
		"auth_url": authURL,
		"state":    state,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) eveCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Verify state matches what we stored
	slog.Info("Validating EVE SSO callback", 
		slog.String("received_state", state),
		slog.String("user_agent", r.UserAgent()),
		slog.String("referer", r.Referer()))
	
	cookie, err := r.Cookie("eve_auth_state")
	if err != nil {
		slog.Warn("Missing state cookie", 
			slog.String("error", err.Error()), 
			slog.String("received_state", state),
			slog.Int("total_cookies", len(r.Cookies())))
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	if cookie.Value != state {
		slog.Warn("Invalid state parameter", 
			slog.String("expected", cookie.Value), 
			slog.String("received", state))
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	
	slog.Info("State validation successful", slog.String("state", state))

	// Exchange code for token
	tokenResp, err := m.eveSSOHandler.ExchangeCodeForToken(r.Context(), code, state)
	if err != nil {
		slog.Error("Failed to exchange code for token", slog.String("error", err.Error()))
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Verify token and get character info
	charInfo, err := m.eveSSOHandler.VerifyToken(r.Context(), tokenResp.AccessToken)
	if err != nil {
		slog.Error("Failed to verify token", slog.String("error", err.Error()))
		http.Error(w, "Token verification failed", http.StatusInternalServerError)
		return
	}

	// Create or update user profile
	profile, err := m.CreateOrUpdateUserProfile(r.Context(), charInfo, tokenResp.RefreshToken)
	if err != nil {
		slog.Error("Failed to create/update user profile", slog.String("error", err.Error()))
		// Continue with authentication even if profile creation fails
	} else {
		slog.Info("User profile updated successfully", slog.Int("character_id", profile.CharacterID))
	}

	// Generate JWT for internal use
	jwtToken, err := m.eveSSOHandler.GenerateJWT(charInfo)
	if err != nil {
		slog.Error("Failed to generate JWT", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "eve_auth_state",
		Value:    "",
		Domain:   ".eveonline.it", // Same domain as when it was set
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Set authentication cookie for cross-subdomain access
	http.SetCookie(w, &http.Cookie{
		Name:     "falcon_auth_token",
		Value:    jwtToken,
		Domain:   ".eveonline.it", // Allow access from all subdomains
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		Secure:   true, // Always secure for production
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("EVE SSO authentication successful", 
		slog.Int("character_id", charInfo.CharacterID),
		slog.String("character_name", charInfo.CharacterName))

	// Redirect to React client
	frontendURL := config.GetFrontendURL()
	
	slog.Info("Redirecting to frontend", slog.String("url", frontendURL))
	http.Redirect(w, r, frontendURL, http.StatusFound)
}

func (m *Module) eveRefreshHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.RefreshToken == "" {
		http.Error(w, "Missing refresh_token", http.StatusBadRequest)
		return
	}

	tokenResp, err := m.eveSSOHandler.RefreshToken(r.Context(), request.RefreshToken)
	if err != nil {
		slog.Error("Failed to refresh token", slog.String("error", err.Error()))
		http.Error(w, "Token refresh failed", http.StatusUnauthorized)
		return
	}

	// Verify the new token and get updated character info
	charInfo, err := m.eveSSOHandler.VerifyToken(r.Context(), tokenResp.AccessToken)
	if err != nil {
		slog.Error("Failed to verify refreshed token", slog.String("error", err.Error()))
		http.Error(w, "Token verification failed", http.StatusInternalServerError)
		return
	}

	// Generate new JWT
	jwtToken, err := m.eveSSOHandler.GenerateJWT(charInfo)
	if err != nil {
		slog.Error("Failed to generate JWT after refresh", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"expires_in":    tokenResp.ExpiresIn,
		"jwt_token":     jwtToken,
		"character_id":  charInfo.CharacterID,
		"character_name": charInfo.CharacterName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) eveVerifyHandler(w http.ResponseWriter, r *http.Request) {
	// Get JWT from cookie or Authorization header
	var jwtToken string
	
	// Try cookie first
	if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
		jwtToken = cookie.Value
	} else {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			jwtToken = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if jwtToken == "" {
		http.Error(w, "No authentication token provided", http.StatusUnauthorized)
		return
	}

	claims, err := m.eveSSOHandler.ValidateJWT(jwtToken)
	if err != nil {
		slog.Warn("Invalid JWT token", slog.String("error", err.Error()))
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"valid":          true,
		"character_id":   (*claims)["character_id"],
		"character_name": (*claims)["character_name"],
		"scopes":         (*claims)["scopes"],
		"expires_at":     (*claims)["exp"],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getCurrentUserHandler returns the current user info from JWT cookie
func (m *Module) getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get JWT from cookie
	cookie, err := r.Cookie("falcon_auth_token")
	if err != nil {
		// No cookie found, user is not authenticated
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Validate JWT
	claims, err := m.eveSSOHandler.ValidateJWT(cookie.Value)
	if err != nil {
		// Invalid token, user is not authenticated
		slog.Warn("Invalid JWT token in getCurrentUser", slog.String("error", err.Error()))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Return user information
	response := map[string]interface{}{
		"authenticated":  true,
		"character_id":   (*claims)["character_id"],
		"character_name": (*claims)["character_name"],
		"scopes":         (*claims)["scopes"],
		"expires_at":     (*claims)["exp"],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// authStatusHandler returns simple authentication status
func (m *Module) authStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get JWT from cookie
	cookie, err := r.Cookie("falcon_auth_token")
	if err != nil {
		// No cookie found, user is not authenticated
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{
			"authenticated": false,
		})
		return
	}

	// Validate JWT
	_, err = m.eveSSOHandler.ValidateJWT(cookie.Value)
	if err != nil {
		// Invalid token, user is not authenticated
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{
			"authenticated": false,
		})
		return
	}

	// Valid token, user is authenticated
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"authenticated": true,
	})
}

// logoutHandler clears the authentication cookie
func (m *Module) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the authentication cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "falcon_auth_token",
		Value:    "",
		Domain:   ".eveonline.it", // Same domain as when it was set
		Path:     "/",
		MaxAge:   -1, // Delete immediately
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("User logged out", slog.String("remote_addr", r.RemoteAddr))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}