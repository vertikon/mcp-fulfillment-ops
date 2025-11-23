// Package main provides the MCP Protocol server entry point
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/core/config"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/generators"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/protocol"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/registry"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/validators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfgLoader := config.NewLoader()
	cfg, err := cfgLoader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	envMgr := config.NewEnvironmentManager()
	if err := logger.Init(cfg.Logging.Level, envMgr.IsDevelopment()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting MCP Protocol server")

	// Initialize MCP components
	server, err := initializeMCPServer(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize MCP server", zap.Error(err))
	}

	// Start MCP server
	if err := server.Start(); err != nil {
		logger.Fatal("Failed to start MCP server", zap.Error(err))
	}

	logger.Info("MCP Protocol server started successfully")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down MCP Protocol server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Graceful shutdown of MCP server
	if err := server.Stop(); err != nil {
		logger.Error("Error during MCP server shutdown", zap.Error(err))
	}

	<-shutdownCtx.Done()
	logger.Info("MCP Protocol server stopped")
}

// initializeMCPServer initializes and configures the MCP server
func initializeMCPServer(cfg *config.Config) (*protocol.MCPServer, error) {
	logger.Info("Initializing MCP server components")

	// Initialize generator factory
	genFactory := generators.NewGeneratorFactory(nil) // Use default config

	// Initialize validator factory
	valFactory := validators.NewValidatorFactory()

	// Initialize registry
	regConfig := &registry.RegistryConfig{
		StoragePath:   cfg.MCP.Registry.StoragePath,
		AutoSave:      cfg.MCP.Registry.AutoSave,
		SaveInterval:  time.Duration(cfg.MCP.Registry.SaveInterval) * time.Second,
		MaxProjects:   cfg.MCP.Registry.MaxProjects,
		MaxTemplates:  cfg.MCP.Registry.MaxTemplates,
		EnableMetrics: cfg.MCP.Registry.EnableMetrics,
		CacheEnabled:  cfg.MCP.Registry.CacheEnabled,
		CacheTTL:      time.Duration(cfg.MCP.Registry.CacheTTL) * time.Second,
	}
	mcpRegistry := registry.NewMCPRegistry(regConfig)

	// Create MCP server configuration
	serverConfig := &protocol.ServerConfig{
		Name:       cfg.MCP.Server.Name,
		Version:    cfg.MCP.Server.Version,
		Protocol:   cfg.MCP.Server.Protocol,
		Transport:  cfg.MCP.Server.Transport,
		Port:       cfg.MCP.Server.Port,
		Host:       cfg.MCP.Server.Host,
		Headers:    cfg.MCP.Server.Headers,
		MaxWorkers: cfg.MCP.Server.MaxWorkers,
		Timeout:    time.Duration(cfg.MCP.Server.Timeout) * time.Second,
		EnableAuth: cfg.MCP.Server.EnableAuth,
		AuthToken:  cfg.MCP.Server.AuthToken,
	}

	// Create MCP server
	server := protocol.NewMCPServer(serverConfig)

	// Initialize handler manager
	handlerManager := protocol.NewHandlerManager(genFactory, valFactory, mcpRegistry)

	// Register all handlers
	handlers := handlerManager.GetAllHandlers()
	for name, handler := range handlers {
		if err := server.RegisterHandler(handler); err != nil {
			logger.Error("Failed to register handler",
				zap.String("handler", name),
				zap.Error(err))
		} else {
			logger.Info("Registered handler", zap.String("handler", name))
		}
	}

	// Register tools from tools package
	tools := protocol.NewMCPTools()
	tools.RegisterTools(server)

	logger.Info("MCP server components initialized successfully",
		zap.String("transport", serverConfig.Transport),
		zap.String("host", serverConfig.Host),
		zap.Int("port", serverConfig.Port),
		zap.Int("max_workers", serverConfig.MaxWorkers))

	return server, nil
}
