package knowledge

import (
	"context"
	"testing"
)

// mockVectorClient is a mock implementation of VectorClient
type mockVectorClient struct {
	upsertFunc func(ctx context.Context, collection string, id string, vector []float64, metadata map[string]interface{}) error
	searchFunc func(ctx context.Context, collection string, queryVector []float64, limit int) ([]VectorResult, error)
}

func (m *mockVectorClient) Upsert(ctx context.Context, collection string, id string, vector []float64, metadata map[string]interface{}) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, collection, id, vector, metadata)
	}
	return nil
}

func (m *mockVectorClient) Search(ctx context.Context, collection string, queryVector []float64, limit int) ([]VectorResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, collection, queryVector, limit)
	}
	return []VectorResult{}, nil
}

func (m *mockVectorClient) Delete(ctx context.Context, collection string, id string) error {
	return nil
}

func (m *mockVectorClient) DeleteCollection(ctx context.Context, collection string) error {
	return nil
}

// mockGraphClient is a mock implementation of GraphClient
type mockGraphClient struct {
	createNodeFunc func(ctx context.Context, collection string, id string, properties map[string]interface{}) error
	createEdgeFunc func(ctx context.Context, fromID string, toID string, relation string, properties map[string]interface{}) error
}

func (m *mockGraphClient) CreateNode(ctx context.Context, collection string, id string, properties map[string]interface{}) error {
	if m.createNodeFunc != nil {
		return m.createNodeFunc(ctx, collection, id, properties)
	}
	return nil
}

func (m *mockGraphClient) CreateEdge(ctx context.Context, fromID string, toID string, relation string, properties map[string]interface{}) error {
	if m.createEdgeFunc != nil {
		return m.createEdgeFunc(ctx, fromID, toID, relation, properties)
	}
	return nil
}

func (m *mockGraphClient) Query(ctx context.Context, cypher string, params map[string]interface{}) ([]GraphResult, error) {
	return []GraphResult{}, nil
}

func (m *mockGraphClient) DeleteNode(ctx context.Context, id string) error {
	return nil
}

func TestNewIndexer(t *testing.T) {
	vectorClient := &mockVectorClient{}
	graphClient := &mockGraphClient{}

	indexer := NewIndexer(vectorClient, graphClient, 1000, 200)

	if indexer == nil {
		t.Fatal("NewIndexer returned nil")
	}
	if indexer.chunkSize != 1000 {
		t.Errorf("Expected chunkSize 1000, got %d", indexer.chunkSize)
	}
	if indexer.chunkOverlap != 200 {
		t.Errorf("Expected chunkOverlap 200, got %d", indexer.chunkOverlap)
	}
}

func TestNewIndexer_Defaults(t *testing.T) {
	indexer := NewIndexer(nil, nil, 0, -1)

	if indexer.chunkSize != 1000 {
		t.Errorf("Expected default chunkSize 1000, got %d", indexer.chunkSize)
	}
	if indexer.chunkOverlap != 200 {
		t.Errorf("Expected default chunkOverlap 200, got %d", indexer.chunkOverlap)
	}
}

func TestIndexer_IndexDocument(t *testing.T) {
	vectorClient := &mockVectorClient{}
	graphClient := &mockGraphClient{
		createNodeFunc: func(ctx context.Context, collection string, id string, properties map[string]interface{}) error {
			return nil
		},
		createEdgeFunc: func(ctx context.Context, fromID string, toID string, relation string, properties map[string]interface{}) error {
			return nil
		},
	}

	indexer := NewIndexer(vectorClient, graphClient, 100, 20)

	ctx := context.Background()
	err := indexer.IndexDocument(ctx, "knowledge1", "doc1", "test content", map[string]interface{}{"key": "value"})

	if err != nil {
		t.Fatalf("IndexDocument failed: %v", err)
	}
}

func TestIndexer_IndexDocument_Chunking(t *testing.T) {
	vectorClient := &mockVectorClient{}
	graphClient := &mockGraphClient{
		createNodeFunc: func(ctx context.Context, collection string, id string, properties map[string]interface{}) error {
			return nil
		},
		createEdgeFunc: func(ctx context.Context, fromID string, toID string, relation string, properties map[string]interface{}) error {
			return nil
		},
	}

	indexer := NewIndexer(vectorClient, graphClient, 50, 10)

	// Create content longer than chunk size
	longContent := ""
	for i := 0; i < 200; i++ {
		longContent += "a"
	}

	ctx := context.Background()
	err := indexer.IndexDocument(ctx, "knowledge1", "doc1", longContent, nil)

	if err != nil {
		t.Fatalf("IndexDocument failed: %v", err)
	}
}

func TestIndexer_UpdateVectorIndex(t *testing.T) {
	vectorClient := &mockVectorClient{
		upsertFunc: func(ctx context.Context, collection string, id string, vector []float64, metadata map[string]interface{}) error {
			if collection != "knowledge_knowledge1" {
				t.Errorf("Expected collection 'knowledge_knowledge1', got '%s'", collection)
			}
			if id != "doc1" {
				t.Errorf("Expected id 'doc1', got '%s'", id)
			}
			return nil
		},
	}

	indexer := NewIndexer(vectorClient, nil, 1000, 200)

	ctx := context.Background()
	vector := []float64{0.1, 0.2, 0.3}
	err := indexer.UpdateVectorIndex(ctx, "knowledge1", "doc1", vector)

	if err != nil {
		t.Fatalf("UpdateVectorIndex failed: %v", err)
	}
}

func TestIndexer_UpdateVectorIndex_NoVectorClient(t *testing.T) {
	indexer := NewIndexer(nil, nil, 1000, 200)

	ctx := context.Background()
	err := indexer.UpdateVectorIndex(ctx, "knowledge1", "doc1", []float64{0.1})

	if err == nil {
		t.Error("Expected error when vector client is nil")
	}
}

func TestIndexer_DeleteKnowledge(t *testing.T) {
	vectorClient := &mockVectorClient{}

	indexer := NewIndexer(vectorClient, nil, 1000, 200)

	ctx := context.Background()
	err := indexer.DeleteKnowledge(ctx, "knowledge1")

	if err != nil {
		t.Fatalf("DeleteKnowledge failed: %v", err)
	}
}

func TestIndexer_chunkDocument(t *testing.T) {
	indexer := NewIndexer(nil, nil, 100, 20)

	tests := []struct {
		name      string
		content   string
		minChunks int
	}{
		{
			name:      "short content",
			content:   "short",
			minChunks: 1,
		},
		{
			name:      "long content",
			content:   "",
			minChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "long content" {
				// Create long content
				for i := 0; i < 500; i++ {
					tt.content += "a"
				}
			}

			chunks := indexer.chunkDocument(tt.content)

			if len(chunks) < tt.minChunks {
				t.Errorf("Expected at least %d chunks, got %d", tt.minChunks, len(chunks))
			}

			// Verify chunks don't exceed chunk size
			for i, chunk := range chunks {
				if len(chunk) > indexer.chunkSize && len(tt.content) > indexer.chunkSize {
					t.Errorf("Chunk %d exceeds chunk size: %d > %d", i, len(chunk), indexer.chunkSize)
				}
			}
		})
	}
}
