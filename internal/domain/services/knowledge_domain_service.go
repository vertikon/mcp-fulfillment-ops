// Package services provides domain services
package services

import (
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// KnowledgeDomainService provides domain logic for Knowledge operations
type KnowledgeDomainService struct{}

// NewKnowledgeDomainService creates a new Knowledge domain service
func NewKnowledgeDomainService() *KnowledgeDomainService {
	return &KnowledgeDomainService{}
}

// ValidateKnowledge validates Knowledge business rules
func (s *KnowledgeDomainService) ValidateKnowledge(knowledge *entities.Knowledge) error {
	if knowledge == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}

	if knowledge.Name() == "" {
		return fmt.Errorf("knowledge name cannot be empty")
	}

	// Business rule: Knowledge must have at least one document
	documents := knowledge.Documents()
	if len(documents) == 0 {
		return fmt.Errorf("knowledge must have at least one document")
	}

	return nil
}

// CanAddDocument checks if a document can be added to knowledge
func (s *KnowledgeDomainService) CanAddDocument(knowledge *entities.Knowledge, content string) error {
	if knowledge == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}
	if content == "" {
		return fmt.Errorf("document content cannot be empty")
	}

	return nil
}

// CanAddEmbedding checks if an embedding can be added
func (s *KnowledgeDomainService) CanAddEmbedding(knowledge *entities.Knowledge, documentID string, vector []float64) error {
	if knowledge == nil {
		return fmt.Errorf("knowledge cannot be nil")
	}
	if documentID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}
	if len(vector) == 0 {
		return fmt.Errorf("embedding vector cannot be empty")
	}

	// Verify document exists
	documents := knowledge.Documents()
	found := false
	for _, doc := range documents {
		if doc.ID() == documentID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("document %s not found", documentID)
	}

	return nil
}

// ShouldIncrementVersion determines if knowledge version should be incremented
func (s *KnowledgeDomainService) ShouldIncrementVersion(knowledge *entities.Knowledge, hasStructuralChanges bool) bool {
	if knowledge == nil {
		return false
	}

	// Business rule: Increment version only on structural changes
	return hasStructuralChanges
}
