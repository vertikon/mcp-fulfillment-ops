package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// TemplateValidator validates template structure and content
type TemplateValidator struct {
	logger *zap.Logger
}

// NewTemplateValidator creates a new template validator
func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{
		logger: logger.GetLogger(),
	}
}

// ValidationResult represents the result of template validation
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// ValidateTemplate validates a complete template directory
func (tv *TemplateValidator) ValidateTemplate(templatePath string) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check if directory exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		result.AddError(fmt.Sprintf("Template directory does not exist: %s", templatePath))
		return result
	}

	// Validate manifest.yaml
	manifestPath := filepath.Join(templatePath, "manifest.yaml")
	tv.validateManifest(manifestPath, result)

	// If manifest is invalid, stop here
	if len(result.Errors) > 0 {
		result.Valid = false
		return result
	}

	// Load manifest to get file list
	manifest, err := tv.loadManifest(manifestPath)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to load manifest: %v", err))
		result.Valid = false
		return result
	}

	// Validate template files
	tv.validateTemplateFiles(templatePath, manifest, result)

	// Check for required files
	tv.validateRequiredFiles(templatePath, manifest, result)

	// Validate placeholders in template files
	tv.validatePlaceholders(templatePath, manifest, result)

	result.Valid = len(result.Errors) == 0
	return result
}

// validateManifest validates the manifest.yaml file
func (tv *TemplateValidator) validateManifest(manifestPath string, result *ValidationResult) {
	// Check if manifest exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		result.AddError("manifest.yaml is required")
		return
	}

	// Try to parse YAML
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		result.AddError(fmt.Sprintf("Failed to read manifest.yaml: %v", err))
		return
	}

	var manifest map[string]interface{}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		result.AddError(fmt.Sprintf("Invalid YAML in manifest.yaml: %v", err))
		return
	}

	// Validate required fields
	requiredFields := []string{"name", "stack", "version"}
	for _, field := range requiredFields {
		if _, exists := manifest[field]; !exists {
			result.AddError(fmt.Sprintf("Required field '%s' missing in manifest.yaml", field))
		}
	}

	// Validate name format
	if name, ok := manifest["name"].(string); ok {
		if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(name) {
			result.AddError("Template name must contain only lowercase letters, numbers, and hyphens")
		}
	}

	// Validate version format
	if version, ok := manifest["version"].(string); ok {
		if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version) {
			result.AddWarning("Version should follow semantic versioning (x.y.z)")
		}
	}
}

// loadManifest loads and parses the manifest.yaml file
func (tv *TemplateValidator) loadManifest(manifestPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest map[string]interface{}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// validateTemplateFiles validates that all files listed in manifest exist
func (tv *TemplateValidator) validateTemplateFiles(templatePath string, manifest map[string]interface{}, result *ValidationResult) {
	files, ok := manifest["files"].([]interface{})
	if !ok {
		result.AddError("Files list is required in manifest.yaml")
		return
	}

	for _, file := range files {
		filename, ok := file.(string)
		if !ok {
			result.AddError("All files in manifest must be strings")
			continue
		}

		fullPath := filepath.Join(templatePath, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			result.AddError(fmt.Sprintf("File listed in manifest does not exist: %s", filename))
		}
	}
}

// validateRequiredFiles checks for required files based on stack type
func (tv *TemplateValidator) validateRequiredFiles(templatePath string, manifest map[string]interface{}, result *ValidationResult) {
	stack, _ := manifest["stack"].(string)

	switch stack {
	case "go", "go-premium":
		requiredFiles := []string{"go.mod.tmpl"}
		for _, file := range requiredFiles {
			if _, err := os.Stat(filepath.Join(templatePath, file)); os.IsNotExist(err) {
				result.AddError(fmt.Sprintf("Go templates must include: %s", file))
			}
		}

	case "web":
		requiredFiles := []string{"package.json.tmpl"}
		for _, file := range requiredFiles {
			if _, err := os.Stat(filepath.Join(templatePath, file)); os.IsNotExist(err) {
				result.AddError(fmt.Sprintf("Web templates must include: %s", file))
			}
		}

	case "wasm":
		requiredFiles := []string{"Cargo.toml.tmpl"}
		for _, file := range requiredFiles {
			if _, err := os.Stat(filepath.Join(templatePath, file)); os.IsNotExist(err) {
				result.AddError(fmt.Sprintf("WASM templates must include: %s", file))
			}
		}

	case "tinygo":
		requiredFiles := []string{"go.mod.tmpl"}
		for _, file := range requiredFiles {
			if _, err := os.Stat(filepath.Join(templatePath, file)); os.IsNotExist(err) {
				result.AddError(fmt.Sprintf("TinyGo templates must include: %s", file))
			}
		}
	}

	// All templates should have README
	if _, err := os.Stat(filepath.Join(templatePath, "README.md.tmpl")); os.IsNotExist(err) {
		result.AddWarning("Templates should include README.md.tmpl")
	}
}

// validatePlaceholders checks for valid placeholder usage in template files
func (tv *TemplateValidator) validatePlaceholders(templatePath string, manifest map[string]interface{}, result *ValidationResult) {
	placeholders, ok := manifest["placeholders"].([]interface{})
	if !ok {
		result.AddWarning("No placeholders defined in manifest")
		return
	}

	placeholderSet := make(map[string]bool)
	for _, ph := range placeholders {
		if name, ok := ph.(string); ok {
			placeholderSet[name] = true
		}
	}

	// Define required placeholders for all templates
	requiredPlaceholders := []string{"Name", "Version"}
	for _, req := range requiredPlaceholders {
		if !placeholderSet[req] {
			result.AddError(fmt.Sprintf("Required placeholder '%s' is missing", req))
		}
	}

	// Scan template files for placeholder usage
	err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".tmpl") {
			tv.validateFilePlaceholders(path, placeholderSet, result)
		}

		return nil
	})

	if err != nil {
		result.AddWarning(fmt.Sprintf("Failed to scan templates for placeholders: %v", err))
	}
}

// validateFilePlaceholders checks placeholder usage in a single template file
func (tv *TemplateValidator) validateFilePlaceholders(filePath string, allowedPlaceholders map[string]bool, result *ValidationResult) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	content := string(data)
	
	// Find all placeholders using regex pattern
	placeholderRegex := regexp.MustCompile(`\{\{\s*\.\s*([A-Za-z][A-Za-z0-9_]*)\s*\}\}`)
	matches := placeholderRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			placeholder := match[1]
			if !allowedPlaceholders[placeholder] {
				result.AddWarning(fmt.Sprintf("Undeclared placeholder '%s' found in %s", placeholder, filepath.Base(filePath)))
			}
		}
	}
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(error string) {
	vr.Errors = append(vr.Errors, error)
}

// AddWarning adds a warning to the validation result
func (vr *ValidationResult) AddWarning(warning string) {
	vr.Warnings = append(vr.Warnings, warning)
}

// ValidateAllTemplates validates all templates in the base directory
func (tv *TemplateValidator) ValidateAllTemplates(basePath string) map[string]*ValidationResult {
	results := make(map[string]*ValidationResult)

	// Find all template directories (those with manifest.yaml)
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.Name() == "manifest.yaml" {
			templateDir := filepath.Dir(path)
			templateName := filepath.Base(templateDir)
			
			results[templateName] = tv.ValidateTemplate(templateDir)
		}

		return nil
	})

	if err != nil {
		tv.logger.Error("Failed to walk templates directory", zap.Error(err))
		// Add a general error result
		results["general"] = &ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("Failed to scan templates: %v", err)},
			Warnings: []string{},
		}
	}

	return results
}