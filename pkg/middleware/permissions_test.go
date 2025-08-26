package middleware

import (
	"context"
	"errors"
	"testing"

	"go-falcon/internal/auth/models"
	"go-falcon/pkg/permissions"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJWTValidator is a mock JWT validator for testing
type MockJWTValidator struct {
	mock.Mock
}

func (m *MockJWTValidator) ValidateJWT(token string) (*models.AuthenticatedUser, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthenticatedUser), args.Error(1)
}

// MockPermissionManager is a mock permission manager for testing
type MockPermissionManager struct {
	mock.Mock
}

func (m *MockPermissionManager) HasPermission(ctx context.Context, characterID int64, permissionID string) (bool, error) {
	args := m.Called(ctx, characterID, permissionID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPermissionManager) CheckPermission(ctx context.Context, characterID int64, permissionID string) (*permissions.PermissionCheck, error) {
	args := m.Called(ctx, characterID, permissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*permissions.PermissionCheck), args.Error(1)
}

func TestNewPermissionMiddleware(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}

	// Test with default options
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.authMiddleware)
	assert.Equal(t, mockPermissionManager, pm.permissionChecker)
	assert.False(t, pm.options.EnableDebugLogging)
	assert.False(t, pm.options.EnableCircuitBreaker)
	assert.True(t, pm.options.FallbackToAuth)

	// Test with options
	pmWithOptions := NewPermissionMiddleware(
		mockValidator,
		mockPermissionManager,
		WithDebugLogging(),
		WithCircuitBreaker(),
		WithoutFallback(),
	)
	assert.True(t, pmWithOptions.options.EnableDebugLogging)
	assert.True(t, pmWithOptions.options.EnableCircuitBreaker)
	assert.False(t, pmWithOptions.options.FallbackToAuth)
}

func TestPermissionMiddleware_RequireAuth(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()

	t.Run("successful authentication", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)

		user, err := pm.RequireAuth(ctx, "Bearer valid-token", "")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil // Reset for next test
	})

	t.Run("invalid token", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "invalid-token").Return(nil, errors.New("invalid token"))

		user, err := pm.RequireAuth(ctx, "Bearer invalid-token", "")
		assert.Error(t, err)
		assert.Nil(t, user)

		// Check that it's a Huma error
		var humaErr *huma.ErrorModel
		assert.True(t, errors.As(err, &humaErr))
		assert.Equal(t, 401, humaErr.Status)

		mockValidator.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
	})

	t.Run("no token provided", func(t *testing.T) {
		user, err := pm.RequireAuth(ctx, "", "")
		assert.Error(t, err)
		assert.Nil(t, user)

		// Check that it's a Huma error
		var humaErr *huma.ErrorModel
		assert.True(t, errors.As(err, &humaErr))
		assert.Equal(t, 401, humaErr.Status)
	})

	t.Run("token from cookie", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "cookie-token").Return(testUser, nil)

		user, err := pm.RequireAuth(ctx, "", "falcon_auth_token=cookie-token; Path=/")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
	})
}

func TestPermissionMiddleware_RequirePermission(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()
	permissionID := "test:permission:read"

	t.Run("successful permission check", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("CheckPermission", ctx, int64(12345), permissionID).Return(
			&permissions.PermissionCheck{
				CharacterID:  12345,
				PermissionID: permissionID,
				Granted:      true,
				GrantedVia:   "Test Group",
			}, nil)

		user, err := pm.RequirePermission(ctx, "Bearer valid-token", "", permissionID)
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("permission denied", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("CheckPermission", ctx, int64(12345), permissionID).Return(
			&permissions.PermissionCheck{
				CharacterID:  12345,
				PermissionID: permissionID,
				Granted:      false,
			}, nil)

		user, err := pm.RequirePermission(ctx, "Bearer valid-token", "", permissionID)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Check that it's a Huma 403 error
		var humaErr *huma.ErrorModel
		assert.True(t, errors.As(err, &humaErr))
		assert.Equal(t, 403, humaErr.Status)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("permission check error", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("CheckPermission", ctx, int64(12345), permissionID).Return(
			nil, errors.New("database error"))

		user, err := pm.RequirePermission(ctx, "Bearer valid-token", "", permissionID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "permission check failed")

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("authentication failure", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "invalid-token").Return(nil, errors.New("invalid token"))

		user, err := pm.RequirePermission(ctx, "Bearer invalid-token", "", permissionID)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Should fail at authentication, not reach permission check
		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertNotCalled(t, "CheckPermission")
		mockValidator.ExpectedCalls = nil
	})
}

func TestPermissionMiddleware_RequireAnyPermission(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()
	permissionIDs := []string{"test:permission:read", "test:permission:write"}

	t.Run("has first permission", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:read").Return(true, nil)

		user, err := pm.RequireAnyPermission(ctx, "Bearer valid-token", "", permissionIDs)
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		// Should not check second permission if first is granted
		mockPermissionManager.AssertNotCalled(t, "HasPermission", ctx, int64(12345), "test:permission:write")
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("has second permission only", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:read").Return(false, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:write").Return(true, nil)

		user, err := pm.RequireAnyPermission(ctx, "Bearer valid-token", "", permissionIDs)
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("has no permissions", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:read").Return(false, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:write").Return(false, nil)

		user, err := pm.RequireAnyPermission(ctx, "Bearer valid-token", "", permissionIDs)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Check that it's a Huma 403 error
		var humaErr *huma.ErrorModel
		assert.True(t, errors.As(err, &humaErr))
		assert.Equal(t, 403, humaErr.Status)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})
}

func TestPermissionMiddleware_RequireAllPermissions(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()
	permissionIDs := []string{"test:permission:read", "test:permission:write"}

	t.Run("has all permissions", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:read").Return(true, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:write").Return(true, nil)

		user, err := pm.RequireAllPermissions(ctx, "Bearer valid-token", "", permissionIDs)
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("missing one permission", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:read").Return(true, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "test:permission:write").Return(false, nil)

		user, err := pm.RequireAllPermissions(ctx, "Bearer valid-token", "", permissionIDs)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Check that it's a Huma 403 error
		var humaErr *huma.ErrorModel
		assert.True(t, errors.As(err, &humaErr))
		assert.Equal(t, 403, humaErr.Status)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})
}

func TestPermissionMiddleware_FallbackBehavior(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	pm := NewPermissionMiddleware(mockValidator, nil) // No permission manager

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()

	t.Run("fallback enabled - returns user", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)

		user, err := pm.RequirePermission(ctx, "Bearer valid-token", "", "test:permission")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
	})

	t.Run("fallback disabled - returns error", func(t *testing.T) {
		pmNoFallback := NewPermissionMiddleware(mockValidator, nil, WithoutFallback())
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)

		user, err := pmNoFallback.RequirePermission(ctx, "Bearer valid-token", "", "test:permission")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "Permission system not available")

		mockValidator.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
	})
}

func TestSitemapAdapter(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)
	adapter := NewSitemapAdapter(pm)

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()

	t.Run("RequireSitemapView", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("CheckPermission", ctx, int64(12345), "sitemap:routes:view").Return(
			&permissions.PermissionCheck{
				CharacterID:  12345,
				PermissionID: "sitemap:routes:view",
				Granted:      true,
				GrantedVia:   "Test Group",
			}, nil)

		user, err := adapter.RequireSitemapView(ctx, "Bearer valid-token", "")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("RequireSitemapAdmin", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("CheckPermission", ctx, int64(12345), "sitemap:admin:manage").Return(
			&permissions.PermissionCheck{
				CharacterID:  12345,
				PermissionID: "sitemap:admin:manage",
				Granted:      true,
				GrantedVia:   "Test Group",
			}, nil)

		user, err := adapter.RequireSitemapAdmin(ctx, "Bearer valid-token", "")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("RequireSitemapNavigation", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("CheckPermission", ctx, int64(12345), "sitemap:navigation:customize").Return(
			&permissions.PermissionCheck{
				CharacterID:  12345,
				PermissionID: "sitemap:navigation:customize",
				Granted:      true,
				GrantedVia:   "Test Group",
			}, nil)

		user, err := adapter.RequireSitemapNavigation(ctx, "Bearer valid-token", "")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})
}

func TestSchedulerAdapter(t *testing.T) {
	mockValidator := &MockJWTValidator{}
	mockPermissionManager := &MockPermissionManager{}
	pm := NewPermissionMiddleware(mockValidator, mockPermissionManager)
	adapter := NewSchedulerAdapter(pm)

	testUser := &models.AuthenticatedUser{
		CharacterID:   12345,
		CharacterName: "Test Character",
		UserID:        "test-user-id",
	}

	ctx := context.Background()

	t.Run("RequireSchedulerManagement - has first permission", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "scheduler:tasks:read").Return(true, nil)

		user, err := adapter.RequireSchedulerManagement(ctx, "Bearer valid-token", "")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})

	t.Run("RequireTaskManagement - has management permission", func(t *testing.T) {
		mockValidator.On("ValidateJWT", "valid-token").Return(testUser, nil)
		mockPermissionManager.On("HasPermission", ctx, int64(12345), "scheduler:tasks:create").Return(true, nil)

		user, err := adapter.RequireTaskManagement(ctx, "Bearer valid-token", "")
		assert.NoError(t, err)
		assert.Equal(t, testUser, user)

		mockValidator.AssertExpectations(t)
		mockPermissionManager.AssertExpectations(t)
		mockValidator.ExpectedCalls = nil
		mockPermissionManager.ExpectedCalls = nil
	})
}
