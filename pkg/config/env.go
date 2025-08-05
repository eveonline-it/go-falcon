package config

import (
	"os"
	"strconv"
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
	prefix := GetEnv("API_PREFIX", "")
	if prefix == "" {
		return ""
	}
	return "/" + prefix
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