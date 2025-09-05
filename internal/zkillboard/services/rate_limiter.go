package services

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter ensures compliance with ZKillboard rate limits
// - 1 concurrent request per queueID
// - 2 requests per second per IP
type RateLimiter struct {
	mu              sync.Mutex
	requestInFlight bool
	lastRequest     time.Time
	minInterval     time.Duration
	backoffLevel    int
	maxBackoffLevel int
	baseBackoff     time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		minInterval:     500 * time.Millisecond, // 2 requests per second
		baseBackoff:     5 * time.Second,
		maxBackoffLevel: 4, // Max 80 seconds (5 * 2^4)
	}
}

// Acquire blocks until it's safe to make a request
func (r *RateLimiter) Acquire() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure only 1 concurrent request
	if r.requestInFlight {
		return fmt.Errorf("request already in flight")
	}

	// Enforce minimum interval between requests
	elapsed := time.Since(r.lastRequest)
	if elapsed < r.minInterval {
		time.Sleep(r.minInterval - elapsed)
	}

	r.requestInFlight = true
	r.lastRequest = time.Now()

	// Reset backoff on successful acquire
	r.backoffLevel = 0

	return nil
}

// Release marks the request as complete
func (r *RateLimiter) Release() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestInFlight = false
}

// IncrementBackoff increases the backoff level for rate limit hits
func (r *RateLimiter) IncrementBackoff() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.backoffLevel < r.maxBackoffLevel {
		r.backoffLevel++
	}
}

// GetBackoffDuration returns the current backoff duration
func (r *RateLimiter) GetBackoffDuration() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Exponential backoff: 5s, 10s, 20s, 40s, 80s
	return r.baseBackoff * time.Duration(1<<r.backoffLevel)
}

// Reset clears all rate limit state
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requestInFlight = false
	r.backoffLevel = 0
	r.lastRequest = time.Time{}
}
