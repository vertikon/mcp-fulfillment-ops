package dtos

import (
	"fmt"
)

// CreateMCPRequest represents a request to create an MCP
type CreateMCPRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// Validate validates the CreateMCPRequest
func (r *CreateMCPRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// UpdateMCPRequest represents a request to update an MCP
type UpdateMCPRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// Validate validates the UpdateMCPRequest
func (r *UpdateMCPRequest) Validate() error {
	return nil
}

// GenerateMCPRequest represents a request to generate an MCP
type GenerateMCPRequest struct {
	TemplateID string                 `json:"template_id" validate:"required"`
	Parameters map[string]interface{} `json:"parameters"`
	OutputPath string                 `json:"output_path"`
}

// Validate validates the GenerateMCPRequest
func (r *GenerateMCPRequest) Validate() error {
	if r.TemplateID == "" {
		return fmt.Errorf("template_id is required")
	}
	return nil
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// GenerateMCPResponse represents a response from MCP generation
type GenerateMCPResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// ValidateMCPResponse represents a response from MCP validation
type ValidateMCPResponse struct {
	MCPID  string   `json:"mcp_id"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}
