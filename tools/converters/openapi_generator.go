package converters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// OpenAPIGenerator generates OpenAPI specifications
type OpenAPIGenerator struct {
	logger *zap.Logger
}

// NewOpenAPIGenerator creates a new OpenAPI generator
func NewOpenAPIGenerator() *OpenAPIGenerator {
	return &OpenAPIGenerator{
		logger: logger.Get(),
	}
}

// GenerateOpenAPI generates an OpenAPI specification from a schema definition
func (g *OpenAPIGenerator) GenerateOpenAPI(req OpenAPIGenerateRequest) (*OpenAPIGenerateResult, error) {
	g.logger.Info("Generating OpenAPI specification",
		zap.String("title", req.Title),
		zap.String("version", req.Version))

	// Validate request
	if err := g.Validate(req); err != nil {
		return nil, err
	}

	// Build OpenAPI specification
	spec := g.buildOpenAPISpec(req)

	// Marshal to JSON
	jsonContent, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAPI spec: %w", err)
	}

	// Write to file if output path is provided
	if req.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(req.OutputPath, jsonContent, 0644); err != nil {
			return nil, fmt.Errorf("failed to write OpenAPI spec: %w", err)
		}
	}

	return &OpenAPIGenerateResult{
		Path:    req.OutputPath,
		Size:    int64(len(jsonContent)),
		Content: string(jsonContent),
		Spec:    spec,
	}, nil
}

// OpenAPIGenerateRequest represents a request to generate OpenAPI spec
type OpenAPIGenerateRequest struct {
	Title       string                 `json:"title"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	BaseURL     string                 `json:"base_url,omitempty"`
	Paths       map[string]interface{} `json:"paths,omitempty"`
	Schemas     map[string]interface{} `json:"schemas,omitempty"`
	OutputPath  string                 `json:"output_path,omitempty"`
}

// OpenAPIGenerateResult represents the result of OpenAPI generation
type OpenAPIGenerateResult struct {
	Path    string                 `json:"path"`
	Size    int64                  `json:"size"`
	Content string                 `json:"content"`
	Spec    map[string]interface{} `json:"spec"`
}

// buildOpenAPISpec builds the OpenAPI specification structure
func (g *OpenAPIGenerator) buildOpenAPISpec(req OpenAPIGenerateRequest) map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       req.Title,
			"version":     req.Version,
			"description": req.Description,
		},
		"paths":   req.Paths,
		"servers": []map[string]interface{}{},
	}

	// Add server if base URL is provided
	if req.BaseURL != "" {
		spec["servers"] = []map[string]interface{}{
			{
				"url":         req.BaseURL,
				"description": "API Server",
			},
		}
	}

	// Add components with schemas
	if len(req.Schemas) > 0 {
		spec["components"] = map[string]interface{}{
			"schemas": req.Schemas,
		}
	}

	// Add default paths if none provided
	if req.Paths == nil || len(req.Paths) == 0 {
		spec["paths"] = map[string]interface{}{
			"/": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check",
					"description": "Returns the health status of the API",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "OK",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type":    "string",
												"example": "ok",
											},
										},
									},
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

// Validate validates the OpenAPI generation request
func (g *OpenAPIGenerator) Validate(req OpenAPIGenerateRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if req.Version == "" {
		return fmt.Errorf("version is required")
	}

	return nil
}

// FromJSONSchema converts a JSON Schema to OpenAPI specification
func (g *OpenAPIGenerator) FromJSONSchema(jsonSchema map[string]interface{}) (map[string]interface{}, error) {
	if jsonSchema == nil {
		return nil, fmt.Errorf("JSON schema is required")
	}

	// Extract title and version from JSON Schema
	title := "API"
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

	// Convert properties to OpenAPI schemas
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

	req := OpenAPIGenerateRequest{
		Title:       title,
		Version:     version,
		Description: description,
		Schemas:     schemas,
	}

	return g.buildOpenAPISpec(req), nil
}
