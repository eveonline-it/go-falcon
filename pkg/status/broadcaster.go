package status

import (
	"context"
	"log/slog"
)

// WebSocketBroadcaster interface for broadcasting status messages
type WebSocketBroadcaster interface {
	SendSystemMessage(ctx context.Context, message string, data map[string]interface{}) error
}

// Broadcaster handles broadcasting status updates via WebSocket
type Broadcaster struct {
	websocket WebSocketBroadcaster
}

// NewBroadcaster creates a new status broadcaster
func NewBroadcaster(websocket WebSocketBroadcaster) *Broadcaster {
	return &Broadcaster{
		websocket: websocket,
	}
}

// BroadcastBackendStatus broadcasts the current backend status to all connected users
func (b *Broadcaster) BroadcastBackendStatus(ctx context.Context, status *BackendStatus) error {
	data := map[string]interface{}{
		"type":           MessageTypeBackendStatus,
		"timestamp":      status.Timestamp,
		"overall_status": status.OverallStatus,
		"services":       status.Services,
		"system_metrics": status.SystemMetrics,
		"alerts":         status.Alerts,
	}

	message := "Backend status update"
	if status.OverallStatus != StatusHealthy {
		message = string("Backend status: " + status.OverallStatus)
	}

	if err := b.websocket.SendSystemMessage(ctx, message, data); err != nil {
		slog.Error("Failed to broadcast backend status", "error", err)
		return err
	}

	slog.Debug("Broadcasted backend status", "overall_status", status.OverallStatus, "services", len(status.Services))
	return nil
}

// BroadcastStatusChanges broadcasts status changes to all connected users
func (b *Broadcaster) BroadcastStatusChanges(ctx context.Context, changes []StatusChange) error {
	if len(changes) == 0 {
		return nil
	}

	for _, change := range changes {
		data := map[string]interface{}{
			"type":        MessageTypeBackendStatus,
			"change_type": "service_status_change",
			"service":     change.Service,
			"old_status":  change.OldStatus,
			"new_status":  change.NewStatus,
			"timestamp":   change.Timestamp,
		}

		message := change.Message

		// Determine if this is a recovery or degradation
		if change.NewStatus == StatusHealthy && change.OldStatus != StatusHealthy {
			data["type"] = MessageTypeServiceRecovery
			message = change.Service + " service has recovered"
		}

		if err := b.websocket.SendSystemMessage(ctx, message, data); err != nil {
			slog.Error("Failed to broadcast status change", "service", change.Service, "error", err)
			continue
		}

		slog.Info("Broadcasted status change",
			"service", change.Service,
			"old_status", change.OldStatus,
			"new_status", change.NewStatus)
	}

	return nil
}

// BroadcastCriticalAlert broadcasts a critical alert to all connected users
func (b *Broadcaster) BroadcastCriticalAlert(ctx context.Context, alert CriticalAlert) error {
	data := map[string]interface{}{
		"type":      MessageTypeCriticalAlert,
		"service":   alert.Service,
		"severity":  alert.Severity,
		"message":   alert.Message,
		"timestamp": alert.Timestamp,
	}

	if alert.Data != nil {
		data["details"] = alert.Data
	}

	message := alert.Message
	if alert.Severity == SeverityFailure {
		message = "CRITICAL: " + alert.Message
	}

	if err := b.websocket.SendSystemMessage(ctx, message, data); err != nil {
		slog.Error("Failed to broadcast critical alert", "service", alert.Service, "severity", alert.Severity, "error", err)
		return err
	}

	slog.Warn("Broadcasted critical alert",
		"service", alert.Service,
		"severity", alert.Severity,
		"message", alert.Message)

	return nil
}

// BroadcastServiceRecovery broadcasts a service recovery notification
func (b *Broadcaster) BroadcastServiceRecovery(ctx context.Context, serviceName string, message string) error {
	data := map[string]interface{}{
		"type":      MessageTypeServiceRecovery,
		"service":   serviceName,
		"message":   message,
		"timestamp": context.TODO(), // Will be set by the system
	}

	recoveryMessage := serviceName + " service has recovered"
	if message != "" {
		recoveryMessage = message
	}

	if err := b.websocket.SendSystemMessage(ctx, recoveryMessage, data); err != nil {
		slog.Error("Failed to broadcast service recovery", "service", serviceName, "error", err)
		return err
	}

	slog.Info("Broadcasted service recovery", "service", serviceName, "message", message)
	return nil
}
