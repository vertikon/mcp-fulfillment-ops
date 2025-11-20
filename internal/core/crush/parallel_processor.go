// Package crush implements parallel processing optimizations for GLM-4.6
package crush

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ParallelProcessorConfig represents configuration for parallel processing
type ParallelProcessorConfig struct {
	MaxWorkers       int           `json:"max_workers"`
	QueueSize       int           `json:"queue_size"`
	WorkerTimeout   time.Duration `json:"worker_timeout"`
	BatchSize       int           `json:"batch_size"`
	LoadBalancing   bool          `json:"load_balancing"`
	AutoScaling     bool          `json:"auto_scaling"`
	CPUThreshold    float64       `json:"cpu_threshold"`
	MemoryThreshold float64       `json:"memory_threshold"`
}

// TaskType represents different types of parallel tasks
type TaskType string

const (
	TaskTypeCPU      TaskType = "cpu_intensive"
	TaskTypeMemory   TaskType = "memory_intensive"
	TaskTypeIO       TaskType = "io_intensive"
	TaskTypeNetwork  TaskType = "network_intensive"
	TaskTypeML       TaskType = "ml_inference"
)

// ParallelTask represents a task to be processed in parallel
type ParallelTask struct {
	ID          string        `json:"id"`
	Type        TaskType      `json:"type"`
	Priority    int           `json:"priority"`
	Data        interface{}   `json:"data"`
	Context     context.Context `json:"-"`
	Handler     TaskHandler   `json:"-"`
	Timeout     time.Duration `json:"timeout"`
	Retries     int           `json:"retries"`
	MaxRetries  int           `json:"max_retries"`
	CreatedAt   time.Time     `json:"created_at"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Error       error         `json:"error,omitempty"`
}

// TaskHandler represents a task processing function
type TaskHandler func(ctx context.Context, task *ParallelTask) (interface{}, error)

// WorkerPool represents a pool of parallel workers
type WorkerPool struct {
	config       ParallelProcessorConfig
	workers      []*Worker
	taskQueue    chan *ParallelTask
	resultQueue  chan *TaskResult
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	stats        *ParallelProcessorStats
	loadBalancer *LoadBalancer
	autoScaler   *AutoScaler
}

// Worker represents a parallel processing worker
type Worker struct {
	ID          int
	WorkerType  TaskType
	CurrentTask *ParallelTask
	Stats       *WorkerStats
	TaskCount   int64
	Busy        bool
	mu          sync.RWMutex
}

// WorkerStats tracks worker performance
type WorkerStats struct {
	TasksProcessed   int64         `json:"tasks_processed"`
	TaskErrors       int64         `json:"task_errors"`
	AvgTaskTime      time.Duration `json:"avg_task_time"`
	LastTaskTime     time.Duration `json:"last_task_time"`
	Utilization      float64       `json:"utilization"`
	LastUpdated      time.Time      `json:"last_updated"`
}

// TaskResult represents the result of a parallel task
type TaskResult struct {
	TaskID    string      `json:"task_id"`
	Result    interface{} `json:"result"`
	Error     error       `json:"error"`
	Duration  time.Duration `json:"duration"`
	WorkerID  int         `json:"worker_id"`
	CompletedAt time.Time  `json:"completed_at"`
}

// LoadBalancer handles task distribution across workers
type LoadBalancer struct {
	strategy LoadBalancingStrategy
	workers  []*Worker
	stats     *LoadBalancerStats
}

// LoadBalancingStrategy represents different load balancing strategies
type LoadBalancingStrategy string

const (
	StrategyRoundRobin LoadBalancingStrategy = "round_robin"
	StrategyLeastLoad LoadBalancingStrategy = "least_load"
	StrategyWeighted  LoadBalancingStrategy = "weighted"
	StrategyRandom    LoadBalancingStrategy = "random"
)

// LoadBalancerStats tracks load balancer performance
type LoadBalancerStats struct {
	TasksAssigned    int64   `json:"tasks_assigned"`
	LoadBalanceScore float64  `json:"load_balance_score"`
	WorkerLoads      []float64 `json:"worker_loads"`
	LastUpdated      int64    `json:"last_updated"`
}

// AutoScaler handles automatic worker scaling
type AutoScaler struct {
	config       AutoScalingConfig
	currentWorkers int
	minWorkers    int
	maxWorkers    int
	scalingMetric ScalingMetric
	stats         *AutoScalerStats
}

// AutoScalingConfig represents auto-scaling configuration
type AutoScalingConfig struct {
	Enabled           bool          `json:"enabled"`
	ScaleUpThreshold  float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64      `json:"scale_down_threshold"`
	ScaleUpCooldown   time.Duration `json:"scale_up_cooldown"`
	ScaleDownCooldown time.Duration `json:"scale_down_cooldown"`
	MaxScaleUpStep   int           `json:"max_scale_up_step"`
	MaxScaleDownStep int           `json:"max_scale_down_step"`
}

// ScalingMetric represents metrics used for auto-scaling
type ScalingMetric string

const (
	MetricCPU     ScalingMetric = "cpu"
	MetricMemory  ScalingMetric = "memory"
	MetricQueue   ScalingMetric = "queue_length"
	MetricLatency ScalingMetric = "latency"
)

// AutoScalerStats tracks auto-scaling performance
type AutoScalerStats struct {
	ScaleUpEvents     int64     `json:"scale_up_events"`
	ScaleDownEvents   int64     `json:"scale_down_events"`
	CurrentWorkers    int        `json:"current_workers"`
	MaxWorkersReached int        `json:"max_workers_reached"`
	MinWorkersReached int        `json:"min_workers_reached"`
	LastScalingTime   *time.Time `json:"last_scaling_time,omitempty"`
}

// ParallelProcessorStats tracks parallel processor performance
type ParallelProcessorStats struct {
	TotalTasksSubmitted int64         `json:"total_tasks_submitted"`
	TotalTasksCompleted int64         `json:"total_tasks_completed"`
	TotalTasksFailed    int64         `json:"total_tasks_failed"`
	AvgTaskTime         time.Duration `json:"avg_task_time"`
	Throughput          float64       `json:"throughput"`
	QueueUtilization    float64       `json:"queue_utilization"`
	WorkerUtilization   float64       `json:"worker_utilization"`
	LastUpdated         time.Time      `json:"last_updated"`
}

// NewParallelProcessor creates a new parallel processor
func NewParallelProcessor(config ParallelProcessorConfig) *WorkerPool {
	if config.MaxWorkers == 0 {
		config.MaxWorkers = runtime.NumCPU() * 2
	}
	if config.QueueSize == 0 {
		config.QueueSize = 1000
	}

	logger.Info("Creating parallel processor",
		zap.Int("max_workers", config.MaxWorkers),
		zap.Int("queue_size", config.QueueSize),
		zap.Bool("auto_scaling", config.AutoScaling),
		zap.Bool("load_balancing", config.LoadBalancing),
	)

	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &WorkerPool{
		config:      config,
		workers:     make([]*Worker, config.MaxWorkers),
		taskQueue:   make(chan *ParallelTask, config.QueueSize),
		resultQueue: make(chan *TaskResult, config.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		stats:       &ParallelProcessorStats{},
	}

	// Initialize workers
	pool.initializeWorkers()

	// Initialize load balancer
	if config.LoadBalancing {
		pool.loadBalancer = NewLoadBalancer(StrategyLeastLoad, pool.workers)
	}

	// Initialize auto scaler
	if config.AutoScaling {
		pool.autoScaler = NewAutoScaler(config.MaxWorkers, config)
	}

	return pool
}

// Start starts the parallel processor
func (pp *WorkerPool) Start() error {
	logger.Info("Starting parallel processor")

	// Start workers
	for i, worker := range pp.workers {
		pp.wg.Add(1)
		go pp.startWorker(i, worker)
	}

	// Start result collector
	go pp.startResultCollector()

	// Start auto scaler if enabled
	if pp.autoScaler != nil {
		go pp.autoScaler.Start(pp.ctx, pp)
	}

	// Start statistics collector
	go pp.startStatsCollector()

	return nil
}

// Stop stops the parallel processor gracefully
func (pp *WorkerPool) Stop() error {
	logger.Info("Stopping parallel processor")

	pp.cancel()
	pp.wg.Wait()

	close(pp.taskQueue)
	close(pp.resultQueue)

	return nil
}

// Submit submits a task for parallel processing
func (pp *WorkerPool) Submit(ctx context.Context, task *ParallelTask) error {
	select {
	case pp.taskQueue <- task:
		atomic.AddInt64(&pp.stats.TotalTasksSubmitted, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-pp.ctx.Done():
		return fmt.Errorf("parallel processor is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitBatch submits a batch of tasks
func (pp *WorkerPool) SubmitBatch(ctx context.Context, tasks []*ParallelTask) error {
	for _, task := range tasks {
		if err := pp.Submit(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// initializeWorkers initializes the worker pool
func (pp *WorkerPool) initializeWorkers() {
	taskTypes := []TaskType{TaskTypeCPU, TaskTypeMemory, TaskTypeIO, TaskTypeNetwork, TaskTypeML}
	
	for i := 0; i < pp.config.MaxWorkers; i++ {
		workerType := taskTypes[i%len(taskTypes)]
		
		pp.workers[i] = &Worker{
			ID:         i,
			WorkerType: workerType,
			Stats:      &WorkerStats{},
		}
	}
}

// startWorker starts a worker goroutine
func (pp *WorkerPool) startWorker(workerID int, worker *Worker) {
	defer pp.wg.Done()
	
	logger.Debug("Worker started", zap.Int("worker_id", workerID))
	
	for {
		select {
		case task := <-pp.taskQueue:
			if task == nil {
				return
			}
			
			pp.processTask(worker, task)
			
		case <-pp.ctx.Done():
			return
		}
	}
}

// processTask processes a single task
func (pp *WorkerPool) processTask(worker *Worker, task *ParallelTask) {
	worker.mu.Lock()
	worker.CurrentTask = task
	worker.Busy = true
	now := time.Now()
	worker.LastTaskTime = time.Since(now)
	worker.mu.Unlock()
	
	startTime := time.Now()
	
	// Update task start time
	task.StartedAt = &startTime
	
	// Process task
	result, err := task.Handler(task.Context, task)
	
	completionTime := time.Now()
	duration := completionTime.Sub(startTime)
	
	// Update task completion
	task.CompletedAt = &completionTime
	task.Error = err
	
	// Update worker stats
	pp.updateWorkerStats(worker, duration, err)
	
	// Send result
	taskResult := &TaskResult{
		TaskID:      task.ID,
		Result:      result,
		Error:       err,
		Duration:    duration,
		WorkerID:    worker.ID,
		CompletedAt: completionTime,
	}
	
	select {
	case pp.resultQueue <- taskResult:
	default:
		logger.Warn("Result queue full, dropping result", zap.String("task_id", task.ID))
	}
	
	worker.mu.Lock()
	worker.CurrentTask = nil
	worker.Busy = false
	worker.mu.Unlock()
}

// updateWorkerStats updates worker statistics
func (pp *WorkerPool) updateWorkerStats(worker *Worker, duration time.Duration, err error) {
	worker.mu.Lock()
	defer worker.mu.Unlock()
	
	atomic.AddInt64(&worker.TaskCount, 1)
	atomic.AddInt64(&worker.Stats.TasksProcessed, 1)
	
	if err != nil {
		atomic.AddInt64(&worker.Stats.TaskErrors, 1)
	}
	
	// Update average task time
	totalTasks := worker.Stats.TasksProcessed
	worker.Stats.AvgTaskTime = time.Duration(
		(int64(worker.Stats.AvgTaskTime)*(totalTasks-1) + int64(duration)) / totalTasks,
	)
	
	worker.Stats.LastTaskTime = duration
	worker.Stats.LastUpdated = time.Now()
}

// startResultCollector collects task results
func (pp *WorkerPool) startResultCollector() {
	for {
		select {
		case result := <-pp.resultQueue:
			if result == nil {
				return
			}
			
			pp.processResult(result)
			
		case <-pp.ctx.Done():
			return
		}
	}
}

// processResult processes a task result
func (pp *WorkerPool) processResult(result *TaskResult) {
	if result.Error != nil {
		atomic.AddInt64(&pp.stats.TotalTasksFailed, 1)
	} else {
		atomic.AddInt64(&pp.stats.TotalTasksCompleted, 1)
	}
	
	// Update overall statistics
	pp.updateStats(result.Duration)
}

// startStatsCollector collects periodic statistics
func (pp *WorkerPool) startStatsCollector() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pp.collectStats()
		case <-pp.ctx.Done():
			return
		}
	}
}

// collectStats collects current statistics
func (pp *WorkerPool) collectStats() {
	pp.stats.LastUpdated = time.Now()
	pp.stats.QueueUtilization = float64(len(pp.taskQueue)) / float64(cap(pp.taskQueue))
	
	// Calculate worker utilization
	busyWorkers := 0
	totalUtilization := 0.0
	
	for _, worker := range pp.workers {
		worker.mu.RLock()
		if worker.Busy {
			busyWorkers++
		}
		totalUtilization += worker.Stats.Utilization
		worker.mu.RUnlock()
	}
	
	if len(pp.workers) > 0 {
		pp.stats.WorkerUtilization = totalUtilization / float64(len(pp.workers))
	}
	
	// Calculate throughput
	completed := atomic.LoadInt64(&pp.stats.TotalTasksCompleted)
	submitted := atomic.LoadInt64(&pp.stats.TotalTasksSubmitted)
	
	if completed > 0 {
		pp.stats.Throughput = float64(completed) / time.Since(time.Now()).Seconds()
	}
	
	logger.Debug("Parallel processor stats",
		zap.Float64("queue_utilization", pp.stats.QueueUtilization),
		zap.Float64("worker_utilization", pp.stats.WorkerUtilization),
		zap.Float64("throughput", pp.stats.Throughput),
		zap.Int("busy_workers", busyWorkers),
	)
}

// updateStats updates overall statistics
func (pp *WorkerPool) updateStats(duration time.Duration) {
	// Update average task time
	completed := atomic.LoadInt64(&pp.stats.TotalTasksCompleted)
	pp.stats.AvgTaskTime = time.Duration(
		(int64(pp.stats.AvgTaskTime)*(completed-1) + int64(duration)) / completed,
	)
}

// GetStats returns current processor statistics
func (pp *WorkerPool) GetStats() ParallelProcessorStats {
	pp.collectStats()
	return *pp.stats
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(strategy LoadBalancingStrategy, workers []*Worker) *LoadBalancer {
	return &LoadBalancer{
		strategy: strategy,
		workers:  workers,
		stats:    &LoadBalancerStats{},
	}
}

// SelectWorker selects the best worker for a task
func (lb *LoadBalancer) SelectWorker(task *ParallelTask) *Worker {
	switch lb.strategy {
	case StrategyRoundRobin:
		return lb.selectRoundRobin()
	case StrategyLeastLoad:
		return lb.selectLeastLoad()
	case StrategyRandom:
		return lb.selectRandom()
	default:
		return lb.selectLeastLoad()
	}
}

// selectRoundRobin selects worker using round-robin strategy
func (lb *LoadBalancer) selectRoundRobin() *Worker {
	lb.stats.TasksAssigned++
	
	// Simplified round-robin
	workerIndex := int(lb.stats.TasksAssigned) % len(lb.workers)
	return lb.workers[workerIndex]
}

// selectLeastLoad selects worker with least load
func (lb *LoadBalancer) selectLeastLoad() *Worker {
	var selectedWorker *Worker
	minTaskCount := int64(^uint64(0) >> 1) // Max int64
	
	for _, worker := range lb.workers {
		taskCount := atomic.LoadInt64(&worker.TaskCount)
		if taskCount < minTaskCount {
			minTaskCount = taskCount
			selectedWorker = worker
		}
	}
	
	lb.stats.TasksAssigned++
	return selectedWorker
}

// selectRandom selects random worker
func (lb *LoadBalancer) selectRandom() *Worker {
	workerIndex := time.Now().Nanosecond() % len(lb.workers)
	return lb.workers[workerIndex]
}

// NewAutoScaler creates a new auto scaler
func NewAutoScaler(maxWorkers int, config *WorkerPool) *AutoScaler {
	return &AutoScaler{
		currentWorkers: maxWorkers,
		minWorkers:     runtime.NumCPU(),
		maxWorkers:     maxWorkers * 4, // Allow 4x scaling
		stats:          &AutoScalerStats{},
	}
}

// Start starts the auto scaler
func (as *AutoScaler) Start(ctx context.Context, pool *WorkerPool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			as.evaluateScaling(ctx, pool)
		case <-ctx.Done():
			return
		}
	}
}

// evaluateScaling evaluates if scaling is needed
func (as *AutoScaler) evaluateScaling(ctx context.Context, pool *WorkerPool) {
	// Get current statistics
	stats := pool.GetStats()
	
	// Simplified scaling logic
	if stats.QueueUtilization > 0.8 {
		as.scaleUp(ctx, pool)
	} else if stats.QueueUtilization < 0.2 && as.currentWorkers > as.minWorkers {
		as.scaleDown(ctx, pool)
	}
}

// scaleUp scales up the worker pool
func (as *AutoScaler) scaleUp(ctx context.Context, pool *WorkerPool) {
	if as.currentWorkers >= as.maxWorkers {
		return
	}
	
	// Add new workers
	newWorkers := min(as.currentWorkers+2, as.maxWorkers)
	
	logger.Info("Scaling up workers",
		zap.Int("from", as.currentWorkers),
		zap.Int("to", newWorkers),
	)
	
	as.currentWorkers = newWorkers
	as.stats.ScaleUpEvents++
	now := time.Now()
	as.stats.LastScalingTime = &now
}

// scaleDown scales down the worker pool
func (as *AutoScaler) scaleDown(ctx context.Context, pool *WorkerPool) {
	if as.currentWorkers <= as.minWorkers {
		return
	}
	
	// Remove workers
	newWorkers := max(as.currentWorkers-2, as.minWorkers)
	
	logger.Info("Scaling down workers",
		zap.Int("from", as.currentWorkers),
		zap.Int("to", newWorkers),
	)
	
	as.currentWorkers = newWorkers
	as.stats.ScaleDownEvents++
	now := time.Now()
	as.stats.LastScalingTime = &now
}

// Helper functions min/max are defined in utils.go