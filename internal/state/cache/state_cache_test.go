package cache

import (
	"context"
	"testing"
	"time"
)

func TestNewStateCache(t *testing.T) {
	config := DefaultCacheConfig()
	cache := NewStateCache(config)

	if cache == nil {
		t.Fatal("NewStateCache returned nil")
	}

	if cache.l1Cache == nil {
		t.Error("l1Cache should not be nil")
	}
}

func TestStateCacheImpl_Get_Set(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	// Set value in L1
	err := cache.Set(ctx, "test-key", "test-value", CacheLevelL1, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get value
	entry, err := cache.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if entry == nil {
		t.Fatal("Get() returned nil")
	}

	if entry.Value != "test-value" {
		t.Errorf("Expected 'test-value', got '%v'", entry.Value)
	}

	if entry.Level != CacheLevelL1 {
		t.Errorf("Expected level L1, got %s", entry.Level)
	}
}

func TestStateCacheImpl_MultiLevel(t *testing.T) {
	config := DefaultCacheConfig()
	config.EnableL1 = true
	config.EnableL2 = true
	config.EnableL3 = true
	cache := NewStateCache(config)
	ctx := context.Background()

	// Set value in L3
	err := cache.Set(ctx, "l3-key", "l3-value", CacheLevelL3, 5*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get should find in L1 (promoted)
	entry, err := cache.Get(ctx, "l3-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if entry == nil {
		t.Fatal("Get() returned nil")
	}

	if entry.Value != "l3-value" {
		t.Errorf("Expected 'l3-value', got '%v'", entry.Value)
	}
}

func TestStateCacheImpl_Delete(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	// Set value
	_ = cache.Set(ctx, "delete-key", "value", CacheLevelL1, 5*time.Minute)

	// Delete value
	err := cache.Delete(ctx, "delete-key")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Try to get deleted value (should fail)
	_, err = cache.Get(ctx, "delete-key")
	if err == nil {
		t.Error("Get() should fail for deleted key")
	}
}

func TestStateCacheImpl_Clear(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	// Set values
	_ = cache.Set(ctx, "key1", "value1", CacheLevelL1, 5*time.Minute)
	_ = cache.Set(ctx, "key2", "value2", CacheLevelL1, 5*time.Minute)

	// Clear L1
	err := cache.Clear(ctx, CacheLevelL1)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Try to get (should fail)
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("Get() should fail after clear")
	}
}

func TestStateCacheImpl_GetStats(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	// Perform operations
	_ = cache.Set(ctx, "key1", "value1", CacheLevelL1, 5*time.Minute)
	_ = cache.Get(ctx, "key1")
	_ = cache.Get(ctx, "nonexistent")

	stats, err := cache.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}

	if stats.TotalRequests < 2 {
		t.Errorf("Expected >= 2 requests, got %d", stats.TotalRequests)
	}
}

func TestStateCacheImpl_GetLevelStats(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	// Set value in L1
	_ = cache.Set(ctx, "key1", "value1", CacheLevelL1, 5*time.Minute)

	// Get L1 stats
	stats, err := cache.GetLevelStats(ctx, CacheLevelL1)
	if err != nil {
		t.Fatalf("GetLevelStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetLevelStats() returned nil")
	}

	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
}

func TestStateCacheImpl_Health(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	health, err := cache.Health(ctx)
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if health == nil {
		t.Fatal("Health() returned nil")
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}
}

func TestStateCacheImpl_TTL(t *testing.T) {
	cache := NewStateCache(nil)
	ctx := context.Background()

	// Set value with short TTL
	err := cache.Set(ctx, "ttl-key", "value", CacheLevelL1, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get immediately (should succeed)
	_, err = cache.Get(ctx, "ttl-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Get after TTL (should fail)
	_, err = cache.Get(ctx, "ttl-key")
	if err == nil {
		t.Error("Get() should fail for expired key")
	}
}

func TestStateCacheImpl_Eviction(t *testing.T) {
	config := DefaultCacheConfig()
	config.L1MaxSize = 2
	config.EvictionPolicy = "LRU"
	cache := NewStateCache(config)
	ctx := context.Background()

	// Fill cache beyond max size
	_ = cache.Set(ctx, "key1", "value1", CacheLevelL1, 5*time.Minute)
	_ = cache.Set(ctx, "key2", "value2", CacheLevelL1, 5*time.Minute)
	_ = cache.Set(ctx, "key3", "value3", CacheLevelL1, 5*time.Minute) // Should evict key1

	// key1 should be evicted
	_, err := cache.Get(ctx, "key1")
	if err == nil {
		t.Error("Get() should fail for evicted key")
	}

	// key2 and key3 should still be there
	_, err = cache.Get(ctx, "key2")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	_, err = cache.Get(ctx, "key3")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
}

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	if config == nil {
		t.Fatal("DefaultCacheConfig returned nil")
	}

	if config.L1MaxSize <= 0 {
		t.Error("L1MaxSize should be > 0")
	}

	if !config.EnableL1 {
		t.Error("L1 should be enabled by default")
	}
}

func TestStateCacheImpl_Close(t *testing.T) {
	cache := NewStateCache(nil)

	err := cache.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
