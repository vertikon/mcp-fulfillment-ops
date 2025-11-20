package protocol

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// mockToolHandler is a mock implementation of ToolHandler for testing
type mockToolHandler struct {
	name        string
	description string
	schema      map[string]interface{}
	handleFunc  func(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error)
}

func (m *mockToolHandler) Name() string {
	return m.name
}

func (m *mockToolHandler) Description() string {
	return m.description
}

func (m *mockToolHandler) Schema() map[string]interface{} {
	return m.schema
}

func (m *mockToolHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, request)
	}
	return NewSuccessResponse(request.ID, map[string]interface{}{"status": "ok"}), nil
}

func TestNewMCPServer(t *testing.T) {
	tests := []struct {
		name   string
		config *ServerConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &ServerConfig{
				Name:       "TestServer",
				Version:    "1.0.0",
				Protocol:   "json-rpc-2.0",
				Transport:  "stdio",
				MaxWorkers: 5,
				Timeout:    10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewMCPServer(tt.config)
			if server == nil {
				t.Fatal("NewMCPServer returned nil")
			}
			if server.handlers == nil {
				t.Error("handlers map should not be nil")
			}
			if server.router == nil {
				t.Error("router should not be nil")
			}
		})
	}
}

func TestMCPServer_RegisterHandler(t *testing.T) {
	server := NewMCPServer(nil)

	handler := &mockToolHandler{
		name:        "test_tool",
		description: "Test tool",
		schema:      map[string]interface{}{"type": "object"},
	}

	// Register handler
	if err := server.RegisterHandler(handler); err != nil {
		t.Fatalf("RegisterHandler() error = %v", err)
	}

	// Try to register same handler again (should fail)
	if err := server.RegisterHandler(handler); err == nil {
		t.Error("RegisterHandler() should fail when registering duplicate handler")
	}
}

func TestMCPServer_Start_Stop(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		wantErr   bool
	}{
		{
			name:      "stdio transport",
			transport: "stdio",
			wantErr:   false,
		},
		{
			name:      "sse transport",
			transport: "sse",
			wantErr:   false,
		},
		{
			name:      "invalid transport",
			transport: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				Transport: tt.transport,
				Host:      "localhost",
				Port:      0, // Use random port
			}
			server := NewMCPServer(config)

			err := server.Start()
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !server.IsRunning() {
					t.Error("Server should be running")
				}

				// Stop server
				if err := server.Stop(); err != nil {
					t.Errorf("Stop() error = %v", err)
				}

				if server.IsRunning() {
					t.Error("Server should not be running after Stop()")
				}
			}
		})
	}
}

func TestMCPServer_Start_AlreadyRunning(t *testing.T) {
	server := NewMCPServer(&ServerConfig{
		Transport: "stdio",
	})

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Try to start again
	err := server.Start()
	if err == nil {
		t.Error("Start() should fail when server is already running")
	}

	_ = server.Stop()
}

func TestMCPServer_Stop_NotRunning(t *testing.T) {
	server := NewMCPServer(nil)

	// Stop when not running should not error
	if err := server.Stop(); err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}
}

func TestMCPServer_ProcessRequest(t *testing.T) {
	server := NewMCPServer(nil)
	server.router = NewToolRouter(make(map[string]ToolHandler))

	tests := []struct {
		name    string
		request *JSONRPCRequest
		wantErr bool
	}{
		{
			name: "initialize request",
			request: &JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "initialize",
				ID:      "1",
			},
			wantErr: false,
		},
		{
			name: "invalid JSON-RPC version",
			request: &JSONRPCRequest{
				JSONRPC: "1.0",
				Method:  "test",
				ID:      "1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := server.processRequest(tt.request)
			if response == nil {
				t.Fatal("processRequest() returned nil")
			}

			if tt.wantErr {
				if response.Error == nil {
					t.Error("Expected error response")
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %v", response.Error)
				}
			}
		})
	}
}

func TestMCPServer_HandleInitialize(t *testing.T) {
	server := NewMCPServer(&ServerConfig{
		Name:     "TestServer",
		Version:  "1.0.0",
		Protocol: "json-rpc-2.0",
	})

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      "1",
	}

	response := server.handleInitialize(context.Background(), request)
	if response == nil {
		t.Fatal("handleInitialize() returned nil")
	}

	if response.Error != nil {
		t.Errorf("handleInitialize() error = %v", response.Error)
	}

	if response.Result == nil {
		t.Error("handleInitialize() should return result")
	}
}

func TestMCPServer_GetCapabilities(t *testing.T) {
	server := NewMCPServer(nil)

	handler := &mockToolHandler{
		name:        "test_tool",
		description: "Test tool",
		schema:      map[string]interface{}{"type": "object"},
	}
	_ = server.RegisterHandler(handler)

	capabilities := server.GetCapabilities()
	if capabilities == nil {
		t.Fatal("GetCapabilities() returned nil")
	}

	if protocolVersion, ok := capabilities["protocolVersion"].(string); !ok || protocolVersion == "" {
		t.Error("GetCapabilities() should include protocolVersion")
	}
}

func TestMCPServer_IsRunning(t *testing.T) {
	server := NewMCPServer(&ServerConfig{
		Transport: "stdio",
	})

	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}

	_ = server.Start()
	if !server.IsRunning() {
		t.Error("Server should be running after Start()")
	}

	_ = server.Stop()
	if server.IsRunning() {
		t.Error("Server should not be running after Stop()")
	}
}

func TestServerConfig_Defaults(t *testing.T) {
	server := NewMCPServer(nil)

	if server.config.Name != "mcp-fulfillment-ops" {
		t.Errorf("Expected default name 'mcp-fulfillment-ops', got '%s'", server.config.Name)
	}
	if server.config.Version != "1.0.0" {
		t.Errorf("Expected default version '1.0.0', got '%s'", server.config.Version)
	}
	if server.config.Protocol != "json-rpc-2.0" {
		t.Errorf("Expected default protocol 'json-rpc-2.0', got '%s'", server.config.Protocol)
	}
	if server.config.Transport != "stdio" {
		t.Errorf("Expected default transport 'stdio', got '%s'", server.config.Transport)
	}
}

func TestMCPServer_ConcurrentAccess(t *testing.T) {
	server := NewMCPServer(nil)

	// Concurrent handler registration
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			handler := &mockToolHandler{
				name:        "tool_" + string(rune(id)),
				description: "Test tool",
				schema:      map[string]interface{}{"type": "object"},
			}
			_ = server.RegisterHandler(handler)
			done <- true
		}(i)
	}

	// Wait for all registrations
	for i := 0; i < 10; i++ {
		<-done
	}

	capabilities := server.GetCapabilities()
	if capabilities == nil {
		t.Fatal("GetCapabilities() returned nil")
	}
}

