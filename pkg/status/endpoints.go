package status

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// GetBackendStatusInput represents the input for getting backend status
type GetBackendStatusInput struct{}

// GetBackendStatusOutput represents the response for backend status
type GetBackendStatusOutput struct {
	Body BackendStatus `json:",inline"`
}

// BroadcastStatusInput represents the input for manual status broadcasting
type BroadcastStatusInput struct{}

// BroadcastStatusOutput represents the response for manual broadcast
type BroadcastStatusOutput struct {
	Body struct {
		Success   bool      `json:"success"`
		Message   string    `json:"message"`
		Timestamp time.Time `json:"timestamp"`
	} `json:",inline"`
}

// GetModuleStatusInput represents the input for getting individual module status
type GetModuleStatusInput struct{}

// GetModuleStatusOutput represents the response for module status
type GetModuleStatusOutput struct {
	Body struct {
		Services map[string]ModuleStatus `json:"services"`
		Count    int                     `json:"count"`
	} `json:",inline"`
}

// GetStatusConfigInput represents the input for getting status configuration
type GetStatusConfigInput struct{}

// GetStatusConfigOutput represents the response for status configuration
type GetStatusConfigOutput struct {
	Body struct {
		Config  Config `json:"config"`
		Running bool   `json:"running"`
		Uptime  string `json:"uptime,omitempty"`
	} `json:",inline"`
}

// UpdateStatusConfigInput represents the input for updating status configuration
type UpdateStatusConfigInput struct {
	Body Config `json:",inline"`
}

// UpdateStatusConfigOutput represents the response for config update
type UpdateStatusConfigOutput struct {
	Body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Config  Config `json:"config"`
	} `json:",inline"`
}

// RegisterStatusEndpoints registers all status-related endpoints
func RegisterStatusEndpoints(api huma.API, service *Service) {
	// GET /status/backend - Get current backend status
	huma.Register(api, huma.Operation{
		OperationID: "get-backend-status",
		Summary:     "Get Backend Status",
		Description: "Get the current aggregated status of all backend services and system metrics",
		Method:      http.MethodGet,
		Path:        "/status/backend",
		Tags:        []string{"Backend Status"},
	}, func(ctx context.Context, input *GetBackendStatusInput) (*GetBackendStatusOutput, error) {
		status, err := service.GetCurrentStatus(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get backend status", err)
		}

		return &GetBackendStatusOutput{
			Body: *status,
		}, nil
	})

	// GET /status/modules - Get individual module status
	huma.Register(api, huma.Operation{
		OperationID: "get-module-status",
		Summary:     "Get Module Status",
		Description: "Get the status of individual backend modules",
		Method:      http.MethodGet,
		Path:        "/status/modules",
		Tags:        []string{"Backend Status"},
	}, func(ctx context.Context, input *GetModuleStatusInput) (*GetModuleStatusOutput, error) {
		status, err := service.GetCurrentStatus(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get module status", err)
		}

		return &GetModuleStatusOutput{
			Body: struct {
				Services map[string]ModuleStatus `json:"services"`
				Count    int                     `json:"count"`
			}{
				Services: status.Services,
				Count:    len(status.Services),
			},
		}, nil
	})

	// POST /status/broadcast - Manual status broadcast (Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "broadcast-status",
		Summary:     "Broadcast Status",
		Description: "Manually broadcast current backend status to all connected WebSocket users (Admin only)",
		Method:      http.MethodPost,
		Path:        "/status/broadcast",
		Tags:        []string{"Backend Status", "Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *BroadcastStatusInput) (*BroadcastStatusOutput, error) {
		// TODO: Add admin permission check once we integrate with auth module

		err := service.BroadcastStatus(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to broadcast status", err)
		}

		return &BroadcastStatusOutput{
			Body: struct {
				Success   bool      `json:"success"`
				Message   string    `json:"message"`
				Timestamp time.Time `json:"timestamp"`
			}{
				Success:   true,
				Message:   "Status broadcasted successfully to all connected users",
				Timestamp: time.Now(),
			},
		}, nil
	})

	// GET /status/config - Get status service configuration (Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "get-status-config",
		Summary:     "Get Status Configuration",
		Description: "Get the current status service configuration and running state (Admin only)",
		Method:      http.MethodGet,
		Path:        "/status/config",
		Tags:        []string{"Backend Status", "Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *GetStatusConfigInput) (*GetStatusConfigOutput, error) {
		// TODO: Add admin permission check once we integrate with auth module

		// Get service uptime if running
		var uptime string
		if service.IsRunning() {
			// This is a simplified uptime - in real implementation, you'd track start time
			uptime = "Service is running"
		}

		return &GetStatusConfigOutput{
			Body: struct {
				Config  Config `json:"config"`
				Running bool   `json:"running"`
				Uptime  string `json:"uptime,omitempty"`
			}{
				Config:  service.config,
				Running: service.IsRunning(),
				Uptime:  uptime,
			},
		}, nil
	})

	// PUT /status/config - Update status service configuration (Admin only)
	huma.Register(api, huma.Operation{
		OperationID: "update-status-config",
		Summary:     "Update Status Configuration",
		Description: "Update the status service configuration (Admin only)",
		Method:      http.MethodPut,
		Path:        "/status/config",
		Tags:        []string{"Backend Status", "Admin"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
		},
	}, func(ctx context.Context, input *UpdateStatusConfigInput) (*UpdateStatusConfigOutput, error) {
		// TODO: Add admin permission check once we integrate with auth module

		// Update service configuration
		// Note: This would require extending the service to support config updates
		// For now, we'll return the current config

		return &UpdateStatusConfigOutput{
			Body: struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
				Config  Config `json:"config"`
			}{
				Success: false,
				Message: "Configuration updates not yet implemented - restart service with new environment variables",
				Config:  service.config,
			},
		}, nil
	})
}
