// Package handlers provides handlers for YAML files
// Handler for .yaml/.yml files
package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/config"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// YAMLFileHandler handles YAML files
type YAMLFileHandler struct {
	*BaseHandler
}

// NewYAMLFileHandler creates a new YAML file handler
func NewYAMLFileHandler(cfg *config.Config) *YAMLFileHandler {
	return &YAMLFileHandler{
		BaseHandler: NewBaseHandler(NewConfigAdapter(cfg)),
	}
}

// Process processes a YAML file
func (h *YAMLFileHandler) Process(path string, info os.FileInfo) error {
	if !h.CanHandle(path, info) {
		return nil
	}

	if !h.ShouldProcess(path) {
		return nil
	}

	logger.Debug("Processing YAML file", zap.String("path", path))

	// Read file
	content, err := h.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	// Parse YAML
	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		logger.Warn("Failed to parse YAML, treating as text", zap.String("path", path), zap.Error(err))
		// Fall back to text processing
		return h.processAsText(string(content), path)
	}

	// Process YAML structure
	processed, err := h.processYAML(data)
	if err != nil {
		return fmt.Errorf("failed to process YAML: %w", err)
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(processed)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Apply config rules
	processedContent := h.config.ApplyRules(string(output), "yaml")

	// Write back if changed
	if processedContent != string(content) {
		if err := h.WriteFile(path, []byte(processedContent)); err != nil {
			return fmt.Errorf("failed to write YAML file: %w", err)
		}
		logger.Info("Updated YAML file", zap.String("path", path))
	}

	return nil
}

// CanHandle checks if this handler can handle the file
func (h *YAMLFileHandler) CanHandle(path string, info os.FileInfo) bool {
	ext := filepath.Ext(path)
	return !info.IsDir() && (ext == ".yaml" || ext == ".yml")
}

// processYAML processes YAML data structure
func (h *YAMLFileHandler) processYAML(data interface{}) (interface{}, error) {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		return h.processMap(v), nil
	case []interface{}:
		return h.processSlice(v), nil
	case string:
		return h.processString(v), nil
	default:
		return v, nil
	}
}

// processMap processes a map
func (h *YAMLFileHandler) processMap(m map[interface{}]interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	for k, v := range m {
		key := h.processKey(k)
		value := h.processValue(v)
		result[key] = value
	}
	return result
}

// processSlice processes a slice
func (h *YAMLFileHandler) processSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = h.processValue(v)
	}
	return result
}

// processKey processes a map key
func (h *YAMLFileHandler) processKey(k interface{}) interface{} {
	if str, ok := k.(string); ok {
		return h.processString(str)
	}
	return k
}

// processValue processes a value
func (h *YAMLFileHandler) processValue(v interface{}) interface{} {
	switch val := v.(type) {
	case string:
		return h.processString(val)
	case map[interface{}]interface{}:
		return h.processMap(val)
	case []interface{}:
		return h.processSlice(val)
	default:
		return val
	}
}

// processString processes a string value
func (h *YAMLFileHandler) processString(s string) string {
	// Apply mappings
	result := s
	for key, value := range h.getAllMappings() {
		if strings.Contains(result, key) {
			result = strings.ReplaceAll(result, key, value)
		}
	}
	return result
}

// processAsText processes file as plain text if YAML parsing fails
func (h *YAMLFileHandler) processAsText(content string, path string) error {
	processed := h.config.ApplyRules(content, "yaml")

	if processed != content {
		if err := h.WriteFile(path, []byte(processed)); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		logger.Info("Updated YAML file (as text)", zap.String("path", path))
	}
	return nil
}

// getAllMappings gets all mappings from config
func (h *YAMLFileHandler) getAllMappings() map[string]string {
	if adapter, ok := h.config.(*ConfigAdapter); ok {
		return adapter.config.Mappings
	}
	return make(map[string]string)
}
