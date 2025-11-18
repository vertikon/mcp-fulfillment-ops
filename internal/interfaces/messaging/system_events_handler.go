package messaging

import (
	"context"
	"encoding/json"

	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// SystemEventsHandler handles system-related events
type SystemEventsHandler struct {
	logger *zap.Logger
}

// NewSystemEventsHandler creates a new system events handler
func NewSystemEventsHandler() *SystemEventsHandler {
	return &SystemEventsHandler{
		logger: logger.WithContext(nil),
	}
}

// HandleDeployEvent handles deployment events
func (h *SystemEventsHandler) HandleDeployEvent(ctx context.Context, eventData []byte) error {
	var event struct {
		DeploymentID string                 `json:"deployment_id"`
		Status       string                 `json:"status"`
		Data         map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal deploy event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling deploy event",
		zap.String("deployment_id", event.DeploymentID),
		zap.String("status", event.Status),
	)
	// Process event - delegate to service
	// Note: System events are informational - service already processed the event
	return nil
}

// HandleConfigUpdate handles configuration update events
func (h *SystemEventsHandler) HandleConfigUpdate(ctx context.Context, eventData []byte) error {
	var event struct {
		ConfigKey string                 `json:"config_key"`
		Value     interface{}            `json:"value"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal config update event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling config update event", zap.String("config_key", event.ConfigKey))
	// Process event - delegate to service
	// Note: System events are informational - service already processed the event
	return nil
}

// HandleAuditEvent handles audit events
func (h *SystemEventsHandler) HandleAuditEvent(ctx context.Context, eventData []byte) error {
	var event struct {
		Action   string                 `json:"action"`
		UserID   string                 `json:"user_id"`
		Resource string                 `json:"resource"`
		Data     map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal audit event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling audit event",
		zap.String("action", event.Action),
		zap.String("user_id", event.UserID),
		zap.String("resource", event.Resource),
	)
	// Process event - delegate to service
	// Note: System events are informational - service already processed the event
	return nil
}
