package models

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

// Model represents a registered model
type Model struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Fingerprint string                 `json:"fingerprint"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Tags        []string               `json:"tags"`
}

// ModelVersion represents a version of a model
type ModelVersion struct {
	ID          string                 `json:"id"`
	ModelID     string                 `json:"model_id"`
	Version     string                 `json:"version"`
	Path        string                 `json:"path"`
	Fingerprint string                 `json:"fingerprint"`
	Size        int64                  `json:"size"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Status      ModelVersionStatus     `json:"status"`
}

// ModelVersionStatus represents the status of a model version
type ModelVersionStatus string

const (
	ModelVersionStatusDraft     ModelVersionStatus = "draft"
	ModelVersionStatusStaging   ModelVersionStatus = "staging"
	ModelVersionStatusProduction ModelVersionStatus = "production"
	ModelVersionStatusDeprecated ModelVersionStatus = "deprecated"
)

// ModelRegistry interface for model registration
type ModelRegistry interface {
	// RegisterModel registers a new model
	RegisterModel(ctx context.Context, model *Model) (*Model, error)
	
	// GetModel retrieves a model by ID
	GetModel(ctx context.Context, modelID string) (*Model, error)
	
	// ListModels lists all registered models
	ListModels(ctx context.Context) ([]*Model, error)
	
	// UpdateModel updates model metadata
	UpdateModel(ctx context.Context, modelID string, metadata map[string]interface{}) error
	
	// DeleteModel deletes a model (soft delete)
	DeleteModel(ctx context.Context, modelID string) error
	
	// RegisterVersion registers a new version of a model
	RegisterVersion(ctx context.Context, modelID string, version *ModelVersion) (*ModelVersion, error)
	
	// GetVersion retrieves a model version
	GetVersion(ctx context.Context, versionID string) (*ModelVersion, error)
	
	// ListVersions lists all versions of a model
	ListVersions(ctx context.Context, modelID string) ([]*ModelVersion, error)
	
	// GetLatestVersion gets the latest version of a model
	GetLatestVersion(ctx context.Context, modelID string) (*ModelVersion, error)
	
	// CalculateFingerprint calculates fingerprint for model data
	CalculateFingerprint(data []byte) string
}

// InMemoryModelRegistry implements ModelRegistry in memory
type InMemoryModelRegistry struct {
	models   map[string]*Model
	versions map[string]*ModelVersion // versionID -> version
	modelVersions map[string][]string  // modelID -> []versionID
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewInMemoryModelRegistry creates a new in-memory model registry
func NewInMemoryModelRegistry() *InMemoryModelRegistry {
	return &InMemoryModelRegistry{
		models:        make(map[string]*Model),
		versions:      make(map[string]*ModelVersion),
		modelVersions: make(map[string][]string),
		logger:        logger.WithContext(context.Background()),
	}
}

// RegisterModel registers a new model
func (mr *InMemoryModelRegistry) RegisterModel(ctx context.Context, model *Model) (*Model, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	logger := logger.WithContext(ctx)
	logger.Info("Registering model", zap.String("name", model.Name))

	if model.ID == "" {
		model.ID = uuid.New().String()
	}

	if model.CreatedAt.IsZero() {
		model.CreatedAt = time.Now()
	}

	if model.CreatedBy == "" {
		model.CreatedBy = getCurrentUser(ctx)
	}

	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}

	mr.models[model.ID] = model
	mr.modelVersions[model.ID] = []string{}

	logger.Info("Model registered", zap.String("model_id", model.ID))
	return model, nil
}

// GetModel retrieves a model by ID
func (mr *InMemoryModelRegistry) GetModel(ctx context.Context, modelID string) (*Model, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	model, exists := mr.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	return model, nil
}

// ListModels lists all registered models
func (mr *InMemoryModelRegistry) ListModels(ctx context.Context) ([]*Model, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	models := make([]*Model, 0, len(mr.models))
	for _, model := range mr.models {
		models = append(models, model)
	}

	return models, nil
}

// UpdateModel updates model metadata
func (mr *InMemoryModelRegistry) UpdateModel(ctx context.Context, modelID string, metadata map[string]interface{}) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	logger := logger.WithContext(ctx)

	model, exists := mr.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}

	for k, v := range metadata {
		model.Metadata[k] = v
	}

	logger.Info("Model updated", zap.String("model_id", modelID))
	return nil
}

// DeleteModel deletes a model (soft delete)
func (mr *InMemoryModelRegistry) DeleteModel(ctx context.Context, modelID string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	logger := logger.WithContext(ctx)

	model, exists := mr.models[modelID]
	if !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	// Soft delete: mark as deleted in metadata
	if model.Metadata == nil {
		model.Metadata = make(map[string]interface{})
	}
	model.Metadata["deleted"] = true
	model.Metadata["deleted_at"] = time.Now()

	logger.Info("Model soft deleted", zap.String("model_id", modelID))
	return nil
}

// RegisterVersion registers a new version of a model
func (mr *InMemoryModelRegistry) RegisterVersion(ctx context.Context, modelID string, version *ModelVersion) (*ModelVersion, error) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	logger := logger.WithContext(ctx)

	// Verify model exists
	_, exists := mr.models[modelID]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	if version.ID == "" {
		version.ID = uuid.New().String()
	}

	version.ModelID = modelID

	// Auto-increment version if not provided
	if version.Version == "" {
		versions := mr.modelVersions[modelID]
		version.Version = fmt.Sprintf("v%d", len(versions)+1)
	}

	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}

	if version.CreatedBy == "" {
		version.CreatedBy = getCurrentUser(ctx)
	}

	if version.Status == "" {
		version.Status = ModelVersionStatusDraft
	}

	if version.Metadata == nil {
		version.Metadata = make(map[string]interface{})
	}

	mr.versions[version.ID] = version
	mr.modelVersions[modelID] = append(mr.modelVersions[modelID], version.ID)

	logger.Info("Model version registered",
		zap.String("model_id", modelID),
		zap.String("version_id", version.ID),
		zap.String("version", version.Version))

	return version, nil
}

// GetVersion retrieves a model version
func (mr *InMemoryModelRegistry) GetVersion(ctx context.Context, versionID string) (*ModelVersion, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	version, exists := mr.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version %s not found", versionID)
	}

	return version, nil
}

// ListVersions lists all versions of a model
func (mr *InMemoryModelRegistry) ListVersions(ctx context.Context, modelID string) ([]*ModelVersion, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	versionIDs, exists := mr.modelVersions[modelID]
	if !exists {
		return []*ModelVersion{}, nil
	}

	versions := make([]*ModelVersion, 0, len(versionIDs))
	for _, versionID := range versionIDs {
		if version, exists := mr.versions[versionID]; exists {
			versions = append(versions, version)
		}
	}

	return versions, nil
}

// GetLatestVersion gets the latest version of a model
func (mr *InMemoryModelRegistry) GetLatestVersion(ctx context.Context, modelID string) (*ModelVersion, error) {
	versions, err := mr.ListVersions(ctx, modelID)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for model %s", modelID)
	}

	// Find latest by version number
	var latest *ModelVersion
	var maxVersion int

	for _, v := range versions {
		var vNum int
		fmt.Sscanf(v.Version, "v%d", &vNum)
		if vNum > maxVersion {
			maxVersion = vNum
			latest = v
		}
	}

	return latest, nil
}

// CalculateFingerprint calculates fingerprint for model data
func (mr *InMemoryModelRegistry) CalculateFingerprint(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

