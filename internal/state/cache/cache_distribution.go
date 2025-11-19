package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// DistributionStrategy represents different distribution strategies
type DistributionStrategy string

const (
	DistributionStrategyPubSub    DistributionStrategy = "pub-sub"
	DistributionStrategyGossip    DistributionStrategy = "gossip"
	DistributionStrategyBroadcast DistributionStrategy = "broadcast"
)

// DistributionConfig represents cache distribution configuration
type DistributionConfig struct {
	Strategy           DistributionStrategy `json:"strategy"`
	EnableDistribution bool                 `json:"enable_distribution"`
	Nodes              []string             `json:"nodes"`
	NodeID             string               `json:"node_id"`
	Channel            string               `json:"channel"`
	TTL                time.Duration        `json:"ttl"`
	MaxRetries         int                  `json:"max_retries"`
	RetryDelay         time.Duration        `json:"retry_delay"`
}

// DefaultDistributionConfig returns default distribution configuration
func DefaultDistributionConfig() *DistributionConfig {
	return &DistributionConfig{
		Strategy:           DistributionStrategyPubSub,
		EnableDistribution: false,
		Nodes:              []string{},
		NodeID:             fmt.Sprintf("node-%d", time.Now().Unix()),
		Channel:            "cache-updates",
		TTL:                5 * time.Minute,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
	}
}

// DistributionMessage represents a cache distribution message
type DistributionMessage struct {
	Type      string                 `json:"type"` // invalidate, update, clear
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value,omitempty"`
	Level     CacheLevel             `json:"level"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CacheDistribution interface for distributing cache updates
type CacheDistribution interface {
	// Distribution operations
	PublishInvalidation(ctx context.Context, key string, level CacheLevel) error
	PublishUpdate(ctx context.Context, key string, value interface{}, level CacheLevel) error
	PublishClear(ctx context.Context, level CacheLevel) error

	// Subscription
	Subscribe(ctx context.Context, handler DistributionHandler) error
	Unsubscribe(ctx context.Context) error

	// Statistics
	GetDistributionStats(ctx context.Context) (*DistributionStats, error)
}

// DistributionHandler handles distribution messages
type DistributionHandler interface {
	HandleInvalidation(ctx context.Context, key string, level CacheLevel) error
	HandleUpdate(ctx context.Context, key string, value interface{}, level CacheLevel) error
	HandleClear(ctx context.Context, level CacheLevel) error
}

// DistributionStats represents distribution statistics
type DistributionStats struct {
	MessagesPublished int64      `json:"messages_published"`
	MessagesReceived  int64      `json:"messages_received"`
	InvalidationsSent int64      `json:"invalidations_sent"`
	UpdatesSent       int64      `json:"updates_sent"`
	ClearsSent        int64      `json:"clears_sent"`
	FailedPublishes   int64      `json:"failed_publishes"`
	LastPublish       *time.Time `json:"last_publish,omitempty"`
}

// CacheDistributionImpl implements CacheDistribution interface
type CacheDistributionImpl struct {
	config      *DistributionConfig
	cache       *StateCacheImpl
	handler     DistributionHandler
	subscribers []chan *DistributionMessage
	stats       *DistributionStats
	logger      *zap.Logger
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewCacheDistribution creates a new cache distribution implementation
func NewCacheDistribution(config *DistributionConfig, cache *StateCacheImpl) *CacheDistributionImpl {
	if config == nil {
		config = DefaultDistributionConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	distribution := &CacheDistributionImpl{
		config:      config,
		cache:       cache,
		subscribers: make([]chan *DistributionMessage, 0),
		stats:       &DistributionStats{},
		logger:      logger.Get(),
		ctx:         ctx,
		cancel:      cancel,
	}

	if config.EnableDistribution {
		go distribution.backgroundProcessor()
	}

	distribution.logger.Info("Cache distribution initialized",
		zap.String("strategy", string(config.Strategy)),
		zap.Bool("enabled", config.EnableDistribution))

	return distribution
}

// PublishInvalidation publishes an invalidation message
func (cd *CacheDistributionImpl) PublishInvalidation(ctx context.Context, key string, level CacheLevel) error {
	if !cd.config.EnableDistribution {
		return nil
	}

	message := &DistributionMessage{
		Type:      "invalidate",
		Key:       key,
		Level:     level,
		Timestamp: time.Now(),
		Source:    cd.config.NodeID,
	}

	err := cd.publishMessage(ctx, message)
	if err != nil {
		cd.mu.Lock()
		cd.stats.FailedPublishes++
		cd.mu.Unlock()
		return err
	}

	cd.mu.Lock()
	cd.stats.MessagesPublished++
	cd.stats.InvalidationsSent++
	lastTime := time.Now()
	cd.stats.LastPublish = &lastTime
	cd.mu.Unlock()

	cd.logger.Debug("Invalidation published",
		zap.String("key", key),
		zap.String("level", string(level)))

	return nil
}

// PublishUpdate publishes an update message
func (cd *CacheDistributionImpl) PublishUpdate(ctx context.Context, key string, value interface{}, level CacheLevel) error {
	if !cd.config.EnableDistribution {
		return nil
	}

	message := &DistributionMessage{
		Type:      "update",
		Key:       key,
		Value:     value,
		Level:     level,
		Timestamp: time.Now(),
		Source:    cd.config.NodeID,
	}

	err := cd.publishMessage(ctx, message)
	if err != nil {
		cd.mu.Lock()
		cd.stats.FailedPublishes++
		cd.mu.Unlock()
		return err
	}

	cd.mu.Lock()
	cd.stats.MessagesPublished++
	cd.stats.UpdatesSent++
	lastTime := time.Now()
	cd.stats.LastPublish = &lastTime
	cd.mu.Unlock()

	return nil
}

// PublishClear publishes a clear message
func (cd *CacheDistributionImpl) PublishClear(ctx context.Context, level CacheLevel) error {
	if !cd.config.EnableDistribution {
		return nil
	}

	message := &DistributionMessage{
		Type:      "clear",
		Level:     level,
		Timestamp: time.Now(),
		Source:    cd.config.NodeID,
	}

	err := cd.publishMessage(ctx, message)
	if err != nil {
		cd.mu.Lock()
		cd.stats.FailedPublishes++
		cd.mu.Unlock()
		return err
	}

	cd.mu.Lock()
	cd.stats.MessagesPublished++
	cd.stats.ClearsSent++
	lastTime := time.Now()
	cd.stats.LastPublish = &lastTime
	cd.mu.Unlock()

	return nil
}

// Subscribe subscribes to distribution messages
func (cd *CacheDistributionImpl) Subscribe(ctx context.Context, handler DistributionHandler) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	cd.handler = handler

	cd.logger.Info("Subscribed to cache distribution")
	return nil
}

// Unsubscribe unsubscribes from distribution messages
func (cd *CacheDistributionImpl) Unsubscribe(ctx context.Context) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	cd.handler = nil

	cd.logger.Info("Unsubscribed from cache distribution")
	return nil
}

// GetDistributionStats returns distribution statistics
func (cd *CacheDistributionImpl) GetDistributionStats(ctx context.Context) (*DistributionStats, error) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	stats := *cd.stats
	return &stats, nil
}

// Private helper methods

func (cd *CacheDistributionImpl) publishMessage(ctx context.Context, message *DistributionMessage) error {
	switch cd.config.Strategy {
	case DistributionStrategyPubSub:
		return cd.publishPubSub(ctx, message)
	case DistributionStrategyGossip:
		return cd.publishGossip(ctx, message)
	case DistributionStrategyBroadcast:
		return cd.publishBroadcast(ctx, message)
	default:
		return cd.publishPubSub(ctx, message)
	}
}

func (cd *CacheDistributionImpl) publishPubSub(ctx context.Context, message *DistributionMessage) error {
	// In a real implementation, this would use NATS/Kafka/RabbitMQ
	// For now, simulate by notifying local subscribers
	cd.mu.RLock()
	subscribers := make([]chan *DistributionMessage, len(cd.subscribers))
	copy(subscribers, cd.subscribers)
	cd.mu.RUnlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- message:
		default:
			cd.logger.Warn("Subscriber channel full, dropping message")
		}
	}

	return nil
}

func (cd *CacheDistributionImpl) publishGossip(ctx context.Context, message *DistributionMessage) error {
	// In a real implementation, this would use gossip protocol
	return cd.publishPubSub(ctx, message)
}

func (cd *CacheDistributionImpl) publishBroadcast(ctx context.Context, message *DistributionMessage) error {
	// In a real implementation, this would broadcast to all nodes
	return cd.publishPubSub(ctx, message)
}

func (cd *CacheDistributionImpl) backgroundProcessor() {
	messageCh := make(chan *DistributionMessage, 10000) // Aumentado de 1000 para 10000

	// Subscribe to messages
	cd.mu.Lock()
	cd.subscribers = append(cd.subscribers, messageCh)
	cd.mu.Unlock()

	for {
		select {
		case <-cd.ctx.Done():
			cd.logger.Info("Cache distribution background processor stopped")
			return
		case message := <-messageCh:
			cd.handleMessage(message)
		}
	}
}

func (cd *CacheDistributionImpl) handleMessage(message *DistributionMessage) {
	if cd.handler == nil {
		return
	}

	cd.mu.Lock()
	cd.stats.MessagesReceived++
	cd.mu.Unlock()

	ctx := context.Background()

	switch message.Type {
	case "invalidate":
		if err := cd.handler.HandleInvalidation(ctx, message.Key, message.Level); err != nil {
			cd.logger.Error("Failed to handle invalidation",
				zap.String("key", message.Key),
				zap.Error(err))
		}
	case "update":
		if err := cd.handler.HandleUpdate(ctx, message.Key, message.Value, message.Level); err != nil {
			cd.logger.Error("Failed to handle update",
				zap.String("key", message.Key),
				zap.Error(err))
		}
	case "clear":
		if err := cd.handler.HandleClear(ctx, message.Level); err != nil {
			cd.logger.Error("Failed to handle clear",
				zap.String("level", string(message.Level)),
				zap.Error(err))
		}
	}
}

// Close closes the distribution
func (cd *CacheDistributionImpl) Close() error {
	cd.cancel()
	cd.mu.Lock()
	defer cd.mu.Unlock()

	for _, subscriber := range cd.subscribers {
		close(subscriber)
	}
	cd.subscribers = []chan *DistributionMessage{}

	cd.logger.Info("Cache distribution closed")
	return nil
}
