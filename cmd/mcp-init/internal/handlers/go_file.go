// Package handlers provides handlers for Go files
// Handler for .go files (focus on imports)
package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/config"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// GoFileHandler handles Go source files
type GoFileHandler struct {
	*BaseHandler
	importRegex *regexp.Regexp
}

// NewGoFileHandler creates a new Go file handler
func NewGoFileHandler(cfg *config.Config) *GoFileHandler {
	// Regex to match import statements
	importRegex := regexp.MustCompile(`(?m)^\s*(import\s+\([^)]*\)|import\s+"[^"]+"|import\s+` + "`[^`]+`" + `)`)

	return &GoFileHandler{
		BaseHandler: NewBaseHandler(NewConfigAdapter(cfg)),
		importRegex: importRegex,
	}
}

// Process processes a Go file
func (h *GoFileHandler) Process(path string, info os.FileInfo) error {
	if !h.CanHandle(path, info) {
		return nil
	}

	if !h.ShouldProcess(path) {
		return nil
	}

	logger.Debug("Processing Go file", zap.String("path", path))

	// Read file
	content, err := h.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Process imports
	processed, err := h.processImports(string(content))
	if err != nil {
		return fmt.Errorf("failed to process imports: %w", err)
	}

	// Apply config rules
	processed = h.config.ApplyRules(processed, "go")

	// Write back if changed
	if processed != string(content) {
		if err := h.WriteFile(path, []byte(processed)); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
		logger.Info("Updated Go file", zap.String("path", path))
	}

	return nil
}

// CanHandle checks if this handler can handle the file
func (h *GoFileHandler) CanHandle(path string, info os.FileInfo) bool {
	return !info.IsDir() && filepath.Ext(path) == ".go"
}

// processImports processes import statements in Go files
func (h *GoFileHandler) processImports(content string) (string, error) {
	// Find all import statements
	matches := h.importRegex.FindAllString(content, -1)

	result := content

	for _, match := range matches {
		// Apply mappings to import paths
		processed := h.applyImportMappings(match)
		if processed != match {
			result = strings.Replace(result, match, processed, 1)
		}
	}

	return result, nil
}

// applyImportMappings applies configuration mappings to import paths
func (h *GoFileHandler) applyImportMappings(importStmt string) string {
	result := importStmt

	// Extract import paths and apply mappings
	// This is a simplified version - production would need more robust parsing
	for key, value := range h.getAllMappings() {
		if strings.Contains(result, key) {
			result = strings.ReplaceAll(result, key, value)
		}
	}

	return result
}

// getAllMappings gets all mappings from config
func (h *GoFileHandler) getAllMappings() map[string]string {
	// Access mappings through config adapter
	if adapter, ok := h.config.(*ConfigAdapter); ok {
		return adapter.config.Mappings
	}
	return make(map[string]string)
}
