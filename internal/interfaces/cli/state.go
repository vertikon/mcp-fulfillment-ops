package cli

import (
	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage application state",
	Long:  `View and manage application state, snapshots, and projections`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("State command")
		cmd.Println("State management - service implementation pending")
		return nil
	},
}

func registerStateCmd() {
	rootCmd.AddCommand(stateCmd)
}
