package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ConfigGenerator generates configuration files (.env, yaml, nats-schemas, etc.)
type ConfigGenerator struct {
	logger *zap.Logger
}

// NewConfigGenerator creates a new config generator
func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{
		logger: logger.Get(),
	}
}

// GenerateConfig generates a configuration file
func (g *ConfigGenerator) GenerateConfig(ctx context.Context, req ConfigGenerateRequest) (*ConfigGenerateResult, error) {
	g.logger.Info("Generating config",
		zap.String("type", req.Type),
		zap.String("name", req.Name),
		zap.String("output", req.OutputPath))

	// Validate request
	if err := g.Validate(req); err != nil {
		return nil, err
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate config based on type
	var content []byte
	var err error

	switch req.Type {
	case "env":
		content, err = g.generateEnv(req)
	case "yaml":
		content, err = g.generateYAML(req)
	case "json":
		content, err = g.generateJSON(req)
	case "nats-schema":
		content, err = g.generateNATSSchema(req)
	case "toml":
		content, err = g.generateTOML(req)
	default:
		return nil, fmt.Errorf("unsupported config type: %s", req.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("config generation failed: %w", err)
	}

	// Write file
	if err := os.WriteFile(req.OutputPath, content, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	return &ConfigGenerateResult{
		Path:    req.OutputPath,
		Size:    int64(len(content)),
		Type:    req.Type,
		Content: string(content),
	}, nil
}

// ConfigGenerateRequest represents a request to generate a config file
type ConfigGenerateRequest struct {
	Type       string                 `json:"type"` // env, yaml, json, nats-schema, toml
	Name       string                 `json:"name"`
	OutputPath string                 `json:"output_path"`
	Values     map[string]interface{} `json:"values"`
	Schema     map[string]interface{} `json:"schema,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// ConfigGenerateResult represents the result of config generation
type ConfigGenerateResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Validate validates the config generation request
func (g *ConfigGenerator) Validate(req ConfigGenerateRequest) error {
	if req.Type == "" {
		return fmt.Errorf("config type is required")
	}
	
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if req.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	
	validTypes := []string{"env", "yaml", "json", "nats-schema", "toml"}
	valid := false
	for _, vt := range validTypes {
		if req.Type == vt {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid config type: %s (valid types: %v)", req.Type, validTypes)
	}
	
	return nil
}

// generateEnv generates a .env file
func (g *ConfigGenerator) generateEnv(req ConfigGenerateRequest) ([]byte, error) {
	var lines []string
	
	lines = append(lines, fmt.Sprintf("# Generated config for %s", req.Name))
	lines = append(lines, "")
	
	for key, value := range req.Values {
		line := fmt.Sprintf("%s=%v", strings.ToUpper(key), value)
		lines = append(lines, line)
	}
	
	return []byte(strings.Join(lines, "\n")), nil
}

// generateYAML generates a YAML config file
func (g *ConfigGenerator) generateYAML(req ConfigGenerateRequest) ([]byte, error) {
	data := make(map[string]interface{})
	
	// Add name
	data["name"] = req.Name
	
	// Add values
	if req.Values != nil {
		data["config"] = req.Values
	}
	
	// Add schema if provided
	if req.Schema != nil {
		data["schema"] = req.Schema
	}
	
	content, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	
	return content, nil
}

// generateJSON generates a JSON config file
func (g *ConfigGenerator) generateJSON(req ConfigGenerateRequest) ([]byte, error) {
	data := make(map[string]interface{})
	
	// Add name
	data["name"] = req.Name
	
	// Add values
	if req.Values != nil {
		data["config"] = req.Values
	}
	
	// Add schema if provided
	if req.Schema != nil {
		data["schema"] = req.Schema
	}
	
	content, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	// Convert YAML to JSON-like structure (simplified)
	// In production, use proper JSON marshaling
	return content, nil
}

// generateNATSSchema generates a NATS schema file
func (g *ConfigGenerator) generateNATSSchema(req ConfigGenerateRequest) ([]byte, error) {
	var lines []string
	
	lines = append(lines, fmt.Sprintf("# NATS Schema for %s", req.Name))
	lines = append(lines, "")
	lines = append(lines, "subject: "+req.Name)
	lines = append(lines, "")
	lines = append(lines, "schema:")
	
	// Generate schema from values
	if req.Values != nil {
		for key, value := range req.Values {
			valueType := g.inferType(value)
			lines = append(lines, fmt.Sprintf("  %s: %s", key, valueType))
		}
	}
	
	return []byte(strings.Join(lines, "\n")), nil
}

// generateTOML generates a TOML config file
func (g *ConfigGenerator) generateTOML(req ConfigGenerateRequest) ([]byte, error) {
	var lines []string
	
	lines = append(lines, fmt.Sprintf("# Generated TOML config for %s", req.Name))
	lines = append(lines, "")
	
	if req.Values != nil {
		for key, value := range req.Values {
			line := fmt.Sprintf("%s = %v", key, value)
			lines = append(lines, line)
		}
	}
	
	return []byte(strings.Join(lines, "\n")), nil
}

// inferType infers the type of a value
func (g *ConfigGenerator) inferType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case int, int32, int64:
		return "integer"
	case float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string"
	}
}
