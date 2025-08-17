package middleware

import (
	"context"
	"net/http"

	"go-falcon/internal/groups/models"
	"go-falcon/internal/groups/services"
	"go-falcon/pkg/handlers"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// PermissionMiddleware provides granular permission checking middleware
type PermissionMiddleware struct {
	granularService *services.GranularPermissionService
	legacyService   *services.PermissionService
}

// NewPermissionMiddleware creates a new permission middleware
func NewPermissionMiddleware(granularService *services.GranularPermissionService, legacyService *services.PermissionService) *PermissionMiddleware {
	return &PermissionMiddleware{
		granularService: granularService,
		legacyService:   legacyService,
	}
}

// RequireGranularPermission creates middleware that requires specific granular permissions
func (m *PermissionMiddleware) RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, r := handlers.StartHTTPSpan(r, "groups.permission_check",
				attribute.String("service", "groups"),
				attribute.String("operation", "require_granular_permission"),
				attribute.String("permission.service", service),
				attribute.String("permission.resource", resource),
				attribute.String("permission.action", action),
			)
			defer span.End()

			result, err := m.granularService.CheckPermissionFromRequest(r, service, resource, action)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Permission check failed")
				handlers.ErrorResponse(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				span.SetAttributes(
					attribute.Bool("permission.allowed", false),
					attribute.String("permission.reason", result.Reason),
				)
				span.SetStatus(codes.Error, "Permission denied")
				handlers.ForbiddenResponse(w, result.Reason)
				return
			}

			span.SetAttributes(
				attribute.Bool("permission.allowed", true),
				attribute.StringSlice("permission.groups", result.Groups),
			)

			// Add permission result to context for handlers that need it
			ctx := context.WithValue(r.Context(), "permission_result", result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalGranularPermission creates middleware that adds permission information to context without blocking
func (m *PermissionMiddleware) OptionalGranularPermission(service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, r := handlers.StartHTTPSpan(r, "groups.optional_permission_check",
				attribute.String("service", "groups"),
				attribute.String("operation", "optional_granular_permission"),
				attribute.String("permission.service", service),
				attribute.String("permission.resource", resource),
				attribute.String("permission.action", action),
			)
			defer span.End()

			result, err := m.granularService.CheckPermissionFromRequest(r, service, resource, action)
			if err != nil {
				span.RecordError(err)
				// For optional checks, we don't fail the request on error
				result = &models.PermissionResult{
					Allowed: false,
					Reason:  "Permission check error",
				}
			}

			span.SetAttributes(
				attribute.Bool("permission.allowed", result.Allowed),
				attribute.String("permission.reason", result.Reason),
			)

			if result.Groups != nil {
				span.SetAttributes(attribute.StringSlice("permission.groups", result.Groups))
			}

			// Add permission result to context
			ctx := context.WithValue(r.Context(), "permission_result", result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireLegacyPermission creates middleware that requires legacy permissions (deprecated)
func (m *PermissionMiddleware) RequireLegacyPermission(resource string, actions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, r := handlers.StartHTTPSpan(r, "groups.legacy_permission_check",
				attribute.String("service", "groups"),
				attribute.String("operation", "require_legacy_permission"),
				attribute.String("permission.resource", resource),
				attribute.StringSlice("permission.actions", actions),
			)
			defer span.End()

			result, err := m.legacyService.CheckPermissionFromRequest(r, resource, actions...)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Legacy permission check failed")
				handlers.ErrorResponse(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				span.SetAttributes(
					attribute.Bool("permission.allowed", false),
					attribute.String("permission.reason", result.Reason),
				)
				span.SetStatus(codes.Error, "Permission denied")
				handlers.ForbiddenResponse(w, result.Reason)
				return
			}

			span.SetAttributes(
				attribute.Bool("permission.allowed", true),
				attribute.StringSlice("permission.groups", result.Groups),
			)

			// Add permission result to context
			ctx := context.WithValue(r.Context(), "permission_result", result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSuperAdmin creates middleware that requires super admin privileges
func (m *PermissionMiddleware) RequireSuperAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span, r := handlers.StartHTTPSpan(r, "groups.super_admin_check",
				attribute.String("service", "groups"),
				attribute.String("operation", "require_super_admin"),
			)
			defer span.End()

			// Check if user is super admin
			isSuperAdmin, err := m.granularService.IsSuperAdmin(r)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "Super admin check failed")
				handlers.ErrorResponse(w, "Authentication check failed", http.StatusInternalServerError)
				return
			}

			if !isSuperAdmin {
				span.SetAttributes(attribute.Bool("is_super_admin", false))
				span.SetStatus(codes.Error, "Super admin required")
				handlers.ForbiddenResponse(w, "Super admin privileges required")
				return
			}

			span.SetAttributes(attribute.Bool("is_super_admin", true))
			next.ServeHTTP(w, r)
		})
	}
}

// GetPermissionResult retrieves the permission result from the request context
func GetPermissionResult(r *http.Request) *models.PermissionResult {
	if result, ok := r.Context().Value("permission_result").(*models.PermissionResult); ok {
		return result
	}
	return nil
}