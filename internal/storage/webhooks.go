package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// Webhook represents an HTTP webhook for notifications
type Webhook struct {
	ID            string            `msgpack:"id" json:"id"`
	Name          string            `msgpack:"name" json:"name"`
	URL           string            `msgpack:"url" json:"url"`
	Method        string            `msgpack:"method" json:"method"` // GET, POST, PUT
	Headers       map[string]string `msgpack:"headers" json:"headers,omitempty"`
	Enabled       bool              `msgpack:"enabled" json:"enabled"`
	CreatedAt     time.Time         `msgpack:"created_at" json:"created_at"`
	UpdatedAt     time.Time         `msgpack:"updated_at" json:"updated_at"`
	LastTriggered *time.Time        `msgpack:"last_triggered" json:"last_triggered,omitempty"`
}

// SaveWebhook stores a webhook in the database
func (b *BoltDB) SaveWebhook(webhook *Webhook) error {
	if webhook.ID == "" {
		webhook.ID = uuid.New().String()
	}

	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = time.Now()
	}

	webhook.UpdatedAt = time.Now()

	data, err := msgpack.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(webhooksBucket))
		if bucket == nil {
			return fmt.Errorf("webhooks bucket not found")
		}

		if err := bucket.Put([]byte(webhook.ID), data); err != nil {
			return fmt.Errorf("failed to save webhook: %w", err)
		}

		if webhook.Name != "" {
			b.logger.Printf("Saved webhook: %s (%s)", webhook.Name, webhook.Method)
		} else {
			b.logger.Printf("Saved webhook: %s (%s)", webhook.URL, webhook.Method)
		}
		return nil
	})
}

// GetWebhook retrieves a webhook by ID
func (b *BoltDB) GetWebhook(id string) (*Webhook, error) {
	var webhook *Webhook

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(webhooksBucket))
		if bucket == nil {
			return fmt.Errorf("webhooks bucket not found")
		}

		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("webhook not found")
		}

		webhook = &Webhook{}
		if err := msgpack.Unmarshal(data, webhook); err != nil {
			return fmt.Errorf("failed to unmarshal webhook: %w", err)
		}

		return nil
	})

	return webhook, err
}

// ListWebhooks retrieves all webhooks
func (b *BoltDB) ListWebhooks() ([]*Webhook, error) {
	var webhooks []*Webhook

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(webhooksBucket))
		if bucket == nil {
			return fmt.Errorf("webhooks bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			webhook := &Webhook{}
			if err := msgpack.Unmarshal(v, webhook); err != nil {
				b.logger.Printf("Failed to unmarshal webhook: %v", err)
				return nil // Skip malformed webhooks
			}
			webhooks = append(webhooks, webhook)
			return nil
		})
	})

	return webhooks, err
}

// DeleteWebhook removes a webhook from the database
func (b *BoltDB) DeleteWebhook(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(webhooksBucket))
		if bucket == nil {
			return fmt.Errorf("webhooks bucket not found")
		}

		if err := bucket.Delete([]byte(id)); err != nil {
			return fmt.Errorf("failed to delete webhook: %w", err)
		}

		b.logger.Printf("Deleted webhook: %s", id)
		return nil
	})
}

// UpdateWebhookLastTriggered updates the last_triggered timestamp
func (b *BoltDB) UpdateWebhookLastTriggered(id string) error {
	webhook, err := b.GetWebhook(id)
	if err != nil {
		return err
	}

	now := time.Now()
	webhook.LastTriggered = &now
	return b.SaveWebhook(webhook)
}
