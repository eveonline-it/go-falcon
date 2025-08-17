package scheduler

import (
	"testing"
	"time"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/internal/scheduler/models"
	"go-falcon/internal/scheduler/services"

	"github.com/go-playground/validator/v10"
)

// TestDTOValidation tests the DTO validation functionality
func TestDTOValidation(t *testing.T) {
	validate := validator.New()
	dto.RegisterCustomValidators(validate)

	tests := []struct {
		name    string
		dto     interface{}
		wantErr bool
	}{
		{
			name: "valid task create request",
			dto: &dto.TaskCreateRequest{
				Name:        "Test Task",
				Description: "A test task",
				Type:        models.TaskTypeHTTP,
				Schedule:    "0 */5 * * * *", // Every 5 minutes
				Priority:    models.TaskPriorityNormal,
				Enabled:     true,
				Config: map[string]interface{}{
					"url":    "https://example.com",
					"method": "GET",
				},
				Tags: []string{"test", "http"},
			},
			wantErr: false,
		},
		{
			name: "invalid task create request - missing name",
			dto: &dto.TaskCreateRequest{
				Type:     models.TaskTypeHTTP,
				Schedule: "0 */5 * * * *",
				Config: map[string]interface{}{
					"url":    "https://example.com",
					"method": "GET",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid task create request - invalid cron",
			dto: &dto.TaskCreateRequest{
				Name:     "Test Task",
				Type:     models.TaskTypeHTTP,
				Schedule: "invalid cron",
				Config: map[string]interface{}{
					"url":    "https://example.com",
					"method": "GET",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.dto)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestModelCreation tests model creation and validation
func TestModelCreation(t *testing.T) {
	task := &models.Task{
		ID:          "test-task-id",
		Name:        "Test Task",
		Description: "A test task for validation",
		Type:        models.TaskTypeHTTP,
		Schedule:    "0 */5 * * * *",
		Status:      models.TaskStatusPending,
		Priority:    models.TaskPriorityNormal,
		Enabled:     true,
		Config: map[string]interface{}{
			"url":    "https://example.com",
			"method": "GET",
		},
		Metadata: models.TaskMetadata{
			MaxRetries:    3,
			RetryInterval: 2 * time.Minute,
			Timeout:       5 * time.Minute,
			Tags:          []string{"test", "http"},
			IsSystem:      false,
			Source:        "api",
			Version:       1,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "test",
	}

	if task.ID == "" {
		t.Error("Task ID should not be empty")
	}

	if task.Name == "" {
		t.Error("Task name should not be empty")
	}

	if task.Type == "" {
		t.Error("Task type should not be empty")
	}

	if task.Schedule == "" {
		t.Error("Task schedule should not be empty")
	}

	if len(task.Config) == 0 {
		t.Error("Task config should not be empty")
	}
}

// TestServiceIntegration tests that services can be created without panics
func TestServiceIntegration(t *testing.T) {
	// This test ensures our service constructors work correctly
	// In a real environment, these would need actual database connections

	t.Run("system tasks creation", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("GetSystemTasks panicked: %v", r)
			}
		}()
		
		// Test that system tasks can be created without database
		// This validates the structure and configuration
		systemTasks := services.GetSystemTasks()
		if len(systemTasks) == 0 {
			t.Error("System tasks should not be empty")
		}

		// Validate each system task
		for _, task := range systemTasks {
			if task.ID == "" {
				t.Error("System task ID should not be empty")
			}
			if task.Name == "" {
				t.Error("System task name should not be empty")
			}
			if task.Type == "" {
				t.Error("System task type should not be empty")
			}
			if !task.Metadata.IsSystem {
				t.Error("System task should have IsSystem=true")
			}
		}
	})

	t.Run("task priority validation", func(t *testing.T) {
		validPriorities := []models.TaskPriority{
			models.TaskPriorityLow,
			models.TaskPriorityNormal,
			models.TaskPriorityHigh,
			models.TaskPriorityCritical,
		}

		for _, priority := range validPriorities {
			if string(priority) == "" {
				t.Errorf("Priority %v should have a string value", priority)
			}
		}
	})

	t.Run("task status validation", func(t *testing.T) {
		validStatuses := []models.TaskStatus{
			models.TaskStatusPending,
			models.TaskStatusRunning,
			models.TaskStatusCompleted,
			models.TaskStatusFailed,
			models.TaskStatusPaused,
			models.TaskStatusDisabled,
		}

		for _, status := range validStatuses {
			if string(status) == "" {
				t.Errorf("Status %v should have a string value", status)
			}
		}
	})
}

// TestDTOConversion tests DTO to model conversion patterns
func TestDTOConversion(t *testing.T) {
	req := &dto.TaskCreateRequest{
		Name:        "Test Task",
		Description: "A test task",
		Type:        models.TaskTypeHTTP,
		Schedule:    "0 */5 * * * *",
		Priority:    models.TaskPriorityNormal,
		Enabled:     true,
		Config: map[string]interface{}{
			"url":    "https://example.com",
			"method": "GET",
		},
		Tags: []string{"test", "http"},
	}

	if req.Name == "" {
		t.Error("Request name should not be empty")
	}

	if req.Type == "" {
		t.Error("Request type should not be empty")
	}

	resp := &dto.TaskResponse{
		ID:          "test-task-id",
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Schedule:    req.Schedule,
		Status:      models.TaskStatusPending,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		Config:      req.Config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if resp.ID == "" {
		t.Error("Response ID should not be empty")
	}

	if resp.Name != req.Name {
		t.Error("Response name should match request name")
	}
}