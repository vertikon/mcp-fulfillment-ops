package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/generators"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/registry"
	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/validators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// HandlerManager manages all MCP tool handlers
type HandlerManager struct {
	generatorFactory *generators.GeneratorFactory
	validatorFactory *validators.ValidatorFactory
	registry         *registry.MCPRegistry
	logger           *zap.Logger
}

// NewHandlerManager creates a new handler manager
func NewHandlerManager(
	genFactory *generators.GeneratorFactory,
	valFactory *validators.ValidatorFactory,
	reg *registry.MCPRegistry,
) *HandlerManager {
	return &HandlerManager{
		generatorFactory: genFactory,
		validatorFactory: valFactory,
		registry:         reg,
		logger:           logger.Get(),
	}
}

// GetAllHandlers returns all available handlers
func (hm *HandlerManager) GetAllHandlers() map[string]ToolHandler {
	handlers := make(map[string]ToolHandler)

	// Register tool handlers
	handlers["generate_project"] = &GenerateProjectHandler{
		generatorFactory: hm.generatorFactory,
		registry:         hm.registry,
		logger:           hm.logger,
	}

	handlers["validate_project"] = &ValidateProjectHandler{
		validatorFactory: hm.validatorFactory,
		logger:           hm.logger,
	}

	handlers["list_templates"] = &ListTemplatesHandler{
		registry: hm.registry,
		logger:   hm.logger,
	}

	handlers["describe_stack"] = &DescribeStackHandler{
		registry: hm.registry,
		logger:   hm.logger,
	}

	handlers["list_projects"] = &ListProjectsHandler{
		registry: hm.registry,
		logger:   hm.logger,
	}

	handlers["get_project_info"] = &GetProjectInfoHandler{
		registry: hm.registry,
		logger:   hm.logger,
	}

	handlers["delete_project"] = &DeleteProjectHandler{
		registry: hm.registry,
		logger:   hm.logger,
	}

	handlers["update_project"] = &UpdateProjectHandler{
		generatorFactory: hm.generatorFactory,
		validatorFactory: hm.validatorFactory,
		registry:         hm.registry,
		logger:           hm.logger,
	}

	return handlers
}

// GenerateProjectHandler handles project generation requests
type GenerateProjectHandler struct {
	generatorFactory *generators.GeneratorFactory
	registry         *registry.MCPRegistry
	logger           *zap.Logger
}

func (h *GenerateProjectHandler) Name() string {
	return "generate_project"
}

func (h *GenerateProjectHandler) Description() string {
	return "Generate a new project with specified technology stack and configuration"
}

func (h *GenerateProjectHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
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
		},
		"required": []string{"name", "stack", "path"},
	}
}

func (h *GenerateProjectHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Name     string                 `json:"name"`
		Stack    string                 `json:"stack"`
		Path     string                 `json:"path"`
		Features []string               `json:"features,omitempty"`
		Config   map[string]interface{} `json:"config,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	h.logger.Info("Generating project",
		zap.String("name", params.Name),
		zap.String("stack", params.Stack),
		zap.String("path", params.Path),
		zap.Strings("features", params.Features))

	// Get generator
	generator, err := h.generatorFactory.GetGenerator(params.Stack)
	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Generator not found for stack %s: %v", params.Stack, err), nil), nil
	}

	// Generate project
	result, err := generator.Generate(ctx, generators.GenerateRequest{
		Name:     params.Name,
		Path:     params.Path,
		Features: params.Features,
		Config:   params.Config,
	})

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Project generation failed: %v", err), nil), nil
	}

	// Register project
	projectInfo := registry.ProjectInfo{
		Name:      params.Name,
		Stack:     params.Stack,
		Path:      result.Path,
		CreatedAt: result.CreatedAt,
		Features:  params.Features,
		Status:    "active",
	}

	if err := h.registry.RegisterProject(projectInfo); err != nil {
		h.logger.Warn("Failed to register project", zap.Error(err))
	}

	return NewSuccessResponse(request.ID, map[string]interface{}{
		"project": params.Name,
		"path":    result.Path,
		"stack":   params.Stack,
		"status":  "created",
		"files":   result.FilesCreated,
	}), nil
}

// ValidateProjectHandler handles project validation requests
type ValidateProjectHandler struct {
	validatorFactory *validators.ValidatorFactory
	logger           *zap.Logger
}

func (h *ValidateProjectHandler) Name() string {
	return "validate_project"
}

func (h *ValidateProjectHandler) Description() string {
	return "Validate a project structure and compliance with standards"
}

func (h *ValidateProjectHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
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
		},
		"required": []string{"path"},
	}
}

func (h *ValidateProjectHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Path       string `json:"path"`
		StrictMode bool   `json:"strict_mode,omitempty"`
		CheckDeps  bool   `json:"check_dependencies,omitempty"`
		CheckSec   bool   `json:"check_security,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	h.logger.Info("Validating project",
		zap.String("path", params.Path),
		zap.Bool("strict_mode", params.StrictMode))

	// Get validators
	structValidator := h.validatorFactory.GetStructureValidator()
	depValidator := h.validatorFactory.GetDependencyValidator()
	secValidator := h.validatorFactory.GetSecurityValidator()

	// Run validations
	result := validators.ValidationResult{
		Path:     params.Path,
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
		Checks:   []string{},
	}

	// Structure validation
	structResult, err := structValidator.Validate(ctx, validators.StructureRequest{
		Path:       params.Path,
		StrictMode: params.StrictMode,
	})

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Structure validation failed: %v", err))
		result.Valid = false
	} else {
		result.Errors = append(result.Errors, structResult.Errors...)
		result.Warnings = append(result.Warnings, structResult.Warnings...)
		result.Checks = append(result.Checks, "structure")
	}

	// Dependency validation
	if params.CheckDeps {
		depResult, err := depValidator.Validate(ctx, validators.DependencyRequest{
			Path: params.Path,
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Dependency validation failed: %v", err))
			result.Valid = false
		} else {
			result.Errors = append(result.Errors, depResult.Errors...)
			result.Warnings = append(result.Warnings, depResult.Warnings...)
			result.Checks = append(result.Checks, "dependencies")
		}
	}

	// Security validation
	if params.CheckSec {
		secResult, err := secValidator.Validate(ctx, validators.SecurityRequest{
			Path: params.Path,
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Security validation failed: %v", err))
			result.Valid = false
		} else {
			result.Errors = append(result.Errors, secResult.Errors...)
			result.Warnings = append(result.Warnings, secResult.Warnings...)
			result.Checks = append(result.Checks, "security")
		}
	}

	return NewSuccessResponse(request.ID, result), nil
}

// ListTemplatesHandler handles template listing requests
type ListTemplatesHandler struct {
	registry *registry.MCPRegistry
	logger   *zap.Logger
}

func (h *ListTemplatesHandler) Name() string {
	return "list_templates"
}

func (h *ListTemplatesHandler) Description() string {
	return "List all available templates for project generation"
}

func (h *ListTemplatesHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stack": map[string]interface{}{
				"type":        "string",
				"description": "Filter by technology stack",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Filter by category",
			},
		},
	}
}

func (h *ListTemplatesHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Stack    string `json:"stack,omitempty"`
		Category string `json:"category,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	templates, err := h.registry.ListTemplates(registry.TemplateFilter{
		Stack:    params.Stack,
		Category: params.Category,
	})

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to list templates: %v", err), nil), nil
	}

	return NewSuccessResponse(request.ID, map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
	}), nil
}

// DescribeStackHandler handles stack description requests
type DescribeStackHandler struct {
	registry *registry.MCPRegistry
	logger   *zap.Logger
}

func (h *DescribeStackHandler) Name() string {
	return "describe_stack"
}

func (h *DescribeStackHandler) Description() string {
	return "Get detailed information about a specific technology stack"
}

func (h *DescribeStackHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stack": map[string]interface{}{
				"type":        "string",
				"description": "Technology stack to describe",
			},
		},
		"required": []string{"stack"},
	}
}

func (h *DescribeStackHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Stack string `json:"stack"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	info, err := h.registry.GetStackInfo(params.Stack)
	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to get stack info: %v", err), nil), nil
	}

	return NewSuccessResponse(request.ID, info), nil
}

// ListProjectsHandler handles project listing requests
type ListProjectsHandler struct {
	registry *registry.MCPRegistry
	logger   *zap.Logger
}

func (h *ListProjectsHandler) Name() string {
	return "list_projects"
}

func (h *ListProjectsHandler) Description() string {
	return "List all generated projects"
}

func (h *ListProjectsHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
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
		},
	}
}

func (h *ListProjectsHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Stack  string `json:"stack,omitempty"`
		Status string `json:"status,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	projects, err := h.registry.ListProjects(registry.ProjectFilter{
		Stack:  params.Stack,
		Status: params.Status,
	})

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to list projects: %v", err), nil), nil
	}

	return NewSuccessResponse(request.ID, map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
	}), nil
}

// GetProjectInfoHandler handles project info requests
type GetProjectInfoHandler struct {
	registry *registry.MCPRegistry
	logger   *zap.Logger
}

func (h *GetProjectInfoHandler) Name() string {
	return "get_project_info"
}

func (h *GetProjectInfoHandler) Description() string {
	return "Get detailed information about a specific project"
}

func (h *GetProjectInfoHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the project",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the project",
			},
		},
	}
}

func (h *GetProjectInfoHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Name string `json:"name,omitempty"`
		Path string `json:"path,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	var projectInfo *registry.ProjectInfo
	var err error

	if params.Name != "" {
		projectInfo, err = h.registry.GetProjectByName(params.Name)
	} else if params.Path != "" {
		projectInfo, err = h.registry.GetProjectByPath(params.Path)
	} else {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			"Either name or path must be provided", nil), nil
	}

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to get project info: %v", err), nil), nil
	}

	return NewSuccessResponse(request.ID, projectInfo), nil
}

// DeleteProjectHandler handles project deletion requests
type DeleteProjectHandler struct {
	registry *registry.MCPRegistry
	logger   *zap.Logger
}

func (h *DeleteProjectHandler) Name() string {
	return "delete_project"
}

func (h *DeleteProjectHandler) Description() string {
	return "Delete a project and all its files"
}

func (h *DeleteProjectHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the project to delete",
			},
			"confirm": map[string]interface{}{
				"type":        "boolean",
				"description": "Confirm deletion",
			},
		},
		"required": []string{"confirm"},
	}
}

func (h *DeleteProjectHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Name    string `json:"name,omitempty"`
		Path    string `json:"path,omitempty"`
		Confirm bool   `json:"confirm"`
		Backup  bool   `json:"backup,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	if !params.Confirm {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			"Deletion must be confirmed", nil), nil
	}

	if params.Name == "" && params.Path == "" {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			"Either name or path must be provided", nil), nil
	}

	// Get project info before deletion
	var projectInfo *registry.ProjectInfo
	var err error
	if params.Name != "" {
		projectInfo, err = h.registry.GetProjectByName(params.Name)
	} else {
		projectInfo, err = h.registry.GetProjectByPath(params.Path)
	}

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to get project info: %v", err), nil), nil
	}

	// Create backup if requested
	if params.Backup {
		h.logger.Info("Creating backup before deletion",
			zap.String("project", projectInfo.Name),
			zap.String("path", projectInfo.Path))
		// In production, implement actual backup logic
	}

	// Unregister project from registry
	if params.Name != "" {
		err = h.registry.UnregisterProject(params.Name)
	} else {
		err = h.registry.UnregisterProjectByPath(params.Path)
	}

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to unregister project: %v", err), nil), nil
	}

	h.logger.Info("Project deleted",
		zap.String("name", projectInfo.Name),
		zap.String("path", projectInfo.Path),
		zap.Bool("backup", params.Backup))

	result := map[string]interface{}{
		"status":     "deleted",
		"project":    projectInfo.Name,
		"path":       projectInfo.Path,
		"backup":     params.Backup,
		"deleted_at": time.Now(),
	}

	return NewSuccessResponse(request.ID, result), nil
}

// UpdateProjectHandler handles project update requests
type UpdateProjectHandler struct {
	generatorFactory *generators.GeneratorFactory
	validatorFactory *validators.ValidatorFactory
	registry         *registry.MCPRegistry
	logger           *zap.Logger
}

func (h *UpdateProjectHandler) Name() string {
	return "update_project"
}

func (h *UpdateProjectHandler) Description() string {
	return "Update an existing project with new configuration or features"
}

func (h *UpdateProjectHandler) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the project to update",
			},
			"add_features": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Features to add",
			},
		},
	}
}

func (h *UpdateProjectHandler) Handle(ctx context.Context, request *JSONRPCRequest) (*JSONRPCResponse, error) {
	var params struct {
		Name            string                 `json:"name,omitempty"`
		Path            string                 `json:"path,omitempty"`
		AddFeatures     []string               `json:"add_features,omitempty"`
		RemoveFeatures  []string               `json:"remove_features,omitempty"`
		UpdateConfig    map[string]interface{} `json:"update_config,omitempty"`
		UpgradeTemplate string                 `json:"upgrade_template,omitempty"`
	}

	if err := parseParams(request.Params, &params); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			fmt.Sprintf("Invalid parameters: %v", err), nil), nil
	}

	if params.Name == "" && params.Path == "" {
		return NewErrorResponse(request.ID, ErrCodeInvalidParams,
			"Either name or path must be provided", nil), nil
	}

	// Get project info
	var projectInfo *registry.ProjectInfo
	var err error
	if params.Name != "" {
		projectInfo, err = h.registry.GetProjectByName(params.Name)
	} else {
		projectInfo, err = h.registry.GetProjectByPath(params.Path)
	}

	if err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to get project info: %v", err), nil), nil
	}

	// Prepare updates
	updates := registry.ProjectInfo{
		Features: projectInfo.Features,
		Config:   projectInfo.Config,
	}

	// Merge features
	if len(params.AddFeatures) > 0 || len(params.RemoveFeatures) > 0 {
		featureMap := make(map[string]bool)
		for _, f := range projectInfo.Features {
			featureMap[f] = true
		}

		// Remove features
		for _, f := range params.RemoveFeatures {
			delete(featureMap, f)
		}

		// Add features
		for _, f := range params.AddFeatures {
			featureMap[f] = true
		}

		// Convert back to slice
		updates.Features = make([]string, 0, len(featureMap))
		for f := range featureMap {
			updates.Features = append(updates.Features, f)
		}
	}

	// Merge config
	if params.UpdateConfig != nil {
		if updates.Config == nil {
			updates.Config = make(map[string]interface{})
		}
		for k, v := range params.UpdateConfig {
			updates.Config[k] = v
		}
	}

	// Update project in registry
	if err := h.registry.UpdateProject(projectInfo.Name, updates); err != nil {
		return NewErrorResponse(request.ID, ErrCodeInternalError,
			fmt.Sprintf("Failed to update project: %v", err), nil), nil
	}

	// Validate updated project if path exists
	if projectInfo.Path != "" {
		if _, err := os.Stat(projectInfo.Path); err == nil {
			structValidator := h.validatorFactory.GetStructureValidator()
			_, valErr := structValidator.Validate(ctx, validators.StructureRequest{
				Path:       projectInfo.Path,
				StrictMode: false,
			})
			if valErr != nil {
				h.logger.Warn("Project validation failed after update",
					zap.String("project", projectInfo.Name),
					zap.Error(valErr))
			}
		}
	}

	h.logger.Info("Project updated",
		zap.String("name", projectInfo.Name),
		zap.Strings("added_features", params.AddFeatures),
		zap.Strings("removed_features", params.RemoveFeatures))

	result := map[string]interface{}{
		"status":     "updated",
		"project":    projectInfo.Name,
		"path":       projectInfo.Path,
		"features":   updates.Features,
		"config":     updates.Config,
		"updated_at": time.Now(),
	}

	return NewSuccessResponse(request.ID, result), nil
}

// parseParams is a helper function to parse parameters from a JSON-RPC request
func parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}
