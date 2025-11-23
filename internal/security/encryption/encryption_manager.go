package encryption

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidKey       = errors.New("invalid encryption key")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrInvalidData      = errors.New("invalid data")
)

// EncryptionManager handles encryption/decryption operations
type EncryptionManager interface {
	// Encrypt encrypts data using AES-256-GCM
	Encrypt(plaintext []byte) ([]byte, error)

	// Decrypt decrypts data using AES-256-GCM
	Decrypt(ciphertext []byte) ([]byte, error)

	// EncryptWithKey encrypts data with a specific key
	EncryptWithKey(plaintext []byte, key []byte) ([]byte, error)

	// DecryptWithKey decrypts data with a specific key
	DecryptWithKey(ciphertext []byte, key []byte) ([]byte, error)

	// HashPassword hashes a password using bcrypt
	HashPassword(password string) (string, error)

	// VerifyPassword verifies a password against a hash
	VerifyPassword(password, hash string) bool

	// HashArgon2 hashes data using Argon2
	HashArgon2(data []byte, salt []byte) []byte

	// Sign signs data using RSA
	Sign(data []byte, privateKey *rsa.PrivateKey) ([]byte, error)

	// Verify verifies a signature using RSA
	Verify(data, signature []byte, publicKey *rsa.PublicKey) bool
}

// encryptionManagerImpl implements EncryptionManager
type encryptionManagerImpl struct {
	keyManager KeyManager
	logger     *zap.Logger
}

// NewEncryptionManager creates a new EncryptionManager
func NewEncryptionManager(keyManager KeyManager) EncryptionManager {
	return &encryptionManagerImpl{
		keyManager: keyManager,
		logger:     logger.WithContext(nil),
	}
}

// Encrypt encrypts data using AES-256-GCM with default key
func (m *encryptionManagerImpl) Encrypt(plaintext []byte) ([]byte, error) {
	key, err := m.keyManager.GetEncryptionKey()
	if err != nil {
		m.logger.Error("Failed to get encryption key", zap.Error(err))
		return nil, err
	}
	return m.EncryptWithKey(plaintext, key)
}

// Decrypt decrypts data using AES-256-GCM with default key
func (m *encryptionManagerImpl) Decrypt(ciphertext []byte) ([]byte, error) {
	key, err := m.keyManager.GetEncryptionKey()
	if err != nil {
		m.logger.Error("Failed to get encryption key", zap.Error(err))
		return nil, err
	}
	return m.DecryptWithKey(ciphertext, key)
}

// EncryptWithKey encrypts data with a specific key using AES-256-GCM
func (m *encryptionManagerImpl) EncryptWithKey(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		m.logger.Error("Failed to create cipher", zap.Error(err))
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		m.logger.Error("Failed to create GCM", zap.Error(err))
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		m.logger.Error("Failed to generate nonce", zap.Error(err))
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptWithKey decrypts data with a specific key using AES-256-GCM
func (m *encryptionManagerImpl) DecryptWithKey(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		m.logger.Error("Failed to create cipher", zap.Error(err))
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		m.logger.Error("Failed to create GCM", zap.Error(err))
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidData
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		m.logger.Warn("Decryption failed", zap.Error(err))
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// HashPassword hashes a password using bcrypt
func (m *encryptionManagerImpl) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		m.logger.Error("Failed to hash password", zap.Error(err))
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func (m *encryptionManagerImpl) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashArgon2 hashes data using Argon2
func (m *encryptionManagerImpl) HashArgon2(data []byte, salt []byte) []byte {
	// Argon2id with recommended parameters
	hash := argon2.IDKey(data, salt, 1, 64*1024, 4, 32)
	return hash
}

// Sign signs data using RSA
func (m *encryptionManagerImpl) Sign(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	hashed := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		m.logger.Error("Failed to sign data", zap.Error(err))
		return nil, err
	}
	return signature, nil
}

// Verify verifies a signature using RSA
func (m *encryptionManagerImpl) Verify(data, signature []byte, publicKey *rsa.PublicKey) bool {
	hashed := sha256.Sum256(data)
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
	return err == nil
}
