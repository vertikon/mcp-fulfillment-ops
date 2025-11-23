package store

import (
	"context"
	"testing"
	"time"
)

func TestNewInMemoryDistributedStore(t *testing.T) {
	config := DefaultStoreConfig()
	store := NewInMemoryDistributedStore(config)

	if store == nil {
		t.Fatal("NewInMemoryDistributedStore returned nil")
	}

	if store.data == nil {
		t.Error("data map should not be nil")
	}

	if store.locks == nil {
		t.Error("locks map should not be nil")
	}
}

func TestInMemoryDistributedStore_Get_Set(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Set a value
	state, err := store.Set(ctx, "test-key", "test-value", nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if state == nil {
		t.Fatal("Set() returned nil state")
	}

	if state.Version != 1 {
		t.Errorf("Expected version 1, got %d", state.Version)
	}

	// Get the value
	retrieved, err := store.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Value != "test-value" {
		t.Errorf("Expected 'test-value', got '%v'", retrieved.Value)
	}

	if retrieved.Version != 1 {
		t.Errorf("Expected version 1, got %d", retrieved.Version)
	}
}

func TestInMemoryDistributedStore_CompareAndSet(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Set initial value
	_, err := store.Set(ctx, "cas-key", "initial", nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// CAS with correct version
	state, err := store.CompareAndSet(ctx, "cas-key", 1, "updated", nil)
	if err != nil {
		t.Fatalf("CompareAndSet() error = %v", err)
	}

	if state.Version != 2 {
		t.Errorf("Expected version 2, got %d", state.Version)
	}

	if state.Value != "updated" {
		t.Errorf("Expected 'updated', got '%v'", state.Value)
	}

	// CAS with incorrect version (should fail)
	_, err = store.CompareAndSet(ctx, "cas-key", 1, "should-fail", nil)
	if err == nil {
		t.Error("CompareAndSet() should fail with incorrect version")
	}
}

func TestInMemoryDistributedStore_Delete(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Set a value
	_, err := store.Set(ctx, "delete-key", "value", nil)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete the value
	err = store.Delete(ctx, "delete-key")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Try to get deleted value (should fail)
	_, err = store.Get(ctx, "delete-key")
	if err == nil {
		t.Error("Get() should fail for deleted key")
	}
}

func TestInMemoryDistributedStore_AcquireLock_ReleaseLock(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Acquire lock
	acquired, err := store.AcquireLock(ctx, "lock-key", 30)
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}

	if !acquired {
		t.Error("AcquireLock() should return true")
	}

	// Try to acquire same lock again (should fail)
	acquired2, err := store.AcquireLock(ctx, "lock-key", 30)
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}

	if acquired2 {
		t.Error("AcquireLock() should return false for locked key")
	}

	// Release lock
	err = store.ReleaseLock(ctx, "lock-key")
	if err != nil {
		t.Fatalf("ReleaseLock() error = %v", err)
	}

	// Acquire lock again (should succeed)
	acquired3, err := store.AcquireLock(ctx, "lock-key", 30)
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}

	if !acquired3 {
		t.Error("AcquireLock() should return true after release")
	}
}

func TestInMemoryDistributedStore_Snapshot(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Set some values
	_, _ = store.Set(ctx, "key1", "value1", nil)
	_, _ = store.Set(ctx, "key2", "value2", nil)

	// Create snapshot
	err := store.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	// Verify stats
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.SnapshotOps == 0 {
		t.Error("SnapshotOps should be > 0")
	}
}

func TestInMemoryDistributedStore_TTL(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Set value with TTL
	ttl := time.Now().Add(100 * time.Millisecond)
	_, err := store.Set(ctx, "ttl-key", "value", &ttl)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get value immediately (should succeed)
	_, err = store.Get(ctx, "ttl-key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Get value after TTL (should fail)
	_, err = store.Get(ctx, "ttl-key")
	if err == nil {
		t.Error("Get() should fail for expired key")
	}
}

func TestInMemoryDistributedStore_Health(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	health, err := store.Health(ctx)
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.NodeID == "" {
		t.Error("NodeID should not be empty")
	}
}

func TestInMemoryDistributedStore_Stats(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)
	ctx := context.Background()

	// Perform some operations
	_, _ = store.Set(ctx, "key1", "value1", nil)
	_, _ = store.Set(ctx, "key2", "value2", nil)
	_, _ = store.Get(ctx, "key1")

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.TotalKeys != 2 {
		t.Errorf("Expected 2 total keys, got %d", stats.TotalKeys)
	}

	if stats.WriteOps < 2 {
		t.Errorf("Expected >= 2 write ops, got %d", stats.WriteOps)
	}

	if stats.ReadOps < 1 {
		t.Errorf("Expected >= 1 read ops, got %d", stats.ReadOps)
	}
}

func TestInMemoryDistributedStore_Close(t *testing.T) {
	store := NewInMemoryDistributedStore(nil)

	err := store.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestDefaultStoreConfig(t *testing.T) {
	config := DefaultStoreConfig()

	if config == nil {
		t.Fatal("DefaultStoreConfig returned nil")
	}

	if config.NodeID == "" {
		t.Error("NodeID should not be empty")
	}

	if config.SnapshotInterval <= 0 {
		t.Error("SnapshotInterval should be > 0")
	}
}
