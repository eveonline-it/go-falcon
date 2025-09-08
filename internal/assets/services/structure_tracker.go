package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	failedStructurePrefix = "falcon:assets:failed_structures:"
	retryCandidatesKey    = "falcon:assets:retry_candidates"
	esiErrorBudgetPrefix  = "falcon:assets:esi_errors:"
	metricsPrefix         = "falcon:assets:metrics:"

	// Retry tiers and probabilities
	tier1MaxFailures = 2
	tier2MaxFailures = 5
	tier3MaxFailures = 10
	tier4MaxFailures = 20

	tier1MaxAge = 24 * time.Hour
	tier2MaxAge = 7 * 24 * time.Hour
	tier3MaxAge = 30 * 24 * time.Hour
	tier4MaxAge = 90 * 24 * time.Hour

	tier1RetryProb = 0.10 // 10% chance
	tier2RetryProb = 0.05 // 5% chance
	tier3RetryProb = 0.02 // 2% chance
	tier4RetryProb = 0.01 // 1% chance

	// Retry constraints
	minRetryInterval = 6 * time.Hour
	maxDailyRetries  = 20
	entryTTL         = 90 * 24 * time.Hour // 90 days
)

// FailedStructure represents a failed structure access attempt
type FailedStructure struct {
	StructureID   int64     `json:"structure_id"`
	CharacterID   int32     `json:"character_id"`
	FirstFailed   time.Time `json:"first_failed"`
	LastAttempted time.Time `json:"last_attempted"`
	FailureCount  int       `json:"failure_count"`
	LastError     string    `json:"last_error"`
	Tier          int       `json:"tier"`
}

// StructureAccessMetrics represents daily metrics for structure access
type StructureAccessMetrics struct {
	Date                   string  `json:"date"`
	TotalStructuresChecked int     `json:"total_structures_checked"`
	Failed403              int     `json:"failed_403"`
	RetryAttempts          int     `json:"retry_attempts"`
	RetrySuccesses         int     `json:"retry_successes"`
	AvgFailureAgeDays      float64 `json:"avg_failure_age_days"`
}

// StructureAccessTracker manages failed structure access tracking and retry logic
type StructureAccessTracker struct {
	redis *redis.Client
}

// NewStructureAccessTracker creates a new structure access tracker
func NewStructureAccessTracker(redis *redis.Client) *StructureAccessTracker {
	return &StructureAccessTracker{
		redis: redis,
	}
}

// RecordFailedAccess records a failed structure access attempt
func (t *StructureAccessTracker) RecordFailedAccess(ctx context.Context, characterID int32, structureID int64, errorMsg string) error {
	key := fmt.Sprintf("%s%d:%d", failedStructurePrefix, characterID, structureID)

	// Try to get existing record
	data, err := t.redis.Get(ctx, key).Result()
	var failed FailedStructure

	if err == redis.Nil {
		// New failure
		failed = FailedStructure{
			StructureID:   structureID,
			CharacterID:   characterID,
			FirstFailed:   time.Now(),
			LastAttempted: time.Now(),
			FailureCount:  1,
			LastError:     errorMsg,
			Tier:          1,
		}
	} else if err != nil {
		return fmt.Errorf("failed to get existing record: %w", err)
	} else {
		// Existing failure - update it
		if err := json.Unmarshal([]byte(data), &failed); err != nil {
			return fmt.Errorf("failed to unmarshal existing record: %w", err)
		}
		failed.LastAttempted = time.Now()
		failed.FailureCount++
		failed.LastError = errorMsg
		failed.Tier = t.calculateTier(failed.FailureCount, time.Since(failed.FirstFailed))
	}

	// Save updated record
	jsonData, err := json.Marshal(failed)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	if err := t.redis.Set(ctx, key, jsonData, entryTTL).Err(); err != nil {
		return fmt.Errorf("failed to save record: %w", err)
	}

	// Add to retry candidates if not tier 5 (dead)
	if failed.Tier < 5 {
		nextRetry := time.Now().Add(minRetryInterval).Unix()
		member := fmt.Sprintf("%d:%d", characterID, structureID)
		if err := t.redis.ZAdd(ctx, retryCandidatesKey, redis.Z{
			Score:  float64(nextRetry),
			Member: member,
		}).Err(); err != nil {
			slog.WarnContext(ctx, "Failed to add to retry candidates",
				"character_id", characterID,
				"structure_id", structureID,
				"error", err)
		}
	}

	// Update daily metrics
	t.updateMetrics(ctx, "failed_403", 1)

	return nil
}

// RecordSuccessfulAccess removes a structure from failed tracking after successful access
func (t *StructureAccessTracker) RecordSuccessfulAccess(ctx context.Context, characterID int32, structureID int64) error {
	key := fmt.Sprintf("%s%d:%d", failedStructurePrefix, characterID, structureID)

	// Check if it was previously failed
	exists, err := t.redis.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}

	if exists > 0 {
		// Remove from failed tracking
		if err := t.redis.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}

		// Remove from retry candidates
		member := fmt.Sprintf("%d:%d", characterID, structureID)
		if err := t.redis.ZRem(ctx, retryCandidatesKey, member).Err(); err != nil {
			slog.WarnContext(ctx, "Failed to remove from retry candidates",
				"character_id", characterID,
				"structure_id", structureID,
				"error", err)
		}

		// Update metrics
		t.updateMetrics(ctx, "retry_successes", 1)

		slog.InfoContext(ctx, "Structure access recovered",
			"character_id", characterID,
			"structure_id", structureID)
	}

	return nil
}

// GetRetryStructures selects structures to retry based on tier and probability
func (t *StructureAccessTracker) GetRetryStructures(ctx context.Context, characterID int32, maxRetries int) ([]int64, error) {
	// Check daily error budget
	todayKey := fmt.Sprintf("%s%s", esiErrorBudgetPrefix, time.Now().Format("2006-01-02"))
	errorCount, _ := t.redis.Get(ctx, todayKey).Int()

	if errorCount >= maxDailyRetries {
		slog.WarnContext(ctx, "Daily ESI error budget exhausted",
			"error_count", errorCount,
			"max_allowed", maxDailyRetries)
		return nil, nil
	}

	remainingBudget := maxDailyRetries - errorCount
	if maxRetries > remainingBudget {
		maxRetries = remainingBudget
	}

	// Get candidates ready for retry
	now := time.Now().Unix()
	candidates, err := t.redis.ZRangeByScore(ctx, retryCandidatesKey, &redis.ZRangeBy{
		Min:   "0",
		Max:   fmt.Sprintf("%d", now),
		Count: int64(maxRetries * 5), // Get more than needed for filtering
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get retry candidates: %w", err)
	}

	var retryStructures []int64
	attemptedCount := 0

	for _, candidate := range candidates {
		if len(retryStructures) >= maxRetries {
			break
		}

		// Parse candidate
		var candCharID int32
		var structureID int64
		fmt.Sscanf(candidate, "%d:%d", &candCharID, &structureID)

		// Skip if not for this character
		if candCharID != characterID {
			continue
		}

		// Get failure details
		key := fmt.Sprintf("%s%d:%d", failedStructurePrefix, characterID, structureID)
		data, err := t.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var failed FailedStructure
		if err := json.Unmarshal([]byte(data), &failed); err != nil {
			continue
		}

		// Check if enough time has passed since last attempt
		if time.Since(failed.LastAttempted) < minRetryInterval {
			continue
		}

		// Apply tier-based probability
		if t.shouldRetry(failed.Tier) {
			retryStructures = append(retryStructures, structureID)
			attemptedCount++

			// Update last attempted time
			failed.LastAttempted = time.Now()
			jsonData, _ := json.Marshal(failed)
			t.redis.Set(ctx, key, jsonData, entryTTL)

			// Update next retry time
			nextRetry := time.Now().Add(minRetryInterval).Unix()
			member := fmt.Sprintf("%d:%d", characterID, structureID)
			t.redis.ZAdd(ctx, retryCandidatesKey, redis.Z{
				Score:  float64(nextRetry),
				Member: member,
			})
		}
	}

	if len(retryStructures) > 0 {
		slog.InfoContext(ctx, "Selected structures for retry",
			"character_id", characterID,
			"retry_count", len(retryStructures),
			"candidates_checked", attemptedCount)

		t.updateMetrics(ctx, "retry_attempts", len(retryStructures))
	}

	return retryStructures, nil
}

// GetFailedStructureStats returns statistics about failed structure access
func (t *StructureAccessTracker) GetFailedStructureStats(ctx context.Context, characterID int32) (map[string]interface{}, error) {
	pattern := fmt.Sprintf("%s%d:*", failedStructurePrefix, characterID)
	keys, err := t.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	stats := map[string]interface{}{
		"total_failed": len(keys),
		"by_tier": map[int]int{
			1: 0, 2: 0, 3: 0, 4: 0, 5: 0,
		},
		"oldest_failure": time.Time{},
		"newest_failure": time.Time{},
	}

	tierCounts := stats["by_tier"].(map[int]int)
	var oldestFailure, newestFailure time.Time
	var totalAge time.Duration

	for _, key := range keys {
		data, err := t.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var failed FailedStructure
		if err := json.Unmarshal([]byte(data), &failed); err != nil {
			continue
		}

		tierCounts[failed.Tier]++
		totalAge += time.Since(failed.FirstFailed)

		if oldestFailure.IsZero() || failed.FirstFailed.Before(oldestFailure) {
			oldestFailure = failed.FirstFailed
		}
		if newestFailure.IsZero() || failed.FirstFailed.After(newestFailure) {
			newestFailure = failed.FirstFailed
		}
	}

	if len(keys) > 0 {
		stats["oldest_failure"] = oldestFailure
		stats["newest_failure"] = newestFailure
		stats["avg_failure_age_days"] = totalAge.Hours() / 24 / float64(len(keys))
	}

	return stats, nil
}

// calculateTier determines the retry tier based on failure count and age
func (t *StructureAccessTracker) calculateTier(failureCount int, age time.Duration) int {
	// Tier 5: Dead (never retry)
	if failureCount > tier4MaxFailures || age > tier4MaxAge {
		return 5
	}

	// Tier 4: Stale
	if failureCount > tier3MaxFailures || age > tier3MaxAge {
		return 4
	}

	// Tier 3: Old
	if failureCount > tier2MaxFailures || age > tier2MaxAge {
		return 3
	}

	// Tier 2: Medium
	if failureCount > tier1MaxFailures || age > tier1MaxAge {
		return 2
	}

	// Tier 1: Recent
	return 1
}

// shouldRetry determines if a structure should be retried based on tier probability
func (t *StructureAccessTracker) shouldRetry(tier int) bool {
	r := rand.Float64()

	switch tier {
	case 1:
		return r < tier1RetryProb
	case 2:
		return r < tier2RetryProb
	case 3:
		return r < tier3RetryProb
	case 4:
		return r < tier4RetryProb
	default:
		return false // Tier 5 or unknown - never retry
	}
}

// updateMetrics updates daily metrics
func (t *StructureAccessTracker) updateMetrics(ctx context.Context, metric string, increment int) {
	date := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("%s%s", metricsPrefix, date)

	// Get current metrics
	data, err := t.redis.Get(ctx, key).Result()
	var metrics StructureAccessMetrics

	if err == redis.Nil {
		metrics = StructureAccessMetrics{
			Date: date,
		}
	} else if err == nil {
		json.Unmarshal([]byte(data), &metrics)
	}

	// Update specific metric
	switch metric {
	case "total_structures_checked":
		metrics.TotalStructuresChecked += increment
	case "failed_403":
		metrics.Failed403 += increment
	case "retry_attempts":
		metrics.RetryAttempts += increment
	case "retry_successes":
		metrics.RetrySuccesses += increment
	}

	// Save updated metrics
	jsonData, _ := json.Marshal(metrics)
	t.redis.Set(ctx, key, jsonData, 48*time.Hour) // Keep for 2 days
}

// IncrementESIErrors increments the daily ESI error count
func (t *StructureAccessTracker) IncrementESIErrors(ctx context.Context, count int) error {
	todayKey := fmt.Sprintf("%s%s", esiErrorBudgetPrefix, time.Now().Format("2006-01-02"))
	return t.redis.IncrBy(ctx, todayKey, int64(count)).Err()
}

// GetRemainingErrorBudget returns the remaining ESI error budget for today
func (t *StructureAccessTracker) GetRemainingErrorBudget(ctx context.Context) int {
	todayKey := fmt.Sprintf("%s%s", esiErrorBudgetPrefix, time.Now().Format("2006-01-02"))
	errorCount, _ := t.redis.Get(ctx, todayKey).Int()

	remaining := maxDailyRetries - errorCount
	if remaining < 0 {
		return 0
	}
	return remaining
}
