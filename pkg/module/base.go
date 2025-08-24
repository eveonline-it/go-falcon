package module

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
)

// Info represents module information
type Info struct {
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Description string           `json:"description"`
	Author      string           `json:"author"`
	Endpoints   []EndpointInfo   `json:"endpoints"`
	Permissions []PermissionInfo `json:"permissions"`
}

// EndpointInfo represents endpoint information
type EndpointInfo struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Description string `json:"description"`
	Permission  string `json:"permission"`
}

// PermissionInfo represents permission information
type PermissionInfo struct {
	Service     string `json:"service"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

// HealthStatus represents module health status
type HealthStatus struct {
	Status  Status `json:"status"`
	Message string `json:"message"`
}

// Status represents health status values
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Module defines the interface that all application modules must implement
type Module interface {
	// Routes sets up the HTTP routes for this module
	Routes(r chi.Router)

	// StartBackgroundTasks starts any background processing for this module
	StartBackgroundTasks(ctx context.Context)

	// Stop gracefully stops the module and its background tasks
	Stop()

	// Name returns the module name for logging and identification
	Name() string
}

// BaseModule provides common functionality for all modules
type BaseModule struct {
	name     string
	mongodb  *database.MongoDB
	redis    *database.Redis
	stopCh   chan struct{}
	stopOnce chan struct{} // Ensures Stop() can only be called once
}

// NewBaseModule creates a new base module with common dependencies
func NewBaseModule(name string, mongodb *database.MongoDB, redis *database.Redis) *BaseModule {
	return &BaseModule{
		name:     name,
		mongodb:  mongodb,
		redis:    redis,
		stopCh:   make(chan struct{}),
		stopOnce: make(chan struct{}),
	}
}

// Name returns the module name
func (b *BaseModule) Name() string {
	return b.name
}

// MongoDB returns the MongoDB connection
func (b *BaseModule) MongoDB() *database.MongoDB {
	return b.mongodb
}

// Redis returns the Redis connection
func (b *BaseModule) Redis() *database.Redis {
	return b.redis
}

// StopChannel returns the stop channel for background tasks
func (b *BaseModule) StopChannel() <-chan struct{} {
	return b.stopCh
}

// Stop gracefully stops the module
func (b *BaseModule) Stop() {
	select {
	case <-b.stopOnce:
		return // Already stopped
	default:
		close(b.stopOnce)
		close(b.stopCh)
		slog.Info("Module stopped", "module", b.name)
	}
}

// StartBackgroundTasks provides a default implementation for background tasks
func (b *BaseModule) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting background tasks", "module", b.name)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Background tasks context cancelled", "module", b.name)
			return
		case <-b.stopCh:
			slog.Info("Background tasks stopped", "module", b.name)
			return
		case <-ticker.C:
			// Modules can override this method to implement specific background work
		}
	}
}

// HealthHandler creates a health check handler for this module
func (b *BaseModule) HealthHandler() http.HandlerFunc {
	return handlers.HealthHandler(b.name)
}

// RegisterHealthRoute registers the health endpoint for this module
func (b *BaseModule) RegisterHealthRoute(r chi.Router) {
	r.Get("/health", b.HealthHandler())
}
