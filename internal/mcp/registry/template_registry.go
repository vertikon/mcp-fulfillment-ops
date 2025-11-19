package registry

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// TemplateInfo holds information about a registered template
type TemplateInfo struct {
	Name         string            `json:"name"`
	Stack        string            `json:"stack"`
	Version      string            `json:"version"`
	Summary      string            `json:"summary"`
	Placeholders []string          `json:"placeholders"`
	Files        []string          `json:"files"`
	Path         string            `json:"path"`
	Metadata     map[string]string `json:"metadata"`
	LastModified string            `json:"last_modified"`
}

// TemplateRegistry manages registration and discovery of templates
type TemplateRegistry struct {
	templates map[string]*TemplateInfo
	mu        sync.RWMutex
	logger    *zap.Logger
	basePath  string
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry(basePath string) *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*TemplateInfo),
		basePath:  basePath,
		logger:    logger.GetLogger(),
	}
}

// LoadTemplates discovers and loads all templates from the base path
func (tr *TemplateRegistry) LoadTemplates(ctx context.Context) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.logger.Info("Loading templates from", zap.String("path", tr.basePath))

	// Walk through the templates directory
	err := filepath.WalkDir(tr.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for manifest.yaml files
		if d.Name() == "manifest.yaml" {
			templateDir := filepath.Dir(path)
			relPath, err := filepath.Rel(tr.basePath, templateDir)
			if err != nil {
				return err
			}

			templateInfo, err := tr.loadTemplateFromManifest(path)
			if err != nil {
				tr.logger.Error("Failed to load template", 
					zap.String("path", path), 
					zap.Error(err))
				return nil // Continue loading other templates
			}

			templateInfo.Path = relPath
			tr.templates[templateInfo.Name] = templateInfo
			tr.logger.Info("Loaded template", 
				zap.String("name", templateInfo.Name),
				zap.String("stack", templateInfo.Stack))
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk templates directory: %w", err)
	}

	tr.logger.Info("Templates loaded successfully", zap.Int("count", len(tr.templates)))
	return nil
}

// loadTemplateFromManifest loads template information from manifest.yaml
func (tr *TemplateRegistry) loadTemplateFromManifest(manifestPath string) (*TemplateInfo, error) {
	data, err := fs.ReadFile(nil, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest struct {
		Name         string   `yaml:"name"`
		Stack        string   `yaml:"stack"`
		Version      string   `yaml:"version"`
		Summary      string   `yaml:"summary"`
		Placeholders []string `yaml:"placeholders"`
		Files        []string `yaml:"files"`
	}

	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if manifest.Stack == "" {
		return nil, fmt.Errorf("template stack is required")
	}

	templateInfo := &TemplateInfo{
		Name:         manifest.Name,
		Stack:        manifest.Stack,
		Version:      manifest.Version,
		Summary:      manifest.Summary,
		Placeholders: manifest.Placeholders,
		Files:        manifest.Files,
		Metadata:     make(map[string]string),
	}

	return templateInfo, nil
}

// GetTemplate returns a template by name
func (tr *TemplateRegistry) GetTemplate(name string) (*TemplateInfo, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	template, exists := tr.templates[name]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", name)
	}

	return template, nil
}

// ListTemplates returns all registered templates
func (tr *TemplateRegistry) ListTemplates() []*TemplateInfo {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	templates := make([]*TemplateInfo, 0, len(tr.templates))
	for _, template := range tr.templates {
		templates = append(templates, template)
	}

	return templates
}

// ListTemplatesByStack returns templates filtered by stack
func (tr *TemplateRegistry) ListTemplatesByStack(stack string) []*TemplateInfo {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var templates []*TemplateInfo
	for _, template := range tr.templates {
		if template.Stack == stack {
			templates = append(templates, template)
		}
	}

	return templates
}

// GetAvailableStacks returns all available stacks
func (tr *TemplateRegistry) GetAvailableStacks() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	stacks := make(map[string]bool)
	for _, template := range tr.templates {
		stacks[template.Stack] = true
	}

	result := make([]string, 0, len(stacks))
	for stack := range stacks {
		result = append(result, stack)
	}

	return result
}

// SearchTemplates searches templates by name or summary
func (tr *TemplateRegistry) SearchTemplates(query string) []*TemplateInfo {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*TemplateInfo

	for _, template := range tr.templates {
		if strings.Contains(strings.ToLower(template.Name), query) ||
			strings.Contains(strings.ToLower(template.Summary), query) {
			results = append(results, template)
		}
	}

	return results
}

// ValidateTemplate checks if a template meets minimum requirements
func (tr *TemplateRegistry) ValidateTemplate(templateInfo *TemplateInfo) error {
	if templateInfo.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if templateInfo.Stack == "" {
		return fmt.Errorf("template stack is required")
	}
	if len(templateInfo.Files) == 0 {
		return fmt.Errorf("template must have at least one file")
	}

	// Check for required placeholders
	requiredPlaceholders := []string{"Name", "Version"}
	for _, required := range requiredPlaceholders {
		found := false
		for _, placeholder := range templateInfo.Placeholders {
			if placeholder == required {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("required placeholder '%s' is missing", required)
		}
	}

	return nil
}

// RegisterTemplate manually registers a template
func (tr *TemplateRegistry) RegisterTemplate(templateInfo *TemplateInfo) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if err := tr.ValidateTemplate(templateInfo); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	tr.templates[templateInfo.Name] = templateInfo
	tr.logger.Info("Template registered", 
		zap.String("name", templateInfo.Name),
		zap.String("stack", templateInfo.Stack))

	return nil
}

// UnregisterTemplate removes a template from the registry
func (tr *TemplateRegistry) UnregisterTemplate(name string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.templates[name]; !exists {
		return fmt.Errorf("template '%s' not found", name)
	}

	delete(tr.templates, name)
	tr.logger.Info("Template unregistered", zap.String("name", name))
	return nil
}

// GetTemplatePath returns the full path to a template directory
func (tr *TemplateRegistry) GetTemplatePath(name string) (string, error) {
	template, err := tr.GetTemplate(name)
	if err != nil {
		return "", err
	}

	return filepath.Join(tr.basePath, template.Path), nil
}