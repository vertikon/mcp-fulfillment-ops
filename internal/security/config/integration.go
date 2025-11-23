package config

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/security/auth"
	"github.com/vertikon/mcp-fulfillment-ops/internal/security/encryption"
	"github.com/vertikon/mcp-fulfillment-ops/internal/security/rbac"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// LoadAndInitializeAuth loads auth config and initializes auth components
func LoadAndInitializeAuth(loader *Loader, userStore auth.UserStore) (auth.AuthManager, error) {
	cfg, err := loader.LoadAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load auth config: %w", err)
	}

	// Parse durations
	tokenTTL, err := ParseDuration(cfg.JWT.TokenTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid token_ttl: %w", err)
	}

	refreshTTL, err := ParseDuration(cfg.JWT.RefreshTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh_ttl: %w", err)
	}

	sessionTTL, err := ParseDuration(cfg.Session.TTL)
	if err != nil {
		return nil, fmt.Errorf("invalid session_ttl: %w", err)
	}

	// Initialize Token Manager
	tokenConfig := auth.TokenManagerConfig{
		SecretKey:     cfg.JWT.SecretKey,
		SigningMethod: cfg.JWT.SigningMethod,
		TokenTTL:      tokenTTL,
		RefreshTTL:    refreshTTL,
	}
	tokenManager := auth.NewTokenManager(tokenConfig)

	// Initialize Session Manager
	sessionStore := auth.NewInMemorySessionStore()
	sessionConfig := auth.SessionManagerConfig{
		SessionTTL:  sessionTTL,
		MaxSessions: cfg.Session.MaxSessionsPerUser,
	}
	sessionManager := auth.NewSessionManager(sessionStore, sessionConfig)

	// Initialize RBAC Manager (will be loaded separately)
	// Create minimal RBAC components for now
	roleStore := rbac.NewInMemoryRoleStore()
	roleManager := rbac.NewRoleManager(roleStore)
	permissionChecker := rbac.NewPermissionChecker()
	policyEnforcer := rbac.NewPolicyEnforcer(rbac.PolicyEnforcerConfig{})
	rbacManager := rbac.NewRBACManager(roleManager, permissionChecker, policyEnforcer)

	// Initialize Auth Manager
	authManager := auth.NewAuthManager(tokenManager, sessionManager, rbacManager, userStore)

	// Initialize OAuth Manager
	oauthManager := auth.NewOAuthManager()

	// Register OAuth providers if enabled
	if cfg.OAuth.Auth0.Enabled {
		auth0Config := auth.OAuthProviderConfig{
			Domain:       cfg.OAuth.Auth0.Domain,
			ClientID:     cfg.OAuth.Auth0.ClientID,
			ClientSecret: cfg.OAuth.Auth0.ClientSecret,
			RedirectURL:  cfg.OAuth.Auth0.RedirectURL,
			Scopes:       cfg.OAuth.Auth0.Scopes,
			AuthURL:      cfg.OAuth.Auth0.AuthURL,
			TokenURL:     cfg.OAuth.Auth0.TokenURL,
			UserInfoURL:  cfg.OAuth.Auth0.UserInfoURL,
		}
		auth0Provider := auth.NewAuth0Provider(auth0Config)
		oauthManager.RegisterProvider(auth.OAuthProviderAuth0, auth0Provider)
		logger.Info("Auth0 OAuth provider registered")
	}

	if cfg.OAuth.Google.Enabled {
		googleConfig := auth.OAuthProviderConfig{
			ClientID:     cfg.OAuth.Google.ClientID,
			ClientSecret: cfg.OAuth.Google.ClientSecret,
			RedirectURL:  cfg.OAuth.Google.RedirectURL,
			Scopes:       cfg.OAuth.Google.Scopes,
			AuthURL:      cfg.OAuth.Google.AuthURL,
			TokenURL:     cfg.OAuth.Google.TokenURL,
			UserInfoURL:  cfg.OAuth.Google.UserInfoURL,
		}
		googleProvider := auth.NewGoogleProvider(googleConfig)
		oauthManager.RegisterProvider(auth.OAuthProviderGoogle, googleProvider)
		logger.Info("Google OAuth provider registered")
	}

	if cfg.OAuth.GitHub.Enabled {
		githubConfig := auth.OAuthProviderConfig{
			ClientID:     cfg.OAuth.GitHub.ClientID,
			ClientSecret: cfg.OAuth.GitHub.ClientSecret,
			RedirectURL:  cfg.OAuth.GitHub.RedirectURL,
			Scopes:       cfg.OAuth.GitHub.Scopes,
			AuthURL:      cfg.OAuth.GitHub.AuthURL,
			TokenURL:     cfg.OAuth.GitHub.TokenURL,
			UserInfoURL:  cfg.OAuth.GitHub.UserInfoURL,
		}
		githubProvider := auth.NewGitHubProvider(githubConfig)
		oauthManager.RegisterProvider(auth.OAuthProviderGitHub, githubProvider)
		logger.Info("GitHub OAuth provider registered")
	}

	if cfg.OAuth.AzureAD.Enabled {
		azureConfig := auth.OAuthProviderConfig{
			ClientID:     cfg.OAuth.AzureAD.ClientID,
			ClientSecret: cfg.OAuth.AzureAD.ClientSecret,
			RedirectURL:  cfg.OAuth.AzureAD.RedirectURL,
			Scopes:       cfg.OAuth.AzureAD.Scopes,
			AuthURL:      cfg.OAuth.AzureAD.AuthURL,
			TokenURL:     cfg.OAuth.AzureAD.TokenURL,
			UserInfoURL:  cfg.OAuth.AzureAD.UserInfoURL,
		}
		azureProvider := auth.NewAzureADProvider(azureConfig, cfg.OAuth.AzureAD.TenantID)
		oauthManager.RegisterProvider(auth.OAuthProviderAzureAD, azureProvider)
		logger.Info("Azure AD OAuth provider registered")
	}

	logger.Info("Auth configuration loaded and initialized",
		zap.String("jwt_method", cfg.JWT.SigningMethod),
		zap.String("token_ttl", cfg.JWT.TokenTTL),
	)

	return authManager, nil
}

// LoadAndInitializeRBAC loads RBAC config and initializes RBAC components
func LoadAndInitializeRBAC(loader *Loader) (rbac.RBACManager, error) {
	cfg, err := loader.LoadRBACConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load RBAC config: %w", err)
	}

	// Initialize Role Manager
	roleStore := rbac.NewInMemoryRoleStore()
	roleManager := rbac.NewRoleManager(roleStore)

	// Load roles from config
	ctx := context.Background()
	for _, roleCfg := range cfg.Roles {
		role := &rbac.Role{
			ID:          roleCfg.ID,
			Name:        roleCfg.Name,
			Description: roleCfg.Description,
		}

		// Convert permissions
		for _, permCfg := range roleCfg.Permissions {
			role.Permissions = append(role.Permissions, rbac.Permission{
				Resource: permCfg.Resource,
				Action:   permCfg.Action,
			})
		}

		if err := roleManager.CreateRole(ctx, role); err != nil {
			logger.Warn("Failed to create role from config",
				zap.String("role_id", roleCfg.ID),
				zap.Error(err),
			)
		}
	}

	// Initialize Permission Checker
	permissionChecker := rbac.NewPermissionChecker()

	// Register overrides
	for _, overrideCfg := range cfg.Overrides {
		// Convert effect string to PolicyEffect
		effect := rbac.EffectAllow
		if overrideCfg.Effect == "deny" {
			effect = rbac.EffectDeny
		}
		override := rbac.PermissionOverride{
			ResourcePattern: overrideCfg.Resource,
			ActionPattern:   overrideCfg.Action,
			Effect:          effect,
		}
		permissionChecker.RegisterOverride(override)
	}

	// Initialize Policy Enforcer
	policyEnforcer := rbac.NewPolicyEnforcer(rbac.PolicyEnforcerConfig{})

	// Load policies from config
	// Note: Full policy loading requires proper PolicyCondition construction
	// This is a simplified version - full implementation would parse conditions properly
	for _, policyCfg := range cfg.Policies {
		policy := &rbac.Policy{
			ID:          policyCfg.ID,
			Description: policyCfg.Description,
			Priority:    policyCfg.Priority,
		}

		// Convert rules - simplified version
		// Full implementation would need to construct PolicyCondition objects
		for _, ruleCfg := range policyCfg.Rules {
			// Convert effect string to PolicyEffect
			effect := rbac.EffectAllow
			if ruleCfg.Effect == "deny" {
				effect = rbac.EffectDeny
			}

			rule := rbac.PolicyRule{
				Resource:    ruleCfg.Resource,
				Action:      ruleCfg.Action,
				Effect:      effect,
				Description: ruleCfg.Description,
				// Conditions would need to be constructed from ruleCfg.Condition and ruleCfg.Params
				// For now, we'll leave it empty - full implementation would parse conditions
				Conditions: []rbac.PolicyCondition{},
			}
			policy.Rules = append(policy.Rules, rule)
		}

		if err := policyEnforcer.Register(policy); err != nil {
			logger.Warn("Failed to register policy from config",
				zap.String("policy_id", policyCfg.ID),
				zap.Error(err),
			)
		}
	}

	// Initialize RBAC Manager
	rbacManager := rbac.NewRBACManager(roleManager, permissionChecker, policyEnforcer)

	logger.Info("RBAC configuration loaded and initialized",
		zap.Int("roles_count", len(cfg.Roles)),
		zap.Int("policies_count", len(cfg.Policies)),
		zap.Int("overrides_count", len(cfg.Overrides)),
	)

	return rbacManager, nil
}

// LoadAndInitializeEncryption loads encryption config and initializes encryption components
func LoadAndInitializeEncryption(loader *Loader) (encryption.EncryptionManager, encryption.KeyManager, error) {
	cfg, err := loader.LoadEncryptionConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load encryption config: %w", err)
	}

	// Parse rotation TTL
	rotationTTL, err := ParseDuration(cfg.KeyRotationTTL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid key_rotation_ttl: %w", err)
	}

	// Initialize Key Manager
	keyConfig := encryption.KeyManagerConfig{
		RotationTTL: rotationTTL,
		KeySize:     cfg.RSAKeySize, // Note: Field is KeySize, not RSAKeySize
	}
	keyManager := encryption.NewKeyManager(keyConfig)

	// Initialize Encryption Manager
	encryptionManager := encryption.NewEncryptionManager(keyManager)

	logger.Info("Encryption configuration loaded and initialized",
		zap.String("algorithm", cfg.Algorithm),
		zap.Int("rsa_key_size", cfg.RSAKeySize),
		zap.String("key_rotation_ttl", cfg.KeyRotationTTL),
	)

	return encryptionManager, keyManager, nil
}
