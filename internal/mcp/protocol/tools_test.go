package protocol

import (
	"testing"
)

func TestNewMCPTools(t *testing.T) {
	tools := NewMCPTools()
	if tools == nil {
		t.Fatal("NewMCPTools returned nil")
	}
}

func TestMCPTools_GetToolDefinitions(t *testing.T) {
	tools := NewMCPTools()
	definitions := tools.GetToolDefinitions()

	expectedTools := []string{
		"generate_project",
		"validate_project",
		"list_templates",
		"describe_stack",
		"list_projects",
		"get_project_info",
		"delete_project",
		"update_project",
	}

	if len(definitions) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(definitions))
	}

	toolMap := make(map[string]bool)
	for _, tool := range definitions {
		toolMap[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolMap[expectedTool] {
			t.Errorf("Tool '%s' not found in definitions", expectedTool)
		}
	}
}

func TestMCPTools_GenerateProjectTool(t *testing.T) {
	tools := NewMCPTools()
	tool := tools.generateProjectTool()

	if tool.Name != "generate_project" {
		t.Errorf("Expected name 'generate_project', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description should not be empty")
	}

	schema := tool.InputSchema
	if schema == nil {
		t.Fatal("Tool schema should not be nil")
	}

	// Check required fields
	if required, ok := schema["required"].([]interface{}); ok {
		requiredFields := []string{"name", "stack", "path"}
		for _, field := range requiredFields {
			found := false
			for _, req := range required {
				if req == field {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Required field '%s' not found in schema", field)
			}
		}
	}
}

func TestMCPTools_ValidateProjectTool(t *testing.T) {
	tools := NewMCPTools()
	tool := tools.validateProjectTool()

	if tool.Name != "validate_project" {
		t.Errorf("Expected name 'validate_project', got '%s'", tool.Name)
	}

	schema := tool.InputSchema
	if required, ok := schema["required"].([]interface{}); ok {
		if len(required) == 0 {
			t.Error("validate_project should have required fields")
		}
	}
}

func TestMCPTools_ListTemplatesTool(t *testing.T) {
	tools := NewMCPTools()
	tool := tools.listTemplatesTool()

	if tool.Name != "list_templates" {
		t.Errorf("Expected name 'list_templates', got '%s'", tool.Name)
	}
}

func TestMCPTools_DescribeStackTool(t *testing.T) {
	tools := NewMCPTools()
	tool := tools.describeStackTool()

	if tool.Name != "describe_stack" {
		t.Errorf("Expected name 'describe_stack', got '%s'", tool.Name)
	}

	schema := tool.InputSchema
	if required, ok := schema["required"].([]interface{}); ok {
		if len(required) == 0 {
			t.Error("describe_stack should have required fields")
		}
	}
}

func TestMCPTools_RegisterTools(t *testing.T) {
	tools := NewMCPTools()
	server := NewMCPServer(nil)

	// Register tools
	tools.RegisterTools(server)

	// Verify tools are registered
	capabilities := server.GetCapabilities()
	if capabilities == nil {
		t.Fatal("GetCapabilities() returned nil")
	}

	// Check that handlers were registered
	if len(server.handlers) == 0 {
		t.Error("No handlers were registered")
	}
}

func TestToolHandlerImpl(t *testing.T) {
	handler := &ToolHandlerImpl{
		name:        "test_tool",
		description: "Test tool description",
		schema: map[string]interface{}{
			"type": "object",
		},
	}

	if handler.Name() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", handler.Name())
	}

	if handler.Description() != "Test tool description" {
		t.Errorf("Expected description 'Test tool description', got '%s'", handler.Description())
	}

	schema := handler.Schema()
	if schema == nil {
		t.Error("Schema should not be nil")
	}
}

func TestMCPTools_ToolSchemas(t *testing.T) {
	tools := NewMCPTools()
	definitions := tools.GetToolDefinitions()

	for _, tool := range definitions {
		if tool.InputSchema == nil {
			t.Errorf("Tool '%s' has nil schema", tool.Name)
		}

		if tool.Description == "" {
			t.Errorf("Tool '%s' has empty description", tool.Name)
		}

		// Verify schema has type
		if schemaType, ok := tool.InputSchema["type"].(string); !ok || schemaType != "object" {
			t.Errorf("Tool '%s' schema should have type 'object'", tool.Name)
		}
	}
}
