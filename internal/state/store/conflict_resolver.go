package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ConflictResolutionStrategy represents different conflict resolution strategies
type ConflictResolutionStrategy string

const (
	LastWriteWins       ConflictResolutionStrategy = "last-write-wins"
	FirstWriteWins      ConflictResolutionStrategy = "first-write-wins"
	VectorClock         ConflictResolutionStrategy = "vector-clock"
	CRDTLastWriterWins  ConflictResolutionStrategy = "crdt-lww"
	CRDTMerge           ConflictResolutionStrategy = "crdt-merge"
)

// Conflict represents a state conflict
type Conflict struct {
	Key           string                 `json:"key"`
	LocalState    *VersionedState        `json:"local_state"`
	RemoteState   *VersionedState        `json:"remote_state"`
	Strategy      ConflictResolutionStrategy `json:"strategy"`
	Timestamp     time.Time              `json:"timestamp"`
	NodeID        string                 `json:"node_id"`
	Resolution    *VersionedState        `json:"resolution,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
}

// ConflictResolver interface for resolving state conflicts
type ConflictResolver interface {
	Resolve(ctx context.Context, conflict *Conflict) (*VersionedState, error)
	GetStrategy() ConflictResolutionStrategy
	SetStrategy(strategy ConflictResolutionStrategy) error
	GetConflictStats() ConflictStats
}

// ConflictStats represents conflict resolution statistics
type ConflictStats struct {
	TotalConflicts      int64                    `json:"total_conflicts"`
	ResolvedConflicts   int64                    `json:"resolved_conflicts"`
	FailedResolutions   int64                    `json:"failed_resolutions"`
	StrategyCounts      map[string]int64          `json:"strategy_counts"`
	AverageResolution   time.Duration            `json:"average_resolution"`
	LastResolution     *time.Time               `json:"last_resolution"`
}

// ConflictResolverConfig represents conflict resolver configuration
type ConflictResolverConfig struct {
	DefaultStrategy    ConflictResolutionStrategy `json:"default_strategy"`
	NodeID            string                  `json:"node_id"`
	EnableAutoMerge    bool                    `json:"enable_auto_merge"`
	MaxRetryAttempts   int                     `json:"max_retry_attempts"`
	RetryDelay         time.Duration           `json:"retry_delay"`
	StatsRetentionDays int                     `json:"stats_retention_days"`
}

// DefaultConflictResolverConfig returns default conflict resolver configuration
func DefaultConflictResolverConfig() *ConflictResolverConfig {
	return &ConflictResolverConfig{
		DefaultStrategy:    LastWriteWins,
		NodeID:            fmt.Sprintf("resolver-%d", time.Now().Unix()),
		EnableAutoMerge:    true,
		MaxRetryAttempts:   3,
		RetryDelay:         100 * time.Millisecond,
		StatsRetentionDays: 30,
	}
}

// ConflictResolverImpl implements ConflictResolver interface
type ConflictResolverImpl struct {
	config *ConflictResolverConfig
	stats  *ConflictStats
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(config *ConflictResolverConfig) *ConflictResolverImpl {
	if config == nil {
		config = DefaultConflictResolverConfig()
	}
	
	resolver := &ConflictResolverImpl{
		config: config,
		stats: &ConflictStats{
			StrategyCounts: make(map[string]int64),
		},
		logger: logger.Get(),
	}
	
	// Initialize strategy count
	resolver.stats.StrategyCounts[string(config.DefaultStrategy)] = 0
	
	resolver.logger.Info("Conflict resolver initialized",
		zap.String("node_id", config.NodeID),
		zap.String("default_strategy", string(config.DefaultStrategy)),
		zap.Bool("auto_merge", config.EnableAutoMerge))
	
	return resolver
}

// Resolve resolves a conflict using the configured strategy
func (r *ConflictResolverImpl) Resolve(ctx context.Context, conflict *Conflict) (*VersionedState, error) {
	r.mu.Lock()
	r.stats.TotalConflicts++
	r.mu.Unlock()
	
	startTime := time.Now()
	
	r.logger.Info("Resolving conflict",
		zap.String("key", conflict.Key),
		zap.String("strategy", string(conflict.Strategy)),
		zap.Uint64("local_version", conflict.LocalState.Version),
		zap.Uint64("remote_version", conflict.RemoteState.Version))
	
	// Use conflict's strategy if specified, otherwise use default
	strategy := conflict.Strategy
	if strategy == "" {
		strategy = r.config.DefaultStrategy
	}
	
	var resolution *VersionedState
	var err error
	
	switch strategy {
	case LastWriteWins:
		resolution, err = r.resolveLastWriteWins(conflict)
	case FirstWriteWins:
		resolution, err = r.resolveFirstWriteWins(conflict)
	case VectorClock:
		resolution, err = r.resolveVectorClock(conflict)
	case CRDTLastWriterWins:
		resolution, err = r.resolveCRDTLastWriterWins(conflict)
	case CRDTMerge:
		resolution, err = r.resolveCRDTMerge(conflict)
	default:
		resolution, err = r.resolveLastWriteWins(conflict) // Fallback
	}
	
	// Update statistics
	duration := time.Since(startTime)
	r.updateStats(strategy, resolution, err, duration)
	
	if err != nil {
		r.logger.Error("Conflict resolution failed",
			zap.String("key", conflict.Key),
			zap.String("strategy", string(strategy)),
			zap.Error(err))
		return nil, err
	}
	
	// Add metadata to resolution
	if resolution.Meta == nil {
		resolution.Meta = make(map[string]interface{})
	}
	resolution.Meta["conflict_resolved"] = true
	resolution.Meta["conflict_strategy"] = string(strategy)
	resolution.Meta["conflict_timestamp"] = conflict.Timestamp
	resolution.Meta["conflict_node_id"] = conflict.NodeID
	
	r.logger.Info("Conflict resolved",
		zap.String("key", conflict.Key),
		zap.String("strategy", string(strategy)),
		zap.Uint64("resolved_version", resolution.Version),
		zap.Duration("resolution_time", duration))
	
	return resolution, nil
}

// resolveLastWriteWins resolves conflict using last-write-wins strategy
func (r *ConflictResolverImpl) resolveLastWriteWins(conflict *Conflict) (*VersionedState, error) {
	localTime := r.getStateTimestamp(conflict.LocalState)
	remoteTime := r.getStateTimestamp(conflict.RemoteState)
	
	if localTime.After(remoteTime) {
		return &VersionedState{
			Key:     conflict.Key,
			Value:   conflict.LocalState.Value,
			Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
			TTL:     conflict.LocalState.TTL,
			Meta:    r.mergeMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta),
		}, nil
	}
	
	return &VersionedState{
		Key:     conflict.Key,
		Value:   conflict.RemoteState.Value,
		Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
		TTL:     conflict.RemoteState.TTL,
		Meta:    r.mergeMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta),
	}, nil
}

// resolveFirstWriteWins resolves conflict using first-write-wins strategy
func (r *ConflictResolverImpl) resolveFirstWriteWins(conflict *Conflict) (*VersionedState, error) {
	localTime := r.getStateTimestamp(conflict.LocalState)
	remoteTime := r.getStateTimestamp(conflict.RemoteState)
	
	if localTime.Before(remoteTime) || localTime.Equal(remoteTime) {
		return &VersionedState{
			Key:     conflict.Key,
			Value:   conflict.LocalState.Value,
			Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
			TTL:     conflict.LocalState.TTL,
			Meta:    r.mergeMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta),
		}, nil
	}
	
	return &VersionedState{
		Key:     conflict.Key,
		Value:   conflict.RemoteState.Value,
		Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
		TTL:     conflict.RemoteState.TTL,
		Meta:    r.mergeMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta),
	}, nil
}

// resolveVectorClock resolves conflict using vector clock strategy
func (r *ConflictResolverImpl) resolveVectorClock(conflict *Conflict) (*VersionedState, error) {
	// Extract vector clocks from metadata
	localVC := r.getVectorClock(conflict.LocalState)
	remoteVC := r.getVectorClock(conflict.RemoteState)
	
	// Compare vector clocks
	comparison := r.compareVectorClocks(localVC, remoteVC)
	
	switch comparison {
	case "local_greater":
		return r.createResolvedState(conflict.LocalState, conflict.RemoteState, "vector-clock-local")
	case "remote_greater":
		return r.createResolvedState(conflict.RemoteState, conflict.LocalState, "vector-clock-remote")
	case "concurrent":
		// Conflict detected, need merge
		return r.resolveCRDTMerge(conflict)
	default:
		// Same vector clock, use timestamp as tie-breaker
		return r.resolveLastWriteWins(conflict)
	}
}

// resolveCRDTLastWriterWins resolves conflict using CRDT LWW strategy
func (r *ConflictResolverImpl) resolveCRDTLastWriterWins(conflict *Conflict) (*VersionedState, error) {
	// CRDT LWW uses timestamps for conflict resolution
	localTime := r.getStateTimestamp(conflict.LocalState)
	remoteTime := r.getStateTimestamp(conflict.RemoteState)
	
	// Ensure both states have timestamps
	localTS := r.ensureTimestamp(conflict.LocalState)
	remoteTS := r.ensureTimestamp(conflict.RemoteState)
	
	if localTS.After(remoteTS) {
		return &VersionedState{
			Key:     conflict.Key,
			Value:   conflict.LocalState.Value,
			Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
			TTL:     conflict.LocalState.TTL,
			Meta:    r.mergeCRDTMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta, localTS, remoteTS),
		}, nil
	}
	
	return &VersionedState{
		Key:     conflict.Key,
		Value:   conflict.RemoteState.Value,
		Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
		TTL:     conflict.RemoteState.TTL,
		Meta:    r.mergeCRDTMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta, localTS, remoteTS),
	}, nil
}

// resolveCRDTMerge resolves conflict using CRDT merge strategy
func (r *ConflictResolverImpl) resolveCRDTMerge(conflict *Conflict) (*VersionedState, error) {
	// For simple values, use last-write-wins
	// For complex values (maps, sets, counters), perform actual CRDT merge
	
	// Check if values are mergeable
	if r.isMergeableValue(conflict.LocalState.Value, conflict.RemoteState.Value) {
		mergedValue, err := r.mergeValues(conflict.LocalState.Value, conflict.RemoteState.Value)
		if err != nil {
			r.logger.Error("CRDT merge failed, falling back to LWW",
				zap.String("key", conflict.Key),
				zap.Error(err))
			return r.resolveCRDTLastWriterWins(conflict)
		}
		
		return &VersionedState{
			Key:     conflict.Key,
			Value:   mergedValue,
			Version:  max(conflict.LocalState.Version, conflict.RemoteState.Version) + 1,
			TTL:     r.mergeTTL(conflict.LocalState.TTL, conflict.RemoteState.TTL),
			Meta:    r.createCRDTMergeMeta(conflict.LocalState.Meta, conflict.RemoteState.Meta),
		}, nil
	}
	
	// Non-mergeable, fall back to LWW
	return r.resolveCRDTLastWriterWins(conflict)
}

// Helper methods

func (r *ConflictResolverImpl) getStateTimestamp(state *VersionedState) time.Time {
	if state.Meta != nil {
		if ts, ok := state.Meta["timestamp"].(time.Time); ok {
			return ts
		}
		if ts, ok := state.Meta["created_at"].(time.Time); ok {
			return ts
		}
	}
	return time.Time{} // Zero time if not found
}

func (r *ConflictResolverImpl) getVectorClock(state *VersionedState) map[string]uint64 {
	if state.Meta == nil {
		return make(map[string]uint64)
	}
	
	if vc, ok := state.Meta["vector_clock"].(map[string]uint64); ok {
		return vc
	}
	
	return make(map[string]uint64)
}

func (r *ConflictResolverImpl) compareVectorClocks(local, remote map[string]uint64) string {
	localGreater := false
	remoteGreater := false
	
	// Get all unique keys
	allKeys := make(map[string]bool)
	for k := range local {
		allKeys[k] = true
	}
	for k := range remote {
		allKeys[k] = true
	}
	
	// Compare each component
	for key := range allKeys {
		localVal := local[key]
		remoteVal := remote[key]
		
		if localVal > remoteVal {
			localGreater = true
		} else if remoteVal > localVal {
			remoteGreater = true
		}
	}
	
	if localGreater && !remoteGreater {
		return "local_greater"
	} else if remoteGreater && !localGreater {
		return "remote_greater"
	} else if localGreater && remoteGreater {
		return "concurrent"
	}
	
	return "equal"
}

func (r *ConflictResolverImpl) createResolvedState(winner, loser *VersionedState, reason string) *VersionedState {
	mergedMeta := r.mergeMeta(winner.Meta, loser.Meta)
	mergedMeta["resolution_reason"] = reason
	mergedMeta["merged_at"] = time.Now()
	
	return &VersionedState{
		Key:     winner.Key,
		Value:   winner.Value,
		Version:  max(winner.Version, loser.Version) + 1,
		TTL:     winner.TTL,
		Meta:    mergedMeta,
	}
}

func (r *ConflictResolverImpl) ensureTimestamp(state *VersionedState) time.Time {
	ts := r.getStateTimestamp(state)
	if ts.IsZero() {
		// Create timestamp if not exists
		if state.Meta == nil {
			state.Meta = make(map[string]interface{})
		}
		ts = time.Now()
		state.Meta["timestamp"] = ts
	}
	return ts
}

func (r *ConflictResolverImpl) isMergeableValue(value interface{}) bool {
	// Check if value is a type that can be merged
	switch v := value.(type) {
	case map[string]interface{}:
		return true
	case []interface{}:
		return true
	case map[string]string:
		return true
	case map[string]int:
		return true
	default:
		return false
	}
}

func (r *ConflictResolverImpl) mergeValues(local, remote interface{}) (interface{}, error) {
	// Implement actual CRDT merge logic based on value type
	switch l := local.(type) {
	case map[string]interface{}:
		if r, ok := remote.(map[string]interface{}); ok {
			return r.mergeMaps(l, r), nil
		}
	case []interface{}:
		if r, ok := remote.([]interface{}); ok {
			return r.mergeArrays(l, r), nil
		}
	}
	
	// Cannot merge, return error
	return nil, fmt.Errorf("values are not mergeable")
}

func (r *ConflictResolverImpl) mergeMaps(local, remote map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Copy local
	for k, v := range local {
		merged[k] = v
	}
	
	// Merge remote
	for k, v := range remote {
		if existing, exists := merged[k]; exists {
			// If both values are maps, merge recursively
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if remoteMap, ok := v.(map[string]interface{}); ok {
					merged[k] = r.mergeMaps(existingMap, remoteMap)
					continue
				}
			}
		}
		merged[k] = v
	}
	
	return merged
}

func (r *ConflictResolverImpl) mergeArrays(local, remote []interface{}) []interface{} {
	// For arrays, concatenate unique values
	merged := make([]interface{}, 0, len(local)+len(remote))
	seen := make(map[string]bool)
	
	// Add local items
	for _, item := range local {
		key := fmt.Sprintf("%v", item)
		if !seen[key] {
			merged = append(merged, item)
			seen[key] = true
		}
	}
	
	// Add remote items
	for _, item := range remote {
		key := fmt.Sprintf("%v", item)
		if !seen[key] {
			merged = append(merged, item)
			seen[key] = true
		}
	}
	
	return merged
}

func (r *ConflictResolverImpl) mergeTTL(local, remote *time.Time) *time.Time {
	if local == nil {
		return remote
	}
	if remote == nil {
		return local
	}
	
	// Return later TTL (longer retention)
	if local.After(*remote) {
		return local
	}
	return remote
}

func (r *ConflictResolverImpl) mergeMeta(local, remote map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	
	// Copy local
	for k, v := range local {
		merged[k] = v
	}
	
	// Copy remote
	for k, v := range remote {
		merged[k] = v
	}
	
	return merged
}

func (r *ConflictResolverImpl) mergeCRDTMeta(local, remote map[string]interface{}, localTS, remoteTS time.Time) map[string]interface{} {
	merged := r.mergeMeta(local, remote)
	
	// Add CRDT-specific metadata
	merged["crdt_strategy"] = "last-writer-wins"
	merged["local_timestamp"] = localTS
	merged["remote_timestamp"] = remoteTS
	merged["winner_timestamp"] = max(localTS, remoteTS)
	
	return merged
}

func (r *ConflictResolverImpl) createCRDTMergeMeta(local, remote map[string]interface{}) map[string]interface{} {
	merged := r.mergeMeta(local, remote)
	
	// Add CRDT merge metadata
	merged["crdt_strategy"] = "merge"
	merged["merge_timestamp"] = time.Now()
	merged["merge_node_id"] = r.config.NodeID
	
	return merged
}

// GetStrategy returns current conflict resolution strategy
func (r *ConflictResolverImpl) GetStrategy() ConflictResolutionStrategy {
	return r.config.DefaultStrategy
}

// SetStrategy sets conflict resolution strategy
func (r *ConflictResolverImpl) SetStrategy(strategy ConflictResolutionStrategy) error {
	// Validate strategy
	validStrategies := []ConflictResolutionStrategy{
		LastWriteWins,
		FirstWriteWins,
		VectorClock,
		CRDTLastWriterWins,
		CRDTMerge,
	}
	
	valid := false
	for _, s := range validStrategies {
		if s == strategy {
			valid = true
			break
		}
	}
	
	if !valid {
		return fmt.Errorf("invalid conflict resolution strategy: %s", strategy)
	}
	
	r.mu.Lock()
	r.config.DefaultStrategy = strategy
	r.mu.Unlock()
	
	r.logger.Info("Conflict resolution strategy changed",
		zap.String("old_strategy", string(r.config.DefaultStrategy)),
		zap.String("new_strategy", string(strategy)))
	
	return nil
}

// GetConflictStats returns conflict resolution statistics
func (r *ConflictResolverImpl) GetConflictStats() ConflictStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a copy of stats
	stats := *r.stats
	stats.StrategyCounts = make(map[string]int64)
	for k, v := range r.stats.StrategyCounts {
		stats.StrategyCounts[k] = v
	}
	
	return stats
}

// updateStats updates conflict resolution statistics
func (r *ConflictResolverImpl) updateStats(strategy ConflictResolutionStrategy, resolution *VersionedState, err error, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if err != nil {
		r.stats.FailedResolutions++
	} else {
		r.stats.ResolvedConflicts++
	}
	
	// Update strategy count
	r.stats.StrategyCounts[string(strategy)]++
	
	// Update average resolution time
	if r.stats.TotalConflicts > 0 {
		totalTime := r.stats.AverageResolution.Nanoseconds() * (r.stats.TotalConflicts - 1)
		totalTime += duration.Nanoseconds()
		r.stats.AverageResolution = time.Duration(totalTime / r.stats.TotalConflicts)
	} else {
		r.stats.AverageResolution = duration
	}
	
	// Update last resolution time
	r.stats.LastResolution = &time.Time{}
	*r.stats.LastResolution = time.Now()
}

// Helper functions
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func max(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}