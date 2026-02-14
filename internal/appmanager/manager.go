package appmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"tg-monitor-bot/internal/storage"
)

// AppManager orchestrates the entire application
type AppManager struct {
	storage       *storage.BoltDB
	configManager *ConfigManager
	botProcess    *BotProcess
	echoServer    *echo.Echo
	apiKey        string
	apiPort       int
	apiEnabled    bool
	startTime     time.Time
	logger        *log.Logger
}

// New creates a new AppManager
func New(db *storage.BoltDB) *AppManager {
	return &AppManager{
		storage:    db,
		startTime:  time.Now(),
		logger:     log.New(log.Writer(), "[APPMANAGER] ", log.LstdFlags),
	}
}

// Start initializes and starts Echo API and Bot
func (am *AppManager) Start() error {
	am.logger.Println("Starting AppManager...")

	// Create ConfigManager
	am.configManager = NewConfigManager(am.storage)

	// Set onChange callback to restart bot
	am.configManager.SetOnChange(func() {
		am.logger.Println("Config changed, triggering bot restart...")
		if err := am.RestartBot(); err != nil {
			am.logger.Printf("Failed to restart bot: %v", err)
		}
	})

	// Load config from DB or env
	if err := am.configManager.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get config for initialization
	cfg, err := am.configManager.AsConfig()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Store API settings
	am.apiEnabled = cfg.APIEnabled
	am.apiPort = cfg.APIPort
	am.apiKey = cfg.APIKey

	// Start Echo server if API is enabled
	if am.apiEnabled {
		if err := am.startEchoServer(); err != nil {
			return fmt.Errorf("failed to start Echo server: %w", err)
		}
	} else {
		am.logger.Println("API disabled, skipping Echo server")
	}

	// Create and start bot process
	am.botProcess = NewBotProcess(am.storage)

	// Set auto-restart callback
	am.botProcess.SetRestartFunc(func() error {
		am.logger.Println("Auto-restart callback triggered")
		return am.RestartBot()
	})

	if err := am.botProcess.Start(cfg); err != nil {
		// Log the error but don't fail - bot process tracks its own health
		am.logger.Printf("⚠️  Bot process started with errors: %v", err)
	}

	am.logger.Println("✅ AppManager started successfully")
	return nil
}

// startEchoServer initializes and starts the Echo HTTP server
func (am *AppManager) startEchoServer() error {
	am.echoServer = echo.New()

	// Configure Echo
	am.echoServer.HideBanner = true
	am.echoServer.HidePort = true

	// Add middleware
	am.echoServer.Use(middleware.Recover())

	// Setup routes
	am.setupRoutes()

	// Log API key in development mode
	if am.isDevMode() {
		if am.apiKey == "" {
			am.logger.Println("⚠️  DEV MODE: No API key configured. API will require X-API-Key header.")
		} else {
			am.logger.Printf("🔑 DEV MODE: API Key = %s", am.apiKey)
		}
	}

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%d", am.apiPort)
		am.logger.Printf("Starting Echo server on %s", addr)

		if err := am.echoServer.Start(addr); err != nil {
			am.logger.Printf("Echo server stopped: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	am.logger.Printf("✅ Echo API server started on port %d", am.apiPort)
	return nil
}

// RestartBot stops and starts the bot with fresh config
func (am *AppManager) RestartBot() error {
	am.logger.Println("Restarting bot...")

	// Get fresh config
	cfg, err := am.configManager.AsConfig()
	if err != nil {
		am.logger.Printf("❌ Failed to get config for restart: %v", err)
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Restart bot process - don't fail if bot has errors, it tracks its own health
	if err := am.botProcess.Restart(cfg); err != nil {
		am.logger.Printf("⚠️  Bot restarted with errors: %v", err)
		// Don't return error - bot is running but may be unhealthy
	} else {
		am.logger.Println("✅ Bot restarted successfully")
	}

	return nil
}

// Shutdown gracefully stops everything
func (am *AppManager) Shutdown() error {
	am.logger.Println("Shutting down AppManager...")

	// Stop bot process
	if am.botProcess != nil {
		if err := am.botProcess.Stop(); err != nil {
			am.logger.Printf("Error stopping bot: %v", err)
		}
	}

	// Stop Echo server
	if am.echoServer != nil && am.apiEnabled {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := am.echoServer.Shutdown(ctx); err != nil {
			am.logger.Printf("Error shutting down Echo: %v", err)
		}
	}

	am.logger.Println("✅ AppManager shutdown complete")
	return nil
}

// isDevMode checks if the application is running in development mode
func (am *AppManager) isDevMode() bool {
	// Check for DEBUG, DEV, or DEVELOPMENT environment variables
	debug := os.Getenv("DEBUG")
	dev := os.Getenv("DEV")
	development := os.Getenv("DEVELOPMENT")

	// Return true if any dev mode indicator is set to a truthy value
	for _, env := range []string{debug, dev, development} {
		if env != "" {
			lower := strings.ToLower(env)
			if lower == "true" || lower == "1" || lower == "yes" || lower == "on" {
				return true
			}
		}
	}

	return false
}
