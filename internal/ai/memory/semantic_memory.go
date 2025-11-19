package memory

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// SemanticMemoryManager manages semantic memory operations
type SemanticMemoryManager struct {
	store *MemoryStore
}

// NewSemanticMemoryManager creates a new semantic memory manager
func NewSemanticMemoryManager(store *MemoryStore) *SemanticMemoryManager {
	return &SemanticMemoryManager{
		store: store,
	}
}

// Create creates a new semantic memory
func (smm *SemanticMemoryManager) Create(ctx context.Context, content string, sessionID string) (*entities.SemanticMemory, error) {
	memory, err := entities.NewSemanticMemory(content, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create semantic memory: %w", err)
	}

	if err := smm.store.SaveSemantic(ctx, memory); err != nil {
		return nil, fmt.Errorf("failed to save semantic memory: %w", err)
	}

	return memory, nil
}

// AddConcept adds a concept to semantic memory
func (smm *SemanticMemoryManager) AddConcept(ctx context.Context, memoryID string, concept string) error {
	mem, err := smm.store.repository.FindByID(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("failed to find semantic memory: %w", err)
	}

	semantic := &entities.SemanticMemory{Memory: mem}
	semantic.AddConcept(concept)

	return smm.store.SaveSemantic(ctx, semantic)
}

// AddRelated adds a related memory reference
func (smm *SemanticMemoryManager) AddRelated(ctx context.Context, memoryID string, relatedID string) error {
	memory, err := smm.store.repository.FindByID(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("failed to find semantic memory: %w", err)
	}

	semantic := &entities.SemanticMemory{Memory: memory}
	semantic.AddRelated(relatedID)

	return smm.store.SaveSemantic(ctx, semantic)
}

// GetByConcept retrieves semantic memories by concept
func (smm *SemanticMemoryManager) GetByConcept(ctx context.Context, concept string, limit int) ([]*entities.SemanticMemory, error) {
	memories, err := smm.store.GetSemantic(ctx, limit*2) // Get more to filter
	if err != nil {
		return nil, err
	}

	filtered := make([]*entities.SemanticMemory, 0)
	for _, memory := range memories {
		concepts := memory.Concepts()
		for _, c := range concepts {
			if c == concept {
				filtered = append(filtered, memory)
				break
			}
		}
		if len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

// GetRelated retrieves related memories
func (smm *SemanticMemoryManager) GetRelated(ctx context.Context, memoryID string) ([]*entities.SemanticMemory, error) {
	mem, err := smm.store.repository.FindByID(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to find semantic memory: %w", err)
	}

	semantic := &entities.SemanticMemory{Memory: mem}
	relatedIDs := semantic.Related()

	related := make([]*entities.SemanticMemory, 0)
	for _, id := range relatedIDs {
		relatedMem, err := smm.store.repository.FindByID(ctx, id)
		if err != nil {
			continue // Skip if not found
		}
		related = append(related, &entities.SemanticMemory{Memory: relatedMem})
	}

	return related, nil
}

// Search searches semantic memories by content
func (smm *SemanticMemoryManager) Search(ctx context.Context, query string, limit int) ([]*entities.SemanticMemory, error) {
	memories, err := smm.store.GetSemantic(ctx, limit*2)
	if err != nil {
		return nil, err
	}

	// Simple text search (in production, use semantic search)
	filtered := make([]*entities.SemanticMemory, 0)
	for _, memory := range memories {
		if contains(memory.Content(), query) {
			filtered = append(filtered, memory)
			if len(filtered) >= limit {
				break
			}
		}
	}

	return filtered, nil
}

// ConsolidateFromEpisodic consolidates episodic memory into semantic
func (smm *SemanticMemoryManager) ConsolidateFromEpisodic(ctx context.Context, episodicMemories []*entities.Memory) error {
	for _, episodic := range episodicMemories {
		// Create semantic memory from episodic
		semantic, err := entities.NewSemanticMemory(episodic.Content(), episodic.SessionID())
		if err != nil {
			return fmt.Errorf("failed to create semantic memory: %w", err)
		}

		// Copy metadata
		semantic.SetMetadata(episodic.Metadata())

		// Set importance
		if err := semantic.SetImportance(episodic.Importance()); err != nil {
			return fmt.Errorf("failed to set importance: %w", err)
		}

		// Save semantic memory
		if err := smm.store.SaveSemantic(ctx, semantic); err != nil {
			return fmt.Errorf("failed to save semantic memory: %w", err)
		}
	}

	return nil
}

// Helper function
func contains(text string, query string) bool {
	// Simple contains check - in production, use proper text search
	return len(text) >= len(query) && (text == query || containsSubstring(text, query))
}

func containsSubstring(text string, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
