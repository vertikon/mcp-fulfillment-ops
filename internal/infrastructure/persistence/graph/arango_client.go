// Package graph provides graph database client implementations
package graph

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

// arangoClient implements GraphClient using ArangoDB
type arangoClient struct {
	url      string
	username string
	password string
	database string
	timeout  time.Duration
	client   *http.Client
}

// NewArangoClient creates a new ArangoDB client
func NewArangoClient(url string, username string, password string, database string, timeout time.Duration) GraphClient {
	if url == "" {
		url = "http://localhost:8529"
	}
	if database == "" {
		database = "_system"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &arangoClient{
		url:      url,
		username: username,
		password: password,
		database: database,
		timeout:  timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateNode creates a node (vertex) in the graph
func (c *arangoClient) CreateNode(ctx context.Context, label string, properties map[string]interface{}) (string, error) {
	if label == "" {
		return "", fmt.Errorf("label cannot be empty")
	}

	logger.Info("Creating ArangoDB vertex",
		zap.String("collection", label),
	)

	if properties == nil {
		properties = make(map[string]interface{})
	}

	// Create vertex using AQL
	aql := fmt.Sprintf("INSERT %s INTO %s RETURN NEW._key", toAQLValue(properties), label)
	result, err := c.Query(ctx, aql, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create vertex: %w", err)
	}

	if len(result.Records) == 0 || len(result.Records[0]) == 0 {
		return "", fmt.Errorf("failed to get vertex ID")
	}

	key, ok := result.Records[0]["_key"]
	if !ok {
		return "", fmt.Errorf("failed to get vertex key from result")
	}

	keyStr := fmt.Sprintf("%v", key)
	return keyStr, nil
}

// CreateRelationship creates a relationship (edge) between nodes
func (c *arangoClient) CreateRelationship(ctx context.Context, fromID string, toID string, relType string, properties map[string]interface{}) error {
	if fromID == "" || toID == "" {
		return fmt.Errorf("fromID and toID cannot be empty")
	}
	if relType == "" {
		return fmt.Errorf("relationship type cannot be empty")
	}

	logger.Info("Creating ArangoDB edge",
		zap.String("from", fromID),
		zap.String("to", toID),
		zap.String("type", relType),
	)

	if properties == nil {
		properties = make(map[string]interface{})
	}

	// Create edge using AQL
	// Assuming vertices are in collections named by their label
	// and edges are in a collection named relType
	aql := fmt.Sprintf(
		"LET from = DOCUMENT('%s/%s') LET to = DOCUMENT('%s/%s') INSERT %s INTO %s RETURN NEW._key",
		"vertices", fromID, "vertices", toID, toAQLValue(properties), relType,
	)

	_, err := c.Query(ctx, aql, nil)
	if err != nil {
		return fmt.Errorf("failed to create edge: %w", err)
	}

	return nil
}

// Query executes an AQL query
func (c *arangoClient) Query(ctx context.Context, aql string, params map[string]interface{}) (*QueryResult, error) {
	if aql == "" {
		return nil, fmt.Errorf("AQL query cannot be empty")
	}

	logger.Debug("Executing AQL query",
		zap.String("query", aql),
	)

	if params == nil {
		params = make(map[string]interface{})
	}

	payload := map[string]interface{}{
		"query":    aql,
		"bindVars": params,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/_api/cursor", c.url)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result       []map[string]interface{} `json:"result"`
		Error        bool                     `json:"error"`
		Code         int                      `json:"code"`
		ErrorMessage string                   `json:"errorMessage,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error {
		return nil, fmt.Errorf("query error: %s (code: %d)", result.ErrorMessage, result.Code)
	}

	return &QueryResult{
		Nodes:         []*Node{},
		Relationships: []*Relationship{},
		Records:       result.Result,
	}, nil
}

// DeleteNode deletes a node (vertex) by ID
func (c *arangoClient) DeleteNode(ctx context.Context, nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	logger.Info("Deleting ArangoDB vertex",
		zap.String("id", nodeID),
	)

	// Assuming vertices are in a collection named "vertices"
	aql := fmt.Sprintf("REMOVE '%s' IN vertices RETURN OLD._key", nodeID)

	_, err := c.Query(ctx, aql, nil)
	if err != nil {
		return fmt.Errorf("failed to delete vertex: %w", err)
	}

	return nil
}

// DeleteRelationship deletes a relationship (edge) by ID
func (c *arangoClient) DeleteRelationship(ctx context.Context, relID string) error {
	if relID == "" {
		return fmt.Errorf("relID cannot be empty")
	}

	logger.Info("Deleting ArangoDB edge",
		zap.String("id", relID),
	)

	// Assuming edges are in collections named by their type
	// This is a simplified implementation
	aql := fmt.Sprintf("REMOVE '%s' IN edges RETURN OLD._key", relID)

	_, err := c.Query(ctx, aql, nil)
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	return nil
}

// FindNode finds a node by ID
func (c *arangoClient) FindNode(ctx context.Context, nodeID string) (*Node, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("nodeID cannot be empty")
	}

	logger.Debug("Finding ArangoDB vertex",
		zap.String("id", nodeID),
	)

	aql := fmt.Sprintf("RETURN DOCUMENT('vertices/%s')", nodeID)
	result, err := c.Query(ctx, aql, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find vertex: %w", err)
	}

	if len(result.Records) == 0 {
		return nil, fmt.Errorf("vertex not found")
	}

	record := result.Records[0]
	props := make(map[string]interface{})
	for k, v := range record {
		if k != "_id" && k != "_key" && k != "_rev" {
			props[k] = v
		}
	}

	return &Node{
		ID:         nodeID,
		Labels:     []string{"vertex"}, // ArangoDB doesn't use labels like Neo4j
		Properties: props,
	}, nil
}

// FindNodesByLabel finds nodes by label (collection name in ArangoDB)
func (c *arangoClient) FindNodesByLabel(ctx context.Context, label string, limit int) ([]*Node, error) {
	if label == "" {
		return nil, fmt.Errorf("label cannot be empty")
	}
	if limit <= 0 {
		limit = 100
	}

	logger.Debug("Finding ArangoDB vertices by collection",
		zap.String("collection", label),
		zap.Int("limit", limit),
	)

	aql := fmt.Sprintf("FOR v IN %s LIMIT %d RETURN v", label, limit)
	result, err := c.Query(ctx, aql, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find vertices: %w", err)
	}

	nodes := make([]*Node, 0, len(result.Records))
	for _, record := range result.Records {
		key, ok := record["_key"]
		if !ok {
			continue
		}

		props := make(map[string]interface{})
		for k, v := range record {
			if k != "_id" && k != "_key" && k != "_rev" {
				props[k] = v
			}
		}

		nodes = append(nodes, &Node{
			ID:         fmt.Sprintf("%v", key),
			Labels:     []string{label},
			Properties: props,
		})
	}

	return nodes, nil
}

// toAQLValue converts a Go value to AQL representation
func toAQLValue(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
