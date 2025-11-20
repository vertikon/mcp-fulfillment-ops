package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var (
	// Version is set at build time
	Version = "dev"
	// BuildDate is set at build time
	BuildDate = "unknown"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hulk",
	Short: "mcp-fulfillment-ops CLI - Generate and manage MCP projects",
	Long: `mcp-fulfillment-ops is a powerful CLI tool for generating and managing
Model Context Protocol (MCP) projects.`,
	Version: fmt.Sprintf("%s (built %s)", Version, BuildDate),
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed", zap.Error(err))
		os.Exit(1)
	}
}

// init initializes the CLI
func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.hulk/config.yaml)")
	
	// Add subcommand groups
	rootCmd.AddCommand(analytics.AnalyticsCmd)
	rootCmd.AddCommand(ci.CICmd)
}

// GetRootCmd returns the root command (for testing)
func GetRootCmd() *cobra.Command {
	return rootCmd
}
