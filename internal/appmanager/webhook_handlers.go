package appmanager

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"tg-monitor-bot/internal/storage"
)

// handleGetWebhooks returns all webhooks
func (am *AppManager) handleGetWebhooks(c echo.Context) error {
	webhooks, err := am.storage.ListWebhooks()
	if err != nil {
		am.logger.Printf("Failed to list webhooks: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list webhooks",
		})
	}

	if webhooks == nil {
		webhooks = []*storage.Webhook{}
	}

	return c.JSON(http.StatusOK, webhooks)
}

// handleCreateWebhook creates a new webhook
func (am *AppManager) handleCreateWebhook(c echo.Context) error {
	var req struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers,omitempty"`
		Enabled bool              `json:"enabled"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validation
	if req.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL is required",
		})
	}

	if req.Method == "" {
		req.Method = "POST"
	}

	// Validate HTTP method
	if req.Method != "GET" && req.Method != "POST" && req.Method != "PUT" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid HTTP method. Use GET, POST, or PUT",
		})
	}

	webhook := &storage.Webhook{
		URL:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
		Enabled: req.Enabled,
	}

	if err := am.storage.SaveWebhook(webhook); err != nil {
		am.logger.Printf("Failed to create webhook: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create webhook",
		})
	}

	return c.JSON(http.StatusCreated, webhook)
}

// handleUpdateWebhook updates a webhook
func (am *AppManager) handleUpdateWebhook(c echo.Context) error {
	webhookID := c.Param("id")

	webhook, err := am.storage.GetWebhook(webhookID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Webhook not found",
		})
	}

	var req struct {
		URL     *string            `json:"url"`
		Method  *string            `json:"method"`
		Headers map[string]string  `json:"headers,omitempty"`
		Enabled *bool              `json:"enabled"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.URL != nil {
		webhook.URL = *req.URL
	}

	if req.Method != nil {
		if *req.Method != "GET" && *req.Method != "POST" && *req.Method != "PUT" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid HTTP method. Use GET, POST, or PUT",
			})
		}
		webhook.Method = *req.Method
	}

	if len(req.Headers) > 0 {
		webhook.Headers = req.Headers
	}

	if req.Enabled != nil {
		webhook.Enabled = *req.Enabled
	}

	if err := am.storage.SaveWebhook(webhook); err != nil {
		am.logger.Printf("Failed to update webhook: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update webhook",
		})
	}

	return c.JSON(http.StatusOK, webhook)
}

// handleDeleteWebhook deletes a webhook
func (am *AppManager) handleDeleteWebhook(c echo.Context) error {
	webhookID := c.Param("id")

	// Verify webhook exists
	if _, err := am.storage.GetWebhook(webhookID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Webhook not found",
		})
	}

	// Remove associations
	sources, err := am.storage.GetWebhookSources(webhookID)
	if err == nil {
		for _, sourceID := range sources {
			am.storage.RemoveSourceWebhook(sourceID, webhookID)
		}
	}

	if err := am.storage.DeleteWebhook(webhookID); err != nil {
		am.logger.Printf("Failed to delete webhook: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete webhook",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Webhook deleted",
		"id":      webhookID,
	})
}

// handleAddSourceWebhook associates a webhook with a source
func (am *AppManager) handleAddSourceWebhook(c echo.Context) error {
	sourceID := c.Param("source_id")
	webhookID := c.Param("webhook_id")

	// Verify source exists
	if _, err := am.storage.GetSource(sourceID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Source not found",
		})
	}

	// Verify webhook exists
	if _, err := am.storage.GetWebhook(webhookID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Webhook not found",
		})
	}

	if err := am.storage.AddSourceWebhook(sourceID, webhookID); err != nil {
		am.logger.Printf("Failed to add source webhook: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add webhook to source",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message":    "Webhook added to source",
		"source_id":  sourceID,
		"webhook_id": webhookID,
	})
}

// handleRemoveSourceWebhook removes a webhook association from a source
func (am *AppManager) handleRemoveSourceWebhook(c echo.Context) error {
	sourceID := c.Param("source_id")
	webhookID := c.Param("webhook_id")

	if err := am.storage.RemoveSourceWebhook(sourceID, webhookID); err != nil {
		am.logger.Printf("Failed to remove source webhook: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to remove webhook from source",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message":    "Webhook removed from source",
		"source_id":  sourceID,
		"webhook_id": webhookID,
	})
}

// handleGetSourceWebhooks returns all webhooks for a source
func (am *AppManager) handleGetSourceWebhooks(c echo.Context) error {
	sourceID := c.Param("source_id")

	// Verify source exists
	if _, err := am.storage.GetSource(sourceID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Source not found",
		})
	}

	webhooks, err := am.storage.GetSourceWebhooks(sourceID)
	if err != nil {
		am.logger.Printf("Failed to get source webhooks: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get source webhooks",
		})
	}

	if webhooks == nil {
		webhooks = []*storage.Webhook{}
	}

	return c.JSON(http.StatusOK, webhooks)
}
