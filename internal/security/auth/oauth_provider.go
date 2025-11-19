package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var (
	ErrOAuthProviderNotFound = errors.New("oauth provider not found")
	ErrOAuthStateMismatch     = errors.New("oauth state mismatch")
	ErrOAuthCodeExchange      = errors.New("oauth code exchange failed")
)

// OAuthProviderType represents supported OAuth providers
type OAuthProviderType string

const (
	OAuthProviderGoogle   OAuthProviderType = "google"
	OAuthProviderGitHub   OAuthProviderType = "github"
	OAuthProviderAzureAD  OAuthProviderType = "azuread"
	OAuthProviderAuth0    OAuthProviderType = "auth0"
	OAuthProviderGeneric  OAuthProviderType = "generic"
)

// OAuthUserInfo represents user information from OAuth provider
type OAuthUserInfo struct {
	ID       string
	Email    string
	Name     string
	Picture  string
	Provider OAuthProviderType
}

// OAuthProvider handles OAuth/OIDC authentication
type OAuthProvider interface {
	// GetAuthURL returns the authorization URL for OAuth flow
	GetAuthURL(ctx context.Context, state string) (string, error)
	
	// ExchangeCode exchanges authorization code for tokens
	ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error)
	
	// GetUserInfo retrieves user information from provider
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
	
	// GetProviderType returns the provider type
	GetProviderType() OAuthProviderType
}

// OAuthTokens represents OAuth tokens
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresIn    int
	TokenType    string
}

// OAuthProviderConfig holds configuration for OAuth provider
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Domain       string // For Auth0: domain like "dev-vertikon.us.auth0.com"
}

// OAuthManager manages multiple OAuth providers
type OAuthManager interface {
	// RegisterProvider registers an OAuth provider
	RegisterProvider(providerType OAuthProviderType, provider OAuthProvider)
	
	// GetProvider returns an OAuth provider by type
	GetProvider(providerType OAuthProviderType) (OAuthProvider, error)
	
	// GetAuthURL returns authorization URL for a provider
	GetAuthURL(ctx context.Context, providerType OAuthProviderType, state string) (string, error)
	
	// HandleCallback processes OAuth callback
	HandleCallback(ctx context.Context, providerType OAuthProviderType, code, state string) (*OAuthUserInfo, error)
}

// Manager implements OAuthManager
type OAuthManagerImpl struct {
	providers map[OAuthProviderType]OAuthProvider
	logger    *zap.Logger
}

// NewOAuthManager creates a new OAuthManager
func NewOAuthManager() OAuthManager {
	return &OAuthManagerImpl{
		providers: make(map[OAuthProviderType]OAuthProvider),
		logger:    logger.WithContext(context.TODO()),
	}
}

// RegisterProvider registers an OAuth provider
func (m *OAuthManagerImpl) RegisterProvider(providerType OAuthProviderType, provider OAuthProvider) {
	m.providers[providerType] = provider
	m.logger.Info("OAuth provider registered",
		zap.String("provider", string(providerType)),
	)
}

// GetProvider returns an OAuth provider by type
func (m *OAuthManagerImpl) GetProvider(providerType OAuthProviderType) (OAuthProvider, error) {
	provider, ok := m.providers[providerType]
	if !ok {
		return nil, ErrOAuthProviderNotFound
	}
	return provider, nil
}

// GetAuthURL returns authorization URL for a provider
func (m *OAuthManagerImpl) GetAuthURL(ctx context.Context, providerType OAuthProviderType, state string) (string, error) {
	provider, err := m.GetProvider(providerType)
	if err != nil {
		return "", err
	}

	authURL, err := provider.GetAuthURL(ctx, state)
	if err != nil {
		m.logger.Error("Failed to get auth URL",
			zap.String("provider", string(providerType)),
			zap.Error(err),
		)
		return "", err
	}

	return authURL, nil
}

// HandleCallback processes OAuth callback
func (m *OAuthManagerImpl) HandleCallback(ctx context.Context, providerType OAuthProviderType, code, state string) (*OAuthUserInfo, error) {
	provider, err := m.GetProvider(providerType)
	if err != nil {
		return nil, err
	}

	// Exchange code for tokens
	tokens, err := provider.ExchangeCode(ctx, code)
	if err != nil {
		m.logger.Error("OAuth code exchange failed",
			zap.String("provider", string(providerType)),
			zap.Error(err),
		)
		return nil, ErrOAuthCodeExchange
	}

	// Get user info
	userInfo, err := provider.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		m.logger.Error("Failed to get user info",
			zap.String("provider", string(providerType)),
			zap.Error(err),
		)
		return nil, err
	}

	m.logger.Info("OAuth callback processed successfully",
		zap.String("provider", string(providerType)),
		zap.String("user_id", userInfo.ID),
		zap.String("email", userInfo.Email),
	)

	return userInfo, nil
}

// GoogleProvider implements OAuthProvider for Google
type GoogleProvider struct {
	config       OAuthProviderConfig
	oauth2Config *oauth2.Config
	logger       *zap.Logger
	httpClient   *http.Client
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(config OAuthProviderConfig) OAuthProvider {
	// Build Google OAuth URLs if not provided
	authURL := config.AuthURL
	tokenURL := config.TokenURL
	userInfoURL := config.UserInfoURL

	if authURL == "" {
		authURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	if userInfoURL == "" {
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}

	// Default scopes for Google
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return &GoogleProvider{
		config:       config,
		oauth2Config: oauth2Config,
		logger:       logger.WithContext(context.TODO()),
		httpClient:   &http.Client{},
	}
}

// GetAuthURL returns the authorization URL for Google OAuth
func (p *GoogleProvider) GetAuthURL(ctx context.Context, state string) (string, error) {
	authURL := p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	p.logger.Debug("Google auth URL generated",
		zap.String("state", state),
	)
	return authURL, nil
}

// ExchangeCode exchanges authorization code for tokens
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		p.logger.Error("Google token exchange failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token from extra if available
	idToken := ""
	if extra := token.Extra("id_token"); extra != nil {
		if idTokenStr, ok := extra.(string); ok {
			idToken = idTokenStr
		}
	}

	oauthTokens := &OAuthTokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      idToken,
		TokenType:    token.TokenType,
	}

	// Calculate expires in seconds
	if !token.Expiry.IsZero() {
		now := time.Now()
		if token.Expiry.After(now) {
			oauthTokens.ExpiresIn = int(token.Expiry.Sub(now).Seconds())
		}
	}

	p.logger.Info("Google token exchange successful",
		zap.Bool("has_refresh_token", token.RefreshToken != ""),
		zap.Bool("has_id_token", idToken != ""),
	)

	return oauthTokens, nil
}

// GetUserInfo retrieves user information from Google
func (p *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := p.config.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("Google userinfo request failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Error("Google userinfo request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Google userinfo response
	var userInfoMap map[string]interface{}
	if err := json.Unmarshal(body, &userInfoMap); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo: %w", err)
	}

	userInfo := &OAuthUserInfo{
		Provider: OAuthProviderGoogle,
	}

	// Google uses "id" for user ID
	if id, ok := userInfoMap["id"].(string); ok {
		userInfo.ID = id
	}

	// Email
	if email, ok := userInfoMap["email"].(string); ok {
		userInfo.Email = email
	}

	// Name
	if name, ok := userInfoMap["name"].(string); ok {
		userInfo.Name = name
	}

	// Picture
	if picture, ok := userInfoMap["picture"].(string); ok {
		userInfo.Picture = picture
	}

	p.logger.Info("Google userinfo retrieved",
		zap.String("user_id", userInfo.ID),
		zap.String("email", userInfo.Email),
	)

	return userInfo, nil
}

// GetProviderType returns the provider type
func (p *GoogleProvider) GetProviderType() OAuthProviderType {
	return OAuthProviderGoogle
}

// GitHubProvider implements OAuthProvider for GitHub
type GitHubProvider struct {
	config       OAuthProviderConfig
	oauth2Config *oauth2.Config
	logger       *zap.Logger
	httpClient   *http.Client
}

// NewGitHubProvider creates a new GitHub OAuth provider
func NewGitHubProvider(config OAuthProviderConfig) OAuthProvider {
	// Build GitHub OAuth URLs if not provided
	authURL := config.AuthURL
	tokenURL := config.TokenURL
	userInfoURL := config.UserInfoURL

	if authURL == "" {
		authURL = "https://github.com/login/oauth/authorize"
	}
	if tokenURL == "" {
		tokenURL = "https://github.com/login/oauth/access_token"
	}
	if userInfoURL == "" {
		userInfoURL = "https://api.github.com/user"
	}

	// Default scopes for GitHub
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:email"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return &GitHubProvider{
		config:       config,
		oauth2Config: oauth2Config,
		logger:       logger.WithContext(context.TODO()),
		httpClient:   &http.Client{},
	}
}

// GetAuthURL returns the authorization URL for GitHub OAuth
func (p *GitHubProvider) GetAuthURL(ctx context.Context, state string) (string, error) {
	authURL := p.oauth2Config.AuthCodeURL(state)
	p.logger.Debug("GitHub auth URL generated",
		zap.String("state", state),
	)
	return authURL, nil
}

// ExchangeCode exchanges authorization code for tokens
func (p *GitHubProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		p.logger.Error("GitHub token exchange failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	oauthTokens := &OAuthTokens{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
	}

	// GitHub doesn't provide refresh tokens by default, but may in the future
	if token.RefreshToken != "" {
		oauthTokens.RefreshToken = token.RefreshToken
	}

	// Calculate expires in seconds
	if !token.Expiry.IsZero() {
		now := time.Now()
		if token.Expiry.After(now) {
			oauthTokens.ExpiresIn = int(token.Expiry.Sub(now).Seconds())
		}
	}

	p.logger.Info("GitHub token exchange successful")

	return oauthTokens, nil
}

// GetUserInfo retrieves user information from GitHub
func (p *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := p.config.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://api.github.com/user"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("GitHub userinfo request failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Error("GitHub userinfo request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse GitHub userinfo response
	var userInfoMap map[string]interface{}
	if err := json.Unmarshal(body, &userInfoMap); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo: %w", err)
	}

	userInfo := &OAuthUserInfo{
		Provider: OAuthProviderGitHub,
	}

	// GitHub uses "id" (as number) for user ID
	if id, ok := userInfoMap["id"].(float64); ok {
		userInfo.ID = fmt.Sprintf("%.0f", id)
	}

	// Name
	if name, ok := userInfoMap["name"].(string); ok {
		userInfo.Name = name
	} else if login, ok := userInfoMap["login"].(string); ok {
		userInfo.Name = login
	}

	// Email (may need separate API call if not public)
	if email, ok := userInfoMap["email"].(string); ok && email != "" {
		userInfo.Email = email
	} else {
		// Try to get email from emails endpoint
		email, err := p.getGitHubEmail(ctx, accessToken)
		if err == nil && email != "" {
			userInfo.Email = email
		}
	}

	// Avatar URL
	if avatarURL, ok := userInfoMap["avatar_url"].(string); ok {
		userInfo.Picture = avatarURL
	}

	p.logger.Info("GitHub userinfo retrieved",
		zap.String("user_id", userInfo.ID),
		zap.String("email", userInfo.Email),
	)

	return userInfo, nil
}

// getGitHubEmail retrieves email from GitHub's emails endpoint
func (p *GitHubProvider) getGitHubEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("emails request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []map[string]interface{}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	// Find primary email
	for _, email := range emails {
		if primary, ok := email["primary"].(bool); ok && primary {
			if emailStr, ok := email["email"].(string); ok {
				return emailStr, nil
			}
		}
	}

	// Return first email if no primary found
	if len(emails) > 0 {
		if emailStr, ok := emails[0]["email"].(string); ok {
			return emailStr, nil
		}
	}

	return "", errors.New("no email found")
}

// GetProviderType returns the provider type
func (p *GitHubProvider) GetProviderType() OAuthProviderType {
	return OAuthProviderGitHub
}

// AzureADProvider implements OAuthProvider for Azure AD
type AzureADProvider struct {
	config       OAuthProviderConfig
	oauth2Config *oauth2.Config
	logger       *zap.Logger
	httpClient   *http.Client
	tenantID     string
}

// NewAzureADProvider creates a new Azure AD OAuth provider
func NewAzureADProvider(config OAuthProviderConfig, tenantID string) OAuthProvider {
	// Build Azure AD OAuth URLs
	authURL := config.AuthURL
	tokenURL := config.TokenURL
	userInfoURL := config.UserInfoURL

	if tenantID == "" {
		tenantID = "common" // Use "common" for multi-tenant apps
	}

	if authURL == "" {
		authURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID)
	}
	if tokenURL == "" {
		tokenURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)
	}
	if userInfoURL == "" {
		userInfoURL = "https://graph.microsoft.com/v1.0/me"
	}

	// Default scopes for Azure AD
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email", "User.Read"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return &AzureADProvider{
		config:       config,
		oauth2Config: oauth2Config,
		logger:       logger.WithContext(context.TODO()),
		httpClient:   &http.Client{},
		tenantID:     tenantID,
	}
}

// GetAuthURL returns the authorization URL for Azure AD OAuth
func (p *AzureADProvider) GetAuthURL(ctx context.Context, state string) (string, error) {
	authURL := p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	p.logger.Debug("Azure AD auth URL generated",
		zap.String("state", state),
		zap.String("tenant_id", p.tenantID),
	)
	return authURL, nil
}

// ExchangeCode exchanges authorization code for tokens
func (p *AzureADProvider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		p.logger.Error("Azure AD token exchange failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token from extra if available
	idToken := ""
	if extra := token.Extra("id_token"); extra != nil {
		if idTokenStr, ok := extra.(string); ok {
			idToken = idTokenStr
		}
	}

	oauthTokens := &OAuthTokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      idToken,
		TokenType:    token.TokenType,
	}

	// Calculate expires in seconds
	if !token.Expiry.IsZero() {
		now := time.Now()
		if token.Expiry.After(now) {
			oauthTokens.ExpiresIn = int(token.Expiry.Sub(now).Seconds())
		}
	}

	p.logger.Info("Azure AD token exchange successful",
		zap.Bool("has_refresh_token", token.RefreshToken != ""),
		zap.Bool("has_id_token", idToken != ""),
	)

	return oauthTokens, nil
}

// GetUserInfo retrieves user information from Azure AD
func (p *AzureADProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := p.config.UserInfoURL
	if userInfoURL == "" {
		userInfoURL = "https://graph.microsoft.com/v1.0/me"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("Azure AD userinfo request failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Error("Azure AD userinfo request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Azure AD userinfo response (Microsoft Graph API)
	var userInfoMap map[string]interface{}
	if err := json.Unmarshal(body, &userInfoMap); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo: %w", err)
	}

	userInfo := &OAuthUserInfo{
		Provider: OAuthProviderAzureAD,
	}

	// Azure AD uses "id" for user ID
	if id, ok := userInfoMap["id"].(string); ok {
		userInfo.ID = id
	}

	// Email (can be "mail" or "userPrincipalName")
	if mail, ok := userInfoMap["mail"].(string); ok && mail != "" {
		userInfo.Email = mail
	} else if upn, ok := userInfoMap["userPrincipalName"].(string); ok {
		userInfo.Email = upn
	}

	// Name (can be "displayName" or constructed from givenName + surname)
	if displayName, ok := userInfoMap["displayName"].(string); ok {
		userInfo.Name = displayName
	} else {
		var nameParts []string
		if givenName, ok := userInfoMap["givenName"].(string); ok {
			nameParts = append(nameParts, givenName)
		}
		if surname, ok := userInfoMap["surname"].(string); ok {
			nameParts = append(nameParts, surname)
		}
		if len(nameParts) > 0 {
			userInfo.Name = strings.Join(nameParts, " ")
		}
	}

	// Picture (may need separate API call)
	if photo, ok := userInfoMap["photo"].(string); ok {
		userInfo.Picture = photo
	}

	p.logger.Info("Azure AD userinfo retrieved",
		zap.String("user_id", userInfo.ID),
		zap.String("email", userInfo.Email),
	)

	return userInfo, nil
}

// GetProviderType returns the provider type
func (p *AzureADProvider) GetProviderType() OAuthProviderType {
	return OAuthProviderAzureAD
}

// Auth0Provider implements OAuthProvider for Auth0
type Auth0Provider struct {
	config     OAuthProviderConfig
	oauth2Config *oauth2.Config
	logger     *zap.Logger
	httpClient *http.Client
}

// NewAuth0Provider creates a new Auth0 OAuth provider
func NewAuth0Provider(config OAuthProviderConfig) OAuthProvider {
	domain := config.Domain
	if domain == "" && config.AuthURL != "" {
		// Extract domain from AuthURL if not provided
		if u, err := url.Parse(config.AuthURL); err == nil {
			domain = u.Host
		}
	}

	// Build Auth0 URLs if not provided
	authURL := config.AuthURL
	tokenURL := config.TokenURL

	if domain != "" {
		if authURL == "" {
			authURL = fmt.Sprintf("https://%s/authorize", domain)
		}
		if tokenURL == "" {
			tokenURL = fmt.Sprintf("https://%s/oauth/token", domain)
		}
		if config.UserInfoURL == "" {
			config.UserInfoURL = fmt.Sprintf("https://%s/userinfo", domain)
		}
	}

	// Default scopes for Auth0
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return &Auth0Provider{
		config:       config,
		oauth2Config: oauth2Config,
		logger:       logger.WithContext(context.TODO()),
		httpClient:   &http.Client{},
	}
}

// GetAuthURL returns the authorization URL for Auth0 OAuth
func (p *Auth0Provider) GetAuthURL(ctx context.Context, state string) (string, error) {
	authURL := p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	p.logger.Debug("Auth0 auth URL generated",
		zap.String("state", state),
	)
	return authURL, nil
}

// ExchangeCode exchanges authorization code for tokens
func (p *Auth0Provider) ExchangeCode(ctx context.Context, code string) (*OAuthTokens, error) {
	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		p.logger.Error("Auth0 token exchange failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract ID token from extra if available
	idToken := ""
	if extra := token.Extra("id_token"); extra != nil {
		if idTokenStr, ok := extra.(string); ok {
			idToken = idTokenStr
		}
	}

	oauthTokens := &OAuthTokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      idToken,
		TokenType:    token.TokenType,
	}

	// Calculate expires in seconds
	if !token.Expiry.IsZero() {
		now := time.Now()
		if token.Expiry.After(now) {
			oauthTokens.ExpiresIn = int(token.Expiry.Sub(now).Seconds())
		}
	}

	p.logger.Info("Auth0 token exchange successful",
		zap.Bool("has_refresh_token", token.RefreshToken != ""),
		zap.Bool("has_id_token", idToken != ""),
	)

	return oauthTokens, nil
}

// GetUserInfo retrieves user information from Auth0
func (p *Auth0Provider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	userInfoURL := p.config.UserInfoURL
	if userInfoURL == "" && p.config.Domain != "" {
		userInfoURL = fmt.Sprintf("https://%s/userinfo", p.config.Domain)
	}
	if userInfoURL == "" {
		return nil, errors.New("userinfo URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("Auth0 userinfo request failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.logger.Error("Auth0 userinfo request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Auth0 userinfo response
	var userInfoMap map[string]interface{}
	if err := json.Unmarshal(body, &userInfoMap); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo: %w", err)
	}

	// Extract user information from Auth0 response
	userInfo := &OAuthUserInfo{
		Provider: OAuthProviderAuth0,
	}

	// Auth0 uses "sub" for user ID
	if sub, ok := userInfoMap["sub"].(string); ok {
		userInfo.ID = sub
	}

	// Email
	if email, ok := userInfoMap["email"].(string); ok {
		userInfo.Email = email
	}

	// Name (can be "name" or constructed from given_name + family_name)
	if name, ok := userInfoMap["name"].(string); ok {
		userInfo.Name = name
	} else {
		var nameParts []string
		if givenName, ok := userInfoMap["given_name"].(string); ok {
			nameParts = append(nameParts, givenName)
		}
		if familyName, ok := userInfoMap["family_name"].(string); ok {
			nameParts = append(nameParts, familyName)
		}
		if len(nameParts) > 0 {
			userInfo.Name = strings.Join(nameParts, " ")
		}
	}

	// Picture
	if picture, ok := userInfoMap["picture"].(string); ok {
		userInfo.Picture = picture
	}

	p.logger.Info("Auth0 userinfo retrieved",
		zap.String("user_id", userInfo.ID),
		zap.String("email", userInfo.Email),
	)

	return userInfo, nil
}

// GetProviderType returns the provider type
func (p *Auth0Provider) GetProviderType() OAuthProviderType {
	return OAuthProviderAuth0
}
