package finetuning

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/entities"
)

// RunPodClient defines interface for RunPod API operations
type RunPodClient interface {
	CreateJob(ctx context.Context, config *RunPodJobConfig) (string, error)
	GetJobStatus(ctx context.Context, jobID string) (*RunPodJobStatus, error)
	CancelJob(ctx context.Context, jobID string) error
	GetJobLogs(ctx context.Context, jobID string) ([]string, error)
}

// RunPodJobConfig represents RunPod job configuration
type RunPodJobConfig struct {
	Image         string
	GPUType       string
	ContainerDisk int
	VolumeMounts  map[string]string
	EnvVars       map[string]string
	Command       string
	Input         map[string]interface{}
}

// RunPodJobStatus represents RunPod job status
type RunPodJobStatus struct {
	ID       string
	Status   string
	Progress float64
	Output   map[string]interface{}
	Error    string
}

// StorageClient defines interface for storage operations (S3/MinIO)
type StorageClient interface {
	Upload(ctx context.Context, bucket string, key string, data []byte) error
	Download(ctx context.Context, bucket string, key string) ([]byte, error)
	Exists(ctx context.Context, bucket string, key string) (bool, error)
	Delete(ctx context.Context, bucket string, key string) error
}

// FinetuningEngine orchestrates the fine-tuning process
type FinetuningEngine struct {
	runpodClient  RunPodClient
	storageClient StorageClient
	store         *FinetuningStore
	memoryManager *MemoryManager
	versioning    *Versioning
}

// NewFinetuningEngine creates a new fine-tuning engine
func NewFinetuningEngine(
	runpodClient RunPodClient,
	storageClient StorageClient,
	store *FinetuningStore,
	memoryManager *MemoryManager,
	versioning *Versioning,
) *FinetuningEngine {
	return &FinetuningEngine{
		runpodClient:  runpodClient,
		storageClient: storageClient,
		store:         store,
		memoryManager: memoryManager,
		versioning:    versioning,
	}
}

// StartTraining starts a fine-tuning training job
func (fe *FinetuningEngine) StartTraining(ctx context.Context, job *entities.TrainingJob, dataset *entities.Dataset) error {
	// Update job status
	job.SetStatus(entities.StatusRunning)

	// Generate dataset from memory if needed
	datasetPath, err := fe.memoryManager.GenerateDataset(ctx, dataset)
	if err != nil {
		job.SetStatus(entities.StatusFailed)
		job.SetErrorMessage(fmt.Sprintf("failed to generate dataset: %v", err))
		return fmt.Errorf("failed to generate dataset: %w", err)
	}

	// Upload dataset to storage
	_ = "datasets"
	_ = fmt.Sprintf("%s/%s.jsonl", dataset.ID(), dataset.Name())
	// In production, read file and upload
	// For now, assume dataset is already uploaded

	// Create RunPod job config
	config := &RunPodJobConfig{
		Image:         "runpod/pytorch:2.0.1-py3.10-cuda11.8.0-devel",
		GPUType:       "NVIDIA RTX 4090",
		ContainerDisk: 20,
		VolumeMounts: map[string]string{
			"/data": "/data",
		},
		EnvVars: map[string]string{
			"DATASET_PATH": datasetPath,
			"BASE_MODEL":   job.BaseModel(),
			"TARGET_MODEL": job.TargetModel(),
		},
		Command: "python train.py",
		Input:   job.Config(),
	}

	// Create RunPod job
	runpodJobID, err := fe.runpodClient.CreateJob(ctx, config)
	if err != nil {
		job.SetStatus(entities.StatusFailed)
		job.SetErrorMessage(fmt.Sprintf("failed to create RunPod job: %v", err))
		return fmt.Errorf("failed to create RunPod job: %w", err)
	}

	job.SetRunpodJobID(runpodJobID)

	// Save job
	if err := fe.store.SaveJob(ctx, job); err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	return nil
}

// CheckStatus checks the status of a training job
func (fe *FinetuningEngine) CheckStatus(ctx context.Context, jobID string) (*entities.TrainingJob, error) {
	job, err := fe.store.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if job.RunpodJobID() == "" {
		return job, nil // Job not started yet
	}

	// Check RunPod status
	status, err := fe.runpodClient.GetJobStatus(ctx, job.RunpodJobID())
	if err != nil {
		return nil, fmt.Errorf("failed to get RunPod status: %w", err)
	}

	// Update job status
	switch status.Status {
	case "RUNNING":
		job.SetStatus(entities.StatusRunning)
	case "COMPLETED":
		job.SetStatus(entities.StatusCompleted)
		// Download checkpoint
		if checkpointPath, ok := status.Output["checkpoint_path"].(string); ok {
			job.SetCheckpointPath(checkpointPath)
		}
	case "FAILED":
		job.SetStatus(entities.StatusFailed)
		job.SetErrorMessage(status.Error)
	case "CANCELLED":
		job.SetStatus(entities.StatusCancelled)
	}

	// Update metrics
	if metrics, ok := status.Output["metrics"].(map[string]interface{}); ok {
		for k, v := range metrics {
			if f, ok := v.(float64); ok {
				job.SetMetric(k, f)
			}
		}
	}

	// Save updated job
	if err := fe.store.SaveJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to save job: %w", err)
	}

	return job, nil
}

// CancelTraining cancels a training job
func (fe *FinetuningEngine) CancelTraining(ctx context.Context, jobID string) error {
	job, err := fe.store.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.RunpodJobID() == "" {
		job.SetStatus(entities.StatusCancelled)
		return fe.store.SaveJob(ctx, job)
	}

	// Cancel RunPod job
	if err := fe.runpodClient.CancelJob(ctx, job.RunpodJobID()); err != nil {
		return fmt.Errorf("failed to cancel RunPod job: %w", err)
	}

	job.SetStatus(entities.StatusCancelled)
	return fe.store.SaveJob(ctx, job)
}

// GetLogs retrieves training logs
func (fe *FinetuningEngine) GetLogs(ctx context.Context, jobID string) ([]string, error) {
	job, err := fe.store.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if job.RunpodJobID() == "" {
		return []string{}, nil
	}

	return fe.runpodClient.GetJobLogs(ctx, job.RunpodJobID())
}

// CompleteTraining completes training and creates model version
func (fe *FinetuningEngine) CompleteTraining(ctx context.Context, jobID string) (*entities.ModelVersion, error) {
	job, err := fe.store.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if job.Status() != entities.StatusCompleted {
		return nil, fmt.Errorf("job is not completed")
	}

	// Create model version
	version, err := fe.versioning.CreateVersion(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return version, nil
}

// Rollback rolls back to a previous model version
func (fe *FinetuningEngine) Rollback(ctx context.Context, modelName string, version int) error {
	return fe.versioning.Rollback(ctx, modelName, version)
}

// MonitorJobs monitors all active jobs
func (fe *FinetuningEngine) MonitorJobs(ctx context.Context) error {
	jobs, err := fe.store.GetActiveJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active jobs: %w", err)
	}

	for _, job := range jobs {
		_, err := fe.CheckStatus(ctx, job.ID())
		if err != nil {
			// Log error but continue
			_ = err
		}
	}

	return nil
}
