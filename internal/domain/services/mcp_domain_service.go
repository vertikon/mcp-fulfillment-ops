// Package services provides domain services
package services

import (
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/value_objects"
)

// MCPDomainService provides domain logic for MCP operations
type MCPDomainService struct{}

// NewMCPDomainService creates a new MCP domain service
func NewMCPDomainService() *MCPDomainService {
	return &MCPDomainService{}
}

// ValidateMCP validates MCP business rules
func (s *MCPDomainService) ValidateMCP(mcp *entities.MCP) error {
	if mcp == nil {
		return fmt.Errorf("MCP cannot be nil")
	}

	if mcp.Name() == "" {
		return fmt.Errorf("MCP name cannot be empty")
	}

	if mcp.Path() == "" {
		return fmt.Errorf("MCP path cannot be empty")
	}

	if !mcp.Stack().IsValid() {
		return fmt.Errorf("MCP stack must be valid")
	}

	// Validate features don't have conflicts
	features := mcp.Features()
	featureNames := make(map[string]bool)
	for _, feature := range features {
		if featureNames[feature.Name()] {
			return fmt.Errorf("duplicate feature name: %s", feature.Name())
		}
		featureNames[feature.Name()] = true
	}

	return nil
}

// CanAddFeature checks if a feature can be added to an MCP
func (s *MCPDomainService) CanAddFeature(mcp *entities.MCP, feature *value_objects.Feature) error {
	if mcp == nil {
		return fmt.Errorf("MCP cannot be nil")
	}
	if feature == nil {
		return fmt.Errorf("feature cannot be nil")
	}

	// Check for duplicates
	features := mcp.Features()
	for _, f := range features {
		if f.Equals(feature) {
			return fmt.Errorf("feature %s already exists", feature.Name())
		}
	}

	return nil
}

// CanAttachContext checks if knowledge context can be attached
func (s *MCPDomainService) CanAttachContext(mcp *entities.MCP, knowledgeID string) error {
	if mcp == nil {
		return fmt.Errorf("MCP cannot be nil")
	}
	if knowledgeID == "" {
		return fmt.Errorf("knowledge ID cannot be empty")
	}

	// Business rule: MCP can only have one context at a time
	if mcp.HasContext() {
		return fmt.Errorf("MCP already has a knowledge context attached")
	}

	return nil
}
