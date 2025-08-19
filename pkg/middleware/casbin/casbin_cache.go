package casbin

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/middleware"
	"github.com/redis/go-redis/v9"
)

// CasbinCacheConfig configures the caching behavior for Casbin decisions
type CasbinCacheConfig struct {
	TTL                   time.Duration // Cache TTL for permission decisions
	HierarchyTTL         time.Duration // Cache TTL for user hierarchies
	PolicyTTL            time.Duration // Cache TTL for policy lists
	EnablePermissionCache bool          // Enable caching of permission decisions
	EnableHierarchyCache  bool          // Enable caching of user hierarchies
	EnablePolicyCache     bool          // Enable caching of policy data
	KeyPrefix             string        // Redis key prefix
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CasbinCacheConfig {
	return &CasbinCacheConfig{
		TTL:                   5 * time.Minute,  // 5 minutes for permission decisions
		HierarchyTTL:         15 * time.Minute, // 15 minutes for hierarchies (more stable)
		PolicyTTL:            10 * time.Minute, // 10 minutes for policies
		EnablePermissionCache: true,
		EnableHierarchyCache:  true,
		EnablePolicyCache:     true,
		KeyPrefix:             "casbin:",
	}
}

// CachedCasbinService wraps CasbinService with Redis caching
type CachedCasbinService struct {
	*CasbinService
	redisClient *redis.Client
	config      *CasbinCacheConfig
}

// NewCachedCasbinService creates a new cached Casbin service
func NewCachedCasbinService(service *CasbinService, redisClient *redis.Client, config *CasbinCacheConfig) *CachedCasbinService {
	if config == nil {
		config = DefaultCacheConfig()
	}

	return &CachedCasbinService{
		CasbinService: service,
		redisClient:   redisClient,
		config:        config,
	}
}

// CheckPermission checks permission with caching
func (s *CachedCasbinService) CheckPermission(ctx context.Context, userID string, resource, action string) (*PermissionCheckResponse, error) {
	if !s.config.EnablePermissionCache {
		return s.CasbinService.CheckPermission(ctx, userID, resource, action)
	}

	// Generate cache key
	cacheKey := s.generatePermissionCacheKey(userID, resource, action)

	// Try to get from cache
	cached, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var response PermissionCheckResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			slog.Debug("Permission check cache hit", "user_id", userID, "resource", resource, "action", action)
			return &response, nil
		}
	}

	// Cache miss, get from service
	response, err := s.CasbinService.CheckPermission(ctx, userID, resource, action)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if responseJSON, err := json.Marshal(response); err == nil {
		s.redisClient.Set(ctx, cacheKey, responseJSON, s.config.TTL)
		slog.Debug("Permission check cached", "user_id", userID, "resource", resource, "action", action)
	}

	return response, nil
}

// getUserExpandedContext overrides the parent method with caching
func (s *CachedCasbinService) getUserExpandedContext(ctx context.Context, userID string) (*middleware.ExpandedAuthContext, error) {
	if !s.config.EnableHierarchyCache {
		return s.CasbinService.getUserExpandedContext(ctx, userID)
	}

	// Generate cache key
	cacheKey := s.generateHierarchyCacheKey(userID)

	// Try to get from cache
	cached, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var expandedCtx middleware.ExpandedAuthContext
		if err := json.Unmarshal([]byte(cached), &expandedCtx); err == nil {
			slog.Debug("Hierarchy cache hit", "user_id", userID)
			return &expandedCtx, nil
		}
	}

	// Cache miss, get from service
	expandedCtx, err := s.CasbinService.getUserExpandedContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if ctxJSON, err := json.Marshal(expandedCtx); err == nil {
		s.redisClient.Set(ctx, cacheKey, ctxJSON, s.config.HierarchyTTL)
		slog.Debug("Hierarchy cached", "user_id", userID)
	}

	return expandedCtx, nil
}

// SyncUserHierarchy overrides parent method and invalidates cache
func (s *CachedCasbinService) SyncUserHierarchy(ctx context.Context, userID string, characters []middleware.UserCharacter) error {
	// Sync the hierarchy
	err := s.CasbinService.SyncUserHierarchy(ctx, userID, characters)
	if err != nil {
		return err
	}

	// Invalidate related caches
	s.invalidateUserCaches(ctx, userID)

	return nil
}

// GrantPermission overrides parent method and invalidates cache
func (s *CachedCasbinService) GrantPermission(ctx context.Context, request *PolicyCreateRequest, performedBy int64) error {
	err := s.CasbinService.GrantPermission(ctx, request, performedBy)
	if err != nil {
		return err
	}

	// Invalidate permission caches for affected subjects
	s.invalidateSubjectCaches(ctx, request.SubjectType, request.SubjectID)

	return nil
}

// RevokePermission overrides parent method and invalidates cache
func (s *CachedCasbinService) RevokePermission(ctx context.Context, subjectType, subjectID, resource, action, effect string, performedBy int64) error {
	err := s.CasbinService.RevokePermission(ctx, subjectType, subjectID, resource, action, effect, performedBy)
	if err != nil {
		return err
	}

	// Invalidate permission caches for affected subjects
	s.invalidateSubjectCaches(ctx, subjectType, subjectID)

	return nil
}

// AssignRole overrides parent method and invalidates cache
func (s *CachedCasbinService) AssignRole(ctx context.Context, request *RoleCreateRequest, performedBy int64) error {
	err := s.CasbinService.AssignRole(ctx, request, performedBy)
	if err != nil {
		return err
	}

	// Invalidate permission caches for affected subjects
	s.invalidateSubjectCaches(ctx, request.SubjectType, request.SubjectID)

	return nil
}

// RevokeRole overrides parent method and invalidates cache
func (s *CachedCasbinService) RevokeRole(ctx context.Context, subjectType, subjectID, roleName string, performedBy int64) error {
	err := s.CasbinService.RevokeRole(ctx, subjectType, subjectID, roleName, performedBy)
	if err != nil {
		return err
	}

	// Invalidate permission caches for affected subjects
	s.invalidateSubjectCaches(ctx, subjectType, subjectID)

	return nil
}

// generatePermissionCacheKey generates a cache key for permission checks
func (s *CachedCasbinService) generatePermissionCacheKey(userID, resource, action string) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", userID, resource, action)))
	return fmt.Sprintf("%sperm:%x", s.config.KeyPrefix, hash)
}

// generateHierarchyCacheKey generates a cache key for user hierarchies
func (s *CachedCasbinService) generateHierarchyCacheKey(userID string) string {
	return fmt.Sprintf("%shier:%s", s.config.KeyPrefix, userID)
}

// generateSubjectCachePattern generates a pattern for subject-related cache keys
func (s *CachedCasbinService) generateSubjectCachePattern(subjectType, subjectID string) string {
	if subjectType == "user" {
		return fmt.Sprintf("%s*:%s:*", s.config.KeyPrefix, subjectID)
	}
	// For non-user subjects, we need to find which users have this subject in their hierarchy
	return fmt.Sprintf("%sperm:*", s.config.KeyPrefix)
}

// invalidateUserCaches invalidates all caches related to a specific user
func (s *CachedCasbinService) invalidateUserCaches(ctx context.Context, userID string) {
	// Invalidate hierarchy cache
	hierarchyKey := s.generateHierarchyCacheKey(userID)
	s.redisClient.Del(ctx, hierarchyKey)

	// Invalidate permission caches (find all permission keys for this user)
	pattern := fmt.Sprintf("%sperm:*", s.config.KeyPrefix)
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err == nil {
		for _, key := range keys {
			// For now, invalidate all permission caches
			// In a more sophisticated implementation, we could parse the key
			// to check if it relates to this user
			s.redisClient.Del(ctx, key)
		}
	}

	slog.Debug("Invalidated user caches", "user_id", userID)
}

// invalidateSubjectCaches invalidates caches for a specific subject
func (s *CachedCasbinService) invalidateSubjectCaches(ctx context.Context, subjectType, subjectID string) {
	if subjectType == "user" {
		s.invalidateUserCaches(ctx, subjectID)
		return
	}

	// For other subjects (character, corporation, alliance), we need to invalidate
	// caches for all users that might be affected by this change
	// For now, we'll invalidate all permission caches (safe but not optimal)
	pattern := fmt.Sprintf("%sperm:*", s.config.KeyPrefix)
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err == nil && len(keys) > 0 {
		s.redisClient.Del(ctx, keys...)
	}

	slog.Debug("Invalidated subject caches", "subject_type", subjectType, "subject_id", subjectID)
}

// InvalidateAllCaches invalidates all Casbin-related caches
func (s *CachedCasbinService) InvalidateAllCaches(ctx context.Context) error {
	pattern := fmt.Sprintf("%s*", s.config.KeyPrefix)
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	if len(keys) > 0 {
		_, err = s.redisClient.Del(ctx, keys...).Result()
		if err != nil {
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
	}

	slog.Info("Invalidated all Casbin caches", "deleted_keys", len(keys))
	return nil
}

// GetCacheStats returns cache statistics
func (s *CachedCasbinService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get cache key counts
	permissionKeys, err := s.redisClient.Keys(ctx, fmt.Sprintf("%sperm:*", s.config.KeyPrefix)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get permission cache keys: %w", err)
	}

	hierarchyKeys, err := s.redisClient.Keys(ctx, fmt.Sprintf("%shier:*", s.config.KeyPrefix)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchy cache keys: %w", err)
	}

	stats["permission_cache_entries"] = len(permissionKeys)
	stats["hierarchy_cache_entries"] = len(hierarchyKeys)
	stats["total_cache_entries"] = len(permissionKeys) + len(hierarchyKeys)
	stats["config"] = map[string]interface{}{
		"permission_ttl":         s.config.TTL.String(),
		"hierarchy_ttl":          s.config.HierarchyTTL.String(),
		"permission_cache_enabled": s.config.EnablePermissionCache,
		"hierarchy_cache_enabled":  s.config.EnableHierarchyCache,
		"key_prefix":              s.config.KeyPrefix,
	}

	return stats, nil
}

// WarmupUserCache pre-loads cache for a user's hierarchies and common permissions
func (s *CachedCasbinService) WarmupUserCache(ctx context.Context, userID string, commonPermissions []string) error {
	// Pre-load user hierarchy
	_, err := s.getUserExpandedContext(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to warmup hierarchy for user %s: %w", userID, err)
	}

	// Pre-load common permission checks
	for _, permission := range commonPermissions {
		parts := parsePermissionString(permission)
		if len(parts) >= 2 {
			resource := parts[0]
			action := parts[1]
			_, err := s.CheckPermission(ctx, userID, resource, action)
			if err != nil {
				slog.Warn("Failed to warmup permission cache", 
					"user_id", userID, 
					"permission", permission, 
					"error", err)
			}
		}
	}

	slog.Debug("Warmed up user cache", "user_id", userID, "permissions", len(commonPermissions))
	return nil
}

// CachedCasbinAuthMiddleware wraps CasbinAuthMiddleware with caching
type CachedCasbinAuthMiddleware struct {
	*CasbinAuthMiddleware
	cachedService *CachedCasbinService
}

// NewCachedCasbinAuthMiddleware creates a new cached Casbin auth middleware
func NewCachedCasbinAuthMiddleware(authMiddleware *CasbinAuthMiddleware, cachedService *CachedCasbinService) *CachedCasbinAuthMiddleware {
	return &CachedCasbinAuthMiddleware{
		CasbinAuthMiddleware: authMiddleware,
		cachedService:        cachedService,
	}
}

// checkHierarchicalPermission overrides the parent method to use cached service
func (c *CachedCasbinAuthMiddleware) checkHierarchicalPermission(ctx context.Context, authCtx *middleware.ExpandedAuthContext, resource, action string) (bool, error) {
	// Use the cached service's permission check
	response, err := c.cachedService.CheckPermission(ctx, authCtx.UserID, resource, action)
	if err != nil {
		return false, err
	}
	
	return response.Allowed, nil
}