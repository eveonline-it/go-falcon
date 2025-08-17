package routes

import (
	"go-falcon/internal/auth/middleware"
	"go-falcon/internal/auth/services"

	"github.com/go-chi/chi/v5"
)

// Routes handles all auth route definitions
type Routes struct {
	authService *services.AuthService
	middleware  *middleware.Middleware
}

// New creates a new routes handler
func New(authService *services.AuthService, middleware *middleware.Middleware) *Routes {
	return &Routes{
		authService: authService,
		middleware:  middleware,
	}
}

// RegisterRoutes registers all auth routes
func (r *Routes) RegisterRoutes(router chi.Router) {
	// Health check endpoint
	router.Get("/health", r.authService.HealthCheck)

	// Basic auth endpoints (legacy - not currently implemented)
	// router.Post("/login", r.basicLoginHandler)
	// router.Post("/register", r.basicRegisterHandler)
	
	// EVE Online SSO routes (public)
	router.Group(func(router chi.Router) {
		router.Get("/eve/login", r.eveBasicLoginHandler)     // Basic login without scopes
		router.Get("/eve/register", r.eveFullLoginHandler)   // Full registration with scopes
		router.Get("/eve/callback", r.eveCallbackHandler)    // OAuth callback
		router.Post("/eve/refresh", r.eveRefreshHandler)     // Token refresh
		router.Get("/eve/verify", r.eveVerifyHandler)        // Token verification
		router.Post("/eve/token", r.eveTokenExchangeHandler) // Mobile token exchange
	})
	
	// Authentication status and user info (public but may include user data)
	router.Group(func(router chi.Router) {
		router.Use(r.middleware.Auth.OptionalAuth) // Optional auth for these endpoints
		router.Get("/status", r.authStatusHandler)
		router.Get("/user", r.getCurrentUserHandler)
	})
	
	// Profile routes (require authentication)
	router.Group(func(router chi.Router) {
		router.Use(r.middleware.Auth.RequireAuth)
		
		router.Get("/profile", r.profileHandler)
		router.Post("/profile/refresh", r.refreshProfileHandler)
		router.Get("/token", r.tokenHandler) // Get bearer token
	})
	
	// Public profile endpoint (no auth required)
	router.Get("/profile/public", r.publicProfileHandler)
	
	// Logout endpoint (public)
	router.Post("/logout", r.logoutHandler)
}