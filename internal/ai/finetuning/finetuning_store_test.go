package finetuning

import (
	"context"
	"fmt"
	"testing"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// mockFinetuningRepository is a mock implementation of FinetuningRepository
type mockFinetuningRepository struct {
	jobs          map[string]*entities.TrainingJob
	datasets      map[string]*entities.Dataset
	modelVersions map[string]*entities.ModelVersion
}

func newMockFinetuningRepository() *mockFinetuningRepository {
	return &mockFinetuningRepository{
		jobs:          make(map[string]*entities.TrainingJob),
		datasets:      make(map[string]*entities.Dataset),
		modelVersions: make(map[string]*entities.ModelVersion),
	}
}

func (m *mockFinetuningRepository) SaveJob(ctx context.Context, job *entities.TrainingJob) error {
	m.jobs[job.ID()] = job
	return nil
}

func (m *mockFinetuningRepository) GetJob(ctx context.Context, id string) (*entities.TrainingJob, error) {
	if job, ok := m.jobs[id]; ok {
		return job, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockFinetuningRepository) ListJobs(ctx context.Context, filters *JobFilters) ([]*entities.TrainingJob, error) {
	result := make([]*entities.TrainingJob, 0)
	for _, job := range m.jobs {
		if filters == nil || filters.Status == "" || job.Status() == filters.Status {
			result = append(result, job)
		}
	}
	return result, nil
}

func (m *mockFinetuningRepository) DeleteJob(ctx context.Context, id string) error {
	delete(m.jobs, id)
	return nil
}

func (m *mockFinetuningRepository) SaveDataset(ctx context.Context, dataset *entities.Dataset) error {
	m.datasets[dataset.ID()] = dataset
	return nil
}

func (m *mockFinetuningRepository) GetDataset(ctx context.Context, id string) (*entities.Dataset, error) {
	if dataset, ok := m.datasets[id]; ok {
		return dataset, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockFinetuningRepository) ListDatasets(ctx context.Context) ([]*entities.Dataset, error) {
	result := make([]*entities.Dataset, 0, len(m.datasets))
	for _, dataset := range m.datasets {
		result = append(result, dataset)
	}
	return result, nil
}

func (m *mockFinetuningRepository) DeleteDataset(ctx context.Context, id string) error {
	delete(m.datasets, id)
	return nil
}

func (m *mockFinetuningRepository) SaveModelVersion(ctx context.Context, version *entities.ModelVersion) error {
	m.modelVersions[version.ID()] = version
	return nil
}

func (m *mockFinetuningRepository) GetModelVersion(ctx context.Context, id string) (*entities.ModelVersion, error) {
	if version, ok := m.modelVersions[id]; ok {
		return version, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockFinetuningRepository) GetModelVersions(ctx context.Context, modelName string) ([]*entities.ModelVersion, error) {
	result := make([]*entities.ModelVersion, 0)
	for _, version := range m.modelVersions {
		if version.ModelName() == modelName {
			result = append(result, version)
		}
	}
	return result, nil
}

func (m *mockFinetuningRepository) GetActiveVersion(ctx context.Context, modelName string) (*entities.ModelVersion, error) {
	for _, version := range m.modelVersions {
		if version.ModelName() == modelName && version.IsActive() {
			return version, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func TestNewFinetuningStore(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	if store == nil {
		t.Fatal("NewFinetuningStore returned nil")
	}
	if store.repository != repo {
		t.Error("repository not set correctly")
	}
}

func TestFinetuningStore_SaveJob(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	dataset, _ := entities.NewDataset("dataset1", "desc", "path")
	job, err := entities.NewTrainingJob(dataset.ID(), "base-model", "target-model")
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	err = store.SaveJob(ctx, job)
	if err != nil {
		t.Fatalf("SaveJob failed: %v", err)
	}

	retrieved, err := store.GetJob(ctx, job.ID())
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if retrieved.ID() != job.ID() {
		t.Errorf("Expected job ID %s, got %s", job.ID(), retrieved.ID())
	}
}

func TestFinetuningStore_GetActiveJobs(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	dataset1, _ := entities.NewDataset("dataset1", "desc1", "path1")
	dataset2, _ := entities.NewDataset("dataset2", "desc2", "path2")
	job1, _ := entities.NewTrainingJob(dataset1.ID(), "base1", "target1")
	job1.SetStatus(entities.StatusRunning)
	store.SaveJob(ctx, job1)

	job2, _ := entities.NewTrainingJob(dataset2.ID(), "base2", "target2")
	job2.SetStatus(entities.StatusCompleted)
	store.SaveJob(ctx, job2)

	active, err := store.GetActiveJobs(ctx)
	if err != nil {
		t.Fatalf("GetActiveJobs failed: %v", err)
	}

	if len(active) == 0 {
		t.Error("Expected active jobs, got empty")
	}
}

func TestFinetuningStore_SaveDataset(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	dataset, err := entities.NewDataset("test-dataset", "description", "path/to/file.jsonl")
	if err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}

	err = store.SaveDataset(ctx, dataset)
	if err != nil {
		t.Fatalf("SaveDataset failed: %v", err)
	}

	retrieved, err := store.GetDataset(ctx, dataset.ID())
	if err != nil {
		t.Fatalf("GetDataset failed: %v", err)
	}

	if retrieved.ID() != dataset.ID() {
		t.Errorf("Expected dataset ID %s, got %s", dataset.ID(), retrieved.ID())
	}
}

func TestFinetuningStore_ListDatasets(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	dataset1, _ := entities.NewDataset("dataset1", "desc1", "path1")
	dataset2, _ := entities.NewDataset("dataset2", "desc2", "path2")
	store.SaveDataset(ctx, dataset1)
	store.SaveDataset(ctx, dataset2)

	datasets, err := store.ListDatasets(ctx)
	if err != nil {
		t.Fatalf("ListDatasets failed: %v", err)
	}

	if len(datasets) != 2 {
		t.Errorf("Expected 2 datasets, got %d", len(datasets))
	}
}

func TestFinetuningStore_SaveModelVersion(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	version, err := entities.NewModelVersion("model1", 1, "job1", "checkpoint/path")
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	err = store.SaveModelVersion(ctx, version)
	if err != nil {
		t.Fatalf("SaveModelVersion failed: %v", err)
	}

	retrieved, err := store.GetModelVersion(ctx, version.ID())
	if err != nil {
		t.Fatalf("GetModelVersion failed: %v", err)
	}

	if retrieved.ID() != version.ID() {
		t.Errorf("Expected version ID %s, got %s", version.ID(), retrieved.ID())
	}
}

func TestFinetuningStore_GetModelVersions(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	version1, _ := entities.NewModelVersion("model1", 1, "job1", "path1")
	version2, _ := entities.NewModelVersion("model1", 2, "job2", "path2")
	store.SaveModelVersion(ctx, version1)
	store.SaveModelVersion(ctx, version2)

	versions, err := store.GetModelVersions(ctx, "model1")
	if err != nil {
		t.Fatalf("GetModelVersions failed: %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}
}

func TestFinetuningStore_GetActiveVersion(t *testing.T) {
	repo := newMockFinetuningRepository()
	store := NewFinetuningStore(repo)

	ctx := context.Background()
	version1, _ := entities.NewModelVersion("model1", 1, "job1", "path1")
	version1.SetActive(true)
	store.SaveModelVersion(ctx, version1)

	version2, _ := entities.NewModelVersion("model1", 2, "job2", "path2")
	version2.SetActive(false)
	store.SaveModelVersion(ctx, version2)

	active, err := store.GetActiveVersion(ctx, "model1")
	if err != nil {
		t.Fatalf("GetActiveVersion failed: %v", err)
	}

	if active.Version() != 1 {
		t.Errorf("Expected active version 1, got %d", active.Version())
	}
}
