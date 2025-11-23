// Package serverless provides serverless compute implementations
package serverless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// RunPodClient provides RunPod API operations
type RunPodClient interface {
	// CreateJob creates a new RunPod job
	CreateJob(ctx context.Context, config *RunPodJobConfig) (string, error)

	// GetJobStatus gets the status of a RunPod job
	GetJobStatus(ctx context.Context, jobID string) (*RunPodJobStatus, error)

	// CancelJob cancels a RunPod job
	CancelJob(ctx context.Context, jobID string) error

	// GetJobLogs gets logs from a RunPod job
	GetJobLogs(ctx context.Context, jobID string) ([]string, error)

	// ListJobs lists all jobs
	ListJobs(ctx context.Context, limit int) ([]*RunPodJob, error)
}

// RunPodJobConfig represents RunPod job configuration
type RunPodJobConfig struct {
	Image       string
	Command     []string
	Environment map[string]string
	GPUType     string
	GPUCount    int
	VolumeMount string
	Timeout     time.Duration
}

// RunPodJobStatus represents RunPod job status
type RunPodJobStatus struct {
	ID        string
	Status    string
	Progress  float64
	StartedAt time.Time
	EndedAt   *time.Time
	Error     string
}

// RunPodJob represents a RunPod job
type RunPodJob struct {
	ID        string
	Status    string
	CreatedAt time.Time
	Config    *RunPodJobConfig
}

// runpodClient implements RunPodClient
type runpodClient struct {
	apiKey  string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

// NewRunPodClient creates a new RunPod client
func NewRunPodClient(apiKey string, baseURL string, timeout time.Duration) RunPodClient {
	if baseURL == "" {
		baseURL = "https://api.runpod.io/v1"
	}
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &runpodClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CreateJob creates a new RunPod job
func (c *runpodClient) CreateJob(ctx context.Context, config *RunPodJobConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config cannot be nil")
	}
	if config.Image == "" {
		return "", fmt.Errorf("image cannot be empty")
	}

	logger.Info("Creating RunPod job",
		zap.String("image", config.Image),
		zap.String("gpu_type", config.GPUType),
		zap.Int("gpu_count", config.GPUCount),
	)

	// Build request payload
	payload := map[string]interface{}{
		"image":               config.Image,
		"gpuTypeId":           config.GPUType,
		"gpuCount":            config.GPUCount,
		"containerDiskSizeGb": 20,
	}

	if len(config.Command) > 0 {
		payload["command"] = config.Command
	}
	if len(config.Environment) > 0 {
		payload["env"] = config.Environment
	}
	if config.VolumeMount != "" {
		payload["volumeMountPath"] = config.VolumeMount
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/pods", c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// GetJobStatus gets the status of a RunPod job
func (c *runpodClient) GetJobStatus(ctx context.Context, jobID string) (*RunPodJobStatus, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID cannot be empty")
	}

	logger.Debug("Getting RunPod job status",
		zap.String("job_id", jobID),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/pods/%s", c.baseURL, jobID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID        string     `json:"id"`
		Status    string     `json:"status"`
		Progress  float64    `json:"progress"`
		StartedAt time.Time  `json:"startedAt"`
		EndedAt   *time.Time `json:"endedAt,omitempty"`
		Error     string     `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &RunPodJobStatus{
		ID:        result.ID,
		Status:    result.Status,
		Progress:  result.Progress,
		StartedAt: result.StartedAt,
		EndedAt:   result.EndedAt,
		Error:     result.Error,
	}, nil
}

// CancelJob cancels a RunPod job
func (c *runpodClient) CancelJob(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("jobID cannot be empty")
	}

	logger.Info("Canceling RunPod job",
		zap.String("job_id", jobID),
	)

	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/pods/%s", c.baseURL, jobID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetJobLogs gets logs from a RunPod job
func (c *runpodClient) GetJobLogs(ctx context.Context, jobID string) ([]string, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobID cannot be empty")
	}

	logger.Debug("Getting RunPod job logs",
		zap.String("job_id", jobID),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/pods/%s/logs", c.baseURL, jobID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Logs []string `json:"logs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Logs, nil
}

// ListJobs lists all jobs
func (c *runpodClient) ListJobs(ctx context.Context, limit int) ([]*RunPodJob, error) {
	if limit <= 0 {
		limit = 100
	}

	logger.Debug("Listing RunPod jobs",
		zap.Int("limit", limit),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/pods?limit=%d", c.baseURL, limit), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Jobs []struct {
			ID        string    `json:"id"`
			Status    string    `json:"status"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	jobs := make([]*RunPodJob, len(result.Jobs))
	for i, j := range result.Jobs {
		jobs[i] = &RunPodJob{
			ID:        j.ID,
			Status:    j.Status,
			CreatedAt: j.CreatedAt,
		}
	}

	return jobs, nil
}
