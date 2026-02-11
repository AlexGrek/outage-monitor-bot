package appmanager

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tg-monitor-bot/internal/storage"
)

// CreateSourceRequest is the request body for creating a source
type CreateSourceRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`           // "ping" or "http"
	Target        string `json:"target"`
	CheckInterval string `json:"check_interval"` // e.g. "30s", "1m"
}

// UpdateSourceRequest is the request body for updating a source
type UpdateSourceRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Target        string `json:"target"`
	CheckInterval string `json:"check_interval"`
	Enabled       bool   `json:"enabled"`
}

// handleGetSources returns all sources
func (am *AppManager) handleGetSources(c echo.Context) error {
	monitor := am.botProcess.GetMonitor()
	if monitor == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Monitor not available",
		})
	}

	sources, err := monitor.GetAllSources()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Ensure we return an empty array instead of null when no sources
	if sources == nil {
		sources = []*storage.Source{}
	}

	return c.JSON(http.StatusOK, sources)
}

// handleCreateSource creates a new monitoring source
func (am *AppManager) handleCreateSource(c echo.Context) error {
	var req CreateSourceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}
	if req.Type != "ping" && req.Type != "http" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Type must be 'ping' or 'http'",
		})
	}
	if req.Target == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Target is required",
		})
	}

	// Parse check interval
	checkInterval, err := time.ParseDuration(req.CheckInterval)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid check_interval format (use '30s', '1m', etc.)",
		})
	}

	// Create source
	source := &storage.Source{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Type:           req.Type,
		Target:         req.Target,
		CheckInterval:  checkInterval,
		CurrentStatus:  -1, // Unknown initially
		Enabled:        true,
		CreatedAt:      time.Now(),
		LastCheckTime:  time.Time{},
		LastChangeTime: time.Time{},
	}

	// Save to database
	if err := am.storage.SaveSource(source); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Add to monitor
	monitor := am.botProcess.GetMonitor()
	if monitor != nil {
		ctx := am.botProcess.GetContext()
		if err := monitor.AddSource(ctx, source); err != nil {
			am.logger.Printf("Warning: Failed to add source to monitor: %v", err)
		}
	}

	am.logger.Printf("Created source via API: %s (%s)", source.Name, source.ID)

	return c.JSON(http.StatusCreated, source)
}

// handleUpdateSource updates an existing source
func (am *AppManager) handleUpdateSource(c echo.Context) error {
	sourceID := c.Param("id")

	var req UpdateSourceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get existing source
	source, err := am.storage.GetSource(sourceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Source not found",
		})
	}

	// Validate input
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name is required",
		})
	}
	if req.Type != "ping" && req.Type != "http" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Type must be 'ping' or 'http'",
		})
	}
	if req.Target == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Target is required",
		})
	}

	// Parse check interval
	checkInterval, err := time.ParseDuration(req.CheckInterval)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid check_interval format",
		})
	}

	// Update source
	source.Name = req.Name
	source.Type = req.Type
	source.Target = req.Target
	source.CheckInterval = checkInterval
	source.Enabled = req.Enabled

	// Save to database
	if err := am.storage.SaveSource(source); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Update in monitor (remove and re-add)
	monitor := am.botProcess.GetMonitor()
	if monitor != nil {
		monitor.RemoveSource(sourceID)
		if req.Enabled {
			ctx := am.botProcess.GetContext()
			if err := monitor.AddSource(ctx, source); err != nil {
				am.logger.Printf("Warning: Failed to update source in monitor: %v", err)
			}
		}
	}

	am.logger.Printf("Updated source via API: %s (%s)", source.Name, source.ID)

	return c.JSON(http.StatusOK, source)
}

// handleDeleteSource deletes a source
func (am *AppManager) handleDeleteSource(c echo.Context) error {
	sourceID := c.Param("id")

	// Get source to log name before deletion
	source, err := am.storage.GetSource(sourceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Source not found",
		})
	}

	// Remove from monitor
	monitor := am.botProcess.GetMonitor()
	if monitor != nil {
		if err := monitor.RemoveSource(sourceID); err != nil {
			am.logger.Printf("Warning: Failed to remove source from monitor: %v", err)
		}
	}

	// Delete from database
	if err := am.storage.DeleteSource(sourceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Deleted source via API: %s (%s)", source.Name, source.ID)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Source deleted successfully",
		"id":      sourceID,
	})
}

// handlePauseSource pauses monitoring for a source
func (am *AppManager) handlePauseSource(c echo.Context) error {
	sourceID := c.Param("id")

	monitor := am.botProcess.GetMonitor()
	if monitor == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Monitor not available",
		})
	}

	if err := monitor.PauseSource(sourceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Paused source via API: %s", sourceID)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Source paused",
		"id":      sourceID,
	})
}

// handleResumeSource resumes monitoring for a source
func (am *AppManager) handleResumeSource(c echo.Context) error {
	sourceID := c.Param("id")

	monitor := am.botProcess.GetMonitor()
	if monitor == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Monitor not available",
		})
	}

	if err := monitor.ResumeSource(sourceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	am.logger.Printf("Resumed source via API: %s", sourceID)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Source resumed",
		"id":      sourceID,
	})
}
