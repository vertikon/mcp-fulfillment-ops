// Package relational provides PostgreSQL relational database implementations
package relational

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/repositories"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/value_objects"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// PostgresProjectRepository implements ProjectRepository using PostgreSQL
type PostgresProjectRepository struct {
	db *sql.DB
}

// NewPostgresProjectRepository creates a new PostgreSQL Project repository
func NewPostgresProjectRepository(db *sql.DB) repositories.ProjectRepository {
	return &PostgresProjectRepository{db: db}
}

// Save saves or updates a Project
func (r *PostgresProjectRepository) Save(ctx context.Context, project *entities.Project) error {
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}

	query := `
		INSERT INTO projects (id, name, description, mcp_id, stack, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			mcp_id = EXCLUDED.mcp_id,
			stack = EXCLUDED.stack,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.ExecContext(ctx, query,
		project.ID(),
		project.Name(),
		project.Description(),
		project.MCPID(),
		string(project.Stack()),
		string(project.Status()),
		project.CreatedAt(),
		project.UpdatedAt(),
	)

	if err != nil {
		logger.Error("Failed to save Project",
			zap.String("id", project.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save Project: %w", err)
	}

	return nil
}

// FindByID finds a Project by ID
func (r *PostgresProjectRepository) FindByID(ctx context.Context, id string) (*entities.Project, error) {
	query := `
		SELECT id, name, description, mcp_id, stack, status, created_at, updated_at
		FROM projects
		WHERE id = $1
	`

	var name, description, mcpID, stackStr, statusStr string
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&id,
		&name,
		&description,
		&mcpID,
		&stackStr,
		&statusStr,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Project not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find Project: %w", err)
	}

	// Parse stack type
	stack := value_objects.StackType(stackStr)
	if !stack.IsValid() {
		return nil, fmt.Errorf("invalid stack type: %s", stackStr)
	}

	// Create entity
	project, err := entities.NewProject(name, description, mcpID, stack)
	if err != nil {
		return nil, fmt.Errorf("failed to create Project entity: %w", err)
	}

	// Set status
	status := entities.ProjectStatus(statusStr)
	if err := project.SetStatus(status); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}

	return project, nil
}

// FindByMCPID finds all Projects for a given MCP ID
func (r *PostgresProjectRepository) FindByMCPID(ctx context.Context, mcpID string) ([]*entities.Project, error) {
	query := `
		SELECT id, name, description, mcp_id, stack, status, created_at, updated_at
		FROM projects
		WHERE mcp_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, mcpID)
	if err != nil {
		return nil, fmt.Errorf("failed to query Projects: %w", err)
	}
	defer rows.Close()

	var projects []*entities.Project
	for rows.Next() {
		var id, name, description, stackStr, statusStr string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &description, &mcpID, &stackStr, &statusStr, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		stack := value_objects.StackType(stackStr)
		if !stack.IsValid() {
			continue // Skip invalid entries
		}

		project, err := entities.NewProject(name, description, mcpID, stack)
		if err != nil {
			continue // Skip invalid entries
		}

		status := entities.ProjectStatus(statusStr)
		if err := project.SetStatus(status); err != nil {
			continue // Skip invalid entries
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// List lists all Projects with optional filters
func (r *PostgresProjectRepository) List(ctx context.Context, filters *repositories.ProjectFilters) ([]*entities.Project, error) {
	query := "SELECT id, name, description, mcp_id, stack, status, created_at, updated_at FROM projects WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filters != nil {
		if filters.MCPID != "" {
			query += fmt.Sprintf(" AND mcp_id = $%d", argPos)
			args = append(args, filters.MCPID)
			argPos++
		}
		if filters.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argPos)
			args = append(args, filters.Status)
			argPos++
		}
		if filters.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filters.Limit)
			argPos++
		}
		if filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filters.Offset)
			argPos++
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list Projects: %w", err)
	}
	defer rows.Close()

	var projects []*entities.Project
	for rows.Next() {
		var id, name, description, mcpID, stackStr, statusStr string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &description, &mcpID, &stackStr, &statusStr, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		stack := value_objects.StackType(stackStr)
		if !stack.IsValid() {
			continue
		}

		project, err := entities.NewProject(name, description, mcpID, stack)
		if err != nil {
			continue
		}

		status := entities.ProjectStatus(statusStr)
		if err := project.SetStatus(status); err != nil {
			continue
		}

		projects = append(projects, project)
	}

	return projects, nil
}

// Delete deletes a Project by ID
func (r *PostgresProjectRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM projects WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete Project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Project not found: %s", id)
	}

	return nil
}

// Exists checks if a Project exists by ID
func (r *PostgresProjectRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}
