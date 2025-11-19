package data

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// DataLineage represents data lineage information
type DataLineage struct {
	ID            string                 `json:"id"`
	DatasetID     string                 `json:"dataset_id"`
	VersionID     string                 `json:"version_id"`
	Source        LineageNode            `json:"source"`
	Transformations []Transformation     `json:"transformations"`
	Output        LineageNode            `json:"output"`
	CreatedAt     time.Time              `json:"created_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// LineageNode represents a node in the lineage graph
type LineageNode struct {
	ID          string                 `json:"id"`
	Type        NodeType               `json:"type"`
	Location    string                 `json:"location"`
	Version     string                 `json:"version,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NodeType represents node type
type NodeType string

const (
	NodeTypeDataset   NodeType = "dataset"
	NodeTypeTable     NodeType = "table"
	NodeTypeFile      NodeType = "file"
	NodeTypeStream    NodeType = "stream"
	NodeTypeModel     NodeType = "model"
)

// Transformation represents a transformation step
type Transformation struct {
	ID          string                 `json:"id"`
	Type        TransformationType     `json:"type"`
	Description string                 `json:"description"`
	Inputs      []string               `json:"inputs"` // node IDs
	Outputs     []string               `json:"outputs"` // node IDs
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
}

// TransformationType represents transformation type
type TransformationType string

const (
	TransformationTypeFilter    TransformationType = "filter"
	TransformationTypeJoin      TransformationType = "join"
	TransformationTypeAggregate TransformationType = "aggregate"
	TransformationTypeTransform TransformationType = "transform"
	TransformationTypeModel     TransformationType = "model"
)

// DataLineageTracker interface for tracking data lineage
type DataLineageTracker interface {
	// RecordLineage records lineage information
	RecordLineage(ctx context.Context, lineage *DataLineage) error
	
	// GetLineage retrieves lineage for a dataset version
	GetLineage(ctx context.Context, versionID string) (*DataLineage, error)
	
	// TraceUpstream traces upstream dependencies
	TraceUpstream(ctx context.Context, versionID string) ([]*DataLineage, error)
	
	// TraceDownstream traces downstream dependencies
	TraceDownstream(ctx context.Context, versionID string) ([]*DataLineage, error)
	
	// AddTransformation adds a transformation step
	AddTransformation(ctx context.Context, lineageID string, transformation *Transformation) error
}

// InMemoryDataLineageTracker implements DataLineageTracker
type InMemoryDataLineageTracker struct {
	lineages map[string]*DataLineage
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewInMemoryDataLineageTracker creates a new data lineage tracker
func NewInMemoryDataLineageTracker() *InMemoryDataLineageTracker {
	return &InMemoryDataLineageTracker{
		lineages: make(map[string]*DataLineage),
		logger:   logger.WithContext(context.Background()),
	}
}

// RecordLineage records lineage information
func (dlt *InMemoryDataLineageTracker) RecordLineage(ctx context.Context, lineage *DataLineage) error {
	dlt.mu.Lock()
	defer dlt.mu.Unlock()

	logger := logger.WithContext(ctx)

	if lineage.ID == "" {
		lineage.ID = uuid.New().String()
	}

	if lineage.CreatedAt.IsZero() {
		lineage.CreatedAt = time.Now()
	}

	if lineage.Metadata == nil {
		lineage.Metadata = make(map[string]interface{})
	}

	dlt.lineages[lineage.ID] = lineage

	logger.Info("Lineage recorded",
		zap.String("lineage_id", lineage.ID),
		zap.String("dataset_id", lineage.DatasetID),
		zap.String("version_id", lineage.VersionID))

	return nil
}

// GetLineage retrieves lineage for a dataset version
func (dlt *InMemoryDataLineageTracker) GetLineage(ctx context.Context, versionID string) (*DataLineage, error) {
	dlt.mu.RLock()
	defer dlt.mu.RUnlock()

	for _, lineage := range dlt.lineages {
		if lineage.VersionID == versionID {
			return lineage, nil
		}
	}

	return nil, fmt.Errorf("lineage not found for version %s", versionID)
}

// TraceUpstream traces upstream dependencies
func (dlt *InMemoryDataLineageTracker) TraceUpstream(ctx context.Context, versionID string) ([]*DataLineage, error) {
	dlt.mu.RLock()
	defer dlt.mu.RUnlock()

	var upstream []*DataLineage
	visited := make(map[string]bool)

	var trace func(versionID string)
	trace = func(vID string) {
		if visited[vID] {
			return
		}
		visited[vID] = true

		for _, lineage := range dlt.lineages {
			if lineage.VersionID == vID {
				// Add source lineage if exists
				if lineage.Source.ID != "" {
					if sourceLineage, err := dlt.findLineageByNodeID(lineage.Source.ID); err == nil {
						upstream = append(upstream, sourceLineage)
						trace(sourceLineage.VersionID)
					}
				}
			}
		}
	}

	trace(versionID)
	return upstream, nil
}

// TraceDownstream traces downstream dependencies
func (dlt *InMemoryDataLineageTracker) TraceDownstream(ctx context.Context, versionID string) ([]*DataLineage, error) {
	dlt.mu.RLock()
	defer dlt.mu.RUnlock()

	var downstream []*DataLineage
	visited := make(map[string]bool)

	var trace func(versionID string)
	trace = func(vID string) {
		if visited[vID] {
			return
		}
		visited[vID] = true

		for _, lineage := range dlt.lineages {
			if lineage.Source.ID == vID || containsNodeID(lineage.Source.ID, vID) {
				downstream = append(downstream, lineage)
				trace(lineage.VersionID)
			}
		}
	}

	trace(versionID)
	return downstream, nil
}

// AddTransformation adds a transformation step
func (dlt *InMemoryDataLineageTracker) AddTransformation(ctx context.Context, lineageID string, transformation *Transformation) error {
	dlt.mu.Lock()
	defer dlt.mu.Unlock()

	logger := logger.WithContext(ctx)

	lineage, exists := dlt.lineages[lineageID]
	if !exists {
		return fmt.Errorf("lineage %s not found", lineageID)
	}

	if transformation.ID == "" {
		transformation.ID = uuid.New().String()
	}

	if transformation.Timestamp.IsZero() {
		transformation.Timestamp = time.Now()
	}

	if transformation.Metadata == nil {
		transformation.Metadata = make(map[string]interface{})
	}

	lineage.Transformations = append(lineage.Transformations, *transformation)

	logger.Info("Transformation added",
		zap.String("lineage_id", lineageID),
		zap.String("transformation_id", transformation.ID))

	return nil
}

// findLineageByNodeID finds lineage by node ID
func (dlt *InMemoryDataLineageTracker) findLineageByNodeID(nodeID string) (*DataLineage, error) {
	for _, lineage := range dlt.lineages {
		if lineage.Source.ID == nodeID || lineage.Output.ID == nodeID {
			return lineage, nil
		}
	}
	return nil, fmt.Errorf("lineage not found for node %s", nodeID)
}

// containsNodeID checks if transformations contain node ID
func containsNodeID(nodeID string, targetID string) bool {
	// Simplified check
	return nodeID == targetID
}
