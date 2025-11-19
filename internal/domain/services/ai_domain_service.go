// Package services provides domain services
package services

import (
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// AIDomainService provides domain logic for AI-related operations
type AIDomainService struct{}

// NewAIDomainService creates a new AI domain service
func NewAIDomainService() *AIDomainService {
	return &AIDomainService{}
}

// ValidateKnowledgeContext validates knowledge context for AI operations
func (s *AIDomainService) ValidateKnowledgeContext(mcp *entities.MCP) error {
	if mcp == nil {
		return fmt.Errorf("MCP cannot be nil")
	}

	if !mcp.HasContext() {
		return fmt.Errorf("MCP does not have knowledge context")
	}

	context := mcp.Context()
	if context == nil {
		return fmt.Errorf("knowledge context is nil")
	}

	if len(context.documents) == 0 {
		return fmt.Errorf("knowledge context must have at least one document")
	}

	return nil
}

// CanUseKnowledgeForInference checks if knowledge can be used for inference
func (s *AIDomainService) CanUseKnowledgeForInference(knowledge *entities.Knowledge) error {
	if knowledge == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}

	documents := knowledge.Documents()
	if len(documents) == 0 {
		return fmt.Errorf("knowledge must have documents for inference")
	}

	embeddings := knowledge.Embeddings()
	if len(embeddings) == 0 {
		return fmt.Errorf("knowledge must have embeddings for inference")
	}

	return nil
}
