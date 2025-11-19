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

// GraphClient provides graph database operations
type GraphClient interface {
	// CreateNode creates a node in the graph
	CreateNode(ctx context.Context, label string, properties map[string]interface{}) (string, error)

	// CreateRelationship creates a relationship between nodes
	CreateRelationship(ctx context.Context, fromID string, toID string, relType string, properties map[string]interface{}) error

	// Query executes a Cypher query
	Query(ctx context.Context, cypher string, params map[string]interface{}) (*QueryResult, error)

	// DeleteNode deletes a node by ID
	DeleteNode(ctx context.Context, nodeID string) error

	// DeleteRelationship deletes a relationship by ID
	DeleteRelationship(ctx context.Context, relID string) error

	// FindNode finds a node by ID
	FindNode(ctx context.Context, nodeID string) (*Node, error)

	// FindNodesByLabel finds nodes by label
	FindNodesByLabel(ctx context.Context, label string, limit int) ([]*Node, error)
}

// Node represents a graph node
type Node struct {
	ID         string
	Labels     []string
	Properties map[string]interface{}
}

// Relationship represents a graph relationship
type Relationship struct {
	ID         string
	Type       string
	FromID     string
	ToID       string
	Properties map[string]interface{}
}

// QueryResult represents a Cypher query result
type QueryResult struct {
	Nodes         []*Node
	Relationships []*Relationship
	Records       []map[string]interface{}
}

// neo4jClient implements GraphClient using Neo4j
type neo4jClient struct {
	uri      string
	username string
	password string
	timeout  time.Duration
	client   *http.Client
}

// NewNeo4jClient creates a new Neo4j client
func NewNeo4jClient(uri string, username string, password string, timeout time.Duration) GraphClient {
	if uri == "" {
		uri = "http://localhost:7474"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &neo4jClient{
		uri:      uri,
		username: username,
		password: password,
		timeout:  timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateNode creates a node in the graph
func (c *neo4jClient) CreateNode(ctx context.Context, label string, properties map[string]interface{}) (string, error) {
	if label == "" {
		return "", fmt.Errorf("label cannot be empty")
	}

	logger.Info("Creating Neo4j node",
		zap.String("label", label),
	)

	if properties == nil {
		properties = make(map[string]interface{})
	}

	cypher := fmt.Sprintf("CREATE (n:%s $props) RETURN id(n) as id", label)
	params := map[string]interface{}{
		"props": properties,
	}

	result, err := c.Query(ctx, cypher, params)
	if err != nil {
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	if len(result.Records) == 0 || len(result.Records[0]) == 0 {
		return "", fmt.Errorf("failed to get node ID")
	}

	id, ok := result.Records[0]["id"]
	if !ok {
		return "", fmt.Errorf("failed to get node ID from result")
	}

	idStr := fmt.Sprintf("%v", id)
	return idStr, nil
}

// CreateRelationship creates a relationship between nodes
func (c *neo4jClient) CreateRelationship(ctx context.Context, fromID string, toID string, relType string, properties map[string]interface{}) error {
	if fromID == "" || toID == "" {
		return fmt.Errorf("fromID and toID cannot be empty")
	}
	if relType == "" {
		return fmt.Errorf("relationship type cannot be empty")
	}

	logger.Info("Creating Neo4j relationship",
		zap.String("from", fromID),
		zap.String("to", toID),
		zap.String("type", relType),
	)

	if properties == nil {
		properties = make(map[string]interface{})
	}

	cypher := fmt.Sprintf("MATCH (a), (b) WHERE id(a) = $fromID AND id(b) = $toID CREATE (a)-[r:%s $props]->(b) RETURN id(r) as id", relType)
	params := map[string]interface{}{
		"fromID": fromID,
		"toID":   toID,
		"props":  properties,
	}

	_, err := c.Query(ctx, cypher, params)
	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// Query executes a Cypher query
func (c *neo4jClient) Query(ctx context.Context, cypher string, params map[string]interface{}) (*QueryResult, error) {
	if cypher == "" {
		return nil, fmt.Errorf("cypher query cannot be empty")
	}

	logger.Debug("Executing Cypher query",
		zap.String("query", cypher),
	)

	if params == nil {
		params = make(map[string]interface{})
	}

	payload := map[string]interface{}{
		"statements": []map[string]interface{}{
			{
				"statement":  cypher,
				"parameters": params,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/db/data/transaction/commit", c.uri)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
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
		Results []struct {
			Columns []string `json:"columns"`
			Data    []struct {
				Row []interface{} `json:"row"`
			} `json:"data"`
		} `json:"results"`
		Errors []map[string]interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("query error: %v", result.Errors)
	}

	if len(result.Results) == 0 {
		return &QueryResult{
			Nodes:         []*Node{},
			Relationships: []*Relationship{},
			Records:       []map[string]interface{}{},
		}, nil
	}

	res := result.Results[0]
	records := make([]map[string]interface{}, len(res.Data))
	for i, data := range res.Data {
		record := make(map[string]interface{})
		for j, col := range res.Columns {
			if j < len(data.Row) {
				record[col] = data.Row[j]
			}
		}
		records[i] = record
	}

	return &QueryResult{
		Nodes:         []*Node{},
		Relationships: []*Relationship{},
		Records:       records,
	}, nil
}

// DeleteNode deletes a node by ID
func (c *neo4jClient) DeleteNode(ctx context.Context, nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID cannot be empty")
	}

	logger.Info("Deleting Neo4j node",
		zap.String("id", nodeID),
	)

	cypher := "MATCH (n) WHERE id(n) = $id DETACH DELETE n"
	params := map[string]interface{}{
		"id": nodeID,
	}

	_, err := c.Query(ctx, cypher, params)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// DeleteRelationship deletes a relationship by ID
func (c *neo4jClient) DeleteRelationship(ctx context.Context, relID string) error {
	if relID == "" {
		return fmt.Errorf("relID cannot be empty")
	}

	logger.Info("Deleting Neo4j relationship",
		zap.String("id", relID),
	)

	cypher := "MATCH ()-[r]->() WHERE id(r) = $id DELETE r"
	params := map[string]interface{}{
		"id": relID,
	}

	_, err := c.Query(ctx, cypher, params)
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	return nil
}

// FindNode finds a node by ID
func (c *neo4jClient) FindNode(ctx context.Context, nodeID string) (*Node, error) {
	if nodeID == "" {
		return nil, fmt.Errorf("nodeID cannot be empty")
	}

	logger.Debug("Finding Neo4j node",
		zap.String("id", nodeID),
	)

	cypher := "MATCH (n) WHERE id(n) = $id RETURN labels(n) as labels, properties(n) as props"
	params := map[string]interface{}{
		"id": nodeID,
	}

	result, err := c.Query(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find node: %w", err)
	}

	if len(result.Records) == 0 {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	record := result.Records[0]
	labels, ok := record["labels"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to get labels")
	}

	props, ok := record["props"].(map[string]interface{})
	if !ok {
		props = make(map[string]interface{})
	}

	labelStrs := make([]string, len(labels))
	for i, l := range labels {
		labelStrs[i] = fmt.Sprintf("%v", l)
	}

	return &Node{
		ID:         nodeID,
		Labels:     labelStrs,
		Properties: props,
	}, nil
}

// FindNodesByLabel finds nodes by label
func (c *neo4jClient) FindNodesByLabel(ctx context.Context, label string, limit int) ([]*Node, error) {
	if label == "" {
		return nil, fmt.Errorf("label cannot be empty")
	}
	if limit <= 0 {
		limit = 100
	}

	logger.Debug("Finding Neo4j nodes by label",
		zap.String("label", label),
		zap.Int("limit", limit),
	)

	cypher := fmt.Sprintf("MATCH (n:%s) RETURN id(n) as id, labels(n) as labels, properties(n) as props LIMIT $limit", label)
	params := map[string]interface{}{
		"limit": limit,
	}

	result, err := c.Query(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find nodes: %w", err)
	}

	nodes := make([]*Node, len(result.Records))
	for i, record := range result.Records {
		id, ok := record["id"]
		if !ok {
			continue
		}

		labels, ok := record["labels"].([]interface{})
		if !ok {
			labels = []interface{}{}
		}

		props, ok := record["props"].(map[string]interface{})
		if !ok {
			props = make(map[string]interface{})
		}

		labelStrs := make([]string, len(labels))
		for j, l := range labels {
			labelStrs[j] = fmt.Sprintf("%v", l)
		}

		nodes[i] = &Node{
			ID:         fmt.Sprintf("%v", id),
			Labels:     labelStrs,
			Properties: props,
		}
	}

	return nodes, nil
}
