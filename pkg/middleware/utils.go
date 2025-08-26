package middleware

import (
	"context"
	"fmt"

	"go-falcon/internal/auth/models"
	authServices "go-falcon/internal/auth/services"

	"github.com/danielgtaylor/huma/v2"
)

// PermissionMiddlewareFactory creates permission middleware with standard configuration
type PermissionMiddlewareFactory struct{}

// NewPermissionMiddlewareFactory creates a new factory
func NewPermissionMiddlewareFactory() *PermissionMiddlewareFactory {
	return &PermissionMiddlewareFactory{}
}

// CreateStandard creates a standard permission middleware with common settings
func (f *PermissionMiddlewareFactory) CreateStandard(
	authService *authServices.AuthService,
	permissionChecker PermissionChecker,
) *PermissionMiddleware {
	return NewPermissionMiddleware(
		authService,
		permissionChecker,
		WithDebugLogging(), // Enable debug logging by default during migration
	)
}

// CreateProduction creates a production permission middleware with optimized settings
func (f *PermissionMiddlewareFactory) CreateProduction(
	authService *authServices.AuthService,
	permissionChecker PermissionChecker,
) *PermissionMiddleware {
	return NewPermissionMiddleware(
		authService,
		permissionChecker,
		WithCircuitBreaker(), // Enable circuit breaker for production
		// Debug logging disabled for production
	)
}

// CreateDevelopment creates a development permission middleware with extensive logging
func (f *PermissionMiddlewareFactory) CreateDevelopment(
	authService *authServices.AuthService,
	permissionChecker PermissionChecker,
) *PermissionMiddleware {
	return NewPermissionMiddleware(
		authService,
		permissionChecker,
		WithDebugLogging(),
		// No circuit breaker for development
	)
}

// PermissionRequirement defines a permission requirement for validation
type PermissionRequirement struct {
	PermissionID string
	Description  string
	Required     bool // If false, permission is optional
}

// ValidationResult contains the result of permission validation
type ValidationResult struct {
	Valid          bool
	MissingPerms   []string
	AvailablePerms []string
	User           *models.AuthenticatedUser
	Errors         []error
}

// PermissionValidator helps validate permission requirements during migration
type PermissionValidator struct {
	permissionMiddleware *PermissionMiddleware
}

// NewPermissionValidator creates a new permission validator
func NewPermissionValidator(permissionMiddleware *PermissionMiddleware) *PermissionValidator {
	return &PermissionValidator{
		permissionMiddleware: permissionMiddleware,
	}
}

// ValidateUserPermissions validates if a user has the required permissions
func (pv *PermissionValidator) ValidateUserPermissions(
	ctx context.Context,
	authHeader, cookieHeader string,
	requirements []PermissionRequirement,
) *ValidationResult {
	result := &ValidationResult{
		Valid:          true,
		MissingPerms:   []string{},
		AvailablePerms: []string{},
		Errors:         []error{},
	}

	// First authenticate the user
	user, err := pv.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err)
		return result
	}
	result.User = user

	// Check each permission requirement
	for _, req := range requirements {
		hasPermission, err := pv.permissionMiddleware.permissionChecker.HasPermission(
			ctx,
			int64(user.CharacterID),
			req.PermissionID,
		)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error checking %s: %w", req.PermissionID, err))
			if req.Required {
				result.Valid = false
			}
			continue
		}

		if hasPermission {
			result.AvailablePerms = append(result.AvailablePerms, req.PermissionID)
		} else {
			result.MissingPerms = append(result.MissingPerms, req.PermissionID)
			if req.Required {
				result.Valid = false
			}
		}
	}

	return result
}

// LegacyCompatibility provides compatibility methods for existing module patterns
type LegacyCompatibility struct {
	permissionMiddleware *PermissionMiddleware
}

// NewLegacyCompatibility creates a new legacy compatibility helper
func NewLegacyCompatibility(permissionMiddleware *PermissionMiddleware) *LegacyCompatibility {
	return &LegacyCompatibility{
		permissionMiddleware: permissionMiddleware,
	}
}

// RequireAuth provides legacy RequireAuth method signature
func (lc *LegacyCompatibility) RequireAuth(ctx context.Context, authHeader, cookieHeader string) (*models.AuthenticatedUser, error) {
	return lc.permissionMiddleware.RequireAuth(ctx, authHeader, cookieHeader)
}

// RequirePermission provides legacy RequirePermission method signature
func (lc *LegacyCompatibility) RequirePermission(ctx context.Context, authHeader, cookieHeader, permissionID string) (*models.AuthenticatedUser, error) {
	return lc.permissionMiddleware.RequirePermission(ctx, authHeader, cookieHeader, permissionID)
}

// RequireAnyPermission provides legacy RequireAnyPermission method signature
func (lc *LegacyCompatibility) RequireAnyPermission(ctx context.Context, authHeader, cookieHeader string, permissionIDs []string) (*models.AuthenticatedUser, error) {
	return lc.permissionMiddleware.RequireAnyPermission(ctx, authHeader, cookieHeader, permissionIDs)
}

// ModuleMigrationGuide provides step-by-step migration guidance
type ModuleMigrationGuide struct {
	ModuleName string
	Steps      []MigrationStep
}

// MigrationStep represents a single step in module migration
type MigrationStep struct {
	Description string
	Before      string
	After       string
	Completed   bool
}

// GetSitemapMigrationGuide returns migration guide for sitemap module
func GetSitemapMigrationGuide() *ModuleMigrationGuide {
	return &ModuleMigrationGuide{
		ModuleName: "sitemap",
		Steps: []MigrationStep{
			{
				Description: "Replace module middleware initialization",
				Before:      `middleware.NewAuthMiddleware(authService, permissionManager)`,
				After:       `middleware.NewPermissionMiddleware(authService, permissionManager, middleware.WithDebugLogging())`,
			},
			{
				Description: "Replace RequireSitemapView calls",
				Before:      `r.middleware.RequireSitemapView(ctx, input.Authorization, input.Cookie)`,
				After:       `r.permissionMiddleware.RequirePermission(ctx, input.Authorization, input.Cookie, "sitemap:routes:view")`,
			},
			{
				Description: "Replace RequireSitemapAdmin calls",
				Before:      `r.middleware.RequireSitemapAdmin(ctx, input.Authorization, input.Cookie)`,
				After:       `r.permissionMiddleware.RequirePermission(ctx, input.Authorization, input.Cookie, "sitemap:admin:manage")`,
			},
			{
				Description: "Replace RequireSitemapNavigation calls",
				Before:      `r.middleware.RequireSitemapNavigation(ctx, input.Authorization, input.Cookie)`,
				After:       `r.permissionMiddleware.RequirePermission(ctx, input.Authorization, input.Cookie, "sitemap:navigation:customize")`,
			},
			{
				Description: "Remove module-specific middleware file",
				Before:      `internal/sitemap/middleware/auth.go`,
				After:       `DELETE FILE`,
			},
		},
	}
}

// GetSchedulerMigrationGuide returns migration guide for scheduler module
func GetSchedulerMigrationGuide() *ModuleMigrationGuide {
	return &ModuleMigrationGuide{
		ModuleName: "scheduler",
		Steps: []MigrationStep{
			{
				Description: "Replace module middleware initialization",
				Before:      `middleware.NewAuthMiddleware(authService, permissionManager)`,
				After:       `middleware.NewSchedulerAdapter(middleware.NewPermissionMiddleware(authService, permissionManager))`,
			},
			{
				Description: "Replace RequireSchedulerManagement calls",
				Before:      `m.authMiddleware.RequireSchedulerManagement(ctx, authHeader, cookieHeader)`,
				After:       `m.schedulerAdapter.RequireSchedulerManagement(ctx, authHeader, cookieHeader)`,
			},
			{
				Description: "Replace RequireTaskManagement calls",
				Before:      `m.authMiddleware.RequireTaskManagement(ctx, authHeader, cookieHeader)`,
				After:       `m.schedulerAdapter.RequireTaskManagement(ctx, authHeader, cookieHeader)`,
			},
		},
	}
}

// HumaIntegrationHelper provides utilities for Huma v2 integration
type HumaIntegrationHelper struct {
	permissionMiddleware *PermissionMiddleware
}

// NewHumaIntegrationHelper creates a new Huma integration helper
func NewHumaIntegrationHelper(permissionMiddleware *PermissionMiddleware) *HumaIntegrationHelper {
	return &HumaIntegrationHelper{
		permissionMiddleware: permissionMiddleware,
	}
}

// CreatePermissionMiddlewareFunc creates a Huma middleware function for permission checking
func (h *HumaIntegrationHelper) CreatePermissionMiddlewareFunc(permissionID string) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		// Extract headers from Huma context
		authHeader := ctx.Header("Authorization")
		cookieHeader := ctx.Header("Cookie")

		// Check permission
		_, err := h.permissionMiddleware.RequirePermission(ctx.Context(), authHeader, cookieHeader, permissionID)
		if err != nil {
			// Handle Huma error - just return, Huma will handle the error response
			// The error is already a properly formatted Huma error from RequirePermission
			return
		}

		// Permission granted, continue
		next(ctx)
	}
}

// CreateAuthMiddlewareFunc creates a Huma middleware function for authentication only
func (h *HumaIntegrationHelper) CreateAuthMiddlewareFunc() func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		// Extract headers from Huma context
		authHeader := ctx.Header("Authorization")
		cookieHeader := ctx.Header("Cookie")

		// Check authentication
		_, err := h.permissionMiddleware.RequireAuth(ctx.Context(), authHeader, cookieHeader)
		if err != nil {
			// Handle Huma error - just return, Huma will handle the error response
			// The error is already a properly formatted Huma error from RequireAuth
			return
		}

		// Authentication successful, continue
		next(ctx)
	}
}
