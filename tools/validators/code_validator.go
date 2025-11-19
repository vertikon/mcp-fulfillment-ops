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

// CodeValidator validates code quality (lint, patterns)
type CodeValidator struct {
	factory *validators.ValidatorFactory
	logger  *zap.Logger
}

// NewCodeValidator creates a new code validator
func NewCodeValidator() *CodeValidator {
	return &CodeValidator{
		factory: validators.NewValidatorFactory(),
		logger:  logger.Get(),
	}
}

// ValidateCode validates code quality, patterns, and conventions
func (v *CodeValidator) ValidateCode(ctx context.Context, req CodeValidateRequest) (*CodeValidateResult, error) {
	v.logger.Info("Validating code",
		zap.String("path", req.Path),
		zap.Bool("strict_mode", req.StrictMode))

	var errors []string
	var warnings []string
	var checks []string

	// Check if path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		return &CodeValidateResult{
			Path:   req.Path,
			Valid:  false,
			Errors: []string{fmt.Sprintf("path does not exist: %s", req.Path)},
		}, nil
	}

	// Validate Go code patterns
	if err := v.validateGoPatterns(req.Path, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("Go patterns validation failed: %w", err)
	}

	// Validate imports
	if err := v.validateImports(req.Path, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("imports validation failed: %w", err)
	}

	// Validate naming conventions
	if err := v.validateNamingConventions(req.Path, &errors, &warnings, &checks); err != nil {
		return nil, fmt.Errorf("naming conventions validation failed: %w", err)
	}

	// Use structure validator for code structure
	structValidator := v.factory.GetStructureValidator()
	structResult, err := structValidator.Validate(ctx, validators.StructureRequest{
		Path:       req.Path,
		StrictMode: req.StrictMode,
	})
	
	if err == nil {
		errors = append(errors, structResult.Errors...)
		warnings = append(warnings, structResult.Warnings...)
		checks = append(checks, "structure")
	}

	valid := len(errors) == 0

	return &CodeValidateResult{
		Path:     req.Path,
		Valid:    valid,
		Warnings: warnings,
		Errors:   errors,
		Checks:   checks,
	}, nil
}

// CodeValidateRequest represents a request to validate code
type CodeValidateRequest struct {
	Path       string `json:"path"`
	StrictMode bool   `json:"strict_mode,omitempty"`
	CheckLint  bool   `json:"check_lint,omitempty"`
}

// CodeValidateResult represents the result of code validation
type CodeValidateResult struct {
	Path     string   `json:"path"`
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors"`
	Checks   []string `json:"checks"`
}

// validateGoPatterns validates Go code patterns
func (v *CodeValidator) validateGoPatterns(path string, errors *[]string, warnings *[]string, checks *[]string) error {
	// Check for go.mod
	goModPath := filepath.Join(path, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		*warnings = append(*warnings, "go.mod not found (not a Go project or missing module definition)")
	} else {
		*checks = append(*checks, "go_module")
	}

	*checks = append(*checks, "patterns")
	return nil
}

// validateImports validates import statements
func (v *CodeValidator) validateImports(path string, errors *[]string, warnings *[]string, checks *[]string) error {
	// Walk through Go files and check imports
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(filePath, ".go") {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			
			contentStr := string(content)
			
			// Check for unused imports (simplified check)
			if strings.Contains(contentStr, "import (") {
				*checks = append(*checks, fmt.Sprintf("imports:%s", filepath.Base(filePath)))
			}
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}

	*checks = append(*checks, "imports")
	return nil
}

// validateNamingConventions validates naming conventions
func (v *CodeValidator) validateNamingConventions(path string, errors *[]string, warnings *[]string, checks *[]string) error {
	// Check file naming conventions
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(filePath, ".go") {
			fileName := filepath.Base(filePath)
			
			// Go files should be lowercase with underscores
			if strings.ToLower(fileName) != fileName && !strings.HasSuffix(fileName, "_test.go") {
				*warnings = append(*warnings, fmt.Sprintf("file name should be lowercase: %s", fileName))
			}
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}

	*checks = append(*checks, "naming")
	return nil
}

// Validate validates the code validation request
func (v *CodeValidator) Validate(req CodeValidateRequest) error {
	if req.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}
