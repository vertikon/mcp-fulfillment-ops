// Package serverless provides serverless compute implementations
package serverless

import (
	"context"
)

// CloudFunctions provides cloud functions operations (GCP/Azure)
type CloudFunctions interface {
	// Invoke invokes a cloud function
	Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error)

	// CreateFunction creates a cloud function
	CreateFunction(ctx context.Context, config *FunctionConfig) error

	// DeleteFunction deletes a cloud function
	DeleteFunction(ctx context.Context, functionName string) error
}

// FunctionConfig represents the configuration required to deploy a serverless function.
// The struct mirrors the cloud-specific versions so callers can share the same shape.
type FunctionConfig struct {
	Name    string
	Runtime string
	Code    []byte
	Timeout int
}
