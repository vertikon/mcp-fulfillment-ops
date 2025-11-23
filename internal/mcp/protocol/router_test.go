package protocol

import (
	"context"
	"testing"
)

func TestNewToolRouter(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	router := NewToolRouter(handlers)

	if router == nil {
		t.Fatal("NewToolRouter returned nil")
	}
	if router.handlers == nil {
		t.Error("handlers map should not be nil")
	}
}

func TestToolRouter_Route_InvalidJSONRPCVersion(t *testing.T) {
	router := NewToolRouter(make(map[string]ToolHandler))

	request := &JSONRPCRequest{
		JSONRPC: "1.0", // Invalid version
		Method:  "test",
		ID:      "1",
	}

	response := router.Route(context.Background(), request)
	if response == nil {
		t.Fatal("Route() returned nil")
	}

	if response.Error == nil {
		t.Error("Expected error for invalid JSON-RPC version")
	}

	if response.Error.Code != ErrCodeInvalidRequest {
		t.Errorf("Expected error code %d, got %d", ErrCodeInvalidRequest, response.Error.Code)
	}
}

func TestToolRouter_Route_ToolsList(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	handler := &mockToolHandler{
		name:        "test_tool",
		description: "Test tool",
		schema:      map[string]interface{}{"type": "object"},
	}
	handlers["test_tool"] = handler

	router := NewToolRouter(handlers)

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      "1",
	}

	response := router.Route(context.Background(), request)
	if response == nil {
		t.Fatal("Route() returned nil")
	}

	if response.Error != nil {
		t.Errorf("Route() error = %v", response.Error)
	}

	if response.Result == nil {
		t.Error("Route() should return result for tools/list")
	}
}

func TestToolRouter_Route_ToolsCall(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	handler := &mockToolHandler{
		name:        "test_tool",
		description: "Test tool",
		schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}
	handlers["test_tool"] = handler

	router := NewToolRouter(handlers)

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "test_tool",
			"arguments": map[string]interface{}{
				"param": "value",
			},
		},
		ID: "1",
	}

	response := router.Route(context.Background(), request)
	if response == nil {
		t.Fatal("Route() returned nil")
	}

	if response.Error != nil {
		t.Errorf("Route() error = %v", response.Error)
	}
}

func TestToolRouter_Route_Initialize(t *testing.T) {
	router := NewToolRouter(make(map[string]ToolHandler))

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2.0",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]interface{}{"name": "test"},
		},
		ID: "1",
	}

	response := router.Route(context.Background(), request)
	if response == nil {
		t.Fatal("Route() returned nil")
	}

	if response.Error != nil {
		t.Errorf("Route() error = %v", response.Error)
	}
}

func TestToolRouter_Route_Ping(t *testing.T) {
	router := NewToolRouter(make(map[string]ToolHandler))

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "ping",
		ID:      "1",
	}

	response := router.Route(context.Background(), request)
	if response == nil {
		t.Fatal("Route() returned nil")
	}

	if response.Error != nil {
		t.Errorf("Route() error = %v", response.Error)
	}

	if response.Result == nil {
		t.Error("Route() should return result for ping")
	}
}

func TestToolRouter_Route_MethodNotFound(t *testing.T) {
	router := NewToolRouter(make(map[string]ToolHandler))

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "nonexistent_method",
		ID:      "1",
	}

	response := router.Route(context.Background(), request)
	if response == nil {
		t.Fatal("Route() returned nil")
	}

	if response.Error == nil {
		t.Error("Expected error for nonexistent method")
	}

	if response.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Expected error code %d, got %d", ErrCodeMethodNotFound, response.Error.Code)
	}
}

func TestToolRouter_GetRegisteredTools(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	handlers["tool1"] = &mockToolHandler{name: "tool1"}
	handlers["tool2"] = &mockToolHandler{name: "tool2"}

	router := NewToolRouter(handlers)

	tools := router.GetRegisteredTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
}

func TestToolRouter_HasTool(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	handlers["test_tool"] = &mockToolHandler{name: "test_tool"}

	router := NewToolRouter(handlers)

	if !router.HasTool("test_tool") {
		t.Error("HasTool() should return true for registered tool")
	}

	if router.HasTool("nonexistent") {
		t.Error("HasTool() should return false for unregistered tool")
	}
}

func TestToolRouter_GetToolHandler(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	expectedHandler := &mockToolHandler{name: "test_tool"}
	handlers["test_tool"] = expectedHandler

	router := NewToolRouter(handlers)

	handler, exists := router.GetToolHandler("test_tool")
	if !exists {
		t.Error("GetToolHandler() should return true for registered tool")
	}
	if handler != expectedHandler {
		t.Error("GetToolHandler() should return the correct handler")
	}

	_, exists = router.GetToolHandler("nonexistent")
	if exists {
		t.Error("GetToolHandler() should return false for unregistered tool")
	}
}

func TestToolRouter_GetStats(t *testing.T) {
	handlers := make(map[string]ToolHandler)
	handlers["tool1"] = &mockToolHandler{name: "tool1"}
	handlers["tool2"] = &mockToolHandler{name: "tool2"}

	router := NewToolRouter(handlers)

	stats := router.GetStats()
	if stats.TotalTools != 2 {
		t.Errorf("Expected 2 total tools, got %d", stats.TotalTools)
	}
	if len(stats.RegisteredTools) != 2 {
		t.Errorf("Expected 2 registered tools, got %d", len(stats.RegisteredTools))
	}
}

func TestToolRouter_ValidateParams(t *testing.T) {
	router := NewToolRouter(make(map[string]ToolHandler))

	tests := []struct {
		name    string
		params  interface{}
		schema  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "nil params with no required fields",
			params:  nil,
			schema:  map[string]interface{}{"type": "object"},
			wantErr: false,
		},
		{
			name:   "missing required field",
			params: map[string]interface{}{},
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"name"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "all required fields present",
			params: map[string]interface{}{"name": "test"},
			schema: map[string]interface{}{
				"type":     "object",
				"required": []interface{}{"name"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid params type",
			params:  "not an object",
			schema:  map[string]interface{}{"type": "object"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.validateParams(tt.params, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
