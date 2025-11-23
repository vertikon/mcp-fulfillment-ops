package generators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/generators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// TemplateGenerator handles generation/instantiation of templates
// Converts templates/* → project
type TemplateGenerator struct {
	factory *generators.GeneratorFactory
	logger  *zap.Logger
}

// NewTemplateGenerator creates a new template generator
func NewTemplateGenerator(templateRoot string) *TemplateGenerator {
	config := &generators.FactoryConfig{
		TemplateRoot:  templateRoot,
		DefaultStack:  "go",
		CacheEnabled:  true,
		MaxConcurrent: 10,
	}

	factory := generators.NewGeneratorFactory(config)

	return &TemplateGenerator{
		factory: factory,
		logger:  logger.Get(),
	}
}

// GenerateFromTemplate generates a project from a template
func (g *TemplateGenerator) GenerateFromTemplate(ctx context.Context, req TemplateGenerateRequest) (*TemplateGenerateResult, error) {
	g.logger.Info("Generating project from template",
		zap.String("template", req.TemplateName),
		zap.String("project", req.ProjectName),
		zap.String("path", req.OutputPath))

	// Validate template exists
	templatePath := filepath.Join(g.factory.GetConfig().TemplateRoot, req.TemplateName)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", req.TemplateName)
	}

	// Convert request to internal format
	internalReq := generators.GenerateRequest{
		Name:     req.ProjectName,
		Path:     req.OutputPath,
		Features: req.Features,
		Config:   req.Config,
	}

	// Determine stack from template name
	stack := g.determineStackFromTemplate(req.TemplateName)

	// Get generator for the stack
	gen, err := g.factory.GetGenerator(stack)
	if err != nil {
		return nil, fmt.Errorf("failed to get generator for stack %s: %w", stack, err)
	}

	// Generate project
	result, err := gen.Generate(ctx, internalReq)
	if err != nil {
		return nil, fmt.Errorf("template generation failed: %w", err)
	}

	// Convert result to external format
	return &TemplateGenerateResult{
		Path:         result.Path,
		FilesCreated: result.FilesCreated,
		CreatedAt:    result.CreatedAt,
		Duration:     result.Duration,
		Size:         result.Size,
		Template:     req.TemplateName,
		Stack:        stack,
	}, nil
}

// TemplateGenerateRequest represents a request to generate from a template
type TemplateGenerateRequest struct {
	TemplateName string                 `json:"template_name"`
	ProjectName  string                 `json:"project_name"`
	OutputPath   string                 `json:"output_path"`
	Features     []string               `json:"features,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// TemplateGenerateResult represents the result of template generation
type TemplateGenerateResult struct {
	Path         string      `json:"path"`
	FilesCreated []string    `json:"files_created"`
	CreatedAt    interface{} `json:"created_at"`
	Duration     interface{} `json:"duration"`
	Size         int64       `json:"size"`
	Template     string      `json:"template"`
	Stack        string      `json:"stack"`
}

// ListAvailableTemplates returns a list of available templates
func (g *TemplateGenerator) ListAvailableTemplates() ([]string, error) {
	templateRoot := g.factory.GetConfig().TemplateRoot
	var templates []string

	err := filepath.Walk(templateRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != templateRoot {
			relPath, err := filepath.Rel(templateRoot, path)
			if err != nil {
				return err
			}
			templates = append(templates, relPath)
			return filepath.SkipDir
		}

		return nil
	})

	return templates, err
}

// GetTemplateInfo returns information about a specific template
func (g *TemplateGenerator) GetTemplateInfo(templateName string) (map[string]interface{}, error) {
	templatePath := filepath.Join(g.factory.GetConfig().TemplateRoot, templateName)

	info := make(map[string]interface{})
	info["name"] = templateName
	info["path"] = templatePath

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// Get generator info if stack can be determined
	stack := g.determineStackFromTemplate(templateName)
	if g.factory.HasGenerator(stack) {
		genInfo, err := g.factory.GetGeneratorInfo(stack)
		if err == nil {
			info["generator"] = genInfo
		}
	}

	return info, nil
}

// determineStackFromTemplate determines the stack from template name
func (g *TemplateGenerator) determineStackFromTemplate(templateName string) string {
	// Map common template names to stacks
	templateMap := map[string]string{
		"base":           "go",
		"go":             "go",
		"web":            "web",
		"tinygo":         "tinygo",
		"wasm":           "wasm",
		"mcp-go-premium": "mcp-go-premium",
	}

	if stack, ok := templateMap[templateName]; ok {
		return stack
	}

	// Default to go if not found
	return "go"
}

// Validate validates the template generation request
func (g *TemplateGenerator) Validate(req TemplateGenerateRequest) error {
	if req.TemplateName == "" {
		return fmt.Errorf("template name is required")
	}

	if req.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}

	if req.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}

	// Validate template exists
	templatePath := filepath.Join(g.factory.GetConfig().TemplateRoot, req.TemplateName)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template not found: %s", req.TemplateName)
	}

	return nil
}
