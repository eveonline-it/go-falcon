package dto

import (
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation rules for SDE DTOs
func RegisterCustomValidators(validate *validator.Validate) {
	validate.RegisterValidation("sde_entity_type", validateSDEEntityType)
	validate.RegisterValidation("sde_entity_id", validateSDEEntityID)
	validate.RegisterValidation("sde_index_type", validateSDEIndexType)
	validate.RegisterValidation("sde_notification_type", validateSDENotificationType)
	validate.RegisterValidation("duration_min", validateDurationMin)
}

// validateSDEEntityType validates SDE entity types
func validateSDEEntityType(fl validator.FieldLevel) bool {
	entityType := fl.Field().String()
	
	// List of valid SDE entity types
	validTypes := map[string]bool{
		"agents":            true,
		"blueprints":        true,
		"categories":        true,
		"certificates":      true,
		"characterAttributes": true,
		"contrabandTypes":   true,
		"controlTowerResources": true,
		"corporationActivities": true,
		"dogmaAttributes":   true,
		"dogmaEffects":      true,
		"factions":          true,
		"graphics":          true,
		"groups":            true,
		"iconIDs":           true,
		"marketGroups":      true,
		"metaGroups":        true,
		"npcCorporations":   true,
		"planetSchematics":  true,
		"races":             true,
		"regions":           true,
		"skinLicenses":      true,
		"skinMaterials":     true,
		"skins":             true,
		"soundIDs":          true,
		"stationOperations": true,
		"stationServices":   true,
		"tournamentRuleSets": true,
		"types":             true,
		"typeDogma":         true,
		"typeMaterials":     true,
		"universe":          true,
		// Universe-specific types
		"solarsystems":      true,
		"constellations":    true,
		"planets":           true,
		"moons":             true,
		"stargates":         true,
		"stations":          true,
	}
	
	return validTypes[entityType]
}

// validateSDEEntityID validates SDE entity IDs
func validateSDEEntityID(fl validator.FieldLevel) bool {
	id := fl.Field().String()
	
	// Basic validation - not empty, reasonable length
	if len(id) == 0 || len(id) > 100 {
		return false
	}
	
	// Allow alphanumeric, hyphens, underscores, and colons (for universe keys)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-:]+$`, id)
	return matched
}

// validateSDEIndexType validates SDE index types
func validateSDEIndexType(fl validator.FieldLevel) bool {
	indexType := fl.Field().String()
	
	validIndexTypes := map[string]bool{
		"solarsystems":    true,
		"regions":         true,
		"constellations":  true,
		"types":           true,
		"agents":          true,
		"corporations":    true,
	}
	
	return validIndexTypes[indexType]
}

// validateSDENotificationType validates SDE notification types
func validateSDENotificationType(fl validator.FieldLevel) bool {
	notificationType := fl.Field().String()
	
	validTypes := map[string]bool{
		"update_available": true,
		"update_started":   true,
		"update_completed": true,
		"update_failed":    true,
		"index_rebuilt":    true,
		"maintenance":      true,
	}
	
	return validTypes[notificationType]
}

// validateDurationMin validates minimum duration
func validateDurationMin(fl validator.FieldLevel) bool {
	duration := fl.Field().Interface().(time.Duration)
	param := fl.Param()
	
	minDuration, err := time.ParseDuration(param)
	if err != nil {
		return false
	}
	
	return duration >= minDuration
}

// ValidateEntityIdentifier validates an entity identifier
func ValidateEntityIdentifier(identifier EntityIdentifier) error {
	validate := validator.New()
	RegisterCustomValidators(validate)
	
	return validate.Struct(identifier)
}

// ValidateSearchQuery validates a search query
func ValidateSearchQuery(query string) bool {
	// Must not be empty
	if strings.TrimSpace(query) == "" {
		return false
	}
	
	// Must be reasonable length
	if len(query) > 100 {
		return false
	}
	
	// Must contain valid characters (alphanumeric, spaces, hyphens)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s\-']+$`, query)
	return matched
}

// ValidateRedisKey validates a Redis key format
func ValidateRedisKey(key string) bool {
	// Basic validation for SDE Redis keys
	if !strings.HasPrefix(key, "sde:") {
		return false
	}
	
	// Must be reasonable length
	if len(key) > 200 {
		return false
	}
	
	// Allow alphanumeric, colons, hyphens, underscores
	matched, _ := regexp.MatchString(`^sde:[a-zA-Z0-9_\-:]+$`, key)
	return matched
}

// ValidatePageSize validates pagination page size
func ValidatePageSize(pageSize int) bool {
	return pageSize > 0 && pageSize <= 1000
}

// ValidatePage validates pagination page number
func ValidatePage(page int) bool {
	return page > 0
}

// ValidateTimeRange validates that start time is before end time
func ValidateTimeRange(startTime, endTime *time.Time) bool {
	if startTime == nil || endTime == nil {
		return true // Optional parameters
	}
	
	return startTime.Before(*endTime)
}