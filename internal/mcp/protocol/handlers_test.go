package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/generators"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/registry"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/validators"
)

func TestNewHandlerManager(t *testing.T) {
	genFactory := generators.NewGeneratorFactory(nil)
	valFactory := validators.NewValidatorFactory()
	reg := registry.NewMCPRegistry(nil)

	hm := NewHandlerManager(genFactory, valFactory, reg)
	if hm == nil {
		t.Fatal("NewHandlerManager returned nil")
	}
	if hm.generatorFactory == nil {
		t.Error("generatorFactory should not be nil")
	}
	if hm.validatorFactory == nil {
		t.Error("validatorFactory should not be nil")
	}
	if hm.registry == nil {
		t.Error("registry should not be nil")
	}
}

func TestHandlerManager_GetAllHandlers(t *testing.T) {
	genFactory := generators.NewGeneratorFactory(nil)
	valFactory := validators.NewValidatorFactory()
	reg := registry.NewMCPRegistry(nil)

	hm := NewHandlerManager(genFactory, valFactory, reg)
	handlers := hm.GetAllHandlers()

	expectedHandlers := []string{
		"generate_project",
		"validate_project",
		"list_templates",
		"describe_stack",
		"list_projects",
		"get_project_info",
		"delete_project",
		"update_project",
	}

	if len(handlers) != len(expectedHandlers) {
		t.Errorf("Expected %d handlers, got %d", len(expectedHandlers), len(handlers))
	}

	for _, expected := range expectedHandlers {
		if _, exists := handlers[expected]; !exists {
			t.Errorf("Handler '%s' not found", expected)
		}
	}
}

func TestGenerateProjectHandler(t *testing.T) {
	genFactory := generators.NewGeneratorFactory(nil)
	reg := registry.NewMCPRegistry(nil)

	handler := &GenerateProjectHandler{
		generatorFactory: genFactory,
		registry:         reg,
	}

	if handler.Name() != "generate_project" {
		t.Errorf("Expected name 'generate_project', got '%s'", handler.Name())
	}

	schema := handler.Schema()
	if schema == nil {
		t.Error("Schema should not be nil")
	}
}

func TestValidateProjectHandler(t *testing.T) {
	valFactory := validators.NewValidatorFactory()

	handler := &ValidateProjectHandler{
		validatorFactory: valFactory,
	}

	if handler.Name() != "validate_project" {
		t.Errorf("Expected name 'validate_project', got '%s'", handler.Name())
	}
}

func TestListTemplatesHandler(t *testing.T) {
	reg := registry.NewMCPRegistry(nil)

	handler := &ListTemplatesHandler{
		registry: reg,
	}

	if handler.Name() != "list_templates" {
		t.Errorf("Expected name 'list_templates', got '%s'", handler.Name())
	}

	// Test handler execution
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "list_templates",
		Params:  map[string]interface{}{},
		ID:      "1",
	}

	response, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error != nil {
		t.Errorf("Handle() returned error: %v", response.Error)
	}
}

func TestDescribeStackHandler(t *testing.T) {
	reg := registry.NewMCPRegistry(nil)

	handler := &DescribeStackHandler{
		registry: reg,
	}

	if handler.Name() != "describe_stack" {
		t.Errorf("Expected name 'describe_stack', got '%s'", handler.Name())
	}

	// Test handler execution
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "describe_stack",
		Params: map[string]interface{}{
			"stack": "go",
		},
		ID: "1",
	}

	response, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error != nil {
		t.Errorf("Handle() returned error: %v", response.Error)
	}
}

func TestListProjectsHandler(t *testing.T) {
	reg := registry.NewMCPRegistry(nil)

	handler := &ListProjectsHandler{
		registry: reg,
	}

	if handler.Name() != "list_projects" {
		t.Errorf("Expected name 'list_projects', got '%s'", handler.Name())
	}

	// Test handler execution
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "list_projects",
		Params:  map[string]interface{}{},
		ID:      "1",
	}

	response, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error != nil {
		t.Errorf("Handle() returned error: %v", response.Error)
	}
}

func TestGetProjectInfoHandler(t *testing.T) {
	reg := registry.NewMCPRegistry(nil)

	// Register a test project
	projectInfo := registry.ProjectInfo{
		Name:      "test-project",
		Stack:     "go",
		Path:      "/tmp/test-project",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = reg.RegisterProject(projectInfo)

	handler := &GetProjectInfoHandler{
		registry: reg,
	}

	if handler.Name() != "get_project_info" {
		t.Errorf("Expected name 'get_project_info', got '%s'", handler.Name())
	}

	// Test handler execution
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "get_project_info",
		Params: map[string]interface{}{
			"name": "test-project",
		},
		ID: "1",
	}

	response, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error != nil {
		t.Errorf("Handle() returned error: %v", response.Error)
	}
}

func TestDeleteProjectHandler(t *testing.T) {
	reg := registry.NewMCPRegistry(nil)

	// Register a test project
	projectInfo := registry.ProjectInfo{
		Name:      "test-delete",
		Stack:     "go",
		Path:      "/tmp/test-delete",
		Status:    "active",
		CreatedAt: time.Now(),
	}
	_ = reg.RegisterProject(projectInfo)

	handler := &DeleteProjectHandler{
		registry: reg,
	}

	if handler.Name() != "delete_project" {
		t.Errorf("Expected name 'delete_project', got '%s'", handler.Name())
	}

	// Test handler execution without confirm (should fail)
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "delete_project",
		Params: map[string]interface{}{
			"name":    "test-delete",
			"confirm": false,
		},
		ID: "1",
	}

	response, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error == nil {
		t.Error("Expected error when confirm is false")
	}

	// Test handler execution with confirm
	request = &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "delete_project",
		Params: map[string]interface{}{
			"name":    "test-delete",
			"confirm": true,
		},
		ID: "1",
	}

	response, err = handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error != nil {
		t.Errorf("Handle() returned error: %v", response.Error)
	}
}

func TestUpdateProjectHandler(t *testing.T) {
	genFactory := generators.NewGeneratorFactory(nil)
	valFactory := validators.NewValidatorFactory()
	reg := registry.NewMCPRegistry(nil)

	// Register a test project
	projectInfo := registry.ProjectInfo{
		Name:      "test-update",
		Stack:     "go",
		Path:      "/tmp/test-update",
		Status:    "active",
		Features:  []string{"feature1"},
		CreatedAt: time.Now(),
	}
	_ = reg.RegisterProject(projectInfo)

	handler := &UpdateProjectHandler{
		generatorFactory: genFactory,
		validatorFactory: valFactory,
		registry:         reg,
	}

	if handler.Name() != "update_project" {
		t.Errorf("Expected name 'update_project', got '%s'", handler.Name())
	}

	// Test handler execution
	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "update_project",
		Params: map[string]interface{}{
			"name":         "test-update",
			"add_features": []string{"feature2"},
		},
		ID: "1",
	}

	response, err := handler.Handle(context.Background(), request)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if response.Error != nil {
		t.Errorf("Handle() returned error: %v", response.Error)
	}
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		params  interface{}
		target  interface{}
		wantErr bool
	}{
		{
			name:    "nil params",
			params:  nil,
			target:  &struct{}{},
			wantErr: false,
		},
		{
			name:   "valid params",
			params: map[string]interface{}{"name": "test"},
			target: &struct {
				Name string `json:"name"`
			}{},
			wantErr: false,
		},
		{
			name:    "invalid params",
			params:  "not an object",
			target:  &struct{}{},
			wantErr: false, // parseParams doesn't validate, just unmarshals
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseParams(tt.params, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
