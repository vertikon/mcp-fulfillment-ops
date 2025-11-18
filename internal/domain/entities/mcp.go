// Package entities provides domain entities
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-hulk/internal/domain/value_objects"
)

// MCP represents a Model Context Protocol entity
type MCP struct {
	id          string
	name        string
	description string
	stack       value_objects.StackType
	path        string
	features    []*value_objects.Feature
	context     *KnowledgeContext
	createdAt   time.Time
	updatedAt   time.Time
}

// KnowledgeContext represents the knowledge context attached to an MCP
type KnowledgeContext struct {
	knowledgeID string
	documents   []string
	embeddings  map[string][]float64
	metadata    map[string]interface{}
}

// NewMCP creates a new MCP entity
func NewMCP(name string, description string, stack value_objects.StackType) (*MCP, error) {
	if name == "" {
		return nil, fmt.Errorf("MCP name cannot be empty")
	}
	if !stack.IsValid() {
		return nil, fmt.Errorf("invalid stack type: %s", stack)
	}

	now := time.Now()
	return &MCP{
		id:          uuid.New().String(),
		name:        name,
		description: description,
		stack:       stack,
		path:        "",
		features:    make([]*value_objects.Feature, 0),
		context:     nil,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ID returns the MCP ID
func (m *MCP) ID() string {
	return m.id
}

// Name returns the MCP name
func (m *MCP) Name() string {
	return m.name
}

// Description returns the MCP description
func (m *MCP) Description() string {
	return m.description
}

// Stack returns the stack type
func (m *MCP) Stack() value_objects.StackType {
	return m.stack
}

// Path returns the MCP path
func (m *MCP) Path() string {
	return m.path
}

// Features returns a copy of the features list
func (m *MCP) Features() []*value_objects.Feature {
	features := make([]*value_objects.Feature, len(m.features))
	copy(features, m.features)
	return features
}

// Context returns the knowledge context
func (m *MCP) Context() *KnowledgeContext {
	if m.context == nil {
		return nil
	}
	// Return a copy to maintain immutability
	return &KnowledgeContext{
		knowledgeID: m.context.knowledgeID,
		documents:   append([]string{}, m.context.documents...),
		embeddings:  copyEmbeddings(m.context.embeddings),
		metadata:    copyMetadataMCP(m.context.metadata),
	}
}

// KnowledgeID returns the knowledge ID from context
func (kc *KnowledgeContext) KnowledgeID() string {
	return kc.knowledgeID
}

// Documents returns a copy of documents from context
func (kc *KnowledgeContext) Documents() []string {
	return append([]string{}, kc.documents...)
}

// Embeddings returns a copy of embeddings from context
func (kc *KnowledgeContext) Embeddings() map[string][]float64 {
	return copyEmbeddings(kc.embeddings)
}

// Metadata returns a copy of metadata from context
func (kc *KnowledgeContext) Metadata() map[string]interface{} {
	return copyMetadataMCP(kc.metadata)
}

// CreatedAt returns the creation timestamp
func (m *MCP) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the last update timestamp
func (m *MCP) UpdatedAt() time.Time {
	return m.updatedAt
}

// SetPath sets the MCP path and updates the timestamp
func (m *MCP) SetPath(path string) error {
	if path == "" {
		return fmt.Errorf("MCP path cannot be empty")
	}
	m.path = path
	m.touch()
	return nil
}

// AddFeature adds a feature to the MCP
func (m *MCP) AddFeature(feature *value_objects.Feature) error {
	if feature == nil {
		return fmt.Errorf("feature cannot be nil")
	}

	// Check for duplicates
	for _, f := range m.features {
		if f.Equals(feature) {
			return fmt.Errorf("feature %s already exists", feature.Name())
		}
	}

	m.features = append(m.features, feature)
	m.touch()
	return nil
}

// RemoveFeature removes a feature from the MCP
func (m *MCP) RemoveFeature(featureName string) error {
	for i, f := range m.features {
		if f.Name() == featureName {
			m.features = append(m.features[:i], m.features[i+1:]...)
			m.touch()
			return nil
		}
	}
	return fmt.Errorf("feature %s not found", featureName)
}

// EnableFeature enables a feature by name
func (m *MCP) EnableFeature(featureName string) error {
	for _, f := range m.features {
		if f.Name() == featureName {
			f.Enable()
			m.touch()
			return nil
		}
	}
	return fmt.Errorf("feature %s not found", featureName)
}

// DisableFeature disables a feature by name
func (m *MCP) DisableFeature(featureName string) error {
	for _, f := range m.features {
		if f.Name() == featureName {
			f.Disable()
			m.touch()
			return nil
		}
	}
	return fmt.Errorf("feature %s not found", featureName)
}

// AddContext adds knowledge context to the MCP
func (m *MCP) AddContext(knowledgeID string, documents []string, embeddings map[string][]float64, metadata map[string]interface{}) error {
	if knowledgeID == "" {
		return fmt.Errorf("knowledge ID cannot be empty")
	}
	if len(documents) == 0 {
		return fmt.Errorf("at least one document is required")
	}

	m.context = &KnowledgeContext{
		knowledgeID: knowledgeID,
		documents:   append([]string{}, documents...),
		embeddings:  copyEmbeddings(embeddings),
		metadata:    copyMetadataMCP(metadata),
	}
	m.touch()
	return nil
}

// RemoveContext removes the knowledge context
func (m *MCP) RemoveContext() {
	m.context = nil
	m.touch()
}

// HasContext checks if the MCP has knowledge context
func (m *MCP) HasContext() bool {
	return m.context != nil
}

// touch updates the updatedAt timestamp
func (m *MCP) touch() {
	m.updatedAt = time.Now()
}

// copyEmbeddings creates a deep copy of embeddings map
func copyEmbeddings(src map[string][]float64) map[string][]float64 {
	if src == nil {
		return nil
	}
	dst := make(map[string][]float64)
	for k, v := range src {
		dst[k] = append([]float64{}, v...)
	}
	return dst
}

// copyMetadata creates a deep copy of metadata map
func copyMetadataMCP(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
