package store

import (
	"context"
	"testing"
	"time"
)

func TestNewConflictResolver(t *testing.T) {
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(config)

	if resolver == nil {
		t.Fatal("NewConflictResolver returned nil")
	}

	if resolver.config == nil {
		t.Error("config should not be nil")
	}
}

func TestConflictResolverImpl_Resolve_LastWriteWins(t *testing.T) {
	config := DefaultConflictResolverConfig()
	config.DefaultStrategy = LastWriteWins
	resolver := NewConflictResolver(config)
	ctx := context.Background()

	localState := &VersionedState{
		Key:     "test-key",
		Value:   "local-value",
		Version: 1,
		Meta: map[string]interface{}{
			"timestamp": time.Now().Add(-1 * time.Hour),
		},
	}

	remoteState := &VersionedState{
		Key:     "test-key",
		Value:   "remote-value",
		Version: 2,
		Meta: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}

	conflict := &Conflict{
		Key:         "test-key",
		LocalState:  localState,
		RemoteState: remoteState,
		Strategy:    LastWriteWins,
		Timestamp:   time.Now(),
	}

	resolved, err := resolver.Resolve(ctx, conflict)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("Resolve() returned nil")
	}

	// Last write wins should prefer remote (newer timestamp)
	if resolved.Value != "remote-value" {
		t.Errorf("Expected 'remote-value', got '%v'", resolved.Value)
	}
}

func TestConflictResolverImpl_Resolve_FirstWriteWins(t *testing.T) {
	config := DefaultConflictResolverConfig()
	config.DefaultStrategy = FirstWriteWins
	resolver := NewConflictResolver(config)
	ctx := context.Background()

	localState := &VersionedState{
		Key:     "test-key",
		Value:   "local-value",
		Version: 1,
		Meta: map[string]interface{}{
			"timestamp": time.Now().Add(-1 * time.Hour),
		},
	}

	remoteState := &VersionedState{
		Key:     "test-key",
		Value:   "remote-value",
		Version: 2,
		Meta: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}

	conflict := &Conflict{
		Key:         "test-key",
		LocalState:  localState,
		RemoteState: remoteState,
		Strategy:    FirstWriteWins,
		Timestamp:   time.Now(),
	}

	resolved, err := resolver.Resolve(ctx, conflict)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved == nil {
		t.Fatal("Resolve() returned nil")
	}

	// First write wins should prefer local (older timestamp)
	if resolved.Value != "local-value" {
		t.Errorf("Expected 'local-value', got '%v'", resolved.Value)
	}
}

func TestConflictResolverImpl_GetStrategy(t *testing.T) {
	config := DefaultConflictResolverConfig()
	config.DefaultStrategy = VectorClock
	resolver := NewConflictResolver(config)

	strategy := resolver.GetStrategy()
	if strategy != VectorClock {
		t.Errorf("Expected VectorClock, got %s", strategy)
	}
}

func TestConflictResolverImpl_SetStrategy(t *testing.T) {
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(config)

	err := resolver.SetStrategy(CRDTMerge)
	if err != nil {
		t.Fatalf("SetStrategy() error = %v", err)
	}

	strategy := resolver.GetStrategy()
	if strategy != CRDTMerge {
		t.Errorf("Expected CRDTMerge, got %s", strategy)
	}
}

func TestConflictResolverImpl_GetConflictStats(t *testing.T) {
	config := DefaultConflictResolverConfig()
	resolver := NewConflictResolver(config)
	ctx := context.Background()

	// Resolve a conflict to generate stats
	localState := &VersionedState{Key: "key", Value: "local", Version: 1}
	remoteState := &VersionedState{Key: "key", Value: "remote", Version: 2}
	conflict := &Conflict{
		Key:         "key",
		LocalState:  localState,
		RemoteState: remoteState,
		Strategy:    LastWriteWins,
		Timestamp:   time.Now(),
	}

	_, _ = resolver.Resolve(ctx, conflict)

	stats := resolver.GetConflictStats()
	if stats == nil {
		t.Fatal("GetConflictStats() returned nil")
	}

	if stats.TotalConflicts == 0 {
		t.Error("TotalConflicts should be > 0")
	}
}

func TestDefaultConflictResolverConfig(t *testing.T) {
	config := DefaultConflictResolverConfig()

	if config == nil {
		t.Fatal("DefaultConflictResolverConfig returned nil")
	}

	if config.DefaultStrategy == "" {
		t.Error("DefaultStrategy should not be empty")
	}

	if config.NodeID == "" {
		t.Error("NodeID should not be empty")
	}
}
