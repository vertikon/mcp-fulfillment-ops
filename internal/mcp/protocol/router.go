package protocol

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ToolRouter handles routing of MCP tool requests to appropriate handlers
type ToolRouter struct {
	handlers map[string]ToolHandler
	logger   *zap.Logger
}

// NewToolRouter creates a new tool router
func NewToolRouter(handlers map[string]ToolHandler) *ToolRouter {
	return &ToolRouter{
		handlers: handlers,
		logger:   logger.Get(),
	}
}

// Route routes a JSON-RPC request to the appropriate handler
func (r *ToolRouter) Route(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	// Validate JSON-RPC request
	if request.JSONRPC != "2.0" {
		return NewErrorResponse(request.ID, ErrCodeInvalidRequest, "Invalid JSON-RPC version", nil)
	}

	// Handle special MCP methods
	switch request.Method {
	case "tools/list":
		return r.handleListTools(ctx, request)
	case "tools/call":
		return r.handleCallTool(ctx, request)
	case "initialize":
		return r.handleInitialize(ctx, request)
	case "ping":
		return r.handlePing(ctx, request)
	default:
		// Try to find a handler for the method
		if handler, exists := r.handlers[request.Method]; exists {
			return r.handleToolCall(ctx, request, handler)
		}

		return NewErrorResponse(request.ID, ErrCodeMethodNotFound,
			fmt.Sprintf("Method '%s' not found", request.Method), nil)
	}
}

// handleListTools handles the tools/list method
func (r *ToolRouter) handleListTools(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	tools := make([]Tool, 0, len(r.handlers))

	for _, handler := range r.handlers {
		tools = append(tools, Tool{
			Name:        handler.Name(),
			Description: handler.Description(),
			InputSchema: handler.Schema(),
		})
	}

	result := ListToolsResponse{
		Tools: tools,
	}

	return NewSuccessResponse(request.ID, result)
}

// handleCallTool handles the tools/call method
func (r *ToolRouter) handleCallTool(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	// Parse call parameters
	var params CallToolRequest
	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil)
	}

	// Find handler for the tool
	handler, exists := r.handlers[params.Name]
	if !exists {
		return NewErrorResponse(request.ID, ErrCodeMethodNotFound,
			fmt.Sprintf("Tool '%s' not found", params.Name), nil)
	}

	// Create a new request for the tool
	toolRequest := &JSONRPCRequest{
		JSONRPC: request.JSONRPC,
		Method:  params.Name,
		Params:  params.Arguments,
		ID:      request.ID,
	}

	return r.handleToolCall(ctx, toolRequest, handler)
}

// handleInitialize handles the initialize method
func (r *ToolRouter) handleInitialize(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	var params InitializeParams
	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid initialize parameters: %v", err), nil)
	}

	// Extract client information
	clientInfo := "unknown"
	if params.ClientInfo != nil {
		if name, ok := params.ClientInfo["name"].(string); ok {
			clientInfo = name
		}
	}

	r.logger.Info("MCP client initialized",
		zap.String("client", clientInfo),
		zap.String("protocol", params.ProtocolVersion),
		zap.Any("capabilities", params.Capabilities))

	result := InitializeResult{
		ProtocolVersion: "2.0",
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		ServerInfo: map[string]interface{}{
			"name":    "mcp-fulfillment-ops",
			"version": "1.0.0",
		},
	}

	return NewSuccessResponse(request.ID, result)
}

// handlePing handles ping requests
func (r *ToolRouter) handlePing(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	result := map[string]interface{}{
		"message": "pong",
		"status":  "healthy",
		"server":  "mcp-fulfillment-ops",
	}

	return NewSuccessResponse(request.ID, result)
}

// handleToolCall handles a specific tool call
func (r *ToolRouter) handleToolCall(ctx context.Context, request *JSONRPCRequest, handler ToolHandler) *JSONRPCResponse {
	// Validate parameters against schema
	if err := r.validateParams(request.Params, handler.Schema()); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Parameter validation failed: %v", err), nil)
	}

	// Call the handler
	response, err := handler.Handle(ctx, request)
	if err != nil {
		r.logger.Error("Tool handler failed",
			zap.String("tool", handler.Name()),
			zap.Error(err))

		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Tool execution failed: %v", err), nil)
	}

	return response
}

// validateParams validates request parameters against a JSON schema
func (r *ToolRouter) validateParams(params interface{}, schema map[string]interface{}) error {
	// This is a simplified validation implementation
	// In a production environment, you would use a proper JSON schema validator

	if params == nil {
		if required, ok := schema["required"].([]interface{}); ok && len(required) > 0 {
			return fmt.Errorf("missing required parameters")
		}
		return nil
	}

	// Convert params to map for validation
	paramMap, ok := params.(map[string]interface{})
	if !ok {
		return fmt.Errorf("parameters must be an object")
	}

	// Check required fields
	if required, ok := schema["required"].([]interface{}); ok {
		properties, _ := schema["properties"].(map[string]interface{})

		for _, req := range required {
			fieldName, ok := req.(string)
			if !ok {
				continue
			}

			if _, exists := paramMap[fieldName]; !exists {
				if properties != nil {
					if fieldInfo, exists := properties[fieldName].(map[string]interface{}); exists {
						if description, ok := fieldInfo["description"].(string); ok {
							return fmt.Errorf("missing required parameter: %s (%s)", fieldName, description)
						}
					}
				}
				return fmt.Errorf("missing required parameter: %s", fieldName)
			}
		}
	}

	return nil
}

// parseParams is defined in handlers.go

// GetRegisteredTools returns a list of all registered tools
func (r *ToolRouter) GetRegisteredTools() []string {
	tools := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		tools = append(tools, name)
	}
	return tools
}

// HasTool checks if a specific tool is registered
func (r *ToolRouter) HasTool(toolName string) bool {
	_, exists := r.handlers[toolName]
	return exists
}

// GetToolHandler returns a specific tool handler
func (r *ToolRouter) GetToolHandler(toolName string) (ToolHandler, bool) {
	handler, exists := r.handlers[toolName]
	return handler, exists
}

// UpdateHandlers updates the router with new handlers
func (r *ToolRouter) UpdateHandlers(handlers map[string]ToolHandler) {
	r.handlers = handlers
	r.logger.Info("Updated tool router handlers",
		zap.Int("count", len(handlers)))
}

// RouterStats returns statistics about the router
type RouterStats struct {
	TotalTools       int      `json:"total_tools"`
	RegisteredTools  []string `json:"registered_tools"`
	SupportedMethods []string `json:"supported_methods"`
}

// GetStats returns router statistics
func (r *ToolRouter) GetStats() RouterStats {
	tools := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		tools = append(tools, name)
	}

	return RouterStats{
		TotalTools:       len(r.handlers),
		RegisteredTools:  tools,
		SupportedMethods: []string{"tools/list", "tools/call", "initialize", "ping"},
	}
}

// LogRouterStatus logs the current status of the router
func (r *ToolRouter) LogRouterStatus() {
	stats := r.GetStats()
	r.logger.Info("Tool Router Status",
		zap.Int("total_tools", stats.TotalTools),
		zap.Strings("registered_tools", stats.RegisteredTools),
		zap.Strings("supported_methods", stats.SupportedMethods))
}
