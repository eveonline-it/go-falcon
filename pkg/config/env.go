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