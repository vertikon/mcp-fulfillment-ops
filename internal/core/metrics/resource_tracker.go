// Package metrics provides resource tracking capabilities
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ResourceTracker tracks system resource usage
type ResourceTracker struct {
	mu             sync.RWMutex
	resources      map[string]*ResourceUsage
	limits         map[string]ResourceLimit
	collectInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

// ResourceUsage tracks usage for a specific resource
type ResourceUsage struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"` // cpu, memory, disk, network
	Current      float64   `json:"current"`
	Max          float64   `json:"max"`
	Average      float64   `json:"average"`
	LastUpdated  time.Time `json:"last_updated"`
	AlertTriggered bool     `json:"alert_triggered"`
	Samples      []float64 `json:"-"`
}

// ResourceLimit defines limits for a resource
type ResourceLimit struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
	Max      float64 `json:"max"`
}

// ResourceStats provides aggregated resource statistics
type ResourceStats struct {
	TotalResources    int                    `json:"total_resources"`
	HealthyResources int                    `json:"healthy_resources"`
	WarningResources int                    `json:"warning_resources"`
	CriticalResources int                   `json:"critical_resources"`
	Resources        map[string]ResourceUsage `json:"resources"`
	LastUpdate       time.Time              `json:"last_update"`
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker(interval time.Duration) *ResourceTracker {
	ctx, cancel := context.WithCancel(context.Background())
	
	rt := &ResourceTracker{
		resources:      make(map[string]*ResourceUsage),
		limits:         make(map[string]ResourceLimit),
		collectInterval: interval,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize default limits
	rt.initDefaultLimits()
	
	return rt
}

// initDefaultLimits sets up default resource limits
func (rt *ResourceTracker) initDefaultLimits() {
	rt.limits["cpu"] = ResourceLimit{
		Name:     "cpu",
		Type:     "cpu",
		Warning:  70.0,
		Critical: 90.0,
		Max:      100.0,
	}

	rt.limits["memory"] = ResourceLimit{
		Name:     "memory",
		Type:     "memory",
		Warning:  75.0,
		Critical: 90.0,
		Max:      100.0,
	}

	rt.limits["disk"] = ResourceLimit{
		Name:     "disk",
		Type:     "disk",
		Warning:  80.0,
		Critical: 95.0,
		Max:      100.0,
	}
}

// Start begins resource tracking
func (rt *ResourceTracker) Start() {
	logger.Info("Starting resource tracker", zap.Duration("interval", rt.collectInterval))

	go func() {
		ticker := time.NewTicker(rt.collectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rt.collectResourceMetrics()
			case <-rt.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops resource tracking
func (rt *ResourceTracker) Stop() {
	logger.Info("Stopping resource tracker")
	rt.cancel()
}

// collectResourceMetrics gathers current resource usage
func (rt *ResourceTracker) collectResourceMetrics() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// CPU usage
	cpuUsage := rt.getCPUUsage()
	rt.updateResource("cpu", cpuUsage)

	// Memory usage
	memUsage := rt.getMemoryUsage()
	rt.updateResource("memory", memUsage)

	// Disk usage
	diskUsage := rt.getDiskUsage()
	rt.updateResource("disk", diskUsage)

	logger.Debug("Resource metrics collected",
		zap.Float64("cpu", cpuUsage),
		zap.Float64("memory", memUsage),
		zap.Float64("disk", diskUsage),
	)
}

// updateResource updates resource usage data
func (rt *ResourceTracker) updateResource(name string, usage float64) {
	if _, exists := rt.resources[name]; !exists {
		rt.resources[name] = &ResourceUsage{
			Name:    name,
			Type:    name,
			Samples: make([]float64, 0, 100), // Keep last 100 samples
		}
	}

	resource := rt.resources[name]
	resource.Current = usage
	resource.LastUpdated = time.Now()

	// Update samples
	if len(resource.Samples) >= 100 {
		resource.Samples = resource.Samples[1:] // Remove oldest
	}
	resource.Samples = append(resource.Samples, usage)

	// Update statistics
	rt.calculateStats(resource)

	// Check limits
	rt.checkLimits(resource)
}

// calculateStats calculates statistical measures for resource
func (rt *ResourceTracker) calculateStats(resource *ResourceUsage) {
	if len(resource.Samples) == 0 {
		return
	}

	var sum float64
	max := resource.Samples[0]
	
	for _, sample := range resource.Samples {
		sum += sample
		if sample > max {
			max = sample
		}
	}

	resource.Average = sum / float64(len(resource.Samples))
	resource.Max = max
}

// checkLimits checks if resource usage exceeds limits
func (rt *ResourceTracker) checkLimits(resource *ResourceUsage) {
	limit, exists := rt.limits[resource.Name]
	if !exists {
		return
	}

	if resource.Current >= limit.Critical {
		if !resource.AlertTriggered {
			resource.AlertTriggered = true
			logger.Error("Resource critical limit exceeded",
				zap.String("resource", resource.Name),
				zap.Float64("current", resource.Current),
				zap.Float64("limit", limit.Critical),
			)
		}
	} else if resource.Current >= limit.Warning {
		logger.Warn("Resource warning limit exceeded",
			zap.String("resource", resource.Name),
			zap.Float64("current", resource.Current),
			zap.Float64("limit", limit.Warning),
		)
	} else {
		resource.AlertTriggered = false
	}
}

// getCPUUsage returns simulated CPU usage
func (rt *ResourceTracker) getCPUUsage() float64 {
	// In a real implementation, you'd use system calls to get actual CPU usage
	// For now, return a simulated value
	return 25.0 + (float64(time.Now().Unix() % 100) / 10)
}

// getMemoryUsage returns simulated memory usage percentage
func (rt *ResourceTracker) getMemoryUsage() float64 {
	// In a real implementation, you'd use system calls to get actual memory usage
	return 60.0 + (float64(time.Now().Unix() % 50) / 10)
}

// getDiskUsage returns simulated disk usage percentage
func (rt *ResourceTracker) getDiskUsage() float64 {
	// In a real implementation, you'd use system calls to get actual disk usage
	return 45.0 + (float64(time.Now().Unix() % 40) / 10)
}

// GetStats returns current resource statistics
func (rt *ResourceTracker) GetStats() ResourceStats {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	stats := ResourceStats{
		Resources:  make(map[string]ResourceUsage),
		LastUpdate:  time.Now(),
	}

	for name, resource := range rt.resources {
		stats.Resources[name] = *resource
		
		if resource.AlertTriggered {
			stats.CriticalResources++
		} else if limit, exists := rt.limits[name]; exists && resource.Current >= limit.Warning {
			stats.WarningResources++
		} else {
			stats.HealthyResources++
		}
	}

	stats.TotalResources = len(stats.Resources)

	return stats
}

// IsHealthy checks if all resources are within healthy limits
func (rt *ResourceTracker) IsHealthy() bool {
	stats := rt.GetStats()
	return stats.CriticalResources == 0
}

// GetResourceUsage returns usage for a specific resource
func (rt *ResourceTracker) GetResourceUsage(name string) (ResourceUsage, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	resource, exists := rt.resources[name]
	if !exists {
		return ResourceUsage{}, false
	}

	return *resource, true
}

// SetLimit sets a custom resource limit
func (rt *ResourceTracker) SetLimit(limit ResourceLimit) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.limits[limit.Name] = limit
	logger.Info("Resource limit set",
		zap.String("name", limit.Name),
		zap.Float64("warning", limit.Warning),
		zap.Float64("critical", limit.Critical),
	)
}