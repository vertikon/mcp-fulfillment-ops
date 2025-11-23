package store

import (
	"context"
	"testing"
)

func TestNewStateSync(t *testing.T) {
	config := DefaultSyncConfig()
	store := NewInMemoryDistributedStore(nil)
	sync := NewStateSync(config, store)

	if sync == nil {
		t.Fatal("NewStateSync returned nil")
	}

	if sync.config == nil {
		t.Error("config should not be nil")
	}
}

func TestStateSyncImpl_SyncWithPeer(t *testing.T) {
	config := DefaultSyncConfig()
	config.Peers = []string{"peer1", "peer2"}
	store := NewInMemoryDistributedStore(nil)
	sync := NewStateSync(config, store)
	ctx := context.Background()

	err := sync.SyncWithPeer(ctx, "peer1")
	if err != nil {
		t.Fatalf("SyncWithPeer() error = %v", err)
	}

	status, err := sync.GetSyncStatus(ctx)
	if err != nil {
		t.Fatalf("GetSyncStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetSyncStatus() returned nil")
	}
}

func TestStateSyncImpl_BroadcastUpdate(t *testing.T) {
	config := DefaultSyncConfig()
	config.Peers = []string{"peer1", "peer2"}
	store := NewInMemoryDistributedStore(nil)
	sync := NewStateSync(config, store)
	ctx := context.Background()

	state := &VersionedState{
		Key:     "test-key",
		Value:   "test-value",
		Version: 1,
	}

	err := sync.BroadcastUpdate(ctx, state)
	if err != nil {
		t.Fatalf("BroadcastUpdate() error = %v", err)
	}
}

func TestStateSyncImpl_SubscribeToUpdates(t *testing.T) {
	config := DefaultSyncConfig()
	store := NewInMemoryDistributedStore(nil)
	sync := NewStateSync(config, store)
	ctx := context.Background()

	ch, err := sync.SubscribeToUpdates(ctx)
	if err != nil {
		t.Fatalf("SubscribeToUpdates() error = %v", err)
	}

	if ch == nil {
		t.Fatal("SubscribeToUpdates() returned nil channel")
	}
}

func TestStateSyncImpl_GetSyncStatus(t *testing.T) {
	config := DefaultSyncConfig()
	config.Peers = []string{"peer1"}
	store := NewInMemoryDistributedStore(nil)
	sync := NewStateSync(config, store)
	ctx := context.Background()

	status, err := sync.GetSyncStatus(ctx)
	if err != nil {
		t.Fatalf("GetSyncStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetSyncStatus() returned nil")
	}

	if status.NodeID == "" {
		t.Error("NodeID should not be empty")
	}
}

func TestStateSyncImpl_Stop(t *testing.T) {
	config := DefaultSyncConfig()
	store := NewInMemoryDistributedStore(nil)
	sync := NewStateSync(config, store)

	sync.Stop()
	// Should not panic
}

func TestDefaultSyncConfig(t *testing.T) {
	config := DefaultSyncConfig()

	if config == nil {
		t.Fatal("DefaultSyncConfig returned nil")
	}

	if config.NodeID == "" {
		t.Error("NodeID should not be empty")
	}

	if config.SyncInterval <= 0 {
		t.Error("SyncInterval should be > 0")
	}
}
