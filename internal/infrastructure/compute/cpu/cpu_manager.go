// Package cpu provides CPU management implementations
package cpu

import (
	"context"
)

// CPUManager provides CPU management operations
type CPUManager interface {
	// GetUsage returns current CPU usage
	GetUsage(ctx context.Context) (float64, error)

	// SetLimit sets CPU limit
	SetLimit(ctx context.Context, limit float64) error

	// GetStats returns CPU statistics
	GetStats(ctx context.Context) (*CPUStats, error)
}

// CPUStats represents CPU statistics
type CPUStats struct {
	Usage   float64
	Limit   float64
	Cores   int
	LoadAvg []float64
}
