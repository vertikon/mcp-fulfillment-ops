package memory

import (
	"context"
	"testing"
	"time"
)

func TestNewEpisodicMemoryManager(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	if manager == nil {
		t.Fatal("NewEpisodicMemoryManager returned nil")
	}
	if manager.store != store {
		t.Error("store not set correctly")
	}
}

func TestEpisodicMemoryManager_Create(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	memory, err := manager.Create(ctx, "test content", "session1")

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if memory == nil {
		t.Fatal("Memory is nil")
	}
	if memory.Content() != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", memory.Content())
	}
	if memory.SessionID() != "session1" {
		t.Errorf("Expected session ID 'session1', got '%s'", memory.SessionID())
	}
}

func TestEpisodicMemoryManager_AddEvent(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	_, _ = manager.Create(ctx, "initial content", "session1")

	err := manager.AddEvent(ctx, "session1", "user_message", "Hello")
	if err != nil {
		t.Fatalf("AddEvent failed: %v", err)
	}

	events, err := manager.GetEvents(ctx, "session1")
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected events, got empty")
	}
}

func TestEpisodicMemoryManager_GetEvents(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	_, _ = manager.Create(ctx, "content", "session1")
	manager.AddEvent(ctx, "session1", "event1", "data1")
	manager.AddEvent(ctx, "session1", "event2", "data2")

	events, err := manager.GetEvents(ctx, "session1")
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	if len(events) < 2 {
		t.Errorf("Expected at least 2 events, got %d", len(events))
	}
}

func TestEpisodicMemoryManager_GetRecentEvents(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	_, _ = manager.Create(ctx, "content", "session1")
	manager.AddEvent(ctx, "session1", "event1", "data1")

	window := 1 * time.Hour
	events, err := manager.GetRecentEvents(ctx, "session1", window)
	if err != nil {
		t.Fatalf("GetRecentEvents failed: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected recent events, got empty")
	}
}

func TestEpisodicMemoryManager_Consolidate(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	memory, _ := manager.Create(ctx, "content", "session1")

	// Set importance high enough for consolidation
	if err := memory.SetImportance(0.8); err != nil {
		t.Fatalf("Failed to set importance: %v", err)
	}

	threshold := 0 * time.Hour // Immediate consolidation
	consolidated, err := manager.Consolidate(ctx, "session1", threshold)
	if err != nil {
		t.Fatalf("Consolidate failed: %v", err)
	}

	if len(consolidated) == 0 {
		t.Error("Expected consolidated memories, got empty")
	}
}

func TestEpisodicMemoryManager_Clear(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	_, _ = manager.Create(ctx, "content", "session1")

	err := manager.Clear(ctx, "session1")
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	events, _ := manager.GetEvents(ctx, "session1")
	if len(events) > 0 {
		t.Error("Expected events to be cleared")
	}
}

func TestEpisodicMemoryManager_GetByImportance(t *testing.T) {
	store := NewMemoryStore(newMockMemoryRepository(), newMockCacheClient(), 24*time.Hour)
	manager := NewEpisodicMemoryManager(store)

	ctx := context.Background()
	memory1, _ := manager.Create(ctx, "content1", "session1")
	memory1.SetImportance(0.9)

	memory2, _ := manager.Create(ctx, "content2", "session1")
	memory2.SetImportance(0.5)

	memories, err := manager.GetByImportance(ctx, "session1", 0.7)
	if err != nil {
		t.Fatalf("GetByImportance failed: %v", err)
	}

	if len(memories) == 0 {
		t.Error("Expected memories with high importance, got empty")
	}

	for _, mem := range memories {
		if mem.Importance() < 0.7 {
			t.Errorf("Expected importance >= 0.7, got %f", mem.Importance())
		}
	}
}
