package knowledge

import (
	"context"
	"fmt"
)

// Indexer handles document indexing and ingestion
type Indexer struct {
	vectorClient VectorClient
	graphClient  GraphClient
	embedder     Embedder
	chunkSize    int
	chunkOverlap int
}

// Embedder defines interface for generating embeddings
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
}

// VectorClient defines interface for vector database operations
type VectorClient interface {
	Upsert(ctx context.Context, collection string, id string, vector []float64, metadata map[string]interface{}) error
	Search(ctx context.Context, collection string, queryVector []float64, limit int) ([]VectorResult, error)
	Delete(ctx context.Context, collection string, id string) error
	DeleteCollection(ctx context.Context, collection string) error
}

// GraphClient defines interface for graph database operations
type GraphClient interface {
	CreateNode(ctx context.Context, collection string, id string, properties map[string]interface{}) error
	CreateEdge(ctx context.Context, fromID string, toID string, relation string, properties map[string]interface{}) error
	Query(ctx context.Context, cypher string, params map[string]interface{}) ([]GraphResult, error)
	DeleteNode(ctx context.Context, id string) error
}

// VectorResult represents a vector search result
type VectorResult struct {
	ID       string
	Score    float64
	Metadata map[string]interface{}
}

// GraphResult represents a graph query result
type GraphResult struct {
	Nodes []map[string]interface{}
	Edges []map[string]interface{}
}

// NewIndexer creates a new indexer
func NewIndexer(vectorClient VectorClient, graphClient GraphClient, embedder Embedder, chunkSize int, chunkOverlap int) *Indexer {
	if chunkSize <= 0 {
		chunkSize = 1000 // Default chunk size
	}
	if chunkOverlap < 0 {
		chunkOverlap = 200 // Default overlap
	}
	return &Indexer{
		vectorClient: vectorClient,
		graphClient:  graphClient,
		embedder:     embedder,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// IndexDocument indexes a document for RAG
func (idx *Indexer) IndexDocument(ctx context.Context, knowledgeID string, documentID string, content string, metadata map[string]interface{}) error {
	// Chunk the document
	chunks := idx.chunkDocument(content)

	// Index each chunk
	for i := range chunks {
		chunkID := fmt.Sprintf("%s_chunk_%d", documentID, i)
		chunkMetadata := copyMetadata(metadata)
		chunkMetadata["chunk_index"] = i
		chunkMetadata["document_id"] = documentID
		chunkMetadata["knowledge_id"] = knowledgeID

		// Note: In production, you would generate embeddings here
		// For now, we assume embeddings are provided separately via UpdateVectorIndex

		// Create graph node for chunk
		if idx.graphClient != nil {
			if err := idx.graphClient.CreateNode(ctx, knowledgeID, chunkID, chunkMetadata); err != nil {
				return fmt.Errorf("failed to create graph node: %w", err)
			}

			// Create edge from document to chunk
			if err := idx.graphClient.CreateEdge(ctx, documentID, chunkID, "contains", nil); err != nil {
				return fmt.Errorf("failed to create graph edge: %w", err)
			}
		}
	}

	return nil
}

// UpdateVectorIndex updates vector index with embeddings
func (idx *Indexer) UpdateVectorIndex(ctx context.Context, knowledgeID string, documentID string, vector []float64) error {
	if idx.vectorClient == nil {
		return fmt.Errorf("vector client not available")
	}

	metadata := map[string]interface{}{
		"document_id":  documentID,
		"knowledge_id": knowledgeID,
	}

	collection := fmt.Sprintf("knowledge_%s", knowledgeID)
	return idx.vectorClient.Upsert(ctx, collection, documentID, vector, metadata)
}

// Search performs semantic search in the index
func (idx *Indexer) Search(ctx context.Context, knowledgeID string, query string, limit int) ([]*RetrievalResult, error) {
	if limit <= 0 {
		limit = 10
	}

	if idx.vectorClient == nil {
		return nil, fmt.Errorf("vector client not available")
	}

	if idx.embedder == nil {
		return nil, fmt.Errorf("embedder not available")
	}

	collection := fmt.Sprintf("knowledge_%s", knowledgeID)

	// Generate query embedding
	queryVector, err := idx.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search in vector database
	results, err := idx.vectorClient.Search(ctx, collection, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector database: %w", err)
	}

	// Convert to RetrievalResult
	retrievalResults := make([]*RetrievalResult, 0, len(results))
	for _, result := range results {
		// Extract content from metadata if available
		content := ""
		if contentVal, ok := result.Metadata["content"].(string); ok {
			content = contentVal
		} else if docID, ok := result.Metadata["document_id"].(string); ok {
			content = fmt.Sprintf("Document: %s", docID)
		}

		retrievalResults = append(retrievalResults, &RetrievalResult{
			ID:       result.ID,
			Content:  content,
			Score:    result.Score,
			Metadata: result.Metadata,
			Source:   MethodVector,
		})
	}

	return retrievalResults, nil
}

// DeleteKnowledge removes all indexed data for a knowledge base
func (idx *Indexer) DeleteKnowledge(ctx context.Context, knowledgeID string) error {
	collection := fmt.Sprintf("knowledge_%s", knowledgeID)

	// Delete from vector DB
	if idx.vectorClient != nil {
		if err := idx.vectorClient.DeleteCollection(ctx, collection); err != nil {
			return fmt.Errorf("failed to delete vector collection: %w", err)
		}
	}

	// Delete from graph DB (would need to delete all nodes/edges)
	// This is a simplified version - production would need proper graph traversal

	return nil
}

// chunkDocument splits document into chunks
func (idx *Indexer) chunkDocument(content string) []string {
	if len(content) <= idx.chunkSize {
		return []string{content}
	}

	chunks := make([]string, 0)
	start := 0

	for start < len(content) {
		end := start + idx.chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunk := content[start:end]
		chunks = append(chunks, chunk)

		// Move start position with overlap
		start = end - idx.chunkOverlap
		if start >= len(content) {
			break
		}
	}

	return chunks
}

// copyMetadata creates a deep copy of metadata
func copyMetadata(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return make(map[string]interface{})
	}
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
