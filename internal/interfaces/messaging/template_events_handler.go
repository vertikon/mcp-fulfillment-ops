package messaging

import (
	"context"
	"encoding/json"

	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// TemplateEventsHandler handles template-related events
type TemplateEventsHandler struct {
	templateService *services.TemplateAppService
	logger          *zap.Logger
}

// NewTemplateEventsHandler creates a new template events handler
func NewTemplateEventsHandler(templateService *services.TemplateAppService) *TemplateEventsHandler {
	return &TemplateEventsHandler{
		templateService: templateService,
		logger:          logger.WithContext(nil),
	}
}

// HandleTemplateCreated handles template created events
func (h *TemplateEventsHandler) HandleTemplateCreated(ctx context.Context, eventData []byte) error {
	var event struct {
		TemplateID string                 `json:"template_id"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal template created event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling template created event", zap.String("template_id", event.TemplateID))
	// Process event - delegate to service
	// Note: Event handlers are informational - service already processed the creation
	return nil
}

// HandleTemplateUpdated handles template updated events
func (h *TemplateEventsHandler) HandleTemplateUpdated(ctx context.Context, eventData []byte) error {
	var event struct {
		TemplateID string                 `json:"template_id"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal template updated event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling template updated event", zap.String("template_id", event.TemplateID))
	// Process event - delegate to service
	// Note: Event handlers are informational - service already processed the update
	return nil
}

// HandleTemplateDeleted handles template deleted events
func (h *TemplateEventsHandler) HandleTemplateDeleted(ctx context.Context, eventData []byte) error {
	var event struct {
		TemplateID string `json:"template_id"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal template deleted event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling template deleted event", zap.String("template_id", event.TemplateID))
	// Process event - delegate to service
	// Note: Event handlers are informational - service already processed the deletion
	return nil
}
