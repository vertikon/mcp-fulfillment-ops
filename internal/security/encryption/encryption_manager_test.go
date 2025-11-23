package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockKeyManager is a mock implementation of KeyManager
type MockKeyManager struct {
	mock.Mock
}

func (m *MockKeyManager) GetEncryptionKey() ([]byte, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKeyManager) GetKeyVersion() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockKeyManager) RotateKey() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockKeyManager) GetRSAPrivateKey() (*rsa.PrivateKey, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rsa.PrivateKey), args.Error(1)
}

func (m *MockKeyManager) GetRSAPublicKey() (*rsa.PublicKey, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rsa.PublicKey), args.Error(1)
}

func (m *MockKeyManager) LoadKeyFromEnv(keyName string) error {
	args := m.Called(keyName)
	return args.Error(0)
}

func (m *MockKeyManager) LoadKeyFromFile(filePath string) error {
	args := m.Called(filePath)
	return args.Error(0)
}

func TestEncryptionManager_EncryptDecrypt(t *testing.T) {
	// Generate a test key
	testKey := make([]byte, 32)
	_, _ = rand.Read(testKey)

	mockKeyManager := new(MockKeyManager)
	mockKeyManager.On("GetEncryptionKey").Return(testKey, nil)

	manager := NewEncryptionManager(mockKeyManager)

	tests := []struct {
		name          string
		plaintext     []byte
		expectedError error
	}{
		{
			name:          "encrypt and decrypt simple text",
			plaintext:     []byte("Hello, World!"),
			expectedError: nil,
		},
		{
			name:          "encrypt and decrypt empty data",
			plaintext:     []byte(""),
			expectedError: nil,
		},
		{
			name:          "encrypt and decrypt large data",
			plaintext:     make([]byte, 1024),
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := manager.Encrypt(tt.plaintext)
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ciphertext)
				assert.NotEqual(t, tt.plaintext, ciphertext) // Should be different

				// Decrypt
				decrypted, err := manager.Decrypt(ciphertext)
				assert.NoError(t, err)
				assert.Equal(t, tt.plaintext, decrypted)
			}
		})
	}
}

func TestEncryptionManager_EncryptDecryptWithKey(t *testing.T) {
	manager := &encryptionManagerImpl{}

	testKey := make([]byte, 32)
	_, _ = rand.Read(testKey)

	plaintext := []byte("Test data")

	// Encrypt with key
	ciphertext, err := manager.EncryptWithKey(plaintext, testKey)
	assert.NoError(t, err)
	assert.NotNil(t, ciphertext)

	// Decrypt with same key
	decrypted, err := manager.DecryptWithKey(ciphertext, testKey)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptionManager_InvalidKey(t *testing.T) {
	manager := &encryptionManagerImpl{}

	plaintext := []byte("Test data")
	invalidKey := make([]byte, 16) // Wrong size

	_, err := manager.EncryptWithKey(plaintext, invalidKey)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidKey, err)
}

func TestEncryptionManager_HashPassword(t *testing.T) {
	mockKeyManager := new(MockKeyManager)
	manager := NewEncryptionManager(mockKeyManager)

	password := "mySecurePassword123"

	hash, err := manager.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify password
	valid := manager.VerifyPassword(password, hash)
	assert.True(t, valid)

	// Verify wrong password
	valid = manager.VerifyPassword("wrongPassword", hash)
	assert.False(t, valid)
}

func TestEncryptionManager_HashArgon2(t *testing.T) {
	mockKeyManager := new(MockKeyManager)
	manager := NewEncryptionManager(mockKeyManager)

	data := []byte("test data")
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)

	hash := manager.HashArgon2(data, salt)
	assert.NotNil(t, hash)
	assert.Len(t, hash, 32) // Argon2id with 32-byte output

	// Same input should produce same hash
	hash2 := manager.HashArgon2(data, salt)
	assert.Equal(t, hash, hash2)

	// Different salt should produce different hash
	salt2 := make([]byte, 16)
	_, _ = rand.Read(salt2)
	hash3 := manager.HashArgon2(data, salt2)
	assert.NotEqual(t, hash, hash3)
}

func TestEncryptionManager_SignVerify(t *testing.T) {
	mockKeyManager := new(MockKeyManager)
	manager := NewEncryptionManager(mockKeyManager)

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	publicKey := &privateKey.PublicKey

	data := []byte("data to sign")

	// Sign
	signature, err := manager.Sign(data, privateKey)
	assert.NoError(t, err)
	assert.NotNil(t, signature)

	// Verify
	valid := manager.Verify(data, signature, publicKey)
	assert.True(t, valid)

	// Verify with wrong data
	invalid := manager.Verify([]byte("wrong data"), signature, publicKey)
	assert.False(t, invalid)

	// Verify with wrong signature
	wrongSignature := make([]byte, len(signature))
	_, _ = rand.Read(wrongSignature)
	invalid = manager.Verify(data, wrongSignature, publicKey)
	assert.False(t, invalid)
}
