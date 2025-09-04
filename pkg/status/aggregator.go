package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// Aggregator collects status information from all modules
type Aggregator struct {
	baseURL         string
	apiPrefix       string
	httpClient      *http.Client
	moduleNames     []string
	systemStartTime time.Time
	mu              sync.RWMutex
	lastStatus      *BackendStatus
}

// NewAggregator creates a new status aggregator
func NewAggregator(baseURL, apiPrefix string, moduleNames []string) *Aggregator {
	return &Aggregator{
		baseURL:         baseURL,
		apiPrefix:       apiPrefix,
		httpClient:      &http.Client{Timeout: 5 * time.Second},
		moduleNames:     moduleNames,
		systemStartTime: time.Now(),
	}
}

// AggregateStatus collects status from all modules and system metrics
func (a *Aggregator) AggregateStatus(ctx context.Context) (*BackendStatus, error) {
	status := &BackendStatus{
		Timestamp: time.Now(),
		Services:  make(map[string]ModuleStatus),
		Alerts:    []string{},
	}

	// Collect module statuses in parallel
	var wg sync.WaitGroup
	statusChan := make(chan ModuleStatus, len(a.moduleNames))

	for _, moduleName := range a.moduleNames {
		wg.Add(1)
		go func(module string) {
			defer wg.Done()
			if moduleStatus, err := a.getModuleStatus(ctx, module); err != nil {
				slog.Error("Failed to get module status", "module", module, "error", err)
				statusChan <- ModuleStatus{
					Module:      module,
					Status:      StatusUnhealthy,
					Message:     fmt.Sprintf("Failed to check status: %v", err),
					LastChecked: time.Now(),
				}
			} else {
				statusChan <- moduleStatus
			}
		}(moduleName)
	}

	// Wait for all status checks to complete
	go func() {
		wg.Wait()
		close(statusChan)
	}()

	// Collect results
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for moduleStatus := range statusChan {
		status.Services[moduleStatus.Module] = moduleStatus

		switch moduleStatus.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
			status.Alerts = append(status.Alerts, fmt.Sprintf("%s service is degraded: %s", moduleStatus.Module, moduleStatus.Message))
		case StatusUnhealthy:
			unhealthyCount++
			status.Alerts = append(status.Alerts, fmt.Sprintf("%s service is unhealthy: %s", moduleStatus.Module, moduleStatus.Message))
		}
	}

	// Calculate overall status
	status.OverallStatus = a.calculateOverallStatus(healthyCount, degradedCount, unhealthyCount)

	// Add system metrics
	status.SystemMetrics = a.getSystemMetrics()

	// Check for system-level alerts
	a.checkSystemAlerts(status)

	// Store for comparison
	a.mu.Lock()
	a.lastStatus = status
	a.mu.Unlock()

	return status, nil
}

// getModuleStatus retrieves status from a specific module
func (a *Aggregator) getModuleStatus(ctx context.Context, moduleName string) (ModuleStatus, error) {
	startTime := time.Now()

	url := fmt.Sprintf("%s%s/%s/status", a.baseURL, a.apiPrefix, moduleName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ModuleStatus{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return ModuleStatus{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(startTime)

	if resp.StatusCode != http.StatusOK {
		return ModuleStatus{}, fmt.Errorf("received status code: %d", resp.StatusCode)
	}

	var moduleResponse struct {
		Module  string                 `json:"module"`
		Status  string                 `json:"status"`
		Message string                 `json:"message,omitempty"`
		Stats   map[string]interface{} `json:"stats,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&moduleResponse); err != nil {
		return ModuleStatus{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert string status to Status type
	var status Status
	switch moduleResponse.Status {
	case "healthy":
		status = StatusHealthy
	case "degraded":
		status = StatusDegraded
	case "unhealthy":
		status = StatusUnhealthy
	default:
		status = StatusUnhealthy
	}

	return ModuleStatus{
		Module:       moduleResponse.Module,
		Status:       status,
		Message:      moduleResponse.Message,
		ResponseTime: fmt.Sprintf("%.2fms", float64(responseTime.Nanoseconds())/1e6),
		Stats:        moduleResponse.Stats,
		LastChecked:  time.Now(),
	}, nil
}

// calculateOverallStatus determines the overall system status
func (a *Aggregator) calculateOverallStatus(healthy, degraded, unhealthy int) Status {
	total := healthy + degraded + unhealthy

	if total == 0 {
		return StatusUnhealthy
	}

	// If any service is unhealthy, overall is unhealthy
	if unhealthy > 0 {
		return StatusUnhealthy
	}

	// If any service is degraded, overall is degraded
	if degraded > 0 {
		return StatusDegraded
	}

	// All services are healthy
	return StatusHealthy
}

// getSystemMetrics collects system-level metrics
func (a *Aggregator) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(a.systemStartTime)

	return SystemMetrics{
		CPUUsage:        0,                              // TODO: Implement CPU usage monitoring
		MemoryUsage:     float64(m.Alloc) / 1024 / 1024, // Convert to MB
		ActiveConns:     0,                              // TODO: Get from connection manager
		UptimeSeconds:   int64(uptime.Seconds()),
		UptimeFormatted: formatUptime(uptime),
	}
}

// checkSystemAlerts checks for system-level alerts
func (a *Aggregator) checkSystemAlerts(status *BackendStatus) {
	// Memory usage alert (if > 500MB)
	if status.SystemMetrics.MemoryUsage > 500 {
		status.Alerts = append(status.Alerts, fmt.Sprintf("High memory usage: %.1fMB", status.SystemMetrics.MemoryUsage))
	}

	// Check for too many unhealthy services
	unhealthyCount := 0
	for _, svc := range status.Services {
		if svc.Status == StatusUnhealthy {
			unhealthyCount++
		}
	}

	if unhealthyCount > len(status.Services)/2 {
		status.Alerts = append(status.Alerts, "More than half of services are unhealthy")
	}
}

// GetLastStatus returns the last aggregated status
func (a *Aggregator) GetLastStatus() *BackendStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastStatus
}

// HasStatusChanged compares current status with previous status
func (a *Aggregator) HasStatusChanged(current, previous *BackendStatus) bool {
	if previous == nil {
		return true
	}

	// Check overall status change
	if current.OverallStatus != previous.OverallStatus {
		return true
	}

	// Check individual service status changes
	for serviceName, currentService := range current.Services {
		if previousService, exists := previous.Services[serviceName]; !exists || currentService.Status != previousService.Status {
			return true
		}
	}

	// Check alert count changes
	if len(current.Alerts) != len(previous.Alerts) {
		return true
	}

	return false
}

// GetStatusChanges returns a list of status changes between current and previous status
func (a *Aggregator) GetStatusChanges(current, previous *BackendStatus) []StatusChange {
	var changes []StatusChange

	if previous == nil {
		return changes
	}

	// Check individual service status changes
	for serviceName, currentService := range current.Services {
		if previousService, exists := previous.Services[serviceName]; exists {
			if currentService.Status != previousService.Status {
				changes = append(changes, StatusChange{
					Service:   serviceName,
					OldStatus: previousService.Status,
					NewStatus: currentService.Status,
					Timestamp: time.Now(),
					Message:   fmt.Sprintf("%s changed from %s to %s", serviceName, previousService.Status, currentService.Status),
				})
			}
		}
	}

	return changes
}

// formatUptime formats duration into human readable string
func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), float64(d.Seconds())-(d.Minutes()*60))
	}

	hours := d.Hours()
	minutes := d.Minutes() - (hours * 60)
	return fmt.Sprintf("%.0fh %.0fm", hours, minutes)
}
