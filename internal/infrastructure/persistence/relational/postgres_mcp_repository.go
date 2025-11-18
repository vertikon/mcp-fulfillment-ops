// Package relational provides PostgreSQL relational database implementations
package relational

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vertikon/mcp-hulk/internal/domain/entities"
	"github.com/vertikon/mcp-hulk/internal/domain/repositories"
	"github.com/vertikon/mcp-hulk/internal/domain/value_objects"
	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// PostgresMCPRepository implements MCPRepository using PostgreSQL
type PostgresMCPRepository struct {
	db *sql.DB
}

// NewPostgresMCPRepository creates a new PostgreSQL MCP repository
func NewPostgresMCPRepository(db *sql.DB) repositories.MCPRepository {
	return &PostgresMCPRepository{db: db}
}

// Save saves or updates an MCP
func (r *PostgresMCPRepository) Save(ctx context.Context, mcp *entities.MCP) error {
	if mcp == nil {
		return fmt.Errorf("MCP cannot be nil")
	}

	// Serialize features
	featuresJSON, err := json.Marshal(mcp.Features())
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	// Serialize context if present
	var contextJSON []byte
	if mcp.HasContext() {
		ctx := mcp.Context()
		contextData := map[string]interface{}{
			"knowledge_id": ctx.KnowledgeID(),
			"documents":    ctx.Documents(),
			"embeddings":   ctx.Embeddings(),
			"metadata":     ctx.Metadata(),
		}
		contextJSON, err = json.Marshal(contextData)
		if err != nil {
			return fmt.Errorf("failed to marshal context: %w", err)
		}
	}

	query := `
		INSERT INTO mcps (id, name, description, stack, path, features, context, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			stack = EXCLUDED.stack,
			path = EXCLUDED.path,
			features = EXCLUDED.features,
			context = EXCLUDED.context,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		mcp.ID(),
		mcp.Name(),
		mcp.Description(),
		string(mcp.Stack()),
		mcp.Path(),
		featuresJSON,
		contextJSON,
		mcp.CreatedAt(),
		mcp.UpdatedAt(),
	)

	if err != nil {
		logger.Error("Failed to save MCP",
			zap.String("id", mcp.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save MCP: %w", err)
	}

	return nil
}

// FindByID finds an MCP by ID
func (r *PostgresMCPRepository) FindByID(ctx context.Context, id string) (*entities.MCP, error) {
	query := `
		SELECT id, name, description, stack, path, features, context, created_at, updated_at
		FROM mcps
		WHERE id = $1
	`

	var mcpID, name, description, stackStr, path string
	var featuresJSON, contextJSON []byte
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&mcpID,
		&name,
		&description,
		&stackStr,
		&path,
		&featuresJSON,
		&contextJSON,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("MCP not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find MCP: %w", err)
	}

	// Reconstruct entity
	stack, err := value_objects.NewStackType(stackStr)
	if err != nil {
		return nil, fmt.Errorf("invalid stack type: %w", err)
	}

	mcp, err := entities.NewMCP(name, description, stack)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP entity: %w", err)
	}

	// Set path if present
	if path != "" {
		if err := mcp.SetPath(path); err != nil {
			logger.Warn("Failed to set path on MCP",
				zap.String("id", mcpID),
				zap.String("path", path),
				zap.Error(err),
			)
		}
	}

	// Unmarshal and add features
	if len(featuresJSON) > 0 {
		var features []struct {
			Name        string                 `json:"name"`
			Status      string                 `json:"status"`
			Description string                 `json:"description"`
			Config      map[string]interface{} `json:"config"`
		}
		if err := json.Unmarshal(featuresJSON, &features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
		for _, f := range features {
			status := value_objects.FeatureStatus(f.Status)
			feature, err := value_objects.NewFeature(f.Name, status, f.Description)
			if err != nil {
				logger.Warn("Failed to create feature",
					zap.String("mcp_id", mcpID),
					zap.String("feature_name", f.Name),
					zap.Error(err),
				)
				continue
			}
			// Set config
			for k, v := range f.Config {
				if err := feature.SetConfig(k, v); err != nil {
					logger.Warn("Failed to set feature config",
						zap.String("mcp_id", mcpID),
						zap.String("feature_name", f.Name),
						zap.String("config_key", k),
						zap.Error(err),
					)
				}
			}
			if err := mcp.AddFeature(feature); err != nil {
				logger.Warn("Failed to add feature to MCP",
					zap.String("mcp_id", mcpID),
					zap.String("feature_name", f.Name),
					zap.Error(err),
				)
			}
		}
	}

	// Unmarshal and add context
	if len(contextJSON) > 0 {
		var contextData struct {
			KnowledgeID string            `json:"knowledge_id"`
			Documents   []string          `json:"documents"`
			Embeddings  map[string][]float64 `json:"embeddings"`
			Metadata    map[string]interface{} `json:"metadata"`
		}
		if err := json.Unmarshal(contextJSON, &contextData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
		if err := mcp.AddContext(contextData.KnowledgeID, contextData.Documents, contextData.Embeddings, contextData.Metadata); err != nil {
			logger.Warn("Failed to add context to MCP",
				zap.String("mcp_id", mcpID),
				zap.String("knowledge_id", contextData.KnowledgeID),
				zap.Error(err),
			)
		}
	}

	return mcp, nil
}

// FindByName finds an MCP by name
func (r *PostgresMCPRepository) FindByName(ctx context.Context, name string) (*entities.MCP, error) {
	query := `
		SELECT id, name, description, stack, path, features, context, created_at, updated_at
		FROM mcps
		WHERE name = $1
	`

	var mcpID, description, stackStr, path string
	var featuresJSON, contextJSON []byte
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&mcpID,
		&name,
		&description,
		&stackStr,
		&path,
		&featuresJSON,
		&contextJSON,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("MCP not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find MCP: %w", err)
	}

	// Reconstruct entity (same logic as FindByID)
	stack, err := value_objects.NewStackType(stackStr)
	if err != nil {
		return nil, fmt.Errorf("invalid stack type: %w", err)
	}

	mcp, err := entities.NewMCP(name, description, stack)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP entity: %w", err)
	}

	// Set path if present
	if path != "" {
		if err := mcp.SetPath(path); err != nil {
			logger.Warn("Failed to set path on MCP",
				zap.String("id", mcpID),
				zap.String("path", path),
				zap.Error(err),
			)
		}
	}

	// Unmarshal and add features
	if len(featuresJSON) > 0 {
		var features []struct {
			Name        string                 `json:"name"`
			Status      string                 `json:"status"`
			Description string                 `json:"description"`
			Config      map[string]interface{} `json:"config"`
		}
		if err := json.Unmarshal(featuresJSON, &features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
		for _, f := range features {
			status := value_objects.FeatureStatus(f.Status)
			feature, err := value_objects.NewFeature(f.Name, status, f.Description)
			if err != nil {
				logger.Warn("Failed to create feature",
					zap.String("mcp_id", mcpID),
					zap.String("feature_name", f.Name),
					zap.Error(err),
				)
				continue
			}
			for k, v := range f.Config {
				if err := feature.SetConfig(k, v); err != nil {
					logger.Warn("Failed to set feature config",
						zap.String("mcp_id", mcpID),
						zap.String("feature_name", f.Name),
						zap.String("config_key", k),
						zap.Error(err),
					)
				}
			}
			if err := mcp.AddFeature(feature); err != nil {
				logger.Warn("Failed to add feature to MCP",
					zap.String("mcp_id", mcpID),
					zap.String("feature_name", f.Name),
					zap.Error(err),
				)
			}
		}
	}

	// Unmarshal and add context
	if len(contextJSON) > 0 {
		var contextData struct {
			KnowledgeID string            `json:"knowledge_id"`
			Documents   []string          `json:"documents"`
			Embeddings  map[string][]float64 `json:"embeddings"`
			Metadata    map[string]interface{} `json:"metadata"`
		}
		if err := json.Unmarshal(contextJSON, &contextData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
		if err := mcp.AddContext(contextData.KnowledgeID, contextData.Documents, contextData.Embeddings, contextData.Metadata); err != nil {
			logger.Warn("Failed to add context to MCP",
				zap.String("mcp_id", mcpID),
				zap.String("knowledge_id", contextData.KnowledgeID),
				zap.Error(err),
			)
		}
	}

	return mcp, nil
}

// List lists all MCPs with optional filters
func (r *PostgresMCPRepository) List(ctx context.Context, filters *repositories.MCPFilters) ([]*entities.MCP, error) {
	query := "SELECT id, name, description, stack, path, features, context, created_at, updated_at FROM mcps WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filters != nil {
		if filters.Stack != "" {
			query += fmt.Sprintf(" AND stack = $%d", argPos)
			args = append(args, filters.Stack)
			argPos++
		}
		if filters.HasContext {
			query += fmt.Sprintf(" AND context IS NOT NULL")
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
		return nil, fmt.Errorf("failed to list MCPs: %w", err)
	}
	defer rows.Close()

	var mcps []*entities.MCP
	for rows.Next() {
		var mcpID, name, description, stackStr, path string
		var featuresJSON, contextJSON []byte
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&mcpID, &name, &description, &stackStr, &path, &featuresJSON, &contextJSON, &createdAt, &updatedAt); err != nil {
			logger.Warn("Failed to scan MCP row",
				zap.Error(err),
			)
			continue
		}

		// Reconstruct entity
		stack, err := value_objects.NewStackType(stackStr)
		if err != nil {
			logger.Warn("Invalid stack type in database",
				zap.String("mcp_id", mcpID),
				zap.String("stack", stackStr),
				zap.Error(err),
			)
			continue
		}

		mcp, err := entities.NewMCP(name, description, stack)
		if err != nil {
			logger.Warn("Failed to create MCP entity",
				zap.String("mcp_id", mcpID),
				zap.Error(err),
			)
			continue
		}

		// Set path if present
		if path != "" {
			if err := mcp.SetPath(path); err != nil {
				logger.Warn("Failed to set path on MCP",
					zap.String("id", mcpID),
					zap.String("path", path),
					zap.Error(err),
				)
			}
		}

		// Unmarshal and add features
		if len(featuresJSON) > 0 {
			var features []struct {
				Name        string                 `json:"name"`
				Status      string                 `json:"status"`
				Description string                 `json:"description"`
				Config      map[string]interface{} `json:"config"`
			}
			if err := json.Unmarshal(featuresJSON, &features); err != nil {
				logger.Warn("Failed to unmarshal features",
					zap.String("mcp_id", mcpID),
					zap.Error(err),
				)
			} else {
				for _, f := range features {
					status := value_objects.FeatureStatus(f.Status)
					feature, err := value_objects.NewFeature(f.Name, status, f.Description)
					if err != nil {
						logger.Warn("Failed to create feature",
							zap.String("mcp_id", mcpID),
							zap.String("feature_name", f.Name),
							zap.Error(err),
						)
						continue
					}
					for k, v := range f.Config {
						if err := feature.SetConfig(k, v); err != nil {
							logger.Warn("Failed to set feature config",
								zap.String("mcp_id", mcpID),
								zap.String("feature_name", f.Name),
								zap.String("config_key", k),
								zap.Error(err),
							)
						}
					}
					if err := mcp.AddFeature(feature); err != nil {
						logger.Warn("Failed to add feature to MCP",
							zap.String("mcp_id", mcpID),
							zap.String("feature_name", f.Name),
							zap.Error(err),
						)
					}
				}
			}
		}

		// Unmarshal and add context
		if len(contextJSON) > 0 {
			var contextData struct {
				KnowledgeID string            `json:"knowledge_id"`
				Documents   []string          `json:"documents"`
				Embeddings  map[string][]float64 `json:"embeddings"`
				Metadata    map[string]interface{} `json:"metadata"`
			}
			if err := json.Unmarshal(contextJSON, &contextData); err != nil {
				logger.Warn("Failed to unmarshal context",
					zap.String("mcp_id", mcpID),
					zap.Error(err),
				)
			} else {
				if err := mcp.AddContext(contextData.KnowledgeID, contextData.Documents, contextData.Embeddings, contextData.Metadata); err != nil {
					logger.Warn("Failed to add context to MCP",
						zap.String("mcp_id", mcpID),
						zap.String("knowledge_id", contextData.KnowledgeID),
						zap.Error(err),
					)
				}
			}
		}

		mcps = append(mcps, mcp)
	}

	return mcps, nil
}

// Delete deletes an MCP by ID
func (r *PostgresMCPRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM mcps WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete MCP: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("MCP not found: %s", id)
	}

	return nil
}

// Exists checks if an MCP exists by ID
func (r *PostgresMCPRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM mcps WHERE id = $1)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

// InitSchema creates the database schema if it doesn't exist
// Deprecated: Use InitAllSchemas instead
func InitSchema(db *sql.DB) error {
	return InitAllSchemas(db)
}
