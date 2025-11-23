// Package handlers provides handlers for text files
// Generic handler for .md, .sh, etc.
package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/config"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// TextFileHandler handles generic text files
type TextFileHandler struct {
	*BaseHandler
	supportedExts map[string]bool
}

// NewTextFileHandler creates a new text file handler
func NewTextFileHandler(cfg *config.Config) *TextFileHandler {
	exts := map[string]bool{
		".md":           true,
		".sh":           true,
		".txt":          true,
		".env":          true,
		".gitignore":    true,
		".dockerignore": true,
	}

	return &TextFileHandler{
		BaseHandler:   NewBaseHandler(NewConfigAdapter(cfg)),
		supportedExts: exts,
	}
}

// Process processes a text file
func (h *TextFileHandler) Process(path string, info os.FileInfo) error {
	if !h.CanHandle(path, info) {
		return nil
	}

	if !h.ShouldProcess(path) {
		return nil
	}

	logger.Debug("Processing text file", zap.String("path", path))

	// Read file
	content, err := h.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read text file: %w", err)
	}

	// Process content
	processed := h.processText(string(content), path)

	// Apply config rules
	ext := filepath.Ext(path)
	if ext == "" {
		ext = filepath.Base(path)
	}
	processed = h.config.ApplyRules(processed, ext)

	// Write back if changed
	if processed != string(content) {
		if err := h.WriteFile(path, []byte(processed)); err != nil {
			return fmt.Errorf("failed to write text file: %w", err)
		}
		logger.Info("Updated text file", zap.String("path", path))
	}

	return nil
}

// CanHandle checks if this handler can handle the file
func (h *TextFileHandler) CanHandle(path string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	ext := filepath.Ext(path)
	if ext == "" {
		// Check for files without extension
		filename := filepath.Base(path)
		return h.supportedExts[filename] || strings.HasPrefix(filename, ".")
	}

	return h.supportedExts[ext]
}

// processText processes text content
func (h *TextFileHandler) processText(content string, path string) string {
	result := content

	// Apply mappings
	for key, value := range h.getAllMappings() {
		if strings.Contains(result, key) {
			result = strings.ReplaceAll(result, key, value)
		}
	}

	// Process based on file type
	ext := filepath.Ext(path)
	switch ext {
	case ".md":
		result = h.processMarkdown(result)
	case ".sh":
		result = h.processShellScript(result)
	case ".env":
		result = h.processEnvFile(result)
	default:
		// Generic text processing
		result = h.processGeneric(result)
	}

	return result
}

// processMarkdown processes markdown files
func (h *TextFileHandler) processMarkdown(content string) string {
	// Apply mappings to markdown content
	result := content

	// Process code blocks, links, etc.
	// This is a simplified version
	return result
}

// processShellScript processes shell script files
func (h *TextFileHandler) processShellScript(content string) string {
	// Apply mappings to shell script
	result := content

	// Process variables, paths, etc.
	return result
}

// processEnvFile processes .env files
func (h *TextFileHandler) processEnvFile(content string) string {
	// Apply mappings to environment variables
	result := content

	lines := strings.Split(result, "\n")
	processed := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Apply mappings to value
				for mapKey, mapValue := range h.getAllMappings() {
					if strings.Contains(value, mapKey) {
						value = strings.ReplaceAll(value, mapKey, mapValue)
					}
				}

				processed = append(processed, key+"="+value)
			} else {
				processed = append(processed, line)
			}
		} else {
			processed = append(processed, line)
		}
	}

	return strings.Join(processed, "\n")
}

// processGeneric processes generic text
func (h *TextFileHandler) processGeneric(content string) string {
	// Simple text replacement
	return content
}

// getAllMappings gets all mappings from config
func (h *TextFileHandler) getAllMappings() map[string]string {
	if adapter, ok := h.config.(*ConfigAdapter); ok {
		return adapter.config.Mappings
	}
	return make(map[string]string)
}
