package middleware

import (
	"context"
	"fmt"
	"net/http"

	"go-falcon/pkg/database"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// ExampleIntegration shows how to integrate the enhanced auth middleware
// This is for demonstration purposes and would typically be in the main application setup

// SetupEnhancedMiddleware creates and configures the enhanced authentication middleware
func SetupEnhancedMiddleware(mongodb *database.MongoDB, jwtValidator JWTValidator) *EnhancedAuthMiddleware {
	// Create the character resolver using our MongoDB implementation
	characterResolver := NewUserCharacterResolver(mongodb)
	
	// Create the enhanced middleware with the auth service as JWT validator
	enhancedAuth := NewEnhancedAuthMiddleware(jwtValidator, characterResolver)
	
	return enhancedAuth
}

// ExampleRouteSetup shows how to apply the middleware to different route patterns
func ExampleRouteSetup(r chi.Router, enhancedAuth *EnhancedAuthMiddleware) {
	// Example 1: Routes that require authentication but not character resolution
	r.Route("/api/basic", func(r chi.Router) {
		r.Use(enhancedAuth.AuthenticationMiddleware())
		
		r.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r.Context())
			fmt.Fprintf(w, "Authenticated user: %s", authCtx.UserID)
		})
	})
	
	// Example 2: Routes that require full character resolution (for CASBIN)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(enhancedAuth.RequireExpandedAuth())
		
		r.Get("/permissions", func(w http.ResponseWriter, r *http.Request) {
			expandedCtx := GetExpandedAuthContext(r.Context())
			fmt.Fprintf(w, "User %s has %d characters in %d corporations", 
				expandedCtx.UserID, 
				len(expandedCtx.CharacterIDs), 
				len(expandedCtx.CorporationIDs))
		})
	})
	
	// Example 3: Routes with optional authentication
	r.Route("/api/public", func(r chi.Router) {
		r.Use(enhancedAuth.OptionalExpandedAuth())
		
		r.Get("/data", func(w http.ResponseWriter, r *http.Request) {
			expandedCtx := GetExpandedAuthContext(r.Context())
			if expandedCtx != nil {
				fmt.Fprintf(w, "Authenticated user with %d characters", len(expandedCtx.CharacterIDs))
			} else {
				fmt.Fprint(w, "Anonymous user")
			}
		})
	})
}

// ExampleHumaIntegration shows how to use the middleware with Huma v2
func ExampleHumaIntegration(api huma.API, enhancedAuth *EnhancedAuthMiddleware) {
	// Example input/output for Huma endpoints
	type AuthenticatedInput struct {
		// Huma will automatically extract auth headers
	}
	
	type UserInfoOutput struct {
		Body struct {
			UserID       string   `json:"user_id"`
			CharacterIDs []int64  `json:"character_ids"`
			Permissions  []string `json:"permissions"`
		}
	}
	
	// Register endpoint with enhanced auth middleware
	// Note: This is a simplified example - actual Huma integration would require
	// proper middleware adaptation for Huma's context system
	huma.Get(api, "/users/me", func(ctx context.Context, input *AuthenticatedInput) (*UserInfoOutput, error) {
		// Extract expanded auth context from the HTTP request context
		// Note: In real implementation, you'd need to extract this from the HTTP context
		// This is simplified for example purposes
		
		expandedCtx := GetExpandedAuthContext(ctx)
		if expandedCtx == nil {
			return nil, huma.Error401Unauthorized("Authentication required")
		}
		
		resp := &UserInfoOutput{}
		resp.Body.UserID = expandedCtx.UserID
		resp.Body.CharacterIDs = expandedCtx.CharacterIDs
		resp.Body.Permissions = expandedCtx.Permissions
		
		return resp, nil
	})
}

// ExampleCASBINPreparation shows how to prepare subject lists for CASBIN
func ExampleCASBINPreparation(expandedCtx *ExpandedAuthContext) []string {
	var subjects []string
	
	// Add user subject
	subjects = append(subjects, fmt.Sprintf("user:%s", expandedCtx.UserID))
	
	// Add primary character subject
	subjects = append(subjects, fmt.Sprintf("character:%d", expandedCtx.PrimaryCharacter.ID))
	
	// Add all character subjects
	for _, charID := range expandedCtx.CharacterIDs {
		subjects = append(subjects, fmt.Sprintf("character:%d", charID))
	}
	
	// Add corporation subjects
	for _, corpID := range expandedCtx.CorporationIDs {
		subjects = append(subjects, fmt.Sprintf("corporation:%d", corpID))
	}
	
	// Add alliance subjects
	for _, allianceID := range expandedCtx.AllianceIDs {
		subjects = append(subjects, fmt.Sprintf("alliance:%d", allianceID))
	}
	
	return subjects
}

// ExampleMiddlewareChain shows a complete middleware chain for maximum security
func ExampleMiddlewareChain(enhancedAuth *EnhancedAuthMiddleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Authentication
			authMiddleware := enhancedAuth.AuthenticationMiddleware()
			
			// 2. Character resolution
			charMiddleware := enhancedAuth.CharacterResolutionMiddleware()
			
			// 3. Apply chain
			handler := authMiddleware(charMiddleware(next))
			handler.ServeHTTP(w, r)
		})
	}
}