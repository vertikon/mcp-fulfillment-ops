package data

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

// DataVersion represents a versioned dataset
type DataVersion struct {
	ID            string                 `json:"id"`
	DatasetID     string                 `json:"dataset_id"`
	Version       string                 `json:"version"`
	SchemaVersion string                 `json:"schema_version"`
	RowCount      int64                  `json:"row_count"`
	Checksum      string                 `json:"checksum"`
	Path          string                 `json:"path"`
	Format        string                 `json:"format"` // "parquet", "csv", "json", etc.
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Tags          []string               `json:"tags"`
}

// DataSnapshot represents a snapshot of data at a point in time
type DataSnapshot struct {
	ID           string                 `json:"id"`
	DatasetID    string                 `json:"dataset_id"`
	VersionID    string                 `json:"version_id"`
	SnapshotType SnapshotType           `json:"snapshot_type"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Checksum     string                 `json:"checksum"`
	CreatedAt    time.Time              `json:"created_at"`
	CreatedBy    string                 `json:"created_by"`
}

// SnapshotType represents snapshot type
type SnapshotType string

const (
	SnapshotTypeFull         SnapshotType = "full"
	SnapshotTypeIncremental  SnapshotType = "incremental"
	SnapshotTypeDifferential SnapshotType = "differential"
)

// DataVersioning interface for data versioning operations
type DataVersioning interface {
	// CreateVersion creates a new version of a dataset
	CreateVersion(ctx context.Context, datasetID string, metadata map[string]interface{}) (*DataVersion, error)

	// GetVersion retrieves a specific version
	GetVersion(ctx context.Context, versionID string) (*DataVersion, error)

	// ListVersions lists all versions for a dataset
	ListVersions(ctx context.Context, datasetID string) ([]*DataVersion, error)

	// GetLatestVersion gets the latest version
	GetLatestVersion(ctx context.Context, datasetID string) (*DataVersion, error)

	// CreateSnapshot creates a snapshot of data
	CreateSnapshot(ctx context.Context, versionID string, snapshotType SnapshotType, data map[string]interface{}) (*DataSnapshot, error)

	// GetSnapshot retrieves a snapshot
	GetSnapshot(ctx context.Context, snapshotID string) (*DataSnapshot, error)

	// ListSnapshots lists snapshots for a version
	ListSnapshots(ctx context.Context, versionID string) ([]*DataSnapshot, error)

	// TagVersion tags a version
	TagVersion(ctx context.Context, versionID string, tags []string) error

	// DeleteVersion deletes a version (soft delete)
	DeleteVersion(ctx context.Context, versionID string) error
}

// InMemoryDataVersioning implements DataVersioning in memory
type InMemoryDataVersioning struct {
	versions         map[string]*DataVersion
	snapshots        map[string]*DataSnapshot
	versionSnapshots map[string][]string // versionID -> []snapshotID
	mu               sync.RWMutex
	logger           *zap.Logger
}

// NewInMemoryDataVersioning creates a new in-memory data versioning instance
func NewInMemoryDataVersioning() *InMemoryDataVersioning {
	return &InMemoryDataVersioning{
		versions:         make(map[string]*DataVersion),
		snapshots:        make(map[string]*DataSnapshot),
		versionSnapshots: make(map[string][]string),
		logger:           logger.WithContext(context.Background()),
	}
}

// CreateVersion creates a new version of a dataset
func (dv *InMemoryDataVersioning) CreateVersion(ctx context.Context, datasetID string, metadata map[string]interface{}) (*DataVersion, error) {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	logger := logger.WithContext(ctx)
	logger.Info("Creating new data version", zap.String("dataset_id", datasetID))

	// Get latest version to increment
	latestVersion := "v1"
	var maxVersion int
	for _, v := range dv.versions {
		if v.DatasetID == datasetID {
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

	version := &DataVersion{
		ID:            versionID,
		DatasetID:     datasetID,
		Version:       latestVersion,
		SchemaVersion: "1.0",
		RowCount:      0,
		Checksum:      "",
		Path:          "",
		Format:        "parquet",
		Metadata:      metadata,
		CreatedAt:     now,
		CreatedBy:     getCurrentUser(ctx),
		Tags:          []string{},
	}

	dv.versions[versionID] = version
	dv.versionSnapshots[versionID] = []string{}

	logger.Info("Data version created",
		zap.String("version_id", versionID),
		zap.String("version", latestVersion))

	return version, nil
}

// GetVersion retrieves a specific version
func (dv *InMemoryDataVersioning) GetVersion(ctx context.Context, versionID string) (*DataVersion, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	version, exists := dv.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	return version, nil
}

// ListVersions lists all versions for a dataset
func (dv *InMemoryDataVersioning) ListVersions(ctx context.Context, datasetID string) ([]*DataVersion, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var versions []*DataVersion
	for _, v := range dv.versions {
		if v.DatasetID == datasetID {
			versions = append(versions, v)
		}
	}

	return versions, nil
}

// GetLatestVersion gets the latest version
func (dv *InMemoryDataVersioning) GetLatestVersion(ctx context.Context, datasetID string) (*DataVersion, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var latest *DataVersion
	var maxVersion int

	for _, v := range dv.versions {
		if v.DatasetID == datasetID {
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
		return nil, fmt.Errorf("no versions found for dataset %s", datasetID)
	}

	return latest, nil
}

// CreateSnapshot creates a snapshot of data
func (dv *InMemoryDataVersioning) CreateSnapshot(ctx context.Context, versionID string, snapshotType SnapshotType, data map[string]interface{}) (*DataSnapshot, error) {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	logger := logger.WithContext(ctx)

	// Verify version exists
	_, exists := dv.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	snapshotID := uuid.New().String()
	now := time.Now()

	// Calculate checksum
	checksum := dv.calculateChecksum(data)

	snapshot := &DataSnapshot{
		ID:           snapshotID,
		VersionID:    versionID,
		SnapshotType: snapshotType,
		Data:         data,
		Checksum:     checksum,
		CreatedAt:    now,
		CreatedBy:    getCurrentUser(ctx),
	}

	dv.snapshots[snapshotID] = snapshot
	dv.versionSnapshots[versionID] = append(dv.versionSnapshots[versionID], snapshotID)

	logger.Info("Snapshot created",
		zap.String("snapshot_id", snapshotID),
		zap.String("version_id", versionID),
		zap.String("type", string(snapshotType)))

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot
func (dv *InMemoryDataVersioning) GetSnapshot(ctx context.Context, snapshotID string) (*DataSnapshot, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	snapshot, exists := dv.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot %s not found", snapshotID)
	}

	return snapshot, nil
}

// ListSnapshots lists snapshots for a version
func (dv *InMemoryDataVersioning) ListSnapshots(ctx context.Context, versionID string) ([]*DataSnapshot, error) {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	snapshotIDs, exists := dv.versionSnapshots[versionID]
	if !exists {
		return []*DataSnapshot{}, nil
	}

	snapshots := make([]*DataSnapshot, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		if snapshot, exists := dv.snapshots[snapshotID]; exists {
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots, nil
}

// TagVersion tags a version
func (dv *InMemoryDataVersioning) TagVersion(ctx context.Context, versionID string, tags []string) error {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	logger := logger.WithContext(ctx)

	version, exists := dv.versions[versionID]
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

// DeleteVersion deletes a version (soft delete)
func (dv *InMemoryDataVersioning) DeleteVersion(ctx context.Context, versionID string) error {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	logger := logger.WithContext(ctx)

	version, exists := dv.versions[versionID]
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

// calculateChecksum calculates checksum for data
func (dv *InMemoryDataVersioning) calculateChecksum(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}

	hasher := sha256.New()
	for k, v := range data {
		hasher.Write([]byte(k))
		hasher.Write([]byte(fmt.Sprintf("%v", v)))
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
