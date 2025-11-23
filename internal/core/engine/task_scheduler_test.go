package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewTaskScheduler(t *testing.T) {
	ts := NewTaskScheduler()
	if ts == nil {
		t.Fatal("NewTaskScheduler returned nil")
	}
	if ts.tasks == nil {
		t.Error("tasks map should not be nil")
	}
	if ts.interval != time.Second {
		t.Errorf("Expected interval 1s, got %v", ts.interval)
	}
	if ts.running {
		t.Error("Scheduler should not be running initially")
	}
}

func TestTaskScheduler_Start(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*TaskScheduler)
		wantErr bool
	}{
		{
			name:    "start successfully",
			setup:   func(ts *TaskScheduler) {},
			wantErr: false,
		},
		{
			name: "start when already running",
			setup: func(ts *TaskScheduler) {
				ctx := context.Background()
				_ = ts.Start(ctx)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTaskScheduler()
			tt.setup(ts)

			ctx := context.Background()
			err := ts.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Wait a bit for scheduler to start
				time.Sleep(50 * time.Millisecond)

				// Verify it's running
				ts.mu.RLock()
				running := ts.running
				ts.mu.RUnlock()

				if !running {
					t.Error("Scheduler should be running")
				}

				// Cleanup
				_ = ts.Stop()
			}
		})
	}
}

func TestTaskScheduler_Stop(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*TaskScheduler)
		wantErr bool
	}{
		{
			name: "stop when running",
			setup: func(ts *TaskScheduler) {
				ctx := context.Background()
				_ = ts.Start(ctx)
			},
			wantErr: false,
		},
		{
			name:    "stop when not running",
			setup:   func(ts *TaskScheduler) {},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTaskScheduler()
			tt.setup(ts)

			err := ts.Stop()
			if (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify it's stopped
			ts.mu.RLock()
			running := ts.running
			ts.mu.RUnlock()

			if running {
				t.Error("Scheduler should be stopped")
			}
		})
	}
}

func TestTaskScheduler_Schedule(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	task := &mockTask{id: "test-task"}
	when := time.Now().Add(time.Second)

	if err := ts.Schedule(task, when); err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}

	// Verify task is scheduled
	ts.mu.RLock()
	scheduledTask, exists := ts.tasks[task.ID()]
	ts.mu.RUnlock()

	if !exists {
		t.Error("Task should be scheduled")
	}
	if scheduledTask.ID != task.ID() {
		t.Errorf("Expected task ID %s, got %s", task.ID(), scheduledTask.ID)
	}
	if scheduledTask.When != when {
		t.Errorf("Expected when %v, got %v", when, scheduledTask.When)
	}
}

func TestTaskScheduler_ScheduleInterval(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	task := &mockTask{id: "interval-task"}
	interval := 500 * time.Millisecond

	if err := ts.ScheduleInterval(task, interval); err != nil {
		t.Fatalf("ScheduleInterval() error = %v", err)
	}

	// Verify task is scheduled
	ts.mu.RLock()
	scheduledTask, exists := ts.tasks[task.ID()]
	ts.mu.RUnlock()

	if !exists {
		t.Error("Task should be scheduled")
	}
	if !scheduledTask.Repeat {
		t.Error("Task should be marked as repeat")
	}
	if scheduledTask.Interval != interval {
		t.Errorf("Expected interval %v, got %v", interval, scheduledTask.Interval)
	}
}

func TestTaskScheduler_Cancel(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	task := &mockTask{id: "cancel-task"}
	when := time.Now().Add(time.Second)

	// Schedule task
	_ = ts.Schedule(task, when)

	// Cancel task
	if err := ts.Cancel(task.ID()); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	// Verify task is cancelled
	ts.mu.RLock()
	_, exists := ts.tasks[task.ID()]
	ts.mu.RUnlock()

	if exists {
		t.Error("Task should be cancelled")
	}
}

func TestTaskScheduler_Cancel_NonExistent(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	err := ts.Cancel("non-existent-task")
	if err != ErrTaskNotFound {
		t.Errorf("Cancel() error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskScheduler_ExecuteScheduledTask(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	executed := make(chan bool, 1)
	task := &mockTask{
		id: "execute-task",
		execute: func(ctx context.Context) error {
			executed <- true
			return nil
		},
	}

	// Schedule task for immediate execution
	when := time.Now().Add(100 * time.Millisecond)
	_ = ts.Schedule(task, when)

	// Wait for task to be executed
	select {
	case <-executed:
		// Task executed successfully
	case <-time.After(500 * time.Millisecond):
		t.Error("Task should have been executed")
	}

	// Verify task is removed (one-time task)
	ts.mu.RLock()
	_, exists := ts.tasks[task.ID()]
	ts.mu.RUnlock()

	if exists {
		t.Error("One-time task should be removed after execution")
	}
}

func TestTaskScheduler_ExecuteRepeatingTask(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	executionCount := 0
	var mu sync.Mutex

	task := &mockTask{
		id: "repeat-task",
		execute: func(ctx context.Context) error {
			mu.Lock()
			executionCount++
			mu.Unlock()
			return nil
		},
	}

	// Schedule repeating task
	interval := 100 * time.Millisecond
	_ = ts.ScheduleInterval(task, interval)

	// Wait for multiple executions
	time.Sleep(350 * time.Millisecond)

	mu.Lock()
	count := executionCount
	mu.Unlock()

	if count < 2 {
		t.Errorf("Expected at least 2 executions, got %d", count)
	}

	// Verify task is still scheduled
	ts.mu.RLock()
	_, exists := ts.tasks[task.ID()]
	ts.mu.RUnlock()

	if !exists {
		t.Error("Repeating task should still be scheduled")
	}
}

func TestTaskScheduler_TaskFailure(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	executed := make(chan bool, 1)
	task := &mockTask{
		id: "fail-task",
		execute: func(ctx context.Context) error {
			executed <- true
			return errors.New("task failed")
		},
	}

	// Schedule task for immediate execution
	when := time.Now().Add(100 * time.Millisecond)
	_ = ts.Schedule(task, when)

	// Wait for task to be executed (even if it fails)
	select {
	case <-executed:
		// Task executed (and failed)
	case <-time.After(500 * time.Millisecond):
		t.Error("Task should have been executed")
	}
}

func TestTaskScheduler_MultipleTasks(t *testing.T) {
	ts := NewTaskScheduler()
	ctx := context.Background()

	if err := ts.Start(ctx); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer ts.Stop()

	// Schedule multiple tasks
	for i := 0; i < 5; i++ {
		task := &mockTask{id: "task-" + string(rune(i))}
		when := time.Now().Add(time.Duration(i) * 100 * time.Millisecond)
		_ = ts.Schedule(task, when)
	}

	// Verify all tasks are scheduled
	ts.mu.RLock()
	taskCount := len(ts.tasks)
	ts.mu.RUnlock()

	if taskCount != 5 {
		t.Errorf("Expected 5 tasks, got %d", taskCount)
	}
}
