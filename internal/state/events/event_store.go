package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeCreate   EventType = "create"
	EventTypeUpdate   EventType = "update"
	EventTypeDelete   EventType = "delete"
	EventTypeSnapshot EventType = "snapshot"
	EventTypeRestore  EventType = "restore"
	EventTypeCustom   EventType = "custom"
)

// Event represents a domain event
type Event struct {
	ID            string                 `json:"id"`
	Type          EventType              `json:"type"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int64                  `json:"version"`
	Data          interface{}            `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
	Timestamp     time.Time              `json:"timestamp"`
	NodeID        string                 `json:"node_id"`
	CausationID   string                 `json:"causation_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// EventStore interface for storing and retrieving events
type EventStore interface {
	// Event operations
	SaveEvent(ctx context.Context, event *Event) error
	SaveEvents(ctx context.Context, events []*Event) error
	GetEvents(ctx context.Context, aggregateID string, fromVersion int64, toVersion int64) ([]*Event, error)
	GetAllEvents(ctx context.Context, aggregateID string) ([]*Event, error)
	GetEventsByType(ctx context.Context, eventType EventType, limit int) ([]*Event, error)
	GetEventsByTimeRange(ctx context.Context, startTime, endTime time.Time, limit int) ([]*Event, error)

	// Stream operations
	StreamEvents(ctx context.Context, aggregateID string, fromVersion int64) (<-chan *Event, error)
	StreamAllEvents(ctx context.Context, fromTime time.Time) (<-chan *Event, error)

	// Metadata and statistics
	GetAggregateInfo(ctx context.Context, aggregateID string) (*AggregateInfo, error)
	GetEventStats(ctx context.Context) (*EventStoreStats, error)
	GetStoreInfo(ctx context.Context) (*EventStoreInfo, error)

	// Snapshot operations
	CreateSnapshot(ctx context.Context, aggregateID string, version int64, snapshotData interface{}) error
	GetSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error)

	// Health and maintenance
	Health(ctx context.Context) (EventStoreHealth, error)
	CompactEvents(ctx context.Context, aggregateID string, targetVersion int64) error
	PruneEvents(ctx context.Context, beforeTime time.Time) error
}

// AggregateInfo represents information about an aggregate
type AggregateInfo struct {
	AggregateID   string     `json:"aggregate_id"`
	AggregateType string     `json:"aggregate_type"`
	Version       int64      `json:"version"`
	EventCount    int64      `json:"event_count"`
	FirstEvent    *time.Time `json:"first_event,omitempty"`
	LastEvent     *time.Time `json:"last_event,omitempty"`
	LastSnapshot  *time.Time `json:"last_snapshot,omitempty"`
	Size          int64      `json:"size"`
}

// EventStoreStats represents event store statistics
type EventStoreStats struct {
	TotalEvents      int64            `json:"total_events"`
	TotalAggregates  int64            `json:"total_aggregates"`
	EventsByType     map[string]int64 `json:"events_by_type"`
	StoreSize        int64            `json:"store_size"`
	WriteOperations  int64            `json:"write_operations"`
	ReadOperations   int64            `json:"read_operations"`
	StreamOperations int64            `json:"stream_operations"`
	SnapshotCount    int64            `json:"snapshot_count"`
	LastEvent        *time.Time       `json:"last_event"`
	AverageEventSize float64          `json:"average_event_size"`
	CompactionStats  *CompactionStats `json:"compaction_stats"`
}

// CompactionStats represents event compaction statistics
type CompactionStats struct {
	LastCompaction   *time.Time `json:"last_compaction,omitempty"`
	CompactionsCount int64      `json:"compactions_count"`
	EventsCompacted  int64      `json:"events_compacted"`
	SpaceReclaimed   int64      `json:"space_reclaimed"`
}

// EventStoreInfo represents information about the event store
type EventStoreInfo struct {
	StoreType         string                 `json:"store_type"`
	Version           string                 `json:"version"`
	NodeID            string                 `json:"node_id"`
	StartTime         time.Time              `json:"start_time"`
	SupportedFeatures []string               `json:"supported_features"`
	Configuration     map[string]interface{} `json:"configuration"`
}

// EventStoreHealth represents health status of event store
type EventStoreHealth struct {
	Status     string                 `json:"status"`
	StoreType  string                 `json:"store_type"`
	NodeID     string                 `json:"node_id"`
	Timestamp  time.Time              `json:"timestamp"`
	EventCount int64                  `json:"event_count"`
	StoreSize  int64                  `json:"store_size"`
	WriteLag   *time.Duration         `json:"write_lag,omitempty"`
	ReadLag    *time.Duration         `json:"read_lag,omitempty"`
	LastError  *string                `json:"last_error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Snapshot represents an aggregate snapshot
type Snapshot struct {
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	Version       int64                  `json:"version"`
	Data          interface{}            `json:"data"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Metadata      map[string]interface{} `json:"metadata"`
	Size          int64                  `json:"size"`
}

// EventStoreConfig represents event store configuration
type EventStoreConfig struct {
	StoreType           string        `json:"store_type"`
	StoragePath         string        `json:"storage_path"`
	MaxEventsPerFile    int64         `json:"max_events_per_file"`
	CompactionThreshold int64         `json:"compaction_threshold"`
	SnapshotInterval    time.Duration `json:"snapshot_interval"`
	SnapshotRetention   int           `json:"snapshot_retention"`
	CompressionEnabled  bool          `json:"compression_enabled"`
	EncryptionEnabled   bool          `json:"encryption_enabled"`
	NodeID              string        `json:"node_id"`
	MaxEventSize        int64         `json:"max_event_size"`
	EventTTL            time.Duration `json:"event_ttl"`
	StreamBufferSize    int           `json:"stream_buffer_size"`
}

// DefaultEventStoreConfig returns default event store configuration
func DefaultEventStoreConfig() *EventStoreConfig {
	return &EventStoreConfig{
		StoreType:           "in-memory",
		StoragePath:         "./data/events",
		MaxEventsPerFile:    10000,
		CompactionThreshold: 5000,
		SnapshotInterval:    time.Hour,
		SnapshotRetention:   10,
		CompressionEnabled:  true,
		EncryptionEnabled:   false,
		NodeID:              fmt.Sprintf("eventstore-%d", time.Now().Unix()),
		MaxEventSize:        1024 * 1024,         // 1MB
		EventTTL:            time.Hour * 24 * 30, // 30 days
		StreamBufferSize:    10000,               // Aumentado de 1000 para 10000
	}
}

// InMemoryEventStore implements EventStore interface in memory
type InMemoryEventStore struct {
	config      *EventStoreConfig
	events      map[string][]*Event // aggregateID -> events
	snapshots   map[string]*Snapshot
	metadata    map[string]*AggregateInfo
	mu          sync.RWMutex
	logger      *zap.Logger
	stats       *EventStoreStats
	ctx         context.Context
	cancel      context.CancelFunc
	streamCh    chan *Event
	subscribers map[string][]chan *Event // eventType -> subscribers
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(config *EventStoreConfig) *InMemoryEventStore {
	if config == nil {
		config = DefaultEventStoreConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	store := &InMemoryEventStore{
		config:      config,
		events:      make(map[string][]*Event),
		snapshots:   make(map[string]*Snapshot),
		metadata:    make(map[string]*AggregateInfo),
		streamCh:    make(chan *Event, config.StreamBufferSize*2), // Buffer maior
		subscribers: make(map[string][]chan *Event),
		stats: &EventStoreStats{
			EventsByType:    make(map[string]int64),
			CompactionStats: &CompactionStats{},
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger.Get(),
	}

	// Start background processes
	go store.backgroundProcesses()

	store.logger.Info("In-memory event store initialized",
		zap.String("node_id", config.NodeID),
		zap.String("store_type", config.StoreType),
		zap.String("storage_path", config.StoragePath))

	return store
}

// SaveEvent saves a single event
func (es *InMemoryEventStore) SaveEvent(ctx context.Context, event *Event) error {
	return es.SaveEvents(ctx, []*Event{event})
}

// SaveEvents saves multiple events atomically
func (es *InMemoryEventStore) SaveEvents(ctx context.Context, events []*Event) error {
	if len(events) == 0 {
		return nil
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	// Validate events
	for _, event := range events {
		if err := es.validateEvent(event); err != nil {
			return fmt.Errorf("event validation failed: %w", err)
		}
	}

	// Save events
	for _, event := range events {
		// Add to aggregate events
		aggregateEvents := es.events[event.AggregateID]
		if aggregateEvents == nil {
			aggregateEvents = []*Event{}
		}

		// Check version continuity
		if len(aggregateEvents) > 0 {
			lastEvent := aggregateEvents[len(aggregateEvents)-1]
			if event.Version != lastEvent.Version+1 {
				return fmt.Errorf("version gap detected for aggregate %s: expected %d, got %d",
					event.AggregateID, lastEvent.Version+1, event.Version)
			}
		} else {
			// First event should have version 1
			if event.Version != 1 {
				return fmt.Errorf("first event version should be 1 for aggregate %s, got %d",
					event.AggregateID, event.Version)
			}
		}

		aggregateEvents = append(aggregateEvents, event)
		es.events[event.AggregateID] = aggregateEvents

		// Update aggregate metadata
		es.updateAggregateMetadata(event)

		// Update statistics
		es.updateEventStats(event)

		// Stream to subscribers
		select {
		case es.streamCh <- event:
		default:
			es.logger.Warn("Event stream channel full, dropping event",
				zap.String("event_id", event.ID))
		}

		es.logger.Debug("Event saved",
			zap.String("event_id", event.ID),
			zap.String("aggregate_id", event.AggregateID),
			zap.String("type", string(event.Type)),
			zap.Int64("version", event.Version))
	}

	es.stats.WriteOperations++
	return nil
}

// GetEvents retrieves events for an aggregate within a version range
func (es *InMemoryEventStore) GetEvents(ctx context.Context, aggregateID string, fromVersion int64, toVersion int64) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	aggregateEvents, exists := es.events[aggregateID]
	if !exists {
		return []*Event{}, nil
	}

	var result []*Event
	for _, event := range aggregateEvents {
		if event.Version >= fromVersion && event.Version <= toVersion {
			result = append(result, event)
		}
	}

	es.stats.ReadOperations++

	es.logger.Debug("Events retrieved",
		zap.String("aggregate_id", aggregateID),
		zap.Int64("from_version", fromVersion),
		zap.Int64("to_version", toVersion),
		zap.Int("count", len(result)))

	return result, nil
}

// GetAllEvents retrieves all events for an aggregate
func (es *InMemoryEventStore) GetAllEvents(ctx context.Context, aggregateID string) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	aggregateEvents, exists := es.events[aggregateID]
	if !exists {
		return []*Event{}, nil
	}

	// Return a copy to avoid external modification
	result := make([]*Event, len(aggregateEvents))
	copy(result, aggregateEvents)

	es.stats.ReadOperations++

	es.logger.Debug("All events retrieved",
		zap.String("aggregate_id", aggregateID),
		zap.Int("count", len(result)))

	return result, nil
}

// GetEventsByType retrieves events by type
func (es *InMemoryEventStore) GetEventsByType(ctx context.Context, eventType EventType, limit int) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var result []*Event
	count := 0

	for _, aggregateEvents := range es.events {
		for _, event := range aggregateEvents {
			if event.Type == eventType {
				result = append(result, event)
				count++
				if limit > 0 && count >= limit {
					goto done
				}
			}
		}
	}

done:
	es.stats.ReadOperations++

	es.logger.Debug("Events retrieved by type",
		zap.String("type", string(eventType)),
		zap.Int("limit", limit),
		zap.Int("count", len(result)))

	return result, nil
}

// GetEventsByTimeRange retrieves events within a time range
func (es *InMemoryEventStore) GetEventsByTimeRange(ctx context.Context, startTime, endTime time.Time, limit int) ([]*Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var result []*Event
	count := 0

	for _, aggregateEvents := range es.events {
		for _, event := range aggregateEvents {
			if event.Timestamp.After(startTime) && event.Timestamp.Before(endTime) {
				result = append(result, event)
				count++
				if limit > 0 && count >= limit {
					goto done
				}
			}
		}
	}

done:
	es.stats.ReadOperations++

	es.logger.Debug("Events retrieved by time range",
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
		zap.Int("limit", limit),
		zap.Int("count", len(result)))

	return result, nil
}

// StreamEvents streams events for an aggregate from a specific version
func (es *InMemoryEventStore) StreamEvents(ctx context.Context, aggregateID string, fromVersion int64) (<-chan *Event, error) {
	es.mu.RLock()
	aggregateEvents := es.events[aggregateID]
	es.mu.RUnlock()

	eventCh := make(chan *Event, es.config.StreamBufferSize)

	go func() {
		defer close(eventCh)

		if aggregateEvents != nil {
			for _, event := range aggregateEvents {
				if event.Version >= fromVersion {
					select {
					case eventCh <- event:
					case <-ctx.Done():
						return
					case <-es.ctx.Done():
						return
					}
				}
			}
		}

		// Continue streaming future events
		futureCh := make(chan *Event, es.config.StreamBufferSize)
		es.subscribeToAggregate(aggregateID, futureCh)

		for {
			select {
			case event := <-futureCh:
				select {
				case eventCh <- event:
				case <-ctx.Done():
					return
				case <-es.ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			case <-es.ctx.Done():
				return
			}
		}
	}()

	es.stats.StreamOperations++

	es.logger.Debug("Event stream started",
		zap.String("aggregate_id", aggregateID),
		zap.Int64("from_version", fromVersion))

	return eventCh, nil
}

// StreamAllEvents streams all events from a specific time
func (es *InMemoryEventStore) StreamAllEvents(ctx context.Context, fromTime time.Time) (<-chan *Event, error) {
	eventCh := make(chan *Event, es.config.StreamBufferSize)

	go func() {
		defer close(eventCh)

		// Stream existing events
		es.mu.RLock()
		for _, aggregateEvents := range es.events {
			for _, event := range aggregateEvents {
				if event.Timestamp.After(fromTime) || event.Timestamp.Equal(fromTime) {
					select {
					case eventCh <- event:
					case <-ctx.Done():
						es.mu.RUnlock()
						return
					case <-es.ctx.Done():
						es.mu.RUnlock()
						return
					}
				}
			}
		}
		es.mu.RUnlock()

		// Continue streaming future events
		for {
			select {
			case event := <-es.streamCh:
				if event.Timestamp.After(fromTime) || event.Timestamp.Equal(fromTime) {
					select {
					case eventCh <- event:
					case <-ctx.Done():
						return
					case <-es.ctx.Done():
						return
					}
				}
			case <-ctx.Done():
				return
			case <-es.ctx.Done():
				return
			}
		}
	}()

	es.stats.StreamOperations++

	es.logger.Debug("All events stream started",
		zap.Time("from_time", fromTime))

	return eventCh, nil
}

// GetAggregateInfo returns information about an aggregate
func (es *InMemoryEventStore) GetAggregateInfo(ctx context.Context, aggregateID string) (*AggregateInfo, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	info, exists := es.metadata[aggregateID]
	if !exists {
		return nil, fmt.Errorf("aggregate not found: %s", aggregateID)
	}

	// Return a copy
	copyInfo := *info

	return &copyInfo, nil
}

// GetEventStats returns event store statistics
func (es *InMemoryEventStore) GetEventStats(ctx context.Context) (*EventStoreStats, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	// Return a copy of stats
	stats := *es.stats
	stats.EventsByType = make(map[string]int64)
	for k, v := range es.stats.EventsByType {
		stats.EventsByType[k] = v
	}
	stats.CompactionStats = &CompactionStats{}
	*stats.CompactionStats = *es.stats.CompactionStats

	return &stats, nil
}

// GetStoreInfo returns information about the event store
func (es *InMemoryEventStore) GetStoreInfo(ctx context.Context) (*EventStoreInfo, error) {
	return &EventStoreInfo{
		StoreType: es.config.StoreType,
		Version:   "1.0.0",
		NodeID:    es.config.NodeID,
		StartTime: time.Now(), // In a real implementation, track actual start time
		SupportedFeatures: []string{
			"save_events",
			"get_events",
			"stream_events",
			"create_snapshots",
			"compaction",
		},
		Configuration: map[string]interface{}{
			"storage_path":        es.config.StoragePath,
			"compression_enabled": es.config.CompressionEnabled,
			"encryption_enabled":  es.config.EncryptionEnabled,
			"max_event_size":      es.config.MaxEventSize,
			"event_ttl":           es.config.EventTTL,
		},
	}, nil
}

// CreateSnapshot creates a snapshot for an aggregate
func (es *InMemoryEventStore) CreateSnapshot(ctx context.Context, aggregateID string, version int64, snapshotData interface{}) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Verify aggregate exists and version is valid
	aggregateEvents, exists := es.events[aggregateID]
	if !exists {
		return fmt.Errorf("aggregate not found: %s", aggregateID)
	}

	// Find the event at the specified version
	var targetEvent *Event
	for _, event := range aggregateEvents {
		if event.Version == version {
			targetEvent = event
			break
		}
	}

	if targetEvent == nil {
		return fmt.Errorf("version %d not found for aggregate %s", version, aggregateID)
	}

	// Create snapshot
	snapshot := &Snapshot{
		AggregateID:   aggregateID,
		AggregateType: targetEvent.AggregateType,
		Version:       version,
		Data:          snapshotData,
		CreatedAt:     time.Now(),
		CreatedBy:     es.config.NodeID,
		Metadata: map[string]interface{}{
			"created_at": time.Now(),
			"node_id":    es.config.NodeID,
		},
	}

	es.snapshots[aggregateID] = snapshot
	es.stats.SnapshotCount++

	es.logger.Info("Snapshot created",
		zap.String("aggregate_id", aggregateID),
		zap.Int64("version", version),
		zap.String("snapshot_id", snapshot.AggregateID))

	return nil
}

// GetSnapshot retrieves a snapshot for an aggregate
func (es *InMemoryEventStore) GetSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	snapshot, exists := es.snapshots[aggregateID]
	if !exists {
		return nil, fmt.Errorf("snapshot not found for aggregate: %s", aggregateID)
	}

	// Return a copy
	copySnapshot := *snapshot

	return &copySnapshot, nil
}

// Health returns the health status of the event store
func (es *InMemoryEventStore) Health(ctx context.Context) (EventStoreHealth, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	return EventStoreHealth{
		Status:     "healthy",
		StoreType:  es.config.StoreType,
		NodeID:     es.config.NodeID,
		Timestamp:  time.Now(),
		EventCount: es.stats.TotalEvents,
		StoreSize:  es.stats.StoreSize,
		Metadata: map[string]interface{}{
			"total_aggregates": es.stats.TotalAggregates,
			"write_operations": es.stats.WriteOperations,
			"read_operations":  es.stats.ReadOperations,
			"snapshot_count":   es.stats.SnapshotCount,
		},
	}, nil
}

// CompactEvents compacts events for an aggregate up to a target version
func (es *InMemoryEventStore) CompactEvents(ctx context.Context, aggregateID string, targetVersion int64) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	aggregateEvents, exists := es.events[aggregateID]
	if !exists {
		return fmt.Errorf("aggregate not found: %s", aggregateID)
	}

	// Count events to compact
	var eventsToCompact int
	for _, event := range aggregateEvents {
		if event.Version <= targetVersion {
			eventsToCompact++
		} else {
			break
		}
	}

	if eventsToCompact == 0 {
		return nil // Nothing to compact
	}

	// Remove events up to target version
	remainingEvents := aggregateEvents[eventsToCompact:]
	es.events[aggregateID] = remainingEvents

	// Update compaction stats
	es.stats.CompactionStats.LastCompaction = &time.Time{}
	*es.stats.CompactionStats.LastCompaction = time.Now()
	es.stats.CompactionStats.CompactionsCount++
	es.stats.CompactionStats.EventsCompacted += int64(eventsToCompact)

	es.logger.Info("Events compacted",
		zap.String("aggregate_id", aggregateID),
		zap.Int64("target_version", targetVersion),
		zap.Int("events_compacted", eventsToCompact),
		zap.Int("remaining_events", len(remainingEvents)))

	return nil
}

// PruneEvents removes events older than the specified time
func (es *InMemoryEventStore) PruneEvents(ctx context.Context, beforeTime time.Time) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	eventsPruned := 0

	for aggregateID, aggregateEvents := range es.events {
		var remainingEvents []*Event

		for _, event := range aggregateEvents {
			if event.Timestamp.After(beforeTime) {
				remainingEvents = append(remainingEvents, event)
			} else {
				eventsPruned++
			}
		}

		if len(remainingEvents) != len(aggregateEvents) {
			es.events[aggregateID] = remainingEvents
		}
	}

	if eventsPruned > 0 {
		es.logger.Info("Events pruned",
			zap.Time("before_time", beforeTime),
			zap.Int("events_pruned", eventsPruned))
	}

	return nil
}

// Private helper methods

func (es *InMemoryEventStore) validateEvent(event *Event) error {
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.AggregateID == "" {
		return fmt.Errorf("aggregate ID is required")
	}
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if event.Version <= 0 {
		return fmt.Errorf("event version must be positive")
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}

	// Check event size
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event for validation: %w", err)
	}

	if int64(len(eventData)) > es.config.MaxEventSize {
		return fmt.Errorf("event size exceeds maximum allowed size")
	}

	return nil
}

func (es *InMemoryEventStore) updateAggregateMetadata(event *Event) {
	info, exists := es.metadata[event.AggregateID]
	if !exists {
		info = &AggregateInfo{
			AggregateID:   event.AggregateID,
			AggregateType: event.AggregateType,
			Version:       0,
			EventCount:    0,
			FirstEvent:    &event.Timestamp,
		}
		es.metadata[event.AggregateID] = info
		es.stats.TotalAggregates++
	}

	info.Version = event.Version
	info.EventCount++
	info.LastEvent = &event.Timestamp
	info.Size += int64(len(event.ID)) + 100 // Rough estimate
}

func (es *InMemoryEventStore) updateEventStats(event *Event) {
	es.stats.TotalEvents++
	es.stats.StoreSize += int64(len(event.ID)) + 100 // Rough estimate

	// Update events by type count
	eventType := string(event.Type)
	es.stats.EventsByType[eventType]++

	// Update last event time
	es.stats.LastEvent = &time.Time{}
	*es.stats.LastEvent = event.Timestamp

	// Update average event size
	if es.stats.TotalEvents > 0 {
		es.stats.AverageEventSize = float64(es.stats.StoreSize) / float64(es.stats.TotalEvents)
	}
}

func (es *InMemoryEventStore) subscribeToAggregate(aggregateID string, eventCh chan *Event) {
	// In a real implementation, this would manage subscriptions more efficiently
	es.mu.Lock()
	defer es.mu.Unlock()

	// Add subscriber for this aggregate
	if es.subscribers[aggregateID] == nil {
		es.subscribers[aggregateID] = []chan *Event{}
	}
	es.subscribers[aggregateID] = append(es.subscribers[aggregateID], eventCh)
}

func (es *InMemoryEventStore) backgroundProcesses() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-es.ctx.Done():
			es.logger.Info("Event store background processes stopped")
			return
		case <-ticker.C:
			es.maintenanceTasks()
		}
	}
}

func (es *InMemoryEventStore) maintenanceTasks() {
	// Perform background maintenance tasks
	// This could include:
	// - Pruning old events
	// - Compacting old events
	// - Updating statistics
	// - Cleaning up expired subscriptions

	es.logger.Debug("Running maintenance tasks")
}

// Close closes the event store
func (es *InMemoryEventStore) Close() error {
	es.cancel()
	close(es.streamCh)

	// Close all subscriber channels
	es.mu.Lock()
	for _, subscribers := range es.subscribers {
		for _, subscriber := range subscribers {
			close(subscriber)
		}
	}
	es.subscribers = make(map[string][]chan *Event)
	es.mu.Unlock()

	es.logger.Info("In-memory event store closed")
	return nil
}
