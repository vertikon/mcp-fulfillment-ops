package finetuning

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// Versioning manages model versioning
type Versioning struct {
	store *FinetuningStore
}

// NewVersioning creates a new versioning manager
func NewVersioning(store *FinetuningStore) *Versioning {
	return &Versioning{
		store: store,
	}
}

// CreateVersion creates a new model version from a completed training job
func (v *Versioning) CreateVersion(ctx context.Context, job *entities.TrainingJob) (*entities.ModelVersion, error) {
	if job.Status() != entities.StatusCompleted {
		return nil, fmt.Errorf("job must be completed to create version")
	}

	// Get current versions to determine next version number
	existingVersions, err := v.store.GetModelVersions(ctx, job.TargetModel())
	if err != nil {
		return nil, fmt.Errorf("failed to get existing versions: %w", err)
	}

	nextVersion := 1
	if len(existingVersions) > 0 {
		// Find highest version
		for _, version := range existingVersions {
			if version.Version() >= nextVersion {
				nextVersion = version.Version() + 1
			}
		}
	}

	// Create new version
	version, err := entities.NewModelVersion(
		job.TargetModel(),
		nextVersion,
		job.ID(),
		job.CheckpointPath(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create model version: %w", err)
	}

	// Copy metrics from job
	for k, v := range job.Metrics() {
		version.SetMetric(k, v)
	}

	// Save version
	if err := v.store.SaveModelVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to save version: %w", err)
	}

	return version, nil
}

// GetVersion retrieves a model version
func (v *Versioning) GetVersion(ctx context.Context, versionID string) (*entities.ModelVersion, error) {
	return v.store.GetModelVersion(ctx, versionID)
}

// ListVersions lists all versions of a model
func (v *Versioning) ListVersions(ctx context.Context, modelName string) ([]*entities.ModelVersion, error) {
	return v.store.GetModelVersions(ctx, modelName)
}

// ActivateVersion activates a model version
func (v *Versioning) ActivateVersion(ctx context.Context, modelName string, version int) error {
	versions, err := v.store.GetModelVersions(ctx, modelName)
	if err != nil {
		return fmt.Errorf("failed to get versions: %w", err)
	}

	// Deactivate all versions
	for _, ver := range versions {
		if ver.IsActive() {
			ver.SetActive(false)
			if err := v.store.SaveModelVersion(ctx, ver); err != nil {
				return fmt.Errorf("failed to deactivate version: %w", err)
			}
		}
	}

	// Activate target version
	for _, ver := range versions {
		if ver.Version() == version {
			ver.SetActive(true)
			return v.store.SaveModelVersion(ctx, ver)
		}
	}

	return fmt.Errorf("version %d not found for model %s", version, modelName)
}

// Rollback rolls back to a previous version
func (v *Versioning) Rollback(ctx context.Context, modelName string, version int) error {
	return v.ActivateVersion(ctx, modelName, version)
}

// GetActiveVersion retrieves the active version of a model
func (v *Versioning) GetActiveVersion(ctx context.Context, modelName string) (*entities.ModelVersion, error) {
	return v.store.GetActiveVersion(ctx, modelName)
}

// CompareVersions compares two model versions
func (v *Versioning) CompareVersions(ctx context.Context, versionID1 string, versionID2 string) (*VersionComparison, error) {
	version1, err := v.store.GetModelVersion(ctx, versionID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get version 1: %w", err)
	}

	version2, err := v.store.GetModelVersion(ctx, versionID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get version 2: %w", err)
	}

	return &VersionComparison{
		Version1: version1,
		Version2: version2,
	}, nil
}

// VersionComparison represents a comparison between two versions
type VersionComparison struct {
	Version1 *entities.ModelVersion
	Version2 *entities.ModelVersion
}
