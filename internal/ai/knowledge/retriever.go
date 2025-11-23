package knowledge

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// RetrievalMethod defines the retrieval method
type RetrievalMethod string

const (
	MethodVector RetrievalMethod = "vector"
	MethodGraph  RetrievalMethod = "graph"
	MethodHybrid RetrievalMethod = "hybrid"
)

// RetrievalResult represents a single retrieval result
type RetrievalResult struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]interface{}
	Source   RetrievalMethod
}

// KnowledgeContext represents enriched knowledge context for AI
type KnowledgeContext struct {
	Results    []*RetrievalResult
	Query      string
	TotalFound int
	FusedScore float64
}

// VectorRetriever defines interface for vector-based retrieval
type VectorRetriever interface {
	Search(ctx context.Context, query string, limit int) ([]*RetrievalResult, error)
}

// GraphRetriever defines interface for graph-based retrieval
type GraphRetriever interface {
	Traverse(ctx context.Context, query string, limit int) ([]*RetrievalResult, error)
}

// HybridRetriever combines vector and graph retrieval
type HybridRetriever struct {
	vectorRetriever VectorRetriever
	graphRetriever  GraphRetriever
	fusionStrategy  FusionStrategy
	reranker        Reranker
}

// NewHybridRetriever creates a new hybrid retriever
func NewHybridRetriever(
	vectorRetriever VectorRetriever,
	graphRetriever GraphRetriever,
	fusionStrategy FusionStrategy,
	reranker Reranker,
) *HybridRetriever {
	if fusionStrategy == nil {
		fusionStrategy = NewReciprocalRankFusion()
	}
	return &HybridRetriever{
		vectorRetriever: vectorRetriever,
		graphRetriever:  graphRetriever,
		fusionStrategy:  fusionStrategy,
		reranker:        reranker,
	}
}

// Retrieve performs hybrid retrieval combining vector and graph search
func (r *HybridRetriever) Retrieve(ctx context.Context, query string, limit int) (*KnowledgeContext, error) {
	if limit <= 0 {
		limit = 10
	}

	var vectorResults, graphResults []*RetrievalResult
	var vectorErr, graphErr error

	// Parallel retrieval
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if r.vectorRetriever != nil {
			vectorResults, vectorErr = r.vectorRetriever.Search(ctx, query, limit*2)
		}
	}()

	go func() {
		defer wg.Done()
		if r.graphRetriever != nil {
			graphResults, graphErr = r.graphRetriever.Traverse(ctx, query, limit*2)
		}
	}()

	wg.Wait()

	// Handle errors (partial results are acceptable)
	if vectorErr != nil && graphErr != nil {
		return nil, fmt.Errorf("both retrievers failed: vector=%v, graph=%v", vectorErr, graphErr)
	}

	// Fuse results
	fusedResults := r.fusionStrategy.Fuse(vectorResults, graphResults)

	// Rerank if reranker is available
	if r.reranker != nil && len(fusedResults) > 0 {
		reranked, err := r.reranker.Rerank(ctx, query, fusedResults)
		if err == nil {
			fusedResults = reranked
		}
	}

	// Limit results
	if len(fusedResults) > limit {
		fusedResults = fusedResults[:limit]
	}

	// Calculate fused score
	fusedScore := 0.0
	if len(fusedResults) > 0 {
		for _, r := range fusedResults {
			fusedScore += r.Score
		}
		fusedScore /= float64(len(fusedResults))
	}

	return &KnowledgeContext{
		Results:    fusedResults,
		Query:      query,
		TotalFound: len(vectorResults) + len(graphResults),
		FusedScore: fusedScore,
	}, nil
}

// FusionStrategy defines how to fuse vector and graph results
type FusionStrategy interface {
	Fuse(vectorResults []*RetrievalResult, graphResults []*RetrievalResult) []*RetrievalResult
}

// ReciprocalRankFusion implements RRF fusion algorithm
type ReciprocalRankFusion struct {
	k float64 // RRF constant (typically 60)
}

// NewReciprocalRankFusion creates a new RRF fusion strategy
func NewReciprocalRankFusion() *ReciprocalRankFusion {
	return &ReciprocalRankFusion{
		k: 60.0,
	}
}

// Fuse fuses results using Reciprocal Rank Fusion
func (rrf *ReciprocalRankFusion) Fuse(vectorResults []*RetrievalResult, graphResults []*RetrievalResult) []*RetrievalResult {
	// Create score map
	scores := make(map[string]*RetrievalResult)

	// Add vector results
	for i, result := range vectorResults {
		key := result.ID
		if existing, exists := scores[key]; exists {
			// Combine scores using RRF
			rank := float64(i + 1)
			rrfScore := 1.0 / (rrf.k + rank)
			existing.Score += rrfScore
			// Merge metadata
			for k, v := range result.Metadata {
				existing.Metadata[k] = v
			}
			existing.Source = MethodHybrid
		} else {
			rank := float64(i + 1)
			rrfScore := 1.0 / (rrf.k + rank)
			newResult := &RetrievalResult{
				ID:       result.ID,
				Content:  result.Content,
				Score:    rrfScore,
				Metadata: copyMetadataRetriever(result.Metadata),
				Source:   MethodVector,
			}
			scores[key] = newResult
		}
	}

	// Add graph results
	for i, result := range graphResults {
		key := result.ID
		if existing, exists := scores[key]; exists {
			// Combine scores using RRF
			rank := float64(i + 1)
			rrfScore := 1.0 / (rrf.k + rank)
			existing.Score += rrfScore
			// Merge metadata
			for k, v := range result.Metadata {
				existing.Metadata[k] = v
			}
			existing.Source = MethodHybrid
		} else {
			rank := float64(i + 1)
			rrfScore := 1.0 / (rrf.k + rank)
			newResult := &RetrievalResult{
				ID:       result.ID,
				Content:  result.Content,
				Score:    rrfScore,
				Metadata: copyMetadataRetriever(result.Metadata),
				Source:   MethodGraph,
			}
			scores[key] = newResult
		}
	}

	// Convert to slice and sort by score
	results := make([]*RetrievalResult, 0, len(scores))
	for _, result := range scores {
		results = append(results, result)
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// Reranker defines interface for reranking results
type Reranker interface {
	Rerank(ctx context.Context, query string, results []*RetrievalResult) ([]*RetrievalResult, error)
}

// SimpleReranker implements basic reranking
type SimpleReranker struct{}

// NewSimpleReranker creates a new simple reranker
func NewSimpleReranker() *SimpleReranker {
	return &SimpleReranker{}
}

// Rerank reranks results (simple implementation - can be enhanced with ML model)
func (r *SimpleReranker) Rerank(ctx context.Context, query string, results []*RetrievalResult) ([]*RetrievalResult, error) {
	// Simple reranking: boost results with query terms in content
	queryTerms := tokenize(query)
	for _, result := range results {
		contentTerms := tokenize(result.Content)
		boost := calculateTermOverlap(queryTerms, contentTerms)
		result.Score *= (1.0 + boost*0.2) // Boost up to 20%
	}

	// Re-sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// Helper functions
func copyMetadataRetriever(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return make(map[string]interface{})
	}
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func tokenize(text string) []string {
	// Simple tokenization - in production, use proper NLP library
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

func calculateTermOverlap(queryTerms []string, contentTerms []string) float64 {
	if len(queryTerms) == 0 {
		return 0
	}
	matched := 0
	querySet := make(map[string]bool)
	for _, term := range queryTerms {
		querySet[term] = true
	}
	for _, term := range contentTerms {
		if querySet[term] {
			matched++
		}
	}
	return float64(matched) / float64(len(queryTerms))
}
