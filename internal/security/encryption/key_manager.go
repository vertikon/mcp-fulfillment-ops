package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	ErrKeyNotFound       = errors.New("key not found")
	ErrKeyRotationFailed = errors.New("key rotation failed")
)

// KeyManager handles encryption key management and rotation
type KeyManager interface {
	// GetEncryptionKey returns the current encryption key
	GetEncryptionKey() ([]byte, error)

	// GetKeyVersion returns the current key version
	GetKeyVersion() string

	// RotateKey rotates the encryption key
	RotateKey() error

	// GetRSAPrivateKey returns RSA private key
	GetRSAPrivateKey() (*rsa.PrivateKey, error)

	// GetRSAPublicKey returns RSA public key
	GetRSAPublicKey() (*rsa.PublicKey, error)

	// LoadKeyFromEnv loads key from environment variable
	LoadKeyFromEnv(keyName string) error

	// LoadKeyFromFile loads key from file
	LoadKeyFromFile(filePath string) error
}

// keyManagerImpl implements KeyManager
type keyManagerImpl struct {
	encryptionKey []byte
	keyVersion    string
	rsaPrivateKey *rsa.PrivateKey
	rsaPublicKey  *rsa.PublicKey
	rotationTTL   time.Duration
	lastRotation  time.Time
	mu            sync.RWMutex
	logger        *zap.Logger
}

// KeyManagerConfig holds configuration for KeyManager
type KeyManagerConfig struct {
	RotationTTL time.Duration
	KeySize     int // RSA key size (2048, 4096)
}

// NewKeyManager creates a new KeyManager
func NewKeyManager(config KeyManagerConfig) KeyManager {
	km := &keyManagerImpl{
		keyVersion:  "v1",
		rotationTTL: config.RotationTTL,
		logger:      logger.WithContext(nil),
	}

	// Generate initial encryption key
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		km.logger.Error("Failed to generate encryption key", zap.Error(err))
	} else {
		km.encryptionKey = key
	}

	// Generate RSA key pair
	if err := km.generateRSAKeys(config.KeySize); err != nil {
		km.logger.Error("Failed to generate RSA keys", zap.Error(err))
	}

	km.lastRotation = time.Now()
	return km
}

// GetEncryptionKey returns the current encryption key
func (m *keyManagerImpl) GetEncryptionKey() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.encryptionKey == nil {
		return nil, ErrKeyNotFound
	}

	// Check if rotation is needed
	if time.Since(m.lastRotation) > m.rotationTTL {
		go m.RotateKey()
	}

	// Return a copy to prevent external modification
	key := make([]byte, len(m.encryptionKey))
	copy(key, m.encryptionKey)
	return key, nil
}

// GetKeyVersion returns the current key version
func (m *keyManagerImpl) GetKeyVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.keyVersion
}

// RotateKey rotates the encryption key
func (m *keyManagerImpl) RotateKey() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		m.logger.Error("Failed to generate new encryption key", zap.Error(err))
		return ErrKeyRotationFailed
	}

	// Update key
	oldKey := m.encryptionKey
	m.encryptionKey = newKey
	m.keyVersion = "v" + time.Now().Format("20060102150405")
	m.lastRotation = time.Now()

	m.logger.Info("Encryption key rotated",
		zap.String("old_version", m.keyVersion),
		zap.String("new_version", m.keyVersion),
	)

	// In production, old key should be kept for decrypting old data
	_ = oldKey

	return nil
}

// GetRSAPrivateKey returns RSA private key
func (m *keyManagerImpl) GetRSAPrivateKey() (*rsa.PrivateKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.rsaPrivateKey == nil {
		return nil, ErrKeyNotFound
	}

	return m.rsaPrivateKey, nil
}

// GetRSAPublicKey returns RSA public key
func (m *keyManagerImpl) GetRSAPublicKey() (*rsa.PublicKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.rsaPublicKey == nil {
		return nil, ErrKeyNotFound
	}

	return m.rsaPublicKey, nil
}

// LoadKeyFromEnv loads key from environment variable
func (m *keyManagerImpl) LoadKeyFromEnv(keyName string) error {
	if keyName == "" {
		return fmt.Errorf("key name cannot be empty")
	}

	keyValue := os.Getenv(keyName)
	if keyValue == "" {
		return fmt.Errorf("environment variable %s not set", keyName)
	}

	// Decode base64 or hex key
	keyBytes, err := decodeKey(keyValue)
	if err != nil {
		m.logger.Error("Failed to decode key from environment",
			zap.String("key_name", keyName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to decode key: %w", err)
	}

	if len(keyBytes) != 32 {
		return fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(keyBytes))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.encryptionKey = keyBytes
	m.keyVersion = "env_" + keyName + "_" + time.Now().Format("20060102150405")
	m.lastRotation = time.Now()

	m.logger.Info("Key loaded from environment",
		zap.String("key_name", keyName),
		zap.String("key_version", m.keyVersion),
	)

	return nil
}

// LoadKeyFromFile loads key from file
func (m *keyManagerImpl) LoadKeyFromFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check file permissions (should be 0600 or 0400)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key file not found: %s", filePath)
		}
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	// Verify file permissions are secure (owner read/write only)
	mode := fileInfo.Mode().Perm()
	if mode&0077 != 0 {
		m.logger.Warn("Key file has insecure permissions",
			zap.String("file_path", filePath),
			zap.String("mode", mode.String()),
		)
		// Don't fail, but log warning
	}

	// Read file
	keyData, err := os.ReadFile(filePath)
	if err != nil {
		m.logger.Error("Failed to read key file",
			zap.String("file_path", filePath),
			zap.Error(err),
		)
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Remove whitespace and newlines
	keyString := strings.TrimSpace(string(keyData))
	keyString = strings.ReplaceAll(keyString, "\n", "")
	keyString = strings.ReplaceAll(keyString, "\r", "")
	keyString = strings.ReplaceAll(keyString, " ", "")

	// Decode base64 or hex key
	keyBytes, err := decodeKey(keyString)
	if err != nil {
		m.logger.Error("Failed to decode key from file",
			zap.String("file_path", filePath),
			zap.Error(err),
		)
		return fmt.Errorf("failed to decode key: %w", err)
	}

	if len(keyBytes) != 32 {
		return fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(keyBytes))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.encryptionKey = keyBytes
	m.keyVersion = "file_" + filepath.Base(filePath) + "_" + time.Now().Format("20060102150405")
	m.lastRotation = time.Now()

	m.logger.Info("Key loaded from file",
		zap.String("file_path", filePath),
		zap.String("key_version", m.keyVersion),
	)

	return nil
}

// generateRSAKeys generates RSA key pair
func (m *keyManagerImpl) generateRSAKeys(keySize int) error {
	if keySize == 0 {
		keySize = 2048 // Default
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		m.logger.Error("Failed to generate RSA private key", zap.Error(err))
		return err
	}

	m.rsaPrivateKey = privateKey
	m.rsaPublicKey = &privateKey.PublicKey

	m.logger.Info("RSA keys generated",
		zap.Int("key_size", keySize),
	)

	return nil
}

// ExportRSAPrivateKey exports RSA private key as PEM
func (m *keyManagerImpl) ExportRSAPrivateKey() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.rsaPrivateKey == nil {
		return nil, ErrKeyNotFound
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(m.rsaPrivateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return privateKeyPEM, nil
}

// ExportRSAPublicKey exports RSA public key as PEM
func (m *keyManagerImpl) ExportRSAPublicKey() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.rsaPublicKey == nil {
		return nil, ErrKeyNotFound
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(m.rsaPublicKey)
	if err != nil {
		return nil, err
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return publicKeyPEM, nil
}

// decodeKey attempts to decode a key string from base64 or hex format
func decodeKey(keyString string) ([]byte, error) {
	if keyString == "" {
		return nil, fmt.Errorf("key string cannot be empty")
	}

	// Try base64 first
	if keyBytes, err := base64.StdEncoding.DecodeString(keyString); err == nil {
		return keyBytes, nil
	}

	// Try base64 URL encoding
	if keyBytes, err := base64.URLEncoding.DecodeString(keyString); err == nil {
		return keyBytes, nil
	}

	// Try hex
	if keyBytes, err := hex.DecodeString(keyString); err == nil {
		return keyBytes, nil
	}

	return nil, fmt.Errorf("key is not in base64 or hex format")
}
