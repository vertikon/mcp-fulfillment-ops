// Package services provides domain services
package services

import (
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// TemplateDomainService provides domain logic for Template operations
type TemplateDomainService struct{}

// NewTemplateDomainService creates a new Template domain service
func NewTemplateDomainService() *TemplateDomainService {
	return &TemplateDomainService{}
}

// ValidateTemplate validates Template business rules
func (s *TemplateDomainService) ValidateTemplate(template *entities.Template) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	if template.Name() == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	if template.Content() == "" {
		return fmt.Errorf("template content cannot be empty")
	}

	if !template.Stack().IsValid() {
		return fmt.Errorf("template stack must be valid")
	}

	return nil
}

// CanAddVariable checks if a variable can be added to template
func (s *TemplateDomainService) CanAddVariable(template *entities.Template, variable string) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}
	if variable == "" {
		return fmt.Errorf("variable name cannot be empty")
	}

	// Check for duplicates
	variables := template.Variables()
	for _, v := range variables {
		if v == variable {
			return fmt.Errorf("variable %s already exists", variable)
		}
	}

	return nil
}

// ShouldIncrementVersion determines if template version should be incremented
func (s *TemplateDomainService) ShouldIncrementVersion(template *entities.Template, hasContentChanges bool) bool {
	if template == nil {
		return false
	}

	// Business rule: Increment version on content changes
	return hasContentChanges
}
