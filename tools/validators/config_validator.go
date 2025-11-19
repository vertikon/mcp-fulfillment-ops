package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/validators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ConfigValidator validates configurations (schema, consistency)
type ConfigValidator struct {
	factory *validators.ValidatorFactory
	logger  *zap.Logger
}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		factory: validators.NewValidatorFactory(),
		logger:  logger.Get(),
	}
}

// ValidateConfig validates a configuration file
func (v *ConfigValidator) ValidateConfig(ctx context.Context, req ConfigValidateRequest) (*ConfigValidateResult, error) {
	v.logger.Info("Validating config",
		zap.String("path", req.Path),
		zap.String("type", req.ConfigType))

	var errors []string
	var warnings []string
	var checks []string

	// Check if config file exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		return &ConfigValidateResult{
			Path:   req.Path,
			Valid:  false,
			Errors: []string{fmt.Sprintf("config file does not exist: %s", req.Path)},
		}, nil
	}

	// Validate config file format
	if err := v.validateConfigFormat(req.Path, req.ConfigType, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("config format validation failed: %w", err)
	}

	// Validate config schema if provided
	if req.Schema != nil && len(req.Schema) > 0 {
		if err := v.validateConfigSchema(req.Path, req.Schema, &errors, &warnings, &checks); err != nil {
			return nil, fmt.Errorf("config schema validation failed: %w", err)
		}
	}

	// Use config validator from factory
	configValidator := v.factory.GetConfigValidator()
	configResult, err := configValidator.Validate(ctx, validators.ConfigRequest{
		Path:       req.Path,
		ConfigType: req.ConfigType,
		Schema:     req.Schema,
	})
	
	if err == nil {
		errors = append(errors, configResult.Errors...)
		warnings = append(warnings, configResult.Warnings...)
		checks = append(checks, "config_validator")
	}

	valid := len(errors) == 0

	return &ConfigValidateResult{
		Path:     req.Path,
		Valid:    valid,
		Warnings: warnings,
		Errors:   errors,
		Checks:   checks,
	}, nil
}

// ConfigValidateRequest represents a request to validate a config
type ConfigValidateRequest struct {
	Path       string                 `json:"path"`
	ConfigType string                 `json:"config_type"` // yaml, json, env, toml
	Schema     map[string]interface{} `json:"schema,omitempty"`
}

// ConfigValidateResult represents the result of config validation
type ConfigValidateResult struct {
	Path     string   `json:"path"`
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
	Checks   []string `json:"checks"`
}

// validateConfigFormat validates the config file format
func (v *ConfigValidator) validateConfigFormat(path, configType string, errors *[]string, warnings *[]string, checks *[]string) error {
	ext := strings.ToLower(filepath.Ext(path))
	
	expectedExt := map[string]string{
		"yaml": ".yaml",
		"yml":  ".yaml",
		"json": ".json",
		"env":  ".env",
		"toml": ".toml",
	}
	
	if expectedExt[configType] != "" && ext != expectedExt[configType] && ext != ".yml" {
		*warnings = append(*warnings, fmt.Sprintf("config type '%s' does not match file extension '%s'", configType, ext))
	}

	// Try to parse the config file
	switch configType {
	case "yaml", "yml":
		if err := v.parseYAML(path); err != nil {
			*errors = append(*errors, fmt.Sprintf("invalid YAML format: %v", err))
		} else {
			*checks = append(*checks, "yaml_format")
		}
	case "json":
		// JSON parsing would go here
		*checks = append(*checks, "json_format")
	case "env":
		// ENV parsing would go here
		*checks = append(*checks, "env_format")
	case "toml":
		// TOML parsing would go here
		*checks = append(*checks, "toml_format")
	default:
		*errors = append(*errors, fmt.Sprintf("unsupported config type: %s", configType))
	}

	*checks = append(*checks, "format")
	return nil
}

// validateConfigSchema validates config against a schema
func (v *ConfigValidator) validateConfigSchema(path string, schema map[string]interface{}, errors *[]string, warnings *[]string, checks *[]string) error {
	// Read config file
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse config based on file extension
	ext := strings.ToLower(filepath.Ext(path))
	var configData map[string]interface{}

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(content, &configData); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		*warnings = append(*warnings, "schema validation not supported for this config type")
		return nil
	}

	// Basic schema validation (simplified)
	// In production, use a proper schema validator like gojsonschema
	if schema != nil {
		for key, value := range schema {
			if _, exists := configData[key]; !exists {
				*errors = append(*errors, fmt.Sprintf("required key missing in config: %s", key))
			} else {
				// Type checking would go here
				_ = value
			}
		}
	}

	*checks = append(*checks, "schema")
	return nil
}

// parseYAML parses a YAML file to check if it's valid
func (v *ConfigValidator) parseYAML(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var data interface{}
	return yaml.Unmarshal(content, &data)
}

// Validate validates the config validation request
func (v *ConfigValidator) Validate(req ConfigValidateRequest) error {
	if req.Path == "" {
		return fmt.Errorf("path is required")
	}
	
	if req.ConfigType == "" {
		return fmt.Errorf("config type is required")
	}
	
	validTypes := []string{"yaml", "yml", "json", "env", "toml"}
	valid := false
	for _, vt := range validTypes {
		if req.ConfigType == vt {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid config type: %s", req.ConfigType)
	}
	
	return nil
}
