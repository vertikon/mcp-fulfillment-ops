package engine

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name        string
		workers     int
		queueSize   int
		timeout     time.Duration
		wantWorkers int
	}{
		{
			name:        "explicit workers",
			workers:     4,
			queueSize:   10,
			timeout:     time.Second,
			wantWorkers: 4,
		},
		{
			name:        "auto workers (0)",
			workers:     0,
			queueSize:   10,
			timeout:     time.Second,
			wantWorkers: runtime.NumCPU() * 2,
		},
		{
			name:        "auto workers (negative)",
			workers:     -1,
			queueSize:   10,
			timeout:     time.Second,
			wantWorkers: runtime.NumCPU() * 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := NewWorkerPool(tt.workers, tt.queueSize, tt.timeout)
			if wp == nil {
				t.Fatal("NewWorkerPool returned nil")
			}
			if wp.workers != tt.wantWorkers {
				t.Errorf("Expected %d workers, got %d", tt.wantWorkers, wp.workers)
			}
			if cap(wp.queue) != tt.queueSize {
				t.Errorf("Expected queue capacity %d, got %d", tt.queueSize, cap(wp.queue))
			}
			if wp.timeout != tt.timeout {
				t.Errorf("Expected timeout %v, got %v", tt.timeout, wp.timeout)
			}
		})
	}
}

func TestWorkerPool_Start(t *testing.T) {
	wp := NewWorkerPool(2, 10, time.Second)
	wp.Start()

	// Wait a bit for workers to start
	time.Sleep(50 * time.Millisecond)

	stats := wp.Stats()
	if stats.Workers != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.Workers)
	}

	_ = wp.Stop()
}

func TestWorkerPool_Stop(t *testing.T) {
	wp := NewWorkerPool(2, 10, time.Second)
	wp.Start()

	// Submit a task
	task := &mockTask{id: "test-task"}
	_ = wp.Submit(task)

	// Stop should wait for workers to finish
	err := wp.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}

	// Verify queue is closed
	select {
	case <-wp.queue:
		// Queue should be closed
	default:
		t.Error("Queue should be closed after Stop()")
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*WorkerPool)
		task        Task
		wantErr     bool
		wantErrType error
	}{
		{
			name: "submit successfully",
			setup: func(wp *WorkerPool) {
				wp.Start()
			},
			task:    &mockTask{id: "test-task"},
			wantErr: false,
		},
		{
			name: "submit when queue full",
			setup: func(wp *WorkerPool) {
				wp.Start()
				// Fill the queue
				for i := 0; i < cap(wp.queue); i++ {
					_ = wp.Submit(&mockTask{id: "fill-task"})
				}
			},
			task:        &mockTask{id: "overflow-task"},
			wantErr:     true,
			wantErrType: ErrQueueFull,
		},
		{
			name: "submit when context cancelled",
			setup: func(wp *WorkerPool) {
				wp.cancel()
			},
			task:    &mockTask{id: "test-task"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := NewWorkerPool(2, 5, time.Second)
			tt.setup(wp)

			err := wp.Submit(tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("Submit() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrType != nil && err != tt.wantErrType {
				t.Errorf("Submit() error = %v, wantErrType %v", err, tt.wantErrType)
			}

			if tt.setup != nil {
				_ = wp.Stop()
			}
		})
	}
}

func TestWorkerPool_ExecuteTask(t *testing.T) {
	wp := NewWorkerPool(2, 10, time.Second)
	wp.Start()
	defer wp.Stop()

	tests := []struct {
		name      string
		task      Task
		wantError bool
	}{
		{
			name: "successful task",
			task: &mockTask{
				id: "success-task",
				execute: func(ctx context.Context) error {
					return nil
				},
			},
			wantError: false,
		},
		{
			name: "failing task",
			task: &mockTask{
				id: "fail-task",
				execute: func(ctx context.Context) error {
					return errors.New("task failed")
				},
			},
			wantError: true,
		},
		{
			name: "task with timeout",
			task: &mockTask{
				id: "timeout-task",
				execute: func(ctx context.Context) error {
					time.Sleep(2 * time.Second)
					return nil
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialFailed := wp.Stats().Failed
			initialProcessed := wp.Stats().Processed

			_ = wp.Submit(tt.task)

			// Wait for task to be processed
			time.Sleep(300 * time.Millisecond)

			stats := wp.Stats()
			if tt.wantError {
				if stats.Failed <= initialFailed {
					t.Error("Expected failed count to increase")
				}
			} else {
				if stats.Processed <= initialProcessed {
					t.Error("Expected processed count to increase")
				}
			}
		})
	}
}

func TestWorkerPool_RetryLogic(t *testing.T) {
	wp := NewWorkerPool(1, 10, 100*time.Millisecond)
	wp.Start()
	defer wp.Stop()

	attempts := 0
	var mu sync.Mutex

	task := &mockTask{
		id: "retry-task",
		execute: func(ctx context.Context) error {
			mu.Lock()
			attempts++
			currentAttempt := attempts
			mu.Unlock()

			if currentAttempt < 3 {
				return errors.New("retryable error")
			}
			return nil
		},
	}

	_ = wp.Submit(task)

	// Wait for retries and success
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	finalAttempts := attempts
	mu.Unlock()

	if finalAttempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", finalAttempts)
	}

	stats := wp.Stats()
	if stats.Processed == 0 {
		t.Error("Task should have been processed successfully after retries")
	}
}

func TestWorkerPool_Stats(t *testing.T) {
	wp := NewWorkerPool(2, 10, time.Second)

	// Stats before start
	stats := wp.Stats()
	if stats.Workers != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.Workers)
	}
	if stats.Active != 0 {
		t.Errorf("Expected 0 active workers, got %d", stats.Active)
	}

	wp.Start()
	defer wp.Stop()

	// Submit a task
	task := &mockTask{
		id: "stats-task",
		execute: func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}
	_ = wp.Submit(task)

	// Check stats while task is processing
	time.Sleep(10 * time.Millisecond)
	stats = wp.Stats()
	if stats.QueueSize == 0 && stats.Active == 0 {
		t.Error("Expected task to be in queue or active")
	}

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)
	stats = wp.Stats()
	if stats.Processed == 0 {
		t.Error("Expected processed count to be > 0")
	}
}

func TestWorkerPool_ConcurrentSubmits(t *testing.T) {
	wp := NewWorkerPool(4, 100, time.Second)
	wp.Start()
	defer wp.Stop()

	var wg sync.WaitGroup
	numTasks := 50

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			task := &mockTask{
				id:      "concurrent-task",
				execute: func(ctx context.Context) error { return nil },
			}
			if err := wp.Submit(task); err != nil {
				t.Errorf("Failed to submit task %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Wait for tasks to be processed
	time.Sleep(500 * time.Millisecond)

	stats := wp.Stats()
	if stats.Processed < int64(numTasks/2) {
		t.Errorf("Expected at least %d tasks processed, got %d", numTasks/2, stats.Processed)
	}
}
