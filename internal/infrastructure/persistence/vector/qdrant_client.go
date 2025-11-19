// Package vector provides vector database client implementations
package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// VectorClient provides vector database operations
type VectorClient interface {
	// CreateCollection creates a new collection
	CreateCollection(ctx context.Context, collectionName string, dimension int) error

	// DeleteCollection deletes a collection
	DeleteCollection(ctx context.Context, collectionName string) error

	// UpsertVectors upserts vectors into a collection
	UpsertVectors(ctx context.Context, collectionName string, vectors []*Vector) error

	// SearchVectors searches for similar vectors
	SearchVectors(ctx context.Context, collectionName string, queryVector []float64, limit int) ([]*SearchResult, error)

	// DeleteVectors deletes vectors by IDs
	DeleteVectors(ctx context.Context, collectionName string, ids []string) error

	// GetVector retrieves a vector by ID
	GetVector(ctx context.Context, collectionName string, id string) (*Vector, error)
}

// Vector represents a vector with metadata
type Vector struct {
	ID       string
	Vector   []float64
	Payload  map[string]interface{}
	Metadata map[string]interface{}
}

// SearchResult represents a search result
type SearchResult struct {
	ID       string
	Score    float64
	Vector   []float64
	Payload  map[string]interface{}
	Metadata map[string]interface{}
}

// qdrantClient implements VectorClient using Qdrant
type qdrantClient struct {
	url     string
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(url string, apiKey string, timeout time.Duration) VectorClient {
	if url == "" {
		url = "http://localhost:6333"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &qdrantClient{
		url:     url,
		apiKey:  apiKey,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateCollection creates a new collection
func (c *qdrantClient) CreateCollection(ctx context.Context, collectionName string, dimension int) error {
	if collectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}
	if dimension <= 0 {
		return fmt.Errorf("dimension must be greater than 0")
	}

	logger.Info("Creating Qdrant collection",
		zap.String("collection", collectionName),
		zap.Int("dimension", dimension),
	)

	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	logger.Info("Created Qdrant collection",
		zap.String("collection", collectionName),
	)

	return nil
}

// DeleteCollection deletes a collection
func (c *qdrantClient) DeleteCollection(ctx context.Context, collectionName string) error {
	if collectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	logger.Info("Deleting Qdrant collection",
		zap.String("collection", collectionName),
	)

	url := fmt.Sprintf("%s/collections/%s", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	logger.Info("Deleted Qdrant collection",
		zap.String("collection", collectionName),
	)

	return nil
}

// UpsertVectors upserts vectors into a collection
func (c *qdrantClient) UpsertVectors(ctx context.Context, collectionName string, vectors []*Vector) error {
	if len(vectors) == 0 {
		return fmt.Errorf("vectors cannot be empty")
	}

	logger.Info("Upserting vectors",
		zap.String("collection", collectionName),
		zap.Int("count", len(vectors)),
	)

	points := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		point := map[string]interface{}{
			"id":      v.ID,
			"vector":  v.Vector,
			"payload": v.Payload,
		}
		points[i] = point
	}

	payload := map[string]interface{}{
		"points": points,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// SearchVectors searches for similar vectors
func (c *qdrantClient) SearchVectors(ctx context.Context, collectionName string, queryVector []float64, limit int) ([]*SearchResult, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	logger.Info("Searching vectors",
		zap.String("collection", collectionName),
		zap.Int("vector_dim", len(queryVector)),
		zap.Int("limit", limit),
	)

	payload := map[string]interface{}{
		"vector": queryVector,
		"limit":  limit,
		"with_payload": true,
		"with_vector":  true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]*SearchResult, len(result.Result))
	for i, r := range result.Result {
		idStr := fmt.Sprintf("%v", r.ID)
		results[i] = &SearchResult{
			ID:       idStr,
			Score:    r.Score,
			Vector:   r.Vector,
			Payload:  r.Payload,
			Metadata:  make(map[string]interface{}),
		}
	}

	return results, nil
}

// DeleteVectors deletes vectors by IDs
func (c *qdrantClient) DeleteVectors(ctx context.Context, collectionName string, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids cannot be empty")
	}

	logger.Info("Deleting vectors",
		zap.String("collection", collectionName),
		zap.Int("count", len(ids)),
	)

	payload := map[string]interface{}{
		"points": ids,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetVector retrieves a vector by ID
func (c *qdrantClient) GetVector(ctx context.Context, collectionName string, id string) (*Vector, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	logger.Debug("Getting vector",
		zap.String("collection", collectionName),
		zap.String("id", id),
	)

	payload := map[string]interface{}{
		"ids":            []string{id},
		"with_payload":   true,
		"with_vector":    true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Result) == 0 {
		return nil, fmt.Errorf("vector not found: %s", id)
	}

	r := result.Result[0]
	idStr := fmt.Sprintf("%v", r.ID)

	return &Vector{
		ID:       idStr,
		Vector:   r.Vector,
		Payload:  r.Payload,
		Metadata: make(map[string]interface{}),
	}, nil
}
