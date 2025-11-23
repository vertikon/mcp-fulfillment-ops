// Package handlers provides file handlers for different file types
// Interface that defines the contract for all handlers
package handlers

import (
	"os"

	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/config"
)

// Handler defines the interface for all file handlers
type Handler interface {
	// Process processes a file or directory
	Process(path string, info os.FileInfo) error

	// CanHandle checks if this handler can handle the given file
	CanHandle(path string, info os.FileInfo) bool
}

// BaseHandler provides common functionality for handlers
type BaseHandler struct {
	config Config
}

// Config interface for handlers to access configuration
// This avoids circular dependency with config package
type Config interface {
	ShouldExclude(path string) bool
	GetMapping(key string) (string, bool)
	ApplyRules(content string, fileType string) string
}

// ConfigAdapter adapts config.Config to handlers.Config interface
type ConfigAdapter struct {
	config *config.Config
}

// NewConfigAdapter creates a new config adapter
func NewConfigAdapter(cfg *config.Config) Config {
	return &ConfigAdapter{config: cfg}
}

// ShouldExclude checks if path should be excluded
func (a *ConfigAdapter) ShouldExclude(path string) bool {
	return a.config.ShouldExclude(path)
}

// GetMapping gets a mapping value
func (a *ConfigAdapter) GetMapping(key string) (string, bool) {
	return a.config.GetMapping(key)
}

// ApplyRules applies transformation rules
func (a *ConfigAdapter) ApplyRules(content string, fileType string) string {
	return a.config.ApplyRules(content, fileType)
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(cfg Config) *BaseHandler {
	return &BaseHandler{
		config: cfg,
	}
}

// ReadFile reads a file safely
func (b *BaseHandler) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes a file safely
func (b *BaseHandler) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// ShouldProcess checks if a file should be processed
func (b *BaseHandler) ShouldProcess(path string) bool {
	return !b.config.ShouldExclude(path)
}
