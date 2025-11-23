package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOAuthManager_RegisterProvider(t *testing.T) {
	manager := NewOAuthManager().(*OAuthManagerImpl)

	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	auth0Provider := NewAuth0Provider(config)
	manager.RegisterProvider(OAuthProviderAuth0, auth0Provider)

	// Verify provider is registered
	provider, err := manager.GetProvider(OAuthProviderAuth0)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, OAuthProviderAuth0, provider.GetProviderType())
}

func TestOAuthManager_GetProvider_NotFound(t *testing.T) {
	manager := NewOAuthManager()

	_, err := manager.GetProvider(OAuthProviderGoogle)

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderNotFound, err)
}

func TestOAuthManager_GetAuthURL(t *testing.T) {
	manager := NewOAuthManager().(*OAuthManagerImpl)

	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	auth0Provider := NewAuth0Provider(config)
	manager.RegisterProvider(OAuthProviderAuth0, auth0Provider)

	authURL, err := manager.GetAuthURL(context.Background(), OAuthProviderAuth0, "test-state")

	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "dev-vertikon.us.auth0.com")
}

func TestOAuthManager_GetAuthURL_ProviderNotFound(t *testing.T) {
	manager := NewOAuthManager()

	_, err := manager.GetAuthURL(context.Background(), OAuthProviderGoogle, "test-state")

	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderNotFound, err)
}

func TestOAuthManager_HandleCallback(t *testing.T) {
	// This test would require mocking the OAuth flow
	// For now, we test the structure
	manager := NewOAuthManager().(*OAuthManagerImpl)

	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	auth0Provider := NewAuth0Provider(config)
	manager.RegisterProvider(OAuthProviderAuth0, auth0Provider)

	// HandleCallback requires real OAuth code exchange
	// This would be tested in integration tests
	_, err := manager.HandleCallback(context.Background(), OAuthProviderAuth0, "invalid_code", "test-state")

	// Should fail because code is invalid
	assert.Error(t, err)
}

func TestOAuthManager_MultipleProviders(t *testing.T) {
	manager := NewOAuthManager().(*OAuthManagerImpl)

	// Register multiple providers
	auth0Config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "auth0_client_id",
		ClientSecret: "auth0_secret",
		RedirectURL:  "http://localhost:8080/callback/auth0",
	}
	auth0Provider := NewAuth0Provider(auth0Config)
	manager.RegisterProvider(OAuthProviderAuth0, auth0Provider)

	googleConfig := OAuthProviderConfig{
		ClientID:     "google_client_id",
		ClientSecret: "google_secret",
		RedirectURL:  "http://localhost:8080/callback/google",
	}
	googleProvider := NewGoogleProvider(googleConfig)
	manager.RegisterProvider(OAuthProviderGoogle, googleProvider)

	// Verify both are registered
	auth0, err := manager.GetProvider(OAuthProviderAuth0)
	assert.NoError(t, err)
	assert.Equal(t, OAuthProviderAuth0, auth0.GetProviderType())

	google, err := manager.GetProvider(OAuthProviderGoogle)
	assert.NoError(t, err)
	assert.Equal(t, OAuthProviderGoogle, google.GetProviderType())
}
