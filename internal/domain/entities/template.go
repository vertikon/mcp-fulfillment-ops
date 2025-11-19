// Package entities provides domain entities
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/value_objects"
)

// Template represents a template entity
type Template struct {
	id          string
	name        string
	description string
	stack       value_objects.StackType
	content     string
	variables   []string
	version     int
	createdAt   time.Time
	updatedAt   time.Time
}

// NewTemplate creates a new Template entity
func NewTemplate(name string, description string, stack value_objects.StackType, content string) (*Template, error) {
	if name == "" {
		return nil, fmt.Errorf("template name cannot be empty")
	}
	if !stack.IsValid() {
		return nil, fmt.Errorf("invalid stack type: %s", stack)
	}
	if content == "" {
		return nil, fmt.Errorf("template content cannot be empty")
	}

	now := time.Now()
	return &Template{
		id:          uuid.New().String(),
		name:        name,
		description: description,
		stack:       stack,
		content:     content,
		variables:   make([]string, 0),
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ID returns the template ID
func (t *Template) ID() string {
	return t.id
}

// Name returns the template name
func (t *Template) Name() string {
	return t.name
}

// Description returns the template description
func (t *Template) Description() string {
	return t.description
}

// Stack returns the stack type
func (t *Template) Stack() value_objects.StackType {
	return t.stack
}

// Content returns the template content
func (t *Template) Content() string {
	return t.content
}

// Variables returns a copy of the variables list
func (t *Template) Variables() []string {
	variables := make([]string, len(t.variables))
	copy(variables, t.variables)
	return variables
}

// Version returns the template version
func (t *Template) Version() int {
	return t.version
}

// CreatedAt returns the creation timestamp
func (t *Template) CreatedAt() time.Time {
	return t.createdAt
}

// UpdatedAt returns the last update timestamp
func (t *Template) UpdatedAt() time.Time {
	return t.updatedAt
}

// SetContent sets the template content
func (t *Template) SetContent(content string) error {
	if content == "" {
		return fmt.Errorf("template content cannot be empty")
	}
	t.content = content
	t.touch()
	return nil
}

// AddVariable adds a variable to the template
func (t *Template) AddVariable(variable string) error {
	if variable == "" {
		return fmt.Errorf("variable name cannot be empty")
	}

	// Check for duplicates
	for _, v := range t.variables {
		if v == variable {
			return fmt.Errorf("variable %s already exists", variable)
		}
	}

	t.variables = append(t.variables, variable)
	t.touch()
	return nil
}

// RemoveVariable removes a variable from the template
func (t *Template) RemoveVariable(variable string) error {
	for i, v := range t.variables {
		if v == variable {
			t.variables = append(t.variables[:i], t.variables[i+1:]...)
			t.touch()
			return nil
		}
	}
	return fmt.Errorf("variable %s not found", variable)
}

// IncrementVersion increments the template version
func (t *Template) IncrementVersion() {
	t.version++
	t.touch()
}

// touch updates the updatedAt timestamp
func (t *Template) touch() {
	t.updatedAt = time.Now()
}
