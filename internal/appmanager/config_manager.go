package appmanager

import (
	"fmt"
	"log"
	"os"
	"sync"

	"tg-monitor-bot/internal/config"
	"tg-monitor-bot/internal/storage"
)

// ConfigManager manages configuration with DB persistence
type ConfigManager struct {
	storage  *storage.BoltDB
	cache    map[string]string
	mu       sync.RWMutex
	onChange func() // Callback when config changes
	logger   *log.Logger
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager(db *storage.BoltDB) *ConfigManager {
	return &ConfigManager{
		storage: db,
		cache:   make(map[string]string),
		logger:  log.New(log.Writer(), "[CONFIG] ", log.LstdFlags),
	}
}

// SetOnChange sets the callback for config changes
func (cm *ConfigManager) SetOnChange(callback func()) {
	cm.onChange = callback
}

// Load reads config from DB, falls back to environment variables
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Try to load from database
	dbConfigs, err := cm.storage.GetAllConfig()
	if err != nil {
		return fmt.Errorf("failed to get config from DB: %w", err)
	}

	if len(dbConfigs) > 0 {
		// Load from database
		cm.logger.Printf("Loading configuration from database (%d entries)", len(dbConfigs))
		for key, entry := range dbConfigs {
			cm.cache[key] = entry.Value
		}
		return nil
	}

	// Database is empty, load from environment and save to DB
	cm.logger.Println("Database empty, loading from environment variables")

	envKeys := []string{
		"TELEGRAM_TOKEN",
		"ALLOWED_USERS",
		"DB_PATH",
		"PING_COUNT",
		"PING_TIMEOUT",
		"HTTP_TIMEOUT",
		"DEFAULT_CHECK_INTERVAL",
		"METRICS_RETENTION",
		"API_ENABLED",
		"API_PORT",
		"API_KEY",
	}

	for _, key := range envKeys {
		value := os.Getenv(key)
		if value != "" {
			cm.cache[key] = value
			// Save to database for future runs
			if err := cm.storage.SaveConfig(key, value, "env"); err != nil {
				cm.logger.Printf("Warning: Failed to save %s to DB: %v", key, err)
			}
		}
	}

	// Set defaults for missing values
	cm.setDefaults()

	cm.logger.Printf("Loaded %d config entries from environment", len(cm.cache))
	return nil
}

// setDefaults sets default values for missing config
func (cm *ConfigManager) setDefaults() {
	defaults := map[string]string{
		"DB_PATH":                "data/state.db",
		"PING_COUNT":             "3",
		"PING_TIMEOUT":           "5s",
		"HTTP_TIMEOUT":           "10s",
		"DEFAULT_CHECK_INTERVAL": "30s",
		"METRICS_RETENTION":      "720h",
		"API_ENABLED":            "true",
		"API_PORT":               "8080",
	}

	for key, defaultValue := range defaults {
		if _, exists := cm.cache[key]; !exists {
			cm.cache[key] = defaultValue
		}
	}
}

// Get retrieves a config value
func (cm *ConfigManager) Get(key string) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cache[key]
}

// Set updates a config value and saves to DB
func (cm *ConfigManager) Set(key, value string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Update cache
	cm.cache[key] = value

	// Save to database
	if err := cm.storage.SaveConfig(key, value, "api"); err != nil {
		return fmt.Errorf("failed to save config to DB: %w", err)
	}

	cm.logger.Printf("Config updated: %s", key)

	// Trigger onChange callback
	if cm.onChange != nil {
		go cm.onChange()
	}

	return nil
}

// GetAll returns all config as map
func (cm *ConfigManager) GetAll() map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make(map[string]string, len(cm.cache))
	for k, v := range cm.cache {
		result[k] = v
	}
	return result
}

// AsConfig converts to config.Config struct
func (cm *ConfigManager) AsConfig() (*config.Config, error) {
	cm.mu.RLock()
	configMap := make(map[string]string, len(cm.cache))
	for k, v := range cm.cache {
		configMap[k] = v
	}
	cm.mu.RUnlock()

	return config.LoadFromMap(configMap)
}

// Delete removes a config entry
func (cm *ConfigManager) Delete(key string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cache, key)

	if err := cm.storage.DeleteConfig(key); err != nil {
		return fmt.Errorf("failed to delete config from DB: %w", err)
	}

	cm.logger.Printf("Config deleted: %s", key)
	return nil
}
