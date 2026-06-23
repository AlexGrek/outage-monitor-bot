package appmanager

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tg-monitor-bot/internal/storage"
)

// CountdownRequest is the request body for creating/updating a countdown
type CountdownRequest struct {
	Name            string   `json:"name"`
	TargetDate      string   `json:"target_date"`      // "2006-01-02"
	NotifyTime      string   `json:"notify_time"`      // "15:04"
	Timezone        string   `json:"timezone"`         // IANA name
	MessageTemplate string   `json:"message_template"` // "{}" replaced with days left
	Enabled         *bool    `json:"enabled,omitempty"`
	ChatIDs         []int64  `json:"chat_ids"`
	WebhookIDs      []string `json:"webhook_ids"`
}

// validateCountdownRequest validates the shared countdown fields and returns the
// resolved timezone location on success, or an error message on failure.
func validateCountdownRequest(req *CountdownRequest) (*time.Location, string) {
	if req.Name == "" {
		return nil, "Name is required"
	}
	if _, err := time.Parse("2006-01-02", req.TargetDate); err != nil {
		return nil, "Invalid target_date (use YYYY-MM-DD)"
	}
	if _, err := time.Parse("15:04", req.NotifyTime); err != nil {
		return nil, "Invalid notify_time (use 24h HH:MM, e.g. 09:00)"
	}
	if req.Timezone == "" {
		return nil, "Timezone is required (e.g. Europe/Kyiv)"
	}
	loc, err := time.LoadLocation(req.Timezone)
	if err != nil {
		return nil, "Invalid timezone (use an IANA name like Europe/Kyiv or UTC)"
	}
	if req.MessageTemplate == "" {
		return nil, "Message template is required"
	}
	return loc, ""
}

// handleGetCountdowns returns all countdowns
func (am *AppManager) handleGetCountdowns(c echo.Context) error {
	countdowns, err := am.storage.ListCountdowns()
	if err != nil {
		am.logger.Printf("Failed to list countdowns: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list countdowns",
		})
	}
	if countdowns == nil {
		countdowns = []*storage.Countdown{}
	}
	return c.JSON(http.StatusOK, countdowns)
}

// handleCreateCountdown creates a new countdown
func (am *AppManager) handleCreateCountdown(c echo.Context) error {
	var req CountdownRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	loc, errMsg := validateCountdownRequest(&req)
	if errMsg != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": errMsg})
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	countdown := &storage.Countdown{
		ID:              uuid.New().String(),
		Name:            req.Name,
		TargetDate:      req.TargetDate,
		NotifyTime:      req.NotifyTime,
		Timezone:        req.Timezone,
		MessageTemplate: req.MessageTemplate,
		Enabled:         enabled,
		ChatIDs:         req.ChatIDs,
		WebhookIDs:      req.WebhookIDs,
		CreatedAt:       time.Now(),
	}

	// If today's notify time has already passed, record it as "sent" so the
	// countdown does not fire immediately on creation; it starts tomorrow.
	countdown.LastSentDate = initialLastSentDate(countdown, loc, time.Now())

	if err := am.storage.SaveCountdown(countdown); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Created countdown via API: %s (%s)", countdown.Name, countdown.ID)
	return c.JSON(http.StatusCreated, countdown)
}

// handleUpdateCountdown updates an existing countdown
func (am *AppManager) handleUpdateCountdown(c echo.Context) error {
	id := c.Param("id")

	existing, err := am.storage.GetCountdown(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Countdown not found",
		})
	}

	var req CountdownRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	loc, errMsg := validateCountdownRequest(&req)
	if errMsg != "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": errMsg})
	}

	// If the schedule (date/time/timezone) changed, reset the daily-send guard so
	// the new schedule is honoured (and won't immediately re-fire if it already
	// passed today).
	scheduleChanged := existing.TargetDate != req.TargetDate ||
		existing.NotifyTime != req.NotifyTime ||
		existing.Timezone != req.Timezone

	existing.Name = req.Name
	existing.TargetDate = req.TargetDate
	existing.NotifyTime = req.NotifyTime
	existing.Timezone = req.Timezone
	existing.MessageTemplate = req.MessageTemplate
	existing.ChatIDs = req.ChatIDs
	existing.WebhookIDs = req.WebhookIDs
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if scheduleChanged {
		existing.LastSentDate = initialLastSentDate(existing, loc, time.Now())
	}

	if err := am.storage.SaveCountdown(existing); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Updated countdown via API: %s (%s)", existing.Name, existing.ID)
	return c.JSON(http.StatusOK, existing)
}

// handleDeleteCountdown deletes a countdown
func (am *AppManager) handleDeleteCountdown(c echo.Context) error {
	id := c.Param("id")

	if _, err := am.storage.GetCountdown(id); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Countdown not found",
		})
	}

	if err := am.storage.DeleteCountdown(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Deleted countdown via API: %s", id)
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Countdown deleted successfully",
		"id":      id,
	})
}

// handleTestCountdown sends a countdown notification immediately to its sinks,
// regardless of schedule and without affecting the daily-send guard.
func (am *AppManager) handleTestCountdown(c echo.Context) error {
	id := c.Param("id")

	countdown, err := am.storage.GetCountdown(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Countdown not found",
		})
	}

	loc, err := time.LoadLocation(countdown.Timezone)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Countdown has an invalid timezone",
		})
	}

	daysLeft, err := computeDaysLeft(countdown.TargetDate, loc, time.Now())
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Countdown has an invalid target date",
		})
	}

	scheduler := am.botProcess.GetCountdownScheduler()
	if scheduler == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Scheduler not available (bot not running)",
		})
	}

	go scheduler.dispatch(countdown, daysLeft)

	am.logger.Printf("Sent test countdown notification: %s (%d days left)", countdown.Name, daysLeft)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Test countdown notification sent",
		"id":        id,
		"days_left": daysLeft,
		"sent_at":   time.Now(),
	})
}

// initialLastSentDate returns the date to seed LastSentDate with so a countdown
// does not fire on the same day it is created/rescheduled once the notify time
// has already passed. Returns "" when the countdown should still fire today.
func initialLastSentDate(countdown *storage.Countdown, loc *time.Location, now time.Time) string {
	nowLoc := now.In(loc)
	nt, err := time.Parse("15:04", countdown.NotifyTime)
	if err != nil {
		return ""
	}
	scheduled := time.Date(nowLoc.Year(), nowLoc.Month(), nowLoc.Day(), nt.Hour(), nt.Minute(), 0, 0, loc)
	if !nowLoc.Before(scheduled) {
		return nowLoc.Format("2006-01-02")
	}
	return ""
}
