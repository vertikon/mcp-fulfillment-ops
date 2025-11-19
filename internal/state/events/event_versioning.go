package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// VersioningStrategy represents different versioning strategies
type VersioningStrategy string

const (
	VersioningStrategySequential VersioningStrategy = "sequential"
	VersioningStrategyTimestamp  VersioningStrategy = "timestamp"
	VersioningStrategyVectorClock VersioningStrategy = "vector-clock"
)

// VersionInfo represents version information for an aggregate
type VersionInfo struct {
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	CurrentVersion int64                  `json:"current_version"`
	LastEventID   string                 `json:"last_event_id"`
	LastEventTime time.Time              `json:"last_event_time"`
	VersionHistory []VersionHistoryEntry  `json:"version_history,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// VersionHistoryEntry represents a version history entry
type VersionHistoryEntry struct {
	Version     int64     `json:"version"`
	EventID     string    `json:"event_id"`
	Timestamp   time.Time `json:"timestamp"`
	EventType   EventType `json:"event_type"`
	Description string    `json:"description,omitempty"`
}

// VersionConflict represents a version conflict
type VersionConflict struct {
	AggregateID     string    `json:"aggregate_id"`
	ExpectedVersion int64     `json:"expected_version"`
	ActualVersion   int64     `json:"actual_version"`
	ConflictTime    time.Time `json:"conflict_time"`
	Resolution      string    `json:"resolution,omitempty"`
}

// VersioningConfig represents versioning configuration
type VersioningConfig struct {
	Strategy        VersioningStrategy `json:"strategy"`
	EnableHistory   bool               `json:"enable_history"`
	HistoryRetention int                `json:"history_retention"`
	ConflictResolution string           `json:"conflict_resolution"`
	AutoIncrement    bool               `json:"auto_increment"`
}

// DefaultVersioningConfig returns default versioning configuration
func DefaultVersioningConfig() *VersioningConfig {
	return &VersioningConfig{
		Strategy:        VersioningStrategySequential,
		EnableHistory:   true,
		HistoryRetention: 100,
		ConflictResolution: "reject",
		AutoIncrement:    true,
	}
}

// EventVersioning interface for managing event versioning
type EventVersioning interface {
	// Version operations
	GetVersion(ctx context.Context, aggregateID string) (*VersionInfo, error)
	IncrementVersion(ctx context.Context, aggregateID string, event *Event) (int64, error)
	ValidateVersion(ctx context.Context, aggregateID string, expectedVersion int64) error
	
	// Version history
	GetVersionHistory(ctx context.Context, aggregateID string, limit int) ([]VersionHistoryEntry, error)
	AddVersionHistory(ctx context.Context, aggregateID string, entry VersionHistoryEntry) error
	
	// Conflict resolution
	ResolveVersionConflict(ctx context.Context, conflict *VersionConflict) (int64, error)
	GetVersionConflicts(ctx context.Context, aggregateID string) ([]*VersionConflict, error)
	
	// Statistics
	GetVersioningStats(ctx context.Context) (*VersioningStats, error)
}

// VersioningStats represents versioning statistics
type VersioningStats struct {
	TotalVersions      int64         `json:"total_versions"`
	TotalConflicts     int64         `json:"total_conflicts"`
	ResolvedConflicts  int64         `json:"resolved_conflicts"`
	AverageVersionGap  float64       `json:"average_version_gap"`
	LastConflict       *time.Time    `json:"last_conflict,omitempty"`
	VersionDistribution map[int64]int64 `json:"version_distribution"`
}

// EventVersioningImpl implements EventVersioning interface
type EventVersioningImpl struct {
	store      EventStore
	config     *VersioningConfig
	versions   map[string]*VersionInfo
	conflicts  map[string][]*VersionConflict
	stats      *VersioningStats
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewEventVersioning creates a new event versioning implementation
func NewEventVersioning(store EventStore, config *VersioningConfig) *EventVersioningImpl {
	if config == nil {
		config = DefaultVersioningConfig()
	}
	
	return &EventVersioningImpl{
		store:    store,
		config:   config,
		versions: make(map[string]*VersionInfo),
		conflicts: make(map[string][]*VersionConflict),
		stats: &VersioningStats{
			VersionDistribution: make(map[int64]int64),
		},
		logger: logger.Get(),
	}
}

// GetVersion returns version information for an aggregate
func (ev *EventVersioningImpl) GetVersion(ctx context.Context, aggregateID string) (*VersionInfo, error) {
	ev.mu.RLock()
	versionInfo, exists := ev.versions[aggregateID]
	ev.mu.RUnlock()
	
	if exists {
		// Return copy
		copy := *versionInfo
		if versionInfo.VersionHistory != nil {
			copy.VersionHistory = make([]VersionHistoryEntry, len(versionInfo.VersionHistory))
			copy(versionInfo.VersionHistory, copy.VersionHistory)
		}
		return &copy, nil
	}
	
	// Load from event store
	aggregateInfo, err := ev.store.GetAggregateInfo(ctx, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregate info: %w", err)
	}
	
	versionInfo = &VersionInfo{
		AggregateID:    aggregateID,
		AggregateType:  aggregateInfo.AggregateType,
		CurrentVersion: aggregateInfo.Version,
		Metadata:       make(map[string]interface{}),
	}
	
	if ev.config.EnableHistory {
		versionInfo.VersionHistory = make([]VersionHistoryEntry, 0)
	}
	
	ev.mu.Lock()
	ev.versions[aggregateID] = versionInfo
	ev.mu.Unlock()
	
	return versionInfo, nil
}

// IncrementVersion increments version for an aggregate
func (ev *EventVersioningImpl) IncrementVersion(ctx context.Context, aggregateID string, event *Event) (int64, error) {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	
	versionInfo, exists := ev.versions[aggregateID]
	if !exists {
		// Load from store
		aggregateInfo, err := ev.store.GetAggregateInfo(ctx, aggregateID)
		if err != nil {
			return 0, fmt.Errorf("failed to get aggregate info: %w", err)
		}
		
		versionInfo = &VersionInfo{
			AggregateID:    aggregateID,
			AggregateType:  aggregateInfo.AggregateType,
			CurrentVersion: aggregateInfo.Version,
			Metadata:       make(map[string]interface{}),
		}
		
		if ev.config.EnableHistory {
			versionInfo.VersionHistory = make([]VersionHistoryEntry, 0)
		}
		
		ev.versions[aggregateID] = versionInfo
	}
	
	// Increment version
	newVersion := versionInfo.CurrentVersion + 1
	
	// Validate version continuity
	if event.Version != 0 && event.Version != newVersion {
		conflict := &VersionConflict{
			AggregateID:     aggregateID,
			ExpectedVersion: newVersion,
			ActualVersion:   event.Version,
			ConflictTime:    time.Now(),
		}
		
		ev.stats.TotalConflicts++
		ev.conflicts[aggregateID] = append(ev.conflicts[aggregateID], conflict)
		
		if ev.config.ConflictResolution == "reject" {
			return 0, fmt.Errorf("version conflict: expected %d, got %d", newVersion, event.Version)
		}
		
		// Resolve conflict
		resolvedVersion, err := ev.resolveVersionConflict(ctx, conflict)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve conflict: %w", err)
		}
		
		newVersion = resolvedVersion
		conflict.Resolution = fmt.Sprintf("resolved to %d", resolvedVersion)
		ev.stats.ResolvedConflicts++
	}
	
	// Update version info
	versionInfo.CurrentVersion = newVersion
	versionInfo.LastEventID = event.ID
	versionInfo.LastEventTime = event.Timestamp
	
	// Add to history
	if ev.config.EnableHistory {
		entry := VersionHistoryEntry{
			Version:   newVersion,
			EventID:   event.ID,
			Timestamp: event.Timestamp,
			EventType: event.Type,
		}
		
		versionInfo.VersionHistory = append(versionInfo.VersionHistory, entry)
		
		// Trim history if needed
		if len(versionInfo.VersionHistory) > ev.config.HistoryRetention {
			versionInfo.VersionHistory = versionInfo.VersionHistory[len(versionInfo.VersionHistory)-ev.config.HistoryRetention:]
		}
	}
	
	// Update statistics
	ev.stats.TotalVersions++
	ev.stats.VersionDistribution[newVersion]++
	
	ev.logger.Debug("Version incremented",
		zap.String("aggregate_id", aggregateID),
		zap.Int64("version", newVersion))
	
	return newVersion, nil
}

// ValidateVersion validates that the expected version matches
func (ev *EventVersioningImpl) ValidateVersion(ctx context.Context, aggregateID string, expectedVersion int64) error {
	versionInfo, err := ev.GetVersion(ctx, aggregateID)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	
	if versionInfo.CurrentVersion != expectedVersion {
		conflict := &VersionConflict{
			AggregateID:     aggregateID,
			ExpectedVersion: expectedVersion,
			ActualVersion:   versionInfo.CurrentVersion,
			ConflictTime:    time.Now(),
		}
		
		ev.mu.Lock()
		ev.stats.TotalConflicts++
		ev.conflicts[aggregateID] = append(ev.conflicts[aggregateID], conflict)
		lastTime := time.Now()
		ev.stats.LastConflict = &lastTime
		ev.mu.Unlock()
		
		return fmt.Errorf("version mismatch: expected %d, actual %d", expectedVersion, versionInfo.CurrentVersion)
	}
	
	return nil
}

// GetVersionHistory returns version history for an aggregate
func (ev *EventVersioningImpl) GetVersionHistory(ctx context.Context, aggregateID string, limit int) ([]VersionHistoryEntry, error) {
	versionInfo, err := ev.GetVersion(ctx, aggregateID)
	if err != nil {
		return nil, err
	}
	
	if !ev.config.EnableHistory || versionInfo.VersionHistory == nil {
		return []VersionHistoryEntry{}, nil
	}
	
	history := versionInfo.VersionHistory
	if limit > 0 && limit < len(history) {
		history = history[len(history)-limit:]
	}
	
	return history, nil
}

// AddVersionHistory adds an entry to version history
func (ev *EventVersioningImpl) AddVersionHistory(ctx context.Context, aggregateID string, entry VersionHistoryEntry) error {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	
	versionInfo, exists := ev.versions[aggregateID]
	if !exists {
		return fmt.Errorf("aggregate not found: %s", aggregateID)
	}
	
	if !ev.config.EnableHistory {
		return nil
	}
	
	if versionInfo.VersionHistory == nil {
		versionInfo.VersionHistory = make([]VersionHistoryEntry, 0)
	}
	
	versionInfo.VersionHistory = append(versionInfo.VersionHistory, entry)
	
	// Trim history if needed
	if len(versionInfo.VersionHistory) > ev.config.HistoryRetention {
		versionInfo.VersionHistory = versionInfo.VersionHistory[len(versionInfo.VersionHistory)-ev.config.HistoryRetention:]
	}
	
	return nil
}

// ResolveVersionConflict resolves a version conflict
func (ev *EventVersioningImpl) ResolveVersionConflict(ctx context.Context, conflict *VersionConflict) (int64, error) {
	switch ev.config.ConflictResolution {
	case "reject":
		return 0, fmt.Errorf("version conflict rejected")
		
	case "accept-higher":
		if conflict.ActualVersion > conflict.ExpectedVersion {
			return conflict.ActualVersion, nil
		}
		return conflict.ExpectedVersion, nil
		
	case "accept-lower":
		if conflict.ActualVersion < conflict.ExpectedVersion {
			return conflict.ActualVersion, nil
		}
		return conflict.ExpectedVersion, nil
		
	case "increment":
		return conflict.ActualVersion + 1, nil
		
	default:
		return conflict.ExpectedVersion, nil
	}
}

// GetVersionConflicts returns version conflicts for an aggregate
func (ev *EventVersioningImpl) GetVersionConflicts(ctx context.Context, aggregateID string) ([]*VersionConflict, error) {
	ev.mu.RLock()
	defer ev.mu.RUnlock()
	
	conflicts, exists := ev.conflicts[aggregateID]
	if !exists {
		return []*VersionConflict{}, nil
	}
	
	// Return copy
	result := make([]*VersionConflict, len(conflicts))
	copy(result, conflicts)
	
	return result, nil
}

// GetVersioningStats returns versioning statistics
func (ev *EventVersioningImpl) GetVersioningStats(ctx context.Context) (*VersioningStats, error) {
	ev.mu.RLock()
	defer ev.mu.RUnlock()
	
	stats := *ev.stats
	stats.VersionDistribution = make(map[int64]int64)
	for k, v := range ev.stats.VersionDistribution {
		stats.VersionDistribution[k] = v
	}
	
	return &stats, nil
}
