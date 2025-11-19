// Package main provides the MCP CLI entry point
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "mcp-cli",
	Short: "MCP CLI",
	Long:  "Command-line interface for MCP operations",
}

func init() {
	// Initialize logger
	if err := logger.Init("info", true); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Add subcommands here
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command failed", zap.Error(err))
		os.Exit(1)
	}
}

