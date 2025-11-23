package models

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// DeploymentStrategy represents deployment strategy
type DeploymentStrategy string

const (
	DeploymentStrategyCanary    DeploymentStrategy = "canary"
	DeploymentStrategyBlueGreen DeploymentStrategy = "blue_green"
	DeploymentStrategyRolling   DeploymentStrategy = "rolling"
	DeploymentStrategyAllAtOnce DeploymentStrategy = "all_at_once"
)

// DeploymentStatus represents deployment status
type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"
	DeploymentStatusRunning    DeploymentStatus = "running"
	DeploymentStatusCompleted  DeploymentStatus = "completed"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusRolledBack DeploymentStatus = "rolled_back"
)

// Deployment represents a model deployment
type Deployment struct {
	ID             string                 `json:"id"`
	ModelID        string                 `json:"model_id"`
	VersionID      string                 `json:"version_id"`
	Strategy       DeploymentStrategy     `json:"strategy"`
	Status         DeploymentStatus       `json:"status"`
	Targets        []DeploymentTarget     `json:"targets"`
	CanaryPercent  float64                `json:"canary_percent,omitempty"`
	HealthChecks   HealthCheckConfig      `json:"health_checks"`
	RollbackPolicy RollbackPolicy         `json:"rollback_policy"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedBy      string                 `json:"created_by"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// DeploymentTarget represents a deployment target
type DeploymentTarget struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // "endpoint", "service", "region"
	Location string                 `json:"location"`
	Status   DeploymentStatus       `json:"status"`
	Metadata map[string]interface{} `json:"metadata"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	MaxFailures      int           `json:"max_failures"`
	SuccessThreshold float64       `json:"success_threshold"`
}

// RollbackPolicy represents rollback policy
type RollbackPolicy struct {
	AutoRollback   bool    `json:"auto_rollback"`
	MaxErrorRate   float64 `json:"max_error_rate"`
	MaxLatencyMs   float64 `json:"max_latency_ms"`
	MinSuccessRate float64 `json:"min_success_rate"`
}

// DeploymentMetrics represents deployment metrics
type DeploymentMetrics struct {
	Requests    int64   `json:"requests"`
	Errors      int64   `json:"errors"`
	LatencyMs   float64 `json:"latency_ms"`
	SuccessRate float64 `json:"success_rate"`
	ErrorRate   float64 `json:"error_rate"`
}

// ModelDeployment interface for model deployment operations
type ModelDeployment interface {
	// CreateDeployment creates a new deployment
	CreateDeployment(ctx context.Context, deployment *Deployment) (*Deployment, error)

	// GetDeployment retrieves a deployment
	GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error)

	// StartDeployment starts a deployment
	StartDeployment(ctx context.Context, deploymentID string) error

	// StopDeployment stops a deployment
	StopDeployment(ctx context.Context, deploymentID string) error

	// RollbackDeployment rolls back a deployment
	RollbackDeployment(ctx context.Context, deploymentID string) error

	// GetDeploymentMetrics gets metrics for a deployment
	GetDeploymentMetrics(ctx context.Context, deploymentID string) (*DeploymentMetrics, error)

	// CheckHealth checks health of a deployment
	CheckHealth(ctx context.Context, deploymentID string) (bool, error)

	// ListDeployments lists deployments for a model
	ListDeployments(ctx context.Context, modelID string) ([]*Deployment, error)

	// GetActiveDeployment gets active deployment for a model
	GetActiveDeployment(ctx context.Context, modelID string) (*Deployment, error)
}

// InMemoryModelDeployment implements ModelDeployment
type InMemoryModelDeployment struct {
	deployments map[string]*Deployment
	metrics     map[string]*DeploymentMetrics
	mu          sync.RWMutex
	logger      *zap.Logger
}

// NewInMemoryModelDeployment creates a new model deployment instance
func NewInMemoryModelDeployment() *InMemoryModelDeployment {
	return &InMemoryModelDeployment{
		deployments: make(map[string]*Deployment),
		metrics:     make(map[string]*DeploymentMetrics),
		logger:      logger.WithContext(context.Background()),
	}
}

// CreateDeployment creates a new deployment
func (md *InMemoryModelDeployment) CreateDeployment(ctx context.Context, deployment *Deployment) (*Deployment, error) {
	logger := logger.WithContext(ctx)
	logger.Info("Creating deployment",
		zap.String("model_id", deployment.ModelID),
		zap.String("version_id", deployment.VersionID))

	if deployment.ID == "" {
		deployment.ID = uuid.New().String()
	}

	if deployment.CreatedAt.IsZero() {
		deployment.CreatedAt = time.Now()
	}

	if deployment.CreatedBy == "" {
		deployment.CreatedBy = getCurrentUser(ctx)
	}

	if deployment.Status == "" {
		deployment.Status = DeploymentStatusPending
	}

	if deployment.Metadata == nil {
		deployment.Metadata = make(map[string]interface{})
	}

	// Initialize metrics
	md.mu.Lock()
	md.deployments[deployment.ID] = deployment
	md.metrics[deployment.ID] = &DeploymentMetrics{}
	md.mu.Unlock()

	logger.Info("Deployment created", zap.String("deployment_id", deployment.ID))
	return deployment, nil
}

// GetDeployment retrieves a deployment
func (md *InMemoryModelDeployment) GetDeployment(ctx context.Context, deploymentID string) (*Deployment, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	deployment, exists := md.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}

	return deployment, nil
}

// StartDeployment starts a deployment
func (md *InMemoryModelDeployment) StartDeployment(ctx context.Context, deploymentID string) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	logger := logger.WithContext(ctx)

	deployment, exists := md.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	if deployment.Status != DeploymentStatusPending {
		return fmt.Errorf("deployment must be pending to start")
	}

	deployment.Status = DeploymentStatusRunning
	now := time.Now()
	deployment.StartedAt = &now

	logger.Info("Deployment started", zap.String("deployment_id", deploymentID))
	return nil
}

// StopDeployment stops a deployment
func (md *InMemoryModelDeployment) StopDeployment(ctx context.Context, deploymentID string) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	logger := logger.WithContext(ctx)

	deployment, exists := md.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	if deployment.Status != DeploymentStatusRunning {
		return fmt.Errorf("deployment must be running to stop")
	}

	deployment.Status = DeploymentStatusCompleted
	now := time.Now()
	deployment.CompletedAt = &now

	logger.Info("Deployment stopped", zap.String("deployment_id", deploymentID))
	return nil
}

// RollbackDeployment rolls back a deployment
func (md *InMemoryModelDeployment) RollbackDeployment(ctx context.Context, deploymentID string) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	logger := logger.WithContext(ctx)

	deployment, exists := md.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment %s not found", deploymentID)
	}

	if deployment.Status != DeploymentStatusRunning && deployment.Status != DeploymentStatusCompleted {
		return fmt.Errorf("cannot rollback deployment in status %s", deployment.Status)
	}

	deployment.Status = DeploymentStatusRolledBack
	now := time.Now()
	deployment.CompletedAt = &now

	logger.Info("Deployment rolled back", zap.String("deployment_id", deploymentID))
	return nil
}

// GetDeploymentMetrics gets metrics for a deployment
func (md *InMemoryModelDeployment) GetDeploymentMetrics(ctx context.Context, deploymentID string) (*DeploymentMetrics, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	metrics, exists := md.metrics[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment %s not found", deploymentID)
	}

	return metrics, nil
}

// CheckHealth checks health of a deployment
func (md *InMemoryModelDeployment) CheckHealth(ctx context.Context, deploymentID string) (bool, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	deployment, exists := md.deployments[deploymentID]
	if !exists {
		return false, fmt.Errorf("deployment %s not found", deploymentID)
	}

	if deployment.Status != DeploymentStatusRunning {
		return false, nil
	}

	metrics, exists := md.metrics[deploymentID]
	if !exists {
		return true, nil // No metrics yet, assume healthy
	}

	// Check against rollback policy
	if deployment.RollbackPolicy.AutoRollback {
		if metrics.Requests > 0 {
			errorRate := float64(metrics.Errors) / float64(metrics.Requests)
			if errorRate > deployment.RollbackPolicy.MaxErrorRate {
				return false, nil
			}

			if metrics.LatencyMs > deployment.RollbackPolicy.MaxLatencyMs {
				return false, nil
			}

			if metrics.SuccessRate < deployment.RollbackPolicy.MinSuccessRate {
				return false, nil
			}
		}
	}

	return true, nil
}

// ListDeployments lists deployments for a model
func (md *InMemoryModelDeployment) ListDeployments(ctx context.Context, modelID string) ([]*Deployment, error) {
	md.mu.RLock()
	defer md.mu.RUnlock()

	var deployments []*Deployment
	for _, deployment := range md.deployments {
		if deployment.ModelID == modelID {
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// GetActiveDeployment gets active deployment for a model
func (md *InMemoryModelDeployment) GetActiveDeployment(ctx context.Context, modelID string) (*Deployment, error) {
	deployments, err := md.ListDeployments(ctx, modelID)
	if err != nil {
		return nil, err
	}

	for _, deployment := range deployments {
		if deployment.Status == DeploymentStatusRunning {
			return deployment, nil
		}
	}

	return nil, fmt.Errorf("no active deployment found for model %s", modelID)
}
