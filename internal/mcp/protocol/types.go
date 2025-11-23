package protocol

import (
	"encoding/json"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)

// NewError creates a new JSON-RPC error
func NewError(code int, message string, data interface{}) *JSONRPCError {
	return &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates a new JSON-RPC error response
func NewErrorResponse(id interface{}, code int, message string, data interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   NewError(code, message, data),
	}
}

// NewSuccessResponse creates a new JSON-RPC success response
func NewSuccessResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCall represents a tool call request
type ToolCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
}

// ToolResult represents a tool call result
type ToolResult struct {
	Content []interface{} `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// TextContent represents text content in tool results
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ImageContent represents image content in tool results
type ImageContent struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// ResourceContent represents resource content
type ResourceContent struct {
	Type     string `json:"type"`
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
}

// ListToolsRequest represents a list tools request
type ListToolsRequest struct {
	Cursor string `json:"cursor,omitempty"`
}

// ListToolsResponse represents a list tools response
type ListToolsResponse struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// CallToolRequest represents a call tool request
type CallToolRequest struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
}

// InitializeParams represents initialize parameters
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo"`
}

// InitializeResult represents initialize result
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]interface{} `json:"serverInfo"`
}

// MarshalJSON implements custom JSON marshaling for JSONRPCRequest
func (r *JSONRPCRequest) MarshalJSON() ([]byte, error) {
	type Alias JSONRPCRequest
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for JSONRPCRequest
func (r *JSONRPCRequest) UnmarshalJSON(data []byte) error {
	type Alias JSONRPCRequest
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	return json.Unmarshal(data, &aux)
}

// MarshalJSON implements custom JSON marshaling for JSONRPCResponse
func (r *JSONRPCResponse) MarshalJSON() ([]byte, error) {
	type Alias JSONRPCResponse
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for JSONRPCResponse
func (r *JSONRPCResponse) UnmarshalJSON(data []byte) error {
	type Alias JSONRPCResponse
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	return json.Unmarshal(data, &aux)
}
