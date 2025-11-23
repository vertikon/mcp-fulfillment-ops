// Package metrics provides performance monitoring capabilities
package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// PerformanceMonitor monitors system performance metrics
type PerformanceMonitor struct {
	mu              sync.RWMutex
	startTime       time.Time
	lastCheck       time.Time
	metrics         PerformanceMetrics
	cpuUsage        float64
	memUsage        uint64
	goroutineCount  int
	gcStats         runtime.MemStats
	collectInterval time.Duration
	ctx             context.Context
	cancel          context.CancelFunc
}

// PerformanceMetrics holds performance data
type PerformanceMetrics struct {
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    uint64    `json:"memory_usage"`
	MemoryMB       float64   `json:"memory_mb"`
	GoroutineCount int       `json:"goroutine_count"`
	HeapAlloc      uint64    `json:"heap_alloc"`
	HeapSys        uint64    `json:"heap_sys"`
	GCCycles       uint32    `json:"gc_cycles"`
	Uptime         string    `json:"uptime"`
	LastUpdate     time.Time `json:"last_update"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(interval time.Duration) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &PerformanceMonitor{
		startTime:       time.Now(),
		lastCheck:       time.Now(),
		collectInterval: interval,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start begins monitoring
func (pm *PerformanceMonitor) Start() {
	logger.Info("Starting performance monitor", zap.Duration("interval", pm.collectInterval))

	go func() {
		ticker := time.NewTicker(pm.collectInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.collectMetrics()
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops monitoring
func (pm *PerformanceMonitor) Stop() {
	logger.Info("Stopping performance monitor")
	pm.cancel()
}

// collectMetrics gathers current performance metrics
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Memory stats
	runtime.ReadMemStats(&pm.gcStats)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.metrics = PerformanceMetrics{
		GoroutineCount: runtime.NumGoroutine(),
		HeapAlloc:      m.HeapAlloc,
		HeapSys:        m.HeapSys,
		GCCycles:       m.NumGC,
		Uptime:         time.Since(pm.startTime).String(),
		LastUpdate:     time.Now(),
		MemoryUsage:    m.Alloc,
		MemoryMB:       float64(m.Alloc) / 1024 / 1024,
	}

	pm.lastCheck = time.Now()

	logger.Debug("Performance metrics collected",
		zap.Float64("memory_mb", pm.metrics.MemoryMB),
		zap.Int("goroutines", pm.metrics.GoroutineCount),
		zap.Uint64("heap_alloc", pm.metrics.HeapAlloc),
	)
}

// GetMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetMetrics() PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.metrics
}

// GetCPUUsage returns current CPU usage (simplified)
func (pm *PerformanceMonitor) GetCPUUsage() float64 {
	// This is a simplified implementation
	// In production, you'd want to use more sophisticated CPU measurement
	return pm.cpuUsage
}

// IsHealthy checks if system performance is healthy
func (pm *PerformanceMonitor) IsHealthy() bool {
	metrics := pm.GetMetrics()

	// Basic health checks
	if metrics.MemoryMB > 1024 { // More than 1GB memory usage
		return false
	}

	if metrics.GoroutineCount > 1000 { // Too many goroutines
		return false
	}

	return true
}

// GetHealthStatus returns detailed health status
func (pm *PerformanceMonitor) GetHealthStatus() map[string]interface{} {
	metrics := pm.GetMetrics()
	return map[string]interface{}{
		"healthy":         pm.IsHealthy(),
		"memory_usage_mb": metrics.MemoryMB,
		"goroutine_count": metrics.GoroutineCount,
		"uptime":          metrics.Uptime,
		"last_update":     metrics.LastUpdate,
		"heap_alloc_mb":   float64(metrics.HeapAlloc) / 1024 / 1024,
		"gc_cycles":       metrics.GCCycles,
	}
}
