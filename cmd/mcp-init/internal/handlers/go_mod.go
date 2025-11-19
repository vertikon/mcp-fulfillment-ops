// Package handlers provides handlers for go.mod files
// Handler for go.mod (safe rewriting)
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

// GoModHandler handles go.mod files
type GoModHandler struct {
	*BaseHandler
}

// NewGoModHandler creates a new go.mod handler
func NewGoModHandler(cfg *config.Config) *GoModHandler {
	return &GoModHandler{
		BaseHandler: NewBaseHandler(NewConfigAdapter(cfg)),
	}
}

// Process processes a go.mod file
func (h *GoModHandler) Process(path string, info os.FileInfo) error {
	if !h.CanHandle(path, info) {
		return nil
	}

	if !h.ShouldProcess(path) {
		return nil
	}

	logger.Debug("Processing go.mod file", zap.String("path", path))

	// Read file
	content, err := h.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	// Process module name and dependencies
	processed, err := h.processGoMod(string(content))
	if err != nil {
		return fmt.Errorf("failed to process go.mod: %w", err)
	}

	// Apply config rules
	processed = h.config.ApplyRules(processed, "mod")

	// Write back if changed
	if processed != string(content) {
		if err := h.WriteFile(path, []byte(processed)); err != nil {
			return fmt.Errorf("failed to write go.mod %s: %w", path, err)
		}
		logger.Info("Updated go.mod", zap.String("path", path))
	}

	return nil
}

// CanHandle checks if this handler can handle the file
func (h *GoModHandler) CanHandle(path string, info os.FileInfo) bool {
	return !info.IsDir() && filepath.Base(path) == "go.mod"
}

// processGoMod processes go.mod content
func (h *GoModHandler) processGoMod(content string) (string, error) {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		processed := h.processLine(line)
		result = append(result, processed)
	}

	return strings.Join(result, "\n"), nil
}

// processLine processes a single line of go.mod
func (h *GoModHandler) processLine(line string) string {
	trimmed := strings.TrimSpace(line)

	// Process module declaration
	if strings.HasPrefix(trimmed, "module ") {
		return h.processModuleLine(line)
	}

	// Process require statements
	if strings.HasPrefix(trimmed, "require ") || strings.HasPrefix(trimmed, "require\t") {
		return h.processRequireLine(line)
	}

	// Process replace statements
	if strings.HasPrefix(trimmed, "replace ") {
		return h.processReplaceLine(line)
	}

	return line
}

// processModuleLine processes module declaration line
func (h *GoModHandler) processModuleLine(line string) string {
	// Apply mappings to module name
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		moduleName := parts[1]
		if mapped, exists := h.config.GetMapping(moduleName); exists {
			return strings.Replace(line, moduleName, mapped, 1)
		}
	}
	return line
}

// processRequireLine processes require statement line
func (h *GoModHandler) processRequireLine(line string) string {
	// Apply mappings to dependency paths
	result := line
	for key, value := range h.getAllMappings() {
		if strings.Contains(result, key) {
			result = strings.ReplaceAll(result, key, value)
		}
	}
	return result
}

// processReplaceLine processes replace statement line
func (h *GoModHandler) processReplaceLine(line string) string {
	// Apply mappings to replace paths
	result := line
	for key, value := range h.getAllMappings() {
		if strings.Contains(result, key) {
			result = strings.ReplaceAll(result, key, value)
		}
	}
	return result
}

// getAllMappings gets all mappings from config
func (h *GoModHandler) getAllMappings() map[string]string {
	if adapter, ok := h.config.(*ConfigAdapter); ok {
		return adapter.config.Mappings
	}
	return make(map[string]string)
}

