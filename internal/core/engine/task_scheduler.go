// Package engine provides the execution engine with worker pools for concurrent task processing.
package engine

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ScheduledTask represents a scheduled task
type ScheduledTask struct {
	Task      Task
	When      time.Time
	Interval  time.Duration
	Repeat    bool
	ID        string
	CreatedAt time.Time
}

// TaskScheduler manages scheduled task execution
type TaskScheduler struct {
	tasks    map[string]*ScheduledTask
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	ticker   *time.Ticker
	interval time.Duration
}

// NewTaskScheduler creates a new task scheduler
func NewTaskScheduler() *TaskScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &TaskScheduler{
		tasks:    make(map[string]*ScheduledTask),
		interval: time.Second, // Check every second
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the scheduler
func (ts *TaskScheduler) Start(ctx context.Context) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.running {
		return ErrSchedulerAlreadyRunning
	}

	logger.Info("Starting task scheduler")

	ts.ticker = time.NewTicker(ts.interval)
	ts.running = true

	ts.wg.Add(1)
	go ts.run()

	return nil
}

// Stop stops the scheduler
func (ts *TaskScheduler) Stop() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return nil
	}

	logger.Info("Stopping task scheduler")

	ts.cancel()
	if ts.ticker != nil {
		ts.ticker.Stop()
	}

	ts.wg.Wait()
	ts.running = false

	logger.Info("Task scheduler stopped")

	return nil
}

// Schedule schedules a task to run at a specific time
func (ts *TaskScheduler) Schedule(task Task, when time.Time) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	scheduledTask := &ScheduledTask{
		Task:      task,
		When:      when,
		ID:        task.ID(),
		CreatedAt: time.Now(),
		Repeat:    false,
	}

	ts.tasks[scheduledTask.ID] = scheduledTask

	logger.Info("Task scheduled",
		zap.String("task_id", scheduledTask.ID),
		zap.Time("when", when),
	)

	return nil
}

// ScheduleInterval schedules a task to run at regular intervals
func (ts *TaskScheduler) ScheduleInterval(task Task, interval time.Duration) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	scheduledTask := &ScheduledTask{
		Task:      task,
		When:      time.Now().Add(interval),
		Interval:  interval,
		ID:        task.ID(),
		CreatedAt: time.Now(),
		Repeat:    true,
	}

	ts.tasks[scheduledTask.ID] = scheduledTask

	logger.Info("Task scheduled with interval",
		zap.String("task_id", scheduledTask.ID),
		zap.Duration("interval", interval),
	)

	return nil
}

// Cancel cancels a scheduled task
func (ts *TaskScheduler) Cancel(taskID string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if _, exists := ts.tasks[taskID]; !exists {
		return ErrTaskNotFound
	}

	delete(ts.tasks, taskID)

	logger.Info("Task cancelled", zap.String("task_id", taskID))

	return nil
}

// run is the main scheduler loop
func (ts *TaskScheduler) run() {
	defer ts.wg.Done()

	logger.Info("Scheduler loop started")

	for {
		select {
		case <-ts.ctx.Done():
			logger.Info("Scheduler loop stopped")
			return
		case <-ts.ticker.C:
			ts.processScheduledTasks()
		}
	}
}

// processScheduledTasks processes tasks that are due
func (ts *TaskScheduler) processScheduledTasks() {
	ts.mu.RLock()
	now := time.Now()
	var dueTasks []*ScheduledTask

	for _, task := range ts.tasks {
		if now.After(task.When) || now.Equal(task.When) {
			dueTasks = append(dueTasks, task)
		}
	}
	ts.mu.RUnlock()

	for _, task := range dueTasks {
		ts.executeScheduledTask(task)
	}
}

// executeScheduledTask executes a scheduled task
func (ts *TaskScheduler) executeScheduledTask(st *ScheduledTask) {
	logger.Debug("Executing scheduled task", zap.String("task_id", st.ID))

	ctx, cancel := context.WithTimeout(ts.ctx, 30*time.Second)
	defer cancel()

	err := st.Task.Execute(ctx)
	if err != nil {
		logger.Error("Scheduled task failed",
			zap.String("task_id", st.ID),
			zap.Error(err),
		)
	}

	ts.mu.Lock()
	if st.Repeat {
		// Reschedule for next interval
		st.When = time.Now().Add(st.Interval)
	} else {
		// Remove one-time tasks
		delete(ts.tasks, st.ID)
	}
	ts.mu.Unlock()
}

// Errors
var (
	ErrSchedulerAlreadyRunning = &Error{Message: "scheduler is already running"}
	ErrTaskNotFound            = &Error{Message: "task not found"}
)
