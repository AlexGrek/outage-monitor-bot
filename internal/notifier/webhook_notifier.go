package notifier

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"tg-monitor-bot/internal/storage"
)

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	Source     *SourceData     `json:"source"`
	StatusChange *StatusChangeData `json:"status_change"`
	Timestamp  string          `json:"timestamp"`
}

// SourceData represents source information in webhook payload
type SourceData struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Target         string `json:"target"`
	CurrentStatus  int    `json:"current_status"`
	LastCheckTime  string `json:"last_check_time"`
	LastChangeTime string `json:"last_change_time"`
}

// StatusChangeData represents status change information in webhook payload
type StatusChangeData struct {
	ID         string `json:"id"`
	OldStatus  int    `json:"old_status"`
	NewStatus  int    `json:"new_status"`
	DurationMs int64  `json:"duration_ms"`
	Timestamp  string `json:"timestamp"`
}

// WebhookNotifier sends webhooks on status changes
type WebhookNotifier struct {
	storage *storage.BoltDB
	logger  *log.Logger
	client  *http.Client
}

// NewWebhookNotifier creates a new webhook notifier
func NewWebhookNotifier(db *storage.BoltDB) *WebhookNotifier {
	return &WebhookNotifier{
		storage: db,
		logger:  log.New(log.Writer(), "[WEBHOOK_NOTIFIER] ", log.LstdFlags),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// OnStatusChange implements the StatusChangeCallback interface
// It sends webhooks to all configured webhooks for the source
func (wn *WebhookNotifier) OnStatusChange(source *storage.Source, change *storage.StatusChange) {
	// Get webhooks for this source
	webhooks, err := wn.storage.GetSourceWebhooks(source.ID)
	if err != nil {
		wn.logger.Printf("Failed to get webhooks for source %s: %v", source.ID, err)
		return
	}

	if len(webhooks) == 0 {
		return // No webhooks configured for this source
	}

	// Build payload
	payload := wn.buildPayload(source, change)

	// Send to each webhook
	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue // Skip disabled webhooks
		}

		wn.logger.Printf("Sending webhook to %s for source %s (status: %d→%d)",
			webhook.URL, source.Name, change.OldStatus, change.NewStatus)

		go wn.sendWebhook(webhook, payload)
	}
}

// sendWebhook sends a single webhook request
func (wn *WebhookNotifier) sendWebhook(webhook *storage.Webhook, payload WebhookPayload) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		wn.logger.Printf("Failed to marshal webhook payload: %v", err)
		return
	}

	// Create request
	req, err := http.NewRequest(webhook.Method, webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		wn.logger.Printf("Failed to create webhook request: %v", err)
		return
	}

	// Set default content type
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := wn.client.Do(req)
	if err != nil {
		wn.logger.Printf("Failed to send webhook to %s: %v", webhook.URL, err)
		return
	}
	defer resp.Body.Close()

	// Read response body (for debugging/logging)
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		wn.logger.Printf("Webhook sent successfully to %s (status: %d)", webhook.URL, resp.StatusCode)
		// Update last triggered timestamp
		wn.storage.UpdateWebhookLastTriggered(webhook.ID)
	} else {
		wn.logger.Printf("Webhook request failed for %s (status: %d, body: %s)",
			webhook.URL, resp.StatusCode, string(body))
	}
}

// buildPayload creates a webhook payload from source and status change
func (wn *WebhookNotifier) buildPayload(source *storage.Source, change *storage.StatusChange) WebhookPayload {
	return WebhookPayload{
		Source: &SourceData{
			ID:             source.ID,
			Name:           source.Name,
			Type:           source.Type,
			Target:         source.Target,
			CurrentStatus:  source.CurrentStatus,
			LastCheckTime:  source.LastCheckTime.Format(time.RFC3339),
			LastChangeTime: source.LastChangeTime.Format(time.RFC3339),
		},
		StatusChange: &StatusChangeData{
			ID:         change.ID,
			OldStatus:  change.OldStatus,
			NewStatus:  change.NewStatus,
			DurationMs: change.DurationMs,
			Timestamp:  change.Timestamp.Format(time.RFC3339),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
