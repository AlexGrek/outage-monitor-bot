package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// Source represents a monitoring source
type Source struct {
	ID                    string        `msgpack:"id" json:"id"`
	Name                  string        `msgpack:"name" json:"name"`
	Type                  string        `msgpack:"type" json:"type"` // "ping", "http", or "webhook"
	Target                string        `msgpack:"target" json:"target"`
	CheckInterval         time.Duration `msgpack:"check_interval" json:"check_interval"`
	CurrentStatus         int           `msgpack:"current_status" json:"current_status"`     // 1 (online) or 0 (offline)
	LastCheckTime         time.Time     `msgpack:"last_check_time" json:"last_check_time"`
	LastChangeTime        time.Time     `msgpack:"last_change_time" json:"last_change_time"` // When status last changed
	Enabled               bool          `msgpack:"enabled" json:"enabled"`
	CreatedAt             time.Time     `msgpack:"created_at" json:"created_at"`
	// Webhook (incoming) source only
	WebhookToken          string  `msgpack:"webhook_token" json:"webhook_token,omitempty"`
	GracePeriodMultiplier float64 `msgpack:"grace_period_multiplier" json:"grace_period_multiplier,omitempty"`
	ExpectedHeaders       string  `msgpack:"expected_headers" json:"expected_headers,omitempty"` // JSON object: {"Header-Name":"value"}
	ExpectedContent       string  `msgpack:"expected_content" json:"expected_content,omitempty"`
}

// SaveSource stores a source in the database
func (b *BoltDB) SaveSource(source *Source) error {
	if source.ID == "" {
		source.ID = uuid.New().String()
	}

	if source.CreatedAt.IsZero() {
		source.CreatedAt = time.Now()
	}

	if source.LastChangeTime.IsZero() {
		source.LastChangeTime = time.Now()
	}

	data, err := msgpack.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		if err := bucket.Put([]byte(source.ID), data); err != nil {
			return fmt.Errorf("failed to save source: %w", err)
		}

		b.logger.Printf("Saved source: %s (%s %s)", source.Name, source.Type, source.Target)
		return nil
	})
}

// GetSource retrieves a source by ID
func (b *BoltDB) GetSource(id string) (*Source, error) {
	var source *Source

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("source not found")
		}

		source = &Source{}
		if err := msgpack.Unmarshal(data, source); err != nil {
			return fmt.Errorf("failed to unmarshal source: %w", err)
		}

		return nil
	})

	return source, err
}

// GetSourceByName retrieves a source by name
func (b *BoltDB) GetSourceByName(name string) (*Source, error) {
	var source *Source

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var s Source
			if err := msgpack.Unmarshal(v, &s); err != nil {
				continue
			}

			if s.Name == name {
				source = &s
				return nil
			}
		}

		return fmt.Errorf("source not found")
	})

	return source, err
}

// GetSourceByWebhookToken retrieves a webhook source by its incoming webhook token
func (b *BoltDB) GetSourceByWebhookToken(token string) (*Source, error) {
	if token == "" {
		return nil, fmt.Errorf("webhook token is empty")
	}

	var source *Source
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var s Source
			if err := msgpack.Unmarshal(v, &s); err != nil {
				continue
			}
			if s.Type == "webhook" && s.WebhookToken == token {
				source = &s
				return nil
			}
		}

		return fmt.Errorf("source not found")
	})

	return source, err
}

// GetAllSources retrieves all sources
func (b *BoltDB) GetAllSources() ([]*Source, error) {
	var sources []*Source

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var source Source
			if err := msgpack.Unmarshal(v, &source); err != nil {
				b.logger.Printf("Failed to unmarshal source: %v", err)
				continue
			}

			sources = append(sources, &source)
		}

		return nil
	})

	return sources, err
}

// GetEnabledSources retrieves all enabled sources
func (b *BoltDB) GetEnabledSources() ([]*Source, error) {
	allSources, err := b.GetAllSources()
	if err != nil {
		return nil, err
	}

	var enabled []*Source
	for _, source := range allSources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}

	return enabled, nil
}

// DeleteSource removes a source from the database
func (b *BoltDB) DeleteSource(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		if err := bucket.Delete([]byte(id)); err != nil {
			return fmt.Errorf("failed to delete source: %w", err)
		}

		b.logger.Printf("Deleted source: %s", id)
		return nil
	})
}

// UpdateSourceStatus updates the status of a source
func (b *BoltDB) UpdateSourceStatus(id string, status int, checkTime time.Time) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("source not found")
		}

		var source Source
		if err := msgpack.Unmarshal(data, &source); err != nil {
			return fmt.Errorf("failed to unmarshal source: %w", err)
		}

		oldStatus := source.CurrentStatus
		source.CurrentStatus = status
		source.LastCheckTime = checkTime

		if status != oldStatus {
			source.LastChangeTime = checkTime
		}

		newData, err := msgpack.Marshal(&source)
		if err != nil {
			return fmt.Errorf("failed to marshal source: %w", err)
		}

		return bucket.Put([]byte(id), newData)
	})
}

// UpdateSourceCurrentStatus updates only CurrentStatus and LastChangeTime without touching LastCheckTime.
// Use for webhook sources where LastCheckTime tracks the last heartbeat received, not the last monitor tick.
func (b *BoltDB) UpdateSourceCurrentStatus(id string, status int, changeTime time.Time) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("source not found")
		}

		var source Source
		if err := msgpack.Unmarshal(data, &source); err != nil {
			return fmt.Errorf("failed to unmarshal source: %w", err)
		}

		oldStatus := source.CurrentStatus
		source.CurrentStatus = status
		if status != oldStatus {
			source.LastChangeTime = changeTime
		}
		// LastCheckTime is intentionally not updated here

		newData, err := msgpack.Marshal(&source)
		if err != nil {
			return fmt.Errorf("failed to marshal source: %w", err)
		}

		return bucket.Put([]byte(id), newData)
	})
}

// UpdateSource updates an entire source
func (b *BoltDB) UpdateSource(source *Source) error {
	data, err := msgpack.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal source: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourcesBucket))
		if bucket == nil {
			return fmt.Errorf("sources bucket not found")
		}

		return bucket.Put([]byte(source.ID), data)
	})
}
