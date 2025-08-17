package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Module struct {
	*module.BaseModule
	eveSSOHandler *EVESSOHandler
	groupsModule  interface{} // Will be set after groups module creation
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	return &Module{
		BaseModule:    module.NewBaseModule("auth", mongodb, redis, sdeService),
		eveSSOHandler: NewEVESSOHandler(),
	}
}

// SetGroupsModule sets the groups module reference after both modules are created
func (m *Module) SetGroupsModule(groupsModule interface{}) {
	m.groupsModule = groupsModule
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Post("/login", m.loginHandler)
	r.Post("/register", m.registerHandler)
	
	// EVE Online SSO routes (not restricted)
	r.Get("/eve/login", m.eveBasicLoginHandler)    // Basic login without scopes
	r.Get("/eve/register", m.eveLoginHandler)      // Full registration with scopes
	r.Get("/eve/callback", m.eveCallbackHandler)
	r.Post("/eve/refresh", m.eveRefreshHandler)
	r.Get("/eve/verify", m.eveVerifyHandler)
	r.Post("/eve/token", m.mobileTokenHandler)
	
	// Profile routes (require authentication)
	r.With(m.JWTMiddleware).Get("/profile", m.profileHandler)
	r.With(m.JWTMiddleware).Post("/profile/refresh", m.refreshProfileHandler)
	r.Get("/profile/public", m.publicProfileHandler)
	
	// User info endpoint (checks JWT cookie and returns user data)
	r.Get("/user", m.getCurrentUserHandler)
	
	// Authentication status and logout endpoints (not restricted)
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
	span, r := handlers.StartHTTPSpan(r, "auth.login",
		attribute.String("service", "auth"),
		attribute.String("operation", "login_redirect"),
	)
	defer span.End()

	slog.Info("Login redirect to EVE SSO basic login", 
		slog.String("remote_addr", r.RemoteAddr), 
		slog.String("module", m.Name()))

	// Generate auth URL without scopes for basic login
	authURL, state, err := m.eveSSOHandler.GenerateAuthURLWithScopes("")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate EVE login URL")
		slog.Error("Failed to generate EVE login URL", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}
	
	span.SetAttributes(
		attribute.String("auth.state", state),
		attribute.Bool("auth.success", true),
		attribute.Bool("auth.basic_login", true),
	)

	// Store state in session/cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "eve_auth_state",
		Value:    state,
		Domain:   ".eveonline.it",
		Path:     "/",
		MaxAge:   900, // 15 minutes
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	response := map[string]string{
		"auth_url": authURL,
		"state":    state,
		"type":     "login",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) registerHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "auth.register",
		attribute.String("service", "auth"),
		attribute.String("operation", "register_redirect"),
	)
	defer span.End()

	slog.Info("Register redirect to EVE SSO with full scopes", 
		slog.String("remote_addr", r.RemoteAddr), 
		slog.String("module", m.Name()))

	// Generate auth URL with full scopes from environment
	authURL, state, err := m.eveSSOHandler.GenerateAuthURL()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate EVE register URL")
		slog.Error("Failed to generate EVE register URL", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}
	
	span.SetAttributes(
		attribute.String("auth.state", state),
		attribute.Bool("auth.success", true),
		attribute.Bool("auth.full_register", true),
	)

	// Store state in session/cookie for CSRF protection
	http.SetCookie(w, &http.Cookie{
		Name:     "eve_auth_state",
		Value:    state,
		Domain:   ".eveonline.it",
		Path:     "/",
		MaxAge:   900, // 15 minutes
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	response := map[string]string{
		"auth_url": authURL,
		"state":    state,
		"type":     "register",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EVE Online SSO Handlers

func (m *Module) eveLoginHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "auth.eve.register",
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_register"),
	)
	defer span.End()

	authURL, state, err := m.eveSSOHandler.GenerateAuthURL()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate EVE register URL")
		slog.Error("Failed to generate EVE register URL", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}
	
	// Debug logging to check URL length
	slog.Info("Generated EVE register URL (with scopes)", 
		slog.Int("url_length", len(authURL)),
		slog.String("url_preview", authURL[:min(200, len(authURL))]))
		
	// Check scope parameter specifically
	if parsedURL, err := url.Parse(authURL); err == nil {
		scopeParam := parsedURL.Query().Get("scope")
		slog.Info("Scope parameter debug",
			slog.Int("scope_length", len(scopeParam)),
			slog.String("scope_preview", scopeParam[:min(100, len(scopeParam))]))
	}

	span.SetAttributes(
		attribute.String("auth.state", state),
		attribute.Bool("auth.success", true),
		attribute.Bool("auth.full_register", true),
	)

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
		"type":     "register",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) eveBasicLoginHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "auth.eve.login",
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_login"),
	)
	defer span.End()

	// Generate auth URL without scopes for basic login
	authURL, state, err := m.eveSSOHandler.GenerateAuthURLWithScopes("")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate EVE login URL")
		slog.Error("Failed to generate EVE login URL", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}
	
	slog.Info("Generated EVE login URL (no scopes)", 
		slog.Int("url_length", len(authURL)),
		slog.String("url_preview", authURL[:min(200, len(authURL))]))

	span.SetAttributes(
		attribute.String("auth.state", state),
		attribute.Bool("auth.success", true),
		attribute.Bool("auth.basic_login", true),
	)

	// Store state in session/cookie for additional security
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
		"type":     "login",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *Module) eveCallbackHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "auth.eve.callback",
		attribute.String("service", "auth"),
		attribute.String("operation", "eve_callback"),
	)
	defer span.End()

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	span.SetAttributes(
		attribute.String("auth.state", state),
		attribute.Bool("auth.code_present", code != ""),
	)

	if code == "" || state == "" {
		span.SetStatus(codes.Error, "Missing required parameters")
		span.SetAttributes(attribute.String("error.type", "missing_parameters"))
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "Missing state cookie")
		span.SetAttributes(attribute.String("error.type", "missing_state_cookie"))
		slog.Warn("Missing state cookie", 
			slog.String("error", err.Error()), 
			slog.String("received_state", state),
			slog.Int("total_cookies", len(r.Cookies())))
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	if cookie.Value != state {
		span.SetStatus(codes.Error, "Invalid state parameter")
		span.SetAttributes(
			attribute.String("error.type", "state_mismatch"),
			attribute.String("expected_state", cookie.Value[:8]+"..."), // Only log prefix for security
		)
		slog.Warn("Invalid state parameter", 
			slog.String("expected", cookie.Value), 
			slog.String("received", state))
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	
	span.SetAttributes(attribute.Bool("auth.state_valid", true))
	slog.Info("State validation successful", slog.String("state", state))

	// Exchange code for token
	tokenResp, err := m.eveSSOHandler.ExchangeCodeForToken(r.Context(), code, state)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to exchange code for token")
		span.SetAttributes(attribute.String("error.type", "token_exchange_failed"))
		slog.Error("Failed to exchange code for token", slog.String("error", err.Error()))
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Bool("auth.token_exchange_success", true))

	// Verify token and get character info
	charInfo, err := m.eveSSOHandler.VerifyToken(r.Context(), tokenResp.AccessToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to verify token")
		span.SetAttributes(attribute.String("error.type", "token_verification_failed"))
		slog.Error("Failed to verify token", slog.String("error", err.Error()))
		http.Error(w, "Token verification failed", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Bool("auth.token_verification_success", true),
		attribute.Int("eve.character_id", charInfo.CharacterID),
		attribute.String("eve.character_name", charInfo.CharacterName),
	)

	// Check for existing falcon_auth_token cookie to get existing user_id
	var existingUserID string
	if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
		if claims, err := m.eveSSOHandler.ValidateJWT(cookie.Value); err == nil {
			if userID, ok := (*claims)["user_id"].(string); ok {
				existingUserID = userID
				slog.Info("Found existing user_id from cookie", slog.String("user_id", userID))
			}
		}
	}

	// Look up character in database to get user_id
	profile, err := m.GetUserProfile(r.Context(), charInfo.CharacterID)
	var userID string
	if err != nil {
		// Character not found - use existing userID or generate new one
		if existingUserID != "" {
			userID = existingUserID
			slog.Info("Using existing user_id for new character", slog.String("user_id", userID))
		} else {
			userID = uuid.New().String()
			slog.Info("Generated new user_id", slog.String("user_id", userID))
		}
	} else {
		// Character exists - use its user_id if it exists, otherwise generate new one
		if profile.UserID != "" {
			userID = profile.UserID
			slog.Info("Using existing character's user_id", slog.String("user_id", userID))
		} else {
			userID = uuid.New().String()
			slog.Info("Generated new user_id for existing character", slog.String("user_id", userID))
		}
	}

	span.SetAttributes(
		attribute.String("auth.user_id", userID),
		attribute.Bool("auth.existing_user", existingUserID != ""),
	)

	// Create or update user profile with user_id and tokens
	profile, err = m.CreateOrUpdateUserProfile(r.Context(), charInfo, userID, tokenResp.AccessToken, tokenResp.RefreshToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create/update user profile")
		span.SetAttributes(attribute.String("error.type", "profile_creation_failed"))
		slog.Error("Failed to create/update user profile", slog.String("error", err.Error()))
		http.Error(w, "Failed to create user profile", http.StatusInternalServerError)
		return
	}
	
	span.SetAttributes(attribute.Bool("auth.profile_updated", true))
	slog.Info("User profile updated successfully", 
		slog.Int("character_id", profile.CharacterID),
		slog.String("user_id", profile.UserID))

	// Generate JWT for internal use
	jwtToken, err := m.eveSSOHandler.GenerateJWT(charInfo, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate JWT")
		span.SetAttributes(attribute.String("error.type", "jwt_generation_failed"))
		slog.Error("Failed to generate JWT", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Bool("auth.jwt_generated", true))

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

	span.SetAttributes(
		attribute.Bool("auth.cookies_set", true),
		attribute.Bool("auth.success", true),
	)

	slog.Info("EVE SSO authentication successful", 
		slog.Int("character_id", charInfo.CharacterID),
		slog.String("character_name", charInfo.CharacterName))

	// Redirect to React client
	frontendURL := config.GetFrontendURL()
	
	span.SetAttributes(attribute.String("auth.redirect_url", frontendURL))
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

	// Get user profile to obtain user_id
	profile, err := m.GetUserProfile(r.Context(), charInfo.CharacterID)
	if err != nil {
		slog.Error("Failed to get user profile for refresh", slog.String("error", err.Error()))
		http.Error(w, "User profile not found", http.StatusInternalServerError)
		return
	}

	// Generate new JWT
	jwtToken, err := m.eveSSOHandler.GenerateJWT(charInfo, profile.UserID)
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
		"user_id":       profile.UserID,
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
		"user_id":        (*claims)["user_id"],
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
		"user_id":        (*claims)["user_id"],
		"character_id":   (*claims)["character_id"],
		"character_name": (*claims)["character_name"],
		"scopes":         (*claims)["scopes"],
		"expires_at":     (*claims)["exp"],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// authStatusHandler returns authentication status with user permissions
func (m *Module) authStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get JWT from cookie
	cookie, err := r.Cookie("falcon_auth_token")
	if err != nil {
		// No cookie found, user is not authenticated
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"permissions": []string{},
		})
		return
	}

	// Validate JWT and extract claims
	claims, err := m.eveSSOHandler.ValidateJWT(cookie.Value)
	if err != nil {
		// Invalid token, user is not authenticated
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
			"permissions": []string{},
		})
		return
	}

	// Extract character ID from claims
	var characterID int
	if charIDFloat, ok := (*claims)["character_id"].(float64); ok {
		characterID = int(charIDFloat)
	}

	// Get user permissions and groups from groups module
	var permissions []string
	var groups []string
	if m.groupsModule != nil && characterID > 0 {
		// Use reflection to call GetUserPermissions method
		groupsModuleValue := reflect.ValueOf(m.groupsModule)
		method := groupsModuleValue.MethodByName("GetUserPermissions")
		if method.IsValid() {
			ctx := reflect.ValueOf(r.Context())
			charID := reflect.ValueOf(characterID)
			results := method.Call([]reflect.Value{ctx, charID})
			
			if len(results) == 2 && results[1].IsNil() { // error is nil
				userPerms := results[0].Interface()
				// Use reflection to access fields
				permsValue := reflect.ValueOf(userPerms).Elem()
				
				// Get permissions
				permissionsField := permsValue.FieldByName("Permissions")
				if permissionsField.IsValid() {
					permsMap := permissionsField.Interface().(map[string]map[string]bool)
					for resource, actions := range permsMap {
						for action, allowed := range actions {
							if allowed {
								permissions = append(permissions, resource+":"+action)
							}
						}
					}
				}
				
				// Get groups
				groupsField := permsValue.FieldByName("Groups")
				if groupsField.IsValid() {
					groupsList := groupsField.Interface().([]string)
					groups = groupsList
				}
			}
		}
	}

	// Valid token, user is authenticated
	response := map[string]interface{}{
		"authenticated": true,
		"permissions": permissions,
		"groups": groups,
	}
	
	// Add user info from claims if available
	if userID, ok := (*claims)["user_id"].(string); ok {
		response["user_id"] = userID
		
		// Get all characters for this user_id
		allCharacters, err := m.GetAllUserCharacters(r.Context(), userID)
		if err != nil {
			slog.Warn("Failed to get all user characters", 
				slog.String("user_id", userID),
				slog.String("error", err.Error()))
			// Still return response without characters if this fails
		} else {
			// Create simplified character list for response
			var characters []map[string]interface{}
			for _, char := range allCharacters {
				characters = append(characters, map[string]interface{}{
					"character_id":   char.CharacterID,
					"character_name": char.CharacterName,
					"scopes":         char.Scopes,
					"valid":          char.Valid,
					"created_at":     char.CreatedAt,
					"updated_at":     char.UpdatedAt,
				})
			}
			response["characters"] = characters
		}
	}
	if charName, ok := (*claims)["character_name"].(string); ok {
		response["character_name"] = charName
	}
	if characterID > 0 {
		response["character_id"] = characterID
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

// mobileTokenHandler converts EVE SSO tokens to JWT tokens for mobile apps
func (m *Module) mobileTokenHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "auth.eve.mobile_token",
		attribute.String("service", "auth"),
		attribute.String("operation", "mobile_token"),
	)
	defer span.End()

	var request struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.AccessToken == "" {
		span.SetStatus(codes.Error, "Missing access_token")
		http.Error(w, "Missing access_token", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.Bool("auth.access_token_present", true),
		attribute.Bool("auth.refresh_token_present", request.RefreshToken != ""),
	)

	// Verify the EVE access token and get character info
	charInfo, err := m.eveSSOHandler.VerifyToken(r.Context(), request.AccessToken)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to verify EVE token")
		slog.Error("Failed to verify EVE token for mobile", slog.String("error", err.Error()))
		http.Error(w, "Invalid EVE access token", http.StatusUnauthorized)
		return
	}

	span.SetAttributes(
		attribute.Bool("auth.token_verification_success", true),
		attribute.Int("eve.character_id", charInfo.CharacterID),
		attribute.String("eve.character_name", charInfo.CharacterName),
	)

	// Look up character in database to get user_id
	profile, err := m.GetUserProfile(r.Context(), charInfo.CharacterID)
	var userID string
	if err != nil {
		// Character not found - create new user
		userID = uuid.New().String()
		slog.Info("Generated new user_id for mobile auth", slog.String("user_id", userID))
		
		// Create user profile with the provided tokens
		profile, err = m.CreateOrUpdateUserProfile(r.Context(), charInfo, userID, request.AccessToken, request.RefreshToken)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to create user profile")
			slog.Error("Failed to create user profile for mobile", slog.String("error", err.Error()))
			http.Error(w, "Failed to create user profile", http.StatusInternalServerError)
			return
		}
	} else {
		// Character exists - use its user_id
		userID = profile.UserID
		if userID == "" {
			userID = uuid.New().String()
			slog.Info("Generated new user_id for existing character in mobile auth", slog.String("user_id", userID))
		}
		
		// Update profile with new tokens if provided
		if request.RefreshToken != "" {
			profile, err = m.CreateOrUpdateUserProfile(r.Context(), charInfo, userID, request.AccessToken, request.RefreshToken)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Failed to update user profile")
				slog.Error("Failed to update user profile for mobile", slog.String("error", err.Error()))
				http.Error(w, "Failed to update user profile", http.StatusInternalServerError)
				return
			}
		}
	}

	span.SetAttributes(
		attribute.String("auth.user_id", userID),
		attribute.Bool("auth.profile_updated", true),
	)

	// Generate JWT for internal use
	jwtToken, err := m.eveSSOHandler.GenerateJWT(charInfo, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate JWT")
		slog.Error("Failed to generate JWT for mobile", slog.String("error", err.Error()))
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Bool("auth.jwt_generated", true),
		attribute.Bool("auth.success", true),
	)

	slog.Info("Mobile token exchange successful", 
		slog.Int("character_id", charInfo.CharacterID),
		slog.String("character_name", charInfo.CharacterName),
		slog.String("user_id", userID))

	response := map[string]interface{}{
		"jwt_token":      jwtToken,
		"user_id":        userID,
		"character_id":   charInfo.CharacterID,
		"character_name": charInfo.CharacterName,
		"scopes":         charInfo.Scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}