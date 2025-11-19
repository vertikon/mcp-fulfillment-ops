package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// KnowledgeStore manages knowledge storage and retrieval for RAG
type KnowledgeStore struct {
	repository KnowledgeRepository
	indexer    *Indexer
	embedder   Embedder
}

// Embedder is defined in indexer.go - reusing the same interface

// KnowledgeRepository defines interface for knowledge persistence
type KnowledgeRepository interface {
	Save(ctx context.Context, knowledge *entities.Knowledge) error
	FindByID(ctx context.Context, id string) (*entities.Knowledge, error)
	FindByName(ctx context.Context, name string) (*entities.Knowledge, error)
	List(ctx context.Context) ([]*entities.Knowledge, error)
	Delete(ctx context.Context, id string) error
}

// NewKnowledgeStore creates a new knowledge store
func NewKnowledgeStore(repository KnowledgeRepository, indexer *Indexer, embedder Embedder) *KnowledgeStore {
	return &KnowledgeStore{
		repository: repository,
		indexer:    indexer,
		embedder:   embedder,
	}
}

// AddKnowledge adds new knowledge to the store
func (ks *KnowledgeStore) AddKnowledge(ctx context.Context, name string, description string) (*entities.Knowledge, error) {
	knowledge, err := entities.NewKnowledge(name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge: %w", err)
	}

	if err := ks.repository.Save(ctx, knowledge); err != nil {
		return nil, fmt.Errorf("failed to save knowledge: %w", err)
	}

	return knowledge, nil
}

// AddDocument adds a document to knowledge base
func (ks *KnowledgeStore) AddDocument(ctx context.Context, knowledgeID string, content string, metadata map[string]interface{}) error {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return fmt.Errorf("failed to find knowledge: %w", err)
	}

	doc, err := knowledge.AddDocument(content, metadata)
	if err != nil {
		return fmt.Errorf("failed to add document: %w", err)
	}

	// Index the document
	if err := ks.indexer.IndexDocument(ctx, knowledgeID, doc.ID(), content, metadata); err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}

	// Save updated knowledge
	if err := ks.repository.Save(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to save knowledge: %w", err)
	}

	return nil
}

// AddEmbedding adds an embedding for a document
func (ks *KnowledgeStore) AddEmbedding(ctx context.Context, knowledgeID string, documentID string, vector []float64, model string) error {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return fmt.Errorf("failed to find knowledge: %w", err)
	}

	if err := knowledge.AddEmbedding(documentID, vector, model); err != nil {
		return fmt.Errorf("failed to add embedding: %w", err)
	}

	// Update vector index
	if err := ks.indexer.UpdateVectorIndex(ctx, knowledgeID, documentID, vector); err != nil {
		return fmt.Errorf("failed to update vector index: %w", err)
	}

	// Save updated knowledge
	if err := ks.repository.Save(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to save knowledge: %w", err)
	}

	return nil
}

// GetKnowledge retrieves knowledge by ID
func (ks *KnowledgeStore) GetKnowledge(ctx context.Context, id string) (*entities.Knowledge, error) {
	return ks.repository.FindByID(ctx, id)
}

// GetKnowledgeByName retrieves knowledge by name
func (ks *KnowledgeStore) GetKnowledgeByName(ctx context.Context, name string) (*entities.Knowledge, error) {
	return ks.repository.FindByName(ctx, name)
}

// ListKnowledge lists all knowledge bases
func (ks *KnowledgeStore) ListKnowledge(ctx context.Context) ([]*entities.Knowledge, error) {
	return ks.repository.List(ctx)
}

// DeleteKnowledge deletes a knowledge base
func (ks *KnowledgeStore) DeleteKnowledge(ctx context.Context, id string) error {
	// Remove from index
	if err := ks.indexer.DeleteKnowledge(ctx, id); err != nil {
		return fmt.Errorf("failed to delete from index: %w", err)
	}

	// Delete from repository
	if err := ks.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete knowledge: %w", err)
	}

	return nil
}

// SearchDocuments searches documents in knowledge base
func (ks *KnowledgeStore) SearchDocuments(ctx context.Context, knowledgeID string, query string, limit int) ([]*entities.Document, error) {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge: %w", err)
	}

	// Use indexer to search
	results, err := ks.indexer.Search(ctx, knowledgeID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Map results to documents
	documents := make([]*entities.Document, 0, len(results))
	docMap := make(map[string]*entities.Document)
	for _, doc := range knowledge.Documents() {
		docMap[doc.ID()] = doc
	}

	for _, result := range results {
		if doc, exists := docMap[result.ID]; exists {
			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// GetDocumentEmbedding retrieves embedding for a document
func (ks *KnowledgeStore) GetDocumentEmbedding(ctx context.Context, knowledgeID string, documentID string) (*entities.Embedding, error) {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge: %w", err)
	}

	return knowledge.GetEmbedding(documentID)
}

// IncrementVersion increments knowledge version
func (ks *KnowledgeStore) IncrementVersion(ctx context.Context, knowledgeID string) error {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return fmt.Errorf("failed to find knowledge: %w", err)
	}

	knowledge.IncrementVersion()

	if err := ks.repository.Save(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to save knowledge: %w", err)
	}

	return nil
}

// BulkIndex indexes multiple documents at once
func (ks *KnowledgeStore) BulkIndex(ctx context.Context, knowledgeID string, documents []DocumentInput) error {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return fmt.Errorf("failed to find knowledge: %w", err)
	}

	for _, docInput := range documents {
		doc, err := knowledge.AddDocument(docInput.Content, docInput.Metadata)
		if err != nil {
			return fmt.Errorf("failed to add document: %w", err)
		}

		if err := ks.indexer.IndexDocument(ctx, knowledgeID, doc.ID(), docInput.Content, docInput.Metadata); err != nil {
			return fmt.Errorf("failed to index document %s: %w", doc.ID(), err)
		}
	}

	if err := ks.repository.Save(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to save knowledge: %w", err)
	}

	return nil
}

// DocumentInput represents input for document creation
type DocumentInput struct {
	Content  string
	Metadata map[string]interface{}
}

// KnowledgeStats represents statistics about knowledge base
type KnowledgeStats struct {
	KnowledgeID    string
	DocumentCount  int
	EmbeddingCount int
	Version        int
	LastUpdated    time.Time
}

// GetStats returns statistics for a knowledge base
func (ks *KnowledgeStore) GetStats(ctx context.Context, knowledgeID string) (*KnowledgeStats, error) {
	knowledge, err := ks.repository.FindByID(ctx, knowledgeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge: %w", err)
	}

	documents := knowledge.Documents()
	embeddings := knowledge.Embeddings()

	return &KnowledgeStats{
		KnowledgeID:    knowledgeID,
		DocumentCount:  len(documents),
		EmbeddingCount: len(embeddings),
		Version:        knowledge.Version(),
		LastUpdated:    knowledge.UpdatedAt(),
	}, nil
}
