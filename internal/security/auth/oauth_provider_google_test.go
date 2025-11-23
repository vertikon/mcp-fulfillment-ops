package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoogleProvider_GetAuthURL(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	authURL, err := provider.GetAuthURL(context.Background(), "test-state-123")

	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "accounts.google.com")
	assert.Contains(t, authURL, "/o/oauth2/v2/auth")
	assert.Contains(t, authURL, "test_client_id")
	assert.Contains(t, authURL, "test-state-123")
	assert.Contains(t, authURL, "openid")
}

func TestGoogleProvider_GetUserInfo(t *testing.T) {
	// Mock Google userinfo endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_access_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock Google userinfo response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "123456789",
			"email": "test@example.com",
			"verified_email": true,
			"name": "Test User",
			"given_name": "Test",
			"family_name": "User",
			"picture": "https://example.com/avatar.jpg"
		}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
		UserInfoURL:  mockServer.URL,
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	userInfo, err := provider.GetUserInfo(context.Background(), "test_access_token")

	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "123456789", userInfo.ID)
	assert.Equal(t, "test@example.com", userInfo.Email)
	assert.Equal(t, "Test User", userInfo.Name)
	assert.Equal(t, "https://example.com/avatar.jpg", userInfo.Picture)
	assert.Equal(t, OAuthProviderGoogle, userInfo.Provider)
}

func TestGoogleProvider_GetUserInfo_Error(t *testing.T) {
	// Mock server returning error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_token"}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		UserInfoURL: mockServer.URL,
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	userInfo, err := provider.GetUserInfo(context.Background(), "invalid_token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "userinfo request failed")
}

func TestGoogleProvider_GetProviderType(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID: "test_client_id",
	}

	provider := NewGoogleProvider(config)

	assert.Equal(t, OAuthProviderGoogle, provider.GetProviderType())
}

func TestNewGoogleProvider_AutoGenerateURLs(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// Verify URLs are auto-generated
	assert.NotNil(t, provider.oauth2Config)
	assert.Contains(t, provider.oauth2Config.Endpoint.AuthURL, "accounts.google.com")
	assert.Contains(t, provider.oauth2Config.Endpoint.AuthURL, "/o/oauth2/v2/auth")
	assert.Contains(t, provider.oauth2Config.Endpoint.TokenURL, "oauth2.googleapis.com")
	assert.Contains(t, provider.oauth2Config.Endpoint.TokenURL, "/token")
}

func TestNewGoogleProvider_DefaultScopes(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
		// Scopes not provided
	}

	provider := NewGoogleProvider(config).(*GoogleProvider)

	// Verify default scopes
	assert.Contains(t, provider.oauth2Config.Scopes, "openid")
	assert.Contains(t, provider.oauth2Config.Scopes, "profile")
	assert.Contains(t, provider.oauth2Config.Scopes, "email")
}
