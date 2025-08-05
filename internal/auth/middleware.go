package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// AuthContext key for storing user info in request context
type AuthContextKey string

const (
	AuthContextKeyUser = AuthContextKey("user")
)

// AuthenticatedUser represents an authenticated EVE Online character
type AuthenticatedUser struct {
	CharacterID   int    `json:"character_id"`
	CharacterName string `json:"character_name"`
	Scopes        string `json:"scopes"`
}

// JWTMiddleware provides JWT-based authentication middleware
func (m *Module) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get JWT from cookie or Authorization header
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
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Validate JWT
		claims, err := m.eveSSOHandler.ValidateJWT(jwtToken)
		if err != nil {
			slog.Warn("Invalid JWT token in middleware", slog.String("error", err.Error()))
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}

		// Extract user information from claims
		user := &AuthenticatedUser{}
		
		if characterID, ok := (*claims)["character_id"].(float64); ok {
			user.CharacterID = int(characterID)
		}
		
		if characterName, ok := (*claims)["character_name"].(string); ok {
			user.CharacterName = characterName
		}
		
		if scopes, ok := (*claims)["scopes"].(string); ok {
			user.Scopes = scopes
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), AuthContextKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalJWTMiddleware provides optional JWT-based authentication middleware
// If a valid token is present, it adds the user to context, but doesn't require authentication
func (m *Module) OptionalJWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get JWT from cookie or Authorization header
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

		if jwtToken != "" {
			// Validate JWT if present
			if claims, err := m.eveSSOHandler.ValidateJWT(jwtToken); err == nil {
				// Extract user information from claims
				user := &AuthenticatedUser{}
				
				if characterID, ok := (*claims)["character_id"].(float64); ok {
					user.CharacterID = int(characterID)
				}
				
				if characterName, ok := (*claims)["character_name"].(string); ok {
					user.CharacterName = characterName
				}
				
				if scopes, ok := (*claims)["scopes"].(string); ok {
					user.Scopes = scopes
				}

				// Add user to request context
				ctx := context.WithValue(r.Context(), AuthContextKeyUser, user)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetAuthenticatedUser extracts the authenticated user from request context
func GetAuthenticatedUser(r *http.Request) (*AuthenticatedUser, bool) {
	user, ok := r.Context().Value(AuthContextKeyUser).(*AuthenticatedUser)
	return user, ok
}

// RequireScopes middleware ensures the authenticated user has specific EVE Online scopes
func (m *Module) RequireScopes(requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetAuthenticatedUser(r)
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has all required scopes
			userScopes := strings.Split(user.Scopes, " ")
			scopeMap := make(map[string]bool)
			for _, scope := range userScopes {
				scopeMap[strings.TrimSpace(scope)] = true
			}

			for _, requiredScope := range requiredScopes {
				if !scopeMap[requiredScope] {
					slog.Warn("User missing required scope", 
						slog.String("character_name", user.CharacterName),
						slog.String("required_scope", requiredScope),
						slog.String("user_scopes", user.Scopes))
					http.Error(w, "Insufficient permissions", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}