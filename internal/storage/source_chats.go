package storage

import (
	"encoding/binary"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
	bolt "go.etcd.io/bbolt"
)

// SourceChat represents a many-to-many relationship between sources and chats
type SourceChat struct {
	SourceID string `msgpack:"source_id"`
	ChatID   int64  `msgpack:"chat_id"`
}

// makeSourceChatKey creates a composite key for source-chat relationship
func makeSourceChatKey(sourceID string, chatID int64) []byte {
	chatBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(chatBytes, uint64(chatID))
	return append([]byte(sourceID+":"), chatBytes...)
}

// AddSourceChat adds a chat to a source
func (b *BoltDB) AddSourceChat(sourceID string, chatID int64) error {
	sc := &SourceChat{
		SourceID: sourceID,
		ChatID:   chatID,
	}

	data, err := msgpack.Marshal(sc)
	if err != nil {
		return fmt.Errorf("failed to marshal source-chat: %w", err)
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceChatsBucket))
		if bucket == nil {
			return fmt.Errorf("source_chats bucket not found")
		}

		key := makeSourceChatKey(sourceID, chatID)
		if err := bucket.Put(key, data); err != nil {
			return fmt.Errorf("failed to add source-chat: %w", err)
		}

		b.logger.Printf("Added chat %d to source %s", chatID, sourceID)
		return nil
	})
}

// RemoveSourceChat removes a chat from a source
func (b *BoltDB) RemoveSourceChat(sourceID string, chatID int64) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceChatsBucket))
		if bucket == nil {
			return fmt.Errorf("source_chats bucket not found")
		}

		key := makeSourceChatKey(sourceID, chatID)
		if err := bucket.Delete(key); err != nil {
			return fmt.Errorf("failed to remove source-chat: %w", err)
		}

		b.logger.Printf("Removed chat %d from source %s", chatID, sourceID)
		return nil
	})
}

// GetSourceChats retrieves all chat IDs for a source
func (b *BoltDB) GetSourceChats(sourceID string) ([]int64, error) {
	var chatIDs []int64

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceChatsBucket))
		if bucket == nil {
			return fmt.Errorf("source_chats bucket not found")
		}

		c := bucket.Cursor()
		prefix := []byte(sourceID + ":")

		for k, v := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, v = c.Next() {
			var sc SourceChat
			if err := msgpack.Unmarshal(v, &sc); err != nil {
				b.logger.Printf("Failed to unmarshal source-chat: %v", err)
				continue
			}

			chatIDs = append(chatIDs, sc.ChatID)
		}

		return nil
	})

	return chatIDs, err
}

// GetChatSources retrieves all source IDs for a chat
func (b *BoltDB) GetChatSources(chatID int64) ([]string, error) {
	var sourceIDs []string

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceChatsBucket))
		if bucket == nil {
			return fmt.Errorf("source_chats bucket not found")
		}

		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var sc SourceChat
			if err := msgpack.Unmarshal(v, &sc); err != nil {
				b.logger.Printf("Failed to unmarshal source-chat: %v", err)
				continue
			}

			if sc.ChatID == chatID {
				sourceIDs = append(sourceIDs, sc.SourceID)
			}
		}

		return nil
	})

	return sourceIDs, err
}

// RemoveAllSourceChats removes all chats for a source (useful when deleting a source)
func (b *BoltDB) RemoveAllSourceChats(sourceID string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sourceChatsBucket))
		if bucket == nil {
			return fmt.Errorf("source_chats bucket not found")
		}

		c := bucket.Cursor()
		prefix := []byte(sourceID + ":")

		// Collect keys to delete
		var keysToDelete [][]byte
		for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, _ = c.Next() {
			keysToDelete = append(keysToDelete, append([]byte(nil), k...))
		}

		// Delete collected keys
		for _, key := range keysToDelete {
			if err := bucket.Delete(key); err != nil {
				return fmt.Errorf("failed to delete source-chat: %w", err)
			}
		}

		b.logger.Printf("Removed all chats from source %s", sourceID)
		return nil
	})
}
