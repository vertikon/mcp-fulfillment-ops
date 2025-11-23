package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var mcpService *services.MCPAppService

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new MCP project",
	Long:  `Generate a new MCP project from a template`,
	RunE: func(cmd *cobra.Command, args []string) error {
		templateID, _ := cmd.Flags().GetString("template")
		outputPath, _ := cmd.Flags().GetString("output")

		if templateID == "" {
			return fmt.Errorf("template ID is required")
		}

		logger.Info("Generating MCP project",
			zap.String("template", templateID),
			zap.String("output", outputPath),
		)

		// TODO: Call service
		// req := &dtos.GenerateMCPRequest{
		// 	TemplateID: templateID,
		// 	OutputPath: outputPath,
		// }
		// result, err := mcpService.GenerateMCP(cmd.Context(), req)

		cmd.Println("MCP project generation initiated")
		return nil
	},
}

func registerGenerateCmd() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringP("template", "t", "", "Template ID to use")
	generateCmd.Flags().StringP("output", "o", ".", "Output directory")
	generateCmd.MarkFlagRequired("template")
}

// SetMCPService sets the MCP service (for dependency injection)
func SetMCPService(service *services.MCPAppService) {
	mcpService = service
}
