// Package repositories provides repository interfaces
package repositories

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// MCPRepository defines the interface for MCP persistence
type MCPRepository interface {
	// Save saves or updates an MCP
	Save(ctx context.Context, mcp *entities.MCP) error

	// FindByID finds an MCP by ID
	FindByID(ctx context.Context, id string) (*entities.MCP, error)

	// FindByName finds an MCP by name
	FindByName(ctx context.Context, name string) (*entities.MCP, error)

	// List lists all MCPs with optional filters
	List(ctx context.Context, filters *MCPFilters) ([]*entities.MCP, error)

	// Delete deletes an MCP by ID
	Delete(ctx context.Context, id string) error

	// Exists checks if an MCP exists by ID
	Exists(ctx context.Context, id string) (bool, error)
}

// MCPFilters represents filters for listing MCPs
type MCPFilters struct {
	Stack      string
	HasContext bool
	Limit      int
	Offset     int
}
