package storage

import (
	"fmt"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	// Bucket names
	sourcesBucket        = "sources"
	sourceChatsBucket    = "source_chats"
	chatsBucket          = "chats" // registry of telegram chats (chat_id -> name, etc.)
	statusChangesBucket  = "status_changes"
	configBucket         = "config"
	webhooksBucket       = "webhooks"
	sourceWebhooksBucket = "source_webhooks"
)

// BoltDB wraps the bbolt database
type BoltDB struct {
	db     *bolt.DB
	logger *log.Logger
}

// NewBoltDB creates a new BoltDB instance
func NewBoltDB(path string) (*BoltDB, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	bdb := &BoltDB{
		db:     db,
		logger: log.New(log.Writer(), "[STORAGE] ", log.LstdFlags),
	}

	// Initialize buckets
	if err := bdb.initBuckets(); err != nil {
		db.Close()
		return nil, err
	}

	bdb.logger.Printf("Database initialized at %s", path)

	return bdb, nil
}

// initBuckets creates required buckets if they don't exist
func (b *BoltDB) initBuckets() error {
	return b.db.Update(func(tx *bolt.Tx) error {
		buckets := []string{
			sourcesBucket,
			sourceChatsBucket,
			chatsBucket,
			statusChangesBucket,
			configBucket,
			webhooksBucket,
			sourceWebhooksBucket,
		}

		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}

		return nil
	})
}

// Close closes the database connection
func (b *BoltDB) Close() error {
	b.logger.Println("Closing database")
	return b.db.Close()
}

// DB returns the underlying bbolt database
func (b *BoltDB) DB() *bolt.DB {
	return b.db
}
