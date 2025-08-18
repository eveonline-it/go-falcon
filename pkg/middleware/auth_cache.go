package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	authModels "go-falcon/internal/auth/models"
	"go-falcon/pkg/database"

	"github.com/redis/go-redis/v9"
)

// AuthCache provides Redis caching for authentication-related operations
type AuthCache struct {
	redis *database.Redis
}

// NewAuthCache creates a new authentication cache
func NewAuthCache(redis *database.Redis) *AuthCache {
	return &AuthCache{
		redis: redis,
	}
}

// CachedAuthenticatedUser represents a cached authenticated user with expiration info
type CachedAuthenticatedUser struct {
	User      *authModels.AuthenticatedUser `json:"user"`
	CachedAt  time.Time                     `json:"cached_at"`
	ExpiresAt time.Time                     `json:"expires_at"`
}

// GetAuthenticatedUser retrieves a cached authenticated user by token hash
func (c *AuthCache) GetAuthenticatedUser(ctx context.Context, tokenHash string) (*authModels.AuthenticatedUser, error) {
	if c.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	cacheKey := fmt.Sprintf("auth_user:%s", tokenHash)
	
	var cachedUser CachedAuthenticatedUser
	err := c.redis.GetJSON(ctx, cacheKey, &cachedUser)
	if err != nil {
		return nil, err
	}

	// Check if cached data is still valid
	if time.Now().After(cachedUser.ExpiresAt) {
		// Cache expired, delete it
		c.redis.Delete(ctx, cacheKey)
		return nil, redis.Nil
	}

	fmt.Printf("[DEBUG] AuthCache: Cache HIT for user %s (valid until %v)\n", 
		cachedUser.User.CharacterName, cachedUser.ExpiresAt)
	
	return cachedUser.User, nil
}

// SetAuthenticatedUser caches an authenticated user with TTL
func (c *AuthCache) SetAuthenticatedUser(ctx context.Context, tokenHash string, user *authModels.AuthenticatedUser, ttl time.Duration) error {
	if c.redis == nil {
		return nil // Silently skip if Redis not available
	}

	cacheKey := fmt.Sprintf("auth_user:%s", tokenHash)
	
	cachedUser := CachedAuthenticatedUser{
		User:      user,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	err := c.redis.SetJSON(ctx, cacheKey, cachedUser, ttl)
	if err != nil {
		fmt.Printf("[DEBUG] AuthCache: Failed to cache user %s: %v\n", user.CharacterName, err)
		return err
	}

	fmt.Printf("[DEBUG] AuthCache: Cached user %s for %v\n", user.CharacterName, ttl)
	return nil
}

// InvalidateUser removes a cached user by token hash
func (c *AuthCache) InvalidateUser(ctx context.Context, tokenHash string) error {
	if c.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("auth_user:%s", tokenHash)
	return c.redis.Delete(ctx, cacheKey)
}

// InvalidateUserByID removes all cached entries for a specific user ID
func (c *AuthCache) InvalidateUserByID(ctx context.Context, userID string) error {
	if c.redis == nil {
		return nil
	}

	fmt.Printf("[DEBUG] AuthCache: Starting user invalidation for userID: %s\n", userID)

	// Pattern-based cache invalidation using Redis SCAN
	// This approach is safer than KEYS as it doesn't block Redis
	pattern := fmt.Sprintf("auth_user:*")
	
	var cursor uint64 = 0
	var keysToDelete []string
	
	for {
		// Use SCAN to iterate through keys matching the pattern
		keys, nextCursor, err := c.redis.Client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			fmt.Printf("[DEBUG] AuthCache: Error scanning keys: %v\n", err)
			return fmt.Errorf("failed to scan cache keys: %w", err)
		}
		
		// For each key, check if it belongs to the target user
		for _, key := range keys {
			// Get the cached user to check if it matches the target userID
			var cachedUser CachedAuthenticatedUser
			err := c.redis.GetJSON(ctx, key, &cachedUser)
			if err != nil {
				// Skip keys that can't be read or are already expired
				continue
			}
			
			// Check if this cached entry belongs to the target user
			if cachedUser.User != nil && cachedUser.User.UserID == userID {
				keysToDelete = append(keysToDelete, key)
			}
		}
		
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	
	// Delete all matching keys in batches
	if len(keysToDelete) > 0 {
		err := c.redis.Client.Del(ctx, keysToDelete...).Err()
		if err != nil {
			fmt.Printf("[DEBUG] AuthCache: Failed to delete %d keys for user %s: %v\n", len(keysToDelete), userID, err)
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
		fmt.Printf("[DEBUG] AuthCache: Successfully invalidated %d cache entries for user %s\n", len(keysToDelete), userID)
	} else {
		fmt.Printf("[DEBUG] AuthCache: No cache entries found for user %s\n", userID)
	}
	
	return nil
}

// HashToken creates a SHA256 hash of a JWT token for use as cache key
func (c *AuthCache) HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

// GetCacheKey generates a cache key for a given token
func (c *AuthCache) GetCacheKey(token string) string {
	return c.HashToken(token)
}

// CacheStats returns statistics about cache usage
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Errors int64 `json:"errors"`
}

// TODO: Implement cache statistics if needed
func (c *AuthCache) GetStats(ctx context.Context) (*CacheStats, error) {
	// This would require maintaining counters in Redis
	return &CacheStats{}, nil
}