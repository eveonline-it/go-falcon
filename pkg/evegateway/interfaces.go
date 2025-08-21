package evegateway

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CacheEntry represents a cached ESI response
type CacheEntry struct {
	Data         []byte
	ETag         string
	LastModified string
	Expires      time.Time
}

// ESIErrorLimits represents ESI error limit headers
type ESIErrorLimits struct {
	Remain int
	Reset  time.Time
	Window int
}

// CacheManager interface for caching operations
type CacheManager interface {
	Get(key string) ([]byte, bool, error)
	GetWithExpiry(key string) ([]byte, bool, *time.Time, error)
	GetForNotModified(key string) ([]byte, bool, error)
	GetMetadata(key string) (map[string]interface{}, error)
	Set(key string, data []byte, headers http.Header) error
	RefreshExpiry(key string, headers http.Header) error
	SetConditionalHeaders(req *http.Request, key string) error
}

// RetryClient interface for retry operations
type RetryClient interface {
	DoWithRetry(ctx context.Context, req *http.Request, maxRetries int) (*http.Response, error)
}

// ErrorLimitManager interface for ESI error limit tracking
type ErrorLimitManager interface {
	CheckLimits() error
	UpdateLimits(headers http.Header)
}

// DefaultCacheManager implements basic in-memory caching
type DefaultCacheManager struct {
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex
}

// NewDefaultCacheManager creates a new default cache manager
func NewDefaultCacheManager() *DefaultCacheManager {
	return &DefaultCacheManager{
		cache: make(map[string]*CacheEntry),
	}
}

// Get retrieves data from cache
func (c *DefaultCacheManager) Get(key string) ([]byte, bool, error) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists || entry.Expires.Before(time.Now()) {
		return nil, false, nil
	}

	return entry.Data, true, nil
}

// GetWithExpiry retrieves data from cache along with expiry time
func (c *DefaultCacheManager) GetWithExpiry(key string) ([]byte, bool, *time.Time, error) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists || entry.Expires.Before(time.Now()) {
		return nil, false, nil, nil
	}

	return entry.Data, true, &entry.Expires, nil
}

// GetForNotModified retrieves data from cache even if expired (for 304 responses)
func (c *DefaultCacheManager) GetForNotModified(key string) ([]byte, bool, error) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false, nil
	}

	return entry.Data, true, nil
}

// GetMetadata retrieves metadata about a cached entry
func (c *DefaultCacheManager) GetMetadata(key string) (map[string]interface{}, error) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, nil
	}

	metadata := map[string]interface{}{
		"expires_at":     entry.Expires,
		"etag":          entry.ETag,
		"last_modified": entry.LastModified,
		"cached":        true,
	}

	return metadata, nil
}

// RefreshExpiry updates the expiry time of a cached entry (for 304 responses)
func (c *DefaultCacheManager) RefreshExpiry(key string, headers http.Header) error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil
	}

	// Parse expires header first (ESI primary cache header)
	if expires := headers.Get("Expires"); expires != "" {
		// Try RFC1123 format first (standard HTTP date format)
		if parsedTime, err := time.Parse(time.RFC1123, expires); err == nil {
			entry.Expires = parsedTime
			c.cache[key] = entry
			return nil
		} else if parsedTime, err := time.Parse(time.RFC1123Z, expires); err == nil {
			// Try RFC1123Z format as fallback
			entry.Expires = parsedTime
			c.cache[key] = entry
			return nil
		}
	}

	// Parse Cache-Control for max-age as fallback
	if cacheControl := headers.Get("Cache-Control"); cacheControl != "" {
		if maxAge := parseCacheControlMaxAge(cacheControl); maxAge > 0 {
			entry.Expires = time.Now().Add(time.Duration(maxAge) * time.Second)
			c.cache[key] = entry
			return nil
		}
	}

	// Default refresh time if no cache headers
	entry.Expires = time.Now().Add(5 * time.Second)
	c.cache[key] = entry
	return nil
}

// Set stores data in cache
func (c *DefaultCacheManager) Set(key string, data []byte, headers http.Header) error {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

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

	c.cache[key] = entry
	return nil
}

// SetConditionalHeaders sets conditional headers if cached data exists
func (c *DefaultCacheManager) SetConditionalHeaders(req *http.Request, key string) error {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil
	}

	if entry.ETag != "" {
		req.Header.Set("If-None-Match", entry.ETag)
	}
	if entry.LastModified != "" {
		req.Header.Set("If-Modified-Since", entry.LastModified)
	}

	return nil
}

// parseCacheControlMaxAge is a simple parser for max-age directive
func parseCacheControlMaxAge(cacheControl string) int {
	// This is a simplified implementation
	// In production, you might want to use a proper HTTP header parser
	if !strings.Contains(cacheControl, "max-age=") {
		return 0
	}
	
	parts := strings.Split(cacheControl, "max-age=")
	if len(parts) < 2 {
		return 0
	}
	
	maxAgeStr := strings.Split(parts[1], ",")[0]
	maxAge, err := strconv.Atoi(strings.TrimSpace(maxAgeStr))
	if err != nil {
		return 0
	}
	
	return maxAge
}