// Package repositories provides repository interfaces
package repositories

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// TemplateRepository defines the interface for Template persistence
type TemplateRepository interface {
	// Save saves or updates a Template
	Save(ctx context.Context, template *entities.Template) error

	// FindByID finds a Template by ID
	FindByID(ctx context.Context, id string) (*entities.Template, error)

	// FindByName finds a Template by name
	FindByName(ctx context.Context, name string) (*entities.Template, error)

	// List lists all Templates with optional filters
	List(ctx context.Context, filters *TemplateFilters) ([]*entities.Template, error)

	// Delete deletes a Template by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a Template exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}

// TemplateFilters represents filters for listing Templates
type TemplateFilters struct {
	Stack  string
	Limit  int
	Offset int
}
