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
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// PostgresKnowledgeRepository implements KnowledgeRepository using PostgreSQL
type PostgresKnowledgeRepository struct {
	db *sql.DB
}

// NewPostgresKnowledgeRepository creates a new PostgreSQL Knowledge repository
func NewPostgresKnowledgeRepository(db *sql.DB) repositories.KnowledgeRepository {
	return &PostgresKnowledgeRepository{db: db}
}

// Save saves or updates a Knowledge entity
func (r *PostgresKnowledgeRepository) Save(ctx context.Context, knowledge *entities.Knowledge) error {
	if knowledge == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}

	// Serialize documents
	documentsJSON, err := json.Marshal(knowledge.Documents())
	if err != nil {
		return fmt.Errorf("failed to marshal documents: %w", err)
	}

	// Serialize embeddings
	embeddingsJSON, err := json.Marshal(knowledge.Embeddings())
	if err != nil {
		return fmt.Errorf("failed to marshal embeddings: %w", err)
	}

	query := `
		INSERT INTO knowledge (id, name, description, documents, embeddings, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			documents = EXCLUDED.documents,
			embeddings = EXCLUDED.embeddings,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		knowledge.ID(),
		knowledge.Name(),
		knowledge.Description(),
		documentsJSON,
		embeddingsJSON,
		knowledge.Version(),
		knowledge.CreatedAt(),
		knowledge.UpdatedAt(),
	)

	if err != nil {
		logger.Error("Failed to save Knowledge",
			zap.String("id", knowledge.ID()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save Knowledge: %w", err)
	}

	return nil
}

// FindByID finds a Knowledge entity by ID
func (r *PostgresKnowledgeRepository) FindByID(ctx context.Context, id string) (*entities.Knowledge, error) {
	query := `
		SELECT id, name, description, documents, embeddings, version, created_at, updated_at
		FROM knowledge
		WHERE id = $1
	`

	var knowledge entities.Knowledge
	var documentsJSON, embeddingsJSON []byte
	var name, description string
	var version int
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&id,
		&name,
		&description,
		&documentsJSON,
		&embeddingsJSON,
		&version,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Knowledge not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find Knowledge: %w", err)
	}

	// Reconstruct entity
	knowledgePtr, err := entities.NewKnowledge(name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create Knowledge entity: %w", err)
	}

	// Unmarshal documents
	var documents []struct {
		ID        string                 `json:"id"`
		Content   string                 `json:"content"`
		Metadata  map[string]interface{} `json:"metadata"`
		CreatedAt time.Time              `json:"created_at"`
	}
	if len(documentsJSON) > 0 {
		if err := json.Unmarshal(documentsJSON, &documents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal documents: %w", err)
		}
		// Add documents to entity
		for _, doc := range documents {
			if _, err := knowledgePtr.AddDocument(doc.Content, doc.Metadata); err != nil {
				logger.Warn("Failed to add document to knowledge",
					zap.String("knowledge_id", id),
					zap.String("document_id", doc.ID),
					zap.Error(err),
				)
			}
		}
	}

	// Unmarshal embeddings
	var embeddings map[string]struct {
		DocumentID string    `json:"document_id"`
		Vector     []float64 `json:"vector"`
		Dimension  int       `json:"dimension"`
		Model      string    `json:"model"`
		CreatedAt  time.Time `json:"created_at"`
	}
	if len(embeddingsJSON) > 0 {
		if err := json.Unmarshal(embeddingsJSON, &embeddings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embeddings: %w", err)
		}
		// Add embeddings to entity
		for docID, emb := range embeddings {
			if err := knowledgePtr.AddEmbedding(docID, emb.Vector, emb.Model); err != nil {
				logger.Warn("Failed to add embedding to knowledge",
					zap.String("knowledge_id", id),
					zap.String("document_id", docID),
					zap.Error(err),
				)
			}
		}
	}

	return knowledgePtr, nil
}

// FindByName finds a Knowledge entity by name
func (r *PostgresKnowledgeRepository) FindByName(ctx context.Context, name string) (*entities.Knowledge, error) {
	query := `
		SELECT id, name, description, documents, embeddings, version, created_at, updated_at
		FROM knowledge
		WHERE name = $1
	`

	var id string
	var documentsJSON, embeddingsJSON []byte
	var description string
	var version int
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&id,
		&name,
		&description,
		&documentsJSON,
		&embeddingsJSON,
		&version,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Knowledge not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find Knowledge: %w", err)
	}

	knowledge, err := entities.NewKnowledge(name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create Knowledge entity: %w", err)
	}

	return &knowledge, nil
}

// List lists all Knowledge entities with optional filters
func (r *PostgresKnowledgeRepository) List(ctx context.Context, filters *repositories.KnowledgeFilters) ([]*entities.Knowledge, error) {
	query := "SELECT id, name, description, documents, embeddings, version, created_at, updated_at FROM knowledge WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if filters != nil {
		if filters.MinVersion > 0 {
			query += fmt.Sprintf(" AND version >= $%d", argPos)
			args = append(args, filters.MinVersion)
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
		return nil, fmt.Errorf("failed to list Knowledge entities: %w", err)
	}
	defer rows.Close()

	var knowledgeList []*entities.Knowledge
	for rows.Next() {
		var id, name, description string
		var documentsJSON, embeddingsJSON []byte
		var version int
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &description, &documentsJSON, &embeddingsJSON, &version, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		knowledgePtr, err := entities.NewKnowledge(name, description)
		if err != nil {
			continue // Skip invalid entries
		}

		knowledgeList = append(knowledgeList, knowledgePtr)
	}

	return knowledgeList, nil
}

// Delete deletes a Knowledge entity by ID
func (r *PostgresKnowledgeRepository) Delete(ctx context.Context, id string) error {
	query := "DELETE FROM knowledge WHERE id = $1"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete Knowledge: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Knowledge not found: %s", id)
	}

	return nil
}

// Exists checks if a Knowledge entity exists by ID
func (r *PostgresKnowledgeRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM knowledge WHERE id = $1)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}
