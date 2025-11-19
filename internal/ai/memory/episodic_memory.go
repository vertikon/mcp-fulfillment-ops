package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// EpisodicMemoryManager manages episodic memory operations
type EpisodicMemoryManager struct {
	store *MemoryStore
}

// NewEpisodicMemoryManager creates a new episodic memory manager
func NewEpisodicMemoryManager(store *MemoryStore) *EpisodicMemoryManager {
	return &EpisodicMemoryManager{
		store: store,
	}
}

// Create creates a new episodic memory
func (emm *EpisodicMemoryManager) Create(ctx context.Context, content string, sessionID string) (*entities.EpisodicMemory, error) {
	memory, err := entities.NewEpisodicMemory(content, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create episodic memory: %w", err)
	}

	if err := emm.store.SaveEpisodic(ctx, memory); err != nil {
		return nil, fmt.Errorf("failed to save episodic memory: %w", err)
	}

	return memory, nil
}

// AddEvent adds an event to episodic memory
func (emm *EpisodicMemoryManager) AddEvent(ctx context.Context, sessionID string, eventType string, content string) error {
	memories, err := emm.store.GetEpisodic(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get episodic memory: %w", err)
	}

	if len(memories) == 0 {
		// Create new episodic memory
		memory, err := entities.NewEpisodicMemory("", sessionID)
		if err != nil {
			return fmt.Errorf("failed to create episodic memory: %w", err)
		}
		memories = []*entities.EpisodicMemory{memory}
	}

	// Add event to first memory (or create new one if needed)
	memory := memories[0]
	event := entities.NewMemoryEvent(eventType, content)
	memory.AddEvent(event)

	return emm.store.SaveEpisodic(ctx, memory)
}

// GetEvents retrieves events from episodic memory
func (emm *EpisodicMemoryManager) GetEvents(ctx context.Context, sessionID string) ([]*entities.MemoryEvent, error) {
	memories, err := emm.store.GetEpisodic(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get episodic memory: %w", err)
	}

	var allEvents []*entities.MemoryEvent
	for _, memory := range memories {
		events := memory.Events()
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// GetRecentEvents retrieves recent events within a time window
func (emm *EpisodicMemoryManager) GetRecentEvents(ctx context.Context, sessionID string, window time.Duration) ([]*entities.MemoryEvent, error) {
	events, err := emm.GetEvents(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	recent := make([]*entities.MemoryEvent, 0)

	for _, event := range events {
		if now.Sub(event.Timestamp) <= window {
			recent = append(recent, event)
		}
	}

	return recent, nil
}

// Consolidate consolidates episodic memories (for conversion to semantic)
func (emm *EpisodicMemoryManager) Consolidate(ctx context.Context, sessionID string, threshold time.Duration) ([]*entities.Memory, error) {
	memories, err := emm.store.GetEpisodic(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get episodic memory: %w", err)
	}

	now := time.Now()
	consolidated := make([]*entities.Memory, 0)

	for _, memory := range memories {
		// Check if memory is old enough to consolidate
		if now.Sub(memory.CreatedAt()) >= threshold {
			// Check importance
			if memory.Importance() >= 0.7 {
				consolidated = append(consolidated, memory.Memory)
			}
		}
	}

	return consolidated, nil
}

// Clear clears episodic memory for a session
func (emm *EpisodicMemoryManager) Clear(ctx context.Context, sessionID string) error {
	return emm.store.DeleteEpisodic(ctx, sessionID)
}

// GetByImportance retrieves memories sorted by importance
func (emm *EpisodicMemoryManager) GetByImportance(ctx context.Context, sessionID string, minImportance float64) ([]*entities.EpisodicMemory, error) {
	memories, err := emm.store.GetEpisodic(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	filtered := make([]*entities.EpisodicMemory, 0)
	for _, memory := range memories {
		if memory.Importance() >= minImportance {
			filtered = append(filtered, memory)
		}
	}

	return filtered, nil
}
