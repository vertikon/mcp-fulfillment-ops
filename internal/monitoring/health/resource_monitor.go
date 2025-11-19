// Package health provides resource monitoring capabilities
package health

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ResourceType represents type of resource
type ResourceType string

const (
	ResourceTypeCPU     ResourceType = "cpu"
	ResourceTypeMemory  ResourceType = "memory"
	ResourceTypeDisk    ResourceType = "disk"
	ResourceTypeNetwork ResourceType = "network"
)

// ResourceUsage represents resource usage information
type ResourceUsage struct {
	Type       ResourceType           `json:"type"`
	Current    float64                `json:"current"`
	Max        float64                `json:"max"`
	Average    float64                `json:"average"`
	Percentage float64                `json:"percentage"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceMonitor monitors system resources
type ResourceMonitor struct {
	mu              sync.RWMutex
	resources       map[ResourceType]*ResourceUsage
	history         map[ResourceType][]ResourceUsage
	maxHistorySize  int
	collectInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
}

// ResourceStats provides resource statistics
type ResourceStats struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	NetworkIn   float64   `json:"network_in"`
	NetworkOut  float64   `json:"network_out"`
	Timestamp   time.Time `json:"timestamp"`
}

// ResourceConfig represents resource monitor configuration
type ResourceConfig struct {
	CollectInterval  time.Duration `json:"collect_interval"`
	MaxHistorySize   int           `json:"max_history_size"`
	EnableMonitoring bool          `json:"enable_monitoring"`
}

// DefaultResourceConfig returns default resource configuration
func DefaultResourceConfig() *ResourceConfig {
	return &ResourceConfig{
		CollectInterval:  10 * time.Second,
		MaxHistorySize:   100,
		EnableMonitoring: true,
	}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(config *ResourceConfig) *ResourceMonitor {
	if config == nil {
		config = DefaultResourceConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &ResourceMonitor{
		resources:       make(map[ResourceType]*ResourceUsage),
		history:         make(map[ResourceType][]ResourceUsage),
		maxHistorySize:  config.MaxHistorySize,
		collectInterval: config.CollectInterval,
		ctx:             ctx,
		cancel:          cancel,
	}

	logger.Info("Resource monitor initialized",
		zap.Duration("collect_interval", config.CollectInterval),
		zap.Int("max_history_size", config.MaxHistorySize))

	return monitor
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start() {
	logger.Info("Starting resource monitor",
		zap.Duration("interval", rm.collectInterval))

	go func() {
		ticker := time.NewTicker(rm.collectInterval)
		defer ticker.Stop()

		// Initial collection
		rm.collectResources()

		for {
			select {
			case <-ticker.C:
				rm.collectResources()
			case <-rm.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	logger.Info("Stopping resource monitor")
	rm.cancel()
}

// GetResourceUsage returns current resource usage
func (rm *ResourceMonitor) GetResourceUsage(resourceType ResourceType) (*ResourceUsage, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	usage, exists := rm.resources[resourceType]
	if !exists {
		return nil, false
	}

	copy := *usage
	return &copy, true
}

// GetAllResourceUsage returns all resource usage
func (rm *ResourceMonitor) GetAllResourceUsage() map[ResourceType]*ResourceUsage {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	usage := make(map[ResourceType]*ResourceUsage)
	for k, v := range rm.resources {
		copy := *v
		usage[k] = &copy
	}

	return usage
}

// GetResourceHistory returns history for a resource type
func (rm *ResourceMonitor) GetResourceHistory(resourceType ResourceType, limit int) []ResourceUsage {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	history, exists := rm.history[resourceType]
	if !exists {
		return []ResourceUsage{}
	}

	if limit > 0 && limit < len(history) {
		return history[len(history)-limit:]
	}

	result := make([]ResourceUsage, len(history))
	copy(result, history)
	return result
}

// GetStats returns current resource statistics
func (rm *ResourceMonitor) GetStats() ResourceStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := ResourceStats{
		Timestamp: time.Now(),
	}

	if cpu, exists := rm.resources[ResourceTypeCPU]; exists {
		stats.CPUUsage = cpu.Percentage
	}

	if mem, exists := rm.resources[ResourceTypeMemory]; exists {
		stats.MemoryUsage = mem.Percentage
	}

	if disk, exists := rm.resources[ResourceTypeDisk]; exists {
		stats.DiskUsage = disk.Percentage
	}

	return stats
}

// collectResources collects current resource usage
func (rm *ResourceMonitor) collectResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()

	// Collect CPU usage (simplified)
	cpuUsage := rm.collectCPUUsage()
	rm.updateResource(ResourceTypeCPU, cpuUsage, now)

	// Collect memory usage
	memUsage := rm.collectMemoryUsage()
	rm.updateResource(ResourceTypeMemory, memUsage, now)

	// Collect disk usage (simplified)
	diskUsage := rm.collectDiskUsage()
	rm.updateResource(ResourceTypeDisk, diskUsage, now)

	logger.Debug("Resource metrics collected",
		zap.Float64("cpu", cpuUsage),
		zap.Float64("memory", memUsage),
		zap.Float64("disk", diskUsage))
}

// updateResource updates resource usage data
func (rm *ResourceMonitor) updateResource(resourceType ResourceType, usage float64, timestamp time.Time) {
	resource, exists := rm.resources[resourceType]
	if !exists {
		resource = &ResourceUsage{
			Type:     resourceType,
			Metadata: make(map[string]interface{}),
		}
		rm.resources[resourceType] = resource
		rm.history[resourceType] = make([]ResourceUsage, 0)
	}

	resource.Current = usage
	resource.Percentage = usage
	resource.Timestamp = timestamp

	// Update history
	history := rm.history[resourceType]
	history = append(history, *resource)
	if len(history) > rm.maxHistorySize {
		history = history[1:]
	}
	rm.history[resourceType] = history

	// Calculate average
	if len(history) > 0 {
		var sum float64
		for _, entry := range history {
			sum += entry.Current
		}
		resource.Average = sum / float64(len(history))
	}

	// Find max
	max := usage
	for _, entry := range history {
		if entry.Current > max {
			max = entry.Current
		}
	}
	resource.Max = max
}

// collectCPUUsage collects CPU usage (simplified)
func (rm *ResourceMonitor) collectCPUUsage() float64 {
	// In a real implementation, this would use system calls
	// For now, return a simulated value
	return 25.0 + (float64(time.Now().Unix()%100) / 10)
}

// collectMemoryUsage collects memory usage
func (rm *ResourceMonitor) collectMemoryUsage() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate memory usage percentage (simplified)
	// In production, would compare against system total memory
	usageMB := float64(memStats.Alloc) / 1024 / 1024
	maxMB := float64(memStats.Sys) / 1024 / 1024

	if maxMB > 0 {
		return (usageMB / maxMB) * 100
	}

	return 0
}

// collectDiskUsage collects disk usage (simplified)
func (rm *ResourceMonitor) collectDiskUsage() float64 {
	// In a real implementation, this would check actual disk usage
	// For now, return a simulated value
	return 45.0 + (float64(time.Now().Unix()%40) / 10)
}
