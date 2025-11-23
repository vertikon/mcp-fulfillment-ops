// Package serverless provides serverless compute implementations
package serverless

import (
	"context"
)

// FunctionOrchestrator orchestrates remote functions and jobs (includes RunPod driver via external clients)
type FunctionOrchestrator interface {
	// Orchestrate orchestrates a function execution
	Orchestrate(ctx context.Context, config *OrchestrationConfig) (string, error)

	// GetStatus gets the status of an orchestration
	GetStatus(ctx context.Context, orchestrationID string) (*OrchestrationStatus, error)

	// Cancel cancels an orchestration
	Cancel(ctx context.Context, orchestrationID string) error
}

// OrchestrationConfig represents orchestration configuration
type OrchestrationConfig struct {
	Type    string
	Config  map[string]interface{}
	Timeout int
	Retries int
}

// OrchestrationStatus represents orchestration status
type OrchestrationStatus struct {
	ID       string
	Status   string
	Progress float64
	Error    string
}
