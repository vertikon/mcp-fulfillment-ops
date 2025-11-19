package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// MemoryRepository defines interface for memory persistence
type MemoryRepository interface {
	Save(ctx context.Context, memory *entities.Memory) error
	FindByID(ctx context.Context, id string) (*entities.Memory, error)
	FindBySession(ctx context.Context, sessionID string, memoryType entities.MemoryType) ([]*entities.Memory, error)
	FindByType(ctx context.Context, memoryType entities.MemoryType, limit int) ([]*entities.Memory, error)
	Delete(ctx context.Context, id string) error
	DeleteBySession(ctx context.Context, sessionID string) error
}

// CacheClient defines interface for cache operations (Redis)
type CacheClient interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// MemoryStore manages memory storage (episodic, semantic, working)
type MemoryStore struct {
	repository MemoryRepository
	cache      CacheClient
	ttl        time.Duration
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(repository MemoryRepository, cache CacheClient, ttl time.Duration) *MemoryStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default TTL
	}
	return &MemoryStore{
		repository: repository,
		cache:      cache,
		ttl:        ttl,
	}
}

// SaveEpisodic saves episodic memory
func (ms *MemoryStore) SaveEpisodic(ctx context.Context, memory *entities.EpisodicMemory) error {
	if err := ms.repository.Save(ctx, memory.Memory); err != nil {
		return fmt.Errorf("failed to save episodic memory: %w", err)
	}

	// Cache episodic memory
	if ms.cache != nil {
		cacheKey := ms.episodicCacheKey(memory.SessionID())
		if err := ms.cache.Set(ctx, cacheKey, memory, ms.ttl); err != nil {
			// Log error but don't fail
			_ = err
		}
	}

	return nil
}

// SaveSemantic saves semantic memory
func (ms *MemoryStore) SaveSemantic(ctx context.Context, memory *entities.SemanticMemory) error {
	if err := ms.repository.Save(ctx, memory.Memory); err != nil {
		return fmt.Errorf("failed to save semantic memory: %w", err)
	}

	// Semantic memory is not cached (long-term storage)
	return nil
}

// SaveWorking saves working memory
func (ms *MemoryStore) SaveWorking(ctx context.Context, memory *entities.WorkingMemory) error {
	if err := ms.repository.Save(ctx, memory.Memory); err != nil {
		return fmt.Errorf("failed to save working memory: %w", err)
	}

	// Cache working memory (short-term)
	if ms.cache != nil {
		cacheKey := ms.workingCacheKey(memory.SessionID(), memory.TaskID())
		if err := ms.cache.Set(ctx, cacheKey, memory, 1*time.Hour); err != nil {
			_ = err
		}
	}

	return nil
}

// GetEpisodic retrieves episodic memory by session
func (ms *MemoryStore) GetEpisodic(ctx context.Context, sessionID string) ([]*entities.EpisodicMemory, error) {
	// Try cache first
	if ms.cache != nil {
		cacheKey := ms.episodicCacheKey(sessionID)
		if cached, err := ms.cache.Get(ctx, cacheKey); err == nil {
			if memories, ok := cached.([]*entities.EpisodicMemory); ok {
				return memories, nil
			}
		}
	}

	// Get from repository
	memories, err := ms.repository.FindBySession(ctx, sessionID, entities.MemoryTypeEpisodic)
	if err != nil {
		return nil, fmt.Errorf("failed to get episodic memory: %w", err)
	}

	// Convert to EpisodicMemory
	episodicMemories := make([]*entities.EpisodicMemory, 0, len(memories))
	for _, mem := range memories {
		episodic := &entities.EpisodicMemory{Memory: mem}
		episodicMemories = append(episodicMemories, episodic)
	}

	return episodicMemories, nil
}

// GetSemantic retrieves semantic memory
func (ms *MemoryStore) GetSemantic(ctx context.Context, limit int) ([]*entities.SemanticMemory, error) {
	memories, err := ms.repository.FindByType(ctx, entities.MemoryTypeSemantic, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get semantic memory: %w", err)
	}

	semanticMemories := make([]*entities.SemanticMemory, 0, len(memories))
	for _, mem := range memories {
		semantic := &entities.SemanticMemory{Memory: mem}
		semanticMemories = append(semanticMemories, semantic)
	}

	return semanticMemories, nil
}

// GetWorking retrieves working memory by session and task
func (ms *MemoryStore) GetWorking(ctx context.Context, sessionID string, taskID string) (*entities.WorkingMemory, error) {
	// Try cache first
	if ms.cache != nil {
		cacheKey := ms.workingCacheKey(sessionID, taskID)
		if cached, err := ms.cache.Get(ctx, cacheKey); err == nil {
			if memory, ok := cached.(*entities.WorkingMemory); ok {
				return memory, nil
			}
		}
	}

	// Get from repository (would need taskID in query - simplified for now)
	memories, err := ms.repository.FindBySession(ctx, sessionID, entities.MemoryTypeWorking)
	if err != nil {
		return nil, fmt.Errorf("failed to get working memory: %w", err)
	}

	// Find by taskID
	for _, mem := range memories {
		working := &entities.WorkingMemory{Memory: mem}
		if working.TaskID() == taskID {
			return working, nil
		}
	}

	return nil, fmt.Errorf("working memory not found for task %s", taskID)
}

// DeleteEpisodic deletes episodic memory by session
func (ms *MemoryStore) DeleteEpisodic(ctx context.Context, sessionID string) error {
	if err := ms.repository.DeleteBySession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete episodic memory: %w", err)
	}

	// Clear cache
	if ms.cache != nil {
		cacheKey := ms.episodicCacheKey(sessionID)
		_ = ms.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// DeleteWorking deletes working memory
func (ms *MemoryStore) DeleteWorking(ctx context.Context, sessionID string, taskID string) error {
	working, err := ms.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return err
	}

	if err := ms.repository.Delete(ctx, working.ID()); err != nil {
		return fmt.Errorf("failed to delete working memory: %w", err)
	}

	// Clear cache
	if ms.cache != nil {
		cacheKey := ms.workingCacheKey(sessionID, taskID)
		_ = ms.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// Helper methods for cache keys
func (ms *MemoryStore) episodicCacheKey(sessionID string) string {
	return fmt.Sprintf("memory:episodic:%s", sessionID)
}

func (ms *MemoryStore) workingCacheKey(sessionID string, taskID string) string {
	return fmt.Sprintf("memory:working:%s:%s", sessionID, taskID)
}
