package knowledge

import (
	"context"
	"fmt"
	"testing"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// mockKnowledgeRepository is a mock implementation of KnowledgeRepository
type mockKnowledgeRepository struct {
	knowledge map[string]*entities.Knowledge
}

func newMockKnowledgeRepository() *mockKnowledgeRepository {
	return &mockKnowledgeRepository{
		knowledge: make(map[string]*entities.Knowledge),
	}
}

func (m *mockKnowledgeRepository) Save(ctx context.Context, knowledge *entities.Knowledge) error {
	m.knowledge[knowledge.ID()] = knowledge
	return nil
}

func (m *mockKnowledgeRepository) FindByID(ctx context.Context, id string) (*entities.Knowledge, error) {
	if k, ok := m.knowledge[id]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("knowledge not found")
}

func (m *mockKnowledgeRepository) FindByName(ctx context.Context, name string) (*entities.Knowledge, error) {
	for _, k := range m.knowledge {
		if k.Name() == name {
			return k, nil
		}
	}
	return nil, fmt.Errorf("knowledge not found")
}

func (m *mockKnowledgeRepository) List(ctx context.Context) ([]*entities.Knowledge, error) {
	result := make([]*entities.Knowledge, 0, len(m.knowledge))
	for _, k := range m.knowledge {
		result = append(result, k)
	}
	return result, nil
}

func (m *mockKnowledgeRepository) Delete(ctx context.Context, id string) error {
	delete(m.knowledge, id)
	return nil
}

// mockVectorClient and mockGraphClient for creating real Indexer
type mockVectorClientForIndexer struct{}
func (m *mockVectorClientForIndexer) Upsert(ctx context.Context, collection string, id string, vector []float64, metadata map[string]interface{}) error { return nil }
func (m *mockVectorClientForIndexer) Search(ctx context.Context, collection string, queryVector []float64, limit int) ([]VectorResult, error) { return nil, nil }
func (m *mockVectorClientForIndexer) Delete(ctx context.Context, collection string, id string) error { return nil }
func (m *mockVectorClientForIndexer) DeleteCollection(ctx context.Context, collection string) error { return nil }

type mockGraphClientForIndexer struct{}
func (m *mockGraphClientForIndexer) CreateNode(ctx context.Context, collection string, id string, properties map[string]interface{}) error { return nil }
func (m *mockGraphClientForIndexer) CreateEdge(ctx context.Context, fromID string, toID string, relation string, properties map[string]interface{}) error { return nil }
func (m *mockGraphClientForIndexer) Query(ctx context.Context, cypher string, params map[string]interface{}) ([]GraphResult, error) { return nil, nil }
func (m *mockGraphClientForIndexer) DeleteNode(ctx context.Context, id string) error { return nil }

func TestNewKnowledgeStore(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)

	store := NewKnowledgeStore(repo, indexer)

	if store == nil {
		t.Fatal("NewKnowledgeStore returned nil")
	}
	if store.repository != repo {
		t.Error("repository not set correctly")
	}
	if store.indexer == nil {
		t.Error("indexer not set correctly")
	}
}

func TestKnowledgeStore_AddKnowledge(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	knowledge, err := store.AddKnowledge(ctx, "test knowledge", "test description")

	if err != nil {
		t.Fatalf("AddKnowledge failed: %v", err)
	}

	if knowledge == nil {
		t.Fatal("Knowledge is nil")
	}
	if knowledge.Name() != "test knowledge" {
		t.Errorf("Expected name 'test knowledge', got '%s'", knowledge.Name())
	}
}

func TestKnowledgeStore_AddDocument(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	knowledge, _ := store.AddKnowledge(ctx, "test", "desc")
	knowledgeID := knowledge.ID()

	err := store.AddDocument(ctx, knowledgeID, "document content", map[string]interface{}{"key": "value"})

	if err != nil {
		t.Fatalf("AddDocument failed: %v", err)
	}
}

func TestKnowledgeStore_GetKnowledge(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	created, _ := store.AddKnowledge(ctx, "test", "desc")
	knowledgeID := created.ID()

	retrieved, err := store.GetKnowledge(ctx, knowledgeID)

	if err != nil {
		t.Fatalf("GetKnowledge failed: %v", err)
	}

	if retrieved.ID() != knowledgeID {
		t.Errorf("Expected ID %s, got %s", knowledgeID, retrieved.ID())
	}
}

func TestKnowledgeStore_DeleteKnowledge(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	knowledge, _ := store.AddKnowledge(ctx, "test", "desc")
	knowledgeID := knowledge.ID()
	store.AddDocument(ctx, knowledgeID, "content", nil)

	err := store.DeleteKnowledge(ctx, knowledgeID)

	if err != nil {
		t.Fatalf("DeleteKnowledge failed: %v", err)
	}

	_, err = store.GetKnowledge(ctx, knowledgeID)
	if err == nil {
		t.Error("Expected error when getting deleted knowledge")
	}
}

func TestKnowledgeStore_GetStats(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	knowledge, _ := store.AddKnowledge(ctx, "test", "desc")
	knowledgeID := knowledge.ID()

	store.AddDocument(ctx, knowledgeID, "doc1", nil)
	store.AddDocument(ctx, knowledgeID, "doc2", nil)

	stats, err := store.GetStats(ctx, knowledgeID)

	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.DocumentCount != 2 {
		t.Errorf("Expected 2 documents, got %d", stats.DocumentCount)
	}
	if stats.KnowledgeID != knowledgeID {
		t.Errorf("Expected knowledge ID %s, got %s", knowledgeID, stats.KnowledgeID)
	}
}

func TestKnowledgeStore_IncrementVersion(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	knowledge, _ := store.AddKnowledge(ctx, "test", "desc")
	knowledgeID := knowledge.ID()
	initialVersion := knowledge.Version()

	err := store.IncrementVersion(ctx, knowledgeID)

	if err != nil {
		t.Fatalf("IncrementVersion failed: %v", err)
	}

	updated, _ := store.GetKnowledge(ctx, knowledgeID)
	if updated.Version() != initialVersion+1 {
		t.Errorf("Expected version %d, got %d", initialVersion+1, updated.Version())
	}
}

func TestKnowledgeStore_BulkIndex(t *testing.T) {
	repo := newMockKnowledgeRepository()
	indexer := NewIndexer(&mockVectorClientForIndexer{}, &mockGraphClientForIndexer{}, 1000, 200)
	store := NewKnowledgeStore(repo, indexer)

	ctx := context.Background()
	knowledge, _ := store.AddKnowledge(ctx, "test", "desc")
	knowledgeID := knowledge.ID()

	documents := []DocumentInput{
		{Content: "doc1", Metadata: map[string]interface{}{"key1": "value1"}},
		{Content: "doc2", Metadata: map[string]interface{}{"key2": "value2"}},
	}

	err := store.BulkIndex(ctx, knowledgeID, documents)

	if err != nil {
		t.Fatalf("BulkIndex failed: %v", err)
	}

	stats, _ := store.GetStats(ctx, knowledgeID)
	if stats.DocumentCount != 2 {
		t.Errorf("Expected 2 documents after bulk index, got %d", stats.DocumentCount)
	}
}

