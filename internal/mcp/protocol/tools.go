package protocol

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MCPTools represents the collection of all MCP tools
type MCPTools struct {
	logger *zap.Logger
}

// NewMCPTools creates a new instance of MCP tools
func NewMCPTools() *MCPTools {
	return &MCPTools{
		logger: logger.Get(),
	}
}

// GetToolDefinitions returns all available tool definitions
func (t *MCPTools) GetToolDefinitions() []Tool {
	return []Tool{
		t.generateProjectTool(),
		t.validateProjectTool(),
		t.listTemplatesTool(),
		t.describeStackTool(),
		t.listProjectsTool(),
		t.getProjectInfoTool(),
		t.deleteProjectTool(),
		t.updateProjectTool(),
	}
}

// generateProjectTool returns the generate_project tool definition
func (t *MCPTools) generateProjectTool() Tool {
	return Tool{
		Name:        "generate_project",
		Description: "Generate a new project with specified technology stack and configuration",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the project to generate",
				},
				"stack": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"go", "web", "tinygo", "wasm", "mcp-go-premium"},
					"description": "Technology stack to use for the project",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Output path where the project will be created",
				},
				"features": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "List of features to include (e.g., ['auth', 'monitoring', 'cache'])",
				},
				"template_version": map[string]interface{}{
					"type":        "string",
					"description": "Specific template version to use",
				},
				"config": map[string]interface{}{
					"type":        "object",
					"description": "Additional configuration parameters",
				},
			},
			"required": []string{"name", "stack", "path"},
		},
	}
}

// validateProjectTool returns the validate_project tool definition
func (t *MCPTools) validateProjectTool() Tool {
	return Tool{
		Name:        "validate_project",
		Description: "Validate a project structure and compliance with standards",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the project to validate",
				},
				"strict_mode": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable strict validation mode",
					"default":     false,
				},
				"check_dependencies": map[string]interface{}{
					"type":        "boolean",
					"description": "Check project dependencies",
					"default":     true,
				},
				"check_security": map[string]interface{}{
					"type":        "boolean",
					"description": "Run security checks",
					"default":     true,
				},
			},
			"required": []string{"path"},
		},
	}
}

// listTemplatesTool returns the list_templates tool definition
func (t *MCPTools) listTemplatesTool() Tool {
	return Tool{
		Name:        "list_templates",
		Description: "List all available templates for project generation",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"stack": map[string]interface{}{
					"type":        "string",
					"description": "Filter by technology stack",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by category (e.g., 'web', 'mobile', 'backend')",
				},
				"include_version": map[string]interface{}{
					"type":        "boolean",
					"description": "Include version information",
					"default":     true,
				},
				"include_features": map[string]interface{}{
					"type":        "boolean",
					"description": "Include feature information",
					"default":     false,
				},
			},
		},
	}
}

// describeStackTool returns the describe_stack tool definition
func (t *MCPTools) describeStackTool() Tool {
	return Tool{
		Name:        "describe_stack",
		Description: "Get detailed information about a specific technology stack",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"stack": map[string]interface{}{
					"type":        "string",
					"description": "Technology stack to describe",
				},
				"include_examples": map[string]interface{}{
					"type":        "boolean",
					"description": "Include usage examples",
					"default":     false,
				},
				"include_dependencies": map[string]interface{}{
					"type":        "boolean",
					"description": "Include dependency information",
					"default":     true,
				},
			},
			"required": []string{"stack"},
		},
	}
}

// listProjectsTool returns the list_projects tool definition
func (t *MCPTools) listProjectsTool() Tool {
	return Tool{
		Name:        "list_projects",
		Description: "List all generated projects",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"stack": map[string]interface{}{
					"type":        "string",
					"description": "Filter by technology stack",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"all", "active", "inactive", "error"},
					"description": "Filter by project status",
					"default":     "all",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of projects to return",
					"default":     50,
				},
				"sort_by": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"name", "created_at", "updated_at", "stack"},
					"description": "Sort projects by field",
					"default":     "created_at",
				},
			},
		},
	}
}

// getProjectInfoTool returns the get_project_info tool definition
func (t *MCPTools) getProjectInfoTool() Tool {
	return Tool{
		Name:        "get_project_info",
		Description: "Get detailed information about a specific project",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the project",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the project (alternative to name)",
				},
				"include_stats": map[string]interface{}{
					"type":        "boolean",
					"description": "Include project statistics",
					"default":     true,
				},
				"include_dependencies": map[string]interface{}{
					"type":        "boolean",
					"description": "Include dependency analysis",
					"default":     false,
				},
			},
		},
	}
}

// deleteProjectTool returns the delete_project tool definition
func (t *MCPTools) deleteProjectTool() Tool {
	return Tool{
		Name:        "delete_project",
		Description: "Delete a project and all its files",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the project to delete",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the project (alternative to name)",
				},
				"confirm": map[string]interface{}{
					"type":        "boolean",
					"description": "Confirm deletion (safety check)",
					"default":     false,
				},
				"backup": map[string]interface{}{
					"type":        "boolean",
					"description": "Create backup before deletion",
					"default":     true,
				},
			},
			"required": []string{"confirm"},
		},
	}
}

// updateProjectTool returns the update_project tool definition
func (t *MCPTools) updateProjectTool() Tool {
	return Tool{
		Name:        "update_project",
		Description: "Update an existing project with new configuration or features",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the project to update",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the project (alternative to name)",
				},
				"add_features": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Features to add to the project",
				},
				"remove_features": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "Features to remove from the project",
				},
				"update_config": map[string]interface{}{
					"type":        "object",
					"description": "Configuration updates",
				},
				"upgrade_template": map[string]interface{}{
					"type":        "string",
					"description": "Target template version for upgrade",
				},
			},
		},
	}
}

// RegisterTools registers all tool handlers with the MCP server
func (t *MCPTools) RegisterTools(server *MCPServer) {
	tools := t.GetToolDefinitions()
	
	for _, tool := range tools {
		handler := &ToolHandlerImpl{
			name:        tool.Name,
			description: tool.Description,
			schema:      tool.InputSchema,
			logger:      t.logger,
		}

		if err := server.RegisterHandler(handler); err != nil {
			t.logger.Error("Failed to register tool handler",
				zap.String("tool", tool.Name),
				zap.Error(err))
		} else {
			t.logger.Info("Successfully registered tool handler",
				zap.String("tool", tool.Name))
		}
	}
}

// ToolHandlerImpl implements the ToolHandler interface
type ToolHandlerImpl struct {
	name        string
	description string
	schema      map[string]interface{}
	logger      *zap.Logger
}

// Name returns the tool name
func (h *ToolHandlerImpl) Name() string {
	return h.name
}

// Description returns the tool description
func (h *ToolHandlerImpl) Description() string {
	return h.description
}

// Schema returns the tool input schema
func (h *ToolHandlerImpl) Schema() map[string]interface{} {
	return h.schema
}

// Handle processes the tool request
func (h *ToolHandlerImpl) Handle(ctx context.Context, request *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	h.logger.Info("Handling tool request",
		zap.String("tool", h.name),
		zap.Any("params", request.Params))

	// This is a placeholder implementation
	// In a real implementation, each tool would have its own handler logic
	result := map[string]interface{}{
		"tool":   h.name,
		"status": "implemented",
		"message": fmt.Sprintf("Tool %s is ready for implementation", h.name),
	}

	return NewSuccessResponse(request.ID, result), nil
}