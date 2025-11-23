// Package cache provides multi-level caching (L1: memory, L2: Redis, L3: BadgerDB)
package cache

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// InvalidationStrategy defines cache invalidation strategies
type InvalidationStrategy int

const (
	// StrategyImmediate invalidates immediately
	StrategyImmediate InvalidationStrategy = iota
	// StrategyTTL invalidates based on TTL
	StrategyTTL
	// StrategyLRU invalidates least recently used items
	StrategyLRU
)

// Invalidator manages cache invalidation
type Invalidator struct {
	cache      Cache
	strategy   InvalidationStrategy
	mu         sync.RWMutex
	patterns   []string
	callbacks  []func(string)
	keyTracker KeyTracker // Tracks keys for pattern matching
}

// KeyTracker tracks cache keys for pattern-based invalidation
type KeyTracker interface {
	GetKeys() []string
	AddKey(key string)
	RemoveKey(key string)
}

// SimpleKeyTracker is a basic implementation of KeyTracker
type SimpleKeyTracker struct {
	keys map[string]bool
	mu   sync.RWMutex
}

// NewSimpleKeyTracker creates a new key tracker
func NewSimpleKeyTracker() *SimpleKeyTracker {
	return &SimpleKeyTracker{
		keys: make(map[string]bool),
	}
}

// GetKeys returns all tracked keys
func (kt *SimpleKeyTracker) GetKeys() []string {
	kt.mu.RLock()
	defer kt.mu.RUnlock()

	keys := make([]string, 0, len(kt.keys))
	for key := range kt.keys {
		keys = append(keys, key)
	}
	return keys
}

// AddKey adds a key to the tracker
func (kt *SimpleKeyTracker) AddKey(key string) {
	kt.mu.Lock()
	defer kt.mu.Unlock()
	kt.keys[key] = true
}

// RemoveKey removes a key from the tracker
func (kt *SimpleKeyTracker) RemoveKey(key string) {
	kt.mu.Lock()
	defer kt.mu.Unlock()
	delete(kt.keys, key)
}

// NewInvalidator creates a new cache invalidator
func NewInvalidator(cache Cache, strategy InvalidationStrategy) *Invalidator {
	return &Invalidator{
		cache:      cache,
		strategy:   strategy,
		keyTracker: NewSimpleKeyTracker(),
	}
}

// Invalidate invalidates a specific key
func (i *Invalidator) Invalidate(ctx context.Context, key string) error {
	logger.Debug("Invalidating cache key", zap.String("key", key))

	if err := i.cache.Delete(ctx, key); err != nil {
		return err
	}

	// Remove from key tracker
	if i.keyTracker != nil {
		i.keyTracker.RemoveKey(key)
	}

	// Call callbacks
	i.mu.RLock()
	for _, cb := range i.callbacks {
		cb(key)
	}
	i.mu.RUnlock()

	return nil
}

// InvalidatePattern invalidates keys matching a pattern
// Supports:
//   - Prefix matching: "prefix:*" or "prefix*"
//   - Suffix matching: "*suffix" or "*:suffix"
//   - Exact match: "exact"
//   - Glob patterns: "path/*/key" (using filepath.Match)
func (i *Invalidator) InvalidatePattern(ctx context.Context, pattern string) error {
	logger.Debug("Invalidating cache pattern", zap.String("pattern", pattern))

	if i.keyTracker == nil {
		logger.Warn("Key tracker not initialized, cannot invalidate by pattern")
		return nil
	}

	keys := i.keyTracker.GetKeys()
	matchedKeys := make([]string, 0)

	// Normalize pattern
	normalizedPattern := strings.TrimSpace(pattern)

	// Check if pattern uses wildcards
	hasWildcard := strings.Contains(normalizedPattern, "*") || strings.Contains(normalizedPattern, "?")

	for _, key := range keys {
		var matches bool

		if hasWildcard {
			// Use filepath.Match for glob-style patterns
			matched, err := filepath.Match(normalizedPattern, key)
			if err == nil && matched {
				matches = true
			}
		} else {
			// Exact match
			matches = key == normalizedPattern
		}

		// Also check prefix/suffix patterns
		if !matches {
			if strings.HasPrefix(normalizedPattern, "*") {
				// Suffix match
				suffix := strings.TrimPrefix(normalizedPattern, "*")
				matches = strings.HasSuffix(key, suffix)
			} else if strings.HasSuffix(normalizedPattern, "*") {
				// Prefix match
				prefix := strings.TrimSuffix(normalizedPattern, "*")
				matches = strings.HasPrefix(key, prefix)
			}
		}

		if matches {
			matchedKeys = append(matchedKeys, key)
		}
	}

	// Invalidate matched keys
	invalidatedCount := 0
	for _, key := range matchedKeys {
		if err := i.Invalidate(ctx, key); err != nil {
			logger.Warn("Failed to invalidate key", zap.String("key", key), zap.Error(err))
			continue
		}
		invalidatedCount++
	}

	logger.Info("Pattern invalidation completed",
		zap.String("pattern", pattern),
		zap.Int("matched", len(matchedKeys)),
		zap.Int("invalidated", invalidatedCount))

	return nil
}

// InvalidateAll invalidates all cache entries
func (i *Invalidator) InvalidateAll(ctx context.Context) error {
	logger.Info("Invalidating all cache entries")

	if err := i.cache.Clear(ctx); err != nil {
		return err
	}

	return nil
}

// RegisterCallback registers a callback to be called when a key is invalidated
func (i *Invalidator) RegisterCallback(callback func(string)) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.callbacks = append(i.callbacks, callback)
}

// StartTTLInvalidation starts TTL-based invalidation
// This performs periodic cleanup of expired entries and updates key tracker
func (i *Invalidator) StartTTLInvalidation(ctx context.Context, interval time.Duration) {
	logger.Info("Starting TTL-based invalidation", zap.Duration("interval", interval))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("TTL invalidation stopped")
				return
			case <-ticker.C:
				// Perform TTL-based cleanup
				i.performTTLCleanup(ctx)
			}
		}
	}()
}

// performTTLCleanup performs cleanup of expired cache entries
func (i *Invalidator) performTTLCleanup(ctx context.Context) {
	if i.keyTracker == nil {
		return
	}

	keys := i.keyTracker.GetKeys()
	cleanedCount := 0

	// Check each key - if Get fails (cache miss due to expiration), remove from tracker
	for _, key := range keys {
		_, err := i.cache.Get(ctx, key)
		if err != nil {
			// Key expired or doesn't exist, remove from tracker
			i.keyTracker.RemoveKey(key)
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		logger.Debug("TTL cleanup completed",
			zap.Int("keys_checked", len(keys)),
			zap.Int("keys_cleaned", cleanedCount))
	}
}

// TrackKey adds a key to the tracker (should be called when setting cache values)
func (i *Invalidator) TrackKey(key string) {
	if i.keyTracker != nil {
		i.keyTracker.AddKey(key)
	}
}
