package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// mockMemoryRepository is a mock implementation of MemoryRepository
type mockMemoryRepository struct {
	memories map[string]*entities.Memory
}

func newMockMemoryRepository() *mockMemoryRepository {
	return &mockMemoryRepository{
		memories: make(map[string]*entities.Memory),
	}
}

func (m *mockMemoryRepository) Save(ctx context.Context, memory *entities.Memory) error {
	m.memories[memory.ID()] = memory
	return nil
}

func (m *mockMemoryRepository) FindByID(ctx context.Context, id string) (*entities.Memory, error) {
	if mem, ok := m.memories[id]; ok {
		return mem, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockMemoryRepository) FindBySession(ctx context.Context, sessionID string, memoryType entities.MemoryType) ([]*entities.Memory, error) {
	result := make([]*entities.Memory, 0)
	for _, mem := range m.memories {
		if mem.SessionID() == sessionID && mem.Type() == memoryType {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockMemoryRepository) FindByType(ctx context.Context, memoryType entities.MemoryType, limit int) ([]*entities.Memory, error) {
	result := make([]*entities.Memory, 0)
	count := 0
	for _, mem := range m.memories {
		if mem.Type() == memoryType {
			result = append(result, mem)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockMemoryRepository) Delete(ctx context.Context, id string) error {
	delete(m.memories, id)
	return nil
}

func (m *mockMemoryRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	toDelete := make([]string, 0)
	for id, mem := range m.memories {
		if mem.SessionID() == sessionID {
			toDelete = append(toDelete, id)
		}
	}
	for _, id := range toDelete {
		delete(m.memories, id)
	}
	return nil
}

// mockCacheClient is a mock implementation of CacheClient
type mockCacheClient struct {
	data map[string]interface{}
}

func newMockCacheClient() *mockCacheClient {
	return &mockCacheClient{
		data: make(map[string]interface{}),
	}
}

func (m *mockCacheClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCacheClient) Get(ctx context.Context, key string) (interface{}, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockCacheClient) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCacheClient) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}

func TestNewMemoryStore(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()

	store := NewMemoryStore(repo, cache, 24*time.Hour)

	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if store.repository != repo {
		t.Error("repository not set correctly")
	}
	if store.cache != cache {
		t.Error("cache not set correctly")
	}
	if store.ttl != 24*time.Hour {
		t.Errorf("Expected TTL 24h, got %v", store.ttl)
	}
}

func TestNewMemoryStore_DefaultTTL(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), nil, 0)

	if store.ttl != 24*time.Hour {
		t.Errorf("Expected default TTL 24h, got %v", store.ttl)
	}
}

func TestMemoryStore_SaveEpisodic(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()
	store := NewMemoryStore(repo, cache, 24*time.Hour)

	ctx := context.Background()
	memory, err := entities.NewEpisodicMemory("test content", "session1")
	if err != nil {
		t.Fatalf("Failed to create episodic memory: %v", err)
	}

	err = store.SaveEpisodic(ctx, memory)
	if err != nil {
		t.Fatalf("SaveEpisodic failed: %v", err)
	}

	// Verify saved in repository
	saved, err := repo.FindByID(ctx, memory.ID())
	if err != nil {
		t.Fatalf("Memory not found in repository: %v", err)
	}
	if saved.ID() != memory.ID() {
		t.Errorf("Expected ID %s, got %s", memory.ID(), saved.ID())
	}
}

func TestMemoryStore_SaveSemantic(t *testing.T) {
	repo := newMockMemoryRepository()
	store := NewMemoryStore(repo, nil, 24*time.Hour)

	ctx := context.Background()
	memory, err := entities.NewSemanticMemory("test content", "session1")
	if err != nil {
		t.Fatalf("Failed to create semantic memory: %v", err)
	}

	err = store.SaveSemantic(ctx, memory)
	if err != nil {
		t.Fatalf("SaveSemantic failed: %v", err)
	}

	// Verify saved in repository
	saved, err := repo.FindByID(ctx, memory.ID())
	if err != nil {
		t.Fatalf("Memory not found in repository: %v", err)
	}
	if saved.ID() != memory.ID() {
		t.Errorf("Expected ID %s, got %s", memory.ID(), saved.ID())
	}
}

func TestMemoryStore_SaveWorking(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()
	store := NewMemoryStore(repo, cache, 24*time.Hour)

	ctx := context.Background()
	memory, err := entities.NewWorkingMemory("test content", "session1", "task1", 5)
	if err != nil {
		t.Fatalf("Failed to create working memory: %v", err)
	}

	err = store.SaveWorking(ctx, memory)
	if err != nil {
		t.Fatalf("SaveWorking failed: %v", err)
	}

	// Verify saved in repository
	saved, err := repo.FindByID(ctx, memory.ID())
	if err != nil {
		t.Fatalf("Memory not found in repository: %v", err)
	}
	if saved.ID() != memory.ID() {
		t.Errorf("Expected ID %s, got %s", memory.ID(), saved.ID())
	}
}

func TestMemoryStore_GetEpisodic(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()
	store := NewMemoryStore(repo, cache, 24*time.Hour)

	ctx := context.Background()
	memory, _ := entities.NewEpisodicMemory("content", "session1")
	repo.Save(ctx, memory.Memory)

	episodic, err := store.GetEpisodic(ctx, "session1")
	if err != nil {
		t.Fatalf("GetEpisodic failed: %v", err)
	}

	if len(episodic) == 0 {
		t.Error("Expected episodic memories, got empty")
	}
}

func TestMemoryStore_GetSemantic(t *testing.T) {
	repo := newMockMemoryRepository()
	store := NewMemoryStore(repo, nil, 24*time.Hour)

	ctx := context.Background()
	memory, _ := entities.NewSemanticMemory("content", "session1")
	repo.Save(ctx, memory.Memory)

	semantic, err := store.GetSemantic(ctx, 10)
	if err != nil {
		t.Fatalf("GetSemantic failed: %v", err)
	}

	if len(semantic) == 0 {
		t.Error("Expected semantic memories, got empty")
	}
}

func TestMemoryStore_GetWorking(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()
	store := NewMemoryStore(repo, cache, 24*time.Hour)

	ctx := context.Background()
	memory, _ := entities.NewWorkingMemory("content", "session1", "task1", 5)
	repo.Save(ctx, memory.Memory)

	working, err := store.GetWorking(ctx, "session1", "task1")
	if err != nil {
		t.Fatalf("GetWorking failed: %v", err)
	}

	if working.TaskID() != "task1" {
		t.Errorf("Expected task ID 'task1', got '%s'", working.TaskID())
	}
}

func TestMemoryStore_DeleteEpisodic(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()
	store := NewMemoryStore(repo, cache, 24*time.Hour)

	ctx := context.Background()
	memory, _ := entities.NewEpisodicMemory("content", "session1")
	repo.Save(ctx, memory.Memory)

	err := store.DeleteEpisodic(ctx, "session1")
	if err != nil {
		t.Fatalf("DeleteEpisodic failed: %v", err)
	}

	episodic, _ := store.GetEpisodic(ctx, "session1")
	if len(episodic) > 0 {
		t.Error("Expected episodic memories to be deleted")
	}
}

func TestMemoryStore_DeleteWorking(t *testing.T) {
	repo := newMockMemoryRepository()
	cache := newMockCacheClient()
	store := NewMemoryStore(repo, cache, 24*time.Hour)

	ctx := context.Background()
	memory, _ := entities.NewWorkingMemory("content", "session1", "task1", 5)
	repo.Save(ctx, memory.Memory)

	err := store.DeleteWorking(ctx, "session1", "task1")
	if err != nil {
		t.Fatalf("DeleteWorking failed: %v", err)
	}

	_, err = store.GetWorking(ctx, "session1", "task1")
	if err == nil {
		t.Error("Expected error when getting deleted working memory")
	}
}
