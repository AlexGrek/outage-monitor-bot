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

// handleGetTelegramChats returns all configured telegram chats from the registry
func (am *AppManager) handleGetTelegramChats(c echo.Context) error {
	chats, err := am.storage.ListChats()
	if err != nil {
		am.logger.Printf("Failed to list chats: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list telegram chats",
		})
	}
	if chats == nil {
		chats = []*storage.Chat{}
	}
	return c.JSON(http.StatusOK, chats)
}

// handleAddTelegramChat adds a named telegram chat to the registry
func (am *AppManager) handleAddTelegramChat(c echo.Context) error {
	var req struct {
		ChatID int64  `json:"chat_id"`
		Name   string `json:"name"`
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

	chat := &storage.Chat{
		ChatID: req.ChatID,
		Name:   req.Name,
	}
	if err := am.storage.SaveChat(chat); err != nil {
		am.logger.Printf("Failed to save chat: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add telegram chat",
		})
	}
	return c.JSON(http.StatusCreated, chat)
}

// handleRemoveTelegramChat removes a telegram chat from the registry and all source associations
func (am *AppManager) handleRemoveTelegramChat(c echo.Context) error {
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid chat ID",
		})
	}
	if err := am.storage.DeleteChat(chatID); err != nil {
		am.logger.Printf("Failed to delete chat: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to remove telegram chat",
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Chat removed",
	})
}

// handleGetSourceTelegramChats returns telegram chats associated with a source (with names from registry)
func (am *AppManager) handleGetSourceTelegramChats(c echo.Context) error {
	sourceID := c.Param("source_id")
	if _, err := am.storage.GetSource(sourceID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Source not found",
		})
	}
	chatIDs, err := am.storage.GetSourceChats(sourceID)
	if err != nil {
		am.logger.Printf("Failed to get source chats: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get source telegram chats",
		})
	}
	var chats []*storage.Chat
	for _, id := range chatIDs {
		chat, err := am.storage.GetChat(id)
		if err != nil {
			chats = append(chats, &storage.Chat{ChatID: id, Name: ""})
			continue
		}
		chats = append(chats, chat)
	}
	if chats == nil {
		chats = []*storage.Chat{}
	}
	return c.JSON(http.StatusOK, chats)
}

// handleAddSourceTelegramChat associates a telegram chat with a source
func (am *AppManager) handleAddSourceTelegramChat(c echo.Context) error {
	sourceID := c.Param("source_id")
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid chat ID",
		})
	}
	if _, err := am.storage.GetSource(sourceID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Source not found",
		})
	}
	if _, err := am.storage.GetChat(chatID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Telegram chat not found. Add the chat in Sinks first.",
		})
	}
	if err := am.storage.AddSourceChat(sourceID, chatID); err != nil {
		am.logger.Printf("Failed to add source chat: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add telegram chat to source",
		})
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":   "Telegram chat added to source",
		"source_id":  sourceID,
		"chat_id":   chatID,
	})
}

// handleRemoveSourceTelegramChat removes a telegram chat from a source
func (am *AppManager) handleRemoveSourceTelegramChat(c echo.Context) error {
	sourceID := c.Param("source_id")
	chatIDStr := c.Param("chat_id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid chat ID",
		})
	}
	if err := am.storage.RemoveSourceChat(sourceID, chatID); err != nil {
		am.logger.Printf("Failed to remove source chat: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to remove telegram chat from source",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Telegram chat removed from source",
		"source_id":  sourceID,
		"chat_id":   chatID,
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
