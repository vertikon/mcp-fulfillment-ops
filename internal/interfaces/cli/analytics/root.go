package analytics

import (
	"sync"

	"github.com/spf13/cobra"
)

// AnalyticsCmd represents the analytics command group
var AnalyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Analytics and metrics commands",
	Long:  `Commands for viewing system analytics, metrics, and performance data`,
}

var registerOnce sync.Once

// RegisterCommands registers all analytics subcommands.
func RegisterCommands() {
	registerOnce.Do(func() {
		registerMetricsCmd()
		registerPerformanceCmd()
	})
}
