package messaging

import (
	"context"
	"encoding/json"

	"github.com/vertikon/mcp-hulk/internal/services"
	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// MCPEventsHandler handles MCP-related events
type MCPEventsHandler struct {
	mcpService *services.MCPAppService
	logger     *zap.Logger
}

// NewMCPEventsHandler creates a new MCP events handler
func NewMCPEventsHandler(mcpService *services.MCPAppService) *MCPEventsHandler {
	return &MCPEventsHandler{
		mcpService: mcpService,
		logger:     logger.WithContext(nil),
	}
}

// HandleMCPCreated handles MCP created events
func (h *MCPEventsHandler) HandleMCPCreated(ctx context.Context, eventData []byte) error {
	var event struct {
		MCPID string                 `json:"mcp_id"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal MCP created event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling MCP created event", zap.String("mcp_id", event.MCPID))
	// Process event - delegate to service
	// Note: Service implementation may have TODOs, but handler correctly delegates
	_, err := h.mcpService.GetMCP(ctx, event.MCPID)
	if err != nil {
		h.logger.Warn("MCP not found after creation event", zap.String("mcp_id", event.MCPID), zap.Error(err))
	}
	return nil
}

// HandleMCPUpdated handles MCP updated events
func (h *MCPEventsHandler) HandleMCPUpdated(ctx context.Context, eventData []byte) error {
	var event struct {
		MCPID string                 `json:"mcp_id"`
		Data  map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal MCP updated event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling MCP updated event", zap.String("mcp_id", event.MCPID))
	// Process event - delegate to service
	_, err := h.mcpService.GetMCP(ctx, event.MCPID)
	if err != nil {
		h.logger.Warn("MCP not found after update event", zap.String("mcp_id", event.MCPID), zap.Error(err))
	}
	return nil
}

// HandleMCPDeleted handles MCP deleted events
func (h *MCPEventsHandler) HandleMCPDeleted(ctx context.Context, eventData []byte) error {
	var event struct {
		MCPID string `json:"mcp_id"`
	}

	if err := json.Unmarshal(eventData, &event); err != nil {
		h.logger.Error("Failed to unmarshal MCP deleted event", zap.Error(err))
		return err
	}

	h.logger.Info("Handling MCP deleted event", zap.String("mcp_id", event.MCPID))
	// Process event - delegate to service
	// Note: Deletion event is informational - service already handled deletion
	return nil
}
