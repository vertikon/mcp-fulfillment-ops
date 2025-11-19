// Package repositories provides repository interfaces
package repositories

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// ProjectRepository defines the interface for Project persistence
type ProjectRepository interface {
	// Save saves or updates a Project
	Save(ctx context.Context, project *entities.Project) error

	// FindByID finds a Project by ID
	FindByID(ctx context.Context, id string) (*entities.Project, error)

	// FindByMCPID finds all Projects for a given MCP ID
	FindByMCPID(ctx context.Context, mcpID string) ([]*entities.Project, error)

	// List lists all Projects with optional filters
	List(ctx context.Context, filters *ProjectFilters) ([]*entities.Project, error)

	// Delete deletes a Project by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a Project exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}

// ProjectFilters represents filters for listing Projects
type ProjectFilters struct {
	MCPID  string
	Status string
	Limit  int
	Offset int
}
