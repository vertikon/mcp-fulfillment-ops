package config

// Note: This package provides YAML configuration loading for security components.
// Full integration with managers is pending resolution of type conflicts in encryption package.

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWT     JWTConfig     `mapstructure:"jwt"`
	Session SessionConfig `mapstructure:"session"`
	OAuth   OAuthConfig   `mapstructure:"oauth"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey     string `mapstructure:"secret_key"`
	SigningMethod string `mapstructure:"signing_method"` // HS256 or RS256
	TokenTTL      string `mapstructure:"token_ttl"`
	RefreshTTL    string `mapstructure:"refresh_ttl"`
}

// SessionConfig represents session configuration
type SessionConfig struct {
	TTL                string `mapstructure:"ttl"`
	MaxSessionsPerUser int    `mapstructure:"max_sessions_per_user"`
}

// OAuthConfig represents OAuth providers configuration
type OAuthConfig struct {
	Auth0   OAuthProviderConfig `mapstructure:"auth0"`
	Google  OAuthProviderConfig `mapstructure:"google"`
	GitHub  OAuthProviderConfig `mapstructure:"github"`
	AzureAD AzureADConfig       `mapstructure:"azuread"`
}

// OAuthProviderConfig represents a single OAuth provider configuration
type OAuthProviderConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	Domain       string   `mapstructure:"domain,omitempty"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
	AuthURL      string   `mapstructure:"auth_url,omitempty"`
	TokenURL     string   `mapstructure:"token_url,omitempty"`
	UserInfoURL  string   `mapstructure:"userinfo_url,omitempty"`
}

// AzureADConfig extends OAuthProviderConfig with tenant ID
type AzureADConfig struct {
	OAuthProviderConfig `mapstructure:",squash"`
	TenantID            string `mapstructure:"tenant_id"`
}

// RBACConfig represents RBAC configuration
type RBACConfig struct {
	Roles     []RoleConfig     `mapstructure:"roles"`
	Policies  []PolicyConfig   `mapstructure:"policies"`
	Overrides []OverrideConfig `mapstructure:"overrides"`
}

// RoleConfig represents a role configuration
type RoleConfig struct {
	ID          string             `mapstructure:"id"`
	Name        string             `mapstructure:"name"`
	Description string             `mapstructure:"description,omitempty"`
	Permissions []PermissionConfig `mapstructure:"permissions"`
}

// PermissionConfig represents a permission configuration
type PermissionConfig struct {
	Resource string `mapstructure:"resource"`
	Action   string `mapstructure:"action"`
}

// PolicyConfig represents a policy configuration
type PolicyConfig struct {
	ID          string             `mapstructure:"id"`
	Name        string             `mapstructure:"name"`
	Description string             `mapstructure:"description,omitempty"`
	Priority    int                `mapstructure:"priority"`
	Rules       []PolicyRuleConfig `mapstructure:"rules"`
}

// PolicyRuleConfig represents a policy rule configuration
type PolicyRuleConfig struct {
	Resource    string                 `mapstructure:"resource"`
	Action      string                 `mapstructure:"action"`
	Condition   string                 `mapstructure:"condition"`
	Params      map[string]interface{} `mapstructure:"params"`
	Effect      string                 `mapstructure:"effect"` // allow or deny
	Description string                 `mapstructure:"description,omitempty"`
}

// OverrideConfig represents a permission override configuration
type OverrideConfig struct {
	ID        string                 `mapstructure:"id"`
	Resource  string                 `mapstructure:"resource"`
	Action    string                 `mapstructure:"action"`
	Condition map[string]interface{} `mapstructure:"condition,omitempty"`
	Effect    string                 `mapstructure:"effect"` // allow or deny
}

// EncryptionConfig represents encryption configuration
type EncryptionConfig struct {
	Algorithm      string    `mapstructure:"algorithm"`
	KeyRotationTTL string    `mapstructure:"key_rotation_ttl"`
	RSAKeySize     int       `mapstructure:"rsa_key_size"`
	CertificateTTL string    `mapstructure:"certificate_ttl"`
	KMS            KMSConfig `mapstructure:"kms,omitempty"`
}

// KMSConfig represents KMS configuration
type KMSConfig struct {
	Provider string            `mapstructure:"provider"` // aws, gcp, vault, none
	Config   map[string]string `mapstructure:"config"`
}
