package analytics

import (
	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
)

var monitoringService *services.MonitoringAppService

// metricsCmd represents the metrics command
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display system metrics",
	Long:  `Display current system metrics and performance data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Displaying metrics")
		// TODO: Call monitoring service
		cmd.Println("Metrics:")
		cmd.Println("  CPU: 0%")
		cmd.Println("  Memory: 0MB")
		cmd.Println("  Requests: 0")
		return nil
	},
}

func registerMetricsCmd() {
	AnalyticsCmd.AddCommand(metricsCmd)
}

// SetMonitoringService sets the monitoring service
func SetMonitoringService(service *services.MonitoringAppService) {
	monitoringService = service
}
