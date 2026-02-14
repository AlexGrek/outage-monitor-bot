package appmanager

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"tg-monitor-bot/internal/storage"
)

// StatusChangeEventResponse represents a status change event with source information
type StatusChangeEventResponse struct {
	ID          string `json:"id"`
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name"`
	OldStatus   int    `json:"old_status"`
	NewStatus   int    `json:"new_status"`
	DurationMs  int64  `json:"duration_ms"`
	Timestamp   string `json:"timestamp"`
}

// handleGetEvents returns status change events
func (am *AppManager) handleGetEvents(c echo.Context) error {
	// Parse query parameters
	sourceID := c.QueryParam("source_id")
	limitStr := c.QueryParam("limit")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	// Get status changes from storage
	var statusChanges []*storage.StatusChange
	var err error

	if sourceID != "" {
		// Get changes for specific source
		statusChanges, err = am.storage.GetStatusChanges(sourceID, limit)
	} else {
		// Get recent changes across all sources
		statusChanges, err = am.storage.GetRecentChanges(limit)
	}

	if err != nil {
		am.logger.Printf("Failed to get status changes: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get events",
		})
	}

	// Convert to response format with source information
	events := make([]StatusChangeEventResponse, 0)
	for _, change := range statusChanges {
		source, err := am.storage.GetSource(change.SourceID)
		if err != nil {
			am.logger.Printf("Failed to get source %s: %v", change.SourceID, err)
			continue
		}

		event := StatusChangeEventResponse{
			ID:         change.ID,
			SourceID:   change.SourceID,
			SourceName: source.Name,
			OldStatus:  change.OldStatus,
			NewStatus:  change.NewStatus,
			DurationMs: change.DurationMs,
			Timestamp:  change.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		}
		events = append(events, event)
	}

	return c.JSON(http.StatusOK, events)
}
