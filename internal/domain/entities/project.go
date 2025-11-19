// Package entities provides domain entities
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/value_objects"
)

// Project represents a project entity
type Project struct {
	id          string
	name        string
	description string
	mcpID       string
	stack       value_objects.StackType
	status      ProjectStatus
	createdAt   time.Time
	updatedAt   time.Time
}

// ProjectStatus represents the project status
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusInactive ProjectStatus = "inactive"
	ProjectStatusArchived ProjectStatus = "archived"
)

// NewProject creates a new Project entity
func NewProject(name string, description string, mcpID string, stack value_objects.StackType) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}
	if mcpID == "" {
		return nil, fmt.Errorf("MCP ID cannot be empty")
	}
	if !stack.IsValid() {
		return nil, fmt.Errorf("invalid stack type: %s", stack)
	}

	now := time.Now()
	return &Project{
		id:          uuid.New().String(),
		name:        name,
		description: description,
		mcpID:       mcpID,
		stack:       stack,
		status:      ProjectStatusActive,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ID returns the project ID
func (p *Project) ID() string {
	return p.id
}

// Name returns the project name
func (p *Project) Name() string {
	return p.name
}

// Description returns the project description
func (p *Project) Description() string {
	return p.description
}

// MCPID returns the associated MCP ID
func (p *Project) MCPID() string {
	return p.mcpID
}

// Stack returns the stack type
func (p *Project) Stack() value_objects.StackType {
	return p.stack
}

// Status returns the project status
func (p *Project) Status() ProjectStatus {
	return p.status
}

// CreatedAt returns the creation timestamp
func (p *Project) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp
func (p *Project) UpdatedAt() time.Time {
	return p.updatedAt
}

// SetStatus sets the project status
func (p *Project) SetStatus(status ProjectStatus) error {
	if status != ProjectStatusActive && status != ProjectStatusInactive && status != ProjectStatusArchived {
		return fmt.Errorf("invalid project status: %s", status)
	}
	p.status = status
	p.touch()
	return nil
}

// Activate activates the project
func (p *Project) Activate() {
	p.status = ProjectStatusActive
	p.touch()
}

// Deactivate deactivates the project
func (p *Project) Deactivate() {
	p.status = ProjectStatusInactive
	p.touch()
}

// Archive archives the project
func (p *Project) Archive() {
	p.status = ProjectStatusArchived
	p.touch()
}

// IsActive checks if the project is active
func (p *Project) IsActive() bool {
	return p.status == ProjectStatusActive
}

// touch updates the updatedAt timestamp
func (p *Project) touch() {
	p.updatedAt = time.Now()
}
