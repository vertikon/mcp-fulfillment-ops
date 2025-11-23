package generators

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// GeneratorFactory manages the creation and lifecycle of generators
type GeneratorFactory struct {
	generators map[string]Generator
	mu         sync.RWMutex
	logger     *zap.Logger
	config     *FactoryConfig
}

// FactoryConfig holds configuration for the generator factory
type FactoryConfig struct {
	TemplateRoot  string                  `json:"template_root"`
	DefaultStack  string                  `json:"default_stack"`
	CacheEnabled  bool                    `json:"cache_enabled"`
	MaxConcurrent int                     `json:"max_concurrent"`
	StackConfigs  map[string]*StackConfig `json:"stack_configs"`
}

// StackConfig holds configuration for a specific stack
type StackConfig struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	TemplateDir  string   `json:"template_dir"`
	Features     []string `json:"features"`
	Dependencies []string `json:"dependencies"`
	SupportedOS  []string `json:"supported_os"`
	MinGoVersion string   `json:"min_go_version,omitempty"`
	Requirements []string `json:"requirements"`
}

// Generator defines the interface for all generators
type Generator interface {
	Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error)
	GetStack() string
	GetName() string
	Validate() error
	GetTemplateFilesInfo() ([]map[string]interface{}, error)
}

// NewGeneratorFactory creates a new generator factory
func NewGeneratorFactory(config *FactoryConfig) *GeneratorFactory {
	if config == nil {
		config = &FactoryConfig{
			TemplateRoot:  "./templates",
			DefaultStack:  "go",
			CacheEnabled:  true,
			MaxConcurrent: 10,
			StackConfigs:  make(map[string]*StackConfig),
		}
	}

	factory := &GeneratorFactory{
		generators: make(map[string]Generator),
		config:     config,
		logger:     logger.Get(),
	}

	// Initialize default stack configurations
	factory.initializeDefaultStackConfigs()

	// Register default generators
	factory.registerDefaultGenerators()

	return factory
}

// initializeDefaultStackConfigs sets up default stack configurations
func (f *GeneratorFactory) initializeDefaultStackConfigs() {
	if len(f.config.StackConfigs) == 0 {
		f.config.StackConfigs = map[string]*StackConfig{
			"go": {
				Name:        "Go",
				Description: "Standard Go microservice with Clean Architecture",
				Version:     "1.0.0",
				TemplateDir: "go",
				Features: []string{
					"clean-architecture",
					"docker",
					"testing",
					"monitoring",
					"graceful-shutdown",
				},
				Dependencies: []string{},
				SupportedOS:  []string{"linux", "darwin", "windows"},
				MinGoVersion: "1.21",
				Requirements: []string{"go", "docker"},
			},
			"web": {
				Name:        "Web",
				Description: "React/Vue.js web application with modern tooling",
				Version:     "1.0.0",
				TemplateDir: "web",
				Features: []string{
					"typescript",
					"vite",
					"testing",
					"eslint",
					"prettier",
					"docker",
				},
				Dependencies: []string{},
				SupportedOS:  []string{"linux", "darwin", "windows"},
				Requirements: []string{"node", "npm", "docker"},
			},
			"tinygo": {
				Name:        "TinyGo",
				Description: "TinyGo project for WebAssembly and embedded systems",
				Version:     "1.0.0",
				TemplateDir: "tinygo",
				Features: []string{
					"wasm",
					"embedded",
					"testing",
					"docker",
				},
				Dependencies: []string{},
				SupportedOS:  []string{"linux", "darwin", "windows"},
				MinGoVersion: "1.21",
				Requirements: []string{"tinygo", "go"},
			},
			"wasm": {
				Name:        "WebAssembly",
				Description: "WebAssembly project using Rust/C++",
				Version:     "1.0.0",
				TemplateDir: "wasm",
				Features: []string{
					"rust",
					"wasm-bindgen",
					"testing",
					"docker",
				},
				Dependencies: []string{},
				SupportedOS:  []string{"linux", "darwin", "windows"},
				Requirements: []string{"rust", "cargo"},
			},
			"mcp-go-premium": {
				Name:        "MCP Go Premium",
				Description: "Premium Go MCP server with AI capabilities",
				Version:     "1.0.0",
				TemplateDir: "mcp-go-premium",
				Features: []string{
					"mcp-protocol",
					"ai-integration",
					"clean-architecture",
					"monitoring",
					"security",
					"docker",
					"kubernetes",
				},
				Dependencies: []string{},
				SupportedOS:  []string{"linux", "darwin", "windows"},
				MinGoVersion: "1.21",
				Requirements: []string{"go", "docker", "kubectl"},
			},
		}
	}
}

// registerDefaultGenerators registers the default set of generators
func (f *GeneratorFactory) registerDefaultGenerators() {
	// Register Go generator
	goGen := NewGoGenerator(filepath.Join(f.config.TemplateRoot, "go"))
	f.RegisterGenerator("go", goGen)

	// Register Web generator
	webGen := NewWebGenerator(filepath.Join(f.config.TemplateRoot, "web"))
	f.RegisterGenerator("web", webGen)

	// Register TinyGo generator
	tinyGen := NewTinyGoGenerator(filepath.Join(f.config.TemplateRoot, "tinygo"))
	f.RegisterGenerator("tinygo", tinyGen)

	// Register WebAssembly generator
	wasmGen := NewWasmGenerator(filepath.Join(f.config.TemplateRoot, "wasm"))
	f.RegisterGenerator("wasm", wasmGen)

	// Register MCP Go Premium generator
	mcpGen := NewMCPGoPremiumGenerator(filepath.Join(f.config.TemplateRoot, "mcp-go-premium"))
	f.RegisterGenerator("mcp-go-premium", mcpGen)
}

// RegisterGenerator registers a new generator
func (f *GeneratorFactory) RegisterGenerator(stack string, generator Generator) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if generator == nil {
		return fmt.Errorf("generator cannot be nil")
	}

	// Validate generator
	if err := generator.Validate(); err != nil {
		return fmt.Errorf("generator validation failed: %w", err)
	}

	// Check if generator already exists
	if _, exists := f.generators[stack]; exists {
		f.logger.Warn("Overriding existing generator", zap.String("stack", stack))
	}

	f.generators[stack] = generator
	f.logger.Info("Registered generator",
		zap.String("stack", stack),
		zap.String("name", generator.GetName()))

	return nil
}

// GetGenerator returns a generator for the specified stack
func (f *GeneratorFactory) GetGenerator(stack string) (Generator, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	generator, exists := f.generators[stack]
	if !exists {
		return nil, fmt.Errorf("generator not found for stack: %s", stack)
	}

	return generator, nil
}

// ListGenerators returns a list of all available generators
func (f *GeneratorFactory) ListGenerators() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	generators := make([]string, 0, len(f.generators))
	for stack := range f.generators {
		generators = append(generators, stack)
	}

	return generators
}

// GetGeneratorInfo returns information about a specific generator
func (f *GeneratorFactory) GetGeneratorInfo(stack string) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	generator, exists := f.generators[stack]
	if !exists {
		return nil, fmt.Errorf("generator not found for stack: %s", stack)
	}

	stackConfig, configExists := f.config.StackConfigs[stack]

	// Get template files info
	templateFiles, err := generator.GetTemplateFilesInfo()
	if err != nil {
		f.logger.Warn("Failed to get template files info", zap.Error(err))
		templateFiles = []map[string]interface{}{}
	}

	info := map[string]interface{}{
		"stack":          generator.GetStack(),
		"name":           generator.GetName(),
		"template_files": templateFiles,
	}

	if configExists {
		info["config"] = map[string]interface{}{
			"description":  stackConfig.Description,
			"version":      stackConfig.Version,
			"features":     stackConfig.Features,
			"dependencies": stackConfig.Dependencies,
			"supported_os": stackConfig.SupportedOS,
			"requirements": stackConfig.Requirements,
		}
	}

	return info, nil
}

// GetAllGeneratorInfo returns information about all generators
func (f *GeneratorFactory) GetAllGeneratorInfo() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]interface{})

	for stack := range f.generators {
		info, err := f.GetGeneratorInfo(stack)
		if err != nil {
			f.logger.Warn("Failed to get generator info",
				zap.String("stack", stack),
				zap.Error(err))
			continue
		}
		result[stack] = info
	}

	return result
}

// ValidateRequest validates a generation request
func (f *GeneratorFactory) ValidateRequest(req GenerateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if req.Path == "" {
		return fmt.Errorf("output path is required")
	}

	// Check if stack is supported
	_, exists := f.generators[req.Stack]
	if !exists {
		return fmt.Errorf("unsupported stack: %s", req.Stack)
	}

	// Validate stack configuration
	if stackConfig, exists := f.config.StackConfigs[req.Stack]; exists {
		// Validate features
		for _, feature := range req.Features {
			validFeature := false
			for _, validFeat := range stackConfig.Features {
				if feature == validFeat {
					validFeature = true
					break
				}
			}
			if !validFeature {
				return fmt.Errorf("invalid feature '%s' for stack '%s'", feature, req.Stack)
			}
		}
	}

	return nil
}

// Generate generates a project using the specified stack
func (f *GeneratorFactory) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Validate request
	if err := f.ValidateRequest(req); err != nil {
		return nil, err
	}

	// Get generator
	generator, err := f.GetGenerator(req.Stack)
	if err != nil {
		return nil, err
	}

	// Generate project
	result, err := generator.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	return result, nil
}

// RemoveGenerator removes a generator from the factory
func (f *GeneratorFactory) RemoveGenerator(stack string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.generators[stack]; !exists {
		return fmt.Errorf("generator not found for stack: %s", stack)
	}

	delete(f.generators, stack)
	f.logger.Info("Removed generator", zap.String("stack", stack))

	return nil
}

// HasGenerator checks if a generator exists for the specified stack
func (f *GeneratorFactory) HasGenerator(stack string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.generators[stack]
	return exists
}

// GetDefaultStack returns the default stack
func (f *GeneratorFactory) GetDefaultStack() string {
	return f.config.DefaultStack
}

// SetDefaultStack sets the default stack
func (f *GeneratorFactory) SetDefaultStack(stack string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.generators[stack]; !exists {
		return fmt.Errorf("generator not found for stack: %s", stack)
	}

	f.config.DefaultStack = stack
	f.logger.Info("Set default stack", zap.String("stack", stack))

	return nil
}

// GetConfig returns the factory configuration
func (f *GeneratorFactory) GetConfig() *FactoryConfig {
	return f.config
}

// UpdateConfig updates the factory configuration
func (f *GeneratorFactory) UpdateConfig(config *FactoryConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.config = config
	f.logger.Info("Updated factory configuration")

	return nil
}

// GetFactoryStats returns statistics about the generator factory
type FactoryStats struct {
	TotalGenerators int      `json:"total_generators"`
	AvailableStacks []string `json:"available_stacks"`
	DefaultStack    string   `json:"default_stack"`
	TemplateRoot    string   `json:"template_root"`
	CacheEnabled    bool     `json:"cache_enabled"`
	MaxConcurrent   int      `json:"max_concurrent"`
}

func (f *GeneratorFactory) GetFactoryStats() FactoryStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stacks := make([]string, 0, len(f.generators))
	for stack := range f.generators {
		stacks = append(stacks, stack)
	}

	return FactoryStats{
		TotalGenerators: len(f.generators),
		AvailableStacks: stacks,
		DefaultStack:    f.config.DefaultStack,
		TemplateRoot:    f.config.TemplateRoot,
		CacheEnabled:    f.config.CacheEnabled,
		MaxConcurrent:   f.config.MaxConcurrent,
	}
}

// Shutdown gracefully shuts down the generator factory
func (f *GeneratorFactory) Shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Info("Shutting down generator factory")

	// Clear generators
	f.generators = make(map[string]Generator)

	return nil
}
