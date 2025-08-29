package evegateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-falcon/pkg/database"

	"github.com/redis/go-redis/v9"
)

// RedisCacheManager implements CacheManager using Redis for persistence
type RedisCacheManager struct {
	redis *database.Redis
	ctx   context.Context
}

// NewRedisCacheManager creates a new Redis-based cache manager
func NewRedisCacheManager(redis *database.Redis) *RedisCacheManager {
	return &RedisCacheManager{
		redis: redis,
		ctx:   context.Background(),
	}
}

// Get retrieves data from Redis cache
func (r *RedisCacheManager) Get(key string) ([]byte, bool, error) {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	// Get cache entry from Redis
	entryJSON, err := r.redis.Get(r.ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil // Cache miss
		}
		return nil, false, err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Check if entry has expired
	if entry.Expires.Before(time.Now()) {
		// Remove expired entry
		r.redis.Delete(r.ctx, cacheKey)
		return nil, false, nil
	}

	return entry.Data, true, nil
}

// GetWithExpiry retrieves data from Redis cache along with expiry time
func (r *RedisCacheManager) GetWithExpiry(key string) ([]byte, bool, *time.Time, error) {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	entryJSON, err := r.redis.Get(r.ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil, nil // Cache miss
		}
		return nil, false, nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return nil, false, nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Check if entry has expired
	if entry.Expires.Before(time.Now()) {
		r.redis.Delete(r.ctx, cacheKey)
		return nil, false, nil, nil
	}

	return entry.Data, true, &entry.Expires, nil
}

// GetForNotModified retrieves data from Redis cache even if expired (for 304 responses)
func (r *RedisCacheManager) GetForNotModified(key string) ([]byte, bool, error) {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	entryJSON, err := r.redis.Get(r.ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil // Cache miss
		}
		return nil, false, err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	return entry.Data, true, nil
}

// GetMetadata retrieves metadata about a cached entry from Redis
func (r *RedisCacheManager) GetMetadata(key string) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	entryJSON, err := r.redis.Get(r.ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	metadata := map[string]interface{}{
		"expires_at":    entry.Expires,
		"etag":          entry.ETag,
		"last_modified": entry.LastModified,
		"cached":        true,
	}

	return metadata, nil
}

// RefreshExpiry updates the expiry time of a cached entry in Redis (for 304 responses)
func (r *RedisCacheManager) RefreshExpiry(key string, headers http.Header) error {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	entryJSON, err := r.redis.Get(r.ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil // Entry doesn't exist, nothing to refresh
		}
		return err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	// Parse expires header first (ESI primary cache header)
	if expires := headers.Get("Expires"); expires != "" {
		// Try RFC1123 format first (standard HTTP date format)
		if parsedTime, err := time.Parse(time.RFC1123, expires); err == nil {
			entry.Expires = parsedTime
		} else if parsedTime, err := time.Parse(time.RFC1123Z, expires); err == nil {
			// Try RFC1123Z format as fallback
			entry.Expires = parsedTime
		}
	} else if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		// Parse Cache-Control for max-age as fallback
		if maxAge := parseCacheControlMaxAge(cacheControl); maxAge > 0 {
			entry.Expires = time.Now().Add(time.Duration(maxAge) * time.Second)
		}
	} else {
		// Default refresh time if no cache headers
		entry.Expires = time.Now().Add(5 * time.Second)
	}

	// Save updated entry back to Redis
	updatedEntryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal updated cache entry: %w", err)
	}

	// Calculate time until expiry for Redis TTL
	ttl := time.Until(entry.Expires)
	if ttl < 0 {
		ttl = 5 * time.Second // Minimum TTL
	}

	return r.redis.Set(r.ctx, cacheKey, updatedEntryJSON, ttl)
}

// Set stores data in Redis cache
func (r *RedisCacheManager) Set(key string, data []byte, headers http.Header) error {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	entry := &CacheEntry{
		Data:         data,
		ETag:         headers.Get("ETag"),
		LastModified: headers.Get("Last-Modified"),
		Expires:      time.Now().Add(5 * time.Second), // Default 5s cache
	}

	// Parse expires header first (ESI primary cache header)
	if expires := headers.Get("Expires"); expires != "" {
		// Try RFC1123 format first (standard HTTP date format)
		if parsedTime, err := time.Parse(time.RFC1123, expires); err == nil {
			entry.Expires = parsedTime
		} else if parsedTime, err := time.Parse(time.RFC1123Z, expires); err == nil {
			// Try RFC1123Z format as fallback
			entry.Expires = parsedTime
		}
	} else if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		// Parse Cache-Control for max-age as fallback
		if maxAge := parseCacheControlMaxAge(cacheControl); maxAge > 0 {
			entry.Expires = time.Now().Add(time.Duration(maxAge) * time.Second)
		}
	}

	// Serialize the cache entry to JSON
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// Calculate time until expiry for Redis TTL
	ttl := time.Until(entry.Expires)
	if ttl < 0 {
		ttl = 5 * time.Second // Minimum TTL
	}

	return r.redis.Set(r.ctx, cacheKey, entryJSON, ttl)
}

// SetConditionalHeaders sets conditional headers if cached data exists in Redis
func (r *RedisCacheManager) SetConditionalHeaders(req *http.Request, key string) error {
	cacheKey := fmt.Sprintf("esi:cache:%s", key)

	entryJSON, err := r.redis.Get(r.ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil // No cached data
		}
		return err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
		return fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	if entry.ETag != "" {
		req.Header.Set("If-None-Match", entry.ETag)
	}
	if entry.LastModified != "" {
		req.Header.Set("If-Modified-Since", entry.LastModified)
	}

	return nil
}
