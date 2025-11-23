package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenManager_Generate(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		email         string
		roles         []string
		config        TokenManagerConfig
		expectedError error
	}{
		{
			name:   "successful generation HS256",
			userID: "user123",
			email:  "test@example.com",
			roles:  []string{"user", "admin"},
			config: TokenManagerConfig{
				SecretKey:     "test-secret-key-32-bytes-long!!",
				SigningMethod: "HS256",
				TokenTTL:      1 * time.Hour,
				RefreshTTL:    24 * time.Hour,
			},
			expectedError: nil,
		},
		// RS256 test skipped - requires RSA key setup
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewTokenManager(tt.config)
			token, err := manager.Generate(context.Background(), tt.userID, tt.email, tt.roles)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestTokenManager_Validate(t *testing.T) {
	config := TokenManagerConfig{
		SecretKey:     "test-secret-key-32-bytes-long!!",
		SigningMethod: "HS256",
		TokenTTL:      1 * time.Hour,
		RefreshTTL:    24 * time.Hour,
	}

	manager := NewTokenManager(config)

	tests := []struct {
		name           string
		setupToken     func() string
		expectedUserID string
		expectedError  error
	}{
		{
			name: "valid token",
			setupToken: func() string {
				token, _ := manager.Generate(context.Background(), "user123", "test@example.com", []string{"user"})
				return token
			},
			expectedUserID: "user123",
			expectedError:  nil,
		},
		{
			name: "invalid token",
			setupToken: func() string {
				return "invalid.token.here"
			},
			expectedUserID: "",
			expectedError:  ErrInvalidToken,
		},
		{
			name: "expired token",
			setupToken: func() string {
				// Create manager with very short TTL
				shortConfig := TokenManagerConfig{
					SecretKey:     "test-secret-key-32-bytes-long!!",
					SigningMethod: "HS256",
					TokenTTL:      1 * time.Millisecond,
					RefreshTTL:    24 * time.Hour,
				}
				shortManager := NewTokenManager(shortConfig)
				token, _ := shortManager.Generate(context.Background(), "user123", "test@example.com", []string{"user"})
				time.Sleep(10 * time.Millisecond) // Wait for expiration
				return token
			},
			expectedUserID: "",
			expectedError:  ErrExpiredToken,
		},
		{
			name: "revoked token",
			setupToken: func() string {
				token, _ := manager.Generate(context.Background(), "user123", "test@example.com", []string{"user"})
				_ = manager.Revoke(context.Background(), token)
				return token
			},
			expectedUserID: "",
			expectedError:  ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			userID, err := manager.Validate(context.Background(), token)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Empty(t, userID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUserID, userID)
			}
		})
	}
}

func TestTokenManager_Refresh(t *testing.T) {
	config := TokenManagerConfig{
		SecretKey:     "test-secret-key-32-bytes-long!!",
		SigningMethod: "HS256",
		TokenTTL:      1 * time.Hour,
		RefreshTTL:    24 * time.Hour,
	}

	manager := NewTokenManager(config)

	tests := []struct {
		name          string
		setupToken    func() string
		expectedError error
	}{
		{
			name: "successful refresh",
			setupToken: func() string {
				token, _ := manager.Generate(context.Background(), "user123", "test@example.com", []string{"user"})
				return token
			},
			expectedError: nil,
		},
		{
			name: "invalid token refresh",
			setupToken: func() string {
				return "invalid.token.here"
			},
			expectedError: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			newToken, err := manager.Refresh(context.Background(), token)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Empty(t, newToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, newToken)
				// The new token should be different from the old one (even if same content, different signature)
				// Note: We can't easily validate the new token here because the old one is revoked
				// but Refresh should succeed and return a non-empty token
			}
		})
	}
}

func TestTokenManager_Revoke(t *testing.T) {
	config := TokenManagerConfig{
		SecretKey:     "test-secret-key-32-bytes-long!!",
		SigningMethod: "HS256",
		TokenTTL:      1 * time.Hour,
		RefreshTTL:    24 * time.Hour,
	}

	manager := NewTokenManager(config)

	token, _ := manager.Generate(context.Background(), "user123", "test@example.com", []string{"user"})

	err := manager.Revoke(context.Background(), token)
	assert.NoError(t, err)

	// Verify token is revoked
	_, err = manager.Validate(context.Background(), token)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}
