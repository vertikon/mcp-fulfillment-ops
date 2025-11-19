package memory

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// WorkingMemoryManager manages working memory operations
type WorkingMemoryManager struct {
	store *MemoryStore
}

// NewWorkingMemoryManager creates a new working memory manager
func NewWorkingMemoryManager(store *MemoryStore) *WorkingMemoryManager {
	return &WorkingMemoryManager{
		store: store,
	}
}

// Create creates a new working memory for a task
func (wmm *WorkingMemoryManager) Create(ctx context.Context, content string, sessionID string, taskID string, maxSteps int) (*entities.WorkingMemory, error) {
	memory, err := entities.NewWorkingMemory(content, sessionID, taskID, maxSteps)
	if err != nil {
		return nil, fmt.Errorf("failed to create working memory: %w", err)
	}

	if err := wmm.store.SaveWorking(ctx, memory); err != nil {
		return nil, fmt.Errorf("failed to save working memory: %w", err)
	}

	return memory, nil
}

// Get retrieves working memory for a task
func (wmm *WorkingMemoryManager) Get(ctx context.Context, sessionID string, taskID string) (*entities.WorkingMemory, error) {
	return wmm.store.GetWorking(ctx, sessionID, taskID)
}

// AdvanceStep advances the task to the next step
func (wmm *WorkingMemoryManager) AdvanceStep(ctx context.Context, sessionID string, taskID string) error {
	memory, err := wmm.store.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return fmt.Errorf("failed to get working memory: %w", err)
	}

	if err := memory.NextStep(); err != nil {
		return fmt.Errorf("failed to advance step: %w", err)
	}

	return wmm.store.SaveWorking(ctx, memory)
}

// SetContext sets context for the current step
func (wmm *WorkingMemoryManager) SetContext(ctx context.Context, sessionID string, taskID string, key string, value interface{}) error {
	memory, err := wmm.store.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return fmt.Errorf("failed to get working memory: %w", err)
	}

	memory.SetContext(key, value)

	return wmm.store.SaveWorking(ctx, memory)
}

// GetContext gets context value
func (wmm *WorkingMemoryManager) GetContext(ctx context.Context, sessionID string, taskID string, key string) (interface{}, bool, error) {
	memory, err := wmm.store.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get working memory: %w", err)
	}

	value, exists := memory.GetContext(key)
	return value, exists, nil
}

// Complete marks the task as completed
func (wmm *WorkingMemoryManager) Complete(ctx context.Context, sessionID string, taskID string) error {
	memory, err := wmm.store.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return fmt.Errorf("failed to get working memory: %w", err)
	}

	memory.Complete()

	return wmm.store.SaveWorking(ctx, memory)
}

// Delete deletes working memory for a task
func (wmm *WorkingMemoryManager) Delete(ctx context.Context, sessionID string, taskID string) error {
	return wmm.store.DeleteWorking(ctx, sessionID, taskID)
}

// IsCompleted checks if task is completed
func (wmm *WorkingMemoryManager) IsCompleted(ctx context.Context, sessionID string, taskID string) (bool, error) {
	memory, err := wmm.store.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to get working memory: %w", err)
	}

	return memory.IsCompleted(), nil
}

// GetProgress returns task progress
func (wmm *WorkingMemoryManager) GetProgress(ctx context.Context, sessionID string, taskID string) (int, int, error) {
	memory, err := wmm.store.GetWorking(ctx, sessionID, taskID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get working memory: %w", err)
	}

	// Note: MaxSteps is not exported, using Step() only
	return memory.Step(), 0, nil
}
