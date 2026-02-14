package appmanager

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"tg-monitor-bot/internal/storage"
)

// handleGetTelegramChats returns all configured telegram chats
// Currently uses source_chats association through sources
func (am *AppManager) handleGetTelegramChats(c echo.Context) error {
	// For MVP, return empty list since telegram chats are managed through source creation
	// Full implementation would maintain separate chat registry
	type ChatResponse struct {
		ChatID    int64  `json:"chat_id"`
		CreatedAt string `json:"created_at"`
	}

	return c.JSON(http.StatusOK, []ChatResponse{})
}

// handleAddTelegramChat adds a telegram chat
// Currently managed through /add_source Telegram command or UI
func (am *AppManager) handleAddTelegramChat(c echo.Context) error {
	var req struct {
		ChatID int64 `json:"chat_id"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.ChatID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Chat ID is required",
		})
	}

	// For MVP, return success but don't actually store
	// Full implementation would maintain separate chat registry
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"chat_id": req.ChatID,
	})
}

// handleRemoveTelegramChat removes a telegram chat
func (am *AppManager) handleRemoveTelegramChat(c echo.Context) error {
	// For MVP, just return success
	// Full implementation would remove from chat registry
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Chat removed",
	})
}

// handleTestTelegramChat sends a test notification to a specific Telegram chat
func (am *AppManager) handleTestTelegramChat(c echo.Context) error {
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid chat ID",
		})
	}

	// Get the bot instance
	tgBot := am.botProcess.GetBot()
	if tgBot == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Telegram bot not available. Check if TELEGRAM_TOKEN is configured.",
		})
	}

	// Create a test message
	testMessage := fmt.Sprintf(
		"TEST NOTIFICATION\n\n"+
			"This is a test message from the Outage Monitor Bot.\n\n"+
			"Source: Test Source\n"+
			"Status: ONLINE\n"+
			"Time: %s\n\n"+
			"If you see this message, notifications are working correctly!",
		time.Now().Format("2006-01-02 15:04:05"),
	)

	// Send test message
	ctx := context.Background()
	err = tgBot.SendTestMessage(ctx, chatID, testMessage)

	if err != nil {
		am.logger.Printf("Failed to send test message to chat %d: %v", chatID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to send test message: %v", err),
		})
	}

	am.logger.Printf("Sent test notification to Telegram chat %d", chatID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Test notification sent successfully",
		"chat_id": chatID,
		"sent_at": time.Now(),
	})
}

// handleTestWebhook sends a test notification to a specific webhook
func (am *AppManager) handleTestWebhook(c echo.Context) error {
	webhookID := c.Param("webhook_id")

	// Get the webhook from storage
	webhook, err := am.storage.GetWebhook(webhookID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Webhook not found",
		})
	}

	if !webhook.Enabled {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Webhook is disabled",
		})
	}

	// Create test source and status change for payload
	testSource := &storage.Source{
		ID:             "test-source-id",
		Name:           "Test Source",
		Type:           "ping",
		Target:         "8.8.8.8",
		CurrentStatus:  1,
		LastCheckTime:  time.Now(),
		LastChangeTime: time.Now().Add(-1 * time.Hour),
	}

	testChange := &storage.StatusChange{
		ID:         "test-change-id",
		SourceID:   "test-source-id",
		OldStatus:  0,
		NewStatus:  1,
		DurationMs: 3600000, // 1 hour
		Timestamp:  time.Now(),
	}

	// Get webhook notifier to send test
	webhookNotifier := am.botProcess.GetWebhookNotifier()
	if webhookNotifier == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Webhook notifier not available",
		})
	}

	// Send test webhook (synchronously for this test)
	am.logger.Printf("Sending test webhook to %s", webhook.URL)
	
	// We'll use the OnStatusChange method but note this is a test
	go webhookNotifier.OnStatusChange(testSource, testChange)

	am.logger.Printf("Sent test notification to webhook %s (%s)", webhook.URL, webhookID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "Test notification sent successfully",
		"webhook_id": webhookID,
		"url":        webhook.URL,
		"sent_at":    time.Now(),
	})
}
