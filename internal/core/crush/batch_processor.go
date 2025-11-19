// Package crush implements batch processing optimizations for GLM-4.6
package crush

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// BatchProcessorConfig represents configuration for batch processing
type BatchProcessorConfig struct {
	MaxBatchSize      int           `json:"max_batch_size"`
	MinBatchSize      int           `json:"min_batch_size"`
	Timeout           time.Duration `json:"timeout"`
	MaxLatency        time.Duration `json:"max_latency"`
	EnableDynamicBatching bool     `json:"enable_dynamic_batching"`
	EnablePrefetch    bool          `json:"enable_prefetch"`
	EnableAsync       bool          `json:"enable_async"`
	CompressionType   string        `json:"compression_type"`
}

// BatchType represents different types of batch processing
type BatchType string

const (
	BatchTypeInference  BatchType = "inference"
	BatchTypeTraining   BatchType = "training"
	BatchTypeEmbedding  BatchType = "embedding"
	BatchTypeToken      BatchType = "tokenization"
	BatchTypeEvaluation BatchType = "evaluation"
)

// Batch represents a collection of items to be processed together
type Batch struct {
	ID          string      `json:"id"`
	Type        BatchType   `json:"type"`
	Items       []interface{} `json:"items"`
	Size        int         `json:"size"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	Timeout     time.Duration `json:"timeout"`
	Priority    int         `json:"priority"`
	Context     context.Context `json:"-"`
	Handler     BatchHandler `json:"-"`
	Results     []interface{} `json:"results,omitempty"`
	Errors      []error     `json:"errors,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BatchHandler represents a batch processing function
type BatchHandler func(ctx context.Context, batch *Batch) ([]interface{}, []error)

// BatchProcessor manages batch processing optimization
type BatchProcessor struct {
	config          BatchProcessorConfig
	batches         map[string]*Batch
	batchQueue      chan *Batch
	resultQueue     chan *BatchResult
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	stats           *BatchProcessorStats
	dynamicSizing   *DynamicBatchSizer
	prefetcher      *BatchPrefetcher
	asyncProcessor  *AsyncBatchProcessor
	mu              sync.RWMutex
}

// BatchResult represents the result of batch processing
type BatchResult struct {
	BatchID     string        `json:"batch_id"`
	Results     []interface{} `json:"results"`
	Errors      []error       `json:"errors"`
	Duration    time.Duration `json:"duration"`
	ProcessedAt time.Time     `json:"processed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DynamicBatchSizer adjusts batch sizes based on performance
type DynamicBatchSizer struct {
	config          DynamicSizingConfig
	currentBatchSize int
	performance     *PerformanceMetrics
	strategy        SizingStrategy
	stats           *DynamicSizingStats
}

// DynamicSizingConfig represents dynamic sizing configuration
type DynamicSizingConfig struct {
	Enabled          bool          `json:"enabled"`
	Strategy         SizingStrategy `json:"strategy"`
	MinSize          int           `json:"min_size"`
	MaxSize          int           `json:"max_size"`
	AdjustmentInterval time.Duration `json:"adjustment_interval"`
	TargetLatency    time.Duration `json:"target_latency"`
	TargetThroughput float64       `json:"target_throughput"`
}

// SizingStrategy represents different batch sizing strategies
type SizingStrategy string

const (
	StrategyLatencyBased  SizingStrategy = "latency_based"
	StrategyThroughputBased SizingStrategy = "throughput_based"
	StrategyMemoryBased   SizingStrategy = "memory_based"
	StrategyAdaptive      SizingStrategy = "adaptive"
)

// PerformanceMetrics tracks batch processing performance
type PerformanceMetrics struct {
	AvgLatency       time.Duration `json:"avg_latency"`
	AvgThroughput    float64       `json:"avg_throughput"`
	MemoryUsage      float64       `json:"memory_usage"`
	ErrorRate        float64       `json:"error_rate"`
	LastUpdated      time.Time      `json:"last_updated"`
}

// DynamicSizingStats tracks dynamic sizing statistics
type DynamicSizingStats struct {
	SizeAdjustments    int64 `json:"size_adjustments"`
	CurrentSize       int    `json:"current_size"`
	MinSizeReached    int    `json:"min_size_reached"`
	MaxSizeReached    int    `json:"max_size_reached"`
	LastAdjustment    int64  `json:"last_adjustment"`
}

// BatchPrefetcher prefetches batches for improved performance
type BatchPrefetcher struct {
	config       PrefetchConfig
	prefetchQueue chan *Batch
	cache        map[string]*Batch
	stats        *PrefetchStats
	mu           sync.RWMutex
}

// PrefetchConfig represents prefetch configuration
type PrefetchConfig struct {
	Enabled         bool          `json:"enabled"`
	PrefetchSize    int           `json:"prefetch_size"`
	PrefetchTimeout time.Duration `json:"prefetch_timeout"`
	CacheSize       int           `json:"cache_size"`
	PrefetchPolicy  PrefetchPolicy `json:"prefetch_policy"`
}

// PrefetchPolicy represents different prefetch policies
type PrefetchPolicy string

const (
	PolicyLRU          PrefetchPolicy = "lru"
	PolicyPredictive   PrefetchPolicy = "predictive"
	PolicyProactive    PrefetchPolicy = "proactive"
	PolicyLazy        PrefetchPolicy = "lazy"
)

// PrefetchStats tracks prefetch performance
type PrefetchStats struct {
	TotalPrefetches   int64 `json:"total_prefetches"`
	CacheHits         int64 `json:"cache_hits"`
	CacheMisses       int64 `json:"cache_misses"`
	HitRate           float64 `json:"hit_rate"`
	LastPrefetch      int64 `json:"last_prefetch"`
}

// AsyncBatchProcessor processes batches asynchronously
type AsyncBatchProcessor struct {
	config       AsyncProcessingConfig
	workerPool   *WorkerPool
	processor    BatchHandler
	stats        *AsyncProcessingStats
}

// AsyncProcessingConfig represents async processing configuration
type AsyncProcessingConfig struct {
	Enabled        bool          `json:"enabled"`
	NumWorkers     int           `json:"num_workers"`
	QueueSize      int           `json:"queue_size"`
	WorkerTimeout  time.Duration `json:"worker_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	BackoffFactor  float64       `json:"backoff_factor"`
}

// AsyncProcessingStats tracks async processing statistics
type AsyncProcessingStats struct {
	TotalBatches     int64   `json:"total_batches"`
	AsyncBatches     int64   `json:"async_batches"`
	BatchErrors      int64   `json:"batch_errors"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	WorkerUtilization float64 `json:"worker_utilization"`
}

// BatchProcessorStats tracks batch processor performance
type BatchProcessorStats struct {
	TotalBatches      int64         `json:"total_batches"`
	TotalItems        int64         `json:"total_items"`
	CompletedBatches  int64         `json:"completed_batches"`
	FailedBatches     int64         `json:"failed_batches"`
	AvgBatchSize      float64       `json:"avg_batch_size"`
	AvgProcessingTime time.Duration `json:"avg_processing_time"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	Throughput       float64       `json:"throughput"`
	QueueUtilization  float64       `json:"queue_utilization"`
	LastUpdated      time.Time      `json:"last_updated"`
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(config BatchProcessorConfig) *BatchProcessor {
	if config.MaxBatchSize == 0 {
		config.MaxBatchSize = 32
	}
	if config.MinBatchSize == 0 {
		config.MinBatchSize = 1
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	logger.Info("Creating batch processor",
		zap.Int("max_batch_size", config.MaxBatchSize),
		zap.Int("min_batch_size", config.MinBatchSize),
		zap.Duration("timeout", config.Timeout),
		zap.Bool("dynamic_batching", config.EnableDynamicBatching),
		zap.Bool("prefetch", config.EnablePrefetch),
		zap.Bool("async", config.EnableAsync),
	)

	ctx, cancel := context.WithCancel(context.Background())
	
	processor := &BatchProcessor{
		config:      config,
		batches:     make(map[string]*Batch),
		batchQueue:  make(chan *Batch, 100),
		resultQueue: make(chan *BatchResult, 100),
		ctx:         ctx,
		cancel:      cancel,
		stats:       &BatchProcessorStats{},
	}

	// Initialize dynamic sizing
	if config.EnableDynamicBatching {
		processor.dynamicSizing = NewDynamicBatchSizer(DynamicSizingConfig{
			Enabled:   true,
			Strategy:  StrategyAdaptive,
			MinSize:   config.MinBatchSize,
			MaxSize:   config.MaxBatchSize,
			TargetLatency: config.MaxLatency,
		})
	}

	// Initialize prefetcher
	if config.EnablePrefetch {
		processor.prefetcher = NewBatchPrefetcher(PrefetchConfig{
			Enabled:      true,
			PrefetchSize: config.MaxBatchSize,
			CacheSize:    100,
			PrefetchPolicy: PolicyPredictive,
		})
	}

	// Initialize async processor
	if config.EnableAsync {
		processor.asyncProcessor = NewAsyncBatchProcessor(AsyncProcessingConfig{
			Enabled:       true,
			NumWorkers:    runtime.NumCPU(),
			QueueSize:     100,
			WorkerTimeout: config.Timeout,
			RetryAttempts: 3,
		})
	}

	return processor
}

// Start starts the batch processor
func (bp *BatchProcessor) Start() error {
	logger.Info("Starting batch processor")

	// Start batch collection
	go bp.startBatchCollector()

	// Start batch processor
	go bp.startBatchProcessor()

	// Start result collector
	go bp.startResultCollector()

	// Start statistics collector
	go bp.startStatsCollector()

	// Start dynamic sizing if enabled
	if bp.dynamicSizing != nil {
		go bp.dynamicSizing.Start(bp.ctx)
	}

	// Start prefetcher if enabled
	if bp.prefetcher != nil {
		go bp.prefetcher.Start(bp.ctx)
	}

	// Start async processor if enabled
	if bp.asyncProcessor != nil {
		go bp.asyncProcessor.Start(bp.ctx)
	}

	return nil
}

// Stop stops the batch processor gracefully
func (bp *BatchProcessor) Stop() error {
	logger.Info("Stopping batch processor")

	bp.cancel()
	bp.wg.Wait()

	close(bp.batchQueue)
	close(bp.resultQueue)

	return nil
}

// Submit submits an item for batch processing
func (bp *BatchProcessor) Submit(ctx context.Context, itemType BatchType, item interface{}, handler BatchHandler) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Find existing batch or create new one
	batch := bp.findOrCreateBatch(itemType, handler)
	
	batch.Items = append(batch.Items, item)
	batch.Size = len(batch.Items)

	// Check if batch is ready for processing
	batchSize := bp.getOptimalBatchSize(itemType)
	if batch.Size >= batchSize {
		return bp.submitBatch(batch)
	}

	// Check if batch should be submitted due to timeout
	if time.Since(batch.CreatedAt) >= batch.Timeout {
		return bp.submitBatch(batch)
	}

	return nil
}

// SubmitBatch submits a complete batch for processing
func (bp *BatchProcessor) SubmitBatch(ctx context.Context, batch *Batch) error {
	batch.Context = ctx
	
	select {
	case bp.batchQueue <- batch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-bp.ctx.Done():
		return fmt.Errorf("batch processor is shutting down")
	}
}

// findOrCreateBatch finds an existing batch or creates a new one
func (bp *BatchProcessor) findOrCreateBatch(itemType BatchType, handler BatchHandler) *Batch {
	// For simplicity, create a new batch each time
	// In practice, this would group items by type and handler
	
	batchID := fmt.Sprintf("batch_%d_%s", time.Now().UnixNano(), itemType)
	batch := &Batch{
		ID:       batchID,
		Type:     itemType,
		Items:    make([]interface{}, 0),
		Size:     0,
		CreatedAt: time.Now(),
		Timeout:  bp.config.Timeout,
		Priority: 0,
		Handler:  handler,
		Metadata: make(map[string]interface{}),
	}

	bp.batches[batchID] = batch
	return batch
}

// submitBatch submits a batch to the processing queue
func (bp *BatchProcessor) submitBatch(batch *Batch) error {
	delete(bp.batches, batch.ID)
	
	return bp.SubmitBatch(batch.Context, batch)
}

// getOptimalBatchSize returns optimal batch size for item type
func (bp *BatchProcessor) getOptimalBatchSize(itemType BatchType) int {
	if bp.dynamicSizing != nil {
		return bp.dynamicSizing.GetCurrentBatchSize()
	}
	
	return bp.config.MaxBatchSize
}

// startBatchCollector collects items into batches
func (bp *BatchProcessor) startBatchCollector() {
	bp.wg.Add(1)
	defer bp.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bp.checkBatchTimeouts()
		case <-bp.ctx.Done():
			return
		}
	}
}

// checkBatchTimeouts checks and submits timed-out batches
func (bp *BatchProcessor) checkBatchTimeouts() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	now := time.Now()
	for id, batch := range bp.batches {
		if now.Sub(batch.CreatedAt) >= batch.Timeout && len(batch.Items) > 0 {
			bp.submitBatch(batch)
		}
	}
}

// startBatchProcessor processes batches
func (bp *BatchProcessor) startBatchProcessor() {
	bp.wg.Add(1)
	defer bp.wg.Done()

	for {
		select {
		case batch := <-bp.batchQueue:
			if batch == nil {
				return
			}
			bp.processBatch(batch)
		case <-bp.ctx.Done():
			return
		}
	}
}

// processBatch processes a single batch
func (bp *BatchProcessor) processBatch(batch *Batch) {
	startTime := time.Now()
	now := startTime
	batch.StartedAt = &now

	logger.Debug("Processing batch",
		zap.String("batch_id", batch.ID),
		zap.String("type", string(batch.Type)),
		zap.Int("size", batch.Size),
	)

	var results []interface{}
	var errors []error

	// Process batch
	if batch.Handler != nil {
		results, errors = batch.Handler(batch.Context, batch)
	}

	completionTime := time.Now()
	batch.CompletedAt = &completionTime
	batch.Results = results
	batch.Errors = errors

	// Send result
	result := &BatchResult{
		BatchID:     batch.ID,
		Results:     results,
		Errors:      errors,
		Duration:    completionTime.Sub(startTime),
		ProcessedAt: completionTime,
		Metadata:    batch.Metadata,
	}

	select {
	case bp.resultQueue <- result:
	default:
		logger.Warn("Result queue full, dropping batch result",
			zap.String("batch_id", batch.ID),
		)
	}

	// Update statistics
	bp.updateStats(batch, result)
}

// startResultCollector collects batch results
func (bp *BatchProcessor) startResultCollector() {
	bp.wg.Add(1)
	defer bp.wg.Done()

	for {
		select {
		case result := <-bp.resultQueue:
			if result == nil {
				return
			}
			bp.handleResult(result)
		case <-bp.ctx.Done():
			return
		}
	}
}

// handleResult handles batch processing result
func (bp *BatchProcessor) handleResult(result *BatchResult) {
	logger.Debug("Batch result received",
		zap.String("batch_id", result.BatchID),
		zap.Duration("duration", result.Duration),
		zap.Int("results", len(result.Results)),
		zap.Int("errors", len(result.Errors)),
	)

	// Notify prefetcher if enabled
	if bp.prefetcher != nil {
		bp.prefetcher.HandleResult(result)
	}

	// Update dynamic sizing if enabled
	if bp.dynamicSizing != nil {
		bp.dynamicSizing.UpdateMetrics(result)
	}
}

// startStatsCollector collects periodic statistics
func (bp *BatchProcessor) startStatsCollector() {
	bp.wg.Add(1)
	defer bp.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bp.collectStats()
		case <-bp.ctx.Done():
			return
		}
	}
}

// updateStats updates batch processing statistics
func (bp *BatchProcessor) updateStats(batch *Batch, result *BatchResult) {
	atomic.AddInt64(&bp.stats.TotalBatches, 1)
	atomic.AddInt64(&bp.stats.TotalItems, int64(batch.Size))
	atomic.AddInt64(&bp.stats.CompletedBatches, 1)
	atomic.AddInt64(&bp.stats.TotalProcessingTime, int64(result.Duration))

	if len(result.Errors) > 0 {
		atomic.AddInt64(&bp.stats.FailedBatches, 1)
	}
}

// collectStats collects current statistics
func (bp *BatchProcessor) collectStats() {
	bp.stats.LastUpdated = time.Now()

	// Calculate queue utilization
	bp.stats.QueueUtilization = float64(len(bp.batchQueue)) / float64(cap(bp.batchQueue))

	// Calculate average batch size
	totalBatches := atomic.LoadInt64(&bp.stats.TotalBatches)
	totalItems := atomic.LoadInt64(&bp.stats.TotalItems)
	
	if totalBatches > 0 {
		bp.stats.AvgBatchSize = float64(totalItems) / float64(totalBatches)
	}

	// Calculate average processing time
	completedBatches := atomic.LoadInt64(&bp.stats.CompletedBatches)
	totalProcessingTime := atomic.LoadInt64(&bp.stats.TotalProcessingTime)
	
	if completedBatches > 0 {
		bp.stats.AvgProcessingTime = time.Duration(totalProcessingTime / completedBatches)
	}

	// Calculate throughput
	if completedBatches > 0 {
		bp.stats.Throughput = float64(totalItems) / time.Since(time.Now()).Seconds()
	}

	logger.Debug("Batch processor stats",
		zap.Float64("queue_utilization", bp.stats.QueueUtilization),
		zap.Float64("avg_batch_size", bp.stats.AvgBatchSize),
		zap.Duration("avg_processing_time", bp.stats.AvgProcessingTime),
		zap.Float64("throughput", bp.stats.Throughput),
	)
}

// GetStats returns current processor statistics
func (bp *BatchProcessor) GetStats() BatchProcessorStats {
	bp.collectStats()
	return *bp.stats
}

// NewDynamicBatchSizer creates a new dynamic batch sizer
func NewDynamicBatchSizer(config DynamicSizingConfig) *DynamicBatchSizer {
	return &DynamicBatchSizer{
		config:          config,
		currentBatchSize: config.MinSize,
		performance:     &PerformanceMetrics{},
		strategy:        config.Strategy,
		stats:           &DynamicSizingStats{},
	}
}

// Start starts dynamic batch sizing
func (dbs *DynamicBatchSizer) Start(ctx context.Context) {
	if !dbs.config.Enabled {
		return
	}

	ticker := time.NewTicker(dbs.config.AdjustmentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dbs.adjustBatchSize()
		case <-ctx.Done():
			return
		}
	}
}

// adjustBatchSize adjusts batch size based on performance
func (dbs *DynamicBatchSizer) adjustBatchSize() {
	// Simplified adjustment logic
	switch dbs.strategy {
	case StrategyLatencyBased:
		dbs.adjustByLatency()
	case StrategyThroughputBased:
		dbs.adjustByThroughput()
	case StrategyAdaptive:
		dbs.adjustByAdaptiveStrategy()
	}
}

// adjustByLatency adjusts batch size based on latency
func (dbs *DynamicBatchSizer) adjustByLatency() {
	if dbs.performance.AvgLatency > dbs.config.TargetLatency {
		// Reduce batch size
		newSize := max(dbs.currentBatchSize-2, dbs.config.MinSize)
		dbs.setBatchSize(newSize)
	} else if dbs.performance.AvgLatency < dbs.config.TargetLatency/2 {
		// Increase batch size
		newSize := min(dbs.currentBatchSize+2, dbs.config.MaxSize)
		dbs.setBatchSize(newSize)
	}
}

// adjustByThroughput adjusts batch size based on throughput
func (dbs *DynamicBatchSizer) adjustByThroughput() {
	if dbs.performance.AvgThroughput < dbs.config.TargetThroughput {
		// Try increasing batch size
		newSize := min(dbs.currentBatchSize+1, dbs.config.MaxSize)
		dbs.setBatchSize(newSize)
	}
}

// adjustByAdaptiveStrategy adjusts batch size using adaptive strategy
func (dbs *DynamicBatchSizer) adjustByAdaptiveStrategy() {
	// Combine multiple metrics
	latencyRatio := float64(dbs.performance.AvgLatency) / float64(dbs.config.TargetLatency)
	throughputRatio := dbs.performance.AvgThroughput / dbs.config.TargetThroughput

	if latencyRatio > 1.2 && throughputRatio < 0.8 {
		// Reduce batch size
		newSize := max(dbs.currentBatchSize-1, dbs.config.MinSize)
		dbs.setBatchSize(newSize)
	} else if latencyRatio < 0.8 && throughputRatio > 1.2 {
		// Increase batch size
		newSize := min(dbs.currentBatchSize+1, dbs.config.MaxSize)
		dbs.setBatchSize(newSize)
	}
}

// setBatchSize sets new batch size
func (dbs *DynamicBatchSizer) setBatchSize(newSize int) {
	if newSize != dbs.currentBatchSize {
		dbs.currentBatchSize = newSize
		dbs.stats.SizeAdjustments++
		dbs.stats.CurrentSize = newSize
		dbs.stats.LastAdjustment = time.Now().Unix()

		if newSize <= dbs.config.MinSize {
			dbs.stats.MinSizeReached++
		}
		if newSize >= dbs.config.MaxSize {
			dbs.stats.MaxSizeReached++
		}

		logger.Debug("Batch size adjusted",
			zap.Int("old_size", dbs.currentBatchSize),
			zap.Int("new_size", newSize),
			zap.String("strategy", string(dbs.strategy)),
		)
	}
}

// UpdateMetrics updates performance metrics
func (dbs *DynamicBatchSizer) UpdateMetrics(result *BatchResult) {
	// Update performance metrics
	dbs.performance.LastUpdated = time.Now()
	// Simplified metric updates
}

// GetCurrentBatchSize returns current batch size
func (dbs *DynamicBatchSizer) GetCurrentBatchSize() int {
	return dbs.currentBatchSize
}

// NewBatchPrefetcher creates a new batch prefetcher
func NewBatchPrefetcher(config PrefetchConfig) *BatchPrefetcher {
	return &BatchPrefetcher{
		config:        config,
		prefetchQueue: make(chan *Batch, 50),
		cache:         make(map[string]*Batch),
		stats:         &PrefetchStats{},
	}
}

// Start starts batch prefetcher
func (bp *BatchPrefetcher) Start(ctx context.Context) {
	if !bp.config.Enabled {
		return
	}

	go bp.runPrefetcher(ctx)
}

// runPrefetcher runs the prefetcher
func (bp *BatchPrefetcher) runPrefetcher(ctx context.Context) {
	for {
		select {
		case batch := <-bp.prefetchQueue:
			bp.prefetchBatch(batch)
		case <-ctx.Done():
			return
		}
	}
}

// prefetchBatch prefetches a batch
func (bp *BatchPrefetcher) prefetchBatch(batch *Batch) {
	// Simplified prefetching
	atomic.AddInt64(&bp.stats.TotalPrefetches, 1)
	atomic.StoreInt64(&bp.stats.LastPrefetch, time.Now().Unix())
}

// HandleResult handles prefetch result
func (bp *BatchPrefetcher) HandleResult(result *BatchResult) {
	// Update cache statistics
	atomic.AddInt64(&bp.stats.CacheHits, 1)
	
	// Calculate hit rate
	total := atomic.LoadInt64(&bp.stats.TotalPrefetches)
	hits := atomic.LoadInt64(&bp.stats.CacheHits)
	if total > 0 {
		bp.stats.HitRate = float64(hits) / float64(total)
	}
}

// NewAsyncBatchProcessor creates a new async batch processor
func NewAsyncBatchProcessor(config AsyncProcessingConfig) *AsyncBatchProcessor {
	return &AsyncBatchProcessor{
		config:     config,
		workerPool: NewWorkerPool(config.NumWorkers, config.QueueSize, config.WorkerTimeout),
		stats:      &AsyncProcessingStats{},
	}
}

// Start starts async batch processor
func (abp *AsyncBatchProcessor) Start(ctx context.Context) {
	if !abp.config.Enabled {
		return
	}

	abp.workerPool.Start(ctx)
}

// Additional helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}