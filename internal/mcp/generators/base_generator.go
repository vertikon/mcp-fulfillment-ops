package generators

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// BaseGenerator provides common functionality for all generators
type BaseGenerator struct {
	name        string
	stack       string
	templateDir string
	logger      *zap.Logger
	funcMap     template.FuncMap
}

// NewBaseGenerator creates a new base generator
func NewBaseGenerator(name, stack, templateDir string) *BaseGenerator {
	return &BaseGenerator{
		name:        name,
		stack:       stack,
		templateDir: templateDir,
		logger:      logger.Get(),
		funcMap:     createTemplateFuncMap(),
	}
}

// GenerateRequest represents a generation request
type GenerateRequest struct {
	Name     string                 `json:"name"`
	Path     string                 `json:"path"`
	Stack    string                 `json:"stack,omitempty"` // Stack type (go, tinygo, wasm, etc.)
	Features []string               `json:"features,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// GenerateResult represents the result of a generation operation
type GenerateResult struct {
	Path         string            `json:"path"`
	FilesCreated []string          `json:"files_created"`
	CreatedAt    time.Time         `json:"created_at"`
	Duration     time.Duration     `json:"duration"`
	Size         int64             `json:"size"`
	Metadata     map[string]string `json:"metadata"`
}

// TemplateFile represents a template file with its target path
type TemplateFile struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	IsTemplate bool   `json:"is_template"`
}

// Generate implements the core generation logic
func (g *BaseGenerator) Generate(ctx interface{}, req GenerateRequest) (*GenerateResult, error) {
	// Type assert to context.Context if possible
	var contextCtx context.Context
	if ctxValue, ok := ctx.(context.Context); ok {
		contextCtx = ctxValue
	} else {
		contextCtx = context.Background()
	}
	startTime := time.Now()

	g.logger.Info("Starting generation",
		zap.String("generator", g.name),
		zap.String("stack", g.stack),
		zap.String("project", req.Name),
		zap.String("path", req.Path))

	// Validate request
	if err := g.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create project structure
	projectPath := filepath.Join(req.Path, req.Name)
	if err := g.createProjectStructure(projectPath); err != nil {
		return nil, fmt.Errorf("failed to create project structure: %w", err)
	}

	// Get template files
	templateFiles, err := g.getTemplateFiles(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get template files: %w", err)
	}

	// Process each template file
	filesCreated := make([]string, 0, len(templateFiles))
	var totalSize int64

	for _, templateFile := range templateFiles {
		// Check context cancellation
		select {
		case <-contextCtx.Done():
			return nil, contextCtx.Err()
		default:
		}

		targetPath := filepath.Join(projectPath, templateFile.TargetPath)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", filepath.Dir(targetPath), err)
		}

		if templateFile.IsTemplate {
			// Process template
			content, size, err := g.processTemplate(templateFile.SourcePath, req)
			if err != nil {
				return nil, fmt.Errorf("failed to process template %s: %w", templateFile.SourcePath, err)
			}

			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return nil, fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}

			totalSize += size
		} else {
			// Copy file as-is
			if err := g.copyFile(templateFile.SourcePath, targetPath); err != nil {
				return nil, fmt.Errorf("failed to copy file %s: %w", templateFile.SourcePath, err)
			}

			if info, err := os.Stat(targetPath); err == nil {
				totalSize += info.Size()
			}
		}

		filesCreated = append(filesCreated, targetPath)
		g.logger.Debug("Created file", zap.String("path", targetPath))
	}

	duration := time.Since(startTime)

	result := &GenerateResult{
		Path:         projectPath,
		FilesCreated: filesCreated,
		CreatedAt:    time.Now(),
		Duration:     duration,
		Size:         totalSize,
		Metadata: map[string]string{
			"generator": g.name,
			"stack":     g.stack,
			"project":   req.Name,
		},
	}

	g.logger.Info("Generation completed",
		zap.String("project", req.Name),
		zap.String("path", result.Path),
		zap.Int("files_created", len(filesCreated)),
		zap.Duration("duration", duration),
		zap.Int64("size", totalSize))

	return result, nil
}

// validateRequest validates the generation request
func (g *BaseGenerator) validateRequest(req GenerateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if req.Path == "" {
		return fmt.Errorf("output path is required")
	}

	// Validate project name (no special characters, no spaces)
	if !isValidProjectName(req.Name) {
		return fmt.Errorf("invalid project name: %s", req.Name)
	}

	// Check if output directory exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		return fmt.Errorf("output path does not exist: %s", req.Path)
	}

	// Check if project already exists
	projectPath := filepath.Join(req.Path, req.Name)
	if _, err := os.Stat(projectPath); err == nil {
		return fmt.Errorf("project already exists: %s", projectPath)
	}

	return nil
}

// createProjectStructure creates the basic project directory structure
func (g *BaseGenerator) createProjectStructure(projectPath string) error {
	// Standard directory structure for all projects
	directories := []string{
		"cmd",
		"internal",
		"pkg",
		"configs",
		"scripts",
		"docs",
		"examples",
		"tests",
	}

	for _, dir := range directories {
		fullPath := filepath.Join(projectPath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

// getTemplateFiles returns the list of template files to process
func (g *BaseGenerator) getTemplateFiles(req GenerateRequest) ([]TemplateFile, error) {
	// This should be implemented by specific generators
	// For now, return a basic set of files
	files := []TemplateFile{
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
	}

	return files, nil
}

// processTemplate processes a template file with the given request
func (g *BaseGenerator) processTemplate(templatePath string, req GenerateRequest) ([]byte, int64, error) {
	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Create template
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(g.funcMap).Parse(string(templateContent))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Prepare template data
	data := g.prepareTemplateData(req)

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, 0, fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	content := buf.Bytes()
	return content, int64(len(content)), nil
}

// prepareTemplateData prepares the data for template execution
func (g *BaseGenerator) prepareTemplateData(req GenerateRequest) map[string]interface{} {
	return map[string]interface{}{
		"ProjectName": req.Name,
		"ProjectPath": req.Path,
		"Stack":       g.stack,
		"Features":    req.Features,
		"Config":      req.Config,
		"Generator":   g.name,
		"Timestamp":   time.Now(),
		"Year":        time.Now().Year(),
		"FeaturesMap": g.createFeaturesMap(req.Features),
		"ConfigMap":   req.Config,
	}
}

// createFeaturesMap creates a map from features list for easier template access
func (g *BaseGenerator) createFeaturesMap(features []string) map[string]bool {
	featureMap := make(map[string]bool)
	for _, feature := range features {
		featureMap[feature] = true
	}
	return featureMap
}

// copyFile copies a file from source to destination
func (g *BaseGenerator) copyFile(src, dst string) error {
	source, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, source, 0644)
}

// isValidProjectName validates the project name
func isValidProjectName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// createTemplateFuncMap creates the template function map
func createTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		// String operations
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,

		// String transformations
		"snakeCase":  toSnakeCase,
		"camelCase":  toCamelCase,
		"pascalCase": toPascalCase,
		"kebabCase":  toKebabCase,

		// Boolean helpers
		"hasFeature": func(features []string, feature string) bool {
			for _, f := range features {
				if f == feature {
					return true
				}
			}
			return false
		},

		// Array helpers
		"join": strings.Join,
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},

		// Config helpers
		"getConfig": func(config map[string]interface{}, key string, defaultValue interface{}) interface{} {
			if val, ok := config[key]; ok {
				return val
			}
			return defaultValue
		},

		// Time helpers
		"now": time.Now,
		"formatTime": func(t time.Time, layout string) string {
			if layout == "" {
				layout = "2006-01-02 15:04:05"
			}
			return t.Format(layout)
		},

		// Path helpers
		"base": filepath.Base,
		"dir":  filepath.Dir,
		"ext":  filepath.Ext,

		// Conditional helpers
		"ternary": func(condition bool, trueValue, falseValue interface{}) interface{} {
			if condition {
				return trueValue
			}
			return falseValue
		},

		// Default value helpers
		"default": func(value, defaultValue interface{}) interface{} {
			if value == nil || value == "" {
				return defaultValue
			}
			return value
		},
	}
}

// String transformation functions
func toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(s, "-", "_"),
			" ", "_"),
		"__", "_"))
}

func toCamelCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])
	for _, word := range words[1:] {
		result += strings.Title(strings.ToLower(word))
	}
	return result
}

func toPascalCase(s string) string {
	words := strings.Fields(strings.ReplaceAll(s, "_", " "))
	if len(words) == 0 {
		return ""
	}

	result := ""
	for _, word := range words {
		result += strings.Title(strings.ToLower(word))
	}
	return result
}

func toKebabCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(
		strings.ReplaceAll(s, "_", "-"),
		"--", "-"))
}

// GetStack returns the generator's stack
func (g *BaseGenerator) GetStack() string {
	return g.stack
}

// GetName returns the generator's name
func (g *BaseGenerator) GetName() string {
	return g.name
}

// Validate validates the generator configuration
func (g *BaseGenerator) Validate() error {
	if g.name == "" {
		return fmt.Errorf("generator name is required")
	}

	if g.stack == "" {
		return fmt.Errorf("generator stack is required")
	}

	if g.templateDir == "" {
		return fmt.Errorf("template directory is required")
	}

	// Check if template directory exists
	if _, err := os.Stat(g.templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template directory does not exist: %s", g.templateDir)
	}

	return nil
}

// GetTemplateFilesInfo returns information about available templates
func (g *BaseGenerator) GetTemplateFilesInfo() ([]map[string]interface{}, error) {
	var templates []map[string]interface{}

	err := filepath.Walk(g.templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tmpl") {
			relPath, err := filepath.Rel(g.templateDir, path)
			if err != nil {
				return err
			}

			templates = append(templates, map[string]interface{}{
				"name":     filepath.Base(path),
				"path":     relPath,
				"size":     info.Size(),
				"modified": info.ModTime(),
			})
		}

		return nil
	})

	return templates, err
}
