package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-falcon/internal/auth/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJWTValidator is a mock implementation of JWTValidator
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

// MockUserCharacterResolver is a mock implementation of UserCharacterResolver
type MockUserCharacterResolver struct {
	mock.Mock
}

func (m *MockUserCharacterResolver) GetUserWithCharacters(ctx context.Context, userID string) (*UserWithCharacters, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*UserWithCharacters), args.Error(1)
}

// TestData contains common test data
type TestData struct {
	TestUser *models.AuthenticatedUser
	TestUserWithCharacters *UserWithCharacters
}

func setupTestData() *TestData {
	return &TestData{
		TestUser: &models.AuthenticatedUser{
			UserID:        "test-user-123",
			CharacterID:   123456789,
			CharacterName: "Test Character",
			Scopes:        "esi-characters.read_contacts.v1",
		},
		TestUserWithCharacters: &UserWithCharacters{
			ID: "test-user-123",
			Characters: []UserCharacter{
				{
					CharacterID:   123456789,
					Name:          "Test Character",
					CorporationID: 987654321,
					AllianceID:    111222333,
					IsPrimary:     true,
				},
				{
					CharacterID:   987654321,
					Name:          "Alt Character",
					CorporationID: 987654321,
					AllianceID:    111222333,
					IsPrimary:     false,
				},
			},
		},
	}
}

func TestCasbinAuthMiddleware_PolicyManagement(t *testing.T) {
	// This test would require a real MongoDB instance
	// For now, we'll test the structure and interfaces
	
	t.Run("PolicyStructure", func(t *testing.T) {
		// Test that our policy structures are correct
		policy := PermissionPolicy{
			SubjectType: "user",
			SubjectID:   "test-user-123",
			Resource:    "scheduler.tasks",
			Action:      "read",
			Domain:      "global",
			Effect:      "allow",
			CreatedAt:   time.Now(),
			IsActive:    true,
		}

		assert.Equal(t, "user", policy.SubjectType)
		assert.Equal(t, "scheduler.tasks", policy.Resource)
		assert.Equal(t, "read", policy.Action)
		assert.Equal(t, "allow", policy.Effect)
		assert.True(t, policy.IsActive)
	})

	t.Run("RoleAssignmentStructure", func(t *testing.T) {
		roleAssignment := RoleAssignment{
			RoleName:    "admin",
			SubjectType: "user",
			SubjectID:   "test-user-123",
			Domain:      "global",
			GrantedAt:   time.Now(),
			IsActive:    true,
		}

		assert.Equal(t, "admin", roleAssignment.RoleName)
		assert.Equal(t, "user", roleAssignment.SubjectType)
		assert.True(t, roleAssignment.IsActive)
	})
}

func TestCasbinIntegration_SubjectBuilding(t *testing.T) {
	testData := setupTestData()
	
	t.Run("buildSubjects", func(t *testing.T) {
		// Create a mock Casbin auth middleware to test subject building
		expandedCtx := &ExpandedAuthContext{
			AuthContext: &AuthContext{
				UserID:          testData.TestUser.UserID,
				PrimaryCharID:   int64(testData.TestUser.CharacterID),
				IsAuthenticated: true,
			},
			CharacterIDs:   []int64{123456789, 987654321},
			CorporationIDs: []int64{987654321},
			AllianceIDs:    []int64{111222333},
			PrimaryCharacter: struct {
				ID            int64  `json:"id"`
				Name          string `json:"name"`
				CorporationID int64  `json:"corporation_id"`
				AllianceID    int64  `json:"alliance_id,omitempty"`
			}{
				ID:            123456789,
				Name:          "Test Character",
				CorporationID: 987654321,
				AllianceID:    111222333,
			},
		}

		// Mock middleware
		middleware := &CasbinAuthMiddleware{}
		subjects := middleware.buildSubjects(expandedCtx)

		// Verify subjects are built correctly
		expectedSubjects := []string{
			"user:test-user-123",
			"character:123456789",
			"corporation:987654321",
			"alliance:111222333",
		}

		assert.Contains(t, subjects, "user:test-user-123")
		assert.Contains(t, subjects, "character:123456789")
		assert.Contains(t, subjects, "corporation:987654321")
		assert.Contains(t, subjects, "alliance:111222333")
		assert.Len(t, subjects, len(expectedSubjects))
	})
}

func TestEnhancedAuthMiddleware_Integration(t *testing.T) {
	testData := setupTestData()

	t.Run("AuthenticationMiddleware", func(t *testing.T) {
		// Setup mocks
		mockJWTValidator := &MockJWTValidator{}
		mockCharacterResolver := &MockUserCharacterResolver{}

		// Configure mock expectations
		mockJWTValidator.On("ValidateJWT", "valid-token").Return(testData.TestUser, nil)

		// Create enhanced middleware
		enhanced := NewEnhancedAuthMiddleware(mockJWTValidator, mockCharacterResolver)

		// Create test request with Bearer token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		
		rr := httptest.NewRecorder()
		
		// Create a simple test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := GetAuthContext(r.Context())
			assert.NotNil(t, authCtx)
			assert.True(t, authCtx.IsAuthenticated)
			assert.Equal(t, testData.TestUser.UserID, authCtx.UserID)
			assert.Equal(t, "bearer", authCtx.RequestType)
			w.WriteHeader(http.StatusOK)
		})

		// Apply middleware and test
		middleware := enhanced.AuthenticationMiddleware()
		handler := middleware(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockJWTValidator.AssertExpectations(t)
	})

	t.Run("CharacterResolutionMiddleware", func(t *testing.T) {
		// Setup mocks
		mockJWTValidator := &MockJWTValidator{}
		mockCharacterResolver := &MockUserCharacterResolver{}

		// Configure mock expectations
		mockCharacterResolver.On("GetUserWithCharacters", 
			mock.Anything, testData.TestUser.UserID).Return(testData.TestUserWithCharacters, nil)

		// Create enhanced middleware
		enhanced := NewEnhancedAuthMiddleware(mockJWTValidator, mockCharacterResolver)

		// Create test request with pre-existing auth context
		req := httptest.NewRequest("GET", "/test", nil)
		authCtx := &AuthContext{
			UserID:          testData.TestUser.UserID,
			PrimaryCharID:   int64(testData.TestUser.CharacterID),
			RequestType:     "bearer",
			IsAuthenticated: true,
		}
		ctx := context.WithValue(req.Context(), AuthContextKeyAuth, authCtx)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		// Create test handler
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expandedCtx := GetExpandedAuthContext(r.Context())
			assert.NotNil(t, expandedCtx)
			assert.True(t, expandedCtx.IsAuthenticated)
			assert.Len(t, expandedCtx.CharacterIDs, 2)
			assert.Contains(t, expandedCtx.CharacterIDs, int64(123456789))
			assert.Contains(t, expandedCtx.CharacterIDs, int64(987654321))
			assert.Len(t, expandedCtx.CorporationIDs, 1)
			assert.Contains(t, expandedCtx.CorporationIDs, int64(987654321))
			w.WriteHeader(http.StatusOK)
		})

		// Apply middleware and test
		middleware := enhanced.CharacterResolutionMiddleware()
		handler := middleware(testHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mockCharacterResolver.AssertExpectations(t)
	})
}

func TestPermissionCheckRequest_Validation(t *testing.T) {
	t.Run("ValidRequest", func(t *testing.T) {
		request := PermissionCheckRequest{
			UserID:   "test-user-123",
			Resource: "scheduler.tasks",
			Action:   "read",
			Domain:   "global",
		}

		assert.Equal(t, "test-user-123", request.UserID)
		assert.Equal(t, "scheduler.tasks", request.Resource)
		assert.Equal(t, "read", request.Action)
	})

	t.Run("PolicyCreateRequest", func(t *testing.T) {
		request := PolicyCreateRequest{
			SubjectType: "user",
			SubjectID:   "test-user-123",
			Resource:    "scheduler.tasks",
			Action:      "read",
			Effect:      "allow",
		}

		assert.Equal(t, "user", request.SubjectType)
		assert.Equal(t, "test-user-123", request.SubjectID)
		assert.Equal(t, "allow", request.Effect)
	})

	t.Run("RoleCreateRequest", func(t *testing.T) {
		request := RoleCreateRequest{
			RoleName:    "admin",
			SubjectType: "user",
			SubjectID:   "test-user-123",
		}

		assert.Equal(t, "admin", request.RoleName)
		assert.Equal(t, "user", request.SubjectType)
	})
}

func TestCasbinCache_KeyGeneration(t *testing.T) {
	config := DefaultCacheConfig()
	
	t.Run("PermissionCacheKey", func(t *testing.T) {
		service := &CachedCasbinService{
			config: config,
		}

		key1 := service.generatePermissionCacheKey("user123", "scheduler.tasks", "read")
		key2 := service.generatePermissionCacheKey("user123", "scheduler.tasks", "read")
		key3 := service.generatePermissionCacheKey("user123", "scheduler.tasks", "write")

		// Same parameters should generate same key
		assert.Equal(t, key1, key2)
		
		// Different parameters should generate different keys
		assert.NotEqual(t, key1, key3)
		
		// Keys should have correct prefix
		assert.Contains(t, key1, config.KeyPrefix+"perm:")
	})

	t.Run("HierarchyCacheKey", func(t *testing.T) {
		service := &CachedCasbinService{
			config: config,
		}

		key1 := service.generateHierarchyCacheKey("user123")
		key2 := service.generateHierarchyCacheKey("user123")
		key3 := service.generateHierarchyCacheKey("user456")

		// Same user should generate same key
		assert.Equal(t, key1, key2)
		
		// Different users should generate different keys
		assert.NotEqual(t, key1, key3)
		
		// Keys should have correct prefix
		assert.Contains(t, key1, config.KeyPrefix+"hier:")
	})
}

func TestMiddlewareFactory_Creation(t *testing.T) {
	t.Run("FactoryStructure", func(t *testing.T) {
		// Test the factory structure without actual MongoDB connection
		mockJWTValidator := &MockJWTValidator{}
		mockCharacterResolver := &MockUserCharacterResolver{}

		// We can't create an actual factory without MongoDB, but we can test
		// that the structures and interfaces are correct
		
		assert.Implements(t, (*JWTValidator)(nil), mockJWTValidator)
		assert.Implements(t, (*UserCharacterResolver)(nil), mockCharacterResolver)
	})
}

func TestCasbinConvenienceMiddleware_Methods(t *testing.T) {
	t.Run("MethodExistence", func(t *testing.T) {
		// Test that convenience methods exist with correct signatures
		// This is a compile-time test to ensure interfaces are correct
		
		var convenience *CasbinConvenienceMiddleware
		
		// These should compile without error
		_ = convenience.RequireAuth()
		_ = convenience.RequireAuthWithCharacters()
		_ = convenience.OptionalAuth()
		_ = convenience.RequirePermission("resource", "action")
		_ = convenience.OptionalPermission("resource", "action")
		_ = convenience.AdminOnly()
		_ = convenience.SuperAdminOnly()
		_ = convenience.ModuleAccess("module", "action")
		_ = convenience.CorporationAccess("resource")
		_ = convenience.AllianceAccess("resource")
	})
}

func TestExpandedAuthContext_Structure(t *testing.T) {
	t.Run("ContextFields", func(t *testing.T) {
		expandedCtx := &ExpandedAuthContext{
			AuthContext: &AuthContext{
				UserID:          "test-user-123",
				PrimaryCharID:   123456789,
				RequestType:     "bearer",
				IsAuthenticated: true,
			},
			CharacterIDs:   []int64{123456789, 987654321},
			CorporationIDs: []int64{987654321},
			AllianceIDs:    []int64{111222333},
			Roles:          []string{"admin", "user"},
			Permissions:    []string{"scheduler.tasks.read", "users.read"},
		}

		assert.True(t, expandedCtx.IsAuthenticated)
		assert.Equal(t, "test-user-123", expandedCtx.UserID)
		assert.Len(t, expandedCtx.CharacterIDs, 2)
		assert.Len(t, expandedCtx.CorporationIDs, 1)
		assert.Len(t, expandedCtx.AllianceIDs, 1)
		assert.Len(t, expandedCtx.Roles, 2)
		assert.Len(t, expandedCtx.Permissions, 2)
		assert.Contains(t, expandedCtx.Roles, "admin")
		assert.Contains(t, expandedCtx.Permissions, "scheduler.tasks.read")
	})
}

// Benchmark tests for performance
func BenchmarkCasbinAuthMiddleware_buildSubjects(b *testing.B) {
	expandedCtx := &ExpandedAuthContext{
		AuthContext: &AuthContext{
			UserID:          "test-user-123",
			PrimaryCharID:   123456789,
			IsAuthenticated: true,
		},
		CharacterIDs:   []int64{123456789, 987654321, 111222333},
		CorporationIDs: []int64{987654321, 555666777},
		AllianceIDs:    []int64{111222333},
	}

	middleware := &CasbinAuthMiddleware{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subjects := middleware.buildSubjects(expandedCtx)
		_ = subjects
	}
}

func BenchmarkPermissionCacheKeyGeneration(b *testing.B) {
	service := &CachedCasbinService{
		config: DefaultCacheConfig(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := service.generatePermissionCacheKey("user123", "scheduler.tasks", "read")
		_ = key
	}
}