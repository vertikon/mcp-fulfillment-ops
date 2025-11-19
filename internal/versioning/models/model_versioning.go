package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// VersioningStrategy represents versioning strategy
type VersioningStrategy string

const (
	VersioningStrategySemantic VersioningStrategy = "semantic" // v1.0.0, v1.1.0, etc.
	VersioningStrategyIncremental VersioningStrategy = "incremental" // v1, v2, v3, etc.
	VersioningStrategyTimestamp VersioningStrategy = "timestamp" // timestamp-based
)

// ModelVersioning interface for model versioning operations
type ModelVersioning interface {
	// CreateVersion creates a new version of a model
	CreateVersion(ctx context.Context, modelID string, strategy VersioningStrategy, metadata map[string]interface{}) (*ModelVersion, error)
	
	// PromoteVersion promotes a version to a new status
	PromoteVersion(ctx context.Context, versionID string, targetStatus ModelVersionStatus) error
	
	// DeprecateVersion deprecates a version
	DeprecateVersion(ctx context.Context, versionID string) error
	
	// GetVersionHistory gets version history for a model
	GetVersionHistory(ctx context.Context, modelID string) ([]*ModelVersion, error)
	
	// CompareVersions compares two versions
	CompareVersions(ctx context.Context, versionID1, versionID2 string) (*VersionComparison, error)
	
	// GetVersionLifecycle gets lifecycle information for a version
	GetVersionLifecycle(ctx context.Context, versionID string) (*VersionLifecycle, error)
}

// VersionComparison represents comparison between two versions
type VersionComparison struct {
	Version1      *ModelVersion `json:"version1"`
	Version2      *ModelVersion `json:"version2"`
	Differences   []string      `json:"differences"`
	Compatibility string       `json:"compatibility"` // "compatible", "breaking", "unknown"
}

// VersionLifecycle represents lifecycle information
type VersionLifecycle struct {
	VersionID     string                 `json:"version_id"`
	CurrentStatus ModelVersionStatus     `json:"current_status"`
	Transitions   []StatusTransition     `json:"transitions"`
	CreatedAt     time.Time              `json:"created_at"`
	LastModified  time.Time              `json:"last_modified"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// StatusTransition represents a status transition
type StatusTransition struct {
	FromStatus   ModelVersionStatus `json:"from_status"`
	ToStatus     ModelVersionStatus `json:"to_status"`
	Timestamp    time.Time          `json:"timestamp"`
	TriggeredBy  string             `json:"triggered_by"`
	Reason       string             `json:"reason"`
}

// InMemoryModelVersioning implements ModelVersioning
type InMemoryModelVersioning struct {
	registry   ModelRegistry
	lifecycles map[string]*VersionLifecycle
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewInMemoryModelVersioning creates a new model versioning instance
func NewInMemoryModelVersioning(registry ModelRegistry) *InMemoryModelVersioning {
	return &InMemoryModelVersioning{
		registry:   registry,
		lifecycles: make(map[string]*VersionLifecycle),
		logger:     logger.WithContext(context.Background()),
	}
}

// CreateVersion creates a new version of a model
func (mv *InMemoryModelVersioning) CreateVersion(ctx context.Context, modelID string, strategy VersioningStrategy, metadata map[string]interface{}) (*ModelVersion, error) {
	logger := logger.WithContext(ctx)
	logger.Info("Creating new model version",
		zap.String("model_id", modelID),
		zap.String("strategy", string(strategy)))

	// Get latest version to determine next version number
	latestVersion, err := mv.registry.GetLatestVersion(ctx, modelID)
	var nextVersion string

	if err != nil {
		// No existing versions, start with v1
		nextVersion = "v1"
	} else {
		switch strategy {
		case VersioningStrategySemantic:
			nextVersion = mv.incrementSemanticVersion(latestVersion.Version)
		case VersioningStrategyIncremental:
			nextVersion = mv.incrementVersion(latestVersion.Version)
		case VersioningStrategyTimestamp:
			nextVersion = fmt.Sprintf("v%d", time.Now().Unix())
		default:
			nextVersion = mv.incrementVersion(latestVersion.Version)
		}
	}

	version := &ModelVersion{
		Version:  nextVersion,
		Status:   ModelVersionStatusDraft,
		Metadata: metadata,
	}

	createdVersion, err := mv.registry.RegisterVersion(ctx, modelID, version)
	if err != nil {
		return nil, err
	}

	// Initialize lifecycle
	lifecycle := &VersionLifecycle{
		VersionID:     createdVersion.ID,
		CurrentStatus: ModelVersionStatusDraft,
		Transitions:   []StatusTransition{},
		CreatedAt:     time.Now(),
		LastModified:  time.Now(),
		Metadata:      metadata,
	}

	mv.mu.Lock()
	mv.lifecycles[createdVersion.ID] = lifecycle
	mv.mu.Unlock()

	logger.Info("Model version created",
		zap.String("version_id", createdVersion.ID),
		zap.String("version", nextVersion))

	return createdVersion, nil
}

// PromoteVersion promotes a version to a new status
func (mv *InMemoryModelVersioning) PromoteVersion(ctx context.Context, versionID string, targetStatus ModelVersionStatus) error {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	logger := logger.WithContext(ctx)

	version, err := mv.registry.GetVersion(ctx, versionID)
	if err != nil {
		return err
	}

	lifecycle, exists := mv.lifecycles[versionID]
	if !exists {
		lifecycle = &VersionLifecycle{
			VersionID:     versionID,
			CurrentStatus: version.Status,
			Transitions:   []StatusTransition{},
			CreatedAt:     version.CreatedAt,
			LastModified:  time.Now(),
			Metadata:      make(map[string]interface{}),
		}
	}

	// Record transition
	transition := StatusTransition{
		FromStatus:  version.Status,
		ToStatus:    targetStatus,
		Timestamp:   time.Now(),
		TriggeredBy: getCurrentUser(ctx),
		Reason:      "promotion",
	}

	lifecycle.Transitions = append(lifecycle.Transitions, transition)
	lifecycle.CurrentStatus = targetStatus
	lifecycle.LastModified = time.Now()

	version.Status = targetStatus
	mv.lifecycles[versionID] = lifecycle

	logger.Info("Version promoted",
		zap.String("version_id", versionID),
		zap.String("from_status", string(transition.FromStatus)),
		zap.String("to_status", string(targetStatus)))

	return nil
}

// DeprecateVersion deprecates a version
func (mv *InMemoryModelVersioning) DeprecateVersion(ctx context.Context, versionID string) error {
	return mv.PromoteVersion(ctx, versionID, ModelVersionStatusDeprecated)
}

// GetVersionHistory gets version history for a model
func (mv *InMemoryModelVersioning) GetVersionHistory(ctx context.Context, modelID string) ([]*ModelVersion, error) {
	return mv.registry.ListVersions(ctx, modelID)
}

// CompareVersions compares two versions
func (mv *InMemoryModelVersioning) CompareVersions(ctx context.Context, versionID1, versionID2 string) (*VersionComparison, error) {
	version1, err := mv.registry.GetVersion(ctx, versionID1)
	if err != nil {
		return nil, err
	}

	version2, err := mv.registry.GetVersion(ctx, versionID2)
	if err != nil {
		return nil, err
	}

	comparison := &VersionComparison{
		Version1:    version1,
		Version2:    version2,
		Differences: []string{},
		Compatibility: "unknown",
	}

	// Compare fingerprints
	if version1.Fingerprint != version2.Fingerprint {
		comparison.Differences = append(comparison.Differences, "fingerprint")
	}

	// Compare size
	if version1.Size != version2.Size {
		comparison.Differences = append(comparison.Differences, "size")
	}

	// Compare path
	if version1.Path != version2.Path {
		comparison.Differences = append(comparison.Differences, "path")
	}

	// Determine compatibility (simplified)
	if len(comparison.Differences) == 0 {
		comparison.Compatibility = "compatible"
	} else if version1.Fingerprint != version2.Fingerprint {
		comparison.Compatibility = "breaking"
	} else {
		comparison.Compatibility = "compatible"
	}

	return comparison, nil
}

// GetVersionLifecycle gets lifecycle information for a version
func (mv *InMemoryModelVersioning) GetVersionLifecycle(ctx context.Context, versionID string) (*VersionLifecycle, error) {
	mv.mu.RLock()
	defer mv.mu.RUnlock()

	lifecycle, exists := mv.lifecycles[versionID]
	if !exists {
		version, err := mv.registry.GetVersion(ctx, versionID)
		if err != nil {
			return nil, err
		}

		lifecycle = &VersionLifecycle{
			VersionID:     versionID,
			CurrentStatus: version.Status,
			Transitions:   []StatusTransition{},
			CreatedAt:     version.CreatedAt,
			LastModified:  version.CreatedAt,
			Metadata:      make(map[string]interface{}),
		}
	}

	return lifecycle, nil
}

// incrementVersion increments a version number (v1 -> v2)
func (mv *InMemoryModelVersioning) incrementVersion(version string) string {
	var vNum int
	fmt.Sscanf(version, "v%d", &vNum)
	return fmt.Sprintf("v%d", vNum+1)
}

// incrementSemanticVersion increments semantic version (v1.0.0 -> v1.0.1)
func (mv *InMemoryModelVersioning) incrementSemanticVersion(version string) string {
	var major, minor, patch int
	fmt.Sscanf(version, "v%d.%d.%d", &major, &minor, &patch)
	patch++
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

