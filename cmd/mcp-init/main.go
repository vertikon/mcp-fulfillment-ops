// Package main provides the MCP initialization tool entry point
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/config"
	"github.com/vertikon/mcp-fulfillment-ops/cmd/mcp-init/internal/processor"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	configPath string
	rootPath   string
)

var rootCmd = &cobra.Command{
	Use:   "mcp-init",
	Short: "MCP Initialization Tool",
	Long:  "Tool for customizing and initializing MCP templates",
	RunE:  runInit,
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	rootCmd.Flags().StringVarP(&rootPath, "path", "p", ".", "Root path to process")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create processor
	proc := processor.NewProcessor(cfg)

	// Process directory tree
	if err := proc.Process(rootPath); err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	logger.Info("Processing completed successfully", zap.String("path", rootPath))
	return nil
}

func main() {
	// Initialize logger
	if err := logger.Init("info", true); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command failed", zap.Error(err))
		os.Exit(1)
	}
}
