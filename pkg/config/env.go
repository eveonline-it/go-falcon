package config

import (
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// parseDurationWithDays parses a duration string with extended support for days.
// A duration string is a possibly signed sequence of
// decimal numbers, each with optional fraction and a unit suffix,
// such as "300ms", "-1.5h", "2h45m", "7d", or "1d12h".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d".
func parseDurationWithDays(s string) (time.Duration, error) {
	// Check if string contains 'd' for days
	if !strings.Contains(s, "d") {
		return time.ParseDuration(s)
	}

	// Use regex to find and replace day units
	dayRegex := regexp.MustCompile(`(\d+(?:\.\d+)?)d`)
	converted := dayRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the number part (without 'd')
		numStr := match[:len(match)-1]
		if num, err := strconv.ParseFloat(numStr, 64); err == nil {
			// Convert days to hours (1 day = 24 hours)
			hours := num * 24
			return strconv.FormatFloat(hours, 'f', -1, 64) + "h"
		}
		return match // Return original if parsing fails
	})

	return time.ParseDuration(converted)
}

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
	// Default to nothing if not set at all
	return ""
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

// GetCookieDuration returns the cookie duration for auth cookies
// Accepts values like "24h", "7d", "30m", "1h30m", "1d12h" (extended format with days support)
func GetCookieDuration() time.Duration {
	durationStr := GetEnv("COOKIE_DURATION", "24h")
	duration, err := parseDurationWithDays(durationStr)
	if err != nil {
		// If parsing fails, default to 24 hours and log a warning
		slog.Warn("⚠️ Failed to parse COOKIE_DURATION, using default",
			slog.String("value", durationStr),
			slog.String("error", err.Error()),
			slog.String("default", "24h"))
		return 24 * time.Hour
	}
	return duration
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

// GetOpenAPIServers returns the OpenAPI servers configuration from environment variables
// Format: OPENAPI_SERVERS="url1|description1,url2|description2"
// Example: OPENAPI_SERVERS="https://api.example.com|Production,http://localhost:3000|Development"
func GetOpenAPIServers() []*OpenAPIServer {
	serversEnv := GetEnv("OPENAPI_SERVERS", "")
	if serversEnv == "" {
		return nil // Return nil to use default configuration
	}

	var servers []*OpenAPIServer
	serverPairs := strings.Split(serversEnv, ",")

	for _, pair := range serverPairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, "|")
		if len(parts) != 2 {
			continue // Skip invalid format
		}

		url := strings.TrimSpace(parts[0])
		description := strings.TrimSpace(parts[1])

		if url != "" && description != "" {
			servers = append(servers, &OpenAPIServer{
				URL:         url,
				Description: description,
			})
		}
	}

	return servers
}

// GetSDEURL returns the SDE download URL from environment
func GetSDEURL() string {
	return GetEnv("SDE_URL", "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip")
}

// GetSDEChecksumsURL returns the SDE checksums file URL from environment
func GetSDEChecksumsURL() string {
	return GetEnv("SDE_CHECKSUMS_URL", "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/checksum")
}

// GetWebSocketURL returns the WebSocket URL from environment
func GetWebSocketURL() string {
	return GetEnv("WEBSOCKET_URL", "wss://localhost:3000/websocket/connect")
}

// GetWebSocketPath returns the WebSocket path from environment (for internal routing)
func GetWebSocketPath() string {
	return GetEnv("WEBSOCKET_PATH", "/websocket/connect")
}

// GetWebSocketAllowedOrigins returns the allowed origins for WebSocket connections
func GetWebSocketAllowedOrigins() []string {
	origins := GetEnv("WEBSOCKET_ALLOWED_ORIGINS", "https://go.eveonline.it,http://localhost:3000,https://localhost:3000")
	if origins == "" {
		return []string{}
	}

	result := []string{}
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

// OpenAPIServer represents an OpenAPI server configuration
type OpenAPIServer struct {
	URL         string
	Description string
}
