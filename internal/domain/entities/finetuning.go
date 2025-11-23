// Package entities provides domain entities
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TrainingStatus represents the status of a training job
type TrainingStatus string

const (
	StatusPending   TrainingStatus = "pending"
	StatusRunning   TrainingStatus = "running"
	StatusCompleted TrainingStatus = "completed"
	StatusFailed    TrainingStatus = "failed"
	StatusCancelled TrainingStatus = "cancelled"
)

// Dataset represents a dataset for fine-tuning
type Dataset struct {
	id          string
	name        string
	description string
	filePath    string
	size        int64
	rowCount    int
	format      string
	version     int
	createdAt   time.Time
	updatedAt   time.Time
}

// NewDataset creates a new dataset entity
func NewDataset(name string, description string, filePath string) (*Dataset, error) {
	if name == "" {
		return nil, fmt.Errorf("dataset name cannot be empty")
	}
	if filePath == "" {
		return nil, fmt.Errorf("dataset file path cannot be empty")
	}

	now := time.Now()
	return &Dataset{
		id:          uuid.New().String(),
		name:        name,
		description: description,
		filePath:    filePath,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ID returns the dataset ID
func (d *Dataset) ID() string {
	return d.id
}

// Name returns the dataset name
func (d *Dataset) Name() string {
	return d.name
}

// Description returns the description
func (d *Dataset) Description() string {
	return d.description
}

// FilePath returns the file path
func (d *Dataset) FilePath() string {
	return d.filePath
}

// SetSize sets the dataset size
func (d *Dataset) SetSize(size int64, rowCount int) {
	d.size = size
	d.rowCount = rowCount
	d.updatedAt = time.Now()
}

// Size returns the dataset size
func (d *Dataset) Size() int64 {
	return d.size
}

// RowCount returns the row count
func (d *Dataset) RowCount() int {
	return d.rowCount
}

// Format returns the format
func (d *Dataset) Format() string {
	return d.format
}

// SetFormat sets the format
func (d *Dataset) SetFormat(format string) {
	d.format = format
	d.updatedAt = time.Now()
}

// Version returns the version
func (d *Dataset) Version() int {
	return d.version
}

// IncrementVersion increments the version
func (d *Dataset) IncrementVersion() {
	d.version++
	d.updatedAt = time.Now()
}

// CreatedAt returns creation timestamp
func (d *Dataset) CreatedAt() time.Time {
	return d.createdAt
}

// UpdatedAt returns update timestamp
func (d *Dataset) UpdatedAt() time.Time {
	return d.updatedAt
}

// TrainingJob represents a fine-tuning training job
type TrainingJob struct {
	id             string
	datasetID      string
	baseModel      string
	targetModel    string
	status         TrainingStatus
	runpodJobID    string
	checkpointPath string
	config         map[string]interface{}
	metrics        map[string]float64
	errorMessage   string
	createdAt      time.Time
	startedAt      *time.Time
	completedAt    *time.Time
}

// NewTrainingJob creates a new training job
func NewTrainingJob(datasetID string, baseModel string, targetModel string) (*TrainingJob, error) {
	if datasetID == "" {
		return nil, fmt.Errorf("dataset ID cannot be empty")
	}
	if baseModel == "" {
		return nil, fmt.Errorf("base model cannot be empty")
	}
	if targetModel == "" {
		return nil, fmt.Errorf("target model cannot be empty")
	}

	now := time.Now()
	return &TrainingJob{
		id:          uuid.New().String(),
		datasetID:   datasetID,
		baseModel:   baseModel,
		targetModel: targetModel,
		status:      StatusPending,
		config:      make(map[string]interface{}),
		metrics:     make(map[string]float64),
		createdAt:   now,
	}, nil
}

// ID returns the job ID
func (tj *TrainingJob) ID() string {
	return tj.id
}

// DatasetID returns the dataset ID
func (tj *TrainingJob) DatasetID() string {
	return tj.datasetID
}

// BaseModel returns the base model
func (tj *TrainingJob) BaseModel() string {
	return tj.baseModel
}

// TargetModel returns the target model
func (tj *TrainingJob) TargetModel() string {
	return tj.targetModel
}

// Status returns the status
func (tj *TrainingJob) Status() TrainingStatus {
	return tj.status
}

// SetStatus sets the status
func (tj *TrainingJob) SetStatus(status TrainingStatus) {
	tj.status = status
	now := time.Now()
	if status == StatusRunning && tj.startedAt == nil {
		tj.startedAt = &now
	}
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		tj.completedAt = &now
	}
}

// RunpodJobID returns the RunPod job ID
func (tj *TrainingJob) RunpodJobID() string {
	return tj.runpodJobID
}

// SetRunpodJobID sets the RunPod job ID
func (tj *TrainingJob) SetRunpodJobID(jobID string) {
	tj.runpodJobID = jobID
}

// CheckpointPath returns the checkpoint path
func (tj *TrainingJob) CheckpointPath() string {
	return tj.checkpointPath
}

// SetCheckpointPath sets the checkpoint path
func (tj *TrainingJob) SetCheckpointPath(path string) {
	tj.checkpointPath = path
}

// Config returns a copy of config
func (tj *TrainingJob) Config() map[string]interface{} {
	return copyMetadata(tj.config)
}

// SetConfig sets config
func (tj *TrainingJob) SetConfig(config map[string]interface{}) {
	tj.config = copyMetadata(config)
}

// Metrics returns a copy of metrics
func (tj *TrainingJob) Metrics() map[string]float64 {
	metrics := make(map[string]float64)
	for k, v := range tj.metrics {
		metrics[k] = v
	}
	return metrics
}

// SetMetric sets a metric
func (tj *TrainingJob) SetMetric(key string, value float64) {
	tj.metrics[key] = value
}

// ErrorMessage returns the error message
func (tj *TrainingJob) ErrorMessage() string {
	return tj.errorMessage
}

// SetErrorMessage sets the error message
func (tj *TrainingJob) SetErrorMessage(msg string) {
	tj.errorMessage = msg
}

// CreatedAt returns creation timestamp
func (tj *TrainingJob) CreatedAt() time.Time {
	return tj.createdAt
}

// StartedAt returns start timestamp
func (tj *TrainingJob) StartedAt() *time.Time {
	return tj.startedAt
}

// CompletedAt returns completion timestamp
func (tj *TrainingJob) CompletedAt() *time.Time {
	return tj.completedAt
}

// ModelVersion represents a versioned model
type ModelVersion struct {
	id             string
	modelName      string
	version        int
	jobID          string
	checkpointPath string
	metrics        map[string]float64
	isActive       bool
	createdAt      time.Time
}

// NewModelVersion creates a new model version
func NewModelVersion(modelName string, version int, jobID string, checkpointPath string) (*ModelVersion, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}
	if checkpointPath == "" {
		return nil, fmt.Errorf("checkpoint path cannot be empty")
	}

	now := time.Now()
	return &ModelVersion{
		id:             uuid.New().String(),
		modelName:      modelName,
		version:        version,
		jobID:          jobID,
		checkpointPath: checkpointPath,
		metrics:        make(map[string]float64),
		isActive:       false,
		createdAt:      now,
	}, nil
}

// ID returns the version ID
func (mv *ModelVersion) ID() string {
	return mv.id
}

// ModelName returns the model name
func (mv *ModelVersion) ModelName() string {
	return mv.modelName
}

// Version returns the version number
func (mv *ModelVersion) Version() int {
	return mv.version
}

// JobID returns the training job ID
func (mv *ModelVersion) JobID() string {
	return mv.jobID
}

// CheckpointPath returns the checkpoint path
func (mv *ModelVersion) CheckpointPath() string {
	return mv.checkpointPath
}

// Metrics returns a copy of metrics
func (mv *ModelVersion) Metrics() map[string]float64 {
	metrics := make(map[string]float64)
	for k, v := range mv.metrics {
		metrics[k] = v
	}
	return metrics
}

// SetMetric sets a metric
func (mv *ModelVersion) SetMetric(key string, value float64) {
	mv.metrics[key] = value
}

// IsActive returns whether this version is active
func (mv *ModelVersion) IsActive() bool {
	return mv.isActive
}

// SetActive sets active status
func (mv *ModelVersion) SetActive(active bool) {
	mv.isActive = active
}

// CreatedAt returns creation timestamp
func (mv *ModelVersion) CreatedAt() time.Time {
	return mv.createdAt
}
