package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockTask is a simple task implementation for testing
type mockTask struct {
	id      string
	execute func(ctx context.Context) error
}

func (m *mockTask) Execute(ctx context.Context) error {
	if m.execute != nil {
		return m.execute(ctx)
	}
	return nil
}

func (m *mockTask) ID() string {
	return m.id
}

func TestNewExecutionEngine(t *testing.T) {
	tests := []struct {
		name      string
		workers   int
		queueSize int
		timeout   time.Duration
		wantErr   bool
	}{
		{
			name:      "valid engine",
			workers:   2,
			queueSize: 10,
			timeout:   time.Second,
			wantErr:   false,
		},
		{
			name:      "auto workers",
			workers:   0,
			queueSize: 10,
			timeout:   time.Second,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ee := NewExecutionEngine(tt.workers, tt.queueSize, tt.timeout)
			if ee == nil {
				t.Fatal("NewExecutionEngine returned nil")
			}
			if ee.workerPool == nil {
				t.Error("workerPool is nil")
			}
			if ee.scheduler == nil {
				t.Error("scheduler is nil")
			}
		})
	}
}

func TestExecutionEngine_Start(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ExecutionEngine)
		wantErr bool
	}{
		{
			name:    "start successfully",
			setup:   func(ee *ExecutionEngine) {},
			wantErr: false,
		},
		{
			name: "start when already running",
			setup: func(ee *ExecutionEngine) {
				ctx := context.Background()
				_ = ee.Start(ctx)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ee := NewExecutionEngine(2, 10, time.Second)
			tt.setup(ee)

			ctx := context.Background()
			err := ee.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify it's running
				stats := ee.Stats()
				if !stats.Running {
					t.Error("Engine should be running")
				}

				// Cleanup
				_ = ee.Stop()
			}
		})
	}
}

func TestExecutionEngine_Stop(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ExecutionEngine)
		wantErr bool
	}{
		{
			name: "stop when running",
			setup: func(ee *ExecutionEngine) {
				ctx := context.Background()
				_ = ee.Start(ctx)
			},
			wantErr: false,
		},
		{
			name:    "stop when not running",
			setup:   func(ee *ExecutionEngine) {},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ee := NewExecutionEngine(2, 10, time.Second)
			tt.setup(ee)

			err := ee.Stop()
			if (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify it's stopped
			stats := ee.Stats()
			if stats.Running {
				t.Error("Engine should be stopped")
			}
		})
	}
}

func TestExecutionEngine_Submit(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ExecutionEngine)
		task    Task
		wantErr bool
	}{
		{
			name: "submit when running",
			setup: func(ee *ExecutionEngine) {
				ctx := context.Background()
				_ = ee.Start(ctx)
			},
			task:    &mockTask{id: "test-task"},
			wantErr: false,
		},
		{
			name:    "submit when not running",
			setup:   func(ee *ExecutionEngine) {},
			task:    &mockTask{id: "test-task"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ee := NewExecutionEngine(2, 10, time.Second)
			tt.setup(ee)

			err := ee.Submit(tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("Submit() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Wait a bit for task to be processed
				time.Sleep(100 * time.Millisecond)
			}

			_ = ee.Stop()
		})
	}
}

func TestExecutionEngine_Schedule(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*ExecutionEngine)
		task    Task
		when    time.Time
		wantErr bool
	}{
		{
			name: "schedule when running",
			setup: func(ee *ExecutionEngine) {
				ctx := context.Background()
				_ = ee.Start(ctx)
			},
			task:    &mockTask{id: "test-task"},
			when:    time.Now().Add(time.Second),
			wantErr: false,
		},
		{
			name:    "schedule when not running",
			setup:   func(ee *ExecutionEngine) {},
			task:    &mockTask{id: "test-task"},
			when:    time.Now().Add(time.Second),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ee := NewExecutionEngine(2, 10, time.Second)
			tt.setup(ee)

			err := ee.Schedule(tt.task, tt.when)
			if (err != nil) != tt.wantErr {
				t.Errorf("Schedule() error = %v, wantErr %v", err, tt.wantErr)
			}

			_ = ee.Stop()
		})
	}
}

func TestExecutionEngine_ScheduleInterval(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ExecutionEngine)
		task     Task
		interval time.Duration
		wantErr  bool
	}{
		{
			name: "schedule interval when running",
			setup: func(ee *ExecutionEngine) {
				ctx := context.Background()
				_ = ee.Start(ctx)
			},
			task:     &mockTask{id: "test-task"},
			interval: time.Second,
			wantErr:  false,
		},
		{
			name:     "schedule interval when not running",
			setup:    func(ee *ExecutionEngine) {},
			task:     &mockTask{id: "test-task"},
			interval: time.Second,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ee := NewExecutionEngine(2, 10, time.Second)
			tt.setup(ee)

			err := ee.ScheduleInterval(tt.task, tt.interval)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScheduleInterval() error = %v, wantErr %v", err, tt.wantErr)
			}

			_ = ee.Stop()
		})
	}
}

func TestExecutionEngine_Stats(t *testing.T) {
	ee := NewExecutionEngine(2, 10, time.Second)

	// Stats when not running
	stats := ee.Stats()
	if stats.Running {
		t.Error("Stats should show engine not running")
	}

	// Start engine
	ctx := context.Background()
	_ = ee.Start(ctx)

	// Stats when running
	stats = ee.Stats()
	if !stats.Running {
		t.Error("Stats should show engine running")
	}
	if stats.Uptime <= 0 {
		t.Error("Uptime should be positive")
	}
	if stats.PoolStats.Workers != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.PoolStats.Workers)
	}

	_ = ee.Stop()
}

func TestExecutionEngine_ConcurrentOperations(t *testing.T) {
	ee := NewExecutionEngine(4, 100, time.Second)
	ctx := context.Background()

	if err := ee.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer ee.Stop()

	// Submit multiple tasks concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			task := &mockTask{
				id: "task-" + string(rune(id)),
				execute: func(ctx context.Context) error {
					time.Sleep(10 * time.Millisecond)
					return nil
				},
			}
			if err := ee.Submit(task); err != nil {
				t.Errorf("Failed to submit task %d: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all tasks to be submitted
	for i := 0; i < 10; i++ {
		<-done
	}

	// Wait for tasks to be processed
	time.Sleep(200 * time.Millisecond)

	stats := ee.Stats()
	if stats.PoolStats.Processed == 0 {
		t.Error("Expected some tasks to be processed")
	}
}

func TestExecutionEngine_ErrorHandling(t *testing.T) {
	ee := NewExecutionEngine(2, 10, time.Second)
	ctx := context.Background()

	if err := ee.Start(ctx); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer ee.Stop()

	// Submit task that fails
	failingTask := &mockTask{
		id: "failing-task",
		execute: func(ctx context.Context) error {
			return errors.New("task failed")
		},
	}

	if err := ee.Submit(failingTask); err != nil {
		t.Errorf("Submit should not fail even if task will fail: %v", err)
	}

	// Wait for task to be processed
	time.Sleep(200 * time.Millisecond)

	stats := ee.Stats()
	if stats.PoolStats.Failed == 0 {
		t.Error("Expected failed count to be incremented")
	}
}
