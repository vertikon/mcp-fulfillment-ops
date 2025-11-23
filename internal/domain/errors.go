// Package entities provides domain errors
package entities

import "fmt"

// DomainError represents a domain-level error
type DomainError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error
func NewDomainError(code string, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common domain error codes
const (
	ErrCodeInvalidInput       = "INVALID_INPUT"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeAlreadyExists      = "ALREADY_EXISTS"
	ErrCodeInvalidState       = "INVALID_STATE"
	ErrCodeBusinessRule       = "BUSINESS_RULE"
	ErrCodeInvariantViolation = "INVARIANT_VIOLATION"
)

// Common domain errors
var (
	ErrMCPNotFound            = NewDomainError(ErrCodeNotFound, "MCP not found", nil)
	ErrKnowledgeNotFound      = NewDomainError(ErrCodeNotFound, "Knowledge not found", nil)
	ErrProjectNotFound        = NewDomainError(ErrCodeNotFound, "Project not found", nil)
	ErrTemplateNotFound       = NewDomainError(ErrCodeNotFound, "Template not found", nil)
	ErrMCPAlreadyExists       = NewDomainError(ErrCodeAlreadyExists, "MCP already exists", nil)
	ErrKnowledgeAlreadyExists = NewDomainError(ErrCodeAlreadyExists, "Knowledge already exists", nil)
	ErrInvalidStackType       = NewDomainError(ErrCodeInvalidInput, "Invalid stack type", nil)
	ErrInvalidFeature         = NewDomainError(ErrCodeInvalidInput, "Invalid feature", nil)
)
