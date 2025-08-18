package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/casbin/casbin/v2"
	mongodbadapter "github.com/casbin/mongodb-adapter/v3"
	"go.mongodb.org/mongo-driver/mongo"
)

// CasbinAuthMiddleware provides Casbin-based authorization middleware
type CasbinAuthMiddleware struct {
	enforcer *casbin.Enforcer
	enabled  bool
}

// NewCasbinAuthMiddleware creates a new Casbin authorization middleware
func NewCasbinAuthMiddleware(mongoClient *mongo.Client, dbName string) (*CasbinAuthMiddleware, error) {
	// Create MongoDB adapter config
	config := &mongodbadapter.AdapterConfig{
		DatabaseName:   dbName,
		CollectionName: "casbin_policies",
	}
	
	// Create MongoDB adapter for Casbin policies
	adapter, err := mongodbadapter.NewAdapterByDB(mongoClient, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin MongoDB adapter: %w", err)
	}

	// Get the path to the Casbin model configuration
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(currentFile), "..", "..")
	modelPath := filepath.Join(projectRoot, "configs", "casbin_model.conf")

	// Create Casbin enforcer
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	// Enable automatic save (policies persist to MongoDB automatically)
	enforcer.EnableAutoSave(true)

	// Load policies from database
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load Casbin policies: %w", err)
	}

	slog.Info("Casbin authorization middleware initialized", 
		"model_path", modelPath,
		"adapter", "mongodb",
		"collection", "casbin_policies")

	return &CasbinAuthMiddleware{
		enforcer: enforcer,
		enabled:  true,
	}, nil
}

// RequirePermission creates middleware that checks for specific permission
func (c *CasbinAuthMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !c.enabled {
				next.ServeHTTP(w, r)
				return
			}

			fmt.Printf("[DEBUG] CasbinAuthMiddleware.RequirePermission: Checking %s.%s for %s %s\n", 
				resource, action, r.Method, r.URL.Path)

			// Get expanded auth context
			expandedCtx := GetExpandedAuthContext(r.Context())
			if expandedCtx == nil || !expandedCtx.IsAuthenticated {
				fmt.Printf("[DEBUG] CasbinAuthMiddleware: No authenticated user found\n")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check permissions at all hierarchy levels
			allowed, err := c.checkHierarchicalPermission(r.Context(), expandedCtx, resource, action)
			if err != nil {
				fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission check failed: %v\n", err)
				slog.Error("Permission check failed", "error", err, "user_id", expandedCtx.UserID)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission denied for user %s\n", expandedCtx.UserID)
				http.Error(w, "Permission denied", http.StatusForbidden)
				return
			}

			fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission granted for user %s\n", expandedCtx.UserID)
			next.ServeHTTP(w, r)
		})
	}
}

// checkHierarchicalPermission checks permissions across character/corp/alliance hierarchy
func (c *CasbinAuthMiddleware) checkHierarchicalPermission(ctx context.Context, authCtx *ExpandedAuthContext, resource, action string) (bool, error) {
	domain := "global" // For now, using global domain
	permission := fmt.Sprintf("%s.%s", resource, action)

	// Build subjects in priority order (character > corp > alliance)
	subjects := c.buildSubjects(authCtx)

	fmt.Printf("[DEBUG] CasbinAuthMiddleware: Checking permission '%s' for subjects: %v\n", permission, subjects)

	// Use the single enforce call - CASBIN handles allow/deny logic internally
	for _, subject := range subjects {
		result, err := c.enforcer.Enforce(subject, permission, action, domain)
		if err != nil {
			return false, fmt.Errorf("failed to check policy for subject %s: %w", subject, err)
		}
		if result {
			fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission granted for subject %s\n", subject)
			return true, nil
		}
		fmt.Printf("[DEBUG] CasbinAuthMiddleware: Permission denied for subject %s\n", subject)
	}

	// Default deny
	fmt.Printf("[DEBUG] CasbinAuthMiddleware: No explicit allow found, defaulting to deny\n")
	return false, nil
}

// buildSubjects creates ordered list of subjects for permission checking
func (c *CasbinAuthMiddleware) buildSubjects(authCtx *ExpandedAuthContext) []string {
	subjects := []string{}

	// Add character-level subjects (highest priority)
	subjects = append(subjects, fmt.Sprintf("user:%s", authCtx.UserID))
	subjects = append(subjects, fmt.Sprintf("character:%d", authCtx.PrimaryCharacter.ID))

	// Add corporation-level subjects
	for _, corpID := range authCtx.CorporationIDs {
		subjects = append(subjects, fmt.Sprintf("corporation:%d", corpID))
	}

	// Add alliance-level subjects (lowest priority)
	for _, allianceID := range authCtx.AllianceIDs {
		subjects = append(subjects, fmt.Sprintf("alliance:%d", allianceID))
	}

	return subjects
}

// AddPolicy adds a policy to Casbin
func (c *CasbinAuthMiddleware) AddPolicy(subject, resource, action, effect string) error {
	domain := "global"
	permission := fmt.Sprintf("%s.%s", resource, action)
	
	// CASBIN model expects: sub, obj, act, dom, eft
	added, err := c.enforcer.AddPolicy(subject, permission, action, domain, effect)
	if err != nil {
		return fmt.Errorf("failed to add policy: %w", err)
	}
	
	if !added {
		return fmt.Errorf("policy already exists")
	}
	
	slog.Info("Added Casbin policy", 
		"subject", subject,
		"permission", permission,
		"effect", effect,
		"domain", domain)
	
	return nil
}

// RemovePolicy removes a policy from Casbin
func (c *CasbinAuthMiddleware) RemovePolicy(subject, resource, action, effect string) error {
	domain := "global"
	permission := fmt.Sprintf("%s.%s", resource, action)
	
	// CASBIN model expects: sub, obj, act, dom, eft
	removed, err := c.enforcer.RemovePolicy(subject, permission, action, domain, effect)
	if err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}
	
	if !removed {
		return fmt.Errorf("policy not found")
	}
	
	slog.Info("Removed Casbin policy", 
		"subject", subject,
		"permission", permission,
		"effect", effect,
		"domain", domain)
	
	return nil
}

// AddRoleForUser adds a role assignment
func (c *CasbinAuthMiddleware) AddRoleForUser(user, role string) error {
	domain := "global"
	
	added, err := c.enforcer.AddGroupingPolicy(user, role, domain)
	if err != nil {
		return fmt.Errorf("failed to add role for user: %w", err)
	}
	
	if !added {
		return fmt.Errorf("role assignment already exists")
	}
	
	slog.Info("Added role assignment", 
		"user", user,
		"role", role,
		"domain", domain)
	
	return nil
}

// RemoveRoleForUser removes a role assignment
func (c *CasbinAuthMiddleware) RemoveRoleForUser(user, role string) error {
	domain := "global"
	
	removed, err := c.enforcer.RemoveGroupingPolicy(user, role, domain)
	if err != nil {
		return fmt.Errorf("failed to remove role for user: %w", err)
	}
	
	if !removed {
		return fmt.Errorf("role assignment not found")
	}
	
	slog.Info("Removed role assignment", 
		"user", user,
		"role", role,
		"domain", domain)
	
	return nil
}

// GetRolesForUser gets all roles for a user
func (c *CasbinAuthMiddleware) GetRolesForUser(user string) ([]string, error) {
	roles, err := c.enforcer.GetRolesForUser(user)
	return roles, err
}

// GetUsersForRole gets all users with a specific role
func (c *CasbinAuthMiddleware) GetUsersForRole(role string) ([]string, error) {
	users, err := c.enforcer.GetUsersForRole(role)
	return users, err
}

// GetPermissionsForUser gets all permissions for a user
func (c *CasbinAuthMiddleware) GetPermissionsForUser(user string) ([][]string, error) {
	permissions, err := c.enforcer.GetPermissionsForUser(user)
	return permissions, err
}

// GetAllPolicies gets all policies
func (c *CasbinAuthMiddleware) GetAllPolicies() ([][]string, error) {
	return c.enforcer.GetPolicy()
}

// GetAllRoles gets all role assignments
func (c *CasbinAuthMiddleware) GetAllRoles() ([][]string, error) {
	return c.enforcer.GetGroupingPolicy()
}

// Disable temporarily disables Casbin enforcement (useful for testing)
func (c *CasbinAuthMiddleware) Disable() {
	c.enabled = false
	slog.Warn("Casbin authorization middleware disabled")
}

// Enable re-enables Casbin enforcement
func (c *CasbinAuthMiddleware) Enable() {
	c.enabled = true
	slog.Info("Casbin authorization middleware enabled")
}

// IsEnabled returns whether Casbin enforcement is enabled
func (c *CasbinAuthMiddleware) IsEnabled() bool {
	return c.enabled
}

// ReloadPolicies reloads policies from database
func (c *CasbinAuthMiddleware) ReloadPolicies() error {
	if err := c.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("failed to reload Casbin policies: %w", err)
	}
	
	slog.Info("Casbin policies reloaded from database")
	return nil
}