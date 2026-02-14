package monitor

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"tg-monitor-bot/internal/config"
	"tg-monitor-bot/internal/storage"
)

// StatusChangeCallback is called when a source's status changes
type StatusChangeCallback func(*storage.Source, *storage.StatusChange)

// Monitor handles all monitoring operations
type Monitor struct {
	storage         *storage.BoltDB
	config          *config.Config
	client          *http.Client
	logger          *log.Logger
	onStatusChange  StatusChangeCallback
	activeMonitors  map[string]context.CancelFunc // sourceID -> cancel function
	monitorsMu      sync.RWMutex
	sources         map[string]*storage.Source // sourceID -> source (in-memory cache)
	sourcesMu       sync.RWMutex
}

// New creates a new Monitor instance
func New(db *storage.BoltDB, cfg *config.Config, callback StatusChangeCallback) *Monitor {
	return &Monitor{
		storage:        db,
		config:         cfg,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
		},
		logger:         log.New(log.Writer(), "[MONITOR] ", log.LstdFlags),
		onStatusChange: callback,
		activeMonitors: make(map[string]context.CancelFunc),
		sources:        make(map[string]*storage.Source),
	}
}

// Start begins monitoring all enabled sources from the database
func (m *Monitor) Start(ctx context.Context) error {
	m.logger.Println("Monitor starting...")

	// Load all enabled sources from database
	sources, err := m.storage.GetEnabledSources()
	if err != nil {
		return fmt.Errorf("failed to load sources: %w", err)
	}

	m.logger.Printf("Loaded %d enabled sources from database", len(sources))

	// Debug: log all source IDs to detect duplicates
	if len(sources) > 0 {
		m.logger.Println("Source IDs from database:")
		sourceIDCount := make(map[string]int)
		for _, source := range sources {
			sourceIDCount[source.ID]++
			m.logger.Printf("  - %s (ID: %s)", source.Name, source.ID)
		}
		// Check for duplicates
		for id, count := range sourceIDCount {
			if count > 1 {
				m.logger.Printf("⚠️  WARNING: Source ID %s appears %d times in database query result!", id, count)
			}
		}
	} else {
		m.logger.Println("No sources to monitor")
	}

	// Start monitoring each source
	successCount := 0
	for _, source := range sources {
		m.logger.Printf("Adding source to monitor: %s (ID: %s)", source.Name, source.ID)
		if err := m.AddSource(ctx, source); err != nil {
			m.logger.Printf("❌ Failed to start monitoring source %s: %v", source.Name, err)
		} else {
			successCount++
		}
	}

	m.logger.Printf("✅ Monitor started successfully with %d/%d sources active", successCount, len(sources))
	return nil
}

// AddSource starts monitoring a new source
func (m *Monitor) AddSource(ctx context.Context, source *storage.Source) error {
	m.monitorsMu.Lock()
	defer m.monitorsMu.Unlock()

	// Check if already monitoring
	if _, exists := m.activeMonitors[source.ID]; exists {
		m.logger.Printf("⚠️  Source %s (ID: %s) already being monitored - skipping", source.Name, source.ID)
		return fmt.Errorf("source already being monitored")
	}

	// Add to cache
	m.sourcesMu.Lock()
	m.sources[source.ID] = source
	m.sourcesMu.Unlock()

	// Create context for this source
	sourceCtx, cancel := context.WithCancel(ctx)
	m.activeMonitors[source.ID] = cancel

	m.logger.Printf("Starting goroutine for: %s (ID: %s, type: %s, target: %s, interval: %v)",
		source.Name, source.ID, source.Type, source.Target, source.CheckInterval)

	// Start monitoring goroutine
	go m.monitorSource(sourceCtx, source)

	m.logger.Printf("✅ Monitoring active for: %s (total active: %d)", source.Name, len(m.activeMonitors))

	return nil
}

// RemoveSource stops monitoring a source
func (m *Monitor) RemoveSource(sourceID string) error {
	m.monitorsMu.Lock()
	defer m.monitorsMu.Unlock()

	cancel, exists := m.activeMonitors[sourceID]
	if !exists {
		m.logger.Printf("⚠️  Cannot remove source %s - not being monitored", sourceID)
		return fmt.Errorf("source not being monitored")
	}

	// Get source name before removing
	m.sourcesMu.RLock()
	sourceName := "Unknown"
	if source, ok := m.sources[sourceID]; ok {
		sourceName = source.Name
	}
	m.sourcesMu.RUnlock()

	m.logger.Printf("Stopping monitor for: %s (ID: %s)", sourceName, sourceID)

	// Stop the monitoring goroutine
	cancel()
	delete(m.activeMonitors, sourceID)

	// Remove from cache
	m.sourcesMu.Lock()
	delete(m.sources, sourceID)
	m.sourcesMu.Unlock()

	m.logger.Printf("✅ Stopped monitoring: %s (total active: %d)", sourceName, len(m.activeMonitors))
	return nil
}

// PauseSource temporarily disables monitoring for a source
func (m *Monitor) PauseSource(sourceID string) error {
	m.sourcesMu.Lock()
	defer m.sourcesMu.Unlock()

	source, exists := m.sources[sourceID]
	if !exists {
		// Source not in cache, try loading from database
		dbSource, err := m.storage.GetSource(sourceID)
		if err != nil {
			return fmt.Errorf("source not found")
		}
		source = dbSource
	}

	source.Enabled = false
	if err := m.storage.UpdateSource(source); err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}

	m.logger.Printf("Paused source: %s", source.Name)
	return nil
}

// ResumeSource re-enables monitoring for a source
func (m *Monitor) ResumeSource(sourceID string) error {
	m.sourcesMu.Lock()
	defer m.sourcesMu.Unlock()

	source, exists := m.sources[sourceID]
	if !exists {
		// Source not in cache, try loading from database
		dbSource, err := m.storage.GetSource(sourceID)
		if err != nil {
			return fmt.Errorf("source not found")
		}
		source = dbSource
	}

	source.Enabled = true
	if err := m.storage.UpdateSource(source); err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}

	m.logger.Printf("Resumed source: %s", source.Name)
	return nil
}

// CheckSource performs a single check of a source and returns the status
func (m *Monitor) CheckSource(source *storage.Source) int {
	switch source.Type {
	case "ping":
		return m.PingTarget(source.Target)
	case "http":
		return m.CheckHTTP(source.Target)
	case "webhook":
		return m.checkWebhookSource(source)
	default:
		m.logger.Printf("Unknown source type: %s", source.Type)
		return 0
	}
}

// checkWebhookSource returns 1 if last heartbeat was within grace period, 0 otherwise
func (m *Monitor) checkWebhookSource(source *storage.Source) int {
	if source.LastCheckTime.IsZero() {
		m.logger.Printf("Webhook check %s: OFFLINE (no heartbeat yet)", source.Name)
		return 0
	}
	mult := source.GracePeriodMultiplier
	if mult <= 0 {
		mult = 2.5
	}
	graceDuration := time.Duration(float64(source.CheckInterval) * mult)
	deadline := source.LastCheckTime.Add(graceDuration)
	if time.Now().After(deadline) {
		m.logger.Printf("Webhook check %s: OFFLINE (last heartbeat %v ago, grace %v)", source.Name, time.Since(source.LastCheckTime).Round(time.Second), graceDuration.Round(time.Second))
		return 0
	}
	m.logger.Printf("Webhook check %s: ONLINE (heartbeat within grace period)", source.Name)
	return 1
}

// RecordWebhookReceived updates the in-memory source after an incoming webhook heartbeat.
// Call this after persisting via storage.UpdateSourceStatus so the next tick uses the new LastCheckTime.
func (m *Monitor) RecordWebhookReceived(sourceID string, receivedAt time.Time) {
	m.sourcesMu.Lock()
	defer m.sourcesMu.Unlock()
	source, exists := m.sources[sourceID]
	if !exists {
		return
	}
	source.LastCheckTime = receivedAt
	source.CurrentStatus = 1
	m.sources[sourceID] = source
}

// monitorSource continuously monitors a single source
func (m *Monitor) monitorSource(ctx context.Context, source *storage.Source) {
	m.logger.Printf("🔵 Goroutine started for: %s (ID: %s)", source.Name, source.ID)

	ticker := time.NewTicker(source.CheckInterval)
	defer ticker.Stop()

	// Perform initial check immediately
	m.logger.Printf("⏱️  Initial check for: %s", source.Name)
	m.performCheck(source)

	for {
		select {
		case <-ctx.Done():
			m.logger.Printf("🔴 Goroutine stopping for: %s (ID: %s)", source.Name, source.ID)
			return
		case <-ticker.C:
			m.logger.Printf("⏱️  Scheduled check for: %s", source.Name)
			m.performCheck(source)
		}
	}
}

// performCheck checks a source and handles status changes
func (m *Monitor) performCheck(source *storage.Source) {
	// Skip if disabled
	if !source.Enabled {
		return
	}

	checkTime := time.Now()
	newStatus := m.CheckSource(source)

	// Update last check time (for ping/http; webhook uses LastCheckTime as last heartbeat received)
	if source.Type != "webhook" {
		source.LastCheckTime = checkTime
	}

	// Check if status changed
	if newStatus != source.CurrentStatus {
		m.logger.Printf("Status change detected for %s: %d → %d", source.Name, source.CurrentStatus, newStatus)

		// Calculate duration since last change
		duration := checkTime.Sub(source.LastChangeTime)

		// Create status change record
		change := &storage.StatusChange{
			SourceID:   source.ID,
			OldStatus:  source.CurrentStatus,
			NewStatus:  newStatus,
			Timestamp:  checkTime,
			DurationMs: duration.Milliseconds(),
		}

		// Save status change to database immediately
		if err := m.storage.SaveStatusChange(change); err != nil {
			m.logger.Printf("Failed to save status change: %v", err)
		}

		// Update source status in database
		if err := m.storage.UpdateSourceStatus(source.ID, newStatus, checkTime); err != nil {
			m.logger.Printf("Failed to update source status: %v", err)
		}

		// Update in-memory source
		source.CurrentStatus = newStatus
		source.LastChangeTime = checkTime

		// Update cache
		m.sourcesMu.Lock()
		m.sources[source.ID] = source
		m.sourcesMu.Unlock()

		// Trigger notification callback
		if m.onStatusChange != nil {
			go m.onStatusChange(source, change)
		}
	} else {
		// No status change, just update check time in database
		if err := m.storage.UpdateSourceStatus(source.ID, source.CurrentStatus, checkTime); err != nil {
			m.logger.Printf("Failed to update check time: %v", err)
		}
	}
}

// CheckHTTP performs an HTTP request and returns binary status
func (m *Monitor) CheckHTTP(url string) int {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.HTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		m.logger.Printf("HTTP check failed for %s: %v", url, err)
		return 0
	}

	resp, err := m.client.Do(req)
	if err != nil {
		m.logger.Printf("HTTP check failed for %s: %v", url, err)
		return 0
	}
	defer resp.Body.Close()

	// Drain and close body
	io.Copy(io.Discard, resp.Body)

	// Online if status code is 2xx or 3xx
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		m.logger.Printf("HTTP check %s: ONLINE (status %d)", url, resp.StatusCode)
		return 1
	}

	m.logger.Printf("HTTP check %s: OFFLINE (status %d)", url, resp.StatusCode)
	return 0
}

// GetSource retrieves a source from the cache or database
func (m *Monitor) GetSource(sourceID string) (*storage.Source, error) {
	m.sourcesMu.RLock()
	source, exists := m.sources[sourceID]
	m.sourcesMu.RUnlock()

	if exists {
		return source, nil
	}

	// Not in cache, load from database
	return m.storage.GetSource(sourceID)
}

// GetAllSources retrieves all sources
func (m *Monitor) GetAllSources() ([]*storage.Source, error) {
	return m.storage.GetAllSources()
}
