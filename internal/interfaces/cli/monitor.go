package cli

import (
	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
)

var monitoringService *services.MonitoringAppService

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor system metrics and health",
	Long:  `Display system metrics, health status, and performance data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Monitoring system")
		// TODO: Call service
		cmd.Println("System status: Operational")
		return nil
	},
}

func registerMonitorCmd() {
	rootCmd.AddCommand(monitorCmd)
}

// SetMonitoringService sets the monitoring service
func SetMonitoringService(service *services.MonitoringAppService) {
	monitoringService = service
}
