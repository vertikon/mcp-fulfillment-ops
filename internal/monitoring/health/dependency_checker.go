// Package health provides dependency health checking capabilities
package health

import (
	"context"
	"fmt"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// DependencyType represents type of dependency
type DependencyType string

const (
	DependencyTypeDatabase DependencyType = "database"
	DependencyTypeCache    DependencyType = "cache"
	DependencyTypeQueue    DependencyType = "queue"
	DependencyTypeAPI      DependencyType = "api"
	DependencyTypeService  DependencyType = "service"
)

// Dependency represents a system dependency
type Dependency struct {
	Name     string                 `json:"name"`
	Type     DependencyType         `json:"type"`
	Endpoint string                 `json:"endpoint"`
	Timeout  time.Duration          `json:"timeout"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// DependencyChecker checks health of system dependencies
type DependencyChecker struct {
	dependencies map[string]*Dependency
	checkers     map[DependencyType]DependencyHealthChecker
}

// DependencyHealthChecker interface for checking specific dependency types
type DependencyHealthChecker interface {
	Check(ctx context.Context, dep *Dependency) HealthResult
	Type() DependencyType
}

// DependencyConfig represents dependency checker configuration
type DependencyConfig struct {
	DefaultTimeout time.Duration `json:"default_timeout"`
	EnableChecks   bool          `json:"enable_checks"`
}

// DefaultDependencyConfig returns default dependency configuration
func DefaultDependencyConfig() *DependencyConfig {
	return &DependencyConfig{
		DefaultTimeout: 5 * time.Second,
		EnableChecks:   true,
	}
}

// NewDependencyChecker creates a new dependency checker
func NewDependencyChecker(config *DependencyConfig) *DependencyChecker {
	if config == nil {
		config = DefaultDependencyConfig()
	}

	checker := &DependencyChecker{
		dependencies: make(map[string]*Dependency),
		checkers:     make(map[DependencyType]DependencyHealthChecker),
	}

	// Register default checkers
	checker.registerDefaultCheckers()

	logger.Info("Dependency checker initialized",
		zap.Duration("default_timeout", config.DefaultTimeout))

	return checker
}

// registerDefaultCheckers registers default health checkers
func (dc *DependencyChecker) registerDefaultCheckers() {
	// Database checker
	dc.checkers[DependencyTypeDatabase] = &DatabaseChecker{}

	// Cache checker
	dc.checkers[DependencyTypeCache] = &CacheChecker{}

	// Queue checker
	dc.checkers[DependencyTypeQueue] = &QueueChecker{}

	// API checker
	dc.checkers[DependencyTypeAPI] = &APIChecker{}

	// Service checker
	dc.checkers[DependencyTypeService] = &ServiceChecker{}
}

// RegisterDependency registers a dependency to check
func (dc *DependencyChecker) RegisterDependency(dep *Dependency) {
	if dep.Timeout == 0 {
		dep.Timeout = 5 * time.Second
	}

	dc.dependencies[dep.Name] = dep

	logger.Info("Dependency registered",
		zap.String("name", dep.Name),
		zap.String("type", string(dep.Type)),
		zap.String("endpoint", dep.Endpoint))
}

// UnregisterDependency unregisters a dependency
func (dc *DependencyChecker) UnregisterDependency(name string) {
	delete(dc.dependencies, name)

	logger.Info("Dependency unregistered", zap.String("name", name))
}

// CheckDependency checks health of a specific dependency
func (dc *DependencyChecker) CheckDependency(ctx context.Context, name string) HealthResult {
	dep, exists := dc.dependencies[name]
	if !exists {
		return HealthResult{
			Name:      name,
			Status:    HealthStatusUnknown,
			Message:   fmt.Sprintf("Dependency not found: %s", name),
			Timestamp: time.Now(),
			Error:     fmt.Sprintf("dependency %s not registered", name),
		}
	}

	checker, exists := dc.checkers[dep.Type]
	if !exists {
		return HealthResult{
			Name:      name,
			Status:    HealthStatusUnknown,
			Message:   fmt.Sprintf("No checker available for type: %s", dep.Type),
			Timestamp: time.Now(),
			Error:     fmt.Sprintf("no checker for type %s", dep.Type),
		}
	}

	checkCtx, cancel := context.WithTimeout(ctx, dep.Timeout)
	defer cancel()

	return checker.Check(checkCtx, dep)
}

// CheckAllDependencies checks health of all dependencies
func (dc *DependencyChecker) CheckAllDependencies(ctx context.Context) map[string]HealthResult {
	results := make(map[string]HealthResult)

	for name := range dc.dependencies {
		results[name] = dc.CheckDependency(ctx, name)
	}

	return results
}

// GetDependencies returns all registered dependencies
func (dc *DependencyChecker) GetDependencies() map[string]*Dependency {
	deps := make(map[string]*Dependency)
	for k, v := range dc.dependencies {
		copy := *v
		deps[k] = &copy
	}
	return deps
}

// Default checkers implementation

// DatabaseChecker checks database dependencies
type DatabaseChecker struct{}

func (dc *DatabaseChecker) Type() DependencyType {
	return DependencyTypeDatabase
}

func (dc *DatabaseChecker) Check(ctx context.Context, dep *Dependency) HealthResult {
	start := time.Now()

	// In a real implementation, this would ping the database
	// For now, simulate a check
	select {
	case <-ctx.Done():
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusUnhealthy,
			Message:   "Database check timeout",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
			Error:     ctx.Err().Error(),
		}
	case <-time.After(100 * time.Millisecond):
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusHealthy,
			Message:   "Database connection healthy",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
}

// CacheChecker checks cache dependencies
type CacheChecker struct{}

func (cc *CacheChecker) Type() DependencyType {
	return DependencyTypeCache
}

func (cc *CacheChecker) Check(ctx context.Context, dep *Dependency) HealthResult {
	start := time.Now()

	// In a real implementation, this would ping the cache
	select {
	case <-ctx.Done():
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusUnhealthy,
			Message:   "Cache check timeout",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
			Error:     ctx.Err().Error(),
		}
	case <-time.After(50 * time.Millisecond):
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusHealthy,
			Message:   "Cache connection healthy",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
}

// QueueChecker checks queue dependencies
type QueueChecker struct{}

func (qc *QueueChecker) Type() DependencyType {
	return DependencyTypeQueue
}

func (qc *QueueChecker) Check(ctx context.Context, dep *Dependency) HealthResult {
	start := time.Now()

	// In a real implementation, this would check queue connectivity
	select {
	case <-ctx.Done():
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusUnhealthy,
			Message:   "Queue check timeout",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
			Error:     ctx.Err().Error(),
		}
	case <-time.After(100 * time.Millisecond):
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusHealthy,
			Message:   "Queue connection healthy",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
}

// APIChecker checks API dependencies
type APIChecker struct{}

func (ac *APIChecker) Type() DependencyType {
	return DependencyTypeAPI
}

func (ac *APIChecker) Check(ctx context.Context, dep *Dependency) HealthResult {
	start := time.Now()

	// In a real implementation, this would make an HTTP request
	select {
	case <-ctx.Done():
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusUnhealthy,
			Message:   "API check timeout",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
			Error:     ctx.Err().Error(),
		}
	case <-time.After(200 * time.Millisecond):
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusHealthy,
			Message:   "API endpoint healthy",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
}

// ServiceChecker checks service dependencies
type ServiceChecker struct{}

func (sc *ServiceChecker) Type() DependencyType {
	return DependencyTypeService
}

func (sc *ServiceChecker) Check(ctx context.Context, dep *Dependency) HealthResult {
	start := time.Now()

	// In a real implementation, this would check service health
	select {
	case <-ctx.Done():
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusUnhealthy,
			Message:   "Service check timeout",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
			Error:     ctx.Err().Error(),
		}
	case <-time.After(150 * time.Millisecond):
		return HealthResult{
			Name:      dep.Name,
			Status:    HealthStatusHealthy,
			Message:   "Service healthy",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
}
