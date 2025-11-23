package memory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// RetrievalStrategy defines how to retrieve memories
type RetrievalStrategy string

const (
	StrategyRecent    RetrievalStrategy = "recent"
	StrategyImportant RetrievalStrategy = "important"
	StrategyRelevant  RetrievalStrategy = "relevant"
	StrategyHybrid    RetrievalStrategy = "hybrid"
)

// MemoryRetrieval handles retrieval of memories for AI context
type MemoryRetrieval struct {
	episodicManager *EpisodicMemoryManager
	semanticManager *SemanticMemoryManager
	workingManager  *WorkingMemoryManager
	store           *MemoryStore
}

// NewMemoryRetrieval creates a new memory retrieval manager
func NewMemoryRetrieval(
	episodicManager *EpisodicMemoryManager,
	semanticManager *SemanticMemoryManager,
	workingManager *WorkingMemoryManager,
	store *MemoryStore,
) *MemoryRetrieval {
	return &MemoryRetrieval{
		episodicManager: episodicManager,
		semanticManager: semanticManager,
		workingManager:  workingManager,
		store:           store,
	}
}

// RetrieveContext retrieves memory context for AI
type RetrieveContext struct {
	SessionID string
	Query     string
	Limit     int
	Strategy  RetrievalStrategy
}

// MemoryContext represents retrieved memory context
type MemoryContext struct {
	Episodic []*entities.EpisodicMemory
	Semantic []*entities.SemanticMemory
	Working  *entities.WorkingMemory
}

// Retrieve retrieves memories based on context
func (mr *MemoryRetrieval) Retrieve(ctx context.Context, req *RetrieveContext) (*MemoryContext, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}

	memCtx := &MemoryContext{}

	// Retrieve episodic memory
	if req.SessionID != "" {
		episodic, err := mr.episodicManager.GetByImportance(ctx, req.SessionID, 0.5)
		if err == nil {
			memCtx.Episodic = episodic
		}
	}

	// Retrieve semantic memory
	if req.Query != "" {
		semantic, err := mr.semanticManager.Search(ctx, req.Query, req.Limit)
		if err == nil {
			memCtx.Semantic = semantic
		}
	} else {
		// Get recent semantic memories
		semantic, err := mr.store.GetSemantic(ctx, req.Limit)
		if err == nil {
			memCtx.Semantic = semantic
		}
	}

	// Retrieve working memory if task ID is provided
	// (would need taskID in request - simplified for now)

	return memCtx, nil
}

// RetrieveForPrompt retrieves memories formatted for prompt inclusion
func (mr *MemoryRetrieval) RetrieveForPrompt(ctx context.Context, sessionID string, query string, limit int) (string, error) {
	req := &RetrieveContext{
		SessionID: sessionID,
		Query:     query,
		Limit:     limit,
		Strategy:  StrategyHybrid,
	}

	memCtx, err := mr.Retrieve(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve memories: %w", err)
	}

	return mr.formatForPrompt(memCtx), nil
}

// formatForPrompt formats memory context for prompt inclusion
func (mr *MemoryRetrieval) formatForPrompt(memCtx *MemoryContext) string {
	var parts []string

	// Add episodic memories
	if len(memCtx.Episodic) > 0 {
		parts = append(parts, "Episodic Memory:")
		for i, mem := range memCtx.Episodic {
			if i >= 5 { // Limit to 5 most recent
				break
			}
			parts = append(parts, fmt.Sprintf("- %s", mem.Content()))
		}
	}

	// Add semantic memories
	if len(memCtx.Semantic) > 0 {
		parts = append(parts, "Semantic Memory:")
		for i, mem := range memCtx.Semantic {
			if i >= 5 {
				break
			}
			parts = append(parts, fmt.Sprintf("- %s", mem.Content()))
		}
	}

	// Add working memory
	if memCtx.Working != nil {
		parts = append(parts, fmt.Sprintf("Working Memory (Task: %s, Step: %d):",
			memCtx.Working.TaskID(), memCtx.Working.Step()))
		parts = append(parts, memCtx.Working.Content())
	}

	if len(parts) == 0 {
		return "No relevant memories found."
	}

	return fmt.Sprintf("Memory Context:\n%s", joinStrings(parts, "\n"))
}

// RetrieveRecent retrieves recent episodic memories
func (mr *MemoryRetrieval) RetrieveRecent(ctx context.Context, sessionID string, window time.Duration, limit int) ([]*entities.MemoryEvent, error) {
	return mr.episodicManager.GetRecentEvents(ctx, sessionID, window)
}

// RetrieveByImportance retrieves memories sorted by importance
func (mr *MemoryRetrieval) RetrieveByImportance(ctx context.Context, sessionID string, minImportance float64) ([]*entities.EpisodicMemory, error) {
	return mr.episodicManager.GetByImportance(ctx, sessionID, minImportance)
}

// RetrieveSemanticByConcept retrieves semantic memories by concept
func (mr *MemoryRetrieval) RetrieveSemanticByConcept(ctx context.Context, concept string, limit int) ([]*entities.SemanticMemory, error) {
	return mr.semanticManager.GetByConcept(ctx, concept, limit)
}

// Helper function
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

// SortByRelevance sorts memories by relevance to query
func (mr *MemoryRetrieval) SortByRelevance(memories []*entities.SemanticMemory, query string) []*entities.SemanticMemory {
	// Simple relevance scoring (in production, use semantic similarity)
	sorted := make([]*entities.SemanticMemory, len(memories))
	copy(sorted, memories)

	sort.Slice(sorted, func(i, j int) bool {
		scoreI := calculateRelevance(sorted[i].Content(), query)
		scoreJ := calculateRelevance(sorted[j].Content(), query)
		return scoreI > scoreJ
	})

	return sorted
}

// calculateRelevance calculates simple relevance score
func calculateRelevance(content string, query string) float64 {
	// Simple word overlap score
	contentWords := tokenize(content)
	queryWords := tokenize(query)

	if len(queryWords) == 0 {
		return 0
	}

	matched := 0
	querySet := make(map[string]bool)
	for _, word := range queryWords {
		querySet[word] = true
	}

	for _, word := range contentWords {
		if querySet[word] {
			matched++
		}
	}

	return float64(matched) / float64(len(queryWords))
}

// tokenize tokenizes text (simple implementation)
func tokenize(text string) []string {
	words := make([]string, 0)
	current := ""
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			current += string(r)
		} else {
			if len(current) > 0 {
				words = append(words, current)
				current = ""
			}
		}
	}
	if len(current) > 0 {
		words = append(words, current)
	}
	return words
}
