package appmanager

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// setupRoutes configures all API routes
func (am *AppManager) setupRoutes() {
	// Incoming webhook heartbeat (no API key) - must be registered before auth middleware applies
	am.echoServer.GET("/webhooks/incoming/:token", am.handleIncomingWebhook)
	am.echoServer.POST("/webhooks/incoming/:token", am.handleIncomingWebhook)

	// Middleware
	am.echoServer.Use(am.apiKeyMiddleware)

	// Config endpoints
	am.echoServer.GET("/config", am.handleGetAllConfig)
	am.echoServer.GET("/config/:key", am.handleGetConfig)
	am.echoServer.PUT("/config/:key", am.handleUpdateConfig)
	am.echoServer.POST("/config/reload", am.handleReloadConfig)

	// Status endpoints
	am.echoServer.GET("/health", am.handleHealth)
	am.echoServer.GET("/status", am.handleStatus)

	// Source endpoints - collection routes
	am.echoServer.GET("/sources", am.handleGetSources)
	am.echoServer.POST("/sources", am.handleCreateSource)
	// Source-specific sub-resource routes (must come BEFORE generic :id routes)
	// These use :source_id or :id as parameter names matching their handlers
	am.echoServer.POST("/sources/:id/pause", am.handlePauseSource)
	am.echoServer.POST("/sources/:id/resume", am.handleResumeSource)
	am.echoServer.GET("/sources/:source_id/webhooks", am.handleGetSourceWebhooks)
	am.echoServer.POST("/sources/:source_id/webhooks/:webhook_id", am.handleAddSourceWebhook)
	am.echoServer.DELETE("/sources/:source_id/webhooks/:webhook_id", am.handleRemoveSourceWebhook)
	am.echoServer.GET("/sources/:source_id/telegram-chats", am.handleGetSourceTelegramChats)
	am.echoServer.POST("/sources/:source_id/telegram-chats/:chat_id", am.handleAddSourceTelegramChat)
	am.echoServer.DELETE("/sources/:source_id/telegram-chats/:chat_id", am.handleRemoveSourceTelegramChat)
	// Generic source routes (must come AFTER specific sub-resource routes)
	am.echoServer.PUT("/sources/:id", am.handleUpdateSource)
	am.echoServer.DELETE("/sources/:id", am.handleDeleteSource)

	// Webhook endpoints
	am.echoServer.GET("/webhooks", am.handleGetWebhooks)
	am.echoServer.POST("/webhooks", am.handleCreateWebhook)
	am.echoServer.PUT("/webhooks/:id", am.handleUpdateWebhook)
	am.echoServer.DELETE("/webhooks/:id", am.handleDeleteWebhook)

	// Events endpoints
	am.echoServer.GET("/events", am.handleGetEvents)

	// Telegram chat endpoints
	am.echoServer.GET("/telegram-chats", am.handleGetTelegramChats)
	am.echoServer.POST("/telegram-chats", am.handleAddTelegramChat)
	am.echoServer.DELETE("/telegram-chats/:chat_id", am.handleRemoveTelegramChat)

	// Test notification endpoints
	am.echoServer.POST("/test/telegram/:chat_id", am.handleTestTelegramChat)
	am.echoServer.POST("/test/webhook/:webhook_id", am.handleTestWebhook)
}

// apiKeyMiddleware validates X-API-Key header
func (am *AppManager) apiKeyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip auth for health endpoint
		if c.Path() == "/health" {
			return next(c)
		}
		// Skip auth for incoming webhook heartbeat (public URL for monitored services)
		if strings.HasPrefix(c.Path(), "/webhooks/incoming/") {
			return next(c)
		}

		apiKey := c.Request().Header.Get("X-API-Key")
		if apiKey == "" {
			am.logger.Printf("Missing API key from %s on %s %s", c.RealIP(), c.Request().Method, c.Path())
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Missing X-API-Key header",
			})
		}

		if apiKey != am.apiKey {
			am.logger.Printf("Invalid API key attempt from %s on %s %s - provided: %q (expected: %q)",
				c.RealIP(), c.Request().Method, c.Path(), apiKey, am.apiKey)
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid API key",
			})
		}

		return next(c)
	}
}

// handleGetAllConfig returns all config entries
func (am *AppManager) handleGetAllConfig(c echo.Context) error {
	configs := am.configManager.GetAll()

	// Mask sensitive values
	masked := make(map[string]string)
	for key, value := range configs {
		if key == "TELEGRAM_TOKEN" || key == "API_KEY" {
			if len(value) > 8 {
				masked[key] = value[:4] + "..." + value[len(value)-4:]
			} else {
				masked[key] = "***"
			}
		} else {
			masked[key] = value
		}
	}

	return c.JSON(http.StatusOK, masked)
}

// handleGetConfig returns a specific config entry
func (am *AppManager) handleGetConfig(c echo.Context) error {
	key := c.Param("key")

	value := am.configManager.Get(key)
	if value == "" {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Config key not found",
		})
	}

	// Get metadata from storage
	entry, err := am.storage.GetConfig(key)
	if err != nil {
		// Key exists in cache but not in DB (probably default)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"key":        key,
			"value":      value,
			"updated_by": "default",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"key":        entry.Key,
		"value":      entry.Value,
		"updated_at": entry.UpdatedAt,
		"updated_by": entry.UpdatedBy,
	})
}

// UpdateConfigRequest is the request body for updating config
type UpdateConfigRequest struct {
	Value string `json:"value"`
}

// handleUpdateConfig updates a config entry
func (am *AppManager) handleUpdateConfig(c echo.Context) error {
	key := c.Param("key")

	var req UpdateConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Value == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Value cannot be empty",
		})
	}

	// Update config
	if err := am.configManager.Set(key, req.Value); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Config updated via API: %s", key)

	// Note: The onChange callback will trigger bot restart automatically

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Config updated successfully",
		"key":       key,
		"restarting": "Bot will restart with new config",
	})
}

// handleReloadConfig forces a bot restart with current config
func (am *AppManager) handleReloadConfig(c echo.Context) error {
	am.logger.Println("Manual reload requested via API")

	// Trigger restart
	go am.RestartBot()

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Bot restart initiated",
	})
}

// handleHealth returns health status
func (am *AppManager) handleHealth(c echo.Context) error {
	uptime := time.Since(am.startTime)

	botRunning := am.botProcess.IsRunning()
	botHealthy := am.botProcess.IsHealthy()
	lastError := am.botProcess.GetLastError()

	// Get detailed status for monitor and telegram
	status := am.botProcess.GetStatus()
	monitorRunning := false
	telegramConnected := false
	if mr, ok := status["monitor_running"].(bool); ok {
		monitorRunning = mr
	}
	if tc, ok := status["telegram_connected"].(bool); ok {
		telegramConnected = tc
	}

	// Determine overall health
	overallStatus := "healthy"
	httpStatus := http.StatusOK

	if !botRunning {
		overallStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else if !botHealthy {
		overallStatus = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status":             overallStatus,
		"bot_running":        botRunning,
		"bot_healthy":        botHealthy,
		"monitor_running":    monitorRunning,
		"telegram_connected": telegramConnected,
		"api_running":        true,
		"uptime":             uptime.String(),
		"uptime_seconds":     int(uptime.Seconds()),
		"version":            am.version,
	}

	if lastError != nil {
		response["last_error"] = lastError.Error()
	}

	return c.JSON(httpStatus, response)
}

// handleStatus returns detailed status
func (am *AppManager) handleStatus(c echo.Context) error {
	botStatus := am.botProcess.GetStatus()
	uptime := time.Since(am.startTime)

	// Get current config from ConfigManager
	allConfig := am.configManager.GetAll()

	// Mask sensitive values
	maskedConfig := make(map[string]string)
	for key, value := range allConfig {
		if key == "TELEGRAM_TOKEN" || key == "API_KEY" {
			if len(value) > 8 {
				maskedConfig[key] = value[:4] + "..." + value[len(value)-4:]
			} else {
				maskedConfig[key] = "***"
			}
		} else {
			maskedConfig[key] = value
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"timestamp": time.Now(),
		"bot":       botStatus,
		"api": map[string]interface{}{
			"enabled": am.apiEnabled,
			"port":    am.apiPort,
			"uptime":  uptime.String(),
		},
		"config": maskedConfig,
		"system": map[string]interface{}{
			"uptime":        uptime.String(),
			"uptime_seconds": int(uptime.Seconds()),
			"started_at":    am.startTime,
		},
	})
}
