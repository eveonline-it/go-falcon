package config

import (
	"os"
	"strconv"
	"strings"
)

// GetEnv returns the value of an environment variable or a default value if not set
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetBoolEnv returns the boolean value of an environment variable or a default value if not set
func GetBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// GetIntEnv returns the integer value of an environment variable or a default value if not set
func GetIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// MustGetEnv returns the value of an environment variable or panics if not set
func MustGetEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	panic("Required environment variable " + key + " is not set")
}

// GetAPIPrefix returns the API prefix from environment or default
func GetAPIPrefix() string {
	// Check if API_PREFIX is explicitly set (even if empty)
	if prefix, exists := os.LookupEnv("API_PREFIX"); exists {
		if prefix == "" {
			return "" // No prefix if explicitly set to empty
		}
		if !strings.HasPrefix(prefix, "/") {
			return "/" + prefix
		}
		return prefix
	}
	// Default to /api if not set at all
	return "/api"
}

// EVE Online SSO Configuration
func GetEVEClientID() string {
	return MustGetEnv("EVE_CLIENT_ID")
}

func GetEVEClientSecret() string {
	return MustGetEnv("EVE_CLIENT_SECRET")
}

func GetEVERedirectURI() string {
	return GetEnv("EVE_REDIRECT_URI", "http://localhost:8080/auth/eve/callback")
}

func GetEVEScopes() string {
	return GetEnv("EVE_SCOPES", "publicData")
}

func GetJWTSecret() string {
	return MustGetEnv("JWT_SECRET")
}

// GetFrontendURL returns the frontend URL for redirects
func GetFrontendURL() string {
	return GetEnv("FRONTEND_URL", "https://go.eveonline.it")
}

// GetCookieDomain returns the cookie domain for auth cookies
func GetCookieDomain() string {
	return GetEnv("COOKIE_DOMAIN", ".eveonline.it")
}

// GetEnvInt is an alias for GetIntEnv for backward compatibility
func GetEnvInt(key string, defaultValue int) int {
	return GetIntEnv(key, defaultValue)
}

// GetEnvIntSlice returns a slice of integers from a comma-separated environment variable
func GetEnvIntSlice(key string) []int {
	value := os.Getenv(key)
	if value == "" {
		return []int{}
	}
	
	parts := strings.Split(value, ",")
	result := make([]int, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		if num, err := strconv.Atoi(part); err == nil {
			result = append(result, num)
		}
	}
	
	return result
}

// GetSuperAdminCharacterID returns the super admin character ID from environment
func GetSuperAdminCharacterID() int {
	return GetIntEnv("SUPER_ADMIN_CHARACTER_ID", 0)
}

// GetHumaPort returns the HUMA server port from environment
func GetHumaPort() string {
	return GetEnv("HUMA_PORT", "")
}

// GetHumaSeparateServer returns whether to run HUMA on a separate server
func GetHumaSeparateServer() bool {
	return GetBoolEnv("HUMA_SEPARATE_SERVER", false)
}

// GetHost returns the host interface to bind to (default: all interfaces)
func GetHost() string {
	return GetEnv("HOST", "0.0.0.0")
}

// GetHumaHost returns the HUMA server host interface to bind to
func GetHumaHost() string {
	return GetEnv("HUMA_HOST", GetHost())
}