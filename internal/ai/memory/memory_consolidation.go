package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// ConsolidationPolicy defines policy for memory consolidation
type ConsolidationPolicy struct {
	EpisodicTTL      time.Duration // Time before episodic can be consolidated
	ImportanceThreshold float64    // Minimum importance for consolidation
	BatchSize        int           // Number of memories to consolidate at once
}

// DefaultConsolidationPolicy returns default consolidation policy
func DefaultConsolidationPolicy() *ConsolidationPolicy {
	return &ConsolidationPolicy{
		EpisodicTTL:       24 * time.Hour,
		ImportanceThreshold: 0.7,
		BatchSize:         10,
	}
}

// MemoryConsolidation handles consolidation of episodic memory to semantic memory
type MemoryConsolidation struct {
	episodicManager *EpisodicMemoryManager
	semanticManager *SemanticMemoryManager
	policy          *ConsolidationPolicy
}

// NewMemoryConsolidation creates a new memory consolidation manager
func NewMemoryConsolidation(
	episodicManager *EpisodicMemoryManager,
	semanticManager *SemanticMemoryManager,
	policy *ConsolidationPolicy,
) *MemoryConsolidation {
	if policy == nil {
		policy = DefaultConsolidationPolicy()
	}
	return &MemoryConsolidation{
		episodicManager: episodicManager,
		semanticManager: semanticManager,
		policy:          policy,
	}
}

// ConsolidateSession consolidates episodic memories for a session
func (mc *MemoryConsolidation) ConsolidateSession(ctx context.Context, sessionID string) error {
	// Get memories ready for consolidation
	memories, err := mc.episodicManager.Consolidate(ctx, sessionID, mc.policy.EpisodicTTL)
	if err != nil {
		return fmt.Errorf("failed to get memories for consolidation: %w", err)
	}

	if len(memories) == 0 {
		return nil // Nothing to consolidate
	}

	// Consolidate to semantic memory
	if err := mc.semanticManager.ConsolidateFromEpisodic(ctx, memories); err != nil {
		return fmt.Errorf("failed to consolidate to semantic: %w", err)
	}

	// Optionally delete consolidated episodic memories
	// (In production, you might want to keep them for a while)
	for _, memory := range memories {
		if memory.Importance() >= mc.policy.ImportanceThreshold {
			// High importance memories are consolidated, can delete episodic
			_ = mc.episodicManager.Clear(ctx, sessionID)
		}
	}

	return nil
}

// ConsolidateAll consolidates all eligible episodic memories
func (mc *MemoryConsolidation) ConsolidateAll(ctx context.Context) error {
	// This would require listing all sessions
	// For now, this is a placeholder that would iterate through sessions
	// In production, you would have a session manager

	return fmt.Errorf("consolidate all not yet implemented - requires session listing")
}

// ShouldConsolidate checks if a memory should be consolidated
func (mc *MemoryConsolidation) ShouldConsolidate(memory *entities.Memory) bool {
	if memory.Type() != entities.MemoryTypeEpisodic {
		return false
	}

	age := time.Since(memory.CreatedAt())
	if age < mc.policy.EpisodicTTL {
		return false
	}

	if memory.Importance() < mc.policy.ImportanceThreshold {
		return false
	}

	return true
}

// ConsolidateBatch consolidates a batch of memories
func (mc *MemoryConsolidation) ConsolidateBatch(ctx context.Context, memories []*entities.Memory) error {
	if len(memories) == 0 {
		return nil
	}

	// Filter memories that should be consolidated
	toConsolidate := make([]*entities.Memory, 0)
	for _, memory := range memories {
		if mc.ShouldConsolidate(memory) {
			toConsolidate = append(toConsolidate, memory)
		}
	}

	if len(toConsolidate) == 0 {
		return nil
	}

	// Consolidate to semantic
	if err := mc.semanticManager.ConsolidateFromEpisodic(ctx, toConsolidate); err != nil {
		return fmt.Errorf("failed to consolidate batch: %w", err)
	}

	return nil
}

// AutoConsolidate runs automatic consolidation (should be called periodically)
func (mc *MemoryConsolidation) AutoConsolidate(ctx context.Context) error {
	// This would be called by a background job/scheduler
	// Implementation:
	// 1. Find all sessions with episodic memories
	// 2. Check each session for memories ready to consolidate
	// 3. Consolidate eligible memories
	
	// Since we don't have session listing, we'll use ConsolidateBatch approach
	// Get all episodic memories that should be consolidated
	// This requires access to all episodic memories
	
	// Simplified implementation: Use ConsolidateBatch with memories retrieved from store
	// In production, you would:
	// 1. List all sessions (requires SessionRepository)
	// 2. For each session, call ConsolidateSession
	
	// For now, we'll implement a basic version that processes memories in batches
	// This requires the episodic manager to support batch retrieval
	
	// Note: This is a simplified implementation
	// Full implementation would require:
	// - SessionRepository.ListSessions()
	// - EpisodicMemoryManager.GetAllReadyForConsolidation()
	
	// Workaround: Process consolidation for sessions that have been accessed recently
	// This is not ideal but provides basic functionality
	
	// Return informative error indicating what's needed
	return fmt.Errorf("auto consolidate requires session listing - implement SessionRepository.ListSessions() or call ConsolidateSession manually for each session")
}
