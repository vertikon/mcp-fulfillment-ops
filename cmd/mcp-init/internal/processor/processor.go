// Package processor provides the core file processing logic for mcp-init
// Orchestrates the walk through the directory tree and delegates to handlers
package processor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/config"
	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/handlers"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Processor orchestrates file processing
type Processor struct {
	config   *config.Config
	handlers map[string]handlers.Handler
}

// NewProcessor creates a new processor
func NewProcessor(cfg *config.Config) *Processor {
	p := &Processor{
		config:   cfg,
		handlers: make(map[string]handlers.Handler),
	}

	// Register handlers
	p.registerHandlers()

	return p
}

// registerHandlers registers all available handlers
func (p *Processor) registerHandlers() {
	// Register handlers for different file types
	p.handlers[".go"] = handlers.NewGoFileHandler(p.config)
	p.handlers[".mod"] = handlers.NewGoModHandler(p.config)
	p.handlers[".yaml"] = handlers.NewYAMLFileHandler(p.config)
	p.handlers[".yml"] = handlers.NewYAMLFileHandler(p.config)
	p.handlers[".md"] = handlers.NewTextFileHandler(p.config)
	p.handlers[".sh"] = handlers.NewTextFileHandler(p.config)
	p.handlers[".txt"] = handlers.NewTextFileHandler(p.config)
	
	// Directory handler
	p.handlers["__dir__"] = handlers.NewDirectoryHandler(p.config)
}

// Process processes a directory tree
func (p *Processor) Process(rootPath string) error {
	logger.Info("Starting directory processing", zap.String("path", rootPath))

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Check exclusions
		if p.config.ShouldExclude(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Process directories
		if info.IsDir() {
			return p.processDirectory(path, info)
		}

		// Process files
		return p.processFile(path, info)
	})
}

// processDirectory processes a directory
func (p *Processor) processDirectory(path string, info os.FileInfo) error {
	handler, exists := p.handlers["__dir__"]
	if !exists {
		return nil
	}

	return handler.Process(path, info)
}

// processFile processes a file
func (p *Processor) processFile(path string, info os.FileInfo) error {
	ext := filepath.Ext(path)
	
	handler, exists := p.handlers[ext]
	if !exists {
		// Try text handler as fallback
		handler, exists = p.handlers[".txt"]
		if !exists {
			logger.Debug("No handler found for file", zap.String("path", path), zap.String("ext", ext))
			return nil
		}
	}

	return handler.Process(path, info)
}

// GetHandler returns a handler for a specific file type
func (p *Processor) GetHandler(fileType string) (handlers.Handler, bool) {
	handler, exists := p.handlers[fileType]
	return handler, exists
}

