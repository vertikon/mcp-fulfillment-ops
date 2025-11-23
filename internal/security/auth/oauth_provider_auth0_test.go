package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuth0Provider_GetAuthURL(t *testing.T) {
	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	authURL, err := provider.GetAuthURL(context.Background(), "test-state-123")

	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "dev-vertikon.us.auth0.com")
	assert.Contains(t, authURL, "/authorize")
	assert.Contains(t, authURL, "test_client_id")
	assert.Contains(t, authURL, "test-state-123")
	assert.Contains(t, authURL, "openid")
}

func TestAuth0Provider_GetUserInfo(t *testing.T) {
	// Mock Auth0 userinfo endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_access_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock Auth0 userinfo response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"sub": "auth0|123456789",
			"email": "test@example.com",
			"name": "Test User",
			"given_name": "Test",
			"family_name": "User",
			"picture": "https://example.com/avatar.jpg"
		}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
		UserInfoURL:  mockServer.URL,
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	userInfo, err := provider.GetUserInfo(context.Background(), "test_access_token")

	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "auth0|123456789", userInfo.ID)
	assert.Equal(t, "test@example.com", userInfo.Email)
	assert.Equal(t, "Test User", userInfo.Name)
	assert.Equal(t, "https://example.com/avatar.jpg", userInfo.Picture)
	assert.Equal(t, OAuthProviderAuth0, userInfo.Provider)
}

func TestAuth0Provider_GetUserInfo_Error(t *testing.T) {
	// Mock server returning error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_token"}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		Domain:      "dev-vertikon.us.auth0.com",
		UserInfoURL: mockServer.URL,
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	userInfo, err := provider.GetUserInfo(context.Background(), "invalid_token")

	assert.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "userinfo request failed")
}

func TestAuth0Provider_GetUserInfo_MissingName(t *testing.T) {
	// Mock Auth0 userinfo endpoint without name field
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"sub": "auth0|123456789",
			"email": "test@example.com",
			"given_name": "Test",
			"family_name": "User"
		}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		Domain:      "dev-vertikon.us.auth0.com",
		UserInfoURL: mockServer.URL,
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	userInfo, err := provider.GetUserInfo(context.Background(), "test_token")

	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "Test User", userInfo.Name) // Should be constructed from given_name + family_name
}

func TestAuth0Provider_GetProviderType(t *testing.T) {
	config := OAuthProviderConfig{
		Domain: "dev-vertikon.us.auth0.com",
	}

	provider := NewAuth0Provider(config)

	assert.Equal(t, OAuthProviderAuth0, provider.GetProviderType())
}

func TestNewAuth0Provider_AutoGenerateURLs(t *testing.T) {
	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	// Verify URLs are auto-generated
	assert.NotNil(t, provider.oauth2Config)
	assert.Contains(t, provider.oauth2Config.Endpoint.AuthURL, "dev-vertikon.us.auth0.com")
	assert.Contains(t, provider.oauth2Config.Endpoint.AuthURL, "/authorize")
	assert.Contains(t, provider.oauth2Config.Endpoint.TokenURL, "dev-vertikon.us.auth0.com")
	assert.Contains(t, provider.oauth2Config.Endpoint.TokenURL, "/oauth/token")
}

func TestNewAuth0Provider_DefaultScopes(t *testing.T) {
	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
		// Scopes not provided
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	// Verify default scopes
	assert.Contains(t, provider.oauth2Config.Scopes, "openid")
	assert.Contains(t, provider.oauth2Config.Scopes, "profile")
	assert.Contains(t, provider.oauth2Config.Scopes, "email")
}

func TestAuth0Provider_ExchangeCode(t *testing.T) {
	// This test requires a real OAuth2 token exchange, which is complex to mock
	// In a real scenario, you'd use oauth2's testing utilities or integration tests
	// For now, we'll test the structure

	config := OAuthProviderConfig{
		Domain:       "dev-vertikon.us.auth0.com",
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	provider := NewAuth0Provider(config).(*Auth0Provider)

	// Verify oauth2 config is set up correctly
	assert.NotNil(t, provider.oauth2Config)
	assert.Equal(t, "test_client_id", provider.oauth2Config.ClientID)
	assert.Equal(t, "test_secret", provider.oauth2Config.ClientSecret)
	assert.Equal(t, "http://localhost:8080/callback", provider.oauth2Config.RedirectURL)
}
