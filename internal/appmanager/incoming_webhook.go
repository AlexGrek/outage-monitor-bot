package appmanager

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const webhookTokenChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const webhookTokenLength = 8

// generateWebhookToken returns a short random token, checking DB for uniqueness
func (am *AppManager) generateWebhookToken() (string, error) {
	for i := 0; i < 10; i++ {
		b := make([]byte, webhookTokenLength)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		for j := range b {
			b[j] = webhookTokenChars[int(b[j])%len(webhookTokenChars)]
		}
		token := string(b)
		_, err := am.storage.GetSourceByWebhookToken(token)
		if err != nil {
			return token, nil
		}
	}
	return "", fmt.Errorf("could not generate unique webhook token")
}

// handleIncomingWebhook processes GET or POST requests to /webhooks/incoming/:token.
// No API key required. Validates optional headers/body and records heartbeat.
func (am *AppManager) handleIncomingWebhook(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing webhook token",
		})
	}

	source, err := am.storage.GetSourceByWebhookToken(token)
	if err != nil {
		am.logger.Printf("Incoming webhook: token not found: %s", token)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Webhook not found",
		})
	}

	if !source.Enabled {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"note":   "Source is paused",
		})
	}

	// Validate expected headers (JSON object: {"Header-Name": "value"})
	if source.ExpectedHeaders != "" {
		var expected map[string]string
		if err := json.Unmarshal([]byte(source.ExpectedHeaders), &expected); err != nil {
			am.logger.Printf("Incoming webhook: invalid expected_headers for source %s: %v", source.ID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Invalid source configuration",
			})
		}
		for k, v := range expected {
			got := c.Request().Header.Get(k)
			if got != v {
				am.logger.Printf("Incoming webhook: header %q mismatch for source %s", k, source.Name)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Header validation failed",
				})
			}
		}
	}

	// Validate expected content (substring in body) for POST/PUT/PATCH
	if source.ExpectedContent != "" {
		if c.Request().Body != nil {
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Failed to read body",
				})
			}
			if !strings.Contains(string(body), source.ExpectedContent) {
				am.logger.Printf("Incoming webhook: body content mismatch for source %s", source.Name)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Content validation failed",
				})
			}
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Expected content in body",
			})
		}
	}

	now := time.Now()

	// Persist heartbeat
	if err := am.storage.UpdateSourceStatus(source.ID, 1, now); err != nil {
		am.logger.Printf("Incoming webhook: failed to update source status: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to record heartbeat",
		})
	}

	// Update monitor cache so next tick sees the new last-check time
	if mon := am.botProcess.GetMonitor(); mon != nil {
		mon.RecordWebhookReceived(source.ID, now)
	}

	am.logger.Printf("Incoming webhook: heartbeat recorded for %s (token %s)", source.Name, token)

	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}
