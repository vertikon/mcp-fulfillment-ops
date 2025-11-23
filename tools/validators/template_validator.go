package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/validators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// TemplateValidator validates Hulk templates (structure, conventions)
type TemplateValidator struct {
	factory *validators.ValidatorFactory
	logger  *zap.Logger
}

// NewTemplateValidator creates a new template validator
func NewTemplateValidator() *TemplateValidator {
	return &TemplateValidator{
		factory: validators.NewValidatorFactory(),
		logger:  logger.Get(),
	}
}

// ValidateTemplate validates a template structure and conventions
func (v *TemplateValidator) ValidateTemplate(ctx context.Context, req TemplateValidateRequest) (*TemplateValidateResult, error) {
	v.logger.Info("Validating template",
		zap.String("path", req.Path),
		zap.Bool("strict_mode", req.StrictMode))

	var errors []string
	var warnings []string
	var checks []string

	// Check if template path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		return &TemplateValidateResult{
			Path:   req.Path,
			Valid:  false,
			Errors: []string{fmt.Sprintf("template path does not exist: %s", req.Path)},
		}, nil
	}

	// Validate template structure
	if err := v.validateTemplateStructure(req.Path, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("template structure validation failed: %w", err)
	}

	// Validate template files
	if err := v.validateTemplateFiles(req.Path, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("template files validation failed: %w", err)
	}

	// Validate template conventions
	if err := v.validateTemplateConventions(req.Path, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("template conventions validation failed: %w", err)
	}

	valid := len(errors) == 0

	return &TemplateValidateResult{
		Path:     req.Path,
		Valid:    valid,
		Warnings: warnings,
		Errors:   errors,
		Checks:   checks,
	}, nil
}

// TemplateValidateRequest represents a request to validate a template
type TemplateValidateRequest struct {
	Path       string `json:"path"`
	StrictMode bool   `json:"strict_mode,omitempty"`
}

// TemplateValidateResult represents the result of template validation
type TemplateValidateResult struct {
	Path     string   `json:"path"`
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
	Checks   []string `json:"checks"`
}

// validateTemplateStructure validates the template directory structure
func (v *TemplateValidator) validateTemplateStructure(path string, errors *[]string, warnings *[]string, checks *[]string) error {
	// Required files/directories for a template
	required := []string{
		"README.md",
		".yaml",
	}

	for _, req := range required {
		reqPath := filepath.Join(path, req)
		if _, err := os.Stat(reqPath); os.IsNotExist(err) {
			*errors = append(*errors, fmt.Sprintf("required template file/directory missing: %s", req))
		}
	}

	*checks = append(*checks, "structure")
	return nil
}

// validateTemplateFiles validates template files
func (v *TemplateValidator) validateTemplateFiles(path string, errors *[]string, warnings *[]string, checks *[]string) error {
	// Check for template files (.tmpl extension)
	templateFiles := 0
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(filePath, ".tmpl") {
			templateFiles++
		}

		return nil
	})

	if err != nil {
		return err
	}

	if templateFiles == 0 {
		*warnings = append(*warnings, "no template files (.tmpl) found in template directory")
	}

	*checks = append(*checks, "files")
	return nil
}

// validateTemplateConventions validates template naming conventions
func (v *TemplateValidator) validateTemplateConventions(path string, errors *[]string, warnings *[]string, checks *[]string) error {
	// Validate template name (should be lowercase, no spaces)
	templateName := filepath.Base(path)
	if strings.Contains(templateName, " ") {
		*errors = append(*errors, fmt.Sprintf("template name contains spaces: %s", templateName))
	}

	if strings.ToLower(templateName) != templateName {
		*warnings = append(*warnings, fmt.Sprintf("template name should be lowercase: %s", templateName))
	}

	*checks = append(*checks, "conventions")
	return nil
}

// Validate validates the template validation request
func (v *TemplateValidator) Validate(req TemplateValidateRequest) error {
	if req.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}
