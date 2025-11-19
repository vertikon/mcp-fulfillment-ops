package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ValidatorFactory manages creation and lifecycle of validators
type ValidatorFactory struct {
	structureValidator  *StructureValidator
	dependencyValidator *DependencyValidator
	treeValidator      *TreeValidator
	securityValidator  *SecurityValidator
	configValidator   *ConfigValidator
	logger           *zap.Logger
}

// NewValidatorFactory creates a new validator factory
func NewValidatorFactory() *ValidatorFactory {
	factory := &ValidatorFactory{
		logger: logger.Get(),
	}

	// Initialize validators
	factory.structureValidator = NewStructureValidator()
	factory.dependencyValidator = NewDependencyValidator()
	factory.treeValidator = NewTreeValidator()
	factory.securityValidator = NewSecurityValidator()
	factory.configValidator = NewConfigValidator()

	return factory
}

// GetStructureValidator returns the structure validator
func (vf *ValidatorFactory) GetStructureValidator() *StructureValidator {
	return vf.structureValidator
}

// GetDependencyValidator returns the dependency validator
func (vf *ValidatorFactory) GetDependencyValidator() *DependencyValidator {
	return vf.dependencyValidator
}

// GetTreeValidator returns the tree validator
func (vf *ValidatorFactory) GetTreeValidator() *TreeValidator {
	return vf.treeValidator
}

// GetSecurityValidator returns the security validator
func (vf *ValidatorFactory) GetSecurityValidator() *SecurityValidator {
	return vf.securityValidator
}

// GetConfigValidator returns the config validator
func (vf *ValidatorFactory) GetConfigValidator() *ConfigValidator {
	return vf.configValidator
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Path       string    `json:"path"`
	Valid      bool      `json:"valid"`
	Warnings   []string  `json:"warnings"`
	Errors     []string  `json:"errors"`
	Checks     []string  `json:"checks"`
	Duration   time.Duration `json:"duration"`
	ValidatedAt time.Time `json:"validated_at"`
}

// ValidationRequest represents a generic validation request
type ValidationRequest struct {
	Path       string                 `json:"path"`
	StrictMode bool                  `json:"strict_mode"`
	Config     map[string]interface{} `json:"config"`
}

// StructureRequest represents structure validation request
type StructureRequest struct {
	Path       string `json:"path"`
	StrictMode bool   `json:"strict_mode"`
}

// DependencyRequest represents dependency validation request
type DependencyRequest struct {
	Path             string `json:"path"`
	CheckVersions    bool   `json:"check_versions"`
	CheckConflicts  bool   `json:"check_conflicts"`
	CheckLicenses   bool   `json:"check_licenses"`
}

// TreeRequest represents tree validation request
type TreeRequest struct {
	Path       string `json:"path"`
	Depth      int    `json:"depth"`
	FollowSymlinks bool `json:"follow_symlinks"`
}

// SecurityRequest represents security validation request
type SecurityRequest struct {
	Path        string `json:"path"`
	CheckSecrets bool   `json:"check_secrets"`
	CheckPerms  bool   `json:"check_permissions"`
}

// ConfigRequest represents configuration validation request
type ConfigRequest struct {
	Path       string                 `json:"path"`
	ConfigType string                 `json:"config_type"`
	Schema     map[string]interface{} `json:"schema"`
}

// StructureValidator validates project structure
type StructureValidator struct {
	logger *zap.Logger
	rules  []StructureRule
}

// NewStructureValidator creates a new structure validator
func NewStructureValidator() *StructureValidator {
	return &StructureValidator{
		logger: logger.Get(),
		rules:  getDefaultStructureRules(),
	}
}

// StructureRule represents a structure validation rule
type StructureRule struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	Required    bool     `json:"required"`
	Type        string   `json:"type"`
	Pattern     string   `json:"pattern,omitempty"`
	Children    []string `json:"children,omitempty"`
}

// getDefaultStructureRules returns default structure validation rules
func getDefaultStructureRules() []StructureRule {
	return []StructureRule{
		{
			Name:        "go.mod",
			Description: "Go module definition",
			Path:        "go.mod",
			Required:    true,
			Type:        "file",
		},
		{
			Name:        "cmd directory",
			Description: "Command line applications",
			Path:        "cmd",
			Required:    true,
			Type:        "directory",
		},
		{
			Name:        "internal directory",
			Description: "Internal application code",
			Path:        "internal",
			Required:    true,
			Type:        "directory",
			Children:    []string{"domain", "application", "infrastructure"},
		},
		{
			Name:        "pkg directory",
			Description: "Library code that can be used by external applications",
			Path:        "pkg",
			Required:    false,
			Type:        "directory",
		},
		{
			Name:        "configs directory",
			Description: "Configuration files",
			Path:        "configs",
			Required:    true,
			Type:        "directory",
		},
		{
			Name:        "README.md",
			Description: "Project documentation",
			Path:        "README.md",
			Required:    true,
			Type:        "file",
		},
		{
			Name:        ".gitignore",
			Description: "Git ignore file",
			Path:        ".gitignore",
			Required:    true,
			Type:        "file",
		},
		{
			Name:        "Makefile",
			Description: "Build automation",
			Path:        "Makefile",
			Required:    false,
			Type:        "file",
		},
		{
			Name:        "Dockerfile",
			Description: "Docker configuration",
			Path:        "Dockerfile",
			Required:    false,
			Type:        "file",
		},
		{
			Name:        "tests directory",
			Description: "Test files",
			Path:        "tests",
			Required:    true,
			Type:        "directory",
		},
	}
}

// Validate validates the project structure
func (sv *StructureValidator) Validate(ctx context.Context, req StructureRequest) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		Path:       req.Path,
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		Checks:     []string{},
		Duration:   0,
		ValidatedAt: time.Now(),
	}

	sv.logger.Info("Starting structure validation",
		zap.String("path", req.Path),
		zap.Bool("strict_mode", req.StrictMode))

	// Validate each rule
	for _, rule := range sv.rules {
		checkResult := sv.validateRule(req.Path, rule, req.StrictMode)
		result.Checks = append(result.Checks, rule.Name)
		
		if !checkResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, checkResult.Errors...)
		}
		
		if len(checkResult.Warnings) > 0 {
			result.Warnings = append(result.Warnings, checkResult.Warnings...)
		}
	}

	result.Duration = time.Since(startTime)

	sv.logger.Info("Structure validation completed",
		zap.String("path", req.Path),
		zap.Bool("valid", result.Valid),
		zap.Int("errors", len(result.Errors)),
		zap.Int("warnings", len(result.Warnings)),
		zap.Duration("duration", result.Duration))

	return result, nil
}

// RuleValidationResult represents validation result for a single rule
type RuleValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// validateRule validates a single structure rule
func (sv *StructureValidator) validateRule(projectPath string, rule StructureRule, strictMode bool) RuleValidationResult {
	result := RuleValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	fullPath := filepath.Join(projectPath, rule.Path)

	// Check if path exists
	if !pathExists(fullPath) {
		if rule.Required {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Required %s '%s' not found: %s", rule.Type, rule.Name, rule.Path))
		} else {
			if strictMode {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Recommended %s '%s' not found: %s", rule.Type, rule.Name, rule.Path))
			}
		}
		return result
	}

	// Check if path is of correct type
	if isDirectory(fullPath) && rule.Type == "file" {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Path '%s' should be a file but is a directory", rule.Path))
	} else if !isDirectory(fullPath) && rule.Type == "directory" {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Path '%s' should be a directory but is a file", rule.Path))
	}

	// Check children for directories
	if rule.Type == "directory" && len(rule.Children) > 0 {
		for _, child := range rule.Children {
			childPath := filepath.Join(fullPath, child)
			if !pathExists(childPath) {
				if strictMode || rule.Required {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("Required child '%s' not found in directory '%s'", child, rule.Path))
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Recommended child '%s' not found in directory '%s'", child, rule.Path))
				}
			}
		}
	}

	// Check pattern if specified
	if rule.Pattern != "" {
		if !matchesPattern(rule.Path, rule.Pattern) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Path '%s' does not match required pattern '%s'", rule.Path, rule.Pattern))
		}
	}

	return result
}

// DependencyValidator validates project dependencies
type DependencyValidator struct {
	logger *zap.Logger
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator() *DependencyValidator {
	return &DependencyValidator{
		logger: logger.Get(),
	}
}

// Validate validates project dependencies
func (dv *DependencyValidator) Validate(ctx context.Context, req DependencyRequest) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		Path:       req.Path,
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		Checks:     []string{"dependencies"},
		Duration:   0,
		ValidatedAt: time.Now(),
	}

	dv.logger.Info("Starting dependency validation",
		zap.String("path", req.Path))

	// Check for go.mod file
	goModPath := filepath.Join(req.Path, "go.mod")
	if !pathExists(goModPath) {
		result.Errors = append(result.Errors, "go.mod file not found")
		result.Valid = false
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Read and parse go.mod for dependency analysis
	goModContent, err := os.ReadFile(goModPath)
	if err == nil {
		// Basic dependency validation
		content := string(goModContent)
		
		// Check for module declaration
		if !strings.Contains(content, "module ") {
			result.Warnings = append(result.Warnings, "go.mod missing module declaration")
		}
		
		// Check for Go version
		if !strings.Contains(content, "go ") {
			result.Warnings = append(result.Warnings, "go.mod missing Go version")
		}
		
		// Count dependencies (lines starting with require or replace)
		lines := strings.Split(content, "\n")
		depCount := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "require ") || strings.HasPrefix(trimmed, "replace ") {
				depCount++
			}
		}
		
		if depCount > 0 {
			result.Checks = append(result.Checks, fmt.Sprintf("dependencies_found: %d", depCount))
		}
		
		// Check for common problematic patterns
		if strings.Contains(content, "replace") {
			result.Warnings = append(result.Warnings, "go.mod contains replace directives - verify compatibility")
		}
	} else {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to read go.mod: %v", err))
	}

	result.Duration = time.Since(startTime)

	dv.logger.Info("Dependency validation completed",
		zap.String("path", req.Path),
		zap.Bool("valid", result.Valid),
		zap.Duration("duration", result.Duration))

	return result, nil
}

// TreeValidator validates project tree structure
type TreeValidator struct {
	logger *zap.Logger
}

// NewTreeValidator creates a new tree validator
func NewTreeValidator() *TreeValidator {
	return &TreeValidator{
		logger: logger.Get(),
	}
}

// Validate validates project tree structure
func (tv *TreeValidator) Validate(ctx context.Context, req TreeRequest) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		Path:       req.Path,
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		Checks:     []string{"tree_structure"},
		Duration:   0,
		ValidatedAt: time.Now(),
	}

	tv.logger.Info("Starting tree validation",
		zap.String("path", req.Path),
		zap.Int("depth", req.Depth))

	// This would implement actual tree structure validation
	// For now, just return a placeholder result

	result.Duration = time.Since(startTime)

	tv.logger.Info("Tree validation completed",
		zap.String("path", req.Path),
		zap.Bool("valid", result.Valid),
		zap.Duration("duration", result.Duration))

	return result, nil
}

// SecurityValidator validates security aspects
type SecurityValidator struct {
	logger *zap.Logger
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		logger: logger.Get(),
	}
}

// Validate validates security aspects
func (sv *SecurityValidator) Validate(ctx context.Context, req SecurityRequest) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		Path:       req.Path,
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		Checks:     []string{"security"},
		Duration:   0,
		ValidatedAt: time.Now(),
	}

	sv.logger.Info("Starting security validation",
		zap.String("path", req.Path),
		zap.Bool("check_secrets", req.CheckSecrets),
		zap.Bool("check_permissions", req.CheckPerms))

	// Security checks
	if req.CheckSecrets {
		// Check for common secret patterns in files
		secretPatterns := []string{
			"password", "secret", "api_key", "apikey", "token", "credential",
			"private_key", "privatekey", "access_token", "accesstoken",
		}
		
		err := filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue on error
			}
			
			// Skip binary files and directories
			if info.IsDir() || strings.HasSuffix(path, ".exe") || strings.HasSuffix(path, ".bin") {
				return nil
			}
			
			// Skip node_modules, .git, vendor directories
			if strings.Contains(path, "node_modules") || strings.Contains(path, ".git") || strings.Contains(path, "vendor") {
				return nil
			}
			
			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			
			contentStr := strings.ToLower(string(content))
			
			// Check for secret patterns
			for _, pattern := range secretPatterns {
				if strings.Contains(contentStr, pattern) {
					// Check if it's a hardcoded value (not just a variable name)
					if strings.Contains(contentStr, pattern+"=") || strings.Contains(contentStr, pattern+":") {
						result.Warnings = append(result.Warnings, 
							fmt.Sprintf("Potential secret pattern found: %s in %s", pattern, filepath.Base(path)))
					}
				}
			}
			
			return nil
		})
		
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Error scanning for secrets: %v", err))
		}
	}
	
	if req.CheckPerms {
		// Check file permissions (basic check)
		err := filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			
			// Check for world-writable files (security risk)
			mode := info.Mode()
			if mode&0002 != 0 && !info.IsDir() {
				result.Warnings = append(result.Warnings, 
					fmt.Sprintf("World-writable file found: %s", filepath.Base(path)))
			}
			
			return nil
		})
		
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Error checking permissions: %v", err))
		}
	}

	result.Duration = time.Since(startTime)

	sv.logger.Info("Security validation completed",
		zap.String("path", req.Path),
		zap.Bool("valid", result.Valid),
		zap.Duration("duration", result.Duration))

	return result, nil
}

// ConfigValidator validates configuration files
type ConfigValidator struct {
	logger *zap.Logger
}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		logger: logger.Get(),
	}
}

// Validate validates configuration files
func (cv *ConfigValidator) Validate(ctx context.Context, req ConfigRequest) (*ValidationResult, error) {
	startTime := time.Now()
	result := &ValidationResult{
		Path:       req.Path,
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		Checks:     []string{"configuration"},
		Duration:   0,
		ValidatedAt: time.Now(),
	}

	cv.logger.Info("Starting config validation",
		zap.String("path", req.Path),
		zap.String("config_type", req.ConfigType))

	// This would implement actual configuration validation
	// For now, just return a placeholder result

	result.Duration = time.Since(startTime)

	cv.logger.Info("Config validation completed",
		zap.String("path", req.Path),
		zap.Bool("valid", result.Valid),
		zap.Duration("duration", result.Duration))

	return result, nil
}

// Helper functions
func pathExists(path string) bool {
	_, err := filepath.Abs(path)
	return err == nil
}

func isDirectory(path string) bool {
	info, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return filepath.Dir(info) != info
}

func matchesPattern(path, pattern string) bool {
	matched, _ := filepath.Match(pattern, path)
	return matched
}

// ValidateAll runs all validators on a project
func (vf *ValidatorFactory) ValidateAll(ctx context.Context, projectPath string, strictMode bool) (*ValidationResult, error) {
	results := make([]*ValidationResult, 0)
	
	// Run structure validation
	structResult, err := vf.structureValidator.Validate(ctx, StructureRequest{
		Path:       projectPath,
		StrictMode: strictMode,
	})
	if err != nil {
		return nil, fmt.Errorf("structure validation failed: %w", err)
	}
	results = append(results, structResult)
	
	// Run dependency validation
	depResult, err := vf.dependencyValidator.Validate(ctx, DependencyRequest{
		Path:           projectPath,
		CheckVersions:   true,
		CheckConflicts:  true,
		CheckLicenses:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}
	results = append(results, depResult)
	
	// Run tree validation
	treeResult, err := vf.treeValidator.Validate(ctx, TreeRequest{
		Path:           projectPath,
		Depth:          10,
		FollowSymlinks:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("tree validation failed: %w", err)
	}
	results = append(results, treeResult)
	
	// Run security validation
	secResult, err := vf.securityValidator.Validate(ctx, SecurityRequest{
		Path:        projectPath,
		CheckSecrets: true,
		CheckPerms:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}
	results = append(results, secResult)
	
	// Combine results
	combined := &ValidationResult{
		Path:       projectPath,
		Valid:      true,
		Warnings:   []string{},
		Errors:     []string{},
		Checks:     []string{},
		Duration:   0,
		ValidatedAt: time.Now(),
	}
	
	for _, result := range results {
		if !result.Valid {
			combined.Valid = false
		}
		combined.Warnings = append(combined.Warnings, result.Warnings...)
		combined.Errors = append(combined.Errors, result.Errors...)
		combined.Checks = append(combined.Checks, result.Checks...)
		combined.Duration += result.Duration
	}
	
	return combined, nil
}