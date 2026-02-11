package appmanager

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"tg-monitor-bot/internal/bot"
	"tg-monitor-bot/internal/config"
	"tg-monitor-bot/internal/monitor"
	"tg-monitor-bot/internal/storage"
)

// RestartFunc is a callback for triggering bot restart
type RestartFunc func() error

// BotProcess manages the bot lifecycle
type BotProcess struct {
	config         *config.Config
	storage        *storage.BoltDB
	bot            *bot.Bot
	monitor        *monitor.Monitor
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
	healthy        bool
	lastError      error
	startTime      time.Time
	restartFunc    RestartFunc
	restartAttempts int
	restartTimer   *time.Timer
	mu             sync.Mutex
	logger         *log.Logger
}

// NewBotProcess creates a new BotProcess
func NewBotProcess(db *storage.BoltDB) *BotProcess {
	return &BotProcess{
		storage: db,
		logger:  log.New(log.Writer(), "[BOT-PROCESS] ", log.LstdFlags),
	}
}

// SetRestartFunc sets the callback for auto-restart
func (bp *BotProcess) SetRestartFunc(fn RestartFunc) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.restartFunc = fn
}

// Start initializes and starts bot + monitor
func (bp *BotProcess) Start(cfg *config.Config) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.running {
		return fmt.Errorf("bot is already running")
	}

	bp.logger.Println("Starting bot process...")
	bp.config = cfg
	bp.startTime = time.Now()
	bp.lastError = nil
	bp.healthy = false

	// Create context for bot
	bp.ctx, bp.cancel = context.WithCancel(context.Background())

	// Check if Telegram token is provided (treat placeholder as empty)
	if cfg.TelegramToken == "" || cfg.TelegramToken == "your_bot_token_here" {
		bp.logger.Println("⚠️  TELEGRAM_TOKEN not set - running in web-only mode")
		bp.logger.Println("   Monitor will check sources but won't send Telegram notifications")
		bp.logger.Println("   API endpoints are fully functional for source management")

		// Initialize Monitor without bot callback (no notifications)
		mon := monitor.New(bp.storage, cfg, nil)
		bp.monitor = mon

		// Start monitor (loads sources and starts goroutines)
		if err := mon.Start(bp.ctx); err != nil {
			bp.lastError = fmt.Errorf("failed to start monitor: %w", err)
			bp.logger.Printf("❌ Monitor start failed: %v", err)
			bp.running = true     // Mark as running but unhealthy
			bp.restartAttempts++
			bp.scheduleAutoRestart() // Schedule auto-restart
			return nil               // Don't kill the app
		}

		bp.running = true
		bp.healthy = true
		bp.restartAttempts = 0
		bp.logger.Println("✅ Bot process started in web-only mode (monitor active, no Telegram bot)")
		return nil
	}

	// Initialize Bot first (with nil monitor)
	telegramBot, err := bot.New(cfg, bp.storage, nil)
	if err != nil {
		bp.lastError = fmt.Errorf("failed to initialize bot: %w", err)
		bp.logger.Printf("❌ Bot initialization failed: %v", bp.formatBotError(err))
		bp.running = true     // Mark as running but unhealthy
		bp.restartAttempts++
		bp.scheduleAutoRestart() // Schedule auto-restart
		return nil               // Don't kill the app
	}
	bp.bot = telegramBot

	// Initialize Monitor with callback to Bot.OnStatusChange
	mon := monitor.New(bp.storage, cfg, telegramBot.OnStatusChange)
	bp.monitor = mon

	// Wire monitor to bot
	telegramBot.SetMonitor(mon)

	// Start monitor (loads sources and starts goroutines)
	if err := mon.Start(bp.ctx); err != nil {
		bp.lastError = fmt.Errorf("failed to start monitor: %w", err)
		bp.logger.Printf("❌ Monitor start failed: %v", err)
		bp.running = true     // Mark as running but unhealthy
		bp.restartAttempts++
		bp.scheduleAutoRestart() // Schedule auto-restart
		return nil               // Don't kill the app
	}

	// Start bot in goroutine with error recovery
	go bp.runBotWithRecovery(telegramBot)

	bp.running = true
	bp.healthy = true
	bp.restartAttempts = 0 // Reset on successful start
	bp.logger.Println("✅ Bot process started successfully")

	return nil
}

// runBotWithRecovery runs the bot with panic recovery
func (bp *BotProcess) runBotWithRecovery(telegramBot *bot.Bot) {
	defer func() {
		if r := recover(); r != nil {
			bp.mu.Lock()
			bp.healthy = false
			bp.lastError = fmt.Errorf("bot panic: %v", r)
			bp.restartAttempts++
			bp.mu.Unlock()
			bp.logger.Printf("❌ Bot panicked: %v", r)

			// Schedule auto-restart
			bp.scheduleAutoRestart()
		}
	}()

	// Start bot - this blocks until context is cancelled
	telegramBot.Start(bp.ctx)

	// If we get here, bot stopped normally
	bp.mu.Lock()
	wasUnexpected := bp.ctx.Err() == nil
	if wasUnexpected {
		// Bot stopped unexpectedly (not due to cancellation)
		bp.healthy = false
		bp.lastError = fmt.Errorf("bot stopped unexpectedly")
		bp.restartAttempts++
		bp.logger.Println("⚠️  Bot stopped unexpectedly")
	}
	bp.mu.Unlock()

	// Schedule auto-restart if it was unexpected
	if wasUnexpected {
		bp.scheduleAutoRestart()
	}
}

// Stop gracefully stops bot and monitor
func (bp *BotProcess) Stop() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if !bp.running {
		return nil
	}

	bp.logger.Println("Stopping bot process...")

	// Cancel any pending auto-restart
	if bp.restartTimer != nil {
		bp.restartTimer.Stop()
		bp.restartTimer = nil
		bp.logger.Println("Cancelled pending auto-restart")
	}

	// Cancel context to stop all goroutines
	if bp.cancel != nil {
		bp.cancel()
	}

	// Give goroutines time to shut down
	time.Sleep(500 * time.Millisecond)

	bp.running = false
	bp.bot = nil
	bp.monitor = nil

	bp.logger.Println("✅ Bot process stopped")

	return nil
}

// Restart stops and starts with new config
func (bp *BotProcess) Restart(cfg *config.Config) error {
	bp.logger.Println("Restarting bot process with new config...")

	if err := bp.Stop(); err != nil {
		return fmt.Errorf("failed to stop bot: %w", err)
	}

	// Small delay to ensure clean shutdown
	time.Sleep(100 * time.Millisecond)

	if err := bp.Start(cfg); err != nil {
		return fmt.Errorf("failed to start bot: %w", err)
	}

	return nil
}

// IsRunning returns current status
func (bp *BotProcess) IsRunning() bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.running
}

// IsHealthy returns whether bot is running without errors
func (bp *BotProcess) IsHealthy() bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.running && bp.healthy
}

// GetLastError returns the last error that occurred
func (bp *BotProcess) GetLastError() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.lastError
}

// GetStatus returns bot status information
func (bp *BotProcess) GetStatus() map[string]interface{} {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Determine mode and component status
	telegramConnected := bp.bot != nil && bp.running && bp.healthy
	monitorRunning := bp.monitor != nil && bp.running
	webOnlyMode := bp.config != nil && (bp.config.TelegramToken == "" || bp.config.TelegramToken == "your_bot_token_here")

	status := map[string]interface{}{
		"running":            bp.running,
		"healthy":            bp.healthy,
		"telegram_connected": telegramConnected,
		"monitor_running":    monitorRunning,
		"web_only_mode":      webOnlyMode,
	}

	if bp.lastError != nil {
		status["last_error"] = bp.lastError.Error()
	}

	// Auto-restart info
	if bp.config != nil {
		status["auto_restart"] = map[string]interface{}{
			"enabled":         bp.config.AutoRestartEnabled,
			"attempts":        bp.restartAttempts,
			"max_attempts":    bp.config.AutoRestartMaxAttempts,
			"next_delay":      bp.calculateBackoffDelay().String(),
			"timer_active":    bp.restartTimer != nil,
		}
	}

	if bp.running {
		status["started_at"] = bp.startTime
		status["uptime"] = time.Since(bp.startTime).String()

		// Get configuration info
		if bp.config != nil {
			status["config"] = map[string]interface{}{
				"telegram_token": maskString(bp.config.TelegramToken),
				"allowed_users":  bp.config.AllowedUsers,
				"check_interval": bp.config.DefaultCheckInterval.String(),
				"ping_count":     bp.config.PingCount,
				"ping_timeout":   bp.config.PingTimeout.String(),
				"http_timeout":   bp.config.HTTPTimeout.String(),
			}
		}

		// Get active sources count
		if bp.monitor != nil {
			sources, err := bp.monitor.GetAllSources()
			if err == nil {
				enabled := 0
				for _, src := range sources {
					if src.Enabled {
						enabled++
					}
				}
				status["total_sources"] = len(sources)
				status["active_sources"] = enabled
			}
		}
	}

	return status
}

// maskString masks sensitive strings (show first 4 and last 4 chars)
func maskString(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// scheduleAutoRestart schedules an automatic restart after backoff delay
func (bp *BotProcess) scheduleAutoRestart() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Check if auto-restart is enabled
	if bp.config == nil || !bp.config.AutoRestartEnabled {
		bp.logger.Println("Auto-restart disabled, not scheduling restart")
		return
	}

	// Check max attempts
	if bp.config.AutoRestartMaxAttempts > 0 && bp.restartAttempts >= bp.config.AutoRestartMaxAttempts {
		bp.logger.Printf("⚠️  Max restart attempts (%d) reached, not scheduling restart", bp.config.AutoRestartMaxAttempts)
		return
	}

	// Calculate backoff delay
	delay := bp.calculateBackoffDelay()

	bp.logger.Printf("🔄 Scheduling auto-restart in %s (attempt %d)", delay, bp.restartAttempts+1)

	// Cancel existing timer if any
	if bp.restartTimer != nil {
		bp.restartTimer.Stop()
	}

	// Schedule restart
	bp.restartTimer = time.AfterFunc(delay, func() {
		bp.logger.Println("⏰ Auto-restart timer triggered")
		if bp.restartFunc != nil {
			if err := bp.restartFunc(); err != nil {
				bp.logger.Printf("❌ Auto-restart failed: %v", err)
			}
		} else {
			bp.logger.Println("⚠️  No restart function set, cannot auto-restart")
		}
	})
}

// calculateBackoffDelay calculates the delay before next restart with exponential backoff
func (bp *BotProcess) calculateBackoffDelay() time.Duration {
	if bp.restartAttempts == 0 {
		return bp.config.AutoRestartDelay
	}

	// Calculate exponential backoff: baseDelay * (multiplier ^ attempts)
	multiplier := bp.config.AutoRestartBackoffMultiplier
	delay := float64(bp.config.AutoRestartDelay) * math.Pow(multiplier, float64(bp.restartAttempts))

	// Cap at max delay
	if time.Duration(delay) > bp.config.AutoRestartMaxDelay {
		return bp.config.AutoRestartMaxDelay
	}

	return time.Duration(delay)
}

// ResetRestartAttempts resets the restart attempt counter (called on successful start)
func (bp *BotProcess) ResetRestartAttempts() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.restartAttempts = 0
	bp.logger.Println("✅ Restart attempts counter reset")
}

// IncrementRestartAttempts increments the restart attempt counter
func (bp *BotProcess) IncrementRestartAttempts() {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.restartAttempts++
}

// GetMonitor returns the monitor instance
func (bp *BotProcess) GetMonitor() *monitor.Monitor {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.monitor
}

// GetContext returns the bot process context
func (bp *BotProcess) GetContext() context.Context {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.ctx
}

// formatBotError converts cryptic Telegram API errors into user-friendly messages
func (bp *BotProcess) formatBotError(err error) string {
	errStr := err.Error()

	// Check for common error patterns
	switch {
	case contains(errStr, "not found", "Not Found", "getMe"):
		return "Invalid TELEGRAM_TOKEN - bot not found. Check your token from @BotFather and update TELEGRAM_TOKEN in .env or via API"
	case contains(errStr, "unauthorized", "Unauthorized"):
		return "Unauthorized TELEGRAM_TOKEN - token may have been revoked. Get a new token from @BotFather"
	case contains(errStr, "connection refused", "network"):
		return "Network error - cannot reach Telegram API. Check your internet connection"
	case contains(errStr, "timeout", "deadline exceeded"):
		return "Connection timeout - Telegram API not responding. Check your network or try again later"
	default:
		return fmt.Sprintf("%v - Check TELEGRAM_TOKEN in .env or leave empty for web-only mode", err)
	}
}

// contains checks if a string contains any of the given substrings (case-insensitive)
func contains(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
