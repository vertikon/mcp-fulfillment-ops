// Package engine provides the execution engine with worker pools for concurrent task processing.
package engine

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Task represents a task to be executed
type Task interface {
	Execute(ctx context.Context) error
	ID() string
}

// WorkerPool manages a pool of workers for concurrent task execution
type WorkerPool struct {
	workers    int
	queue      chan Task
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	active     int
	processed  int64
	failed     int64
	timeout    time.Duration
	retryCount int
	backoff    time.Duration
}

// NewWorkerPool creates a new worker pool
// If workers is 0 or "auto", it uses runtime.NumCPU() * 2
func NewWorkerPool(workers int, queueSize int, timeout time.Duration) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers:    workers,
		queue:      make(chan Task, queueSize),
		ctx:        ctx,
		cancel:     cancel,
		timeout:    timeout,
		retryCount: 3,
		backoff:    time.Second,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	logger.Info("Starting worker pool", zap.Int("workers", wp.workers))

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop stops the worker pool gracefully
func (wp *WorkerPool) Stop() error {
	logger.Info("Stopping worker pool")
	wp.cancel()
	close(wp.queue)
	wp.wg.Wait()
	return nil
}

// Submit submits a task to the pool
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.queue <- task:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return ErrQueueFull
	}
}

// worker is the worker goroutine that processes tasks
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	logger.Debug("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case task, ok := <-wp.queue:
			if !ok {
				logger.Debug("Worker stopping", zap.Int("worker_id", id))
				return
			}

			wp.mu.Lock()
			wp.active++
			wp.mu.Unlock()

			err := wp.executeTask(task)

			wp.mu.Lock()
			wp.active--
			if err != nil {
				wp.failed++
			} else {
				wp.processed++
			}
			wp.mu.Unlock()

		case <-wp.ctx.Done():
			logger.Debug("Worker context cancelled", zap.Int("worker_id", id))
			return
		}
	}
}

// executeTask executes a task with timeout and retry logic
func (wp *WorkerPool) executeTask(task Task) error {
	ctx, cancel := context.WithTimeout(wp.ctx, wp.timeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= wp.retryCount; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * wp.backoff
			logger.Debug("Retrying task",
				zap.String("task_id", task.ID()),
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
		}

		err := task.Execute(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}

	logger.Error("Task failed after retries",
		zap.String("task_id", task.ID()),
		zap.Error(lastErr),
		zap.Int("attempts", wp.retryCount+1),
	)

	return lastErr
}

// Stats returns current pool statistics
func (wp *WorkerPool) Stats() Stats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	return Stats{
		Workers:   wp.workers,
		Active:    wp.active,
		Processed: wp.processed,
		Failed:    wp.failed,
		QueueSize: len(wp.queue),
		QueueCap:  cap(wp.queue),
	}
}

// Stats represents worker pool statistics
type Stats struct {
	Workers   int
	Active    int
	Processed int64
	Failed    int64
	QueueSize int
	QueueCap  int
}

// Errors
var (
	ErrQueueFull = &Error{Message: "queue is full"}
)

// Error represents a worker pool error
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
