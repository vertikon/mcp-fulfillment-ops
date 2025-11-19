// Package relational provides PostgreSQL relational database implementations
package relational

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/repositories"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/value_objects"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// PostgresTemplateRepository implements TemplateRepository using PostgreSQL
type PostgresTemplateRepository struct {
	db *sql.DB
}

// NewPostgresTemplateRepository creates a new PostgreSQL Template repository
func NewPostgresTemplateRepository(db *sql.DB) repositories.TemplateRepository {
	return &PostgresTemplateRepository{db: db}
}

// Save saves or updates a Template
func (r *PostgresTemplateRepository) Save(ctx context.Context, template *entities.Template) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	// Serialize variables
	variablesJSON, err := json.Marshal(template.Variables())
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	query := `
		INSERT INTO templates (id, name, description, stack, content, variables, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			stack = EXCLUDED.stack,
			content = EXCLUDED.content,
			variables = EXCLUDED.variables,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		template.ID(),
		template.Name(),
		template.Description(),
		string(template.Stack()),
		template.Content(),
		variablesJSON,
		template.Version(),
		template.CreatedAt(),
		template.UpdatedAt(),
	)

	if err != nil {
		logger.Error("Failed to save Template",
			zap.String("id", template.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save Template: %w", err)
	}

	return nil
}

// FindByID finds a Template by ID
func (r *PostgresTemplateRepository) FindByID(ctx context.Context, id string) (*entities.Template, error) {
	query := `
		SELECT id, name, description, stack, content, variables, version, created_at, updated_at
		FROM templates
		WHERE id = $1
	`

	var name, description, stackStr, content string
	var variablesJSON []byte
	var version int
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&id,
		&name,
		&description,
		&stackStr,
		&content,
		&variablesJSON,
		&version,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Template not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find Template: %w", err)
	}

	// Parse stack type
	stack := value_objects.StackType(stackStr)
	if !stack.IsValid() {
		return nil, fmt.Errorf("invalid stack type: %s", stackStr)
	}

	// Create entity
	template, err := entities.NewTemplate(name, description, stack, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create Template entity: %w", err)
	}

	// Unmarshal variables
	var variables []string
	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
		// Add variables to template
		for _, v := range variables {
			if err := template.AddVariable(v); err != nil {
				// Skip duplicates
				continue
			}
		}
	}

	return template, nil
}

// FindByName finds a Template by name
func (r *PostgresTemplateRepository) FindByName(ctx context.Context, name string) (*entities.Template, error) {
	query := `
		SELECT id, name, description, stack, content, variables, version, created_at, updated_at
		FROM templates
		WHERE name = $1
	`

	var id, description, stackStr, content string
	var variablesJSON []byte
	var version int
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&id,
		&name,
		&description,
		&stackStr,
		&content,
		&variablesJSON,
		&version,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Template not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find Template: %w", err)
	}

	stack := value_objects.StackType(stackStr)
	if !stack.IsValid() {
		return nil, fmt.Errorf("invalid stack type: %s", stackStr)
	}

	template, err := entities.NewTemplate(name, description, stack, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create Template entity: %w", err)
	}

	var variables []string
	if len(variablesJSON) > 0 {
		if err := json.Unmarshal(variablesJSON, &variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
		for _, v := range variables {
			if err := template.AddVariable(v); err != nil {
				continue
			}
		}
	}

	return template, nil
}

// List lists all Templates with optional filters
func (r *PostgresTemplateRepository) List(ctx context.Context, filters *repositories.TemplateFilters) ([]*entities.Template, error) {
	query := "SELECT id, name, description, stack, content, variables, version, created_at, updated_at FROM templates WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filters != nil {
		if filters.Stack != "" {
			query += fmt.Sprintf(" AND stack = $%d", argPos)
			args = append(args, filters.Stack)
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
		return nil, fmt.Errorf("failed to list Templates: %w", err)
	}
	defer rows.Close()

	var templates []*entities.Template
	for rows.Next() {
		var id, name, description, stackStr, content string
		var variablesJSON []byte
		var version int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &description, &stackStr, &content, &variablesJSON, &version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		stack := value_objects.StackType(stackStr)
		if !stack.IsValid() {
			continue
		}

		template, err := entities.NewTemplate(name, description, stack, content)
		if err != nil {
			continue
		}

		var variables []string
		if len(variablesJSON) > 0 {
			if err := json.Unmarshal(variablesJSON, &variables); err != nil {
				continue
			}
			for _, v := range variables {
				if err := template.AddVariable(v); err != nil {
					continue
				}
			}
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// Delete deletes a Template by ID
func (r *PostgresTemplateRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM templates WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete Template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Template not found: %s", id)
	}

	return nil
}

// Exists checks if a Template exists by ID
func (r *PostgresTemplateRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM templates WHERE id = $1)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}
