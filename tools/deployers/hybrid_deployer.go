package deployers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// HybridDeployer handles hybrid deployment combining K8s + Serverless + Docker
// This deployer intelligently selects the best deployment strategy based on workload characteristics
type HybridDeployer struct {
	dockerDeployer     *DockerDeployer
	kubernetesDeployer *KubernetesDeployer
	serverlessDeployer *ServerlessDeployer
	logger             *zap.Logger
}

// NewHybridDeployer creates a new hybrid deployer
func NewHybridDeployer(kubeconfig string) (*HybridDeployer, error) {
	dockerDeployer, err := NewDockerDeployer()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker deployer: %w", err)
	}

	kubernetesDeployer, err := NewKubernetesDeployer(kubeconfig)
	if err != nil {
		// Kubernetes deployer is optional for hybrid mode
		logger.Get().Warn("Failed to create Kubernetes deployer, continuing without it",
			zap.Error(err))
	}

	serverlessDeployer, err := NewServerlessDeployer("aws") // Default provider
	if err != nil {
		return nil, fmt.Errorf("failed to create Serverless deployer: %w", err)
	}

	return &HybridDeployer{
		dockerDeployer:     dockerDeployer,
		kubernetesDeployer: kubernetesDeployer,
		serverlessDeployer: serverlessDeployer,
		logger:             logger.Get(),
	}, nil
}

// Deploy deploys a project using hybrid strategy
// It analyzes the project and selects the best deployment approach
func (h *HybridDeployer) Deploy(ctx context.Context, req HybridDeployRequest) (*HybridDeployResult, error) {
	h.logger.Info("Deploying with hybrid strategy",
		zap.String("project", req.ProjectName),
		zap.String("path", req.ProjectPath),
		zap.String("strategy", req.Strategy))

	// Validate request
	if err := h.Validate(req); err != nil {
		return nil, err
	}

	// Determine deployment strategy
	strategy := h.determineStrategy(req)

	// Execute deployment based on strategy
	var result *HybridDeployResult
	var err error

	switch strategy {
	case "kubernetes":
		result, err = h.deployKubernetes(ctx, req)
	case "serverless":
		result, err = h.deployServerless(ctx, req)
	case "docker":
		result, err = h.deployDocker(ctx, req)
	case "multi":
		result, err = h.deployMulti(ctx, req)
	default:
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}

	if err != nil {
		return nil, fmt.Errorf("hybrid deployment failed: %w", err)
	}

	return result, nil
}

// determineStrategy determines the best deployment strategy
func (h *HybridDeployer) determineStrategy(req HybridDeployRequest) string {
	// If strategy is explicitly set, use it
	if req.Strategy != "" {
		return req.Strategy
	}

	// Auto-detect based on project characteristics
	projectPath := req.ProjectPath

	// Check for Kubernetes manifests
	if h.hasKubernetesManifests(projectPath) {
		return "kubernetes"
	}

	// Check for serverless configuration
	if h.hasServerlessConfig(projectPath) {
		return "serverless"
	}

	// Check for Dockerfile
	if h.hasDockerfile(projectPath) {
		return "docker"
	}

	// Default to multi-strategy if multiple options available
	if h.hasDockerfile(projectPath) && h.hasKubernetesManifests(projectPath) {
		return "multi"
	}

	// Default to Docker
	return "docker"
}

// deployKubernetes deploys using Kubernetes
func (h *HybridDeployer) deployKubernetes(ctx context.Context, req HybridDeployRequest) (*HybridDeployResult, error) {
	if h.kubernetesDeployer == nil {
		return nil, fmt.Errorf("Kubernetes deployer not available")
	}

	k8sReq := KubernetesDeployRequest{
		ProjectName:   req.ProjectName,
		ProjectPath:   req.ProjectPath,
		Namespace:     req.Namespace,
		Image:         req.Image,
		Replicas:      req.Replicas,
		Labels:        req.Labels,
		Env:           req.Env,
		Ports:         req.Ports,
		ManifestsPath: req.ManifestsPath,
	}

	k8sResult, err := h.kubernetesDeployer.Deploy(ctx, k8sReq)
	if err != nil {
		return nil, err
	}

	return &HybridDeployResult{
		Strategy:    "kubernetes",
		ProjectName: req.ProjectName,
		Status:      k8sResult.Status,
		Namespace:   k8sResult.Namespace,
		Deployments: []string{"kubernetes"},
	}, nil
}

// deployServerless deploys using serverless
func (h *HybridDeployer) deployServerless(ctx context.Context, req HybridDeployRequest) (*HybridDeployResult, error) {
	serverlessReq := ServerlessDeployRequest{
		FunctionName: req.ProjectName,
		FunctionPath: req.ProjectPath,
		Provider:     req.ServerlessProvider,
		Runtime:      req.Runtime,
		Handler:      req.Handler,
		Environment:  req.Env,
		Timeout:      req.Timeout,
		MemorySize:   req.MemorySize,
	}

	if serverlessReq.Provider == "" {
		serverlessReq.Provider = "aws"
	}
	if serverlessReq.Runtime == "" {
		serverlessReq.Runtime = "go"
	}
	if serverlessReq.Handler == "" {
		serverlessReq.Handler = "main"
	}

	serverlessResult, err := h.serverlessDeployer.Deploy(ctx, serverlessReq)
	if err != nil {
		return nil, err
	}

	return &HybridDeployResult{
		Strategy:    "serverless",
		ProjectName: req.ProjectName,
		Status:      serverlessResult.Status,
		Provider:    serverlessResult.Provider,
		FunctionURL: serverlessResult.URL,
		Deployments: []string{"serverless"},
	}, nil
}

// deployDocker deploys using Docker
func (h *HybridDeployer) deployDocker(ctx context.Context, req HybridDeployRequest) (*HybridDeployResult, error) {
	dockerReq := DockerDeployRequest{
		ProjectName:   req.ProjectName,
		ProjectPath:   req.ProjectPath,
		ImageName:     req.Image,
		ContainerName: req.ContainerName,
		Ports:         h.convertPorts(req.Ports),
		Env:           req.Env,
		RunContainer:  req.RunContainer,
	}

	dockerResult, err := h.dockerDeployer.Deploy(ctx, dockerReq)
	if err != nil {
		return nil, err
	}

	return &HybridDeployResult{
		Strategy:    "docker",
		ProjectName: req.ProjectName,
		Status:      dockerResult.Status,
		ImageName:   dockerResult.ImageName,
		ContainerID: dockerResult.ContainerID,
		Deployments: []string{"docker"},
	}, nil
}

// deployMulti deploys using multiple strategies simultaneously
func (h *HybridDeployer) deployMulti(ctx context.Context, req HybridDeployRequest) (*HybridDeployResult, error) {
	h.logger.Info("Deploying with multi-strategy",
		zap.String("project", req.ProjectName))

	var deployments []string
	var errors []error

	// Deploy to Docker
	if h.hasDockerfile(req.ProjectPath) {
		dockerResult, err := h.deployDocker(ctx, req)
		if err != nil {
			h.logger.Warn("Docker deployment failed", zap.Error(err))
			errors = append(errors, fmt.Errorf("docker: %w", err))
		} else {
			deployments = append(deployments, "docker")
			h.logger.Info("Docker deployment successful", zap.String("image", dockerResult.ImageName))
		}
	}

	// Deploy to Kubernetes if available
	if h.kubernetesDeployer != nil && h.hasKubernetesManifests(req.ProjectPath) {
		k8sResult, err := h.deployKubernetes(ctx, req)
		if err != nil {
			h.logger.Warn("Kubernetes deployment failed", zap.Error(err))
			errors = append(errors, fmt.Errorf("kubernetes: %w", err))
		} else {
			deployments = append(deployments, "kubernetes")
			h.logger.Info("Kubernetes deployment successful", zap.String("namespace", k8sResult.Namespace))
		}
	}

	// Deploy to Serverless if configured
	if h.hasServerlessConfig(req.ProjectPath) {
		serverlessResult, err := h.deployServerless(ctx, req)
		if err != nil {
			h.logger.Warn("Serverless deployment failed", zap.Error(err))
			errors = append(errors, fmt.Errorf("serverless: %w", err))
		} else {
			deployments = append(deployments, "serverless")
			h.logger.Info("Serverless deployment successful", zap.String("provider", serverlessResult.Provider))
		}
	}

	if len(deployments) == 0 {
		return nil, fmt.Errorf("all deployment strategies failed: %v", errors)
	}

	status := "partial"
	if len(errors) == 0 {
		status = "deployed"
	}

	return &HybridDeployResult{
		Strategy:    "multi",
		ProjectName: req.ProjectName,
		Status:      status,
		Deployments: deployments,
		Errors:      h.formatErrors(errors),
	}, nil
}

// Helper methods

func (h *HybridDeployer) hasDockerfile(path string) bool {
	dockerfilePath := filepath.Join(path, "Dockerfile")
	_, err := os.Stat(dockerfilePath)
	return err == nil
}

func (h *HybridDeployer) hasKubernetesManifests(path string) bool {
	k8sPath := filepath.Join(path, "k8s")
	if _, err := os.Stat(k8sPath); err == nil {
		return true
	}

	// Check for common manifest files
	manifestFiles := []string{"deployment.yaml", "service.yaml", "kustomization.yaml"}
	for _, file := range manifestFiles {
		manifestPath := filepath.Join(path, file)
		if _, err := os.Stat(manifestPath); err == nil {
			return true
		}
	}

	return false
}

func (h *HybridDeployer) hasServerlessConfig(path string) bool {
	// Check for serverless configuration files
	serverlessFiles := []string{"serverless.yml", "serverless.yaml", "sam.yaml", "template.yaml"}
	for _, file := range serverlessFiles {
		configPath := filepath.Join(path, file)
		if _, err := os.Stat(configPath); err == nil {
			return true
		}
	}

	return false
}

func (h *HybridDeployer) convertPorts(ports []int) map[string]string {
	result := make(map[string]string)
	for _, port := range ports {
		portStr := fmt.Sprintf("%d", port)
		result[portStr] = portStr
	}
	return result
}

func (h *HybridDeployer) formatErrors(errors []error) []string {
	if len(errors) == 0 {
		return nil
	}
	result := make([]string, len(errors))
	for i, err := range errors {
		result[i] = err.Error()
	}
	return result
}

// HybridDeployRequest represents a hybrid deployment request
type HybridDeployRequest struct {
	ProjectName        string            `json:"project_name"`
	ProjectPath        string            `json:"project_path"`
	Strategy           string            `json:"strategy,omitempty"` // kubernetes, serverless, docker, multi, auto
	Namespace          string            `json:"namespace,omitempty"`
	Image              string            `json:"image,omitempty"`
	Replicas           int               `json:"replicas,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Env                map[string]string `json:"env,omitempty"`
	Ports              []int             `json:"ports,omitempty"`
	ManifestsPath      string            `json:"manifests_path,omitempty"`
	ServerlessProvider string            `json:"serverless_provider,omitempty"` // aws, azure, gcp
	Runtime            string            `json:"runtime,omitempty"`
	Handler            string            `json:"handler,omitempty"`
	Timeout            int               `json:"timeout,omitempty"`
	MemorySize         int               `json:"memory_size,omitempty"`
	ContainerName      string            `json:"container_name,omitempty"`
	RunContainer       bool              `json:"run_container,omitempty"`
}

// HybridDeployResult represents the result of hybrid deployment
type HybridDeployResult struct {
	Strategy    string   `json:"strategy"`
	ProjectName string   `json:"project_name"`
	Status      string   `json:"status"`
	Deployments []string `json:"deployments"` // List of successful deployments

	// Kubernetes-specific fields
	Namespace string `json:"namespace,omitempty"`

	// Docker-specific fields
	ImageName   string `json:"image_name,omitempty"`
	ContainerID string `json:"container_id,omitempty"`

	// Serverless-specific fields
	Provider    string `json:"provider,omitempty"`
	FunctionURL string `json:"function_url,omitempty"`

	// Multi-strategy fields
	Errors []string `json:"errors,omitempty"`
}

// Validate validates the hybrid deployment request
func (h *HybridDeployer) Validate(req HybridDeployRequest) error {
	if req.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}

	if req.ProjectPath == "" {
		return fmt.Errorf("project path is required")
	}

	if _, err := os.Stat(req.ProjectPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", req.ProjectPath)
	}

	// Validate strategy if provided
	if req.Strategy != "" {
		validStrategies := []string{"kubernetes", "serverless", "docker", "multi", "auto"}
		valid := false
		for _, vs := range validStrategies {
			if req.Strategy == vs {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid strategy: %s (valid: %v)", req.Strategy, validStrategies)
		}
	}

	// Validate serverless provider if provided
	if req.ServerlessProvider != "" {
		validProviders := []string{"aws", "azure", "gcp"}
		valid := false
		for _, vp := range validProviders {
			if req.ServerlessProvider == vp {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid serverless provider: %s (valid: %v)", req.ServerlessProvider, validProviders)
		}
	}

	return nil
}

// GetDeploymentStatus returns the status of all deployments
func (h *HybridDeployer) GetDeploymentStatus(ctx context.Context, projectName string) (*HybridDeployStatus, error) {
	h.logger.Info("Getting deployment status",
		zap.String("project", projectName))

	status := &HybridDeployStatus{
		ProjectName: projectName,
		Deployments: make(map[string]DeploymentInfo),
	}

	// Check Kubernetes deployment status
	if h.kubernetesDeployer != nil {
		// In production, query Kubernetes API for deployment status
		status.Deployments["kubernetes"] = DeploymentInfo{
			Type:   "kubernetes",
			Status: "unknown", // Would query actual status in production
		}
	}

	// Check Docker deployment status
	status.Deployments["docker"] = DeploymentInfo{
		Type:   "docker",
		Status: "unknown", // Would query Docker daemon in production
	}

	// Check Serverless deployment status
	status.Deployments["serverless"] = DeploymentInfo{
		Type:   "serverless",
		Status: "unknown", // Would query provider API in production
	}

	return status, nil
}

// HybridDeployStatus represents the status of hybrid deployments
type HybridDeployStatus struct {
	ProjectName string                    `json:"project_name"`
	Deployments map[string]DeploymentInfo `json:"deployments"`
	LastUpdated time.Time                 `json:"last_updated"`
}

// DeploymentInfo represents information about a single deployment
type DeploymentInfo struct {
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	Details   map[string]string `json:"details,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}
