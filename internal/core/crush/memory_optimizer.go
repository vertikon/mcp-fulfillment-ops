// Package crush implements memory optimization strategies for GLM-4.6
package crush

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// MemoryOptimizerConfig represents configuration for memory optimization
type MemoryOptimizerConfig struct {
	GCThreshold      float64        `json:"gc_threshold"`
	Interval         time.Duration  `json:"interval"`
	MaxMemoryMB      int            `json:"max_memory_mb"`
	CompactionRatio  float64        `json:"compaction_ratio"`
	EvictionPolicy   EvictionPolicy `json:"eviction_policy"`
	EnableGC         bool           `json:"enable_gc"`
	EnableCompaction bool           `json:"enable_compaction"`
}

// EvictionPolicy represents different eviction strategies
type EvictionPolicy string

const (
	PolicyLRU    EvictionPolicy = "lru"
	PolicyLFU    EvictionPolicy = "lfu"
	PolicyFIFO   EvictionPolicy = "fifo"
	PolicyRandom EvictionPolicy = "random"
	PolicyTTL    EvictionPolicy = "ttl"
)

// MemoryPool represents a pooled memory allocator
type MemoryPool struct {
	config           MemoryOptimizerConfig
	pools            map[int]*MemorySegment
	freeSegments     chan *MemorySegment
	usedSegments     map[string]*MemorySegment
	segmentsMu       sync.RWMutex
	stats            *MemoryPoolStats
	compactor        *MemoryCompactor
	garbageCollector *GarbageCollector
	ctx              context.Context
	cancel           context.CancelFunc
}

// MemorySegment represents a pooled memory segment
type MemorySegment struct {
	ID          string     `json:"id"`
	Size        int        `json:"size"`
	Data        []byte     `json:"data"`
	InUse       bool       `json:"in_use"`
	LastUsed    time.Time  `json:"last_used"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	AccessCount int64      `json:"access_count"`
	Locked      bool       `json:"locked"`
	Tags        []string   `json:"tags,omitempty"`
}

// MemoryCompactor handles memory compaction
type MemoryCompactor struct {
	config   CompactionConfig
	segments []*MemorySegment
	strategy CompactionStrategy
	stats    *CompactorStats
	running  bool
	mu       sync.RWMutex
}

// CompactionConfig represents compaction configuration
type CompactionConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Threshold        float64       `json:"threshold"`
	MinSegmentSize   int           `json:"min_segment_size"`
	MaxFragmentation float64       `json:"max_fragmentation"`
}

// CompactionStrategy represents different compaction strategies
type CompactionStrategy string

const (
	StrategyDefragment  CompactionStrategy = "defragment"
	StrategyConsolidate CompactionStrategy = "consolidate"
	StrategyReclaim     CompactionStrategy = "reclaim"
)

// GarbageCollector handles garbage collection optimization
type GarbageCollector struct {
	config  GCConfig
	stats   *GCStats
	running bool
	mu      sync.RWMutex
}

// GCConfig represents garbage collection configuration
type GCConfig struct {
	Enabled         bool          `json:"enabled"`
	Interval        time.Duration `json:"interval"`
	TargetGCPause   time.Duration `json:"target_gc_pause"`
	MaxGCPause      time.Duration `json:"max_gc_pause"`
	GCPercent       int           `json:"gc_percent"`
	SoftMemoryLimit int           `json:"soft_memory_limit"`
	HardMemoryLimit int           `json:"hard_memory_limit"`
}

// MemoryPoolStats tracks memory pool performance
type MemoryPoolStats struct {
	TotalSegments int64   `json:"total_segments"`
	UsedSegments  int64   `json:"used_segments"`
	FreeSegments  int64   `json:"free_segments"`
	TotalMemoryMB int64   `json:"total_memory_mb"` // MB as int64
	UsedMemoryMB  int64   `json:"used_memory_mb"`  // MB as int64
	FreeMemoryMB  int64   `json:"free_memory_mb"`  // MB as int64
	HitRate       float64 `json:"hit_rate"`
	MissRate      float64 `json:"miss_rate"`
	Evictions     int64   `json:"evictions"`
	Compactions   int64   `json:"compactions"`
	GCRuns        int64   `json:"gc_runs"`
	LastGC        int64   `json:"last_gc"`
	LastUpdated   int64   `json:"last_updated"`
}

// CompactorStats tracks compaction performance
type CompactorStats struct {
	TotalCompactions  int64   `json:"total_compactions"`
	CompactedBytes    int64   `json:"compacted_bytes"`
	FreedBytes        int64   `json:"freed_bytes"`
	AvgCompactionTime int64   `json:"avg_compaction_time"` // nanoseconds
	FragmentationRate float64 `json:"fragmentation_rate"`
	LastCompaction    int64   `json:"last_compaction"`
	LastUpdated       int64   `json:"last_updated"`
}

// GCStats tracks garbage collection performance
type GCStats struct {
	TotalGCs          int64 `json:"total_gcs"`
	AvgGCPause        int64 `json:"avg_gc_pause"`        // nanoseconds
	MaxGCPause        int64 `json:"max_gc_pause"`        // nanoseconds
	MemoryReclaimedMB int64 `json:"memory_reclaimed_mb"` // MB as int64
	LastGCTime        int64 `json:"last_gc_time"`
	LastUpdated       int64 `json:"last_updated"`
}

// NewMemoryOptimizer creates a new memory optimizer
func NewMemoryOptimizer(config MemoryOptimizerConfig) *MemoryPool {
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.MaxMemoryMB == 0 {
		config.MaxMemoryMB = 1024 // Default 1GB
	}

	logger.Info("Creating memory optimizer",
		zap.Float64("gc_threshold", config.GCThreshold),
		zap.Duration("interval", config.Interval),
		zap.Int("max_memory_mb", config.MaxMemoryMB),
		zap.String("eviction_policy", string(config.EvictionPolicy)),
	)

	ctx, cancel := context.WithCancel(context.Background())

	pool := &MemoryPool{
		config:       config,
		pools:        make(map[int]*MemorySegment),
		freeSegments: make(chan *MemorySegment, 1000),
		usedSegments: make(map[string]*MemorySegment),
		stats:        &MemoryPoolStats{},
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize compactor
	if config.EnableCompaction {
		pool.compactor = NewMemoryCompactor(CompactionConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Threshold:        config.CompactionRatio,
			MaxFragmentation: 0.3, // 30% fragmentation threshold
		})
	}

	// Initialize garbage collector
	if config.EnableGC {
		pool.garbageCollector = NewGarbageCollector(GCConfig{
			Enabled:         true,
			Interval:        10 * time.Second,
			TargetGCPause:   5 * time.Millisecond,
			MaxGCPause:      10 * time.Millisecond,
			GCPercent:       100,
			SoftMemoryLimit: int(float64(config.MaxMemoryMB) * 0.8),
			HardMemoryLimit: config.MaxMemoryMB,
		})
	}

	return pool
}

// Start starts the memory optimizer
func (mo *MemoryPool) Start() error {
	logger.Info("Starting memory optimizer")

	// Start memory monitoring
	go mo.startMemoryMonitoring()

	// Start compactor if enabled
	if mo.compactor != nil {
		go mo.compactor.Start(mo.ctx)
	}

	// Start garbage collector if enabled
	if mo.garbageCollector != nil {
		go mo.garbageCollector.Start(mo.ctx)
	}

	// Start statistics collector
	go mo.startStatsCollector()

	return nil
}

// Stop stops the memory optimizer
func (mo *MemoryPool) Stop() error {
	logger.Info("Stopping memory optimizer")

	mo.cancel()

	// Force garbage collection
	runtime.GC()

	return nil
}

// Allocate allocates memory from pool
func (mo *MemoryPool) Allocate(size int, tags ...string) (*MemorySegment, error) {
	// Try to find free segment
	select {
	case segment := <-mo.freeSegments:
		if segment.Size >= size {
			segment.InUse = true
			segment.LastUsed = time.Now()
			segment.AccessCount++
			segment.Tags = tags

			mo.segmentsMu.Lock()
			mo.usedSegments[segment.ID] = segment
			mo.segmentsMu.Unlock()

			return segment, nil
		}
	default:
	}

	// Create new segment
	segment := &MemorySegment{
		ID:          fmt.Sprintf("seg_%d_%d", size, time.Now().UnixNano()),
		Size:        size,
		Data:        make([]byte, size),
		InUse:       true,
		LastUsed:    time.Now(),
		AccessCount: 1,
		Tags:        tags,
	}

	mo.segmentsMu.Lock()
	mo.usedSegments[segment.ID] = segment
	mo.segmentsMu.Unlock()

	return segment, nil
}

// Deallocate deallocates memory back to pool
func (mo *MemoryPool) Deallocate(segment *MemorySegment) error {
	if segment == nil {
		return fmt.Errorf("segment is nil")
	}

	mo.segmentsMu.Lock()
	defer mo.segmentsMu.Unlock()

	if _, exists := mo.usedSegments[segment.ID]; !exists {
		return fmt.Errorf("segment not found in used segments")
	}

	// Clear data
	segment.Data = make([]byte, len(segment.Data))
	segment.InUse = false
	segment.LastUsed = time.Now()

	// Add back to free pool
	select {
	case mo.freeSegments <- segment:
	default:
		// Pool full, let GC handle it
	}

	delete(mo.usedSegments, segment.ID)

	return nil
}

// startMemoryMonitoring starts memory monitoring
func (mo *MemoryPool) startMemoryMonitoring() {
	ticker := time.NewTicker(mo.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mo.checkMemoryUsage()
		case <-mo.ctx.Done():
			return
		}
	}
}

// checkMemoryUsage checks current memory usage
func (mo *MemoryPool) checkMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentMemoryMB := float64(m.Alloc) / 1024 / 1024
	maxMemoryMB := float64(mo.config.MaxMemoryMB)

	// Calculate usage ratio
	usageRatio := currentMemoryMB / maxMemoryMB

	logger.Debug("Memory usage check",
		zap.Float64("current_mb", currentMemoryMB),
		zap.Float64("max_mb", maxMemoryMB),
		zap.Float64("usage_ratio", usageRatio),
	)

	// Trigger GC if threshold exceeded
	if usageRatio > mo.config.GCThreshold {
		logger.Info("Memory threshold exceeded, triggering GC",
			zap.Float64("usage_ratio", usageRatio),
			zap.Float64("threshold", mo.config.GCThreshold),
		)

		mo.triggerGarbageCollection()
	}

	// Trigger eviction if needed
	if usageRatio > 0.9 {
		logger.Warn("High memory usage, triggering eviction",
			zap.Float64("usage_ratio", usageRatio),
		)

		mo.triggerEviction()
	}
}

// triggerGarbageCollection triggers optimized garbage collection
func (mo *MemoryPool) triggerGarbageCollection() {
	if mo.garbageCollector != nil {
		mo.garbageCollector.TriggerGC()
	} else {
		// Default GC
		runtime.GC()
	}
}

// triggerEviction triggers memory eviction
func (mo *MemoryPool) triggerEviction() {
	mo.segmentsMu.Lock()
	defer mo.segmentsMu.Unlock()

	if len(mo.usedSegments) == 0 {
		return
	}

	// Collect candidates for eviction
	var candidates []*MemorySegment
	now := time.Now()

	for _, segment := range mo.usedSegments {
		if segment.Locked {
			continue
		}

		// Check TTL
		if segment.ExpiresAt != nil && now.After(*segment.ExpiresAt) {
			candidates = append(candidates, segment)
			continue
		}

		// Check based on policy
		switch mo.config.EvictionPolicy {
		case PolicyLRU:
			// Will be sorted by LastUsed
		case PolicyLFU:
			// Will be sorted by AccessCount
		case PolicyTTL:
			// Already handled
		}
	}

	// Sort candidates based on eviction policy
	mo.sortEvictionCandidates(candidates)

	// Evict candidates (remove oldest 20%)
	evictCount := max(1, len(candidates)/5)
	for i := 0; i < evictCount && i < len(candidates); i++ {
		segment := candidates[i]
		delete(mo.usedSegments, segment.ID)

		// Add to free pool
		select {
		case mo.freeSegments <- segment:
		default:
		}

		atomic.AddInt64(&mo.stats.Evictions, 1)

		logger.Debug("Evicted memory segment",
			zap.String("segment_id", segment.ID),
			zap.Int("size", segment.Size),
		)
	}
}

// sortEvictionCandidates sorts candidates based on eviction policy
func (mo *MemoryPool) sortEvictionCandidates(candidates []*MemorySegment) {
	switch mo.config.EvictionPolicy {
	case PolicyLRU:
		// Sort by LastUsed (oldest first)
		for i := 0; i < len(candidates)-1; i++ {
			for j := i + 1; j < len(candidates); j++ {
				if candidates[i].LastUsed.After(candidates[j].LastUsed) {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}
			}
		}
	case PolicyLFU:
		// Sort by AccessCount (lowest first)
		for i := 0; i < len(candidates)-1; i++ {
			for j := i + 1; j < len(candidates); j++ {
				if candidates[i].AccessCount > candidates[j].AccessCount {
					candidates[i], candidates[j] = candidates[j], candidates[i]
				}
			}
		}
	}
}

// startStatsCollector collects periodic statistics
func (mo *MemoryPool) startStatsCollector() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mo.collectStats()
		case <-mo.ctx.Done():
			return
		}
	}
}

// collectStats collects current statistics
func (mo *MemoryPool) collectStats() {
	mo.segmentsMu.RLock()
	defer mo.segmentsMu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Update segment statistics
	atomic.StoreInt64(&mo.stats.TotalSegments, int64(len(mo.usedSegments)+len(mo.freeSegments)))
	atomic.StoreInt64(&mo.stats.UsedSegments, int64(len(mo.usedSegments)))
	atomic.StoreInt64(&mo.stats.FreeSegments, int64(len(mo.freeSegments)))

	// Calculate memory statistics
	totalMemory := 0
	usedMemory := 0

	for _, segment := range mo.usedSegments {
		usedMemory += segment.Size
	}

	for segment := range mo.freeSegments {
		totalMemory += segment.Size
	}

	totalMemory += usedMemory

	atomic.StoreInt64(&mo.stats.TotalMemoryMB, int64(float64(totalMemory)/1024/1024))
	atomic.StoreInt64(&mo.stats.UsedMemoryMB, int64(float64(usedMemory)/1024/1024))
	atomic.StoreInt64(&mo.stats.FreeMemoryMB, int64(float64(totalMemory-usedMemory)/1024/1024))

	// Update hit/miss rates
	totalRequests := atomic.LoadInt64(&mo.stats.UsedSegments) + atomic.LoadInt64(&mo.stats.Evictions)
	if totalRequests > 0 {
		hitRate := float64(atomic.LoadInt64(&mo.stats.UsedSegments)) / float64(totalRequests)
		missRate := float64(atomic.LoadInt64(&mo.stats.Evictions)) / float64(totalRequests)

		// These are simplified calculations
		logger.Debug("Memory pool statistics",
			zap.Float64("hit_rate", hitRate),
			zap.Float64("miss_rate", missRate),
			zap.Float64("used_memory_mb", float64(usedMemory)/1024/1024),
		)
	}

	atomic.StoreInt64(&mo.stats.LastUpdated, time.Now().Unix())
}

// GetStats returns current memory pool statistics
func (mo *MemoryPool) GetStats() MemoryPoolStats {
	mo.collectStats()
	return *mo.stats
}

// NewMemoryCompactor creates a new memory compactor
func NewMemoryCompactor(config CompactionConfig) *MemoryCompactor {
	return &MemoryCompactor{
		config:   config,
		segments: make([]*MemorySegment, 0),
		strategy: StrategyDefragment,
		stats:    &CompactorStats{},
	}
}

// Start starts memory compactor
func (mc *MemoryCompactor) Start(ctx context.Context) {
	if !mc.config.Enabled {
		return
	}

	logger.Info("Starting memory compactor")
	mc.running = true

	ticker := time.NewTicker(mc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.compact()
		case <-ctx.Done():
			mc.running = false
			return
		}
	}
}

// compact performs memory compaction
func (mc *MemoryCompactor) compact() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.segments) == 0 {
		return
	}

	start := time.Now()

	// Simplified compaction logic
	compactedBytes := int64(0)
	freedBytes := int64(0)

	// In practice, this would:
	// 1. Identify fragmented segments
	// 2. Consolidate adjacent segments
	// 3. Reclaim unused space

	atomic.AddInt64(&mc.stats.TotalCompactions, 1)
	atomic.AddInt64(&mc.stats.CompactedBytes, compactedBytes)
	atomic.AddInt64(&mc.stats.FreedBytes, freedBytes)

	duration := time.Since(start)

	// Update average compaction time
	total := atomic.LoadInt64(&mc.stats.TotalCompactions)
	current := atomic.LoadInt64(&mc.stats.AvgCompactionTime)
	avg := (current*total + duration.Nanoseconds()) / (total + 1)
	atomic.StoreInt64(&mc.stats.AvgCompactionTime, avg)

	atomic.StoreInt64(&mc.stats.LastCompaction, time.Now().Unix())

	logger.Debug("Memory compaction completed",
		zap.Int64("compacted_bytes", compactedBytes),
		zap.Int64("freed_bytes", freedBytes),
		zap.Duration("duration", duration),
	)
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector(config GCConfig) *GarbageCollector {
	return &GarbageCollector{
		config: config,
		stats:  &GCStats{},
	}
}

// Start starts garbage collector
func (gc *GarbageCollector) Start(ctx context.Context) {
	if !gc.config.Enabled {
		return
	}

	logger.Info("Starting optimized garbage collector")
	gc.running = true

	// Set GOGC if specified
	if gc.config.GCPercent > 0 {
		debug.SetGCPercent(gc.config.GCPercent)
	}

	ticker := time.NewTicker(gc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gc.checkAndTriggerGC()
		case <-ctx.Done():
			gc.running = false
			return
		}
	}
}

// checkAndTriggerGC checks memory and triggers GC if needed
func (gc *GarbageCollector) checkAndTriggerGC() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check if we need to trigger GC
	if gc.config.HardMemoryLimit > 0 {
		currentMemoryMB := float64(m.Alloc) / 1024 / 1024
		if currentMemoryMB > float64(gc.config.HardMemoryLimit) {
			gc.TriggerGC()
			return
		}
	}

	if gc.config.SoftMemoryLimit > 0 {
		currentMemoryMB := float64(m.Alloc) / 1024 / 1024
		if currentMemoryMB > float64(gc.config.SoftMemoryLimit) {
			gc.TriggerGC()
			return
		}
	}
}

// TriggerGC triggers optimized garbage collection
func (gc *GarbageCollector) TriggerGC() {
	start := time.Now()

	// Get memory stats before GC
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	// Force garbage collection
	runtime.GC()

	// Get memory stats after GC
	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	pause := time.Since(start)
	reclaimedMB := float64(mBefore.Alloc-mAfter.Alloc) / 1024 / 1024

	// Update statistics
	atomic.AddInt64(&gc.stats.TotalGCs, 1)
	atomic.StoreInt64(&gc.stats.MemoryReclaimedMB, int64(reclaimedMB))
	atomic.StoreInt64(&gc.stats.LastGCTime, time.Now().Unix())

	// Update average pause time
	total := atomic.LoadInt64(&gc.stats.TotalGCs)
	current := atomic.LoadInt64(&gc.stats.AvgGCPause)
	avg := (current*(total-1) + pause.Nanoseconds()) / total
	atomic.StoreInt64(&gc.stats.AvgGCPause, avg)

	// Update max pause time
	maxPause := atomic.LoadInt64(&gc.stats.MaxGCPause)
	if pause.Nanoseconds() > maxPause {
		atomic.StoreInt64(&gc.stats.MaxGCPause, pause.Nanoseconds())
	}

	logger.Debug("Optimized GC completed",
		zap.Duration("pause", pause),
		zap.Float64("reclaimed_mb", reclaimedMB),
		zap.Uint64("heap_alloc_before", mBefore.HeapAlloc),
		zap.Uint64("heap_alloc_after", mAfter.HeapAlloc),
	)
}
