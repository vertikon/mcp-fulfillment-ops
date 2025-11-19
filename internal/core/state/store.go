// Package state provides internal state persistence using BadgerDB
package state

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Store provides persistent state storage using BadgerDB
type Store struct {
	db *badger.DB
}

// NewStore creates a new state store
func NewStore(path string) (*Store, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable Badger's default logger

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	// Start garbage collection goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			for {
				err := db.RunValueLogGC(0.5)
				if err != nil {
					break
				}
			}
		}
	}()

	return &Store{db: db}, nil
}

// Close closes the state store
func (s *Store) Close() error {
	return s.db.Close()
}

// Get retrieves a value from the store
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			value = make([]byte, len(val))
			copy(value, val)
			return nil
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, ErrKeyNotFound
	}

	return value, err
}

// Set stores a value in the store
func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), value)
		if ttl > 0 {
			entry.WithTTL(ttl)
		}
		return txn.SetEntry(entry)
	})
}

// Delete deletes a key from the store
func (s *Store) Delete(ctx context.Context, key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// GetJSON retrieves and unmarshals a JSON value
func (s *Store) GetJSON(ctx context.Context, key string, v interface{}) error {
	data, err := s.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// SetJSON marshals and stores a JSON value
func (s *Store) SetJSON(ctx context.Context, key string, v interface{}, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return s.Set(ctx, key, data, ttl)
}

// ListKeys lists all keys with the given prefix
func (s *Store) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			keys = append(keys, string(item.Key()))
		}

		return nil
	})

	return keys, err
}

// Errors
var (
	ErrKeyNotFound = &StateError{Message: "key not found"}
)

// StateError represents a state store error
type StateError struct {
	Message string
}

func (e *StateError) Error() string {
	return e.Message
}

