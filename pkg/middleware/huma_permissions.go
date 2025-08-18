package middleware

import (
	"context"

	"go-falcon/internal/auth/models"

	"github.com/danielgtaylor/huma/v2"
)

// PermissionChecker interface for checking permissions
type PermissionChecker interface {
	CheckPermission(ctx context.Context, characterID int, service, resource, action string) (bool, error)
}

// HumaPermissionMiddleware provides permission validation utilities for Huma operations
type HumaPermissionMiddleware struct {
	permissionChecker PermissionChecker
}

// NewHumaPermissionMiddleware creates a new Huma permission middleware
func NewHumaPermissionMiddleware(checker PermissionChecker) *HumaPermissionMiddleware {
	return &HumaPermissionMiddleware{
		permissionChecker: checker,
	}
}

// ValidatePermission validates that authenticated user has required permission
func (m *HumaPermissionMiddleware) ValidatePermission(ctx context.Context, user *models.AuthenticatedUser, service, resource, action string) error {
	if user == nil {
		return huma.Error401Unauthorized("Authentication required for permission check")
	}

	// Check if user has required permission
	hasPermission, err := m.permissionChecker.CheckPermission(ctx, user.CharacterID, service, resource, action)
	if err != nil {
		return huma.Error500InternalServerError("Failed to check permissions", err)
	}

	if !hasPermission {
		return huma.Error403Forbidden("Insufficient permissions for " + service + "." + resource + "." + action)
	}

	return nil
}

// ValidateAuthAndPermission validates both authentication and permission in one call
func (m *HumaPermissionMiddleware) ValidateAuthAndPermission(
	humaAuth *HumaAuthMiddleware,
	authHeader, cookieHeader string,
	service, resource, action string,
) (*models.AuthenticatedUser, error) {
	// First validate authentication
	user, err := humaAuth.ValidateAuthFromHeaders(authHeader, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Then validate permission
	err = m.ValidatePermission(context.Background(), user, service, resource, action)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ValidateOptionalAuthAndPermission validates optional authentication and permission
func (m *HumaPermissionMiddleware) ValidateOptionalAuthAndPermission(
	humaAuth *HumaAuthMiddleware,
	authHeader, cookieHeader string,
	service, resource, action string,
) *models.AuthenticatedUser {
	// First validate authentication (optional)
	user := humaAuth.ValidateOptionalAuthFromHeaders(authHeader, cookieHeader)
	if user == nil {
		return nil // No auth, no permission needed
	}

	// If authenticated, validate permission
	err := m.ValidatePermission(context.Background(), user, service, resource, action)
	if err != nil {
		return nil // Permission denied, return nil for optional auth
	}

	return user
}