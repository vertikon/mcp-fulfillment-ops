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

// weaviateClient implements VectorClient using Weaviate
type weaviateClient struct {
	url     string
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

// NewWeaviateClient creates a new Weaviate client
func NewWeaviateClient(url string, apiKey string, timeout time.Duration) VectorClient {
	if url == "" {
		url = "http://localhost:8080"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &weaviateClient{
		url:     url,
		apiKey:  apiKey,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateCollection creates a new collection (class in Weaviate)
func (c *weaviateClient) CreateCollection(ctx context.Context, collectionName string, dimension int) error {
	if collectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}
	if dimension <= 0 {
		return fmt.Errorf("dimension must be greater than 0")
	}

	logger.Info("Creating Weaviate class",
		zap.String("class", collectionName),
		zap.Int("dimension", dimension),
	)

	payload := map[string]interface{}{
		"class":       collectionName,
		"description": fmt.Sprintf("Vector collection %s", collectionName),
		"vectorizer":  "none", // We provide vectors directly
		"properties":  []map[string]interface{}{},
		"vectorIndexConfig": map[string]interface{}{
			"distance": "cosine",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/schema", c.url)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
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

	logger.Info("Created Weaviate class",
		zap.String("class", collectionName),
	)

	return nil
}

// DeleteCollection deletes a collection (class in Weaviate)
func (c *weaviateClient) DeleteCollection(ctx context.Context, collectionName string) error {
	if collectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	logger.Info("Deleting Weaviate class",
		zap.String("class", collectionName),
	)

	url := fmt.Sprintf("%s/v1/schema/%s", c.url, collectionName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	logger.Info("Deleted Weaviate class",
		zap.String("class", collectionName),
	)

	return nil
}

// UpsertVectors upserts vectors into a collection
func (c *weaviateClient) UpsertVectors(ctx context.Context, collectionName string, vectors []*Vector) error {
	if len(vectors) == 0 {
		return fmt.Errorf("vectors cannot be empty")
	}

	logger.Info("Upserting vectors to Weaviate",
		zap.String("collection", collectionName),
		zap.Int("count", len(vectors)),
	)

	// Weaviate batch API
	objects := make([]map[string]interface{}, len(vectors))
	for i, v := range vectors {
		obj := map[string]interface{}{
			"id":     v.ID,
			"class":  collectionName,
			"vector": v.Vector,
		}

		// Add properties from payload
		if v.Payload != nil {
			obj["properties"] = v.Payload
		}

		objects[i] = obj
	}

	payload := map[string]interface{}{
		"objects": objects,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/batch/objects", c.url)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
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
func (c *weaviateClient) SearchVectors(ctx context.Context, collectionName string, queryVector []float64, limit int) ([]*SearchResult, error) {
	if collectionName == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}
	if limit <= 0 {
		limit = 10
	}

	logger.Debug("Searching vectors in Weaviate",
		zap.String("collection", collectionName),
		zap.Int("limit", limit),
	)

	payload := map[string]interface{}{
		"query": map[string]interface{}{
			"@vector": map[string]interface{}{
				"nearVector": map[string]interface{}{
					"vector": queryVector,
				},
				"limit": limit,
			},
		},
		"className": collectionName,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/graphql", c.url)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
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
		Data struct {
			Get map[string]interface{} `json:"Get"`
		} `json:"data"`
		Errors []map[string]interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("query error: %v", result.Errors)
	}

	// Parse results (simplified - Weaviate GraphQL response structure is complex)
	results := make([]*SearchResult, 0)
	// Note: Full parsing would require handling Weaviate's GraphQL response structure
	// This is a simplified implementation

	return results, nil
}

// DeleteVectors deletes vectors by IDs
func (c *weaviateClient) DeleteVectors(ctx context.Context, collectionName string, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids cannot be empty")
	}

	logger.Info("Deleting vectors from Weaviate",
		zap.String("collection", collectionName),
		zap.Int("count", len(ids)),
	)

	// Weaviate batch delete
	for _, id := range ids {
		url := fmt.Sprintf("%s/v1/objects/%s", c.url, id)
		req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		if c.apiKey != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to execute request: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

// GetVector retrieves a vector by ID
func (c *weaviateClient) GetVector(ctx context.Context, collectionName string, id string) (*Vector, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	logger.Debug("Getting vector from Weaviate",
		zap.String("collection", collectionName),
		zap.String("id", id),
	)

	url := fmt.Sprintf("%s/v1/objects/%s", c.url, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
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
		ID         string                 `json:"id"`
		Class      string                 `json:"class"`
		Vector     []float64              `json:"vector"`
		Properties map[string]interface{} `json:"properties"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Vector{
		ID:       result.ID,
		Vector:   result.Vector,
		Payload:  result.Properties,
		Metadata: make(map[string]interface{}),
	}, nil
}
