package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go-falcon/internal/zkillboard/dto"
	"go-falcon/internal/zkillboard/models"
)

// Helper functions for environment variables
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// ServiceState represents the state of the consumer service
type ServiceState int

const (
	StateStopped ServiceState = iota
	StateStarting
	StateRunning
	StateThrottled
	StateDraining
)

func (s ServiceState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateThrottled:
		return "throttled"
	case StateDraining:
		return "draining"
	default:
		return "unknown"
	}
}

// RedisQConsumer handles polling ZKillboard RedisQ for killmails
type RedisQConsumer struct {
	httpClient *http.Client
	processor  *KillmailProcessor
	repository *Repository

	// Configuration
	queueID       string
	endpoint      string
	ttw           int
	ttwMin        int
	ttwMax        int
	nullThreshold int

	// State management
	mu         sync.RWMutex
	state      atomic.Int32
	running    atomic.Bool
	lastPoll   time.Time
	nullStreak int
	startTime  time.Time
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// Metrics
	metrics ConsumerMetrics

	// Rate limiting
	rateLimiter *RateLimiter
}

// ConsumerMetrics tracks performance metrics
type ConsumerMetrics struct {
	TotalPolls     atomic.Int64
	NullResponses  atomic.Int64
	KillmailsFound atomic.Int64
	HTTPErrors     atomic.Int64
	ParseErrors    atomic.Int64
	StoreErrors    atomic.Int64
	RateLimitHits  atomic.Int64
	LastKillmailID atomic.Int64
}

// NewRedisQConsumer creates a new RedisQ consumer instance
func NewRedisQConsumer(processor *KillmailProcessor, repository *Repository) *RedisQConsumer {
	// Get configuration from environment
	queueID := os.Getenv("ZKB_QUEUE_ID")
	if queueID == "" {
		hostname, _ := os.Hostname()
		queueID = fmt.Sprintf("go-falcon-%s-%d", hostname, time.Now().Unix())
	}

	endpoint := os.Getenv("ZKB_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://zkillredisq.stream/listen.php"
	}

	ttwMin := getEnvAsInt("ZKB_TTW_MIN", 1)
	ttwMax := getEnvAsInt("ZKB_TTW_MAX", 10)
	nullThreshold := getEnvAsInt("ZKB_NULL_THRESHOLD", 5)
	httpTimeout := getEnvAsDuration("ZKB_HTTP_TIMEOUT", 30*time.Second)

	// Create HTTP client with timeout and redirect handling
	httpClient := &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	consumer := &RedisQConsumer{
		httpClient:    httpClient,
		processor:     processor,
		repository:    repository,
		queueID:       queueID,
		endpoint:      endpoint,
		ttw:           ttwMin,
		ttwMin:        ttwMin,
		ttwMax:        ttwMax,
		nullThreshold: nullThreshold,
		rateLimiter:   NewRateLimiter(),
	}

	consumer.state.Store(int32(StateStopped))
	return consumer
}

// Start begins the consumer polling loop
func (c *RedisQConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already running
	if c.running.Load() {
		return fmt.Errorf("consumer already running")
	}

	// Set state to starting
	c.state.Store(int32(StateStarting))

	// Create cancellable context
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Reset metrics for new session
	c.nullStreak = 0
	c.ttw = c.ttwMin
	c.startTime = time.Now()

	// Start polling loop
	c.wg.Add(1)
	go c.pollLoop()

	// Set state to running
	c.running.Store(true)
	c.state.Store(int32(StateRunning))

	// Persist state
	if err := c.repository.SaveConsumerState(c.ctx, c.getState()); err != nil {
		slog.Warn("Failed to save consumer state", "error", err)
	}

	slog.Info("RedisQ consumer started", "queue_id", c.queueID, "endpoint", c.endpoint)

	return nil
}

// Stop gracefully stops the consumer
func (c *RedisQConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running.Load() {
		return fmt.Errorf("consumer not running")
	}

	// Set state to draining
	c.state.Store(int32(StateDraining))

	slog.Info("Stopping RedisQ consumer...")

	// Cancel context
	if c.cancel != nil {
		c.cancel()
	}

	// Wait for polling loop to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		slog.Info("RedisQ consumer stopped gracefully")
	case <-time.After(30 * time.Second):
		slog.Warn("RedisQ consumer stop timeout")
	}

	// Flush any remaining killmails in the batch
	if err := c.processor.Flush(context.Background()); err != nil {
		slog.Error("Failed to flush pending killmails on stop", "error", err)
	} else {
		slog.Info("Flushed pending killmails on stop")
	}

	// Update state
	c.running.Store(false)
	c.state.Store(int32(StateStopped))

	// Save final state
	state := c.getState()
	now := time.Now()
	state.StoppedAt = &now
	if err := c.repository.SaveConsumerState(context.Background(), state); err != nil {
		slog.Warn("Failed to save final consumer state", "error", err)
	}

	return nil
}

// pollLoop is the main polling loop
func (c *RedisQConsumer) pollLoop() {
	defer c.wg.Done()

	slog.Info("Starting RedisQ poll loop")

	// Periodic state save ticker
	stateTicker := time.NewTicker(30 * time.Second)
	defer stateTicker.Stop()

	// Periodic batch flush ticker (every 3 seconds to ensure killmails are saved)
	flushTicker := time.NewTicker(3 * time.Second)
	defer flushTicker.Stop()

	// Flush any pending killmails when we exit
	defer func() {
		if err := c.processor.Flush(context.Background()); err != nil {
			slog.Error("Failed to flush pending killmails on exit", "error", err)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			slog.Info("Poll loop context cancelled")
			// Flush pending killmails before exiting
			if err := c.processor.Flush(context.Background()); err != nil {
				slog.Error("Failed to flush pending killmails", "error", err)
			}
			return

		case <-stateTicker.C:
			// Save current state periodically
			if err := c.repository.SaveConsumerState(c.ctx, c.getState()); err != nil {
				slog.Warn("Failed to save consumer state", "error", err)
			}

		case <-flushTicker.C:
			// Flush any pending killmails periodically
			if err := c.processor.Flush(c.ctx); err != nil {
				slog.Error("Failed to flush pending killmails", "error", err)
			}

		default:
			// Perform poll
			c.poll()
		}
	}
}

// poll performs a single RedisQ poll
func (c *RedisQConsumer) poll() {
	// Rate limiting
	if err := c.rateLimiter.Acquire(); err != nil {
		slog.Warn("Rate limit acquisition failed", "error", err)
		c.metrics.RateLimitHits.Add(1)
		c.state.Store(int32(StateThrottled))
		time.Sleep(5 * time.Second)
		c.state.Store(int32(StateRunning))
		return
	}
	defer c.rateLimiter.Release()

	// Calculate adaptive TTW
	ttw := c.calculateTTW()

	// Build request URL
	url := fmt.Sprintf("%s?queueID=%s&ttw=%d", c.endpoint, c.queueID, ttw)

	// Create request with context
	req, err := http.NewRequestWithContext(c.ctx, "GET", url, nil)
	if err != nil {
		slog.Error("Failed to create request", "error", err)
		c.metrics.HTTPErrors.Add(1)
		time.Sleep(5 * time.Second)
		return
	}

	// Add headers
	req.Header.Set("User-Agent", "go-falcon/1.0")
	req.Header.Set("Accept", "application/json")

	// Perform request
	c.metrics.TotalPolls.Add(1)
	c.lastPoll = time.Now()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("HTTP request failed", "error", err)
		c.metrics.HTTPErrors.Add(1)
		time.Sleep(5 * time.Second)
		return
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		slog.Warn("Rate limited by server")
		c.metrics.RateLimitHits.Add(1)
		c.state.Store(int32(StateThrottled))

		// Exponential backoff for rate limits
		backoffDuration := c.rateLimiter.GetBackoffDuration()
		slog.Info("Backing off due to rate limit", "backoff", backoffDuration)
		time.Sleep(backoffDuration)

		c.state.Store(int32(StateRunning))
		return
	}

	// Check for other HTTP errors
	if resp.StatusCode != http.StatusOK {
		slog.Error("Unexpected HTTP status", "status", resp.StatusCode)
		c.metrics.HTTPErrors.Add(1)
		time.Sleep(5 * time.Second)
		return
	}

	// Parse response
	var redisqResp dto.RedisQResponse
	if err := json.NewDecoder(resp.Body).Decode(&redisqResp); err != nil {
		slog.Error("Failed to decode response", "error", err)
		c.metrics.ParseErrors.Add(1)
		return
	}

	// Handle response
	c.processResponse(&redisqResp)
}

// processResponse handles the RedisQ response
func (c *RedisQConsumer) processResponse(resp *dto.RedisQResponse) {
	// Check for null package
	if resp.Package == nil {
		c.metrics.NullResponses.Add(1)
		c.mu.Lock()
		c.nullStreak++
		c.mu.Unlock()
		return
	}

	// Reset null streak on killmail received
	c.mu.Lock()
	c.nullStreak = 0
	c.ttw = c.ttwMin
	c.mu.Unlock()

	// Process the killmail
	c.metrics.KillmailsFound.Add(1)
	c.metrics.LastKillmailID.Store(resp.Package.KillID)

	// Process through pipeline
	if err := c.processor.ProcessKillmail(c.ctx, resp.Package); err != nil {
		slog.Error("Failed to process killmail", "error", err, "killmail_id", resp.Package.KillID)
		c.metrics.StoreErrors.Add(1)
		return
	}

	slog.Info("Killmail processed", "killmail_id", resp.Package.KillID, "value", resp.Package.ZKB.TotalValue, "solo", resp.Package.ZKB.Solo, "npc", resp.Package.ZKB.NPC)
}

// calculateTTW calculates adaptive time-to-wait
func (c *RedisQConsumer) calculateTTW() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Increase TTW after threshold of null responses
	if c.nullStreak >= c.nullThreshold {
		return c.ttwMax
	}

	// Use minimum TTW when active
	return c.ttwMin
}

// GetStatus returns the current service status
func (c *RedisQConsumer) GetStatus() *dto.ServiceStatusOutput {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var lastPoll *time.Time
	if !c.lastPoll.IsZero() {
		lastPoll = &c.lastPoll
	}

	var lastKillmail *int64
	if id := c.metrics.LastKillmailID.Load(); id > 0 {
		lastKillmail = &id
	}

	var uptime time.Duration
	if !c.startTime.IsZero() {
		uptime = time.Since(c.startTime)
	}

	return &dto.ServiceStatusOutput{
		Body: dto.ServiceStatusResponse{
			Status:       ServiceState(c.state.Load()).String(),
			QueueID:      c.queueID,
			LastPoll:     lastPoll,
			LastKillmail: lastKillmail,
			Metrics: dto.ServiceMetrics{
				TotalPolls:     c.metrics.TotalPolls.Load(),
				NullResponses:  c.metrics.NullResponses.Load(),
				KillmailsFound: c.metrics.KillmailsFound.Load(),
				HTTPErrors:     c.metrics.HTTPErrors.Load(),
				ParseErrors:    c.metrics.ParseErrors.Load(),
				StoreErrors:    c.metrics.StoreErrors.Load(),
				RateLimitHits:  c.metrics.RateLimitHits.Load(),
				CurrentTTW:     c.ttw,
				NullStreak:     c.nullStreak,
				Uptime:         uptime,
			},
			Config: dto.ServiceConfig{
				Endpoint:      c.endpoint,
				TTWMin:        c.ttwMin,
				TTWMax:        c.ttwMax,
				NullThreshold: c.nullThreshold,
				BatchSize:     getEnvAsInt("ZKB_BATCH_SIZE", 10),
			},
			Message: c.getStatusMessage(),
		},
	}
}

// getStatusMessage returns a descriptive status message
func (c *RedisQConsumer) getStatusMessage() string {
	state := ServiceState(c.state.Load())
	switch state {
	case StateRunning:
		return fmt.Sprintf("Consumer running, %d killmails processed", c.metrics.KillmailsFound.Load())
	case StateThrottled:
		return "Consumer throttled due to rate limiting"
	case StateDraining:
		return "Consumer draining, shutdown in progress"
	case StateStopped:
		return "Consumer stopped"
	default:
		return "Consumer in unknown state"
	}
}

// getState returns the current consumer state for persistence
func (c *RedisQConsumer) getState() *models.ConsumerState {
	return &models.ConsumerState{
		QueueID:        c.queueID,
		State:          ServiceState(c.state.Load()).String(),
		LastPollTime:   c.lastPoll,
		LastKillmailID: c.metrics.LastKillmailID.Load(),
		TotalPolls:     c.metrics.TotalPolls.Load(),
		NullResponses:  c.metrics.NullResponses.Load(),
		KillmailsFound: c.metrics.KillmailsFound.Load(),
		HTTPErrors:     c.metrics.HTTPErrors.Load(),
		ParseErrors:    c.metrics.ParseErrors.Load(),
		StoreErrors:    c.metrics.StoreErrors.Load(),
		RateLimitHits:  c.metrics.RateLimitHits.Load(),
		CurrentTTW:     c.ttw,
		NullStreak:     c.nullStreak,
		StartedAt:      c.startTime,
		UpdatedAt:      time.Now(),
	}
}
