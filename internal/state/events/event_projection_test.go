package events

import (
	"context"
	"testing"
	"time"
)

// MockProjectionHandler for testing
type MockProjectionHandler struct {
	projectedEvents []*Event
}

func (h *MockProjectionHandler) Project(ctx context.Context, event *Event, projection *Projection) (interface{}, error) {
	h.projectedEvents = append(h.projectedEvents, event)
	return map[string]interface{}{"processed": true}, nil
}

func (h *MockProjectionHandler) CanHandle(event *Event) bool {
	return true
}

func (h *MockProjectionHandler) GetHandlerType() string {
	return "mock"
}

func TestNewEventProjection(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)

	if projection == nil {
		t.Fatal("NewEventProjection returned nil")
	}
}

func TestEventProjectionImpl_CreateProjection(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)
	ctx := context.Background()

	handler := &MockProjectionHandler{}
	proj := &Projection{
		ID:            "proj-1",
		Type:          ProjectionTypeState,
		Name:          "test-projection",
		AggregateID:   "agg-1",
		AggregateType: "test",
		Handler:       handler,
		IsActive:      true,
		CreatedAt:     time.Now(),
	}

	err := projection.CreateProjection(ctx, proj)
	if err != nil {
		t.Fatalf("CreateProjection() error = %v", err)
	}

	// Get projection
	retrieved, err := projection.GetProjection(ctx, "proj-1")
	if err != nil {
		t.Fatalf("GetProjection() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetProjection() returned nil")
	}

	if retrieved.Name != "test-projection" {
		t.Errorf("Expected name 'test-projection', got '%s'", retrieved.Name)
	}
}

func TestEventProjectionImpl_ProcessEvent(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)
	ctx := context.Background()

	handler := &MockProjectionHandler{}
	proj := &Projection{
		ID:        "proj-1",
		Type:      ProjectionTypeState,
		Name:      "test-projection",
		Handler:   handler,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	_ = projection.CreateProjection(ctx, proj)

	// Process event
	event := &Event{
		ID:            "event-1",
		Type:          EventTypeCreate,
		AggregateID:   "agg-1",
		AggregateType: "test",
		Version:       1,
		Timestamp:     time.Now(),
		NodeID:        "node-1",
	}

	err := projection.ProcessEvent(ctx, event)
	if err != nil {
		t.Fatalf("ProcessEvent() error = %v", err)
	}

	if len(handler.projectedEvents) != 1 {
		t.Errorf("Expected 1 projected event, got %d", len(handler.projectedEvents))
	}
}

func TestEventProjectionImpl_RebuildProjection(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)
	ctx := context.Background()

	// Save events
	events := []*Event{
		{ID: "e1", Type: EventTypeCreate, AggregateID: "agg-1", AggregateType: "test", Version: 1, Timestamp: time.Now(), NodeID: "node-1"},
		{ID: "e2", Type: EventTypeUpdate, AggregateID: "agg-1", AggregateType: "test", Version: 2, Timestamp: time.Now(), NodeID: "node-1"},
	}
	_ = store.SaveEvents(ctx, events)

	handler := &MockProjectionHandler{}
	proj := &Projection{
		ID:          "proj-1",
		Type:        ProjectionTypeState,
		Name:        "test-projection",
		AggregateID: "agg-1",
		Handler:     handler,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
	_ = projection.CreateProjection(ctx, proj)

	// Rebuild projection
	err := projection.RebuildProjection(ctx, "proj-1")
	if err != nil {
		t.Fatalf("RebuildProjection() error = %v", err)
	}

	// Verify events were processed
	if len(handler.projectedEvents) < 2 {
		t.Errorf("Expected >= 2 projected events, got %d", len(handler.projectedEvents))
	}
}

func TestEventProjectionImpl_GetProjectionState(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)
	ctx := context.Background()

	handler := &MockProjectionHandler{}
	proj := &Projection{
		ID:        "proj-1",
		Type:      ProjectionTypeState,
		Name:      "test-projection",
		Handler:   handler,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	_ = projection.CreateProjection(ctx, proj)

	// Process event
	event := &Event{
		ID:            "event-1",
		Type:          EventTypeCreate,
		AggregateID:   "agg-1",
		AggregateType: "test",
		Version:       1,
		Timestamp:     time.Now(),
		NodeID:        "node-1",
	}
	_ = projection.ProcessEvent(ctx, event)

	// Get projection state
	state, err := projection.GetProjectionState(ctx, "proj-1")
	if err != nil {
		t.Fatalf("GetProjectionState() error = %v", err)
	}

	if state == nil {
		t.Fatal("GetProjectionState() returned nil")
	}

	if state.EventsProcessed == 0 {
		t.Error("EventsProcessed should be > 0")
	}
}

func TestEventProjectionImpl_ListProjections(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)
	ctx := context.Background()

	handler := &MockProjectionHandler{}

	// Create multiple projections
	proj1 := &Projection{ID: "proj-1", Type: ProjectionTypeState, Name: "proj1", Handler: handler, IsActive: true, CreatedAt: time.Now()}
	proj2 := &Projection{ID: "proj-2", Type: ProjectionTypeAggregation, Name: "proj2", Handler: handler, IsActive: true, CreatedAt: time.Now()}

	_ = projection.CreateProjection(ctx, proj1)
	_ = projection.CreateProjection(ctx, proj2)

	// List projections
	projections, err := projection.ListProjections(ctx, nil)
	if err != nil {
		t.Fatalf("ListProjections() error = %v", err)
	}

	if len(projections) < 2 {
		t.Errorf("Expected >= 2 projections, got %d", len(projections))
	}
}

func TestEventProjectionImpl_GetProjectionStats(t *testing.T) {
	store := NewInMemoryEventStore(nil)
	projection := NewEventProjection(nil, store)
	ctx := context.Background()

	handler := &MockProjectionHandler{}
	proj := &Projection{
		ID:        "proj-1",
		Type:      ProjectionTypeState,
		Name:      "test-projection",
		Handler:   handler,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	_ = projection.CreateProjection(ctx, proj)

	// Process some events
	events := []*Event{
		{ID: "e1", Type: EventTypeCreate, AggregateID: "agg-1", AggregateType: "test", Version: 1, Timestamp: time.Now(), NodeID: "node-1"},
		{ID: "e2", Type: EventTypeUpdate, AggregateID: "agg-1", AggregateType: "test", Version: 2, Timestamp: time.Now(), NodeID: "node-1"},
	}
	_ = projection.ProcessEvents(ctx, events)

	// Get stats
	stats, err := projection.GetProjectionStats(ctx)
	if err != nil {
		t.Fatalf("GetProjectionStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetProjectionStats() returned nil")
	}

	if stats.TotalProjections == 0 {
		t.Error("TotalProjections should be > 0")
	}
}
