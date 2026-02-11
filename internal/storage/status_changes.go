package storage

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// StatusChange represents a status change event (time-series data)
type StatusChange struct {
	ID         string    `msgpack:"id"`
	SourceID   string    `msgpack:"source_id"`
	OldStatus  int       `msgpack:"old_status"`
	NewStatus  int       `msgpack:"new_status"`
	Timestamp  time.Time `msgpack:"timestamp"`
	DurationMs int64     `msgpack:"duration_ms"` // Duration since last change in milliseconds
}

// makeStatusChangeKey creates a sortable key from source ID and timestamp
func makeStatusChangeKey(sourceID string, timestamp time.Time) []byte {
	// Format: sourceID + ":" + timestamp (nanoseconds as uint64)
	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, uint64(timestamp.UnixNano()))
	return append([]byte(sourceID+":"), tsBytes...)
}

// SaveStatusChange stores a status change in the database
func (b *BoltDB) SaveStatusChange(change *StatusChange) error {
	if change.ID == "" {
		change.ID = uuid.New().String()
	}

	if change.Timestamp.IsZero() {
		change.Timestamp = time.Now()
	}

	data, err := msgpack.Marshal(change)
	if err != nil {
		return fmt.Errorf("failed to marshal status change: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(statusChangesBucket))
		if bucket == nil {
			return fmt.Errorf("status_changes bucket not found")
		}

		key := makeStatusChangeKey(change.SourceID, change.Timestamp)

		if err := bucket.Put(key, data); err != nil {
			return fmt.Errorf("failed to save status change: %w", err)
		}

		b.logger.Printf("Saved status change: source=%s, %d→%d, duration=%dms",
			change.SourceID, change.OldStatus, change.NewStatus, change.DurationMs)
		return nil
	})
}

// GetStatusChanges retrieves the latest N status changes for a specific source
func (b *BoltDB) GetStatusChanges(sourceID string, limit int) ([]*StatusChange, error) {
	var changes []*StatusChange

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(statusChangesBucket))
		if bucket == nil {
			return fmt.Errorf("status_changes bucket not found")
		}

		c := bucket.Cursor()
		prefix := []byte(sourceID + ":")

		// Collect all keys for this source
		var keys [][]byte
		for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, _ = c.Next() {
			keys = append(keys, append([]byte(nil), k...))
		}

		// Iterate in reverse to get newest first
		count := 0
		for i := len(keys) - 1; i >= 0 && count < limit; i-- {
			v := bucket.Get(keys[i])
			if v == nil {
				continue
			}

			var change StatusChange
			if err := msgpack.Unmarshal(v, &change); err != nil {
				b.logger.Printf("Failed to unmarshal status change: %v", err)
				continue
			}

			changes = append(changes, &change)
			count++
		}

		return nil
	})

	return changes, err
}

// GetRecentChanges retrieves the latest N status changes across all sources
func (b *BoltDB) GetRecentChanges(limit int) ([]*StatusChange, error) {
	var changes []*StatusChange

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(statusChangesBucket))
		if bucket == nil {
			return fmt.Errorf("status_changes bucket not found")
		}

		c := bucket.Cursor()

		// Collect all keys
		var keys [][]byte
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, append([]byte(nil), k...))
		}

		// Iterate in reverse to get newest first
		count := 0
		for i := len(keys) - 1; i >= 0 && count < limit; i-- {
			v := bucket.Get(keys[i])
			if v == nil {
				continue
			}

			var change StatusChange
			if err := msgpack.Unmarshal(v, &change); err != nil {
				b.logger.Printf("Failed to unmarshal status change: %v", err)
				continue
			}

			changes = append(changes, &change)
			count++
		}

		return nil
	})

	return changes, err
}

// GetLastStatusChange retrieves the most recent status change for a source
func (b *BoltDB) GetLastStatusChange(sourceID string) (*StatusChange, error) {
	changes, err := b.GetStatusChanges(sourceID, 1)
	if err != nil {
		return nil, err
	}

	if len(changes) == 0 {
		return nil, fmt.Errorf("no status changes found for source")
	}

	return changes[0], nil
}

// DeleteOldStatusChanges removes status changes older than the specified duration
func (b *BoltDB) DeleteOldStatusChanges(olderThan time.Duration) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	deleted := 0

	err := b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(statusChangesBucket))
		if bucket == nil {
			return fmt.Errorf("status_changes bucket not found")
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var change StatusChange
			if err := msgpack.Unmarshal(v, &change); err != nil {
				continue
			}

			if change.Timestamp.Before(cutoff) {
				if err := bucket.Delete(k); err != nil {
					return err
				}
				deleted++
			}
		}

		return nil
	})

	if err == nil && deleted > 0 {
		b.logger.Printf("Deleted %d old status changes", deleted)
	}

	return deleted, err
}
