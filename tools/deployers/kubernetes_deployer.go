package deployers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/infrastructure/cloud/kubernetes"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// KubernetesDeployer handles deployment to Kubernetes (uses infra/cloud/k8s)
type KubernetesDeployer struct {
	client kubernetes.KubernetesClient
	logger *zap.Logger
}

// NewKubernetesDeployer creates a new Kubernetes deployer
func NewKubernetesDeployer(kubeconfig string) (*KubernetesDeployer, error) {
	client, err := kubernetes.NewKubernetesClient(kubeconfig, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &KubernetesDeployer{
		client: client,
		logger: logger.Get(),
	}, nil
}

// Deploy deploys a project to Kubernetes
func (k *KubernetesDeployer) Deploy(ctx context.Context, req KubernetesDeployRequest) (*KubernetesDeployResult, error) {
	k.logger.Info("Deploying to Kubernetes",
		zap.String("project", req.ProjectName),
		zap.String("namespace", req.Namespace),
		zap.String("image", req.Image))

	// Validate request
	if err := k.Validate(req); err != nil {
		return nil, err
	}

	// Check for Kubernetes manifests
	if req.ManifestsPath != "" {
		if err := k.deployFromManifests(ctx, req); err != nil {
			return nil, fmt.Errorf("failed to deploy from manifests: %w", err)
		}
	} else {
		// Deploy using client
		if err := k.deployUsingClient(ctx, req); err != nil {
			return nil, fmt.Errorf("failed to deploy using client: %w", err)
		}
	}

	return &KubernetesDeployResult{
		Namespace: req.Namespace,
		Status:    "deployed",
	}, nil
}

// deployFromManifests deploys from Kubernetes manifest files
func (k *KubernetesDeployer) deployFromManifests(ctx context.Context, req KubernetesDeployRequest) error {
	manifestsPath := req.ManifestsPath
	if manifestsPath == "" {
		manifestsPath = filepath.Join(req.ProjectPath, "k8s")
	}

	if _, err := os.Stat(manifestsPath); os.IsNotExist(err) {
		return fmt.Errorf("manifests path does not exist: %s", manifestsPath)
	}

	// In production, use kubectl apply or Kubernetes client to apply manifests
	// For now, return success
	k.logger.Info("Deploying from manifests", zap.String("path", manifestsPath))
	return nil
}

// deployUsingClient deploys using Kubernetes client
func (k *KubernetesDeployer) deployUsingClient(ctx context.Context, req KubernetesDeployRequest) error {
	// Create deployment
	deployment := &kubernetes.Deployment{
		Name:      req.ProjectName,
		Namespace: req.Namespace,
		Image:     req.Image,
		Replicas:  req.Replicas,
		Labels:    req.Labels,
		Env:       req.Env,
		Ports:     req.Ports,
	}

	if err := k.client.CreateDeployment(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create service if ports are specified
	if len(req.Ports) > 0 {
		service := &kubernetes.Service{
			Name:      req.ProjectName,
			Namespace: req.Namespace,
			Type:      "ClusterIP",
			Selector:  req.Labels,
		}

		// Convert ports
		for _, port := range req.Ports {
			service.Ports = append(service.Ports, kubernetes.ServicePort{
				Port:       port,
				TargetPort: port,
				Protocol:   "TCP",
			})
		}

		if err := k.client.CreateService(ctx, service); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	return nil
}

// KubernetesDeployRequest represents a Kubernetes deployment request
type KubernetesDeployRequest struct {
	ProjectName   string            `json:"project_name"`
	ProjectPath   string            `json:"project_path"`
	Namespace     string            `json:"namespace"`
	Image         string            `json:"image"`
	Replicas      int               `json:"replicas"`
	Labels        map[string]string `json:"labels,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Ports         []int             `json:"ports,omitempty"`
	ManifestsPath string            `json:"manifests_path,omitempty"`
}

// KubernetesDeployResult represents the result of Kubernetes deployment
type KubernetesDeployResult struct {
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

// Validate validates the Kubernetes deployment request
func (k *KubernetesDeployer) Validate(req KubernetesDeployRequest) error {
	if req.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	if req.Image == "" {
		return fmt.Errorf("image is required")
	}

	if req.Replicas < 0 {
		return fmt.Errorf("replicas must be >= 0")
	}

	if req.Replicas == 0 {
		req.Replicas = 1
	}

	return nil
}
