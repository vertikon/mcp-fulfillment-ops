package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureADProvider_GetAuthURL(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"openid", "profile", "email"},
	}

	provider := NewAzureADProvider(config, "test-tenant-id").(*AzureADProvider)

	authURL, err := provider.GetAuthURL(context.Background(), "test-state-123")

	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Contains(t, authURL, "login.microsoftonline.com")
	assert.Contains(t, authURL, "test-tenant-id")
	assert.Contains(t, authURL, "/oauth2/v2.0/authorize")
	assert.Contains(t, authURL, "test_client_id")
	assert.Contains(t, authURL, "test-state-123")
}

func TestAzureADProvider_GetUserInfo(t *testing.T) {
	// Mock Microsoft Graph API endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test_access_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock Microsoft Graph user response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "azure-user-123",
			"mail": "test@example.com",
			"userPrincipalName": "test@example.com",
			"displayName": "Test User",
			"givenName": "Test",
			"surname": "User"
		}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_client_secret",
		RedirectURL:  "http://localhost:8080/callback",
		UserInfoURL:  mockServer.URL,
	}

	provider := NewAzureADProvider(config, "test-tenant").(*AzureADProvider)

	userInfo, err := provider.GetUserInfo(context.Background(), "test_access_token")

	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "azure-user-123", userInfo.ID)
	assert.Equal(t, "test@example.com", userInfo.Email)
	assert.Equal(t, "Test User", userInfo.Name)
	assert.Equal(t, OAuthProviderAzureAD, userInfo.Provider)
}

func TestAzureADProvider_GetUserInfo_UPNFallback(t *testing.T) {
	// Mock Microsoft Graph API without mail field
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "azure-user-123",
			"userPrincipalName": "test@example.com",
			"displayName": "Test User"
		}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		UserInfoURL: mockServer.URL,
	}

	provider := NewAzureADProvider(config, "test-tenant").(*AzureADProvider)

	userInfo, err := provider.GetUserInfo(context.Background(), "test_token")

	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "test@example.com", userInfo.Email) // Should use userPrincipalName
}

func TestAzureADProvider_GetUserInfo_ConstructedName(t *testing.T) {
	// Mock Microsoft Graph API without displayName
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "azure-user-123",
			"mail": "test@example.com",
			"givenName": "Test",
			"surname": "User"
		}`))
	}))
	defer mockServer.Close()

	config := OAuthProviderConfig{
		UserInfoURL: mockServer.URL,
	}

	provider := NewAzureADProvider(config, "test-tenant").(*AzureADProvider)

	userInfo, err := provider.GetUserInfo(context.Background(), "test_token")

	assert.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "Test User", userInfo.Name) // Should be constructed from givenName + surname
}

func TestAzureADProvider_GetProviderType(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID: "test_client_id",
	}

	provider := NewAzureADProvider(config, "test-tenant")

	assert.Equal(t, OAuthProviderAzureAD, provider.GetProviderType())
}

func TestNewAzureADProvider_DefaultTenant(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
	}

	provider := NewAzureADProvider(config, "").(*AzureADProvider)

	// Should use "common" as default tenant
	assert.Contains(t, provider.oauth2Config.Endpoint.AuthURL, "common")
}

func TestNewAzureADProvider_DefaultScopes(t *testing.T) {
	config := OAuthProviderConfig{
		ClientID:     "test_client_id",
		ClientSecret: "test_secret",
		RedirectURL:  "http://localhost:8080/callback",
		// Scopes not provided
	}

	provider := NewAzureADProvider(config, "test-tenant").(*AzureADProvider)

	// Verify default scopes
	assert.Contains(t, provider.oauth2Config.Scopes, "openid")
	assert.Contains(t, provider.oauth2Config.Scopes, "profile")
	assert.Contains(t, provider.oauth2Config.Scopes, "email")
	assert.Contains(t, provider.oauth2Config.Scopes, "User.Read")
}
