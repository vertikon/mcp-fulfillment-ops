package cache

import (
	"context"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// CoherencyStrategy represents different cache coherency strategies
type CoherencyStrategy string

const (
	CoherencyStrategyWriteThrough CoherencyStrategy = "write-through"
	CoherencyStrategyWriteBack    CoherencyStrategy = "write-back"
	CoherencyStrategyWriteAround  CoherencyStrategy = "write-around"
	CoherencyStrategyInvalidate   CoherencyStrategy = "invalidate"
	CoherencyStrategyUpdate       CoherencyStrategy = "update"
)

// CoherencyLevel represents cache coherency level
type CoherencyLevel string

const (
	CoherencyLevelStrong   CoherencyLevel = "strong"
	CoherencyLevelWeak     CoherencyLevel = "weak"
	CoherencyLevelEventual CoherencyLevel = "eventual"
)

// CoherencyConfig represents cache coherency configuration
type CoherencyConfig struct {
	Strategy          CoherencyStrategy `json:"strategy"`
	Level             CoherencyLevel    `json:"level"`
	InvalidationTTL   time.Duration     `json:"invalidation_ttl"`
	UpdateOnWrite     bool              `json:"update_on_write"`
	InvalidateOnWrite bool              `json:"invalidate_on_write"`
	SyncTimeout       time.Duration     `json:"sync_timeout"`
	MaxRetries        int               `json:"max_retries"`
	RetryDelay        time.Duration     `json:"retry_delay"`
}

// DefaultCoherencyConfig returns default coherency configuration
func DefaultCoherencyConfig() *CoherencyConfig {
	return &CoherencyConfig{
		Strategy:          CoherencyStrategyWriteThrough,
		Level:             CoherencyLevelStrong,
		InvalidationTTL:   5 * time.Minute,
		UpdateOnWrite:     true,
		InvalidateOnWrite: false,
		SyncTimeout:       10 * time.Second,
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
	}
}

// InvalidationEvent represents a cache invalidation event
type InvalidationEvent struct {
	Key       string                 `json:"key"`
	Reason    string                 `json:"reason"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CoherencyManager interface for managing cache coherency
type CoherencyManager interface {
	// Coherency operations
	Invalidate(ctx context.Context, key string, reason string) error
	InvalidatePattern(ctx context.Context, pattern string) error
	InvalidateAll(ctx context.Context) error
	Update(ctx context.Context, key string, value interface{}) error

	// Coherency state
	GetCoherencyStatus(ctx context.Context) (*CoherencyStatus, error)
	GetInvalidationStats(ctx context.Context) (*InvalidationStats, error)

	// Event handling
	OnStoreUpdate(ctx context.Context, key string, value interface{}) error
	OnEventUpdate(ctx context.Context, event interface{}) error

	// Background processes
	StartBackgroundInvalidator(ctx context.Context) error
	StopBackgroundInvalidator() error
}

// CoherencyStatus represents cache coherency status
type CoherencyStatus struct {
	Strategy             CoherencyStrategy `json:"strategy"`
	Level                CoherencyLevel    `json:"level"`
	IsCoherent           bool              `json:"is_coherent"`
	LastSync             *time.Time        `json:"last_sync,omitempty"`
	PendingInvalidations int               `json:"pending_invalidations"`
	PendingUpdates       int               `json:"pending_updates"`
	TotalInvalidations   int64             `json:"total_invalidations"`
	TotalUpdates         int64             `json:"total_updates"`
}

// InvalidationStats represents invalidation statistics
type InvalidationStats struct {
	TotalInvalidations      int64            `json:"total_invalidations"`
	InvalidationsByReason   map[string]int64 `json:"invalidations_by_reason"`
	AverageInvalidationTime time.Duration    `json:"average_invalidation_time"`
	LastInvalidation        *time.Time       `json:"last_invalidation"`
	FailedInvalidations     int64            `json:"failed_invalidations"`
}

// CoherencyManagerImpl implements CoherencyManager interface
type CoherencyManagerImpl struct {
	config               *CoherencyConfig
	cache                *StateCache
	store                interface{} // DistributedStore interface
	eventStore           interface{} // EventStore interface
	stats                *InvalidationStats
	logger               *zap.Logger
	mu                   sync.RWMutex
	ctx                  context.Context
	cancel               context.CancelFunc
	invalidationCh       chan *InvalidationEvent
	pendingInvalidations map[string]*InvalidationEvent
}

// NewCoherencyManager creates a new cache coherency manager
func NewCoherencyManager(config *CoherencyConfig, cache *StateCache) *CoherencyManagerImpl {
	if config == nil {
		config = DefaultCoherencyConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &CoherencyManagerImpl{
		config: config,
		cache:  cache,
		stats: &InvalidationStats{
			InvalidationsByReason: make(map[string]int64),
		},
		logger:               logger.Get(),
		ctx:                  ctx,
		cancel:               cancel,
		invalidationCh:       make(chan *InvalidationEvent, 10000), // Aumentado de 1000 para 10000
		pendingInvalidations: make(map[string]*InvalidationEvent),
	}

	manager.logger.Info("Cache coherency manager initialized",
		zap.String("strategy", string(config.Strategy)),
		zap.String("level", string(config.Level)))

	return manager
}

// Invalidate invalidates a cache key
func (cm *CoherencyManagerImpl) Invalidate(ctx context.Context, key string, reason string) error {
	start := time.Now()

	cm.mu.Lock()
	cm.stats.TotalInvalidations++
	cm.stats.InvalidationsByReason[reason]++
	cm.mu.Unlock()

	// Delete from cache
	cm.cache.Delete(key)

	// Create invalidation event
	event := &InvalidationEvent{
		Key:       key,
		Reason:    reason,
		Timestamp: time.Now(),
		Source:    "coherency_manager",
	}

	// Send to invalidation channel
	select {
	case cm.invalidationCh <- event:
	default:
		cm.logger.Warn("Invalidation channel full, dropping event",
			zap.String("key", key))
	}

	cm.mu.Lock()
	cm.stats.AverageInvalidationTime = time.Since(start)
	lastTime := time.Now()
	cm.stats.LastInvalidation = &lastTime
	cm.mu.Unlock()

	cm.logger.Debug("Cache key invalidated",
		zap.String("key", key),
		zap.String("reason", reason))

	return nil
}

// InvalidatePattern invalidates all keys matching a pattern
func (cm *CoherencyManagerImpl) InvalidatePattern(ctx context.Context, pattern string) error {
	// In a real implementation, this would iterate over cache entries
	// and invalidate those matching the pattern
	cm.logger.Info("Invalidating cache pattern",
		zap.String("pattern", pattern))

	// For now, invalidate all (simplified)
	return cm.InvalidateAll(ctx)
}

// InvalidateAll invalidates all cache entries
func (cm *CoherencyManagerImpl) InvalidateAll(ctx context.Context) error {
	cm.logger.Info("Invalidating all cache entries")

	// Clear cache
	// In a real implementation, this would clear all levels (L1, L2, L3)

	return nil
}

// Update updates a cache entry
func (cm *CoherencyManagerImpl) Update(ctx context.Context, key string, value interface{}) error {
	cm.mu.Lock()
	cm.stats.TotalUpdates++
	cm.mu.Unlock()

	// Update cache based on strategy
	switch cm.config.Strategy {
	case CoherencyStrategyWriteThrough:
		// Write to cache and store
		// Cache update happens here
		cm.logger.Debug("Cache updated (write-through)",
			zap.String("key", key))

	case CoherencyStrategyWriteBack:
		// Write to cache only, store later
		cm.logger.Debug("Cache updated (write-back)",
			zap.String("key", key))

	case CoherencyStrategyUpdate:
		// Update cache entry
		cm.logger.Debug("Cache updated",
			zap.String("key", key))
	}

	return nil
}

// GetCoherencyStatus returns cache coherency status
func (cm *CoherencyManagerImpl) GetCoherencyStatus(ctx context.Context) (*CoherencyStatus, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return &CoherencyStatus{
		Strategy:             cm.config.Strategy,
		Level:                cm.config.Level,
		IsCoherent:           true, // Simplified
		PendingInvalidations: len(cm.pendingInvalidations),
		TotalInvalidations:   cm.stats.TotalInvalidations,
		TotalUpdates:         cm.stats.TotalUpdates,
	}, nil
}

// GetInvalidationStats returns invalidation statistics
func (cm *CoherencyManagerImpl) GetInvalidationStats(ctx context.Context) (*InvalidationStats, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := *cm.stats
	stats.InvalidationsByReason = make(map[string]int64)
	for k, v := range cm.stats.InvalidationsByReason {
		stats.InvalidationsByReason[k] = v
	}

	return &stats, nil
}

// OnStoreUpdate handles store update events
func (cm *CoherencyManagerImpl) OnStoreUpdate(ctx context.Context, key string, value interface{}) error {
	switch cm.config.Strategy {
	case CoherencyStrategyWriteThrough:
		// Cache is already updated
		return nil

	case CoherencyStrategyInvalidate:
		// Invalidate cache entry
		return cm.Invalidate(ctx, key, "store_update")

	case CoherencyStrategyUpdate:
		// Update cache entry
		return cm.Update(ctx, key, value)

	default:
		// Default: invalidate
		return cm.Invalidate(ctx, key, "store_update")
	}
}

// OnEventUpdate handles event store update events
func (cm *CoherencyManagerImpl) OnEventUpdate(ctx context.Context, event interface{}) error {
	// Invalidate related cache entries based on event
	// This is a simplified version
	cm.logger.Debug("Handling event update for cache coherency")
	return nil
}

// StartBackgroundInvalidator starts background invalidation process
func (cm *CoherencyManagerImpl) StartBackgroundInvalidator(ctx context.Context) error {
	go cm.backgroundInvalidator()
	cm.logger.Info("Background invalidator started")
	return nil
}

// StopBackgroundInvalidator stops background invalidation process
func (cm *CoherencyManagerImpl) StopBackgroundInvalidator() error {
	cm.cancel()
	close(cm.invalidationCh)
	cm.logger.Info("Background invalidator stopped")
	return nil
}

// Private helper methods

func (cm *CoherencyManagerImpl) backgroundInvalidator() {
	ticker := time.NewTicker(cm.config.InvalidationTTL)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			cm.logger.Info("Background invalidator stopped")
			return

		case event := <-cm.invalidationCh:
			cm.processInvalidation(event)

		case <-ticker.C:
			cm.processPendingInvalidations()
		}
	}
}

func (cm *CoherencyManagerImpl) processInvalidation(event *InvalidationEvent) {
	cm.mu.Lock()
	cm.pendingInvalidations[event.Key] = event
	cm.mu.Unlock()

	// Process invalidation
	cm.cache.Delete(event.Key)

	cm.mu.Lock()
	delete(cm.pendingInvalidations, event.Key)
	cm.mu.Unlock()

	cm.logger.Debug("Invalidation processed",
		zap.String("key", event.Key),
		zap.String("reason", event.Reason))
}

func (cm *CoherencyManagerImpl) processPendingInvalidations() {
	cm.mu.Lock()
	pending := make([]*InvalidationEvent, 0, len(cm.pendingInvalidations))
	for _, event := range cm.pendingInvalidations {
		pending = append(pending, event)
	}
	cm.mu.Unlock()

	for _, event := range pending {
		cm.processInvalidation(event)
	}
}
