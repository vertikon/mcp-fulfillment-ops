package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/protocol"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MCPServer represents the main MCP protocol server
type MCPServer struct {
	config      *ServerConfig
	handlers    map[string]ToolHandler
	router      *ToolRouter
	logger      *zap.Logger
	httpServer  *http.Server
	running     bool
	mu          sync.RWMutex
	shutdownCtx context.Context
	cancelFunc  context.CancelFunc
}

// ServerConfig holds configuration for the MCP server
type ServerConfig struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Protocol    string            `json:"protocol"`
	Transport   string            `json:"transport"` // "stdio" or "sse"
	Port        int               `json:"port,omitempty"`
	Host        string            `json:"host,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	MaxWorkers  int               `json:"max_workers"`
	Timeout     time.Duration     `json:"timeout"`
	EnableAuth  bool              `json:"enable_auth"`
	AuthToken   string            `json:"auth_token,omitempty"`
}

// ToolHandler represents a handler for MCP tools
type ToolHandler interface {
	Handle(ctx context.Context, request *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error)
	Name() string
	Description() string
	Schema() map[string]interface{}
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(config *ServerConfig) *MCPServer {
	if config == nil {
		config = &ServerConfig{
			Name:       "MCP-Hulk",
			Version:    "1.0.0",
			Protocol:   "json-rpc-2.0",
			Transport:  "stdio",
			MaxWorkers: 10,
			Timeout:    30 * time.Second,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &MCPServer{
		config:      config,
		handlers:    make(map[string]ToolHandler),
		logger:      logger.Get(),
		shutdownCtx: ctx,
		cancelFunc:  cancel,
	}

	server.router = NewToolRouter(server.handlers)

	return server
}

// RegisterHandler registers a new tool handler
func (s *MCPServer) RegisterHandler(handler ToolHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := handler.Name()
	if _, exists := s.handlers[name]; exists {
		return fmt.Errorf("handler %s already registered", name)
	}

	s.handlers[name] = handler
	s.router = NewToolRouter(s.handlers)

	s.logger.Info("Registered MCP tool handler", 
		zap.String("tool", name),
		zap.String("description", handler.Description()))

	return nil
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	s.logger.Info("Starting MCP server",
		zap.String("name", s.config.Name),
		zap.String("version", s.config.Version),
		zap.String("transport", s.config.Transport))

	switch s.config.Transport {
	case "stdio":
		return s.startStdioServer()
	case "sse":
		return s.startHTTPServer()
	default:
		return fmt.Errorf("unsupported transport: %s", s.config.Transport)
	}
}

// startStdioServer starts the server in stdio mode
func (s *MCPServer) startStdioServer() error {
	s.logger.Info("Starting MCP server in stdio mode")
	
	go func() {
		decoder := json.NewDecoder(os.Stdin)
		encoder := json.NewEncoder(os.Stdout)

		for {
			select {
			case <-s.shutdownCtx.Done():
				return
			default:
			}

			var request protocol.JSONRPCRequest
			if err := decoder.Decode(&request); err != nil {
				if err == io.EOF {
					s.logger.Info("EOF received, shutting down stdio server")
					return
				}
				s.logger.Error("Error decoding JSON-RPC request", zap.Error(err))
				continue
			}

			response := s.processRequest(&request)
			if err := encoder.Encode(response); err != nil {
				s.logger.Error("Error encoding JSON-RPC response", zap.Error(err))
			}
		}
	}()

	s.running = true
	return nil
}

// startHTTPServer starts the server in HTTP/SSE mode
func (s *MCPServer) startHTTPServer() error {
	mux := http.NewServeMux()
	
	// MCP endpoint
	mux.HandleFunc("/mcp", s.handleHTTPRequest)
	
	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealthCheck)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		Handler:      mux,
		ReadTimeout:  s.config.Timeout,
		WriteTimeout: s.config.Timeout,
	}

	s.logger.Info("Starting HTTP server", 
		zap.String("addr", s.httpServer.Addr))

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	s.running = true
	return nil
}

// processRequest processes a JSON-RPC request
func (s *MCPServer) processRequest(request *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	// Add authentication context if enabled
	if s.config.EnableAuth && s.config.AuthToken != "" {
		ctx = context.WithValue(ctx, "auth_token", s.config.AuthToken)
	}

	if request.Method == "initialize" {
		return s.handleInitialize(ctx, request)
	}

	return s.router.Route(ctx, request)
}

// handleInitialize handles the MCP initialize method
func (s *MCPServer) handleInitialize(ctx context.Context, request *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
	result := map[string]interface{}{
		"protocolVersion": s.config.Protocol,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.config.Name,
			"version": s.config.Version,
		},
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

// handleHTTPRequest handles HTTP requests
func (s *MCPServer) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	if s.config.EnableAuth {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+s.config.AuthToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var request protocol.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.logger.Error("Error decoding request", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := s.processRequest(&request)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Error encoding response", zap.Error(err))
	}
}

// handleHealthCheck handles health check requests
func (s *MCPServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"server":    s.config.Name,
		"version":   s.config.Version,
		"protocol":  s.config.Protocol,
		"transport": s.config.Transport,
		"timestamp": time.Now().UTC(),
		"handlers":  len(s.handlers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Stop stops the MCP server
func (s *MCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping MCP server")
	
	s.cancelFunc()

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("Error shutting down HTTP server", zap.Error(err))
		}
	}

	s.running = false
	s.logger.Info("MCP server stopped")
	return nil
}

// IsRunning returns whether the server is currently running
func (s *MCPServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetCapabilities returns the server capabilities
func (s *MCPServer) GetCapabilities() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(s.handlers))
	for _, handler := range s.handlers {
		tools = append(tools, map[string]interface{}{
			"name":        handler.Name(),
			"description": handler.Description(),
			"inputSchema": handler.Schema(),
		})
	}

	return map[string]interface{}{
		"protocolVersion": s.config.Protocol,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
				"tools":       tools,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.config.Name,
			"version": s.config.Version,
		},
	}
}