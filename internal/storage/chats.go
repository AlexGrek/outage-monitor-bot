package storage

import (
	"fmt"
	"strconv"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// Chat represents a named Telegram chat in the registry
type Chat struct {
	ChatID    int64     `msgpack:"chat_id" json:"chat_id"`
	Name      string    `msgpack:"name" json:"name"`
	CreatedAt time.Time `msgpack:"created_at" json:"created_at"`
}

func chatKey(chatID int64) []byte {
	return []byte(strconv.FormatInt(chatID, 10))
}

// SaveChat stores or updates a chat in the registry
func (b *BoltDB) SaveChat(chat *Chat) error {
	if chat.CreatedAt.IsZero() {
		chat.CreatedAt = time.Now()
	}

	data, err := msgpack.Marshal(chat)
	if err != nil {
		return fmt.Errorf("failed to marshal chat: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(chatsBucket))
		if bucket == nil {
			return fmt.Errorf("chats bucket not found")
		}
		if err := bucket.Put(chatKey(chat.ChatID), data); err != nil {
			return fmt.Errorf("failed to save chat: %w", err)
		}
		b.logger.Printf("Saved chat %d (%s)", chat.ChatID, chat.Name)
		return nil
	})
}

// GetChat retrieves a chat from the registry by ID
func (b *BoltDB) GetChat(chatID int64) (*Chat, error) {
	var chat *Chat
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(chatsBucket))
		if bucket == nil {
			return fmt.Errorf("chats bucket not found")
		}
		data := bucket.Get(chatKey(chatID))
		if data == nil {
			return fmt.Errorf("chat not found")
		}
		chat = &Chat{}
		return msgpack.Unmarshal(data, chat)
	})
	return chat, err
}

// ListChats returns all chats in the registry
func (b *BoltDB) ListChats() ([]*Chat, error) {
	var chats []*Chat
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(chatsBucket))
		if bucket == nil {
			return fmt.Errorf("chats bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			chat := &Chat{}
			if err := msgpack.Unmarshal(v, chat); err != nil {
				b.logger.Printf("Failed to unmarshal chat: %v", err)
				return nil
			}
			chats = append(chats, chat)
			return nil
		})
	})
	return chats, err
}

// DeleteChat removes a chat from the registry and from all source associations
func (b *BoltDB) DeleteChat(chatID int64) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		chatsB := tx.Bucket([]byte(chatsBucket))
		if chatsB == nil {
			return fmt.Errorf("chats bucket not found")
		}
		if err := chatsB.Delete(chatKey(chatID)); err != nil {
			return fmt.Errorf("failed to delete chat: %w", err)
		}

		// Remove from all source_chats
		scB := tx.Bucket([]byte(sourceChatsBucket))
		if scB != nil {
			c := scB.Cursor()
			var keysToDelete [][]byte
			for k, v := c.First(); k != nil; k, v = c.Next() {
				var sc SourceChat
				if err := msgpack.Unmarshal(v, &sc); err != nil {
					continue
				}
				if sc.ChatID == chatID {
					keysToDelete = append(keysToDelete, append([]byte(nil), k...))
				}
			}
			for _, key := range keysToDelete {
				_ = scB.Delete(key)
			}
		}

		b.logger.Printf("Deleted chat %d", chatID)
		return nil
	})
}
