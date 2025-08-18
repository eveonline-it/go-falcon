package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// UserCharacterResolver interface for resolving user characters
type UserCharacterResolver interface {
	GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error)
}

// UserWithCharacters represents a user with all their characters
type UserWithCharacters struct {
	ID         string           `json:"id"`
	Characters []UserCharacter  `json:"characters"`
}

// UserCharacter represents a character linked to a user
type UserCharacter struct {
	CharacterID   int64  `json:"character_id"`
	Name          string `json:"name"`
	CorporationID int64  `json:"corporation_id"`
	AllianceID    int64  `json:"alliance_id,omitempty"`
	IsPrimary     bool   `json:"is_primary"`
}

// EnhancedAuthMiddleware provides enhanced authentication with character resolution
type EnhancedAuthMiddleware struct {
	jwtValidator    JWTValidator
	characterResolver UserCharacterResolver
}

// NewEnhancedAuthMiddleware creates a new enhanced authentication middleware
func NewEnhancedAuthMiddleware(validator JWTValidator, resolver UserCharacterResolver) *EnhancedAuthMiddleware {
	return &EnhancedAuthMiddleware{
		jwtValidator:    validator,
		characterResolver: resolver,
	}
}

// AuthenticationMiddleware extracts and validates authentication from cookie or bearer token
func (m *EnhancedAuthMiddleware) AuthenticationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware.AuthenticationMiddleware: Processing request %s %s\n", r.Method, r.URL.Path)
			var token string
			var requestType string
			
			// Check for Bearer token first
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				fmt.Printf("[DEBUG] EnhancedAuthMiddleware: Found Authorization header: %q\n", authHeader)
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
					requestType = "bearer"
					fmt.Printf("[DEBUG] EnhancedAuthMiddleware: Extracted bearer token (length=%d)\n", len(token))
				}
			} else {
				fmt.Printf("[DEBUG] EnhancedAuthMiddleware: No Authorization header found\n")
			}
			
			// Fallback to cookie
			if token == "" {
				fmt.Printf("[DEBUG] EnhancedAuthMiddleware: No bearer token, checking cookie\n")
				if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
					token = cookie.Value
					requestType = "cookie"
					fmt.Printf("[DEBUG] EnhancedAuthMiddleware: Extracted cookie token (length=%d)\n", len(token))
				} else {
					fmt.Printf("[DEBUG] EnhancedAuthMiddleware: No falcon_auth_token cookie found: %v\n", err)
				}
			}
			
			if token == "" {
				fmt.Printf("[DEBUG] EnhancedAuthMiddleware: No authentication token found\n")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Validate JWT and extract claims
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware: Validating JWT token\n")
			user, err := m.jwtValidator.ValidateJWT(token)
			if err != nil {
				fmt.Printf("[DEBUG] EnhancedAuthMiddleware: JWT validation failed: %v\n", err)
				slog.Warn("Invalid authentication token", "error", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			
			// Create base auth context
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware: JWT validation successful, creating auth context\n")
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware: UserID=%s, CharacterID=%d, RequestType=%s\n", user.UserID, user.CharacterID, requestType)
			authCtx := &AuthContext{
				UserID:          user.UserID,
				PrimaryCharID:   int64(user.CharacterID),
				RequestType:     requestType,
				IsAuthenticated: true,
			}
			
			// Add to request context
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware: Adding auth context to request\n")
			ctx := context.WithValue(r.Context(), AuthContextKeyAuth, authCtx)
			ctx = context.WithValue(ctx, AuthContextKeyUser, user) // Keep backward compatibility
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CharacterResolutionMiddleware expands auth context with all user characters
func (m *EnhancedAuthMiddleware) CharacterResolutionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] CharacterResolutionMiddleware: Processing request %s %s\n", r.Method, r.URL.Path)
			authCtx := GetAuthContext(r.Context())
			if authCtx == nil || !authCtx.IsAuthenticated {
				fmt.Printf("[DEBUG] CharacterResolutionMiddleware: No authenticated user found\n")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			fmt.Printf("[DEBUG] CharacterResolutionMiddleware: Authenticated user found: %s\n", authCtx.UserID)
			
			// Resolve all characters for the user
			fmt.Printf("[DEBUG] CharacterResolutionMiddleware: Resolving characters for user %s\n", authCtx.UserID)
			expandedCtx, err := m.resolveUserCharacters(r.Context(), authCtx)
			if err != nil {
				fmt.Printf("[DEBUG] CharacterResolutionMiddleware: Character resolution failed: %v\n", err)
				slog.Error("Failed to resolve user characters", "error", err, "user_id", authCtx.UserID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			fmt.Printf("[DEBUG] CharacterResolutionMiddleware: Resolved %d characters for user\n", len(expandedCtx.CharacterIDs))
			
			// Add expanded context to request
			fmt.Printf("[DEBUG] CharacterResolutionMiddleware: Adding expanded context to request\n")
			ctx := context.WithValue(r.Context(), AuthContextKeyExpanded, expandedCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalCharacterResolutionMiddleware expands auth context if user is authenticated
func (m *EnhancedAuthMiddleware) OptionalCharacterResolutionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r.Context())
			if authCtx != nil && authCtx.IsAuthenticated {
				// Resolve all characters for the user
				expandedCtx, err := m.resolveUserCharacters(r.Context(), authCtx)
				if err != nil {
					slog.Warn("Failed to resolve user characters (optional)", "error", err, "user_id", authCtx.UserID)
					// Continue without expanded context for optional middleware
				} else {
					// Add expanded context to request
					ctx := context.WithValue(r.Context(), AuthContextKeyExpanded, expandedCtx)
					r = r.WithContext(ctx)
				}
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// resolveUserCharacters resolves all characters for a user and creates expanded context
func (m *EnhancedAuthMiddleware) resolveUserCharacters(ctx context.Context, authCtx *AuthContext) (*ExpandedAuthContext, error) {
	// Get user with all characters
	user, err := m.characterResolver.GetUserWithCharacters(ctx, authCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user characters: %w", err)
	}
	
	var characterIDs []int64
	var corporationIDs []int64
	var allianceIDs []int64
	
	// Extract unique IDs
	corpMap := make(map[int64]bool)
	allianceMap := make(map[int64]bool)
	
	for _, char := range user.Characters {
		characterIDs = append(characterIDs, char.CharacterID)
		
		if !corpMap[char.CorporationID] {
			corporationIDs = append(corporationIDs, char.CorporationID)
			corpMap[char.CorporationID] = true
		}
		
		if char.AllianceID > 0 && !allianceMap[char.AllianceID] {
			allianceIDs = append(allianceIDs, char.AllianceID)
			allianceMap[char.AllianceID] = true
		}
	}
	
	// Find primary character details
	var primaryChar UserCharacter
	for _, char := range user.Characters {
		if char.CharacterID == authCtx.PrimaryCharID {
			primaryChar = char
			break
		}
	}
	
	return &ExpandedAuthContext{
		AuthContext:    authCtx,
		CharacterIDs:   characterIDs,
		CorporationIDs: corporationIDs,
		AllianceIDs:    allianceIDs,
		PrimaryCharacter: struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			CorporationID int64  `json:"corporation_id"`
			AllianceID    int64  `json:"alliance_id,omitempty"`
		}{
			ID:            primaryChar.CharacterID,
			Name:          primaryChar.Name,
			CorporationID: primaryChar.CorporationID,
			AllianceID:    primaryChar.AllianceID,
		},
		Roles:       []string{}, // To be populated by CASBIN
		Permissions: []string{}, // To be populated by CASBIN
	}, nil
}

// RequireExpandedAuth middleware that requires both authentication and character resolution
func (m *EnhancedAuthMiddleware) RequireExpandedAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		authMiddleware := m.AuthenticationMiddleware()
		charMiddleware := m.CharacterResolutionMiddleware()
		
		return authMiddleware(charMiddleware(next))
	}
}

// OptionalExpandedAuth middleware that provides expanded auth if available
func (m *EnhancedAuthMiddleware) OptionalExpandedAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware.OptionalExpandedAuth: Processing request %s %s\n", r.Method, r.URL.Path)
			// Try authentication first
			var token string
			var requestType string
			
			// Check for Bearer token first
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
					requestType = "bearer"
				}
			}
			
			// Fallback to cookie
			if token == "" {
				if cookie, err := r.Cookie("falcon_auth_token"); err == nil {
					token = cookie.Value
					requestType = "cookie"
				}
			}
			
			// If we have a token, try to authenticate
			if token != "" {
				user, err := m.jwtValidator.ValidateJWT(token)
				if err == nil {
					// Create auth context
					authCtx := &AuthContext{
						UserID:          user.UserID,
						PrimaryCharID:   int64(user.CharacterID),
						RequestType:     requestType,
						IsAuthenticated: true,
					}
					
					ctx := context.WithValue(r.Context(), AuthContextKeyAuth, authCtx)
					ctx = context.WithValue(ctx, AuthContextKeyUser, user)
					
					// Try to resolve characters
					expandedCtx, err := m.resolveUserCharacters(ctx, authCtx)
					if err == nil {
						ctx = context.WithValue(ctx, AuthContextKeyExpanded, expandedCtx)
						fmt.Printf("[DEBUG] EnhancedAuthMiddleware.OptionalExpandedAuth: Character resolution successful\n")
					} else {
						fmt.Printf("[DEBUG] EnhancedAuthMiddleware.OptionalExpandedAuth: Character resolution failed: %v\n", err)
					}
					
					r = r.WithContext(ctx)
				}
			}
			
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware.OptionalExpandedAuth: Calling next handler\n")
			next.ServeHTTP(w, r)
			fmt.Printf("[DEBUG] EnhancedAuthMiddleware.OptionalExpandedAuth: Request completed\n")
		})
	}
}