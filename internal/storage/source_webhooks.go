package storage

import (
	"fmt"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// composeKey creates a composite key from sourceID and webhookID
func composeKey(sourceID, webhookID string) string {
	return sourceID + ":" + webhookID
}

// decomposeKey extracts sourceID and webhookID from a composite key
func decomposeKey(key string) (sourceID, webhookID string) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// startsWithPrefix checks if a byte slice starts with a prefix
func startsWithPrefix(data, prefix []byte) bool {
	if len(prefix) > len(data) {
		return false
	}
	for i := range prefix {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}

// SourceWebhook represents the association between a source and a webhook
type SourceWebhook struct {
	SourceID  string
	WebhookID string
}

// AddSourceWebhook associates a webhook with a source
func (b *BoltDB) AddSourceWebhook(sourceID, webhookID string) error {
	// Verify both source and webhook exist
	if _, err := b.GetSource(sourceID); err != nil {
		return fmt.Errorf("source not found: %w", err)
	}
	if _, err := b.GetWebhook(webhookID); err != nil {
		return fmt.Errorf("webhook not found: %w", err)
	}

	key := composeKey(sourceID, webhookID)

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceWebhooksBucket))
		if bucket == nil {
			return fmt.Errorf("source_webhooks bucket not found")
		}

		if err := bucket.Put([]byte(key), []byte("1")); err != nil {
			return fmt.Errorf("failed to add source webhook: %w", err)
		}

		b.logger.Printf("Associated webhook %s with source %s", webhookID, sourceID)
		return nil
	})
}

// RemoveSourceWebhook removes the association between a source and a webhook
func (b *BoltDB) RemoveSourceWebhook(sourceID, webhookID string) error {
	key := composeKey(sourceID, webhookID)

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceWebhooksBucket))
		if bucket == nil {
			return fmt.Errorf("source_webhooks bucket not found")
		}

		if err := bucket.Delete([]byte(key)); err != nil {
			return fmt.Errorf("failed to remove source webhook: %w", err)
		}

		b.logger.Printf("Removed webhook %s from source %s", webhookID, sourceID)
		return nil
	})
}

// GetSourceWebhooks retrieves all webhooks for a source
func (b *BoltDB) GetSourceWebhooks(sourceID string) ([]*Webhook, error) {
	var webhooks []*Webhook

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceWebhooksBucket))
		if bucket == nil {
			return fmt.Errorf("source_webhooks bucket not found")
		}

		cursor := bucket.Cursor()
		prefix := []byte(sourceID + ":")

		for k, _ := cursor.Seek(prefix); k != nil && startsWithPrefix(k, prefix); k, _ = cursor.Next() {
			// Extract webhook ID from composite key
			webhookID := string(k[len(prefix):])

			webhook, err := b.GetWebhook(webhookID)
			if err != nil {
				b.logger.Printf("Failed to get webhook %s: %v", webhookID, err)
				continue
			}

			webhooks = append(webhooks, webhook)
		}

		return nil
	})

	return webhooks, err
}

// GetWebhookSources retrieves all sources that use a webhook
func (b *BoltDB) GetWebhookSources(webhookID string) ([]string, error) {
	var sourceIDs []string

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceWebhooksBucket))
		if bucket == nil {
			return fmt.Errorf("source_webhooks bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			sourceID, wID := decomposeKey(string(k))
			if wID == webhookID {
				sourceIDs = append(sourceIDs, sourceID)
			}
			return nil
		})
	})

	return sourceIDs, err
}
