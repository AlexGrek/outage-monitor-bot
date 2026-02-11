package storage

import (
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// ConfigEntry represents a configuration key-value pair
type ConfigEntry struct {
	Key       string    `msgpack:"key"`
	Value     string    `msgpack:"value"`
	UpdatedAt time.Time `msgpack:"updated_at"`
	UpdatedBy string    `msgpack:"updated_by"` // "env", "api", "initial"
}

// SaveConfig stores a config entry in the database
func (b *BoltDB) SaveConfig(key, value, updatedBy string) error {
	entry := &ConfigEntry{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
		UpdatedBy: updatedBy,
	}

	data, err := msgpack.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal config entry: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(configBucket))
		if bucket == nil {
			return fmt.Errorf("config bucket not found")
		}

		if err := bucket.Put([]byte(key), data); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		b.logger.Printf("Saved config: %s (by %s)", key, updatedBy)
		return nil
	})
}

// GetConfig retrieves a config entry by key
func (b *BoltDB) GetConfig(key string) (*ConfigEntry, error) {
	var entry *ConfigEntry

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(configBucket))
		if bucket == nil {
			return fmt.Errorf("config bucket not found")
		}

		data := bucket.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("config not found")
		}

		entry = &ConfigEntry{}
		if err := msgpack.Unmarshal(data, entry); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		return nil
	})

	return entry, err
}

// GetAllConfig retrieves all config entries
func (b *BoltDB) GetAllConfig() (map[string]*ConfigEntry, error) {
	configs := make(map[string]*ConfigEntry)

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(configBucket))
		if bucket == nil {
			return fmt.Errorf("config bucket not found")
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var entry ConfigEntry
			if err := msgpack.Unmarshal(v, &entry); err != nil {
				b.logger.Printf("Failed to unmarshal config %s: %v", string(k), err)
				continue
			}

			configs[string(k)] = &entry
		}

		return nil
	})

	return configs, err
}

// DeleteConfig removes a config entry
func (b *BoltDB) DeleteConfig(key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(configBucket))
		if bucket == nil {
			return fmt.Errorf("config bucket not found")
		}

		if err := bucket.Delete([]byte(key)); err != nil {
			return fmt.Errorf("failed to delete config: %w", err)
		}

		b.logger.Printf("Deleted config: %s", key)
		return nil
	})
}

// ConfigExists checks if a config key exists
func (b *BoltDB) ConfigExists(key string) bool {
	exists := false

	b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(configBucket))
		if bucket == nil {
			return nil
		}

		data := bucket.Get([]byte(key))
		exists = data != nil
		return nil
	})

	return exists
}
