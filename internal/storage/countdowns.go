package storage

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// Countdown represents a scheduled "days until a date" notification.
// Every day at NotifyTime (in Timezone) it sends MessageTemplate to its sinks,
// with every "{}" in the template replaced by the number of days left.
type Countdown struct {
	ID              string    `msgpack:"id" json:"id"`
	Name            string    `msgpack:"name" json:"name"`
	TargetDate      string    `msgpack:"target_date" json:"target_date"`           // "2006-01-02" (interpreted in Timezone)
	NotifyTime      string    `msgpack:"notify_time" json:"notify_time"`           // "15:04" (24h, in Timezone)
	Timezone        string    `msgpack:"timezone" json:"timezone"`                 // IANA name, e.g. "Europe/Kyiv"
	MessageTemplate string    `msgpack:"message_template" json:"message_template"` // "{}" replaced with days left
	Enabled         bool      `msgpack:"enabled" json:"enabled"`
	ChatIDs         []int64   `msgpack:"chat_ids" json:"chat_ids"`             // Telegram chat sinks
	WebhookIDs      []string  `msgpack:"webhook_ids" json:"webhook_ids"`       // Webhook sinks
	LastSentDate    string    `msgpack:"last_sent_date" json:"last_sent_date"` // "2006-01-02" (in Timezone) of last send; prevents double-send
	CreatedAt       time.Time `msgpack:"created_at" json:"created_at"`
	UpdatedAt       time.Time `msgpack:"updated_at" json:"updated_at"`
}

// SaveCountdown stores or updates a countdown in the database
func (b *BoltDB) SaveCountdown(c *Countdown) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	c.UpdatedAt = time.Now()

	data, err := msgpack.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal countdown: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(countdownsBucket))
		if bucket == nil {
			return fmt.Errorf("countdowns bucket not found")
		}
		if err := bucket.Put([]byte(c.ID), data); err != nil {
			return fmt.Errorf("failed to save countdown: %w", err)
		}
		b.logger.Printf("Saved countdown: %s (target %s, notify %s %s)", c.Name, c.TargetDate, c.NotifyTime, c.Timezone)
		return nil
	})
}

// GetCountdown retrieves a countdown by ID
func (b *BoltDB) GetCountdown(id string) (*Countdown, error) {
	var c *Countdown
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(countdownsBucket))
		if bucket == nil {
			return fmt.Errorf("countdowns bucket not found")
		}
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("countdown not found")
		}
		c = &Countdown{}
		return msgpack.Unmarshal(data, c)
	})
	return c, err
}

// ListCountdowns retrieves all countdowns
func (b *BoltDB) ListCountdowns() ([]*Countdown, error) {
	var countdowns []*Countdown
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(countdownsBucket))
		if bucket == nil {
			return fmt.Errorf("countdowns bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			c := &Countdown{}
			if err := msgpack.Unmarshal(v, c); err != nil {
				b.logger.Printf("Failed to unmarshal countdown: %v", err)
				return nil // Skip malformed entries
			}
			countdowns = append(countdowns, c)
			return nil
		})
	})
	return countdowns, err
}

// DeleteCountdown removes a countdown from the database
func (b *BoltDB) DeleteCountdown(id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(countdownsBucket))
		if bucket == nil {
			return fmt.Errorf("countdowns bucket not found")
		}
		if err := bucket.Delete([]byte(id)); err != nil {
			return fmt.Errorf("failed to delete countdown: %w", err)
		}
		b.logger.Printf("Deleted countdown: %s", id)
		return nil
	})
}

// UpdateCountdownLastSent records the date (in the countdown's timezone) a notification was last sent.
// This is used by the scheduler to ensure at most one notification per day per countdown.
func (b *BoltDB) UpdateCountdownLastSent(id, date string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(countdownsBucket))
		if bucket == nil {
			return fmt.Errorf("countdowns bucket not found")
		}
		data := bucket.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("countdown not found")
		}
		var c Countdown
		if err := msgpack.Unmarshal(data, &c); err != nil {
			return fmt.Errorf("failed to unmarshal countdown: %w", err)
		}
		c.LastSentDate = date
		newData, err := msgpack.Marshal(&c)
		if err != nil {
			return fmt.Errorf("failed to marshal countdown: %w", err)
		}
		return bucket.Put([]byte(id), newData)
	})
}
