package encryption

import (
	"context"
	"errors"
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrInvalidSecret  = errors.New("invalid secret")
)

// SecureStorage provides secure storage for secrets
type SecureStorage interface {
	// Store stores a secret securely
	Store(ctx context.Context, key string, value []byte) error

	// Retrieve retrieves a secret
	Retrieve(ctx context.Context, key string) ([]byte, error)

	// Delete deletes a secret
	Delete(ctx context.Context, key string) error

	// Exists checks if a secret exists
	Exists(ctx context.Context, key string) (bool, error)

	// List lists all secret keys (with optional prefix)
	List(ctx context.Context, prefix string) ([]string, error)
}

// Manager implements SecureStorage
type SecureStorageManager struct {
	encryptionManager EncryptionManager
	storage           StorageBackend
	mu                sync.RWMutex
	logger            *zap.Logger
}

// StorageBackend defines interface for storage backend
type StorageBackend interface {
	Put(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]string, error)
}

// SecureStorageConfig holds configuration for SecureStorage
type SecureStorageConfig struct {
	EncryptAtRest bool
}

// NewSecureStorage creates a new SecureStorage
func NewSecureStorage(encryptionManager EncryptionManager, backend StorageBackend) SecureStorage {
	return &SecureStorageManager{
		encryptionManager: encryptionManager,
		storage:           backend,
		logger:            logger.WithContext(nil),
	}
}

// Store stores a secret securely
func (m *SecureStorageManager) Store(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return ErrInvalidSecret
	}

	// Encrypt before storing
	encryptedValue, err := m.encryptionManager.Encrypt(value)
	if err != nil {
		m.logger.Error("Failed to encrypt secret",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	// Store encrypted value
	if err := m.storage.Put(ctx, key, encryptedValue); err != nil {
		m.logger.Error("Failed to store secret",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	m.logger.Debug("Secret stored",
		zap.String("key", key),
	)

	return nil
}

// Retrieve retrieves a secret
func (m *SecureStorageManager) Retrieve(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, ErrInvalidSecret
	}

	// Retrieve encrypted value
	encryptedValue, err := m.storage.Get(ctx, key)
	if err != nil {
		m.logger.Warn("Secret not found",
			zap.String("key", key),
		)
		return nil, ErrSecretNotFound
	}

	// Decrypt
	value, err := m.encryptionManager.Decrypt(encryptedValue)
	if err != nil {
		m.logger.Error("Failed to decrypt secret",
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, err
	}

	return value, nil
}

// Delete deletes a secret
func (m *SecureStorageManager) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidSecret
	}

	if err := m.storage.Delete(ctx, key); err != nil {
		m.logger.Error("Failed to delete secret",
			zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	m.logger.Debug("Secret deleted",
		zap.String("key", key),
	)

	return nil
}

// Exists checks if a secret exists
func (m *SecureStorageManager) Exists(ctx context.Context, key string) (bool, error) {
	return m.storage.Exists(ctx, key)
}

// List lists all secret keys (with optional prefix)
func (m *SecureStorageManager) List(ctx context.Context, prefix string) ([]string, error) {
	return m.storage.List(ctx, prefix)
}

// InMemoryBackend implements StorageBackend using in-memory storage
type InMemoryBackend struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewInMemoryBackend creates a new in-memory backend
func NewInMemoryBackend() StorageBackend {
	return &InMemoryBackend{
		data: make(map[string][]byte),
	}
}

// Put stores a value
func (b *InMemoryBackend) Put(ctx context.Context, key string, value []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[key] = value
	return nil
}

// Get retrieves a value
func (b *InMemoryBackend) Get(ctx context.Context, key string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	value, ok := b.data[key]
	if !ok {
		return nil, ErrSecretNotFound
	}
	return value, nil
}

// Delete deletes a value
func (b *InMemoryBackend) Delete(ctx context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.data, key)
	return nil
}

// Exists checks if a key exists
func (b *InMemoryBackend) Exists(ctx context.Context, key string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.data[key]
	return exists, nil
}

// List lists all keys with prefix
func (b *InMemoryBackend) List(ctx context.Context, prefix string) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	keys := make([]string, 0)
	for key := range b.data {
		if prefix == "" || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
