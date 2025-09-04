package status

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Config holds configuration for the status service
type Config struct {
	BroadcastEnabled        bool
	BroadcastInterval       time.Duration
	ChangeDetectionInterval time.Duration
	CriticalAlertEnabled    bool
	AlertCPUThreshold       float64
	AlertMemoryThreshold    float64
	AlertErrorRateThreshold float64
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BroadcastEnabled:        true,
		BroadcastInterval:       30 * time.Second,
		ChangeDetectionInterval: 5 * time.Second,
		CriticalAlertEnabled:    true,
		AlertCPUThreshold:       90.0,
		AlertMemoryThreshold:    500.0, // MB
		AlertErrorRateThreshold: 5.0,   // percentage
	}
}

// Service coordinates status aggregation and broadcasting
type Service struct {
	config      Config
	aggregator  *Aggregator
	broadcaster *Broadcaster

	// Internal state
	running bool
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewService creates a new status service
func NewService(config Config, baseURL, apiPrefix string, moduleNames []string, websocket WebSocketBroadcaster) *Service {
	aggregator := NewAggregator(baseURL, apiPrefix, moduleNames)
	broadcaster := NewBroadcaster(websocket)

	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		config:      config,
		aggregator:  aggregator,
		broadcaster: broadcaster,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the status monitoring and broadcasting service
func (s *Service) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil // Already running
	}
	s.running = true
	s.mu.Unlock()

	slog.Info("Starting status service",
		"broadcast_enabled", s.config.BroadcastEnabled,
		"broadcast_interval", s.config.BroadcastInterval,
		"change_detection_interval", s.config.ChangeDetectionInterval)

	// Start periodic status broadcasting
	if s.config.BroadcastEnabled {
		s.wg.Add(1)
		go s.periodicBroadcasting()
	}

	// Start status change monitoring
	s.wg.Add(1)
	go s.statusChangeMonitoring()

	// Start critical alert monitoring
	if s.config.CriticalAlertEnabled {
		s.wg.Add(1)
		go s.criticalAlertMonitoring()
	}

	return nil
}

// Stop stops the status service
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return // Not running
	}
	s.running = false
	s.mu.Unlock()

	slog.Info("Stopping status service")
	s.cancel()
	s.wg.Wait()
	slog.Info("Status service stopped")
}

// GetCurrentStatus returns the current aggregated status
func (s *Service) GetCurrentStatus(ctx context.Context) (*BackendStatus, error) {
	return s.aggregator.AggregateStatus(ctx)
}

// GetLastStatus returns the last cached status
func (s *Service) GetLastStatus() *BackendStatus {
	return s.aggregator.GetLastStatus()
}

// BroadcastStatus manually broadcasts the current status
func (s *Service) BroadcastStatus(ctx context.Context) error {
	status, err := s.aggregator.AggregateStatus(ctx)
	if err != nil {
		return err
	}

	return s.broadcaster.BroadcastBackendStatus(ctx, status)
}

// periodicBroadcasting handles regular status broadcasting
func (s *Service) periodicBroadcasting() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.BroadcastInterval)
	defer ticker.Stop()

	slog.Info("Started periodic status broadcasting", "interval", s.config.BroadcastInterval)

	// Send initial status
	if err := s.BroadcastStatus(s.ctx); err != nil {
		slog.Error("Failed to broadcast initial status", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.BroadcastStatus(s.ctx); err != nil {
				slog.Error("Failed to broadcast periodic status", "error", err)
			} else {
				slog.Debug("Broadcasted periodic status update")
			}

		case <-s.ctx.Done():
			slog.Info("Periodic broadcasting stopped")
			return
		}
	}
}

// statusChangeMonitoring monitors for status changes and broadcasts them immediately
func (s *Service) statusChangeMonitoring() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.ChangeDetectionInterval)
	defer ticker.Stop()

	slog.Info("Started status change monitoring", "interval", s.config.ChangeDetectionInterval)

	var lastStatus *BackendStatus

	for {
		select {
		case <-ticker.C:
			currentStatus, err := s.aggregator.AggregateStatus(s.ctx)
			if err != nil {
				slog.Error("Failed to aggregate status for change detection", "error", err)
				continue
			}

			// Check for changes
			if s.aggregator.HasStatusChanged(currentStatus, lastStatus) {
				slog.Info("Status changes detected")

				// Get detailed changes
				changes := s.aggregator.GetStatusChanges(currentStatus, lastStatus)

				// Broadcast the changes
				if err := s.broadcaster.BroadcastStatusChanges(s.ctx, changes); err != nil {
					slog.Error("Failed to broadcast status changes", "error", err)
				}

				// Also broadcast the full status if overall status changed
				if lastStatus == nil || currentStatus.OverallStatus != lastStatus.OverallStatus {
					if err := s.broadcaster.BroadcastBackendStatus(s.ctx, currentStatus); err != nil {
						slog.Error("Failed to broadcast status update", "error", err)
					}
				}
			}

			lastStatus = currentStatus

		case <-s.ctx.Done():
			slog.Info("Status change monitoring stopped")
			return
		}
	}
}

// criticalAlertMonitoring monitors for critical system conditions
func (s *Service) criticalAlertMonitoring() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds for critical issues
	defer ticker.Stop()

	slog.Info("Started critical alert monitoring")

	var lastAlertStatus *BackendStatus

	for {
		select {
		case <-ticker.C:
			currentStatus, err := s.aggregator.AggregateStatus(s.ctx)
			if err != nil {
				slog.Error("Failed to aggregate status for critical monitoring", "error", err)
				continue
			}

			// Check for critical conditions
			alerts := s.detectCriticalAlerts(currentStatus, lastAlertStatus)

			// Broadcast critical alerts
			for _, alert := range alerts {
				if err := s.broadcaster.BroadcastCriticalAlert(s.ctx, alert); err != nil {
					slog.Error("Failed to broadcast critical alert", "error", err)
				}
			}

			lastAlertStatus = currentStatus

		case <-s.ctx.Done():
			slog.Info("Critical alert monitoring stopped")
			return
		}
	}
}

// detectCriticalAlerts identifies critical system conditions
func (s *Service) detectCriticalAlerts(current, previous *BackendStatus) []CriticalAlert {
	var alerts []CriticalAlert
	now := time.Now()

	// Check for service failures (healthy/degraded -> unhealthy)
	if previous != nil {
		for serviceName, currentService := range current.Services {
			if previousService, exists := previous.Services[serviceName]; exists {
				if previousService.Status != StatusUnhealthy && currentService.Status == StatusUnhealthy {
					alerts = append(alerts, CriticalAlert{
						Service:   serviceName,
						Severity:  SeverityFailure,
						Message:   fmt.Sprintf("%s service has failed: %s", serviceName, currentService.Message),
						Timestamp: now,
						Data: map[string]interface{}{
							"previous_status": previousService.Status,
							"current_status":  currentService.Status,
						},
					})
				}
			}
		}
	}

	// Check system-wide critical conditions
	if current.OverallStatus == StatusUnhealthy && (previous == nil || previous.OverallStatus != StatusUnhealthy) {
		alerts = append(alerts, CriticalAlert{
			Service:   "system",
			Severity:  SeverityFailure,
			Message:   "Overall system status is now unhealthy",
			Timestamp: now,
			Data: map[string]interface{}{
				"services_count": len(current.Services),
				"alerts_count":   len(current.Alerts),
			},
		})
	}

	// Check memory threshold
	if current.SystemMetrics.MemoryUsage > s.config.AlertMemoryThreshold {
		if previous == nil || previous.SystemMetrics.MemoryUsage <= s.config.AlertMemoryThreshold {
			alerts = append(alerts, CriticalAlert{
				Service:   "system",
				Severity:  SeverityCritical,
				Message:   fmt.Sprintf("High memory usage detected: %.1fMB", current.SystemMetrics.MemoryUsage),
				Timestamp: now,
				Data: map[string]interface{}{
					"memory_usage_mb": current.SystemMetrics.MemoryUsage,
					"threshold_mb":    s.config.AlertMemoryThreshold,
				},
			})
		}
	}

	// Check for too many alerts
	if len(current.Alerts) > 3 && (previous == nil || len(previous.Alerts) <= 3) {
		alerts = append(alerts, CriticalAlert{
			Service:   "system",
			Severity:  SeverityWarning,
			Message:   fmt.Sprintf("Multiple system alerts detected (%d alerts)", len(current.Alerts)),
			Timestamp: now,
			Data: map[string]interface{}{
				"alerts": current.Alerts,
			},
		})
	}

	return alerts
}

// IsRunning returns whether the service is currently running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
