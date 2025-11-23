package knowledge

import (
	"context"
	"testing"
)

// mockVectorRetriever is a mock implementation of VectorRetriever
type mockVectorRetriever struct {
	searchFunc func(ctx context.Context, query string, limit int) ([]*RetrievalResult, error)
}

func (m *mockVectorRetriever) Search(ctx context.Context, query string, limit int) ([]*RetrievalResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, limit)
	}
	return []*RetrievalResult{
		{
			ID:      "vec1",
			Content: "vector result 1",
			Score:   0.9,
			Source:  MethodVector,
		},
	}, nil
}

// mockGraphRetriever is a mock implementation of GraphRetriever
type mockGraphRetriever struct {
	traverseFunc func(ctx context.Context, query string, limit int) ([]*RetrievalResult, error)
}

func (m *mockGraphRetriever) Traverse(ctx context.Context, query string, limit int) ([]*RetrievalResult, error) {
	if m.traverseFunc != nil {
		return m.traverseFunc(ctx, query, limit)
	}
	return []*RetrievalResult{
		{
			ID:      "graph1",
			Content: "graph result 1",
			Score:   0.8,
			Source:  MethodGraph,
		},
	}, nil
}

func TestNewHybridRetriever(t *testing.T) {
	vectorRetriever := &mockVectorRetriever{}
	graphRetriever := &mockGraphRetriever{}

	retriever := NewHybridRetriever(vectorRetriever, graphRetriever, nil, nil)

	if retriever == nil {
		t.Fatal("NewHybridRetriever returned nil")
	}
	if retriever.vectorRetriever != vectorRetriever {
		t.Error("vectorRetriever not set correctly")
	}
	if retriever.graphRetriever != graphRetriever {
		t.Error("graphRetriever not set correctly")
	}
	if retriever.fusionStrategy == nil {
		t.Error("fusionStrategy should have default RRF")
	}
}

func TestHybridRetriever_Retrieve_Success(t *testing.T) {
	vectorRetriever := &mockVectorRetriever{}
	graphRetriever := &mockGraphRetriever{}
	fusionStrategy := NewReciprocalRankFusion()

	retriever := NewHybridRetriever(vectorRetriever, graphRetriever, fusionStrategy, nil)

	ctx := context.Background()
	result, err := retriever.Retrieve(ctx, "test query", 10)

	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}
	if result.Query != "test query" {
		t.Errorf("Expected query 'test query', got '%s'", result.Query)
	}
	if len(result.Results) == 0 {
		t.Error("Expected results, got empty")
	}
}

func TestHybridRetriever_Retrieve_WithReranker(t *testing.T) {
	vectorRetriever := &mockVectorRetriever{}
	graphRetriever := &mockGraphRetriever{}
	fusionStrategy := NewReciprocalRankFusion()
	reranker := NewSimpleReranker()

	retriever := NewHybridRetriever(vectorRetriever, graphRetriever, fusionStrategy, reranker)

	ctx := context.Background()
	result, err := retriever.Retrieve(ctx, "test query", 10)

	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}
}

func TestHybridRetriever_Retrieve_Limit(t *testing.T) {
	vectorRetriever := &mockVectorRetriever{
		searchFunc: func(ctx context.Context, query string, limit int) ([]*RetrievalResult, error) {
			results := make([]*RetrievalResult, 20)
			for i := 0; i < 20; i++ {
				results[i] = &RetrievalResult{
					ID:      "vec" + string(rune(i)),
					Content: "result",
					Score:   0.9 - float64(i)*0.01,
					Source:  MethodVector,
				}
			}
			return results, nil
		},
	}
	graphRetriever := &mockGraphRetriever{
		traverseFunc: func(ctx context.Context, query string, limit int) ([]*RetrievalResult, error) {
			results := make([]*RetrievalResult, 20)
			for i := 0; i < 20; i++ {
				results[i] = &RetrievalResult{
					ID:      "graph" + string(rune(i)),
					Content: "result",
					Score:   0.8 - float64(i)*0.01,
					Source:  MethodGraph,
				}
			}
			return results, nil
		},
	}

	retriever := NewHybridRetriever(vectorRetriever, graphRetriever, NewReciprocalRankFusion(), nil)

	ctx := context.Background()
	result, err := retriever.Retrieve(ctx, "test", 5)

	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if len(result.Results) > 5 {
		t.Errorf("Expected max 5 results, got %d", len(result.Results))
	}
}

func TestReciprocalRankFusion_Fuse(t *testing.T) {
	rrf := NewReciprocalRankFusion()

	vectorResults := []*RetrievalResult{
		{ID: "1", Content: "result 1", Score: 0.9, Source: MethodVector},
		{ID: "2", Content: "result 2", Score: 0.8, Source: MethodVector},
	}

	graphResults := []*RetrievalResult{
		{ID: "1", Content: "result 1", Score: 0.85, Source: MethodGraph},
		{ID: "3", Content: "result 3", Score: 0.7, Source: MethodGraph},
	}

	fused := rrf.Fuse(vectorResults, graphResults)

	if len(fused) == 0 {
		t.Fatal("Expected fused results, got empty")
	}

	// Check that ID "1" appears only once (merged)
	foundID1 := false
	for _, r := range fused {
		if r.ID == "1" {
			if foundID1 {
				t.Error("ID '1' appears multiple times (should be merged)")
			}
			foundID1 = true
			if r.Source != MethodHybrid {
				t.Errorf("Expected merged result to have MethodHybrid source, got %v", r.Source)
			}
		}
	}
}

func TestSimpleReranker_Rerank(t *testing.T) {
	reranker := NewSimpleReranker()

	results := []*RetrievalResult{
		{ID: "1", Content: "test query result", Score: 0.5},
		{ID: "2", Content: "other content", Score: 0.8},
		{ID: "3", Content: "test query match", Score: 0.6},
	}

	ctx := context.Background()
	reranked, err := reranker.Rerank(ctx, "test query", results)

	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}

	if len(reranked) != len(results) {
		t.Errorf("Expected %d results, got %d", len(results), len(reranked))
	}

	// Results with "test query" should have higher scores
	if reranked[0].Score < reranked[1].Score {
		t.Error("Expected reranked results to be sorted by score")
	}
}
