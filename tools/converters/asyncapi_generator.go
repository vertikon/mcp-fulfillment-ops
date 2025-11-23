package converters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// AsyncAPIGenerator generates AsyncAPI specifications
type AsyncAPIGenerator struct {
	logger *zap.Logger
}

// NewAsyncAPIGenerator creates a new AsyncAPI generator
func NewAsyncAPIGenerator() *AsyncAPIGenerator {
	return &AsyncAPIGenerator{
		logger: logger.Get(),
	}
}

// GenerateAsyncAPI generates an AsyncAPI specification from a schema definition
func (g *AsyncAPIGenerator) GenerateAsyncAPI(req AsyncAPIGenerateRequest) (*AsyncAPIGenerateResult, error) {
	g.logger.Info("Generating AsyncAPI specification",
		zap.String("title", req.Title),
		zap.String("version", req.Version))

	// Validate request
	if err := g.Validate(req); err != nil {
		return nil, err
	}

	// Build AsyncAPI specification
	spec := g.buildAsyncAPISpec(req)

	// Marshal to JSON
	jsonContent, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AsyncAPI spec: %w", err)
	}

	// Write to file if output path is provided
	if req.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(req.OutputPath, jsonContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to write AsyncAPI spec: %w", err)
		}
	}

	return &AsyncAPIGenerateResult{
		Path:    req.OutputPath,
		Size:    int64(len(jsonContent)),
		Content: string(jsonContent),
		Spec:    spec,
	}, nil
}

// AsyncAPIGenerateRequest represents a request to generate AsyncAPI spec
type AsyncAPIGenerateRequest struct {
	Title       string                 `json:"title"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Broker      string                 `json:"broker,omitempty"`
	Channels    map[string]interface{} `json:"channels,omitempty"`
	Schemas     map[string]interface{} `json:"schemas,omitempty"`
	OutputPath  string                 `json:"output_path,omitempty"`
}

// AsyncAPIGenerateResult represents the result of AsyncAPI generation
type AsyncAPIGenerateResult struct {
	Path    string                 `json:"path"`
	Size    int64                  `json:"size"`
	Content string                 `json:"content"`
	Spec    map[string]interface{} `json:"spec"`
}

// buildAsyncAPISpec builds the AsyncAPI specification structure
func (g *AsyncAPIGenerator) buildAsyncAPISpec(req AsyncAPIGenerateRequest) map[string]interface{} {
	spec := map[string]interface{}{
		"asyncapi": "2.6.0",
		"info": map[string]interface{}{
			"title":       req.Title,
			"version":     req.Version,
			"description": req.Description,
		},
		"servers":  map[string]interface{}{},
		"channels": req.Channels,
	}

	// Add default server if broker is provided
	if req.Broker != "" {
		spec["servers"] = map[string]interface{}{
			"production": map[string]interface{}{
				"url":         req.Broker,
				"protocol":    "nats",
				"description": "NATS broker",
			},
		}
	}

	// Add components with schemas
	if len(req.Schemas) > 0 {
		spec["components"] = map[string]interface{}{
			"schemas": req.Schemas,
		}
	}

	// Add default channels if none provided
	if req.Channels == nil || len(req.Channels) == 0 {
		spec["channels"] = map[string]interface{}{
			"events": map[string]interface{}{
				"publish": map[string]interface{}{
					"message": map[string]interface{}{
						"payload": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"event": map[string]interface{}{
									"type":    "string",
									"example": "example.event",
								},
								"data": map[string]interface{}{
									"type":    "object",
									"example": map[string]interface{}{},
								},
							},
						},
					},
				},
			},
		}
	}

	return spec
}

// Validate validates the AsyncAPI generation request
func (g *AsyncAPIGenerator) Validate(req AsyncAPIGenerateRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if req.Version == "" {
		return fmt.Errorf("version is required")
	}

	return nil
}

// FromJSONSchema converts a JSON Schema to AsyncAPI specification
func (g *AsyncAPIGenerator) FromJSONSchema(jsonSchema map[string]interface{}) (map[string]interface{}, error) {
	if jsonSchema == nil {
		return nil, fmt.Errorf("JSON schema is required")
	}

	// Extract title and version from JSON Schema
	title := "Async API"
	if t, ok := jsonSchema["title"].(string); ok {
		title = t
	}

	version := "1.0.0"
	if v, ok := jsonSchema["version"].(string); ok {
		version = v
	}

	description := ""
	if d, ok := jsonSchema["description"].(string); ok {
		description = d
	}

	// Convert properties to AsyncAPI schemas
	schemas := make(map[string]interface{})
	if props, ok := jsonSchema["properties"].(map[string]interface{}); ok {
		schemaName := title
		schemas[schemaName] = map[string]interface{}{
			"type":       "object",
			"properties": props,
		}

		if required, ok := jsonSchema["required"].([]interface{}); ok {
			schemas[schemaName].(map[string]interface{})["required"] = required
		}
	}

	req := AsyncAPIGenerateRequest{
		Title:       title,
		Version:     version,
		Description: description,
		Broker:      "nats://localhost:4222",
		Schemas:     schemas,
	}

	return g.buildAsyncAPISpec(req), nil
}
