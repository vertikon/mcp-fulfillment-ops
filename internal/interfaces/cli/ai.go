package cli

import (
	"github.com/spf13/cobra"
	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

var aiService *services.AIAppService

// aiCmd represents the ai command
var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI assistance commands",
	Long:  `Interact with AI models for assistance`,
}

// aiChatCmd performs chat
var aiChatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Chat with AI",
	RunE: func(cmd *cobra.Command, args []string) error {
		message, _ := cmd.Flags().GetString("message")
		logger.Info("AI chat", zap.String("message", message))
		// TODO: Call service
		cmd.Println("AI response: Service implementation pending")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(aiChatCmd)
	aiChatCmd.Flags().StringP("message", "m", "", "Message to send")
	aiChatCmd.MarkFlagRequired("message")
}

// SetAIService sets the AI service
func SetAIService(service *services.AIAppService) {
	aiService = service
}
