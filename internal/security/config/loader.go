package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Loader loads security configuration from YAML files
type Loader struct {
	viper *viper.Viper
}

// NewLoader creates a new security configuration loader
func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigType("yaml")

	// Add config paths
	v.AddConfigPath("config/security")
	v.AddConfigPath("./config/security")
	v.AddConfigPath(".")

	// Environment variables
	v.SetEnvPrefix("MCP_HULK")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return &Loader{viper: v}
}

// LoadAuthConfig loads authentication configuration
func (l *Loader) LoadAuthConfig() (*AuthConfig, error) {
	l.viper.SetConfigName("auth")

	// Set defaults
	l.setAuthDefaults()

	// Read config file
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading auth config file: %w", err)
		}
		logger.Info("No auth config file found, using defaults and environment variables")
	}

	var cfg AuthConfig
	if err := l.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling auth config: %w", err)
	}

	// Resolve environment variables
	l.resolveEnvVars(&cfg)

	logger.Info("Auth configuration loaded",
		zap.String("config_file", l.viper.ConfigFileUsed()),
	)

	return &cfg, nil
}

// LoadRBACConfig loads RBAC configuration
func (l *Loader) LoadRBACConfig() (*RBACConfig, error) {
	l.viper.SetConfigName("rbac")

	// Set defaults
	l.setRBACDefaults()

	// Read config file
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading rbac config file: %w", err)
		}
		logger.Info("No rbac config file found, using defaults")
	}

	var cfg RBACConfig
	if err := l.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling rbac config: %w", err)
	}

	logger.Info("RBAC configuration loaded",
		zap.String("config_file", l.viper.ConfigFileUsed()),
	)

	return &cfg, nil
}

// LoadEncryptionConfig loads encryption configuration
func (l *Loader) LoadEncryptionConfig() (*EncryptionConfig, error) {
	l.viper.SetConfigName("encryption")

	// Set defaults
	l.setEncryptionDefaults()

	// Read config file
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading encryption config file: %w", err)
		}
		logger.Info("No encryption config file found, using defaults")
	}

	var cfg EncryptionConfig
	if err := l.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling encryption config: %w", err)
	}

	logger.Info("Encryption configuration loaded",
		zap.String("config_file", l.viper.ConfigFileUsed()),
	)

	return &cfg, nil
}

// setAuthDefaults sets default values for auth configuration
func (l *Loader) setAuthDefaults() {
	l.viper.SetDefault("jwt.secret_key", "change-me-in-production-use-strong-random-key-32-bytes-minimum")
	l.viper.SetDefault("jwt.signing_method", "HS256")
	l.viper.SetDefault("jwt.token_ttl", "1h")
	l.viper.SetDefault("jwt.refresh_ttl", "24h")
	l.viper.SetDefault("session.ttl", "24h")
	l.viper.SetDefault("session.max_sessions_per_user", 5)
}

// setRBACDefaults sets default values for RBAC configuration
func (l *Loader) setRBACDefaults() {
	// Default roles
	l.viper.SetDefault("roles", []map[string]interface{}{
		{
			"id":   "admin",
			"name": "Administrator",
			"permissions": []map[string]interface{}{
				{"resource": "*", "action": "*"},
			},
		},
		{
			"id":   "user",
			"name": "User",
			"permissions": []map[string]interface{}{
				{"resource": "mcp", "action": "read"},
				{"resource": "mcp", "action": "create"},
			},
		},
	})
}

// setEncryptionDefaults sets default values for encryption configuration
func (l *Loader) setEncryptionDefaults() {
	l.viper.SetDefault("algorithm", "AES-256-GCM")
	l.viper.SetDefault("key_rotation_ttl", "720h") // 30 days
	l.viper.SetDefault("rsa_key_size", 2048)
	l.viper.SetDefault("certificate_ttl", "8760h") // 1 year
}

// resolveEnvVars resolves environment variable placeholders in config
func (l *Loader) resolveEnvVars(cfg *AuthConfig) {
	// Resolve JWT secret
	if strings.HasPrefix(cfg.JWT.SecretKey, "${") {
		cfg.JWT.SecretKey = l.resolveEnvVar(cfg.JWT.SecretKey, "change-me-in-production")
	}

	// Resolve OAuth client secrets
	if cfg.OAuth.Auth0.Enabled {
		if strings.HasPrefix(cfg.OAuth.Auth0.ClientSecret, "${") {
			envVar := extractEnvVarName(cfg.OAuth.Auth0.ClientSecret)
			cfg.OAuth.Auth0.ClientSecret = os.Getenv(envVar)
		}
	}

	if cfg.OAuth.Google.Enabled {
		if strings.HasPrefix(cfg.OAuth.Google.ClientSecret, "${") {
			envVar := extractEnvVarName(cfg.OAuth.Google.ClientSecret)
			cfg.OAuth.Google.ClientSecret = os.Getenv(envVar)
		}
	}

	if cfg.OAuth.GitHub.Enabled {
		if strings.HasPrefix(cfg.OAuth.GitHub.ClientSecret, "${") {
			envVar := extractEnvVarName(cfg.OAuth.GitHub.ClientSecret)
			cfg.OAuth.GitHub.ClientSecret = os.Getenv(envVar)
		}
	}

	if cfg.OAuth.AzureAD.Enabled {
		if strings.HasPrefix(cfg.OAuth.AzureAD.ClientSecret, "${") {
			envVar := extractEnvVarName(cfg.OAuth.AzureAD.ClientSecret)
			cfg.OAuth.AzureAD.ClientSecret = os.Getenv(envVar)
		}
	}
}

// resolveEnvVar resolves a single environment variable placeholder
func (l *Loader) resolveEnvVar(value, defaultValue string) string {
	if !strings.HasPrefix(value, "${") {
		return value
	}

	envVar := extractEnvVarName(value)
	if val := os.Getenv(envVar); val != "" {
		return val
	}

	return defaultValue
}

// extractEnvVarName extracts environment variable name from ${VAR:default} format
func extractEnvVarName(value string) string {
	value = strings.TrimPrefix(value, "${")
	value = strings.TrimSuffix(value, "}")

	parts := strings.Split(value, ":")
	return parts[0]
}

// ParseDuration parses a duration string (e.g., "1h", "24h")
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
