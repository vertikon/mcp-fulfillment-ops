package validators

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/mcp/validators"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MCPValidator validates MCP structure and configuration
type MCPValidator struct {
	factory *validators.ValidatorFactory
	logger  *zap.Logger
}

// NewMCPValidator creates a new MCP validator
func NewMCPValidator() *MCPValidator {
	return &MCPValidator{
		factory: validators.NewValidatorFactory(),
		logger:  logger.Get(),
	}
}

// ValidateMCP validates an MCP project structure and configuration
func (v *MCPValidator) ValidateMCP(ctx context.Context, req MCPValidateRequest) (*MCPValidateResult, error) {
	v.logger.Info("Validating MCP",
		zap.String("path", req.Path),
		zap.Bool("strict_mode", req.StrictMode))

	// Use structure validator from factory
	structValidator := v.factory.GetStructureValidator()

	// Validate structure
	structResult, err := structValidator.Validate(ctx, validators.StructureRequest{
		Path:       req.Path,
		StrictMode: req.StrictMode,
	})

	if err != nil {
		return nil, fmt.Errorf("MCP validation failed: %w", err)
	}

	// Use security validator if requested
	if req.CheckSecurity {
		secValidator := v.factory.GetSecurityValidator()
		secResult, err := secValidator.Validate(ctx, validators.SecurityRequest{
			Path:         req.Path,
			CheckSecrets: true,
			CheckPerms:   true,
		})

		if err == nil {
			// Merge security errors and warnings
			structResult.Errors = append(structResult.Errors, secResult.Errors...)
			structResult.Warnings = append(structResult.Warnings, secResult.Warnings...)
			structResult.Checks = append(structResult.Checks, "security")
		}
	}

	// Use dependency validator if requested
	if req.CheckDependencies {
		depValidator := v.factory.GetDependencyValidator()
		depResult, err := depValidator.Validate(ctx, validators.DependencyRequest{
			Path:           req.Path,
			CheckVersions:  true,
			CheckConflicts: true,
			CheckLicenses:  false,
		})

		if err == nil {
			// Merge dependency errors and warnings
			structResult.Errors = append(structResult.Errors, depResult.Errors...)
			structResult.Warnings = append(structResult.Warnings, depResult.Warnings...)
			structResult.Checks = append(structResult.Checks, "dependencies")
		}
	}

	// Convert to external format
	result := &MCPValidateResult{
		Path:        structResult.Path,
		Valid:       structResult.Valid,
		Warnings:    structResult.Warnings,
		Errors:      structResult.Errors,
		Checks:      structResult.Checks,
		Duration:    structResult.Duration.String(),
		ValidatedAt: structResult.ValidatedAt,
	}

	// Determine overall validity
	if len(structResult.Errors) > 0 {
		result.Valid = false
	}

	return result, nil
}

// MCPValidateRequest represents a request to validate an MCP
type MCPValidateRequest struct {
	Path              string `json:"path"`
	StrictMode        bool   `json:"strict_mode,omitempty"`
	CheckSecurity     bool   `json:"check_security,omitempty"`
	CheckDependencies bool   `json:"check_dependencies,omitempty"`
}

// MCPValidateResult represents the result of MCP validation
type MCPValidateResult struct {
	Path        string      `json:"path"`
	Valid       bool        `json:"valid"`
	Warnings    []string    `json:"warnings"`
	Errors      []string    `json:"errors"`
	Checks      []string    `json:"checks"`
	Duration    string      `json:"duration"`
	ValidatedAt interface{} `json:"validated_at"`
}

// Validate validates the MCP validation request
func (v *MCPValidator) Validate(req MCPValidateRequest) error {
	if req.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// GetFactory returns the internal validator factory
func (v *MCPValidator) GetFactory() *validators.ValidatorFactory {
	return v.factory
}
