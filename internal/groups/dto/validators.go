package dto

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation rules for groups DTOs
func RegisterCustomValidators(validate *validator.Validate) {
	validate.RegisterValidation("alphanum", validateAlphaNum)
	validate.RegisterValidation("group_name", validateGroupName)
	validate.RegisterValidation("discord_server_id", validateDiscordServerID)
	validate.RegisterValidation("permission_action", validatePermissionAction)
	validate.RegisterValidation("subject_type", validateSubjectType)
	validate.RegisterValidation("service_name", validateServiceName)
}

// validateAlphaNum validates alphanumeric strings with underscores and hyphens
func validateAlphaNum(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	
	// Allow alphanumeric characters, underscores, hyphens, and spaces
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\s]+$`, value)
	return matched
}

// validateGroupName validates group names (alphanumeric with underscores, no spaces at start/end)
func validateGroupName(fl validator.FieldLevel) bool {
	value := strings.TrimSpace(fl.Field().String())
	if value == "" {
		return false
	}
	
	// Group names: alphanumeric, underscores, hyphens, no leading/trailing spaces
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-]+$`, value)
	return matched
}

// validateDiscordServerID validates Discord server IDs (numeric string)
func validateDiscordServerID(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	
	// Discord server IDs are numeric strings
	matched, _ := regexp.MatchString(`^\d{17,19}$`, value)
	return matched
}

// validatePermissionAction validates permission action values
func validatePermissionAction(fl validator.FieldLevel) bool {
	action := fl.Field().String()
	validActions := []string{"read", "write", "delete", "admin"}
	
	for _, validAction := range validActions {
		if action == validAction {
			return true
		}
	}
	return false
}

// validateSubjectType validates subject type values
func validateSubjectType(fl validator.FieldLevel) bool {
	subjectType := fl.Field().String()
	validTypes := []string{"group", "member", "corporation", "alliance"}
	
	for _, validType := range validTypes {
		if subjectType == validType {
			return true
		}
	}
	return false
}

// validateServiceName validates service names (lowercase alphanumeric with underscores)
func validateServiceName(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return false
	}
	
	// Service names: lowercase alphanumeric with underscores
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, value)
	return matched
}