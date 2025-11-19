// Package handlers provides handlers for directories
// Handler for renaming directories/files
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

// DirectoryHandler handles directory operations
type DirectoryHandler struct {
	*BaseHandler
}

// NewDirectoryHandler creates a new directory handler
func NewDirectoryHandler(cfg *config.Config) *DirectoryHandler {
	return &DirectoryHandler{
		BaseHandler: NewBaseHandler(NewConfigAdapter(cfg)),
	}
}

// Process processes a directory
func (h *DirectoryHandler) Process(path string, info os.FileInfo) error {
	if !h.CanHandle(path, info) {
		return nil
	}

	if !h.ShouldProcess(path) {
		return nil
	}

	logger.Debug("Processing directory", zap.String("path", path))

	// Check if directory name needs to be renamed
	newName := h.getNewName(path)
	if newName != "" && newName != filepath.Base(path) {
		return h.renameDirectory(path, newName)
	}

	return nil
}

// CanHandle checks if this handler can handle the path
func (h *DirectoryHandler) CanHandle(path string, info os.FileInfo) bool {
	return info.IsDir()
}

// getNewName gets the new name for a directory based on mappings
func (h *DirectoryHandler) getNewName(path string) string {
	baseName := filepath.Base(path)
	
	// Check if there's a mapping for this directory name
	if mapped, exists := h.config.GetMapping(baseName); exists {
		return mapped
	}
	
	// Apply pattern-based mappings
	for key, value := range h.getAllMappings() {
		if strings.Contains(baseName, key) {
			return strings.ReplaceAll(baseName, key, value)
		}
	}
	
	return ""
}

// renameDirectory renames a directory
func (h *DirectoryHandler) renameDirectory(oldPath string, newName string) error {
	parentDir := filepath.Dir(oldPath)
	newPath := filepath.Join(parentDir, newName)

	logger.Info("Renaming directory",
		zap.String("old", oldPath),
		zap.String("new", newPath),
	)

	// Check if target already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target directory already exists: %s", newPath)
	}

	// Rename directory
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename directory: %w", err)
	}

	return nil
}

// RenameFile renames a file (utility method)
func (h *DirectoryHandler) RenameFile(oldPath string, newName string) error {
	parentDir := filepath.Dir(oldPath)
	newPath := filepath.Join(parentDir, newName)

	logger.Info("Renaming file",
		zap.String("old", oldPath),
		zap.String("new", newPath),
	)

	// Check if target already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target file already exists: %s", newPath)
	}

	// Rename file
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// getAllMappings gets all mappings from config
func (h *DirectoryHandler) getAllMappings() map[string]string {
	if adapter, ok := h.config.(*ConfigAdapter); ok {
		return adapter.config.Mappings
	}
	return make(map[string]string)
}

