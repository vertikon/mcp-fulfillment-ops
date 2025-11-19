// Package health provides health monitoring capabilities
package health

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// HealthStatus represents health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a health check
type HealthCheck interface {
	Check(ctx context.Context) HealthResult
	Name() string
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// ComponentHealth represents health of a component
type ComponentHealth struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	LastCheck   time.Time              `json:"last_check"`
	LastSuccess *time.Time             `json:"last_success,omitempty"`
	LastFailure *time.Time             `json:"last_failure,omitempty"`
	Checks      []HealthResult         `json:"checks"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// HealthMonitor monitors system health
type HealthMonitor struct {
	mu            sync.RWMutex
	checks        map[string]HealthCheck
	components    map[string]*ComponentHealth
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	overallStatus HealthStatus
}

// HealthStats provides health statistics
type HealthStats struct {
	TotalComponents   int                    `json:"total_components"`
	HealthyComponents int                    `json:"healthy_components"`
	DegradedComponents int                    `json:"degraded_components"`
	UnhealthyComponents int                  `json:"unhealthy_components"`
	OverallStatus     HealthStatus           `json:"overall_status"`
	Components        map[string]ComponentHealth `json:"components"`
	LastUpdate        time.Time              `json:"last_update"`
}

// HealthConfig represents health monitor configuration
type HealthConfig struct {
	CheckInterval time.Duration `json:"check_interval"`
	Timeout       time.Duration `json:"timeout"`
	EnableChecks  bool          `json:"enable_checks"`
}

// DefaultHealthConfig returns default health configuration
func DefaultHealthConfig() *HealthConfig {
	return &HealthConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
		EnableChecks:  true,
	}
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(config *HealthConfig) *HealthMonitor {
	if config == nil {
		config = DefaultHealthConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &HealthMonitor{
		checks:        make(map[string]HealthCheck),
		components:    make(map[string]*ComponentHealth),
		checkInterval: config.CheckInterval,
		ctx:           ctx,
		cancel:        cancel,
		overallStatus: HealthStatusUnknown,
	}

	logger.Info("Health monitor initialized",
		zap.Duration("check_interval", config.CheckInterval),
		zap.Bool("enabled", config.EnableChecks))

	return monitor
}

// RegisterCheck registers a health check
func (hm *HealthMonitor) RegisterCheck(check HealthCheck) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.checks[check.Name()] = check

	logger.Info("Health check registered", zap.String("name", check.Name()))
}

// UnregisterCheck unregisters a health check
func (hm *HealthMonitor) UnregisterCheck(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.checks, name)
	delete(hm.components, name)

	logger.Info("Health check unregistered", zap.String("name", name))
}

// Start begins health monitoring
func (hm *HealthMonitor) Start() {
	logger.Info("Starting health monitor",
		zap.Duration("check_interval", hm.checkInterval))

	go func() {
		ticker := time.NewTicker(hm.checkInterval)
		defer ticker.Stop()

		// Run initial check
		hm.runHealthChecks()

		for {
			select {
			case <-ticker.C:
				hm.runHealthChecks()
			case <-hm.ctx.Done():
				return
			}
		}
	}()
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop() {
	logger.Info("Stopping health monitor")
	hm.cancel()
}

// CheckHealth performs a health check
func (hm *HealthMonitor) CheckHealth(ctx context.Context) HealthStats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	stats := HealthStats{
		Components:    make(map[string]ComponentHealth),
		LastUpdate:    time.Now(),
		OverallStatus: hm.overallStatus,
	}

	for name, component := range hm.components {
		stats.Components[name] = *component
		stats.TotalComponents++

		switch component.Status {
		case HealthStatusHealthy:
			stats.HealthyComponents++
		case HealthStatusDegraded:
			stats.DegradedComponents++
		case HealthStatusUnhealthy:
			stats.UnhealthyComponents++
		}
	}

	return stats
}

// GetComponentHealth returns health of a specific component
func (hm *HealthMonitor) GetComponentHealth(name string) (*ComponentHealth, bool) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	component, exists := hm.components[name]
	if !exists {
		return nil, false
	}

	copy := *component
	return &copy, true
}

// IsHealthy checks if overall system is healthy
func (hm *HealthMonitor) IsHealthy() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return hm.overallStatus == HealthStatusHealthy
}

// GetOverallStatus returns overall health status
func (hm *HealthMonitor) GetOverallStatus() HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return hm.overallStatus
}

// runHealthChecks runs all registered health checks
func (hm *HealthMonitor) runHealthChecks() {
	hm.mu.Lock()
	checks := make(map[string]HealthCheck)
	for k, v := range hm.checks {
		checks[k] = v
	}
	hm.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for name, check := range checks {
		result := hm.executeCheck(ctx, check)
		
		hm.mu.Lock()
		component, exists := hm.components[name]
		if !exists {
			component = &ComponentHealth{
				Name:     name,
				Status:   HealthStatusUnknown,
				Checks:   make([]HealthResult, 0),
				Metadata: make(map[string]interface{}),
			}
			hm.components[name] = component
		}

		component.Status = result.Status
		component.LastCheck = result.Timestamp
		
		if result.Status == HealthStatusHealthy {
			now := time.Now()
			component.LastSuccess = &now
			healthyCount++
		} else {
			now := time.Now()
			component.LastFailure = &now
			if result.Status == HealthStatusDegraded {
				degradedCount++
			} else {
				unhealthyCount++
			}
		}

		// Keep last 10 check results
		component.Checks = append(component.Checks, result)
		if len(component.Checks) > 10 {
			component.Checks = component.Checks[1:]
		}

		hm.mu.Unlock()
	}

	// Update overall status
	hm.mu.Lock()
	if unhealthyCount > 0 {
		hm.overallStatus = HealthStatusUnhealthy
	} else if degradedCount > 0 {
		hm.overallStatus = HealthStatusDegraded
	} else if healthyCount > 0 {
		hm.overallStatus = HealthStatusHealthy
	} else {
		hm.overallStatus = HealthStatusUnknown
	}
	hm.mu.Unlock()

	logger.Debug("Health checks completed",
		zap.Int("healthy", healthyCount),
		zap.Int("degraded", degradedCount),
		zap.Int("unhealthy", unhealthyCount),
		zap.String("overall_status", string(hm.overallStatus)))
}

// executeCheck executes a single health check
func (hm *HealthMonitor) executeCheck(ctx context.Context, check HealthCheck) HealthResult {
	start := time.Now()

	result := check.Check(ctx)

	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	return result
}
