// Package cache provides multi-level caching (L1: memory, L2: Redis, L3: BadgerDB)
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Cache represents a multi-level cache
// This interface is exported for use by other packages
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Stats() CacheStats
}

// MultiLevelCache implements L1/L2/L3 caching
type MultiLevelCache struct {
	l1 *L1Cache
	l2 Cache // Redis (optional)
	l3 Cache // BadgerDB (optional)
	mu sync.RWMutex
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(l1Size int, l2 Cache, l3 Cache) *MultiLevelCache {
	return &MultiLevelCache{
		l1: NewL1Cache(l1Size),
		l2: l2,
		l3: l3,
	}
}

// Get retrieves a value from cache (L1 -> L2 -> L3)
func (c *MultiLevelCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Try L1 first
	value, err := c.l1.Get(key)
	if err == nil {
		return value, nil
	}

	// Try L2
	if c.l2 != nil {
		value, err = c.l2.Get(ctx, key)
		if err == nil {
			// Promote to L1
			_ = c.l1.Set(key, value, 0)
			return value, nil
		}
	}

	// Try L3
	if c.l3 != nil {
		value, err = c.l3.Get(ctx, key)
		if err == nil {
			// Promote to L2 and L1
			if c.l2 != nil {
				_ = c.l2.Set(ctx, key, value, time.Hour)
			}
			_ = c.l1.Set(key, value, 0)
			return value, nil
		}
	}

	return nil, ErrCacheMiss
}

// Set stores a value in all cache levels
func (c *MultiLevelCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Set in L1
	if err := c.l1.Set(key, value, ttl); err != nil {
		logger.Warn("Failed to set L1 cache", zap.String("key", key), zap.Error(err))
	}

	// Set in L2
	if c.l2 != nil {
		if err := c.l2.Set(ctx, key, value, ttl); err != nil {
			logger.Warn("Failed to set L2 cache", zap.String("key", key), zap.Error(err))
		}
	}

	// Set in L3 (no TTL for persistence)
	if c.l3 != nil {
		if err := c.l3.Set(ctx, key, value, 0); err != nil {
			logger.Warn("Failed to set L3 cache", zap.String("key", key), zap.Error(err))
		}
	}

	return nil
}

// Delete deletes a key from all cache levels
func (c *MultiLevelCache) Delete(ctx context.Context, key string) error {
	c.l1.Delete(key)

	if c.l2 != nil {
		_ = c.l2.Delete(ctx, key)
	}

	if c.l3 != nil {
		_ = c.l3.Delete(ctx, key)
	}

	return nil
}

// Clear clears all cache levels
func (c *MultiLevelCache) Clear(ctx context.Context) error {
	c.l1.Clear()

	if c.l2 != nil {
		_ = c.l2.Clear(ctx)
	}

	if c.l3 != nil {
		_ = c.l3.Clear(ctx)
	}

	return nil
}

// Stats returns cache statistics
func (c *MultiLevelCache) Stats() CacheStats {
	stats := c.l1.Stats()

	if c.l2 != nil {
		l2Stats := c.l2.Stats()
		stats.Hits += l2Stats.Hits
		stats.Misses += l2Stats.Misses
		stats.Size += l2Stats.Size
	}

	if c.l3 != nil {
		l3Stats := c.l3.Stats()
		stats.Hits += l3Stats.Hits
		stats.Misses += l3Stats.Misses
		stats.Size += l3Stats.Size
	}

	return stats
}

// L1Cache is the in-memory cache using sync.Map
type L1Cache struct {
	data  sync.Map
	size  int
	mu    sync.RWMutex
	stats CacheStats
}

// L1Entry represents a cache entry
type L1Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

// NewL1Cache creates a new L1 cache
func NewL1Cache(size int) *L1Cache {
	return &L1Cache{
		size: size,
	}
}

// Get retrieves a value from L1 cache
func (c *L1Cache) Get(key string) ([]byte, error) {
	val, ok := c.data.Load(key)
	if !ok {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}

	// Nil pointer check: ensure val is not nil before type assertion
	if val == nil {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}

	entry, ok := val.(*L1Entry)
	if !ok || entry == nil {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		c.data.Delete(key)
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}

	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()

	return entry.Value, nil
}

// Set stores a value in L1 cache
func (c *L1Cache) Set(key string, value []byte, ttl time.Duration) error {
	entry := &L1Entry{
		Value: value,
	}

	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}

	c.data.Store(key, entry)

	// Simple size management (evict oldest if needed)
	// In production, use LRU or similar
	if c.size > 0 {
		count := 0
		c.data.Range(func(_, _ interface{}) bool {
			count++
			return count <= c.size
		})
	}

	return nil
}

// Delete deletes a key from L1 cache
func (c *L1Cache) Delete(key string) {
	c.data.Delete(key)
}

// Clear clears the L1 cache
func (c *L1Cache) Clear() {
	c.data.Range(func(key, _ interface{}) bool {
		c.data.Delete(key)
		return true
	})
}

// Stats returns L1 cache statistics
func (c *L1Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	c.data.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	return CacheStats{
		Hits:   c.stats.Hits,
		Misses: c.stats.Misses,
		Size:   int64(count),
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits   int64
	Misses int64
	Size   int64
}

// Errors
var (
	ErrCacheMiss = &CacheError{Message: "cache miss"}
)

// CacheError represents a cache error
type CacheError struct {
	Message string
}

func (e *CacheError) Error() string {
	return e.Message
}
