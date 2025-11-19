package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ReplayStrategy represents different replay strategies
type ReplayStrategy string

const (
	ReplayStrategySequential ReplayStrategy = "sequential"
	ReplayStrategyParallel   ReplayStrategy = "parallel"
	ReplayStrategyBatch      ReplayStrategy = "batch"
)

// ReplayConfig represents replay configuration
type ReplayConfig struct {
	Strategy         ReplayStrategy                `json:"strategy"`
	BatchSize        int                           `json:"batch_size"`
	ParallelWorkers  int                           `json:"parallel_workers"`
	StopOnError      bool                          `json:"stop_on_error"`
	MaxRetries       int                           `json:"max_retries"`
	RetryDelay       time.Duration                 `json:"retry_delay"`
	ProgressCallback func(progress ReplayProgress) `json:"-"`
}

// DefaultReplayConfig returns default replay configuration
func DefaultReplayConfig() *ReplayConfig {
	return &ReplayConfig{
		Strategy:        ReplayStrategySequential,
		BatchSize:       1000, // Aumentado de 100 para 1000
		ParallelWorkers: 16,   // Aumentado de 4 para 16
		StopOnError:     false,
		MaxRetries:      3,
		RetryDelay:      1 * time.Second,
	}
}

// ReplayProgress represents replay progress information
type ReplayProgress struct {
	TotalEvents     int64         `json:"total_events"`
	ProcessedEvents int64         `json:"processed_events"`
	FailedEvents    int64         `json:"failed_events"`
	CurrentVersion  int64         `json:"current_version"`
	StartTime       time.Time     `json:"start_time"`
	ElapsedTime     time.Duration `json:"elapsed_time"`
	Percentage      float64       `json:"percentage"`
	IsComplete      bool          `json:"is_complete"`
	LastError       string        `json:"last_error,omitempty"`
}

// ReplayHandler defines a handler function for replaying events
type ReplayHandler interface {
	Handle(ctx context.Context, event *Event) error
	CanHandle(event *Event) bool
	GetHandlerType() string
}

// EventReplay interface for replaying events
type EventReplay interface {
	// Replay operations
	ReplayEvents(ctx context.Context, aggregateID string, fromVersion int64, toVersion int64, handler ReplayHandler) (*ReplayProgress, error)
	ReplayAllEvents(ctx context.Context, aggregateID string, handler ReplayHandler) (*ReplayProgress, error)
	ReplayEventsByType(ctx context.Context, eventType EventType, fromTime time.Time, handler ReplayHandler) (*ReplayProgress, error)

	// Replay with snapshot
	ReplayFromSnapshot(ctx context.Context, aggregateID string, snapshotVersion int64, handler ReplayHandler) (*ReplayProgress, error)

	// Replay state
	ReplayToState(ctx context.Context, aggregateID string, targetVersion int64, handler ReplayHandler) (interface{}, error)

	// Statistics
	GetReplayStats(ctx context.Context) (*ReplayStats, error)
}

// ReplayStats represents replay statistics
type ReplayStats struct {
	TotalReplays        int64         `json:"total_replays"`
	SuccessfulReplays   int64         `json:"successful_replays"`
	FailedReplays       int64         `json:"failed_replays"`
	TotalEventsReplayed int64         `json:"total_events_replayed"`
	AverageReplayTime   time.Duration `json:"average_replay_time"`
	LastReplayTime      *time.Time    `json:"last_replay_time"`
	LastReplayError     string        `json:"last_replay_error,omitempty"`
}

// EventReplayImpl implements EventReplay interface
type EventReplayImpl struct {
	store  EventStore
	config *ReplayConfig
	stats  *ReplayStats
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewEventReplay creates a new event replay implementation
func NewEventReplay(store EventStore, config *ReplayConfig) *EventReplayImpl {
	if config == nil {
		config = DefaultReplayConfig()
	}

	return &EventReplayImpl{
		store:  store,
		config: config,
		stats:  &ReplayStats{},
		logger: logger.Get(),
	}
}

// ReplayEvents replays events for an aggregate within a version range
func (er *EventReplayImpl) ReplayEvents(ctx context.Context, aggregateID string, fromVersion int64, toVersion int64, handler ReplayHandler) (*ReplayProgress, error) {
	er.mu.Lock()
	er.stats.TotalReplays++
	er.mu.Unlock()

	startTime := time.Now()

	// Get events
	events, err := er.store.GetEvents(ctx, aggregateID, fromVersion, toVersion)
	if err != nil {
		er.mu.Lock()
		er.stats.FailedReplays++
		er.mu.Unlock()
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	if len(events) == 0 {
		return &ReplayProgress{
			TotalEvents:     0,
			ProcessedEvents: 0,
			IsComplete:      true,
			StartTime:       startTime,
			ElapsedTime:     time.Since(startTime),
		}, nil
	}

	progress := &ReplayProgress{
		TotalEvents:    int64(len(events)),
		StartTime:      startTime,
		CurrentVersion: fromVersion,
	}

	// Replay events based on strategy
	var replayErr error
	switch er.config.Strategy {
	case ReplayStrategySequential:
		replayErr = er.replaySequential(ctx, events, handler, progress)
	case ReplayStrategyParallel:
		replayErr = er.replayParallel(ctx, events, handler, progress)
	case ReplayStrategyBatch:
		replayErr = er.replayBatch(ctx, events, handler, progress)
	default:
		replayErr = er.replaySequential(ctx, events, handler, progress)
	}

	progress.ElapsedTime = time.Since(startTime)
	progress.IsComplete = true

	if replayErr != nil {
		progress.LastError = replayErr.Error()
		er.mu.Lock()
		er.stats.FailedReplays++
		er.mu.Unlock()
		return progress, replayErr
	}

	er.mu.Lock()
	er.stats.SuccessfulReplays++
	er.stats.TotalEventsReplayed += progress.ProcessedEvents
	er.updateAverageReplayTime(progress.ElapsedTime)
	lastTime := time.Now()
	er.stats.LastReplayTime = &lastTime
	er.mu.Unlock()

	return progress, nil
}

// ReplayAllEvents replays all events for an aggregate
func (er *EventReplayImpl) ReplayAllEvents(ctx context.Context, aggregateID string, handler ReplayHandler) (*ReplayProgress, error) {
	events, err := er.store.GetAllEvents(ctx, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	if len(events) == 0 {
		return &ReplayProgress{
			TotalEvents:     0,
			ProcessedEvents: 0,
			IsComplete:      true,
		}, nil
	}

	fromVersion := int64(1)
	toVersion := events[len(events)-1].Version

	return er.ReplayEvents(ctx, aggregateID, fromVersion, toVersion, handler)
}

// ReplayEventsByType replays events of a specific type from a time
func (er *EventReplayImpl) ReplayEventsByType(ctx context.Context, eventType EventType, fromTime time.Time, handler ReplayHandler) (*ReplayProgress, error) {
	events, err := er.store.GetEventsByTimeRange(ctx, fromTime, time.Now(), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by time range: %w", err)
	}

	// Filter by type
	filteredEvents := make([]*Event, 0)
	for _, event := range events {
		if event.Type == eventType {
			filteredEvents = append(filteredEvents, event)
		}
	}

	if len(filteredEvents) == 0 {
		return &ReplayProgress{
			TotalEvents:     0,
			ProcessedEvents: 0,
			IsComplete:      true,
		}, nil
	}

	startTime := time.Now()
	progress := &ReplayProgress{
		TotalEvents: int64(len(filteredEvents)),
		StartTime:   startTime,
	}

	err = er.replaySequential(ctx, filteredEvents, handler, progress)
	progress.ElapsedTime = time.Since(startTime)
	progress.IsComplete = true

	if err != nil {
		progress.LastError = err.Error()
		return progress, err
	}

	return progress, nil
}

// ReplayFromSnapshot replays events from a snapshot version
func (er *EventReplayImpl) ReplayFromSnapshot(ctx context.Context, aggregateID string, snapshotVersion int64, handler ReplayHandler) (*ReplayProgress, error) {
	// Get snapshot
	snapshot, err := er.store.GetSnapshot(ctx, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Get events after snapshot version
	events, err := er.store.GetEvents(ctx, aggregateID, snapshotVersion+1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get events after snapshot: %w", err)
	}

	if len(events) == 0 {
		return &ReplayProgress{
			TotalEvents:     0,
			ProcessedEvents: 0,
			IsComplete:      true,
		}, nil
	}

	startTime := time.Now()
	progress := &ReplayProgress{
		TotalEvents:    int64(len(events)),
		StartTime:      startTime,
		CurrentVersion: snapshotVersion + 1,
	}

	err = er.replaySequential(ctx, events, handler, progress)
	progress.ElapsedTime = time.Since(startTime)
	progress.IsComplete = true

	if err != nil {
		progress.LastError = err.Error()
		return progress, err
	}

	return progress, nil
}

// ReplayToState replays events to rebuild state at a specific version
func (er *EventReplayImpl) ReplayToState(ctx context.Context, aggregateID string, targetVersion int64, handler ReplayHandler) (interface{}, error) {
	// Get all events up to target version
	events, err := er.store.GetEvents(ctx, aggregateID, 1, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Replay events
	progress, err := er.ReplayEvents(ctx, aggregateID, 1, targetVersion, handler)
	if err != nil {
		return nil, fmt.Errorf("replay failed: %w", err)
	}

	// Return final state (handler should maintain state)
	// This is a simplified version - in practice, handler would return state
	return progress, nil
}

// GetReplayStats returns replay statistics
func (er *EventReplayImpl) GetReplayStats(ctx context.Context) (*ReplayStats, error) {
	er.mu.RLock()
	defer er.mu.RUnlock()

	stats := *er.stats
	return &stats, nil
}

// Private helper methods

func (er *EventReplayImpl) replaySequential(ctx context.Context, events []*Event, handler ReplayHandler, progress *ReplayProgress) error {
	for _, event := range events {
		if !handler.CanHandle(event) {
			continue
		}

		var err error
		for attempt := 0; attempt <= er.config.MaxRetries; attempt++ {
			err = handler.Handle(ctx, event)
			if err == nil {
				break
			}

			if attempt < er.config.MaxRetries {
				time.Sleep(er.config.RetryDelay)
			}
		}

		if err != nil {
			if er.config.StopOnError {
				progress.LastError = err.Error()
				return fmt.Errorf("replay failed at version %d: %w", event.Version, err)
			}
			progress.FailedEvents++
			er.logger.Warn("Event replay failed",
				zap.String("event_id", event.ID),
				zap.Int64("version", event.Version),
				zap.Error(err))
		} else {
			progress.ProcessedEvents++
			progress.CurrentVersion = event.Version
		}

		progress.Percentage = float64(progress.ProcessedEvents) / float64(progress.TotalEvents) * 100

		if er.config.ProgressCallback != nil {
			er.config.ProgressCallback(*progress)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

func (er *EventReplayImpl) replayParallel(ctx context.Context, events []*Event, handler ReplayHandler, progress *ReplayProgress) error {
	// Use worker pool for parallel replay
	workerCount := er.config.ParallelWorkers
	if workerCount <= 0 {
		workerCount = 4
	}

	eventCh := make(chan *Event, len(events))
	for _, event := range events {
		eventCh <- event
	}
	close(eventCh)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := range eventCh {
				if !handler.CanHandle(event) {
					continue
				}

				var err error
				for attempt := 0; attempt <= er.config.MaxRetries; attempt++ {
					err = handler.Handle(ctx, event)
					if err == nil {
						break
					}
					if attempt < er.config.MaxRetries {
						time.Sleep(er.config.RetryDelay)
					}
				}

				mu.Lock()
				if err != nil {
					if er.config.StopOnError {
						progress.LastError = err.Error()
					}
					progress.FailedEvents++
				} else {
					progress.ProcessedEvents++
					if event.Version > progress.CurrentVersion {
						progress.CurrentVersion = event.Version
					}
				}
				progress.Percentage = float64(progress.ProcessedEvents) / float64(progress.TotalEvents) * 100
				mu.Unlock()

				if err != nil && er.config.StopOnError {
					return
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

func (er *EventReplayImpl) replayBatch(ctx context.Context, events []*Event, handler ReplayHandler, progress *ReplayProgress) error {
	batchSize := er.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}

		batch := events[i:end]
		err := er.replaySequential(ctx, batch, handler, progress)
		if err != nil && er.config.StopOnError {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

func (er *EventReplayImpl) updateAverageReplayTime(duration time.Duration) {
	if er.stats.TotalReplays > 0 {
		total := er.stats.AverageReplayTime * time.Duration(er.stats.TotalReplays-1)
		er.stats.AverageReplayTime = (total + duration) / time.Duration(er.stats.TotalReplays)
	} else {
		er.stats.AverageReplayTime = duration
	}
}
