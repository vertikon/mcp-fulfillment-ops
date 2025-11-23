// Package config provides configuration for mcp-init tool
// Defines mappings and transformation rules for customizing MCP templates
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration for mcp-init
type Config struct {
	// Mappings define how to replace values in templates
	Mappings map[string]string `yaml:"mappings"`

	// Rules define transformation rules
	Rules []Rule `yaml:"rules"`

	// Exclusions define paths to exclude from processing
	Exclusions []string `yaml:"exclusions"`
}

// Rule represents a transformation rule
type Rule struct {
	// Pattern is the pattern to match
	Pattern string `yaml:"pattern"`

	// Replacement is the replacement value
	Replacement string `yaml:"replacement"`

	// FileTypes are the file types this rule applies to
	FileTypes []string `yaml:"file_types"`
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		// Return default config
		return &Config{
			Mappings:   make(map[string]string),
			Rules:      []Rule{},
			Exclusions: []string{".git", "node_modules", ".crush"},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Mappings == nil {
		config.Mappings = make(map[string]string)
	}
	if config.Exclusions == nil {
		config.Exclusions = []string{".git", "node_modules", ".crush"}
	}

	return &config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ShouldExclude checks if a path should be excluded
func (c *Config) ShouldExclude(path string) bool {
	for _, exclusion := range c.Exclusions {
		if filepath.Base(path) == exclusion {
			return true
		}
		if matched, _ := filepath.Match(exclusion, path); matched {
			return true
		}
	}
	return false
}

// GetMapping returns the mapped value for a key
func (c *Config) GetMapping(key string) (string, bool) {
	value, exists := c.Mappings[key]
	return value, exists
}

// ApplyRules applies transformation rules to content
func (c *Config) ApplyRules(content string, fileType string) string {
	result := content
	for _, rule := range c.Rules {
		// Check if rule applies to this file type
		applies := false
		if len(rule.FileTypes) == 0 {
			applies = true
		} else {
			for _, ft := range rule.FileTypes {
				if ft == fileType {
					applies = true
					break
				}
			}
		}

		if applies {
			// Simple string replacement (can be enhanced with regex)
			result = replaceAll(result, rule.Pattern, rule.Replacement)
		}
	}
	return result
}

// replaceAll replaces all occurrences of old with new
func replaceAll(s, old, new string) string {
	// Use strings.ReplaceAll for simple string replacement
	return strings.ReplaceAll(s, old, new)
}
