package middleware

import (
	"fmt"
	"go-falcon/pkg/database"
)

// MiddlewareConfig holds configuration for middleware initialization
type MiddlewareConfig struct {
	EnableDebug   bool
	EnableCaching bool
	CacheTTL      int // seconds
}

// DefaultConfig returns default middleware configuration
func DefaultConfig() *MiddlewareConfig {
	return &MiddlewareConfig{
		EnableDebug:   true,  // Enable debug in development
		EnableCaching: false, // Disable caching for now
		CacheTTL:      3600,  // 1 hour
	}
}

// InitializeMiddleware initializes all middleware components with the given configuration
func InitializeMiddleware(jwtValidator JWTValidator, mongodb *database.MongoDB, config *MiddlewareConfig) (*MiddlewareFactory, error) {
	fmt.Printf("[DEBUG] InitializeMiddleware: Setting up middleware with config: debug=%v, caching=%v\n", 
		config.EnableDebug, config.EnableCaching)
	
	if jwtValidator == nil {
		return nil, fmt.Errorf("jwtValidator is required")
	}
	
	if mongodb == nil {
		return nil, fmt.Errorf("mongodb is required")
	}
	
	// Create the factory with all dependencies
	factory := NewMiddlewareFactory(jwtValidator, mongodb)
	
	fmt.Printf("[DEBUG] InitializeMiddleware: Middleware factory created successfully\n")
	
	return factory, nil
}

// QuickSetup provides a simple way to initialize middleware for common use cases
func QuickSetup(jwtValidator JWTValidator, mongodb *database.MongoDB) (*MiddlewareFactory, error) {
	return InitializeMiddleware(jwtValidator, mongodb, DefaultConfig())
}

// ProductionSetup provides middleware setup optimized for production
func ProductionSetup(jwtValidator JWTValidator, mongodb *database.MongoDB) (*MiddlewareFactory, error) {
	config := &MiddlewareConfig{
		EnableDebug:   false, // Disable debug in production
		EnableCaching: true,  // Enable caching in production
		CacheTTL:      1800,  // 30 minutes
	}
	
	return InitializeMiddleware(jwtValidator, mongodb, config)
}

// Summary provides a summary of what middleware components are available
func Summary() {
	fmt.Printf(`
[INFO] Go Falcon Middleware Package Summary:

Core Components:
- AuthMiddleware: Basic JWT validation from headers/cookies
- EnhancedAuthMiddleware: Authentication + character resolution
- ConvenienceMiddleware: Easy-to-use wrapper functions
- ContextHelper: Utilities for working with auth context
- MiddlewareFactory: Pre-configured middleware stacks

Integration:
- HumaIntegration: Helpers for Huma v2 framework
- HTTP Middleware: Standard net/http middleware support
- Debug Helpers: Development debugging tools

Available Middleware Stacks:
- PublicWithOptionalAuth(): Public endpoints with optional authentication
- RequireBasicAuth(): Basic authentication required
- RequireAuthWithCharacters(): Authentication + character resolution
- RequireScope(scopes...): EVE Online scope validation
- RequirePermission(resource, action): Permission-based access (Phase 3)
- AdminOnly(): Admin-only access
- CorporationAccess(resource): Corporation-level access
- AllianceAccess(resource): Alliance-level access

Context Helpers:
- GetUserID(r): Extract user ID from request
- GetPrimaryCharacterID(r): Extract primary character ID
- GetAllCharacterIDs(r): Extract all character IDs
- GetAllCorporationIDs(r): Extract corporation IDs
- GetAllAllianceIDs(r): Extract alliance IDs
- IsAuthenticated(r): Check authentication status
- GetAuthInfo(r): Get comprehensive auth information

Usage Example:
  factory, err := middleware.QuickSetup(authService, mongodb)
  if err != nil {
      log.Fatal(err)
  }
  
  // Use in HTTP handlers
  r.Use(factory.RequireBasicAuth())
  
  // Use in Huma handlers
  authInfo, err := factory.GetAuthMiddleware().ValidateAuthFromHeaders(auth, cookie)

`)
}