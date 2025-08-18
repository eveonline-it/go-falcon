package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/pkg/database"

	"github.com/redis/go-redis/v9"
)

// SchedulerCache provides Redis caching for scheduler operations
type SchedulerCache struct {
	redis *database.Redis
}

// NewSchedulerCache creates a new scheduler cache
func NewSchedulerCache(redis *database.Redis) *SchedulerCache {
	return &SchedulerCache{
		redis: redis,
	}
}

// CachedSchedulerStatus represents cached scheduler status
type CachedSchedulerStatus struct {
	Status    *dto.SchedulerStatusResponse `json:"status"`
	CachedAt  time.Time                    `json:"cached_at"`
	ExpiresAt time.Time                    `json:"expires_at"`
}

// CachedSchedulerStats represents cached scheduler statistics
type CachedSchedulerStats struct {
	Stats     *dto.SchedulerStatsResponse `json:"stats"`
	CachedAt  time.Time                   `json:"cached_at"`
	ExpiresAt time.Time                   `json:"expires_at"`
}

// GetSchedulerStatus retrieves cached scheduler status
func (c *SchedulerCache) GetSchedulerStatus(ctx context.Context) (*dto.SchedulerStatusResponse, error) {
	if c.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	cacheKey := "scheduler:status"
	
	var cachedStatus CachedSchedulerStatus
	err := c.redis.GetJSON(ctx, cacheKey, &cachedStatus)
	if err != nil {
		return nil, err
	}

	// Check if cached data is still valid
	if time.Now().After(cachedStatus.ExpiresAt) {
		// Cache expired, delete it
		c.redis.Delete(ctx, cacheKey)
		return nil, redis.Nil
	}

	fmt.Printf("[DEBUG] SchedulerCache: Status cache HIT (valid until %v)\n", cachedStatus.ExpiresAt)
	return cachedStatus.Status, nil
}

// SetSchedulerStatus caches scheduler status with TTL
func (c *SchedulerCache) SetSchedulerStatus(ctx context.Context, status *dto.SchedulerStatusResponse, ttl time.Duration) error {
	if c.redis == nil {
		return nil // Silently skip if Redis not available
	}

	cacheKey := "scheduler:status"
	
	cachedStatus := CachedSchedulerStatus{
		Status:    status,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	err := c.redis.SetJSON(ctx, cacheKey, cachedStatus, ttl)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to cache status: %v\n", err)
		return err
	}

	fmt.Printf("[DEBUG] SchedulerCache: Cached status for %v\n", ttl)
	return nil
}

// GetSchedulerStats retrieves cached scheduler statistics
func (c *SchedulerCache) GetSchedulerStats(ctx context.Context) (*dto.SchedulerStatsResponse, error) {
	if c.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	cacheKey := "scheduler:stats"
	
	var cachedStats CachedSchedulerStats
	err := c.redis.GetJSON(ctx, cacheKey, &cachedStats)
	if err != nil {
		return nil, err
	}

	// Check if cached data is still valid
	if time.Now().After(cachedStats.ExpiresAt) {
		// Cache expired, delete it
		c.redis.Delete(ctx, cacheKey)
		return nil, redis.Nil
	}

	fmt.Printf("[DEBUG] SchedulerCache: Stats cache HIT (valid until %v)\n", cachedStats.ExpiresAt)
	return cachedStats.Stats, nil
}

// SetSchedulerStats caches scheduler statistics with TTL
func (c *SchedulerCache) SetSchedulerStats(ctx context.Context, stats *dto.SchedulerStatsResponse, ttl time.Duration) error {
	if c.redis == nil {
		return nil // Silently skip if Redis not available
	}

	cacheKey := "scheduler:stats"
	
	cachedStats := CachedSchedulerStats{
		Stats:     stats,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	err := c.redis.SetJSON(ctx, cacheKey, cachedStats, ttl)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to cache stats: %v\n", err)
		return err
	}

	fmt.Printf("[DEBUG] SchedulerCache: Cached stats for %v\n", ttl)
	return nil
}

// InvalidateStatus removes cached status
func (c *SchedulerCache) InvalidateStatus(ctx context.Context) error {
	if c.redis == nil {
		return nil
	}

	cacheKey := "scheduler:status"
	err := c.redis.Delete(ctx, cacheKey)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to invalidate status cache: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] SchedulerCache: Status cache invalidated\n")
	}
	return err
}

// InvalidateStats removes cached statistics
func (c *SchedulerCache) InvalidateStats(ctx context.Context) error {
	if c.redis == nil {
		return nil
	}

	cacheKey := "scheduler:stats"
	err := c.redis.Delete(ctx, cacheKey)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to invalidate stats cache: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] SchedulerCache: Stats cache invalidated\n")
	}
	return err
}

// InvalidateAll removes all scheduler-related cached data
func (c *SchedulerCache) InvalidateAll(ctx context.Context) error {
	if c.redis == nil {
		return nil
	}

	fmt.Printf("[DEBUG] SchedulerCache: Invalidating all scheduler cache entries\n")
	
	// Delete status and stats caches
	keys := []string{"scheduler:status", "scheduler:stats"}
	err := c.redis.Delete(ctx, keys...)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to invalidate all caches: %v\n", err)
	} else {
		fmt.Printf("[DEBUG] SchedulerCache: All scheduler caches invalidated\n")
	}
	return err
}

// InvalidateOnTaskChange invalidates relevant caches when tasks are modified
func (c *SchedulerCache) InvalidateOnTaskChange(ctx context.Context, operation string) error {
	if c.redis == nil {
		return nil
	}

	fmt.Printf("[DEBUG] SchedulerCache: Invalidating caches due to task %s\n", operation)
	
	// Task changes affect both status and stats
	// Status: changes task counts and engine state
	// Stats: changes task counts, success/failure rates, etc.
	keys := []string{"scheduler:status", "scheduler:stats"}
	
	err := c.redis.Delete(ctx, keys...)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to invalidate task-related caches: %v\n", err)
		return err
	}
	
	fmt.Printf("[DEBUG] SchedulerCache: Successfully invalidated %d caches for task %s\n", len(keys), operation)
	return nil
}

// InvalidateOnExecutionChange invalidates stats cache when executions complete
func (c *SchedulerCache) InvalidateOnExecutionChange(ctx context.Context, taskID string, status string) error {
	if c.redis == nil {
		return nil
	}

	fmt.Printf("[DEBUG] SchedulerCache: Invalidating stats cache due to execution %s (task: %s)\n", status, taskID)
	
	// Execution changes primarily affect stats (success/failure counts, averages)
	// Status is less frequently affected unless it changes running task count
	statsKey := "scheduler:stats"
	
	err := c.redis.Delete(ctx, statsKey)
	if err != nil {
		fmt.Printf("[DEBUG] SchedulerCache: Failed to invalidate stats cache: %v\n", err)
		return err
	}
	
	fmt.Printf("[DEBUG] SchedulerCache: Successfully invalidated stats cache for execution %s\n", status)
	return nil
}