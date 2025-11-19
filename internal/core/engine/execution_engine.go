// Package engine provides the execution engine with worker pools for concurrent task processing.
package engine

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ExecutionEngine orchestrates task execution using worker pools
type ExecutionEngine struct {
	workerPool *WorkerPool
	scheduler  *TaskScheduler
	mu         sync.RWMutex
	running    bool
	startTime  time.Time
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(workers int, queueSize int, timeout time.Duration) *ExecutionEngine {
	wp := NewWorkerPool(workers, queueSize, timeout)
	scheduler := NewTaskScheduler()

	return &ExecutionEngine{
		workerPool: wp,
		scheduler:  scheduler,
	}
}

// Start starts the execution engine
func (ee *ExecutionEngine) Start(ctx context.Context) error {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if ee.running {
		return ErrAlreadyRunning
	}

	logger.Info("Starting execution engine")

	ee.workerPool.Start()
	ee.scheduler.Start(ctx)

	ee.running = true
	ee.startTime = time.Now()

	logger.Info("Execution engine started",
		zap.Int("workers", ee.workerPool.workers),
	)

	return nil
}

// Stop stops the execution engine gracefully
func (ee *ExecutionEngine) Stop() error {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if !ee.running {
		return nil
	}

	logger.Info("Stopping execution engine")

	if err := ee.scheduler.Stop(); err != nil {
		logger.Error("Error stopping scheduler", zap.Error(err))
	}

	if err := ee.workerPool.Stop(); err != nil {
		logger.Error("Error stopping worker pool", zap.Error(err))
	}

	ee.running = false

	logger.Info("Execution engine stopped",
		zap.Duration("uptime", time.Since(ee.startTime)),
	)

	return nil
}

// Submit submits a task for execution
func (ee *ExecutionEngine) Submit(task Task) error {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	if !ee.running {
		return ErrNotRunning
	}

	return ee.workerPool.Submit(task)
}

// Schedule schedules a task to run at a specific time
func (ee *ExecutionEngine) Schedule(task Task, when time.Time) error {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	if !ee.running {
		return ErrNotRunning
	}

	return ee.scheduler.Schedule(task, when)
}

// ScheduleInterval schedules a task to run at regular intervals
func (ee *ExecutionEngine) ScheduleInterval(task Task, interval time.Duration) error {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	if !ee.running {
		return ErrNotRunning
	}

	return ee.scheduler.ScheduleInterval(task, interval)
}

// Stats returns engine statistics
func (ee *ExecutionEngine) Stats() EngineStats {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	wpStats := ee.workerPool.Stats()

	return EngineStats{
		Running:   ee.running,
		Uptime:    time.Since(ee.startTime),
		PoolStats: wpStats,
	}
}

// EngineStats represents execution engine statistics
type EngineStats struct {
	Running   bool
	Uptime    time.Duration
	PoolStats Stats
}

// Errors
var (
	ErrAlreadyRunning = &Error{Message: "execution engine is already running"}
	ErrNotRunning     = &Error{Message: "execution engine is not running"}
)
