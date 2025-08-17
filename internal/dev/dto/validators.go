package dto

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation rules for Dev DTOs
func RegisterCustomValidators(validate *validator.Validate) {
	validate.RegisterValidation("esi_endpoint", validateESIEndpoint)
	validate.RegisterValidation("character_id", validateCharacterID)
	validate.RegisterValidation("alliance_id", validateAllianceID)
	validate.RegisterValidation("corporation_id", validateCorporationID)
	validate.RegisterValidation("system_id", validateSystemID)
	validate.RegisterValidation("type_id", validateTypeID)
	validate.RegisterValidation("universe_type", validateUniverseType)
	validate.RegisterValidation("sde_entity_type", validateSDEEntityType)
	validate.RegisterValidation("cache_key", validateCacheKey)
	validate.RegisterValidation("dev_component", validateDevComponent)
	validate.RegisterValidation("test_operation", validateTestOperation)
	validate.RegisterValidation("http_method", validateHTTPMethod)
}

// validateESIEndpoint validates ESI endpoint format
func validateESIEndpoint(fl validator.FieldLevel) bool {
	endpoint := fl.Field().String()
	
	// Basic validation - must start with /
	if !strings.HasPrefix(endpoint, "/") {
		return false
	}
	
	// Must not contain invalid characters
	matched, _ := regexp.MatchString(`^/[a-zA-Z0-9/_\-{}]+/?$`, endpoint)
	return matched
}

// validateCharacterID validates EVE character ID format
func validateCharacterID(fl validator.FieldLevel) bool {
	charID := fl.Field().Int()
	
	// Character IDs are typically 7-10 digits and positive
	return charID > 90000000 && charID <= 2147483647
}

// validateAllianceID validates EVE alliance ID format
func validateAllianceID(fl validator.FieldLevel) bool {
	allianceID := fl.Field().Int()
	
	// Alliance IDs are typically 8-10 digits and positive
	return allianceID > 99000000 && allianceID <= 2147483647
}

// validateCorporationID validates EVE corporation ID format
func validateCorporationID(fl validator.FieldLevel) bool {
	corpID := fl.Field().Int()
	
	// Corporation IDs are typically 7-10 digits and positive
	return corpID > 1000000 && corpID <= 2147483647
}

// validateSystemID validates EVE solar system ID format
func validateSystemID(fl validator.FieldLevel) bool {
	systemID := fl.Field().Int()
	
	// System IDs are typically 8 digits starting with 3
	return systemID >= 30000000 && systemID <= 33000000
}

// validateTypeID validates EVE type ID format
func validateTypeID(fl validator.FieldLevel) bool {
	typeID := fl.Field().Int()
	
	// Type IDs are positive integers, can be quite large
	return typeID > 0 && typeID <= 100000000
}

// validateUniverseType validates universe type values
func validateUniverseType(fl validator.FieldLevel) bool {
	universeType := fl.Field().String()
	
	validTypes := map[string]bool{
		"eve":      true,
		"abyssal":  true,
		"wormhole": true,
		"void":     true,
		"hidden":   true,
	}
	
	return validTypes[universeType]
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
		"solarsystems":      true,
		"constellations":    true,
		"planets":           true,
		"moons":             true,
		"stargates":         true,
		"stations":          true,
		"flags":             true,
	}
	
	return validTypes[entityType]
}

// validateCacheKey validates cache key format
func validateCacheKey(fl validator.FieldLevel) bool {
	key := fl.Field().String()
	
	// Must not be empty and reasonable length
	if len(key) == 0 || len(key) > 255 {
		return false
	}
	
	// Allow alphanumeric, colons, hyphens, underscores, dots, slashes
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-:./]+$`, key)
	return matched
}

// validateDevComponent validates development component names
func validateDevComponent(fl validator.FieldLevel) bool {
	component := fl.Field().String()
	
	validComponents := map[string]bool{
		"esi":         true,
		"sde":         true,
		"cache":       true,
		"auth":        true,
		"permissions": true,
		"database":    true,
		"redis":       true,
		"telemetry":   true,
		"logging":     true,
	}
	
	return validComponents[component]
}

// validateTestOperation validates test operation types
func validateTestOperation(fl validator.FieldLevel) bool {
	operation := fl.Field().String()
	
	validOperations := map[string]bool{
		"esi":        true,
		"sde":        true,
		"cache":      true,
		"validation": true,
		"performance": true,
		"health":     true,
		"mock":       true,
		"debug":      true,
	}
	
	return validOperations[operation]
}

// validateHTTPMethod validates HTTP method names
func validateHTTPMethod(fl validator.FieldLevel) bool {
	method := strings.ToUpper(fl.Field().String())
	
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	
	return validMethods[method]
}

// ValidateEVEID validates EVE Online entity IDs
func ValidateEVEID(id interface{}, entityType string) bool {
	var intID int
	
	switch v := id.(type) {
	case int:
		intID = v
	case string:
		var err error
		intID, err = strconv.Atoi(v)
		if err != nil {
			return false
		}
	default:
		return false
	}
	
	switch entityType {
	case "character":
		return intID > 90000000 && intID <= 2147483647
	case "alliance":
		return intID > 99000000 && intID <= 2147483647
	case "corporation":
		return intID > 1000000 && intID <= 2147483647
	case "system":
		return intID >= 30000000 && intID <= 33000000
	case "type":
		return intID > 0 && intID <= 100000000
	default:
		return intID > 0
	}
}

// ValidateTimeRange validates that start time is before end time
func ValidateTimeRange(startTime, endTime *time.Time) bool {
	if startTime == nil || endTime == nil {
		return true // Optional parameters
	}
	
	return startTime.Before(*endTime)
}

// ValidateAccessToken validates EVE Online access token format
func ValidateAccessToken(token string) bool {
	// Basic validation for JWT-like tokens
	if len(token) < 20 {
		return false
	}
	
	// Check for JWT format (three parts separated by dots)
	parts := strings.Split(token, ".")
	return len(parts) == 3
}

// ValidateESIScopes validates EVE Online ESI scopes
func ValidateESIScopes(scopes []string) bool {
	if len(scopes) == 0 {
		return true // Empty scopes are valid
	}
	
	// Basic validation - scopes should contain only valid characters
	for _, scope := range scopes {
		if len(scope) == 0 || len(scope) > 100 {
			return false
		}
		
		// ESI scopes format: esi-{service}.{action}_{version}
		matched, _ := regexp.MatchString(`^esi-[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+_v[0-9]+$`, scope)
		if !matched {
			return false
		}
	}
	
	return true
}

// ValidateUniversePath validates universe path components
func ValidateUniversePath(universeType, region, constellation, system string) bool {
	// Universe type is required
	if !ValidateUniverseType(universeType) {
		return false
	}
	
	// Validate path components if provided
	if region != "" && !ValidatePathComponent(region) {
		return false
	}
	
	if constellation != "" && !ValidatePathComponent(constellation) {
		return false
	}
	
	if system != "" && !ValidatePathComponent(system) {
		return false
	}
	
	return true
}

// ValidateUniverseType validates universe type values
func ValidateUniverseType(universeType string) bool {
	validTypes := map[string]bool{
		"eve":      true,
		"abyssal":  true,
		"wormhole": true,
		"void":     true,
		"hidden":   true,
	}
	
	return validTypes[universeType]
}

// ValidatePathComponent validates universe path components (region, constellation, system names)
func ValidatePathComponent(component string) bool {
	if len(component) == 0 || len(component) > 100 {
		return false
	}
	
	// Allow alphanumeric, spaces, hyphens, apostrophes, and periods
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s\-'.]+$`, component)
	return matched
}

// ValidateCacheExpiration validates cache expiration duration
func ValidateCacheExpiration(expiration time.Duration) bool {
	// Between 1 second and 24 hours
	return expiration >= time.Second && expiration <= 24*time.Hour
}

// ValidateIterationCount validates iteration count for performance tests
func ValidateIterationCount(iterations int) bool {
	return iterations >= 1 && iterations <= 10000
}

// ValidateTimeout validates timeout values in seconds
func ValidateTimeout(timeout int) bool {
	return timeout >= 1 && timeout <= 300 // 1 second to 5 minutes
}