package status

import (
	"time"
)

// Status represents the health status of a service
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ModuleStatus represents the status of an individual module
type ModuleStatus struct {
	Module       string                 `json:"module"`
	Status       Status                 `json:"status"`
	Message      string                 `json:"message,omitempty"`
	ResponseTime string                 `json:"response_time,omitempty"`
	Stats        map[string]interface{} `json:"stats,omitempty"`
	LastChecked  time.Time              `json:"last_checked"`
}

// SystemMetrics represents overall system health metrics
type SystemMetrics struct {
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	ActiveConns     int     `json:"active_connections"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
	UptimeFormatted string  `json:"uptime_formatted"`
}

// BackendStatus represents the complete backend status
type BackendStatus struct {
	Timestamp     time.Time               `json:"timestamp"`
	OverallStatus Status                  `json:"overall_status"`
	Services      map[string]ModuleStatus `json:"services"`
	SystemMetrics SystemMetrics           `json:"system_metrics"`
	Alerts        []string                `json:"alerts,omitempty"`
}

// StatusChange represents a detected status change
type StatusChange struct {
	Service   string    `json:"service"`
	OldStatus Status    `json:"old_status"`
	NewStatus Status    `json:"new_status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// CriticalAlert represents a critical system alert
type CriticalAlert struct {
	Service   string                 `json:"service"`
	Severity  string                 `json:"severity"` // warning, critical, failure
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Alert severity levels
const (
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
	SeverityFailure  = "failure"
)

// WebSocket message types for status broadcasting
const (
	MessageTypeBackendStatus   = "backend_status"
	MessageTypeCriticalAlert   = "critical_alert"
	MessageTypeServiceRecovery = "service_recovery"
)
