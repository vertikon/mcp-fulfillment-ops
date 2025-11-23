package cache

import (
	"context"
	"testing"
	"time"
)

// MockDistributionHandler for testing
type MockDistributionHandler struct {
	invalidations []string
	updates       []string
	clears        []CacheLevel
}

func (h *MockDistributionHandler) HandleInvalidation(ctx context.Context, key string, level CacheLevel) error {
	h.invalidations = append(h.invalidations, key)
	return nil
}

func (h *MockDistributionHandler) HandleUpdate(ctx context.Context, key string, value interface{}, level CacheLevel) error {
	h.updates = append(h.updates, key)
	return nil
}

func (h *MockDistributionHandler) HandleClear(ctx context.Context, level CacheLevel) error {
	h.clears = append(h.clears, level)
	return nil
}

func TestNewCacheDistribution(t *testing.T) {
	config := DefaultDistributionConfig()
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)

	if distribution == nil {
		t.Fatal("NewCacheDistribution returned nil")
	}
}

func TestCacheDistributionImpl_PublishInvalidation(t *testing.T) {
	config := DefaultDistributionConfig()
	config.EnableDistribution = true
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)
	ctx := context.Background()

	err := distribution.PublishInvalidation(ctx, "key1", CacheLevelL1)
	if err != nil {
		t.Fatalf("PublishInvalidation() error = %v", err)
	}
}

func TestCacheDistributionImpl_PublishUpdate(t *testing.T) {
	config := DefaultDistributionConfig()
	config.EnableDistribution = true
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)
	ctx := context.Background()

	err := distribution.PublishUpdate(ctx, "key1", "value1", CacheLevelL1)
	if err != nil {
		t.Fatalf("PublishUpdate() error = %v", err)
	}
}

func TestCacheDistributionImpl_PublishClear(t *testing.T) {
	config := DefaultDistributionConfig()
	config.EnableDistribution = true
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)
	ctx := context.Background()

	err := distribution.PublishClear(ctx, CacheLevelL1)
	if err != nil {
		t.Fatalf("PublishClear() error = %v", err)
	}
}

func TestCacheDistributionImpl_Subscribe_Unsubscribe(t *testing.T) {
	config := DefaultDistributionConfig()
	config.EnableDistribution = true
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)
	ctx := context.Background()

	handler := &MockDistributionHandler{}

	// Subscribe
	err := distribution.Subscribe(ctx, handler)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	// Publish invalidation
	_ = distribution.PublishInvalidation(ctx, "key1", CacheLevelL1)

	// Give handler time to process
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe
	err = distribution.Unsubscribe(ctx)
	if err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
}

func TestCacheDistributionImpl_GetDistributionStats(t *testing.T) {
	config := DefaultDistributionConfig()
	config.EnableDistribution = true
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)
	ctx := context.Background()

	// Publish some messages
	_ = distribution.PublishInvalidation(ctx, "key1", CacheLevelL1)
	_ = distribution.PublishUpdate(ctx, "key2", "value2", CacheLevelL1)

	// Get stats
	stats, err := distribution.GetDistributionStats(ctx)
	if err != nil {
		t.Fatalf("GetDistributionStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetDistributionStats() returned nil")
	}

	if stats.MessagesPublished < 2 {
		t.Errorf("Expected >= 2 published messages, got %d", stats.MessagesPublished)
	}
}

func TestDefaultDistributionConfig(t *testing.T) {
	config := DefaultDistributionConfig()

	if config == nil {
		t.Fatal("DefaultDistributionConfig returned nil")
	}

	if config.Strategy == "" {
		t.Error("Strategy should not be empty")
	}
}

func TestCacheDistributionImpl_Close(t *testing.T) {
	config := DefaultDistributionConfig()
	config.EnableDistribution = true
	cache := NewStateCache(nil)
	distribution := NewCacheDistribution(config, cache)

	err := distribution.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
