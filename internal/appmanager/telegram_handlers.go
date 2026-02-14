package appmanager

import (
	"net/http"

	"github.com/labstack/echo/v4"
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
