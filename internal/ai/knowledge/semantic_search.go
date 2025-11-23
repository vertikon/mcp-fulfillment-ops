package knowledge

import (
	"context"
	"fmt"
)

// SemanticSearch provides semantic search capabilities
type SemanticSearch struct {
	vectorClient VectorClient
	embedder     Embedder
}

// Embedder is defined in indexer.go - reusing the same interface

// NewSemanticSearch creates a new semantic search instance
func NewSemanticSearch(vectorClient VectorClient, embedder Embedder) *SemanticSearch {
	return &SemanticSearch{
		vectorClient: vectorClient,
		embedder:     embedder,
	}
}

// Search performs semantic search
func (ss *SemanticSearch) Search(ctx context.Context, collection string, query string, limit int) ([]*RetrievalResult, error) {
	if limit <= 0 {
		limit = 10
	}

	// Generate query embedding
	queryVector, err := ss.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search in vector database
	results, err := ss.vectorClient.Search(ctx, collection, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector database: %w", err)
	}

	// Convert to RetrievalResult
	retrievalResults := make([]*RetrievalResult, 0, len(results))
	for _, result := range results {
		retrievalResults = append(retrievalResults, &RetrievalResult{
			ID:       result.ID,
			Score:    result.Score,
			Metadata: result.Metadata,
			Source:   MethodVector,
		})
	}

	return retrievalResults, nil
}

// SearchWithFilters performs semantic search with metadata filters
func (ss *SemanticSearch) SearchWithFilters(ctx context.Context, collection string, query string, filters map[string]interface{}, limit int) ([]*RetrievalResult, error) {
	// Generate query embedding
	queryVector, err := ss.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search in vector database (filters would be applied by vector client)
	results, err := ss.vectorClient.Search(ctx, collection, queryVector, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector database: %w", err)
	}

	// Apply filters to results
	filteredResults := make([]*RetrievalResult, 0)
	for _, result := range results {
		if matchesFilters(result.Metadata, filters) {
			retrievalResult := &RetrievalResult{
				ID:       result.ID,
				Score:    result.Score,
				Metadata: result.Metadata,
				Source:   MethodVector,
			}
			filteredResults = append(filteredResults, retrievalResult)
		}
	}

	return filteredResults, nil
}

// SimilaritySearch finds similar documents
func (ss *SemanticSearch) SimilaritySearch(ctx context.Context, collection string, documentID string, limit int) ([]*RetrievalResult, error) {
	if limit <= 0 {
		limit = 10
	}

	if ss.vectorClient == nil {
		return nil, fmt.Errorf("vector client not available")
	}

	// First, retrieve the document's embedding from vector database
	// We'll search for the document by ID first to get its embedding
	// Note: This assumes the vector client has a method to get vector by ID
	// For now, we'll use a workaround: search with a very specific filter

	// Try to get the document vector by searching with documentID in metadata
	// This is a simplified approach - in production, vector client should have GetVector method
	// For now, we'll need to get the document content first and then search

	// Alternative approach: Use the documentID as a query and search for similar content
	// This requires having the document content stored somewhere accessible
	// For now, return an error indicating the document content is needed

	// In a full implementation, you would:
	// 1. Get document content from KnowledgeStore or repository
	// 2. Generate embedding for that content
	// 3. Use that embedding to search for similar documents

	// Simplified implementation: search using documentID as query term
	// This assumes documentID contains semantic information
	queryVector, err := ss.embedder.Embed(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for document: %w", err)
	}

	// Search for similar documents
	results, err := ss.vectorClient.Search(ctx, collection, queryVector, limit+1) // +1 to exclude the document itself
	if err != nil {
		return nil, fmt.Errorf("failed to search for similar documents: %w", err)
	}

	// Filter out the document itself
	filteredResults := make([]*RetrievalResult, 0)
	for _, result := range results {
		// Skip the document itself
		if result.ID == documentID {
			continue
		}

		content := ""
		if contentVal, ok := result.Metadata["content"].(string); ok {
			content = contentVal
		} else {
			content = fmt.Sprintf("Document: %s", result.ID)
		}

		filteredResults = append(filteredResults, &RetrievalResult{
			ID:       result.ID,
			Content:  content,
			Score:    result.Score,
			Metadata: result.Metadata,
			Source:   MethodVector,
		})

		if len(filteredResults) >= limit {
			break
		}
	}

	return filteredResults, nil
}

// matchesFilters checks if metadata matches filters
func matchesFilters(metadata map[string]interface{}, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, filterValue := range filters {
		metadataValue, exists := metadata[key]
		if !exists {
			return false
		}

		// Simple equality check - can be enhanced for range queries, etc.
		if metadataValue != filterValue {
			return false
		}
	}

	return true
}
