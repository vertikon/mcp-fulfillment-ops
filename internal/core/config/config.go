// Package config provides configuration loading and validation
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

// Config represents the application configuration
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	AI        AIConfig        `mapstructure:"ai"`
	Paths     PathsConfig     `mapstructure:"paths"`
	Features  FeatureConfig   `mapstructure:"features"`
	Engine    EngineConfig    `mapstructure:"engine"`
	Cache     CacheConfig     `mapstructure:"cache"`
	NATS      NATSConfig      `mapstructure:"nats"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	MCP       MCPConfig       `mapstructure:"mcp"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Host         string        `mapstructure:"host"`
}

// EngineConfig represents engine configuration
type EngineConfig struct {
	Workers   interface{}   `mapstructure:"workers"` // "auto" or int
	QueueSize int           `mapstructure:"queue_size"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	L1Size int           `mapstructure:"l1_size"`
	L2TTL  time.Duration `mapstructure:"l2_ttl"`
	L3Path string        `mapstructure:"l3_path"`
}

// NATSConfig represents NATS configuration
type NATSConfig struct {
	URLs []string `mapstructure:"urls"`
	User string   `mapstructure:"user"`
	Pass string   `mapstructure:"pass"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// TelemetryConfig represents telemetry configuration
type TelemetryConfig struct {
	Tracing TracingConfig `mapstructure:"tracing"`
	Metrics MetricsConfig `mapstructure:"metrics"`
}

// TracingConfig represents tracing configuration
type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Exporter string `mapstructure:"exporter"`
	Endpoint string `mapstructure:"endpoint"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// MCPConfig represents MCP protocol configuration
type MCPConfig struct {
	Registry MCPRegistryConfig `mapstructure:"registry"`
	Server   MCPServerConfig   `mapstructure:"server"`
}

// MCPRegistryConfig represents MCP registry configuration
type MCPRegistryConfig struct {
	StoragePath   string `mapstructure:"storage_path"`
	AutoSave      bool   `mapstructure:"auto_save"`
	SaveInterval  int    `mapstructure:"save_interval"` // in seconds
	MaxProjects   int    `mapstructure:"max_projects"`
	MaxTemplates  int    `mapstructure:"max_templates"`
	EnableMetrics bool   `mapstructure:"enable_metrics"`
	CacheEnabled  bool   `mapstructure:"cache_enabled"`
	CacheTTL      int    `mapstructure:"cache_ttl"` // in seconds
}

// MCPServerConfig represents MCP server configuration
type MCPServerConfig struct {
	Name       string            `mapstructure:"name"`
	Version    string            `mapstructure:"version"`
	Protocol   string            `mapstructure:"protocol"`
	Transport  string            `mapstructure:"transport"` // "stdio" or "sse"
	Port       int               `mapstructure:"port"`
	Host       string            `mapstructure:"host"`
	Headers    map[string]string `mapstructure:"headers"`
	MaxWorkers int               `mapstructure:"max_workers"`
	Timeout    int               `mapstructure:"timeout"` // in seconds
	EnableAuth bool              `mapstructure:"enable_auth"`
	AuthToken  string            `mapstructure:"auth_token"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	URL      string `mapstructure:"url"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
	MaxConns int    `mapstructure:"max_conns"`
	MinConns int    `mapstructure:"min_conns"`
}

// AIConfig represents AI provider configuration
type AIConfig struct {
	Provider    string  `mapstructure:"provider"`     // "openai", "gemini", "glm"
	Model       string  `mapstructure:"model"`          // default model
	APIKey      string  `mapstructure:"api_key"`       // from env
	Endpoint    string  `mapstructure:"endpoint"`      // API endpoint
	Temperature float64 `mapstructure:"temperature"`  // default temperature
	MaxTokens   int     `mapstructure:"max_tokens"`    // default max tokens
	Timeout     string  `mapstructure:"timeout"`       // request timeout
}

// PathsConfig represents path configuration for templates and output
type PathsConfig struct {
	Templates string `mapstructure:"templates"` // path to templates directory
	Output    string `mapstructure:"output"`     // path to output directory
	Data      string `mapstructure:"data"`       // path to data directory
	Cache     string `mapstructure:"cache"`     // path to cache directory
}

// FeatureConfig represents feature flags configuration
type FeatureConfig struct {
	ExternalGPU   bool `mapstructure:"external_gpu"`
	AuditLogging  bool `mapstructure:"audit_logging"`
	BetaGenerators bool `mapstructure:"beta_generators"`
}

// Loader loads and validates configuration
type Loader struct {
	viper *viper.Viper
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("$HOME/.mcp-fulfillment-ops")

	// Environment variables - prefix HULK_ as per blueprint
	v.SetEnvPrefix("HULK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Loader{viper: v}
}

// Load loads configuration from files and environment
func (l *Loader) Load() (*Config, error) {
	// Set defaults
	l.setDefaults()

	// Read main config file
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		logger.Info("No config file found, using defaults and environment variables")
	}

	// Load features.yaml (merge)
	if err := l.loadFeatures(); err != nil {
		logger.Warn("Failed to load features.yaml", zap.Error(err))
	}

	// Load environment-specific config (merge)
	if err := l.loadEnvironmentConfig(); err != nil {
		logger.Warn("Failed to load environment config", zap.Error(err))
	}

	var cfg Config
	if err := l.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	logger.Info("Configuration loaded",
		zap.String("config_file", l.viper.ConfigFileUsed()),
	)

	return &cfg, nil
}

// loadFeatures loads features.yaml and merges with existing config
func (l *Loader) loadFeatures() error {
	featuresViper := viper.New()
	featuresViper.SetConfigType("yaml")
	featuresViper.SetConfigName("features")
	featuresViper.AddConfigPath(".")
	featuresViper.AddConfigPath("./config")

	if err := featuresViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // features.yaml is optional
		}
		return fmt.Errorf("error reading features.yaml: %w", err)
	}

	// Merge features into main viper
	features := featuresViper.AllSettings()
	for key, value := range features {
		l.viper.Set(fmt.Sprintf("features.%s", key), value)
	}

	return nil
}

// loadEnvironmentConfig loads environment-specific YAML file
func (l *Loader) loadEnvironmentConfig() error {
	env := os.Getenv("HULK_ENV")
	if env == "" {
		env = os.Getenv("MCP_HULK_ENV") // fallback for backward compatibility
	}
	if env == "" {
		env = "dev" // default
	}

	env = strings.ToLower(env)
	envMap := map[string]string{
		"development": "dev",
		"production":   "prod",
		"staging":      "staging",
		"test":         "test",
		"dev":          "dev",
		"prod":         "prod",
		"stage":        "staging",
	}

	if mappedEnv, ok := envMap[env]; ok {
		env = mappedEnv
	}

	envViper := viper.New()
	envViper.SetConfigType("yaml")
	envViper.SetConfigName(env)
	envViper.AddConfigPath("./config/environments")
	envViper.AddConfigPath("config/environments")

	if err := envViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil // environment config is optional
		}
		return fmt.Errorf("error reading environment config: %w", err)
	}

	// Merge environment config into main viper
	envSettings := envViper.AllSettings()
	for key, value := range envSettings {
		l.viper.Set(key, value)
	}

	logger.Info("Environment config loaded", zap.String("environment", env))
	return nil
}

// setDefaults sets default configuration values
func (l *Loader) setDefaults() {
	// Server defaults
	l.viper.SetDefault("server.port", 8080)
	l.viper.SetDefault("server.host", "0.0.0.0")
	l.viper.SetDefault("server.read_timeout", "30s")
	l.viper.SetDefault("server.write_timeout", "30s")

	// Engine defaults
	l.viper.SetDefault("engine.workers", "auto")
	l.viper.SetDefault("engine.queue_size", 2000)
	l.viper.SetDefault("engine.timeout", "20s")

	// Cache defaults
	l.viper.SetDefault("cache.l1_size", 5000)
	l.viper.SetDefault("cache.l2_ttl", "1h")
	l.viper.SetDefault("cache.l3_path", "data/cache")

	// NATS defaults
	l.viper.SetDefault("nats.urls", []string{"nats://localhost:4222"})
	l.viper.SetDefault("nats.user", "")
	l.viper.SetDefault("nats.pass", "")

	// Logging defaults
	l.viper.SetDefault("logging.level", "info")
	l.viper.SetDefault("logging.format", "json")

	// Telemetry defaults
	l.viper.SetDefault("telemetry.tracing.enabled", true)
	l.viper.SetDefault("telemetry.tracing.exporter", "jaeger")
	l.viper.SetDefault("telemetry.tracing.endpoint", "http://localhost:4318/v1/traces")
	l.viper.SetDefault("telemetry.metrics.enabled", true)

	// MCP Registry defaults
	l.viper.SetDefault("mcp.registry.storage_path", "./registry")
	l.viper.SetDefault("mcp.registry.auto_save", true)
	l.viper.SetDefault("mcp.registry.save_interval", 300) // 5 minutes in seconds
	l.viper.SetDefault("mcp.registry.max_projects", 1000)
	l.viper.SetDefault("mcp.registry.max_templates", 100)
	l.viper.SetDefault("mcp.registry.enable_metrics", true)
	l.viper.SetDefault("mcp.registry.cache_enabled", true)
	l.viper.SetDefault("mcp.registry.cache_ttl", 3600) // 1 hour in seconds

	// MCP Server defaults
	l.viper.SetDefault("mcp.server.name", "mcp-fulfillment-ops")
	l.viper.SetDefault("mcp.server.version", "1.0.0")
	l.viper.SetDefault("mcp.server.protocol", "2024-11-05")
	l.viper.SetDefault("mcp.server.transport", "stdio")
	l.viper.SetDefault("mcp.server.port", 3000)
	l.viper.SetDefault("mcp.server.host", "localhost")
	l.viper.SetDefault("mcp.server.max_workers", 10)
	l.viper.SetDefault("mcp.server.timeout", 30) // 30 seconds
	l.viper.SetDefault("mcp.server.enable_auth", false)
	l.viper.SetDefault("mcp.server.auth_token", "")

	// Database defaults
	l.viper.SetDefault("database.url", "")
	l.viper.SetDefault("database.host", "localhost")
	l.viper.SetDefault("database.port", 5432)
	l.viper.SetDefault("database.user", "postgres")
	l.viper.SetDefault("database.password", "")
	l.viper.SetDefault("database.database", "mcp_hulk")
	l.viper.SetDefault("database.ssl_mode", "disable")
	l.viper.SetDefault("database.max_conns", 25)
	l.viper.SetDefault("database.min_conns", 5)

	// AI defaults
	l.viper.SetDefault("ai.provider", "glm")
	l.viper.SetDefault("ai.model", "glm-4.6-z.ai")
	l.viper.SetDefault("ai.api_key", "")
	l.viper.SetDefault("ai.endpoint", "https://api.z.ai/v1")
	l.viper.SetDefault("ai.temperature", 0.3)
	l.viper.SetDefault("ai.max_tokens", 4000)
	l.viper.SetDefault("ai.timeout", "60s")

	// Paths defaults
	l.viper.SetDefault("paths.templates", "./templates")
	l.viper.SetDefault("paths.output", "./output")
	l.viper.SetDefault("paths.data", "./data")
	l.viper.SetDefault("paths.cache", "./data/cache")

	// Features defaults
	l.viper.SetDefault("features.external_gpu", false)
	l.viper.SetDefault("features.audit_logging", false)
	l.viper.SetDefault("features.beta_generators", false)
}

// GetEngineWorkers returns the number of workers (handles "auto")
func GetEngineWorkers(cfg *EngineConfig) int {
	if cfg.Workers == nil {
		return 0 // auto
	}

	if str, ok := cfg.Workers.(string); ok && str == "auto" {
		return 0 // auto
	}

	if num, ok := cfg.Workers.(int); ok {
		return num
	}

	return 0 // auto
}
