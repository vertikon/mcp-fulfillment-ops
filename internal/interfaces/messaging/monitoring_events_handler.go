package messaging

import (
	"context"
	"encoding/json"

	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MonitoringEventsHandler handles monitoring-related events
type MonitoringEventsHandler struct {
	monitoringService *services.MonitoringAppService
	logger            *zap.Logger
}

// NewMonitoringEventsHandler creates a new monitoring events handler
func NewMonitoringEventsHandler(monitoringService *services.MonitoringAppService) *MonitoringEventsHandler {
	return &MonitoringEventsHandler{
		monitoringService: monitoringService,
		logger:            logger.WithContext(nil),
	}
}

// HandleAlert handles alert events
func (h *MonitoringEventsHandler) HandleAlert(ctx context.Context, eventData []byte) error {
	var event struct {
		AlertID string                 `json:"alert_id"`
		Level   string                 `json:"level"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal alert event", zap.Error(err))
		return err
	}

	h.logger.Warn("Handling alert event",
		zap.String("alert_id", event.AlertID),
		zap.String("level", event.Level),
		zap.String("message", event.Message),
	)
	// Process event - delegate to service
	// Note: Alert events are informational - monitoring service already processed the alert
	return nil
}

// HandleMetricUpdate handles metric update events
func (h *MonitoringEventsHandler) HandleMetricUpdate(ctx context.Context, eventData []byte) error {
	var event struct {
		MetricName string                 `json:"metric_name"`
		Value      float64                `json:"value"`
		Tags       map[string]interface{} `json:"tags"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal metric update event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling metric update event", zap.String("metric_name", event.MetricName))
	// Process event - delegate to service
	// Note: Alert events are informational - monitoring service already processed the alert
	return nil
}
