package store

import (
	"context"
	"os"
	"testing"
)

func TestNewSnapshotManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)

	manager := NewSnapshotManager(config, store)
	if manager == nil {
		t.Fatal("NewSnapshotManager returned nil")
	}
}

func TestSnapshotManagerImpl_CreateSnapshot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)
	manager := NewSnapshotManager(config, store)
	ctx := context.Background()

	// Set some data
	_, _ = store.Set(ctx, "key1", "value1", nil)
	_, _ = store.Set(ctx, "key2", "value2", nil)

	// Create snapshot
	info, err := manager.CreateSnapshot(ctx, FullSnapshot, "")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	if info == nil {
		t.Fatal("CreateSnapshot() returned nil")
	}

	if info.Type != FullSnapshot {
		t.Errorf("Expected FullSnapshot, got %s", info.Type)
	}

	if info.KeysCount != 2 {
		t.Errorf("Expected 2 keys, got %d", info.KeysCount)
	}
}

func TestSnapshotManagerImpl_RestoreSnapshot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)
	manager := NewSnapshotManager(config, store)
	ctx := context.Background()

	// Set some data and create snapshot
	_, _ = store.Set(ctx, "key1", "value1", nil)
	info, err := manager.CreateSnapshot(ctx, FullSnapshot, "")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	// Clear store
	_ = store.Delete(ctx, "key1")

	// Restore snapshot
	err = manager.RestoreSnapshot(ctx, info.ID)
	if err != nil {
		t.Fatalf("RestoreSnapshot() error = %v", err)
	}

	// Verify data restored
	state, err := store.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if state.Value != "value1" {
		t.Errorf("Expected 'value1', got '%v'", state.Value)
	}
}

func TestSnapshotManagerImpl_ListSnapshots(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)
	manager := NewSnapshotManager(config, store)
	ctx := context.Background()

	// Create multiple snapshots
	_, _ = manager.CreateSnapshot(ctx, FullSnapshot, "")
	_, _ = manager.CreateSnapshot(ctx, FullSnapshot, "")

	// List snapshots
	snapshots, err := manager.ListSnapshots(ctx)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}

	if len(snapshots) < 2 {
		t.Errorf("Expected >= 2 snapshots, got %d", len(snapshots))
	}
}

func TestSnapshotManagerImpl_GetSnapshotInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)
	manager := NewSnapshotManager(config, store)
	ctx := context.Background()

	// Create snapshot
	created, err := manager.CreateSnapshot(ctx, FullSnapshot, "")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	// Get snapshot info
	info, err := manager.GetSnapshotInfo(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSnapshotInfo() error = %v", err)
	}

	if info == nil {
		t.Fatal("GetSnapshotInfo() returned nil")
	}

	if info.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, info.ID)
	}
}

func TestSnapshotManagerImpl_IncrementalSnapshot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)
	manager := NewSnapshotManager(config, store)
	ctx := context.Background()

	// Create base snapshot
	base, err := manager.CreateSnapshot(ctx, FullSnapshot, "")
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}

	// Add more data
	_, _ = store.Set(ctx, "key2", "value2", nil)

	// Create incremental snapshot
	inc, err := manager.IncrementalSnapshot(ctx, base.ID)
	if err != nil {
		t.Fatalf("IncrementalSnapshot() error = %v", err)
	}

	if inc == nil {
		t.Fatal("IncrementalSnapshot() returned nil")
	}

	if inc.Type != IncrementalSnapshot {
		t.Errorf("Expected IncrementalSnapshot, got %s", inc.Type)
	}

	if inc.BaseSnapshotID != base.ID {
		t.Errorf("Expected base snapshot ID %s, got %s", base.ID, inc.BaseSnapshotID)
	}
}

func TestSnapshotManagerImpl_GetSnapshotStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "snapshot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := DefaultSnapshotConfig()
	config.SnapshotPath = tmpDir
	store := NewInMemoryDistributedStore(nil)
	manager := NewSnapshotManager(config, store)
	ctx := context.Background()

	// Create some snapshots
	_, _ = manager.CreateSnapshot(ctx, FullSnapshot, "")
	_, _ = manager.CreateSnapshot(ctx, IncrementalSnapshot, "")

	stats := manager.GetSnapshotStats()
	if stats == nil {
		t.Fatal("GetSnapshotStats() returned nil")
	}

	if stats.TotalSnapshots < 2 {
		t.Errorf("Expected >= 2 snapshots, got %d", stats.TotalSnapshots)
	}
}

func TestDefaultSnapshotConfig(t *testing.T) {
	config := DefaultSnapshotConfig()

	if config == nil {
		t.Fatal("DefaultSnapshotConfig returned nil")
	}

	if config.SnapshotPath == "" {
		t.Error("SnapshotPath should not be empty")
	}
}
