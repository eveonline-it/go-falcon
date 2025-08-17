package dto

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation rules for auth module
func RegisterCustomValidators(validate *validator.Validate) error {
	// Register EVE character name validator
	if err := validate.RegisterValidation("eve_character_name", validateEVECharacterName); err != nil {
		return fmt.Errorf("failed to register eve_character_name validator: %w", err)
	}

	// Register EVE scope validator
	if err := validate.RegisterValidation("eve_scopes", validateEVEScopes); err != nil {
		return fmt.Errorf("failed to register eve_scopes validator: %w", err)
	}

	return nil
}

// validateEVECharacterName validates EVE Online character names
func validateEVECharacterName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	// EVE character names: 3-37 characters, letters, numbers, spaces, some special chars
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s'\-\.]{3,37}$`, name)
	return matched
}

// validateEVEScopes validates EVE Online scopes format
func validateEVEScopes(fl validator.FieldLevel) bool {
	scopes := fl.Field().String()
	// EVE scopes are space-separated and follow esi-* pattern or publicData
	if scopes == "" || scopes == "publicData" {
		return true
	}
	// Basic validation for ESI scope format
	matched, _ := regexp.MatchString(`^(publicData|esi-[a-zA-Z0-9_\-\.]+)(\s+(publicData|esi-[a-zA-Z0-9_\-\.]+))*$`, scopes)
	return matched
}

// ValidateStruct validates a struct using the validator instance
func ValidateStruct(validate *validator.Validate, s interface{}) []string {
	var errors []string
	
	if err := validate.Struct(s); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, formatValidationError(err))
		}
	}
	
	return errors
}

// formatValidationError formats validation errors for user-friendly messages
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", err.Field(), err.Param())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", err.Field())
	case "eve_character_name":
		return fmt.Sprintf("%s must be a valid EVE Online character name", err.Field())
	case "eve_scopes":
		return fmt.Sprintf("%s must be valid EVE Online scopes", err.Field())
	default:
		return fmt.Sprintf("%s is invalid", err.Field())
	}
}