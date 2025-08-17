package dto

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/robfig/cron/v3"
)

// RegisterCustomValidators registers custom validation rules for scheduler DTOs
func RegisterCustomValidators(validate *validator.Validate) {
	validate.RegisterValidation("cron", validateCronExpression)
	validate.RegisterValidation("task_type", validateTaskType)
	validate.RegisterValidation("task_priority", validateTaskPriority)
	validate.RegisterValidation("task_status", validateTaskStatus)
}

// validateCronExpression validates a cron schedule expression
func validateCronExpression(fl validator.FieldLevel) bool {
	schedule := fl.Field().String()
	if schedule == "" {
		return false
	}

	// Validate 6-field cron expression (with seconds)
	parts := strings.Fields(schedule)
	if len(parts) != 6 {
		return false
	}
	
	// Use cron library for proper validation
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(schedule)
	return err == nil
}

// validateTaskType validates task type values
func validateTaskType(fl validator.FieldLevel) bool {
	taskType := fl.Field().String()
	validTypes := []string{"http", "function", "system", "custom"}
	
	for _, validType := range validTypes {
		if taskType == validType {
			return true
		}
	}
	return false
}

// validateTaskPriority validates task priority values
func validateTaskPriority(fl validator.FieldLevel) bool {
	priority := fl.Field().String()
	validPriorities := []string{"low", "normal", "high", "critical"}
	
	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return true
		}
	}
	return false
}

// validateTaskStatus validates task status values
func validateTaskStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"pending", "running", "completed", "failed", "paused", "disabled"}
	
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}