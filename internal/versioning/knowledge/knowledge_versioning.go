package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// KnowledgeVersion represents a versioned knowledge base
type KnowledgeVersion struct {
	ID            string                 `json:"id"`
	KnowledgeID   string                 `json:"knowledge_id"`
	Version       string                 `json:"version"`
	DocumentCount int                    `json:"document_count"`
	EmbeddingCount int                   `json:"embedding_count"`
	Checksum      string                 `json:"checksum"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Tags          []string               `json:"tags"`
}

// KnowledgeDocument represents a versioned document
type KnowledgeDocument struct {
	ID          string                 `json:"id"`
	VersionID  string                 `json:"version_id"`
	Content    string                 `json:"content"`
	Embedding  []float32              `json:"embedding,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// KnowledgeVersioning interface for knowledge versioning operations
type KnowledgeVersioning interface {
	// CreateVersion creates a new version of a knowledge base
	CreateVersion(ctx context.Context, knowledgeID string, metadata map[string]interface{}) (*KnowledgeVersion, error)
	
	// GetVersion retrieves a specific version
	GetVersion(ctx context.Context, versionID string) (*KnowledgeVersion, error)
	
	// ListVersions lists all versions for a knowledge base
	ListVersions(ctx context.Context, knowledgeID string) ([]*KnowledgeVersion, error)
	
	// AddDocument adds a document to a version
	AddDocument(ctx context.Context, versionID string, doc *KnowledgeDocument) error
	
	// GetDocument retrieves a document from a version
	GetDocument(ctx context.Context, versionID string, documentID string) (*KnowledgeDocument, error)
	
	// ListDocuments lists all documents in a version
	ListDocuments(ctx context.Context, versionID string) ([]*KnowledgeDocument, error)
	
	// DeleteVersion deletes a version (soft delete)
	DeleteVersion(ctx context.Context, versionID string) error
	
	// GetLatestVersion gets the latest version for a knowledge base
	GetLatestVersion(ctx context.Context, knowledgeID string) (*KnowledgeVersion, error)
	
	// TagVersion tags a version with labels
	TagVersion(ctx context.Context, versionID string, tags []string) error
}

// InMemoryKnowledgeVersioning implements KnowledgeVersioning in memory
type InMemoryKnowledgeVersioning struct {
	versions  map[string]*KnowledgeVersion
	documents map[string][]*KnowledgeDocument // versionID -> documents
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewInMemoryKnowledgeVersioning creates a new in-memory knowledge versioning instance
func NewInMemoryKnowledgeVersioning() *InMemoryKnowledgeVersioning {
	return &InMemoryKnowledgeVersioning{
		versions:  make(map[string]*KnowledgeVersion),
		documents: make(map[string][]*KnowledgeDocument),
		logger:    logger.WithContext(context.Background()),
	}
}

// CreateVersion creates a new version of a knowledge base
func (kv *InMemoryKnowledgeVersioning) CreateVersion(ctx context.Context, knowledgeID string, metadata map[string]interface{}) (*KnowledgeVersion, error) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	logger := logger.WithContext(ctx)
	logger.Info("Creating new knowledge version", zap.String("knowledge_id", knowledgeID))

	// Get latest version to increment
	latestVersion := "v1"
	var maxVersion int
	for _, v := range kv.versions {
		if v.KnowledgeID == knowledgeID {
			var vNum int
			fmt.Sscanf(v.Version, "v%d", &vNum)
			if vNum > maxVersion {
				maxVersion = vNum
			}
		}
	}
	if maxVersion > 0 {
		latestVersion = fmt.Sprintf("v%d", maxVersion+1)
	}

	versionID := uuid.New().String()
	now := time.Now()

	version := &KnowledgeVersion{
		ID:            versionID,
		KnowledgeID:   knowledgeID,
		Version:       latestVersion,
		DocumentCount: 0,
		EmbeddingCount: 0,
		Checksum:      "",
		Metadata:      metadata,
		CreatedAt:     now,
		CreatedBy:     getCurrentUser(ctx),
		Tags:          []string{},
	}

	kv.versions[versionID] = version
	kv.documents[versionID] = []*KnowledgeDocument{}

	logger.Info("Knowledge version created", 
		zap.String("version_id", versionID),
		zap.String("version", latestVersion))

	return version, nil
}

// GetVersion retrieves a specific version
func (kv *InMemoryKnowledgeVersioning) GetVersion(ctx context.Context, versionID string) (*KnowledgeVersion, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	version, exists := kv.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	return version, nil
}

// ListVersions lists all versions for a knowledge base
func (kv *InMemoryKnowledgeVersioning) ListVersions(ctx context.Context, knowledgeID string) ([]*KnowledgeVersion, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	var versions []*KnowledgeVersion
	for _, v := range kv.versions {
		if v.KnowledgeID == knowledgeID {
			versions = append(versions, v)
		}
	}

	return versions, nil
}

// AddDocument adds a document to a version
func (kv *InMemoryKnowledgeVersioning) AddDocument(ctx context.Context, versionID string, doc *KnowledgeDocument) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	logger := logger.WithContext(ctx)

	version, exists := kv.versions[versionID]
	if !exists {
		return fmt.Errorf("version %s not found", versionID)
	}

	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.VersionID = versionID
	now := time.Now()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	kv.documents[versionID] = append(kv.documents[versionID], doc)
	version.DocumentCount++

	if len(doc.Embedding) > 0 {
		version.EmbeddingCount++
	}

	// Update checksum
	version.Checksum = kv.calculateChecksum(versionID)

	logger.Info("Document added to version",
		zap.String("version_id", versionID),
		zap.String("document_id", doc.ID))

	return nil
}

// GetDocument retrieves a document from a version
func (kv *InMemoryKnowledgeVersioning) GetDocument(ctx context.Context, versionID string, documentID string) (*KnowledgeDocument, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	docs, exists := kv.documents[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	for _, doc := range docs {
		if doc.ID == documentID {
			return doc, nil
		}
	}

	return nil, fmt.Errorf("document %s not found in version %s", documentID, versionID)
}

// ListDocuments lists all documents in a version
func (kv *InMemoryKnowledgeVersioning) ListDocuments(ctx context.Context, versionID string) ([]*KnowledgeDocument, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	docs, exists := kv.documents[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	// Return a copy
	result := make([]*KnowledgeDocument, len(docs))
	copy(result, docs)
	return result, nil
}

// DeleteVersion deletes a version (soft delete)
func (kv *InMemoryKnowledgeVersioning) DeleteVersion(ctx context.Context, versionID string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	logger := logger.WithContext(ctx)

	version, exists := kv.versions[versionID]
	if !exists {
		return fmt.Errorf("version %s not found", versionID)
	}

	// Soft delete: mark as deleted in metadata
	if version.Metadata == nil {
		version.Metadata = make(map[string]interface{})
	}
	version.Metadata["deleted"] = true
	version.Metadata["deleted_at"] = time.Now()

	logger.Info("Version soft deleted", zap.String("version_id", versionID))
	return nil
}

// GetLatestVersion gets the latest version for a knowledge base
func (kv *InMemoryKnowledgeVersioning) GetLatestVersion(ctx context.Context, knowledgeID string) (*KnowledgeVersion, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	var latest *KnowledgeVersion
	var maxVersion int

	for _, v := range kv.versions {
		if v.KnowledgeID == knowledgeID {
			// Check if deleted
			if deleted, ok := v.Metadata["deleted"].(bool); ok && deleted {
				continue
			}

			var vNum int
			fmt.Sscanf(v.Version, "v%d", &vNum)
			if vNum > maxVersion {
				maxVersion = vNum
				latest = v
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no versions found for knowledge base %s", knowledgeID)
	}

	return latest, nil
}

// TagVersion tags a version with labels
func (kv *InMemoryKnowledgeVersioning) TagVersion(ctx context.Context, versionID string, tags []string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	logger := logger.WithContext(ctx)

	version, exists := kv.versions[versionID]
	if !exists {
		return fmt.Errorf("version %s not found", versionID)
	}

	version.Tags = append(version.Tags, tags...)
	
	// Remove duplicates
	seen := make(map[string]bool)
	result := []string{}
	for _, tag := range version.Tags {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	version.Tags = result

	logger.Info("Version tagged", 
		zap.String("version_id", versionID),
		zap.Strings("tags", tags))

	return nil
}

// calculateChecksum calculates checksum for a version
func (kv *InMemoryKnowledgeVersioning) calculateChecksum(versionID string) string {
	docs := kv.documents[versionID]
	if len(docs) == 0 {
		return ""
	}

	hasher := sha256.New()
	for _, doc := range docs {
		hasher.Write([]byte(doc.ID))
		hasher.Write([]byte(doc.Content))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// getCurrentUser extracts user from context (placeholder)
func getCurrentUser(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		return userID.(string)
	}
	return "system"
}
