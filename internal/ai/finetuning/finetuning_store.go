package finetuning

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// FinetuningRepository defines interface for fine-tuning persistence
type FinetuningRepository interface {
	SaveJob(ctx context.Context, job *entities.TrainingJob) error
	GetJob(ctx context.Context, id string) (*entities.TrainingJob, error)
	ListJobs(ctx context.Context, filters *JobFilters) ([]*entities.TrainingJob, error)
	DeleteJob(ctx context.Context, id string) error

	SaveDataset(ctx context.Context, dataset *entities.Dataset) error
	GetDataset(ctx context.Context, id string) (*entities.Dataset, error)
	ListDatasets(ctx context.Context) ([]*entities.Dataset, error)
	DeleteDataset(ctx context.Context, id string) error

	SaveModelVersion(ctx context.Context, version *entities.ModelVersion) error
	GetModelVersion(ctx context.Context, id string) (*entities.ModelVersion, error)
	GetModelVersions(ctx context.Context, modelName string) ([]*entities.ModelVersion, error)
	GetActiveVersion(ctx context.Context, modelName string) (*entities.ModelVersion, error)
}

// JobFilters represents filters for listing jobs
type JobFilters struct {
	Status    entities.TrainingStatus
	DatasetID string
	Limit     int
	Offset    int
}

// FinetuningStore manages fine-tuning data storage
type FinetuningStore struct {
	repository FinetuningRepository
}

// NewFinetuningStore creates a new fine-tuning store
func NewFinetuningStore(repository FinetuningRepository) *FinetuningStore {
	return &FinetuningStore{
		repository: repository,
	}
}

// SaveJob saves a training job
func (fs *FinetuningStore) SaveJob(ctx context.Context, job *entities.TrainingJob) error {
	return fs.repository.SaveJob(ctx, job)
}

// GetJob retrieves a training job
func (fs *FinetuningStore) GetJob(ctx context.Context, id string) (*entities.TrainingJob, error) {
	return fs.repository.GetJob(ctx, id)
}

// ListJobs lists training jobs with filters
func (fs *FinetuningStore) ListJobs(ctx context.Context, filters *JobFilters) ([]*entities.TrainingJob, error) {
	return fs.repository.ListJobs(ctx, filters)
}

// GetActiveJobs retrieves active (running/pending) jobs
func (fs *FinetuningStore) GetActiveJobs(ctx context.Context) ([]*entities.TrainingJob, error) {
	filters := &JobFilters{
		Status: entities.StatusRunning, // Will need to handle multiple statuses
		Limit:  100,
	}
	return fs.repository.ListJobs(ctx, filters)
}

// DeleteJob deletes a training job
func (fs *FinetuningStore) DeleteJob(ctx context.Context, id string) error {
	return fs.repository.DeleteJob(ctx, id)
}

// SaveDataset saves a dataset
func (fs *FinetuningStore) SaveDataset(ctx context.Context, dataset *entities.Dataset) error {
	return fs.repository.SaveDataset(ctx, dataset)
}

// GetDataset retrieves a dataset
func (fs *FinetuningStore) GetDataset(ctx context.Context, id string) (*entities.Dataset, error) {
	return fs.repository.GetDataset(ctx, id)
}

// ListDatasets lists all datasets
func (fs *FinetuningStore) ListDatasets(ctx context.Context) ([]*entities.Dataset, error) {
	return fs.repository.ListDatasets(ctx)
}

// DeleteDataset deletes a dataset
func (fs *FinetuningStore) DeleteDataset(ctx context.Context, id string) error {
	return fs.repository.DeleteDataset(ctx, id)
}

// SaveModelVersion saves a model version
func (fs *FinetuningStore) SaveModelVersion(ctx context.Context, version *entities.ModelVersion) error {
	return fs.repository.SaveModelVersion(ctx, version)
}

// GetModelVersion retrieves a model version
func (fs *FinetuningStore) GetModelVersion(ctx context.Context, id string) (*entities.ModelVersion, error) {
	return fs.repository.GetModelVersion(ctx, id)
}

// GetModelVersions retrieves all versions of a model
func (fs *FinetuningStore) GetModelVersions(ctx context.Context, modelName string) ([]*entities.ModelVersion, error) {
	return fs.repository.GetModelVersions(ctx, modelName)
}

// GetActiveVersion retrieves the active version of a model
func (fs *FinetuningStore) GetActiveVersion(ctx context.Context, modelName string) (*entities.ModelVersion, error) {
	return fs.repository.GetActiveVersion(ctx, modelName)
}
