package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Telegram
	TelegramToken string
	AllowedUsers  []int64

	// Database
	DBPath string

	// Monitoring
	PingCount            int
	PingTimeout          time.Duration
	HTTPTimeout          time.Duration
	DefaultCheckInterval time.Duration
	MetricsRetention     time.Duration

	// API
	APIEnabled bool
	APIPort    int
	APIKey     string

	// Auto-restart
	AutoRestartEnabled         bool
	AutoRestartDelay           time.Duration
	AutoRestartMaxAttempts     int
	AutoRestartBackoffMultiplier float64
	AutoRestartMaxDelay        time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Defaults
		DBPath:               getEnv("DB_PATH", "data/state.db"),
		PingCount:            getEnvInt("PING_COUNT", 3),
		PingTimeout:          getEnvDuration("PING_TIMEOUT", 5*time.Second),
		HTTPTimeout:          getEnvDuration("HTTP_TIMEOUT", 10*time.Second),
		DefaultCheckInterval: getEnvDuration("DEFAULT_CHECK_INTERVAL", 30*time.Second),
		MetricsRetention:     getEnvDuration("METRICS_RETENTION", 30*24*time.Hour), // 30 days
		APIEnabled:           getEnvBool("API_ENABLED", true),
		APIPort:              getEnvInt("API_PORT", 8080),
		APIKey:               getEnv("API_KEY", ""),
		// Auto-restart defaults
		AutoRestartEnabled:         getEnvBool("AUTO_RESTART_ENABLED", true),
		AutoRestartDelay:           getEnvDuration("AUTO_RESTART_DELAY", 30*time.Second),
		AutoRestartMaxAttempts:     getEnvInt("AUTO_RESTART_MAX_ATTEMPTS", 0), // 0 = unlimited
		AutoRestartBackoffMultiplier: getEnvFloat("AUTO_RESTART_BACKOFF_MULTIPLIER", 2.0),
		AutoRestartMaxDelay:        getEnvDuration("AUTO_RESTART_MAX_DELAY", 5*time.Minute),
	}

	// Optional: Telegram token (if not set, bot will be disabled)
	cfg.TelegramToken = os.Getenv("TELEGRAM_TOKEN")

	// Optional: Allowed users (comma-separated list of user IDs)
	if allowedUsersStr := os.Getenv("ALLOWED_USERS"); allowedUsersStr != "" {
		userIDs := strings.Split(allowedUsersStr, ",")
		for _, idStr := range userIDs {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				cfg.AllowedUsers = append(cfg.AllowedUsers, id)
			}
		}
	}

	// Generate random API key if not provided
	if cfg.APIEnabled && cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable is required when API_ENABLED=true")
	}

	return cfg, nil
}

// LoadFromMap creates Config from string map (used by ConfigManager)
func LoadFromMap(configMap map[string]string) (*Config, error) {
	cfg := &Config{
		// Set defaults first
		DBPath:               "data/state.db",
		PingCount:            3,
		PingTimeout:          5 * time.Second,
		HTTPTimeout:          10 * time.Second,
		DefaultCheckInterval: 30 * time.Second,
		MetricsRetention:     30 * 24 * time.Hour,
		APIEnabled:           true,
		APIPort:              8080,
		// Auto-restart defaults
		AutoRestartEnabled:         true,
		AutoRestartDelay:           30 * time.Second,
		AutoRestartMaxAttempts:     0,
		AutoRestartBackoffMultiplier: 2.0,
		AutoRestartMaxDelay:        5 * time.Minute,
	}

	// Override with values from map
	if val, ok := configMap["TELEGRAM_TOKEN"]; ok {
		cfg.TelegramToken = val
	}

	if val, ok := configMap["ALLOWED_USERS"]; ok && val != "" {
		userIDs := strings.Split(val, ",")
		for _, idStr := range userIDs {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				cfg.AllowedUsers = append(cfg.AllowedUsers, id)
			}
		}
	}

	if val, ok := configMap["DB_PATH"]; ok {
		cfg.DBPath = val
	}

	if val, ok := configMap["PING_COUNT"]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.PingCount = intVal
		}
	}

	if val, ok := configMap["PING_TIMEOUT"]; ok {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.PingTimeout = duration
		}
	}

	if val, ok := configMap["HTTP_TIMEOUT"]; ok {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.HTTPTimeout = duration
		}
	}

	if val, ok := configMap["DEFAULT_CHECK_INTERVAL"]; ok {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.DefaultCheckInterval = duration
		}
	}

	if val, ok := configMap["METRICS_RETENTION"]; ok {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.MetricsRetention = duration
		}
	}

	if val, ok := configMap["API_ENABLED"]; ok {
		cfg.APIEnabled = val == "true" || val == "1"
	}

	if val, ok := configMap["API_PORT"]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.APIPort = intVal
		}
	}

	if val, ok := configMap["API_KEY"]; ok {
		cfg.APIKey = val
	}

	if val, ok := configMap["AUTO_RESTART_ENABLED"]; ok {
		cfg.AutoRestartEnabled = val == "true" || val == "1"
	}

	if val, ok := configMap["AUTO_RESTART_DELAY"]; ok {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.AutoRestartDelay = duration
		}
	}

	if val, ok := configMap["AUTO_RESTART_MAX_ATTEMPTS"]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			cfg.AutoRestartMaxAttempts = intVal
		}
	}

	if val, ok := configMap["AUTO_RESTART_BACKOFF_MULTIPLIER"]; ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.AutoRestartBackoffMultiplier = floatVal
		}
	}

	if val, ok := configMap["AUTO_RESTART_MAX_DELAY"]; ok {
		if duration, err := time.ParseDuration(val); err == nil {
			cfg.AutoRestartMaxDelay = duration
		}
	}

	// Validate required fields
	if cfg.APIEnabled && cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY is required when API is enabled")
	}

	return cfg, nil
}

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt returns environment variable as int or default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvDuration returns environment variable as duration or default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvBool returns environment variable as bool or default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// getEnvFloat returns environment variable as float64 or default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
