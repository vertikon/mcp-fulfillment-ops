// Package cache provides multi-level caching (L1: memory, L2: Redis, L3: BadgerDB)
package cache

import (
	"context"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Warmer warms up the cache with frequently accessed data
type Warmer struct {
	cache Cache
}

// NewWarmer creates a new cache warmer
func NewWarmer(cache Cache) *Warmer {
	return &Warmer{cache: cache}
}

// WarmUp warms up the cache with the provided key-value pairs
func (w *Warmer) WarmUp(ctx context.Context, data map[string][]byte, ttl time.Duration) error {
	logger.Info("Starting cache warm-up", zap.Int("items", len(data)))

	for key, value := range data {
		if err := w.cache.Set(ctx, key, value, ttl); err != nil {
			logger.Warn("Failed to warm cache key",
				zap.String("key", key),
				zap.Error(err),
			)
			continue
		}
	}

	logger.Info("Cache warm-up completed", zap.Int("items", len(data)))
	return nil
}

// WarmUpFunc warms up the cache using a function that generates key-value pairs
func (w *Warmer) WarmUpFunc(ctx context.Context, generator func() (map[string][]byte, error), ttl time.Duration) error {
	data, err := generator()
	if err != nil {
		return err
	}

	return w.WarmUp(ctx, data, ttl)
}
