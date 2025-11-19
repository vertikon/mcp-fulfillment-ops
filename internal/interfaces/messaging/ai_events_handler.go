package messaging

import (
	"context"
	"encoding/json"

	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// AIEventsHandler handles AI-related events
type AIEventsHandler struct {
	aiService *services.AIAppService
	logger    *zap.Logger
}

// NewAIEventsHandler creates a new AI events handler
func NewAIEventsHandler(aiService *services.AIAppService) *AIEventsHandler {
	return &AIEventsHandler{
		aiService: aiService,
		logger:    logger.WithContext(nil),
	}
}

// HandleAIJobCompleted handles AI job completed events
func (h *AIEventsHandler) HandleAIJobCompleted(ctx context.Context, eventData []byte) error {
	var event struct {
		JobID string                 `json:"job_id"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal AI job completed event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling AI job completed event", zap.String("job_id", event.JobID))
	// Process event - delegate to service
	// Note: Event handlers are informational - service already processed the job
	return nil
}

// HandleAIFeedback handles AI feedback events
func (h *AIEventsHandler) HandleAIFeedback(ctx context.Context, eventData []byte) error {
	var event struct {
		FeedbackID string                 `json:"feedback_id"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal AI feedback event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling AI feedback event", zap.String("feedback_id", event.FeedbackID))
	// Process event - delegate to service
	// Note: Event handlers are informational - service already processed the feedback
	return nil
}
