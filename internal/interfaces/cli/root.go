package cli

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/internal/interfaces/cli/analytics"
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
	Use:   "fulfillment-ops",
	Short: "mcp-fulfillment-ops CLI - Orchestrate logistics operations",
	Long: `mcp-fulfillment-ops is a powerful CLI tool for orchestrating
logistics operations in the Vertikon ecosystem.`,
	Version: fmt.Sprintf("%s (built %s)", Version, BuildDate),
}

var configureOnce sync.Once

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	configureOnce.Do(configureRoot)
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed", zap.Error(err))
		os.Exit(1)
	}
}

func configureRoot() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.fulfillment-ops/config.yaml)")

	// Add subcommand groups
	rootCmd.AddCommand(analytics.AnalyticsCmd)
	analytics.RegisterCommands()

	registerAICmd()
	registerGenerateCmd()
	registerMonitorCmd()
	registerStateCmd()
	registerTemplateCmd()
	registerVersionCmd()
}

// GetRootCmd returns the root command (for testing)
func GetRootCmd() *cobra.Command {
	return rootCmd
}
