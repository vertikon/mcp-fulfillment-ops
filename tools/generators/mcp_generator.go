package generators

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/generators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MCPGenerator orchestrates MCP generation using internal/mcp/generators
// This is the CLI/Tool interface for generating MCPs
type MCPGenerator struct {
	factory *generators.GeneratorFactory
	logger  *zap.Logger
}

// NewMCPGenerator creates a new MCP generator
func NewMCPGenerator(templateRoot string) *MCPGenerator {
	config := &generators.FactoryConfig{
		TemplateRoot:  templateRoot,
		DefaultStack:  "mcp-go-premium",
		CacheEnabled:  true,
		MaxConcurrent: 10,
	}

	factory := generators.NewGeneratorFactory(config)

	return &MCPGenerator{
		factory: factory,
		logger:  logger.Get(),
	}
}

// GenerateMCP generates a complete MCP project
func (g *MCPGenerator) GenerateMCP(ctx context.Context, req MCPGenerateRequest) (*MCPGenerateResult, error) {
	g.logger.Info("Generating MCP project",
		zap.String("name", req.Name),
		zap.String("stack", req.Stack),
		zap.String("path", req.Path))

	// Convert request to internal format
	internalReq := generators.GenerateRequest{
		Name:     req.Name,
		Path:     req.Path,
		Features: req.Features,
		Config:   req.Config,
	}

	// Use internal generator factory
	result, err := g.factory.Generate(ctx, internalReq)
	if err != nil {
		return nil, fmt.Errorf("MCP generation failed: %w", err)
	}

	// Convert result to external format
	return &MCPGenerateResult{
		Path:         result.Path,
		FilesCreated: result.FilesCreated,
		CreatedAt:    result.CreatedAt,
		Duration:     result.Duration,
		Size:         result.Size,
		Metadata:     result.Metadata,
	}, nil
}

// MCPGenerateRequest represents a request to generate an MCP
type MCPGenerateRequest struct {
	Name     string                 `json:"name"`
	Stack    string                 `json:"stack"`
	Path     string                 `json:"path"`
	Features []string               `json:"features,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// MCPGenerateResult represents the result of MCP generation
type MCPGenerateResult struct {
	Path         string            `json:"path"`
	FilesCreated []string          `json:"files_created"`
	CreatedAt    interface{}       `json:"created_at"`
	Duration     interface{}       `json:"duration"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata"`
}

// Validate validates the MCP generation request
func (g *MCPGenerator) Validate(req MCPGenerateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("MCP name is required")
	}

	if req.Path == "" {
		return fmt.Errorf("output path is required")
	}

	if req.Stack == "" {
		req.Stack = "mcp-go-premium"
	}

	// Validate stack is supported
	if !g.factory.HasGenerator(req.Stack) {
		return fmt.Errorf("unsupported stack: %s", req.Stack)
	}

	return nil
}

// ListAvailableStacks returns a list of available MCP stacks
func (g *MCPGenerator) ListAvailableStacks() []string {
	return g.factory.ListGenerators()
}

// GetStackInfo returns information about a specific stack
func (g *MCPGenerator) GetStackInfo(stack string) (map[string]interface{}, error) {
	return g.factory.GetGeneratorInfo(stack)
}

// GetFactory returns the internal generator factory
func (g *MCPGenerator) GetFactory() *generators.GeneratorFactory {
	return g.factory
}
