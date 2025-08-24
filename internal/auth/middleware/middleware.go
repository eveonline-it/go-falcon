package middleware

// Middleware aggregates all auth middleware components
type Middleware struct {
	Auth       *AuthMiddleware
	Validation *ValidationMiddleware
}

// New creates a new middleware aggregator with all auth middleware
func New(jwtValidator JWTValidator) *Middleware {
	return &Middleware{
		Auth:       NewAuthMiddleware(jwtValidator),
		Validation: NewValidationMiddleware(),
	}
}
