// Package repositories provides repository interfaces
package repositories

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// KnowledgeRepository defines the interface for Knowledge persistence
type KnowledgeRepository interface {
	// Save saves or updates a Knowledge entity
	Save(ctx context.Context, knowledge *entities.Knowledge) error

	// FindByID finds a Knowledge entity by ID
	FindByID(ctx context.Context, id string) (*entities.Knowledge, error)

	// FindByName finds a Knowledge entity by name
	FindByName(ctx context.Context, name string) (*entities.Knowledge, error)

	// List lists all Knowledge entities with optional filters
	List(ctx context.Context, filters *KnowledgeFilters) ([]*entities.Knowledge, error)

	// Delete deletes a Knowledge entity by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if a Knowledge entity exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}

// KnowledgeFilters represents filters for listing Knowledge entities
type KnowledgeFilters struct {
	MinVersion int
	Limit      int
	Offset     int
}
