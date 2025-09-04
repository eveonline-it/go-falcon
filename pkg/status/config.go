package status

import (
	"os"
	"strconv"
	"time"
)

// LoadConfigFromEnv loads status service configuration from environment variables
func LoadConfigFromEnv() Config {
	config := DefaultConfig()

	// STATUS_BROADCAST_ENABLED
	if val := os.Getenv("STATUS_BROADCAST_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.BroadcastEnabled = enabled
		}
	}

	// STATUS_BROADCAST_INTERVAL
	if val := os.Getenv("STATUS_BROADCAST_INTERVAL"); val != "" {
		if interval, err := time.ParseDuration(val); err == nil {
			config.BroadcastInterval = interval
		}
	}

	// STATUS_CHANGE_DETECTION_INTERVAL
	if val := os.Getenv("STATUS_CHANGE_DETECTION_INTERVAL"); val != "" {
		if interval, err := time.ParseDuration(val); err == nil {
			config.ChangeDetectionInterval = interval
		}
	}

	// STATUS_CRITICAL_ALERT_ENABLED
	if val := os.Getenv("STATUS_CRITICAL_ALERT_ENABLED"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.CriticalAlertEnabled = enabled
		}
	}

	// STATUS_ALERT_CPU_THRESHOLD
	if val := os.Getenv("STATUS_ALERT_CPU_THRESHOLD"); val != "" {
		if threshold, err := strconv.ParseFloat(val, 64); err == nil {
			config.AlertCPUThreshold = threshold
		}
	}

	// STATUS_ALERT_MEMORY_THRESHOLD
	if val := os.Getenv("STATUS_ALERT_MEMORY_THRESHOLD"); val != "" {
		if threshold, err := strconv.ParseFloat(val, 64); err == nil {
			config.AlertMemoryThreshold = threshold
		}
	}

	// STATUS_ALERT_ERROR_RATE_THRESHOLD
	if val := os.Getenv("STATUS_ALERT_ERROR_RATE_THRESHOLD"); val != "" {
		if threshold, err := strconv.ParseFloat(val, 64); err == nil {
			config.AlertErrorRateThreshold = threshold
		}
	}

	return config
}
