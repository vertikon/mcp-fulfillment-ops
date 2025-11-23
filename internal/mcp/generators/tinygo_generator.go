package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// TinyGoGenerator generates TinyGo projects for WASM and embedded systems
type TinyGoGenerator struct {
	*BaseGenerator
	target string // wasm, embedded, microcontroller
}

// NewTinyGoGenerator creates a new TinyGo generator
func NewTinyGoGenerator(templateDir string) *TinyGoGenerator {
	base := NewBaseGenerator("TinyGo Generator", "tinygo", templateDir)
	return &TinyGoGenerator{
		BaseGenerator: base,
		target:        "wasm", // Default target
	}
}

// SetTarget sets the TinyGo compilation target
func (g *TinyGoGenerator) SetTarget(target string) {
	g.target = target
}

// Generate implements Generator interface
func (g *TinyGoGenerator) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Determine target from config
	if target, ok := req.Config["target"].(string); ok {
		g.target = target
	}

	// Call base generator with TinyGo-specific modifications
	result, err := g.BaseGenerator.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add TinyGo-specific post-processing
	if err := g.postProcessTinyGoProject(result.Path, req); err != nil {
		return nil, fmt.Errorf("TinyGo post-processing failed: %w", err)
	}

	return result, nil
}

// getTemplateFiles returns TinyGo-specific template files
func (g *TinyGoGenerator) getTemplateFiles(req GenerateRequest) ([]TemplateFile, error) {
	files := []TemplateFile{
		// Basic project files
		{
			SourcePath: filepath.Join(g.templateDir, "go.mod.tmpl"),
			TargetPath: "go.mod",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "README.md.tmpl"),
			TargetPath: "README.md",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, ".gitignore.tmpl"),
			TargetPath: ".gitignore",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "Makefile.tmpl"),
			TargetPath: "Makefile",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "Dockerfile.tmpl"),
			TargetPath: "Dockerfile",
			IsTemplate: true,
		},

		// Command files
		{
			SourcePath: filepath.Join(g.templateDir, "cmd", "__NAME__", "main.go.tmpl"),
			TargetPath: "cmd/__NAME__/main.go",
			IsTemplate: true,
		},

		// WASM files
		{
			SourcePath: filepath.Join(g.templateDir, "wasm", "exports.go.tmpl"),
			TargetPath: "wasm/exports.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "wasm", "main.go.tmpl"),
			TargetPath: "main.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "wasm", "index.html.tmpl"),
			TargetPath: "wasm/index.html",
			IsTemplate: true,
		},

		// Build scripts
		{
			SourcePath: filepath.Join(g.templateDir, "build.sh"),
			TargetPath: "build.sh",
			IsTemplate: true,
		},
	}

	return files, nil
}

// hasFeature checks if a feature is enabled
func (g *TinyGoGenerator) hasFeature(features []string, feature string) bool {
	for _, f := range features {
		if strings.EqualFold(f, feature) {
			return true
		}
	}
	return false
}

// postProcessTinyGoProject performs TinyGo-specific post-processing
func (g *TinyGoGenerator) postProcessTinyGoProject(projectPath string, req GenerateRequest) error {
	g.logger.Info("Starting TinyGo project post-processing",
		zap.String("project", req.Name),
		zap.String("path", projectPath),
		zap.String("target", g.target),
		zap.Strings("features", req.Features))

	// Verify go.mod exists
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		g.logger.Warn("go.mod not found, skipping post-processing",
			zap.String("path", goModPath))
		return nil
	}

	// Verify target-specific files
	switch g.target {
	case "wasm":
		wasmFiles := []string{"cmd/wasm/main.go", "pkg/wasm/bindings.go"}
		for _, file := range wasmFiles {
			filePath := filepath.Join(projectPath, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				g.logger.Debug("WASM file not found",
					zap.String("file", file))
			}
		}
	case "embedded":
		embeddedFiles := []string{"cmd/embedded/main.go"}
		for _, file := range embeddedFiles {
			filePath := filepath.Join(projectPath, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				g.logger.Debug("Embedded file not found",
					zap.String("file", file))
			}
		}
	}

	// Verify required directories exist
	requiredDirs := []string{"cmd", "pkg", "internal"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(projectPath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			g.logger.Warn("Required directory missing",
				zap.String("directory", dir))
		}
	}

	g.logger.Info("TinyGo project post-processing completed",
		zap.String("project", req.Name),
		zap.String("target", g.target),
		zap.Strings("features", req.Features))

	return nil
}

// Validate validates tinygo generator
func (g *TinyGoGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	// TinyGo-specific validation
	if g.stack != "tinygo" {
		return fmt.Errorf("generator stack mismatch: expected 'tinygo', got '%s'", g.stack)
	}

	validTargets := []string{"wasm", "embedded", "microcontroller"}
	if g.target == "" {
		g.target = "wasm" // Default to WASM
	}

	validTarget := false
	for _, t := range validTargets {
		if g.target == t {
			validTarget = true
			break
		}
	}

	if !validTarget {
		return fmt.Errorf("invalid target: %s, valid options: %v", g.target, validTargets)
	}

	return nil
}

// WasmGenerator generates WebAssembly projects using Rust/C++
type WasmGenerator struct {
	*BaseGenerator
	language string // rust, cpp
}

// NewWasmGenerator creates a new WebAssembly generator
func NewWasmGenerator(templateDir string) *WasmGenerator {
	base := NewBaseGenerator("WASM Generator", "wasm", templateDir)
	return &WasmGenerator{
		BaseGenerator: base,
		language:      "rust", // Default language
	}
}

// SetLanguage sets the WASM programming language
func (g *WasmGenerator) SetLanguage(language string) {
	g.language = language
}

// Generate implements Generator interface
func (g *WasmGenerator) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Determine language from config
	if lang, ok := req.Config["language"].(string); ok {
		g.language = lang
	}

	// Call base generator with WASM-specific modifications
	result, err := g.BaseGenerator.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add WASM-specific post-processing
	if err := g.postProcessWasmProject(result.Path, req); err != nil {
		return nil, fmt.Errorf("WASM post-processing failed: %w", err)
	}

	return result, nil
}

// getTemplateFiles returns WASM-specific template files
func (g *WasmGenerator) getTemplateFiles(req GenerateRequest) ([]TemplateFile, error) {
	files := []TemplateFile{
		// Basic project files
		{
			SourcePath: filepath.Join(g.templateDir, "README.md.tmpl"),
			TargetPath: "README.md",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, ".gitignore.tmpl"),
			TargetPath: ".gitignore",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "Makefile.tmpl"),
			TargetPath: "Makefile",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "Dockerfile.tmpl"),
			TargetPath: "Dockerfile",
			IsTemplate: true,
		},
	}

	// Add language-specific files
	switch g.language {
	case "rust":
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "rust", "Cargo.toml.tmpl"),
				TargetPath: "Cargo.toml",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "rust", "src", "lib.rs.tmpl"),
				TargetPath: "src/lib.rs",
				IsTemplate: true,
			},
		}...)
	case "cpp":
		files = append(files, []TemplateFile{
			{
				SourcePath: filepath.Join(g.templateDir, "cpp", "CMakeLists.txt.tmpl"),
				TargetPath: "CMakeLists.txt",
				IsTemplate: true,
			},
			{
				SourcePath: filepath.Join(g.templateDir, "cpp", "src", "main.cpp.tmpl"),
				TargetPath: "src/main.cpp",
				IsTemplate: true,
			},
		}...)
	}

	return files, nil
}

// postProcessWasmProject performs WASM-specific post-processing
func (g *WasmGenerator) postProcessWasmProject(projectPath string, req GenerateRequest) error {
	// Build WASM project
	// Compile to WebAssembly
	// Generate bindings
	// This would involve executing shell commands
	// For now, just return nil as placeholder

	g.logger.Info("WASM project post-processing completed",
		zap.String("project", req.Name),
		zap.String("language", g.language),
		zap.Strings("features", req.Features))

	return nil
}

// Validate validates wasm generator
func (g *WasmGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	// WASM-specific validation
	if g.stack != "wasm" {
		return fmt.Errorf("generator stack mismatch: expected 'wasm', got '%s'", g.stack)
	}

	validLanguages := []string{"rust", "cpp"}
	if g.language == "" {
		g.language = "rust" // Default to Rust
	}

	validLanguage := false
	for _, lang := range validLanguages {
		if g.language == lang {
			validLanguage = true
			break
		}
	}

	if !validLanguage {
		return fmt.Errorf("invalid language: %s, valid options: %v", g.language, validLanguages)
	}

	return nil
}

// MCPGoPremiumGenerator generates premium MCP Go projects
type MCPGoPremiumGenerator struct {
	*BaseGenerator
}

// NewMCPGoPremiumGenerator creates a new MCP Go Premium generator
func NewMCPGoPremiumGenerator(templateDir string) *MCPGoPremiumGenerator {
	base := NewBaseGenerator("MCP Go Premium Generator", "mcp-go-premium", templateDir)
	return &MCPGoPremiumGenerator{
		BaseGenerator: base,
	}
}

// Generate implements Generator interface
func (g *MCPGoPremiumGenerator) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Call base generator with MCP-specific modifications
	result, err := g.BaseGenerator.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add MCP-specific post-processing
	if err := g.postProcessMCPProject(result.Path, req); err != nil {
		return nil, fmt.Errorf("MCP post-processing failed: %w", err)
	}

	return result, nil
}

// getTemplateFiles returns MCP-specific template files
func (g *MCPGoPremiumGenerator) getTemplateFiles(req GenerateRequest) ([]TemplateFile, error) {
	files := []TemplateFile{
		// Basic project files
		{
			SourcePath: filepath.Join(g.templateDir, "Makefile"),
			TargetPath: "Makefile",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "cmd", "main.go.tmpl"),
			TargetPath: "cmd/main.go",
			IsTemplate: true,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "go.mod.tmpl"),
			TargetPath: "go.mod",
			IsTemplate: true,
		},

		// Internal directories with all MCP components
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "ai", "rag", ".gitkeep"),
			TargetPath: "internal/ai/rag/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "ai", "agents", ".gitkeep"),
			TargetPath: "internal/ai/agents/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "ai", "core", ".gitkeep"),
			TargetPath: "internal/ai/core/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "core", "cache", ".gitkeep"),
			TargetPath: "internal/core/cache/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "core", "engine", ".gitkeep"),
			TargetPath: "internal/core/engine/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "monitoring", ".gitkeep"),
			TargetPath: "internal/monitoring/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "interfaces", ".gitkeep"),
			TargetPath: "internal/interfaces/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "infrastructure", ".gitkeep"),
			TargetPath: "internal/infrastructure/.gitkeep",
			IsTemplate: false,
		},
		{
			SourcePath: filepath.Join(g.templateDir, "internal", "state", ".gitkeep"),
			TargetPath: "internal/state/.gitkeep",
			IsTemplate: false,
		},

		// Configuration files
		{
			SourcePath: filepath.Join(g.templateDir, "configs", "dev.yaml.tmpl"),
			TargetPath: "configs/dev.yaml",
			IsTemplate: true,
		},
	}

	return files, nil
}

// hasFeature checks if a feature is enabled
func (g *MCPGoPremiumGenerator) hasFeature(features []string, feature string) bool {
	for _, f := range features {
		if strings.EqualFold(f, feature) {
			return true
		}
	}
	return false
}

// postProcessMCPProject performs MCP-specific post-processing
func (g *MCPGoPremiumGenerator) postProcessMCPProject(projectPath string, req GenerateRequest) error {
	// Initialize MCP modules
	// Set up AI integration
	// Configure monitoring
	// This would involve executing shell commands
	// For now, just return nil as placeholder

	g.logger.Info("MCP Go Premium project post-processing completed",
		zap.String("project", req.Name),
		zap.Strings("features", req.Features))

	return nil
}

// Validate validates MCP Go Premium generator
func (g *MCPGoPremiumGenerator) Validate() error {
	if err := g.BaseGenerator.Validate(); err != nil {
		return err
	}

	// MCP-specific validation
	if g.stack != "mcp-go-premium" {
		return fmt.Errorf("generator stack mismatch: expected 'mcp-go-premium', got '%s'", g.stack)
	}

	return nil
}
