package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// CacheLevel represents different cache levels
type CacheLevel string

const (
	CacheLevelL1 CacheLevel = "L1" // Local in-memory cache
	CacheLevelL2 CacheLevel = "L2" // Cluster cache (shared memory)
	CacheLevelL3 CacheLevel = "L3" // Distributed cache (Redis/Consul)
)

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key        string                 `json:"key"`
	Value      interface{}            `json:"value"`
	Level      CacheLevel             `json:"level"`
	ExpiresAt  time.Time              `json:"expires_at"`
	CreatedAt  time.Time              `json:"created_at"`
	AccessedAt time.Time              `json:"accessed_at"`
	Hits       int64                  `json:"hits"`
	Size       int64                  `json:"size"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	L1MaxSize      int           `json:"l1_max_size"`
	L1TTL          time.Duration `json:"l1_ttl"`
	L2MaxSize      int           `json:"l2_max_size"`
	L2TTL          time.Duration `json:"l2_ttl"`
	L3MaxSize      int           `json:"l3_max_size"`
	L3TTL          time.Duration `json:"l3_ttl"`
	EnableL1       bool          `json:"enable_l1"`
	EnableL2       bool          `json:"enable_l2"`
	EnableL3       bool          `json:"enable_l3"`
	EvictionPolicy string        `json:"eviction_policy"` // LRU, LFU, FIFO
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		L1MaxSize:      10000,        // Aumentado de 1000 para 10000
		L1TTL:          5 * time.Minute,
		L2MaxSize:      100000,       // Aumentado de 10000 para 100000
		L2TTL:          15 * time.Minute,
		L3MaxSize:      1000000,      // Aumentado de 100000 para 1000000
		L3TTL:          1 * time.Hour,
		EnableL1:       true,
		EnableL2:       true,
		EnableL3:       false,
		EvictionPolicy: "LRU",
		CleanupInterval: 5 * time.Minute, // Aumentado de 1min para 5min (menos overhead)
	}
}

// StateCache interface for state caching
type StateCache interface {
	// Basic operations
	Get(ctx context.Context, key string) (*CacheEntry, error)
	Set(ctx context.Context, key string, value interface{}, level CacheLevel, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context, level CacheLevel) error
	
	// Multi-level operations
	GetFromLevel(ctx context.Context, key string, level CacheLevel) (*CacheEntry, error)
	SetToLevel(ctx context.Context, key string, value interface{}, level CacheLevel, ttl time.Duration) error
	
	// Statistics
	GetStats(ctx context.Context) (*CacheStats, error)
	GetLevelStats(ctx context.Context, level CacheLevel) (*LevelStats, error)
	
	// Health
	Health(ctx context.Context) (*CacheHealth, error)
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalRequests   int64                  `json:"total_requests"`
	TotalHits       int64                  `json:"total_hits"`
	TotalMisses     int64                  `json:"total_misses"`
	HitRate         float64                `json:"hit_rate"`
	L1Stats         *LevelStats            `json:"l1_stats"`
	L2Stats         *LevelStats            `json:"l2_stats"`
	L3Stats         *LevelStats            `json:"l3_stats"`
	TotalSize       int64                  `json:"total_size"`
	Evictions       int64                  `json:"evictions"`
}

// LevelStats represents statistics for a cache level
type LevelStats struct {
	Level         CacheLevel `json:"level"`
	Size          int        `json:"size"`
	MaxSize       int        `json:"max_size"`
	Hits          int64      `json:"hits"`
	Misses        int64      `json:"misses"`
	HitRate       float64    `json:"hit_rate"`
	Evictions     int64      `json:"evictions"`
	TotalSize     int64      `json:"total_size"`
}

// CacheHealth represents cache health status
type CacheHealth struct {
	Status      string                 `json:"status"`
	L1Healthy   bool                  `json:"l1_healthy"`
	L2Healthy   bool                  `json:"l2_healthy"`
	L3Healthy   bool                  `json:"l3_healthy"`
	Timestamp   time.Time             `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// StateCacheImpl implements StateCache interface with L1/L2/L3 levels
type StateCacheImpl struct {
	config    *CacheConfig
	l1Cache   map[string]*CacheEntry // L1: Local in-memory
	l2Cache   map[string]*CacheEntry // L2: Cluster (simulated)
	l3Cache   map[string]*CacheEntry // L3: Distributed (simulated)
	mu        sync.RWMutex
	logger    *zap.Logger
	stats     *CacheStats
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewStateCache creates a new multi-level state cache
func NewStateCache(config *CacheConfig) *StateCacheImpl {
	if config == nil {
		config = DefaultCacheConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	cache := &StateCacheImpl{
		config: config,
		l1Cache: make(map[string]*CacheEntry),
		l2Cache: make(map[string]*CacheEntry),
		l3Cache: make(map[string]*CacheEntry),
		stats: &CacheStats{
			L1Stats: &LevelStats{Level: CacheLevelL1},
			L2Stats: &LevelStats{Level: CacheLevelL2},
			L3Stats: &LevelStats{Level: CacheLevelL3},
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger.Get(),
	}
	
	// Start background cleanup
	go cache.backgroundCleanup()
	
	cache.logger.Info("Multi-level state cache initialized",
		zap.Bool("L1", config.EnableL1),
		zap.Bool("L2", config.EnableL2),
		zap.Bool("L3", config.EnableL3))
	
	return cache
}

// Get retrieves a value from cache (checks L1 -> L2 -> L3)
func (sc *StateCacheImpl) Get(ctx context.Context, key string) (*CacheEntry, error) {
	atomic.AddInt64(&sc.stats.TotalRequests, 1)
	
	// Try L1 first
	if sc.config.EnableL1 {
		entry, err := sc.GetFromLevel(ctx, key, CacheLevelL1)
		if err == nil && entry != nil {
			atomic.AddInt64(&sc.stats.TotalHits, 1)
			atomic.AddInt64(&sc.stats.L1Stats.Hits, 1)
			return entry, nil
		}
		atomic.AddInt64(&sc.stats.L1Stats.Misses, 1)
	}
	
	// Try L2
	if sc.config.EnableL2 {
		entry, err := sc.GetFromLevel(ctx, key, CacheLevelL2)
		if err == nil && entry != nil {
			atomic.AddInt64(&sc.stats.TotalHits, 1)
			atomic.AddInt64(&sc.stats.L2Stats.Hits, 1)
			// Promote to L1
			if sc.config.EnableL1 {
				sc.SetToLevel(ctx, key, entry.Value, CacheLevelL1, sc.config.L1TTL)
			}
			return entry, nil
		}
		atomic.AddInt64(&sc.stats.L2Stats.Misses, 1)
	}
	
	// Try L3
	if sc.config.EnableL3 {
		entry, err := sc.GetFromLevel(ctx, key, CacheLevelL3)
		if err == nil && entry != nil {
			atomic.AddInt64(&sc.stats.TotalHits, 1)
			atomic.AddInt64(&sc.stats.L3Stats.Hits, 1)
			// Promote to L2 and L1
			if sc.config.EnableL2 {
				sc.SetToLevel(ctx, key, entry.Value, CacheLevelL2, sc.config.L2TTL)
			}
			if sc.config.EnableL1 {
				sc.SetToLevel(ctx, key, entry.Value, CacheLevelL1, sc.config.L1TTL)
			}
			return entry, nil
		}
		atomic.AddInt64(&sc.stats.L3Stats.Misses, 1)
	}
	
	// Cache miss
	atomic.AddInt64(&sc.stats.TotalMisses, 1)
	return nil, fmt.Errorf("key not found in cache: %s", key)
}

// Set stores a value in cache (stores in all enabled levels)
func (sc *StateCacheImpl) Set(ctx context.Context, key string, value interface{}, level CacheLevel, ttl time.Duration) error {
	// Determine TTL based on level if not specified
	if ttl == 0 {
		switch level {
		case CacheLevelL1:
			ttl = sc.config.L1TTL
		case CacheLevelL2:
			ttl = sc.config.L2TTL
		case CacheLevelL3:
			ttl = sc.config.L3TTL
		}
	}
	
	// Store in specified level
	err := sc.SetToLevel(ctx, key, value, level, ttl)
	if err != nil {
		return err
	}
	
	// Also store in lower levels if enabled
	if level == CacheLevelL3 {
		if sc.config.EnableL2 {
			sc.SetToLevel(ctx, key, value, CacheLevelL2, sc.config.L2TTL)
		}
		if sc.config.EnableL1 {
			sc.SetToLevel(ctx, key, value, CacheLevelL1, sc.config.L1TTL)
		}
	} else if level == CacheLevelL2 && sc.config.EnableL1 {
		sc.SetToLevel(ctx, key, value, CacheLevelL1, sc.config.L1TTL)
	}
	
	return nil
}

// Delete removes a key from all cache levels
func (sc *StateCacheImpl) Delete(ctx context.Context, key string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	delete(sc.l1Cache, key)
	delete(sc.l2Cache, key)
	delete(sc.l3Cache, key)
	
	sc.logger.Debug("Cache key deleted", zap.String("key", key))
	return nil
}

// Clear clears all entries from a specific level
func (sc *StateCacheImpl) Clear(ctx context.Context, level CacheLevel) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	switch level {
	case CacheLevelL1:
		sc.l1Cache = make(map[string]*CacheEntry)
	case CacheLevelL2:
		sc.l2Cache = make(map[string]*CacheEntry)
	case CacheLevelL3:
		sc.l3Cache = make(map[string]*CacheEntry)
	default:
		return fmt.Errorf("invalid cache level: %s", level)
	}
	
	sc.logger.Info("Cache level cleared", zap.String("level", string(level)))
	return nil
}

// GetFromLevel retrieves a value from a specific cache level
func (sc *StateCacheImpl) GetFromLevel(ctx context.Context, key string, level CacheLevel) (*CacheEntry, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	var cache map[string]*CacheEntry
	switch level {
	case CacheLevelL1:
		if !sc.config.EnableL1 {
			return nil, fmt.Errorf("L1 cache not enabled")
		}
		cache = sc.l1Cache
	case CacheLevelL2:
		if !sc.config.EnableL2 {
			return nil, fmt.Errorf("L2 cache not enabled")
		}
		cache = sc.l2Cache
	case CacheLevelL3:
		if !sc.config.EnableL3 {
			return nil, fmt.Errorf("L3 cache not enabled")
		}
		cache = sc.l3Cache
	default:
		return nil, fmt.Errorf("invalid cache level: %s", level)
	}
	
	entry, exists := cache[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}
	
	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return nil, fmt.Errorf("entry expired")
	}
	
	// Update access info
	entry.AccessedAt = time.Now()
	entry.Hits++
	
	return entry, nil
}

// SetToLevel stores a value in a specific cache level
func (sc *StateCacheImpl) SetToLevel(ctx context.Context, key string, value interface{}, level CacheLevel, ttl time.Duration) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	var cache map[string]*CacheEntry
	var maxSize int
	
	switch level {
	case CacheLevelL1:
		if !sc.config.EnableL1 {
			return fmt.Errorf("L1 cache not enabled")
		}
		cache = sc.l1Cache
		maxSize = sc.config.L1MaxSize
	case CacheLevelL2:
		if !sc.config.EnableL2 {
			return fmt.Errorf("L2 cache not enabled")
		}
		cache = sc.l2Cache
		maxSize = sc.config.L2MaxSize
	case CacheLevelL3:
		if !sc.config.EnableL3 {
			return fmt.Errorf("L3 cache not enabled")
		}
		cache = sc.l3Cache
		maxSize = sc.config.L3MaxSize
	default:
		return fmt.Errorf("invalid cache level: %s", level)
	}
	
	// Check size limit and evict if needed
	if len(cache) >= maxSize {
		sc.evictEntry(cache, level)
		atomic.AddInt64(&sc.stats.Evictions, 1)
	}
	
	// Create entry
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Level:      level,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Hits:       0,
		Metadata:   make(map[string]interface{}),
	}
	
	cache[key] = entry
	
	sc.logger.Debug("Cache entry set",
		zap.String("key", key),
		zap.String("level", string(level)))
	
	return nil
}

// GetStats returns cache statistics
func (sc *StateCacheImpl) GetStats(ctx context.Context) (*CacheStats, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	stats := *sc.stats
	
	// Calculate hit rate
	if stats.TotalRequests > 0 {
		stats.HitRate = float64(stats.TotalHits) / float64(stats.TotalRequests) * 100
	}
	
	// Update level stats
	stats.L1Stats.Size = len(sc.l1Cache)
	stats.L1Stats.MaxSize = sc.config.L1MaxSize
	if stats.L1Stats.Hits+stats.L1Stats.Misses > 0 {
		stats.L1Stats.HitRate = float64(stats.L1Stats.Hits) / float64(stats.L1Stats.Hits+stats.L1Stats.Misses) * 100
	}
	
	stats.L2Stats.Size = len(sc.l2Cache)
	stats.L2Stats.MaxSize = sc.config.L2MaxSize
	if stats.L2Stats.Hits+stats.L2Stats.Misses > 0 {
		stats.L2Stats.HitRate = float64(stats.L2Stats.Hits) / float64(stats.L2Stats.Hits+stats.L2Stats.Misses) * 100
	}
	
	stats.L3Stats.Size = len(sc.l3Cache)
	stats.L3Stats.MaxSize = sc.config.L3MaxSize
	if stats.L3Stats.Hits+stats.L3Stats.Misses > 0 {
		stats.L3Stats.HitRate = float64(stats.L3Stats.Hits) / float64(stats.L3Stats.Hits+stats.L3Stats.Misses) * 100
	}
	
	return &stats, nil
}

// GetLevelStats returns statistics for a specific level
func (sc *StateCacheImpl) GetLevelStats(ctx context.Context, level CacheLevel) (*LevelStats, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	var stats *LevelStats
	var cache map[string]*CacheEntry
	
	switch level {
	case CacheLevelL1:
		stats = sc.stats.L1Stats
		cache = sc.l1Cache
	case CacheLevelL2:
		stats = sc.stats.L2Stats
		cache = sc.l2Cache
	case CacheLevelL3:
		stats = sc.stats.L3Stats
		cache = sc.l3Cache
	default:
		return nil, fmt.Errorf("invalid cache level: %s", level)
	}
	
	levelStats := *stats
	levelStats.Size = len(cache)
	
	return &levelStats, nil
}

// Health returns cache health status
func (sc *StateCacheImpl) Health(ctx context.Context) (*CacheHealth, error) {
	return &CacheHealth{
		Status:    "healthy",
		L1Healthy: sc.config.EnableL1,
		L2Healthy: sc.config.EnableL2,
		L3Healthy: sc.config.EnableL3,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"l1_size": len(sc.l1Cache),
			"l2_size": len(sc.l2Cache),
			"l3_size": len(sc.l3Cache),
		},
	}, nil
}

// Private helper methods

func (sc *StateCacheImpl) evictEntry(cache map[string]*CacheEntry, level CacheLevel) {
	switch sc.config.EvictionPolicy {
	case "LRU":
		sc.evictLRU(cache, level)
	case "LFU":
		sc.evictLFU(cache, level)
	case "FIFO":
		sc.evictFIFO(cache, level)
	default:
		sc.evictLRU(cache, level)
	}
}

func (sc *StateCacheImpl) evictLRU(cache map[string]*CacheEntry, level CacheLevel) {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	
	for key, entry := range cache {
		if entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
		}
	}
	
	if oldestKey != "" {
		delete(cache, oldestKey)
		atomic.AddInt64(&sc.stats.Evictions, 1)
	}
}

func (sc *StateCacheImpl) evictLFU(cache map[string]*CacheEntry, level CacheLevel) {
	var leastKey string
	var leastHits int64 = int64(^uint64(0) >> 1)
	
	for key, entry := range cache {
		if entry.Hits < leastHits {
			leastKey = key
			leastHits = entry.Hits
		}
	}
	
	if leastKey != "" {
		delete(cache, leastKey)
		atomic.AddInt64(&sc.stats.Evictions, 1)
	}
}

func (sc *StateCacheImpl) evictFIFO(cache map[string]*CacheEntry, level CacheLevel) {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	
	for key, entry := range cache {
		if entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}
	
	if oldestKey != "" {
		delete(cache, oldestKey)
		atomic.AddInt64(&sc.stats.Evictions, 1)
	}
}

func (sc *StateCacheImpl) backgroundCleanup() {
	ticker := time.NewTicker(sc.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-sc.ctx.Done():
			sc.logger.Info("Cache background cleanup stopped")
			return
		case <-ticker.C:
			sc.cleanupExpired()
		}
	}
}

func (sc *StateCacheImpl) cleanupExpired() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	now := time.Now()
	
	// Cleanup L1
	for key, entry := range sc.l1Cache {
		if now.After(entry.ExpiresAt) {
			delete(sc.l1Cache, key)
		}
	}
	
	// Cleanup L2
	for key, entry := range sc.l2Cache {
		if now.After(entry.ExpiresAt) {
			delete(sc.l2Cache, key)
		}
	}
	
	// Cleanup L3
	for key, entry := range sc.l3Cache {
		if now.After(entry.ExpiresAt) {
			delete(sc.l3Cache, key)
		}
	}
}

// Close closes the cache
func (sc *StateCacheImpl) Close() error {
	sc.cancel()
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.l1Cache = make(map[string]*CacheEntry)
	sc.l2Cache = make(map[string]*CacheEntry)
	sc.l3Cache = make(map[string]*CacheEntry)
	
	sc.logger.Info("State cache closed")
	return nil
}
