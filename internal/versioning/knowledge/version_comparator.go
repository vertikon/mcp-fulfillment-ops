package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// VersionDiff represents differences between two versions
type VersionDiff struct {
	AddedDocuments   []string               `json:"added_documents"`
	RemovedDocuments []string               `json:"removed_documents"`
	ModifiedDocuments []DocumentChange      `json:"modified_documents"`
	MetadataChanges  map[string]interface{} `json:"metadata_changes"`
	SemanticSimilarity float64             `json:"semantic_similarity"`
	StructuralSimilarity float64            `json:"structural_similarity"`
}

// DocumentChange represents changes to a document
type DocumentChange struct {
	DocumentID string                 `json:"document_id"`
	Changes    map[string]interface{} `json:"changes"`
	OldContent string                 `json:"old_content,omitempty"`
	NewContent string                 `json:"new_content,omitempty"`
}

// VersionComparator interface for comparing knowledge versions
type VersionComparator interface {
	// CompareVersions compares two versions and returns differences
	CompareVersions(ctx context.Context, versionID1, versionID2 string) (*VersionDiff, error)
	
	// CompareSemantic compares semantic similarity between versions
	CompareSemantic(ctx context.Context, versionID1, versionID2 string) (float64, error)
	
	// CompareStructural compares structural similarity between versions
	CompareStructural(ctx context.Context, versionID1, versionID2 string) (float64, error)
	
	// GetDiffSummary returns a human-readable summary of differences
	GetDiffSummary(ctx context.Context, diff *VersionDiff) string
}

// InMemoryVersionComparator implements VersionComparator
type InMemoryVersionComparator struct {
	versioning KnowledgeVersioning
	logger     *zap.Logger
}

// NewInMemoryVersionComparator creates a new version comparator
func NewInMemoryVersionComparator(versioning KnowledgeVersioning) *InMemoryVersionComparator {
	return &InMemoryVersionComparator{
		versioning: versioning,
		logger:     logger.WithContext(context.Background()),
	}
}

// CompareVersions compares two versions and returns differences
func (vc *InMemoryVersionComparator) CompareVersions(ctx context.Context, versionID1, versionID2 string) (*VersionDiff, error) {
	logger := logger.WithContext(ctx)
	logger.Info("Comparing versions",
		zap.String("version_id1", versionID1),
		zap.String("version_id2", versionID2))

	// Get both versions
	version1, err := vc.versioning.GetVersion(ctx, versionID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %s: %w", versionID1, err)
	}

	version2, err := vc.versioning.GetVersion(ctx, versionID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %s: %w", versionID2, err)
	}

	// Get documents for both versions
	docs1, err := vc.versioning.ListDocuments(ctx, versionID1)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for version %s: %w", versionID1, err)
	}

	docs2, err := vc.versioning.ListDocuments(ctx, versionID2)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for version %s: %w", versionID2, err)
	}

	// Build document maps
	docMap1 := make(map[string]*KnowledgeDocument)
	for _, doc := range docs1 {
		docMap1[doc.ID] = doc
	}

	docMap2 := make(map[string]*KnowledgeDocument)
	for _, doc := range docs2 {
		docMap2[doc.ID] = doc
	}

	diff := &VersionDiff{
		AddedDocuments:    []string{},
		RemovedDocuments:  []string{},
		ModifiedDocuments: []DocumentChange{},
		MetadataChanges:   make(map[string]interface{}),
	}

	// Find added documents
	for id := range docMap2 {
		if _, exists := docMap1[id]; !exists {
			diff.AddedDocuments = append(diff.AddedDocuments, id)
		}
	}

	// Find removed documents
	for id := range docMap1 {
		if _, exists := docMap2[id]; !exists {
			diff.RemovedDocuments = append(diff.RemovedDocuments, id)
		}
	}

	// Find modified documents
	for id, doc1 := range docMap1 {
		if doc2, exists := docMap2[id]; exists {
			changes := make(map[string]interface{})
			hasChanges := false

			if doc1.Content != doc2.Content {
				changes["content"] = true
				hasChanges = true
			}

			if len(doc1.Embedding) != len(doc2.Embedding) {
				changes["embedding"] = true
				hasChanges = true
			}

			// Compare metadata
			if !compareMetadata(doc1.Metadata, doc2.Metadata) {
				changes["metadata"] = true
				hasChanges = true
			}

			if hasChanges {
				diff.ModifiedDocuments = append(diff.ModifiedDocuments, DocumentChange{
					DocumentID: id,
					Changes:    changes,
					OldContent: doc1.Content,
					NewContent: doc2.Content,
				})
			}
		}
	}

	// Compare metadata
	if !compareMetadata(version1.Metadata, version2.Metadata) {
		diff.MetadataChanges["version_metadata"] = true
	}

	// Calculate similarities
	semantic, _ := vc.CompareSemantic(ctx, versionID1, versionID2)
	structural, _ := vc.CompareStructural(ctx, versionID1, versionID2)

	diff.SemanticSimilarity = semantic
	diff.StructuralSimilarity = structural

	return diff, nil
}

// CompareSemantic compares semantic similarity between versions
func (vc *InMemoryVersionComparator) CompareSemantic(ctx context.Context, versionID1, versionID2 string) (float64, error) {
	docs1, err := vc.versioning.ListDocuments(ctx, versionID1)
	if err != nil {
		return 0, err
	}

	docs2, err := vc.versioning.ListDocuments(ctx, versionID2)
	if err != nil {
		return 0, err
	}

	if len(docs1) == 0 && len(docs2) == 0 {
		return 1.0, nil
	}

	if len(docs1) == 0 || len(docs2) == 0 {
		return 0.0, nil
	}

	// Build document maps
	docMap1 := make(map[string]*KnowledgeDocument)
	for _, doc := range docs1 {
		docMap1[doc.ID] = doc
	}

	docMap2 := make(map[string]*KnowledgeDocument)
	for _, doc := range docs2 {
		docMap2[doc.ID] = doc
	}

	// Calculate Jaccard similarity based on document IDs
	intersection := 0
	union := len(docMap1) + len(docMap2)

	for id := range docMap1 {
		if _, exists := docMap2[id]; exists {
			intersection++
			union--
		}
	}

	if union == 0 {
		return 1.0, nil
	}

	similarity := float64(intersection) / float64(union)
	return similarity, nil
}

// CompareStructural compares structural similarity between versions
func (vc *InMemoryVersionComparator) CompareStructural(ctx context.Context, versionID1, versionID2 string) (float64, error) {
	version1, err := vc.versioning.GetVersion(ctx, versionID1)
	if err != nil {
		return 0, err
	}

	version2, err := vc.versioning.GetVersion(ctx, versionID2)
	if err != nil {
		return 0, err
	}

	// Compare document counts
	docCount1 := version1.DocumentCount
	docCount2 := version2.DocumentCount

	if docCount1 == 0 && docCount2 == 0 {
		return 1.0, nil
	}

	if docCount1 == 0 || docCount2 == 0 {
		return 0.0, nil
	}

	// Calculate similarity based on document count ratio
	var ratio float64
	if docCount1 > docCount2 {
		ratio = float64(docCount2) / float64(docCount1)
	} else {
		ratio = float64(docCount1) / float64(docCount2)
	}

	return ratio, nil
}

// GetDiffSummary returns a human-readable summary of differences
func (vc *InMemoryVersionComparator) GetDiffSummary(ctx context.Context, diff *VersionDiff) string {
	var parts []string

	if len(diff.AddedDocuments) > 0 {
		parts = append(parts, fmt.Sprintf("Added %d documents", len(diff.AddedDocuments)))
	}

	if len(diff.RemovedDocuments) > 0 {
		parts = append(parts, fmt.Sprintf("Removed %d documents", len(diff.RemovedDocuments)))
	}

	if len(diff.ModifiedDocuments) > 0 {
		parts = append(parts, fmt.Sprintf("Modified %d documents", len(diff.ModifiedDocuments)))
	}

	if len(diff.MetadataChanges) > 0 {
		parts = append(parts, "Metadata changed")
	}

	if len(parts) == 0 {
		return "No differences found"
	}

	summary := strings.Join(parts, ", ")
	summary += fmt.Sprintf(" (Semantic: %.2f%%, Structural: %.2f%%)",
		diff.SemanticSimilarity*100,
		diff.StructuralSimilarity*100)

	return summary
}

// compareMetadata compares two metadata maps
func compareMetadata(m1, m2 map[string]interface{}) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v1 := range m1 {
		if v2, exists := m2[k]; !exists || v1 != v2 {
			return false
		}
	}

	return true
}
