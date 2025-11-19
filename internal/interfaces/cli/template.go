package cli

import (
	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var templateService *services.TemplateAppService

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  `List, create, update, and delete templates`,
}

// templateListCmd lists templates
var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Listing templates")
		// TODO: Call service
		cmd.Println("Templates:")
		return nil
	},
}

// templateCreateCmd creates a template
var templateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new template",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		logger.Info("Creating template", zap.String("name", name))
		// TODO: Call service
		cmd.Println("Template created")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCreateCmd.Flags().StringP("name", "n", "", "Template name")
	templateCreateCmd.MarkFlagRequired("name")
}

// SetTemplateService sets the template service
func SetTemplateService(service *services.TemplateAppService) {
	templateService = service
}
