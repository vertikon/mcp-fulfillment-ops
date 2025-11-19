// Package health provides performance profiling capabilities
package health

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ProfileType represents type of profile
type ProfileType string

const (
	ProfileTypeCPU       ProfileType = "cpu"
	ProfileTypeMemory    ProfileType = "memory"
	ProfileTypeGoroutine ProfileType = "goroutine"
	ProfileTypeBlock     ProfileType = "block"
	ProfileTypeMutex     ProfileType = "mutex"
)

// ProfileData represents profiling data
type ProfileData struct {
	Type      ProfileType            `json:"type"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
	Data      []byte                 `json:"data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceProfiler provides performance profiling capabilities
type PerformanceProfiler struct {
	mu          sync.RWMutex
	profiles    map[string]*ProfileData
	profiling   bool
	profileType ProfileType
	ctx         context.Context
	cancel      context.CancelFunc
}

// ProfilerConfig represents profiler configuration
type ProfilerConfig struct {
	EnableProfiling bool          `json:"enable_profiling"`
	ProfileInterval time.Duration `json:"profile_interval"`
	ProfileDuration time.Duration `json:"profile_duration"`
}

// DefaultProfilerConfig returns default profiler configuration
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		EnableProfiling: false,
		ProfileInterval: 5 * time.Minute,
		ProfileDuration: 30 * time.Second,
	}
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler(config *ProfilerConfig) *PerformanceProfiler {
	if config == nil {
		config = DefaultProfilerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	profiler := &PerformanceProfiler{
		profiles: make(map[string]*ProfileData),
		ctx:      ctx,
		cancel:   cancel,
	}

	logger.Info("Performance profiler initialized",
		zap.Bool("enabled", config.EnableProfiling),
		zap.Duration("interval", config.ProfileInterval))

	return profiler
}

// StartCPUProfile starts CPU profiling
func (pp *PerformanceProfiler) StartCPUProfile(duration time.Duration) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.profiling {
		return fmt.Errorf("profiling already in progress")
	}

	pp.profiling = true
	pp.profileType = ProfileTypeCPU

	logger.Info("CPU profiling started", zap.Duration("duration", duration))

	go func() {
		defer func() {
			pp.mu.Lock()
			pp.profiling = false
			pp.mu.Unlock()
		}()

		pprof.StartCPUProfile(nil)
		time.Sleep(duration)
		pprof.StopCPUProfile()

		logger.Info("CPU profiling stopped")
	}()

	return nil
}

// StopCPUProfile stops CPU profiling
func (pp *PerformanceProfiler) StopCPUProfile() {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.profiling && pp.profileType == ProfileTypeCPU {
		pprof.StopCPUProfile()
		pp.profiling = false
		logger.Info("CPU profiling stopped")
	}
}

// ProfileMemory profiles memory usage
func (pp *PerformanceProfiler) ProfileMemory() (*ProfileData, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	profile := &ProfileData{
		Type:      ProfileTypeMemory,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"alloc":       memStats.Alloc,
			"total_alloc": memStats.TotalAlloc,
			"sys":         memStats.Sys,
			"heap_alloc":  memStats.HeapAlloc,
			"heap_sys":    memStats.HeapSys,
			"num_gc":      memStats.NumGC,
		},
	}

	pp.mu.Lock()
	pp.profiles["memory"] = profile
	pp.mu.Unlock()

	return profile, nil
}

// ProfileGoroutines profiles goroutine information
func (pp *PerformanceProfiler) ProfileGoroutines() (*ProfileData, error) {
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return nil, fmt.Errorf("goroutine profile not available")
	}

	var buf bytes.Buffer
	if err := profile.WriteTo(&buf, 0); err != nil {
		return nil, fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	profileData := &ProfileData{
		Type:      ProfileTypeGoroutine,
		Timestamp: time.Now(),
		Data:      buf.Bytes(),
		Metadata: map[string]interface{}{
			"goroutine_count": runtime.NumGoroutine(),
		},
	}

	pp.mu.Lock()
	pp.profiles["goroutine"] = profileData
	pp.mu.Unlock()

	return profileData, nil
}

// GetProfile returns a profile by name
func (pp *PerformanceProfiler) GetProfile(name string) (*ProfileData, bool) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	profile, exists := pp.profiles[name]
	if !exists {
		return nil, false
	}

	copy := *profile
	return &copy, true
}

// GetAllProfiles returns all profiles
func (pp *PerformanceProfiler) GetAllProfiles() map[string]*ProfileData {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	profiles := make(map[string]*ProfileData)
	for k, v := range pp.profiles {
		copy := *v
		profiles[k] = &copy
	}

	return profiles
}

// IsProfiling returns whether profiling is active
func (pp *PerformanceProfiler) IsProfiling() bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	return pp.profiling
}
