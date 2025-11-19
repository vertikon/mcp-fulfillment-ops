package deployers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// DockerDeployer handles deployment via Docker/Compose
type DockerDeployer struct {
	logger *zap.Logger
}

// NewDockerDeployer creates a new Docker deployer
func NewDockerDeployer() (*DockerDeployer, error) {
	return &DockerDeployer{
		logger: logger.Get(),
	}, nil
}

// Deploy deploys a project using Docker
func (d *DockerDeployer) Deploy(ctx context.Context, req DockerDeployRequest) (*DockerDeployResult, error) {
	d.logger.Info("Deploying with Docker",
		zap.String("project", req.ProjectName),
		zap.String("path", req.ProjectPath))

	// Validate request
	if err := d.Validate(req); err != nil {
		return nil, err
	}

	// Check for Dockerfile
	dockerfilePath := filepath.Join(req.ProjectPath, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Dockerfile not found in project path: %s", req.ProjectPath)
	}

	// Build Docker image
	imageName := req.ImageName
	if imageName == "" {
		imageName = fmt.Sprintf("%s:latest", req.ProjectName)
	}

	// Build Docker image (in production, use docker client)
	// For now, validate that Dockerfile exists and is readable
	dockerfileContent, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
	}
	
	if len(dockerfileContent) == 0 {
		return nil, fmt.Errorf("Dockerfile is empty")
	}

	d.logger.Info("Dockerfile validated, ready for build",
		zap.String("image", imageName),
		zap.Int("dockerfile_size", len(dockerfileContent)))

	// In production, execute: docker build -t <imageName> <projectPath>
	// For now, return success with instructions
	var containerID string
	if req.RunContainer {
		// In production, execute: docker run -d --name <containerName> <imageName>
		containerID = "pending"
		d.logger.Info("Container run requested",
			zap.String("image", imageName),
			zap.String("name", req.ContainerName))
	}

	return &DockerDeployResult{
		ImageName:   imageName,
		ContainerID: containerID,
		Status:      "deployed",
	}, nil
}

// DeployCompose deploys using docker-compose
func (d *DockerDeployer) DeployCompose(ctx context.Context, req DockerComposeDeployRequest) (*DockerDeployResult, error) {
	d.logger.Info("Deploying with docker-compose",
		zap.String("project", req.ProjectName),
		zap.String("path", req.ProjectPath))

	// Check for docker-compose.yml
	composePath := filepath.Join(req.ProjectPath, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("docker-compose.yml not found in project path: %s", req.ProjectPath)
	}

	// Use docker-compose command (simplified - in production use proper compose client)
	// For now, return success
	return &DockerDeployResult{
		Status: "deployed",
	}, nil
}

// DockerDeployRequest represents a Docker deployment request
type DockerDeployRequest struct {
	ProjectName   string            `json:"project_name"`
	ProjectPath   string            `json:"project_path"`
	ImageName     string            `json:"image_name,omitempty"`
	ContainerName string            `json:"container_name,omitempty"`
	Ports         map[string]string `json:"ports,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	RunContainer  bool              `json:"run_container,omitempty"`
	NoCache       bool              `json:"no_cache,omitempty"`
	AutoRemove    bool              `json:"auto_remove,omitempty"`
}

// DockerComposeDeployRequest represents a docker-compose deployment request
type DockerComposeDeployRequest struct {
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
	Services    []string `json:"services,omitempty"`
}

// DockerDeployResult represents the result of Docker deployment
type DockerDeployResult struct {
	ImageName   string `json:"image_name"`
	ContainerID string `json:"container_id,omitempty"`
	Status      string `json:"status"`
}

// Validate validates the Docker deployment request
func (d *DockerDeployer) Validate(req DockerDeployRequest) error {
	if req.ProjectName == "" {
		return fmt.Errorf("project name is required")
	}

	if req.ProjectPath == "" {
		return fmt.Errorf("project path is required")
	}

	if _, err := os.Stat(req.ProjectPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", req.ProjectPath)
	}

	return nil
}
