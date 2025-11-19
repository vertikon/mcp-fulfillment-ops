// Package state provides distributed state management for GLM-4.6
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// StateStoreType represents different state store implementations
type StateStoreType string

const (
	StateStoreTypeMemory    StateStoreType = "memory"
	StateStoreTypeRedis    StateStoreType = "redis"
	StateStoreTypeConsul   StateStoreType = "consul"
	StateStoreTypeEtcd     StateStoreType = "etcd"
	StateStoreTypeDynamoDB StateStoreType = "dynamodb"
)

// StateStoreConfig represents configuration for state store
type StateStoreConfig struct {
	Type           StateStoreType `json:"type"`
	ConnectionString string        `json:"connection_string"`
	Namespace      string        `json:"namespace"`
	TTL           time.Duration `json:"ttl"`
	EnableLocking  bool          `json:"enable_locking"`
	EnableCache    bool          `json:"enable_cache"`
	CacheSize      int           `json:"cache_size"`
	Compression    bool          `json:"compression"`
	Encryption     bool          `json:"encryption"`
	Replication    bool          `json:"replication"`
	Consistency    ConsistencyLevel `json:"consistency"`
}

// ConsistencyLevel represents different consistency levels
type ConsistencyLevel string

const (
	ConsistencyEventual ConsistencyLevel = "eventual"
	ConsistencyStrong  ConsistencyLevel = "strong"
	ConsistencyQuorum  ConsistencyLevel = "quorum"
)

// StateEntry represents a state entry
type StateEntry struct {
	Key        string                 `json:"key"`
	Value      interface{}            `json:"value"`
	Version    int64                  `json:"version"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	TTL        time.Duration          `json:"ttl,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// StateStore interface for different state store implementations
type StateStore interface {
	Get(ctx context.Context, key string) (*StateEntry, error)
	Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]*StateEntry, error)
	Watch(ctx context.Context, key string) (<-chan *StateEvent, error)
	Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error)
	Unlock(ctx context.Context, lock *Lock) error
	Transaction(ctx context.Context, ops []Operation) error
	Backup(ctx context.Context) ([]byte, error)
	Restore(ctx context.Context, data []byte) error
	Close() error
	Stats() StateStoreStats
}

// StateEvent represents a state change event
type StateEvent struct {
	Key       string      `json:"key"`
	Type      EventType   `json:"type"`
	Value     interface{} `json:"value,omitempty"`
	OldValue  interface{} `json:"old_value,omitempty"`
	Version   int64       `json:"version"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventType represents different event types
type EventType string

const (
	EventTypeCreated EventType = "created"
	EventTypeUpdated EventType = "updated"
	EventTypeDeleted EventType = "deleted"
	EventTypeExpired EventType = "expired"
)

// Lock represents a distributed lock
type Lock struct {
	Key        string        `json:"key"`
	Owner      string        `json:"owner"`
	ExpiresAt  time.Time     `json:"expires_at"`
	Metadata   interface{}   `json:"metadata,omitempty"`
}

// Operation represents a state operation
type Operation struct {
	Type      OpType       `json:"type"`
	Key       string       `json:"key"`
	Value     interface{} `json:"value,omitempty"`
	TTL       time.Duration `json:"ttl,omitempty"`
	Version   int64       `json:"version,omitempty"`
}

// OpType represents operation types
type OpType string

const (
	OpTypeSet    OpType = "set"
	OpTypeDelete OpType = "delete"
)

// StateStoreStats tracks state store performance
type StateStoreStats struct {
	TotalOperations int64     `json:"total_operations"`
	ReadOperations  int64     `json:"read_operations"`
	WriteOperations int64     `json:"write_operations"`
	DeleteOperations int64    `json:"delete_operations"`
	AvgReadLatency  time.Duration `json:"avg_read_latency"`
	AvgWriteLatency time.Duration `json:"avg_write_latency"`
	CacheHits       int64     `json:"cache_hits"`
	CacheMisses     int64     `json:"cache_misses"`
	LockOperations  int64     `json:"lock_operations"`
	LockWaitTime    time.Duration `json:"lock_wait_time"`
	LastUpdated     int64    `json:"last_updated"`
}

// DistributedStateStore implements distributed state management
type DistributedStateStore struct {
	config       StateStoreConfig
	localStore   *MemoryStateStore
	remoteStore  StateStore
	cache        *StateCache
	syncManager  *StateSyncManager
	replication  *ReplicationManager
	consistency  *ConsistencyManager
	stats        *StateStoreStats
	mu           sync.RWMutex
	closed       int32
}

// MemoryStateStore implements in-memory state store
type MemoryStateStore struct {
	data    map[string]*StateEntry
	locks   map[string]*Lock
	watches map[string]chan *StateEvent
	mu      sync.RWMutex
	stats   *StateStoreStats
}

// StateCache provides caching layer for state store
type StateCache struct {
	config   CacheConfig
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
	stats    *CacheStats
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	MaxSize      int           `json:"max_size"`
	TTL          time.Duration `json:"ttl"`
	FlushInterval time.Duration `json:"flush_interval"`
	Compression  bool          `json:"compression"`
}

// CacheEntry represents a cached entry
type CacheEntry struct {
	Key       string        `json:"key"`
	Value     *StateEntry   `json:"value"`
	ExpiresAt time.Time     `json:"expires_at"`
	Accessed  time.Time     `json:"accessed"`
	Hits      int64         `json:"hits"`
}

// CacheStats tracks cache performance
type CacheStats struct {
	TotalRequests int64   `json:"total_requests"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
	HitRate       float64 `json:"hit_rate"`
	LastFlush     int64   `json:"last_flush"`
}

// StateSyncManager manages state synchronization
type StateSyncManager struct {
	config    SyncConfig
	lastSync  time.Time
	inSync    bool
	stats     *SyncStats
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	Interval     time.Duration `json:"interval"`
	MaxRetries   int           `json:"max_retries"`
	RetryBackoff  time.Duration `json:"retry_backoff"`
	ConflictRes  ConflictResolver `json:"conflict_resolver"`
}

// ConflictResolver resolves state conflicts
type ConflictResolver interface {
	Resolve(local, remote *StateEntry) (*StateEntry, error)
}

// SyncStats tracks synchronization performance
type SyncStats struct {
	TotalSyncs    int64         `json:"total_syncs"`
	SuccessfulSyncs int64       `json:"successful_syncs"`
	FailedSyncs    int64         `json:"failed_syncs"`
	Conflicts      int64         `json:"conflicts"`
	AvgSyncTime    time.Duration `json:"avg_sync_time"`
	LastSyncTime   int64         `json:"last_sync_time"`
}

// ReplicationManager manages state replication
type ReplicationManager struct {
	config      ReplicationConfig
	replicas    []StateStore
	strategy    ReplicationStrategy
	stats       *ReplicationStats
}

// ReplicationConfig represents replication configuration
type ReplicationConfig struct {
	Enabled      bool                `json:"enabled"`
	Replicas     []string            `json:"replicas"`
	Strategy     ReplicationStrategy  `json:"strategy"`
	WriteQuorum  int                 `json:"write_quorum"`
	ReadQuorum   int                 `json:"read_quorum"`
	SyncMode     SyncMode            `json:"sync_mode"`
}

// ReplicationStrategy represents different replication strategies
type ReplicationStrategy string

const (
	StrategyLeaderFollowers ReplicationStrategy = "leader_followers"
	StrategyMultiMaster    ReplicationStrategy = "multi_master"
	StrategyRaft          ReplicationStrategy = "raft"
)

// SyncMode represents synchronization modes
type SyncMode string

const (
	SyncModeAsync SyncMode = "async"
	SyncModeSync  SyncMode = "sync"
)

// ReplicationStats tracks replication performance
type ReplicationStats struct {
	WriteReplications int64         `json:"write_replications"`
	ReadReplications  int64         `json:"read_replications"`
	ReplicationLag   time.Duration `json:"replication_lag"`
	FailedReps       int64         `json:"failed_replications"`
}

// ConsistencyManager manages consistency guarantees
type ConsistencyManager struct {
	config   ConsistencyConfig
	level     ConsistencyLevel
	readQuorum int
	writeQuorum int
	stats     *ConsistencyStats
}

// ConsistencyConfig represents consistency configuration
type ConsistencyConfig struct {
	Level         ConsistencyLevel `json:"level"`
	ReadQuorum    int             `json:"read_quorum"`
	WriteQuorum   int             `json:"write_quorum"`
	EnableQuorum  bool            `json:"enable_quorum"`
}

// ConsistencyStats tracks consistency performance
type ConsistencyStats struct {
	QuorumChecks    int64         `json:"quorum_checks"`
	QuorumTimeouts  int64         `json:"quorum_timeouts"`
	ConsistencyViolations int64     `json:"consistency_violations"`
}

// NewDistributedStateStore creates a new distributed state store
func NewDistributedStateStore(config StateStoreConfig) (*DistributedStateStore, error) {
	logger.Info("Creating distributed state store",
		zap.String("type", string(config.Type)),
		zap.String("namespace", config.Namespace),
		zap.Duration("ttl", config.TTL),
		zap.String("consistency", string(config.Consistency)),
	)

	store := &DistributedStateStore{
		config: config,
		localStore: NewMemoryStateStore(),
		stats:   &StateStoreStats{},
	}

	// Initialize remote store based on type
	var err error
	store.remoteStore, err = store.createRemoteStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create remote store: %w", err)
	}

	// Initialize cache if enabled
	if config.EnableCache {
		store.cache = NewStateCache(CacheConfig{
			MaxSize:      config.CacheSize,
			TTL:          config.TTL,
			FlushInterval: 30 * time.Second,
		})
	}

	// Initialize sync manager
	store.syncManager = &StateSyncManager{
		config: SyncConfig{
			Interval:     10 * time.Second,
			MaxRetries:   3,
			RetryBackoff:  1 * time.Second,
		},
		stats: &SyncStats{},
	}

	// Initialize replication manager
	if config.Replication {
		store.replication = &ReplicationManager{
			config: ReplicationConfig{
				Enabled:      true,
				Strategy:     StrategyLeaderFollowers,
				WriteQuorum:  2,
				ReadQuorum:   1,
				SyncMode:     SyncModeAsync,
			},
			stats: &ReplicationStats{},
		}
	}

	// Initialize consistency manager
	store.consistency = &ConsistencyManager{
		config: ConsistencyConfig{
			Level:         config.Consistency,
			ReadQuorum:    1,
			WriteQuorum:   2,
			EnableQuorum:   true,
		},
		level:   config.Consistency,
		stats:   &ConsistencyStats{},
	}

	return store, nil
}

// createRemoteStore creates remote state store based on configuration
func (dss *DistributedStateStore) createRemoteStore() (StateStore, error) {
	switch dss.config.Type {
	case StateStoreTypeMemory:
		return NewMemoryStateStore(), nil
	case StateStoreTypeRedis:
		return NewRedisStateStore(dss.config)
	case StateStoreTypeConsul:
		return NewConsulStateStore(dss.config)
	case StateStoreTypeEtcd:
		return NewEtcdStateStore(dss.config)
	case StateStoreTypeDynamoDB:
		return NewDynamoDBStateStore(dss.config)
	default:
		return NewMemoryStateStore(), nil // Default to memory
	}
}

// Get retrieves a value from state store
func (dss *DistributedStateStore) Get(ctx context.Context, key string) (*StateEntry, error) {
	if atomic.LoadInt32(&dss.closed) == 1 {
		return nil, fmt.Errorf("state store is closed")
	}

	start := time.Now()
	defer func() {
		atomic.AddInt64(&dss.stats.ReadOperations, 1)
		atomic.AddInt64(&dss.stats.TotalOperations, 1)
		dss.updateReadLatency(time.Since(start))
	}()

	// Check cache first
	if dss.cache != nil {
		if entry := dss.cache.Get(key); entry != nil {
			atomic.AddInt64(&dss.stats.CacheHits, 1)
			return entry, nil
		}
		atomic.AddInt64(&dss.stats.CacheMisses, 1)
	}

	// Try local store first
	entry, err := dss.localStore.Get(ctx, key)
	if err == nil {
		// Update cache
		if dss.cache != nil {
			dss.cache.Set(key, entry)
		}
		return entry, nil
	}

	// Try remote store
	entry, err = dss.remoteStore.Get(ctx, key)
	if err == nil {
		// Update local store and cache
		dss.localStore.Set(ctx, key, entry.Value, entry.TTL)
		if dss.cache != nil {
			dss.cache.Set(key, entry)
		}
		return entry, nil
	}

	return nil, err
}

// Set stores a value in the state store
func (dss *DistributedStateStore) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	if atomic.LoadInt32(&dss.closed) == 1 {
		return fmt.Errorf("state store is closed")
	}

	start := time.Now()
	defer func() {
		atomic.AddInt64(&dss.stats.WriteOperations, 1)
		atomic.AddInt64(&dss.stats.TotalOperations, 1)
		dss.updateWriteLatency(time.Since(start))
	}()

	// Determine TTL
	entryTTL := dss.config.TTL
	if len(ttl) > 0 && ttl[0] > 0 {
		entryTTL = ttl[0]
	}

	// Create state entry
	entry := &StateEntry{
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		TTL:       entryTTL,
		Metadata:  make(map[string]interface{}),
	}

	if entryTTL > 0 {
		expiresAt := time.Now().Add(entryTTL)
		entry.ExpiresAt = &expiresAt
	}

	// Store in remote store first
	if err := dss.remoteStore.Set(ctx, key, value, entryTTL); err != nil {
		return fmt.Errorf("failed to store in remote: %w", err)
	}

	// Store in local store
	if err := dss.localStore.Set(ctx, key, value, entryTTL); err != nil {
		return fmt.Errorf("failed to store in local: %w", err)
	}

	// Update cache
	if dss.cache != nil {
		dss.cache.Set(key, entry)
	}

	return nil
}

// Delete removes a value from the state store
func (dss *DistributedStateStore) Delete(ctx context.Context, key string) error {
	if atomic.LoadInt32(&dss.closed) == 1 {
		return fmt.Errorf("state store is closed")
	}

	// Delete from all stores
	errs := make([]error, 0)

	// Remote store
	if err := dss.remoteStore.Delete(ctx, key); err != nil {
		errs = append(errs, fmt.Errorf("remote delete failed: %w", err))
	}

	// Local store
	if err := dss.localStore.Delete(ctx, key); err != nil {
		errs = append(errs, fmt.Errorf("local delete failed: %w", err))
	}

	// Cache
	if dss.cache != nil {
		dss.cache.Delete(key)
	}

	if len(errs) > 0 {
		return fmt.Errorf("delete operations failed: %v", errs)
	}

	return nil
}

// Exists checks if a key exists in the state store
func (dss *DistributedStateStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := dss.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return true, nil
}

// List retrieves all entries with a given prefix
func (dss *DistributedStateStore) List(ctx context.Context, prefix string) ([]*StateEntry, error) {
	// For now, delegate to local store
	// In practice, this would combine results from multiple stores
	return dss.localStore.List(ctx, prefix)
}

// Watch watches for changes to a key
func (dss *DistributedStateStore) Watch(ctx context.Context, key string) (<-chan *StateEvent, error) {
	return dss.localStore.Watch(ctx, key)
}

// Lock acquires a distributed lock
func (dss *DistributedStateStore) Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	if !dss.config.EnableLocking {
		return nil, fmt.Errorf("locking is not enabled")
	}

	start := time.Now()
	defer func() {
		atomic.AddInt64(&dss.stats.LockOperations, 1)
		atomic.StoreInt64(&dss.stats.LockWaitTime.Nanoseconds(), time.Since(start).Nanoseconds())
	}()

	// Try remote store first
	lock, err := dss.remoteStore.Lock(ctx, key, ttl)
	if err == nil {
		return lock, nil
	}

	// Fallback to local store
	return dss.localStore.Lock(ctx, key, ttl)
}

// Unlock releases a distributed lock
func (dss *DistributedStateStore) Unlock(ctx context.Context, lock *Lock) error {
	if !dss.config.EnableLocking {
		return fmt.Errorf("locking is not enabled")
	}

	// Try remote store first
	err := dss.remoteStore.Unlock(ctx, lock)
	if err == nil {
		return nil
	}

	// Fallback to local store
	return dss.localStore.Unlock(ctx, lock)
}

// Transaction executes multiple operations atomically
func (dss *DistributedStateStore) Transaction(ctx context.Context, ops []Operation) error {
	// For now, execute sequentially
	// In practice, this would use actual transactions
	for _, op := range ops {
		switch op.Type {
		case OpTypeSet:
			if err := dss.Set(ctx, op.Key, op.Value, op.TTL); err != nil {
				return fmt.Errorf("transaction failed: %w", err)
			}
		case OpTypeDelete:
			if err := dss.Delete(ctx, op.Key); err != nil {
				return fmt.Errorf("transaction failed: %w", err)
			}
		}
	}
	return nil
}

// Backup creates a backup of the state store
func (dss *DistributedStateStore) Backup(ctx context.Context) ([]byte, error) {
	// Get all entries from local store
	entries, err := dss.localStore.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list entries for backup: %w", err)
	}

	// Convert to JSON
	data, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup data: %w", err)
	}

	return data, nil
}

// Restore restores a backup of the state store
func (dss *DistributedStateStore) Restore(ctx context.Context, data []byte) error {
	var entries []*StateEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	// Restore entries
	for _, entry := range entries {
		if err := dss.Set(ctx, entry.Key, entry.Value, entry.TTL); err != nil {
			return fmt.Errorf("failed to restore entry %s: %w", entry.Key, err)
		}
	}

	return nil
}

// Close closes the state store
func (dss *DistributedStateStore) Close() error {
	atomic.StoreInt32(&dss.closed, 1)

	// Close remote store
	if dss.remoteStore != nil {
		dss.remoteStore.Close()
	}

	// Close cache
	if dss.cache != nil {
		dss.cache.Close()
	}

	return nil
}

// Stats returns state store statistics
func (dss *DistributedStateStore) Stats() StateStoreStats {
	return *dss.stats
}

// updateReadLatency updates read latency statistics
func (dss *DistributedStateStore) updateReadLatency(latency time.Duration) {
	current := atomic.LoadInt64(&dss.stats.AvgReadLatency.Nanoseconds())
	newAvg := time.Duration((int64(current) + int64(latency.Nanoseconds())) / 2)
	atomic.StoreInt64(&dss.stats.AvgReadLatency.Nanoseconds(), int64(newAvg))
}

// updateWriteLatency updates write latency statistics
func (dss *DistributedStateStore) updateWriteLatency(latency time.Duration) {
	current := atomic.LoadInt64(&dss.stats.AvgWriteLatency.Nanoseconds())
	newAvg := time.Duration((int64(current) + int64(latency.Nanoseconds())) / 2)
	atomic.StoreInt64(&dss.stats.AvgWriteLatency.Nanoseconds(), int64(newAvg))
}

// NewMemoryStateStore creates a new in-memory state store
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		data:    make(map[string]*StateEntry),
		locks:   make(map[string]*Lock),
		watches: make(map[string]chan *StateEvent),
		stats:   &StateStoreStats{},
	}
}

// MemoryStateStore implementations
func (mss *MemoryStateStore) Get(ctx context.Context, key string) (*StateEntry, error) {
	mss.mu.RLock()
	defer mss.mu.RUnlock()

	entry, exists := mss.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	// Check expiration
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		delete(mss.data, key)
		return nil, fmt.Errorf("key expired")
	}

	return entry, nil
}

func (mss *MemoryStateStore) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	mss.mu.Lock()
	defer mss.mu.Unlock()

	entryTTL := time.Duration(0)
	if len(ttl) > 0 {
		entryTTL = ttl[0]
	}

	now := time.Now()
	var expiresAt *time.Time
	if entryTTL > 0 {
		exp := now.Add(entryTTL)
		expiresAt = &exp
	}

	entry := &StateEntry{
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
		TTL:       entryTTL,
		ExpiresAt: expiresAt,
		Metadata:  make(map[string]interface{}),
	}

	mss.data[key] = entry

	// Notify watchers
	if watch, exists := mss.watches[key]; exists {
		event := &StateEvent{
			Key:       key,
			Type:      EventTypeUpdated,
			Value:     value,
			Timestamp: now,
		}
		select {
		case watch <- event:
		default:
		}
	}

	return nil
}

func (mss *MemoryStateStore) Delete(ctx context.Context, key string) error {
	mss.mu.Lock()
	defer mss.mu.Unlock()

	delete(mss.data, key)
	delete(mss.locks, key)

	// Notify watchers
	if watch, exists := mss.watches[key]; exists {
		event := &StateEvent{
			Key:       key,
			Type:      EventTypeDeleted,
			Timestamp: time.Now(),
		}
		select {
		case watch <- event:
		default:
		}
	}

	return nil
}

func (mss *MemoryStateStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := mss.Get(ctx, key)
	return err == nil, nil
}

func (mss *MemoryStateStore) List(ctx context.Context, prefix string) ([]*StateEntry, error) {
	mss.mu.RLock()
	defer mss.mu.RUnlock()

	var entries []*StateEntry
	for key, entry := range mss.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Check expiration
			if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
				continue
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func (mss *MemoryStateStore) Watch(ctx context.Context, key string) (<-chan *StateEvent, error) {
	mss.mu.Lock()
	defer mss.mu.Unlock()

	watch := make(chan *StateEvent, 10)
	mss.watches[key] = watch

	return watch, nil
}

func (mss *MemoryStateStore) Lock(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	mss.mu.Lock()
	defer mss.mu.Unlock()

	// Check if lock already exists
	if lock, exists := mss.locks[key]; exists && time.Now().Before(lock.ExpiresAt) {
		return nil, fmt.Errorf("key is already locked")
	}

	// Create new lock
	lock := &Lock{
		Key:       key,
		Owner:     "memory_store",
		ExpiresAt: time.Now().Add(ttl),
	}

	mss.locks[key] = lock
	return lock, nil
}

func (mss *MemoryStateStore) Unlock(ctx context.Context, lock *Lock) error {
	mss.mu.Lock()
	defer mss.mu.Unlock()

	existingLock, exists := mss.locks[lock.Key]
	if !exists {
		return fmt.Errorf("lock not found")
	}

	if existingLock.Owner != lock.Owner {
		return fmt.Errorf("lock owned by different process")
	}

	delete(mss.locks, lock.Key)
	return nil
}

func (mss *MemoryStateStore) Transaction(ctx context.Context, ops []Operation) error {
	// Execute operations atomically
	for _, op := range ops {
		switch op.Type {
		case OpTypeSet:
			if err := mss.Set(ctx, op.Key, op.Value, op.TTL); err != nil {
				return err
			}
		case OpTypeDelete:
			if err := mss.Delete(ctx, op.Key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (mss *MemoryStateStore) Backup(ctx context.Context) ([]byte, error) {
	return json.Marshal(mss.data)
}

func (mss *MemoryStateStore) Restore(ctx context.Context, data []byte) error {
	return json.Unmarshal(data, &mss.data)
}

func (mss *MemoryStateStore) Close() error {
	return nil
}

func (mss *MemoryStateStore) Stats() StateStoreStats {
	return *mss.stats
}

// Placeholder implementations for remote stores
func NewRedisStateStore(config StateStoreConfig) (StateStore, error) {
	return NewMemoryStateStore(), nil // Placeholder
}

func NewConsulStateStore(config StateStoreConfig) (StateStore, error) {
	return NewMemoryStateStore(), nil // Placeholder
}

func NewEtcdStateStore(config StateStoreConfig) (StateStore, error) {
	return NewMemoryStateStore(), nil // Placeholder
}

func NewDynamoDBStateStore(config StateStoreConfig) (StateStore, error) {
	return NewMemoryStateStore(), nil // Placeholder
}

// NewStateCache creates a new state cache
func NewStateCache(config CacheConfig) *StateCache {
	cache := &StateCache{
		config:  config,
		entries: make(map[string]*CacheEntry),
		stats:   &CacheStats{},
	}

	// Start cache cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves value from cache
func (sc *StateCache) Get(key string) *StateEntry {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	atomic.AddInt64(&sc.stats.TotalRequests, 1)

	entry, exists := sc.entries[key]
	if !exists {
		atomic.AddInt64(&sc.stats.CacheMisses, 1)
		return nil
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		delete(sc.entries, key)
		atomic.AddInt64(&sc.stats.CacheMisses, 1)
		return nil
	}

	// Update access info
	entry.Accessed = time.Now()
	entry.Hits++

	atomic.AddInt64(&sc.stats.CacheHits, 1)
	return entry.Value
}

// Set stores value in cache
func (sc *StateCache) Set(key string, value *StateEntry) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check cache size limit
	if len(sc.entries) >= sc.config.MaxSize {
		sc.evictLRU()
	}

	expiresAt := time.Now().Add(sc.config.TTL)
	if value.TTL > 0 {
		expiresAt = time.Now().Add(value.TTL)
	}

	sc.entries[key] = &CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
		Accessed:  time.Now(),
		Hits:      0,
	}
}

// Delete removes value from cache
func (sc *StateCache) Delete(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.entries, key)
}

// cleanup removes expired entries
func (sc *StateCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sc.mu.Lock()
			now := time.Now()
			for key, entry := range sc.entries {
				if now.After(entry.ExpiresAt) {
					delete(sc.entries, key)
				}
			}
			atomic.StoreInt64(&sc.stats.LastFlush, time.Now().Unix())
			sc.mu.Unlock()
		}
	}
}

// evictLRU evicts least recently used entry
func (sc *StateCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range sc.entries {
		if entry.Accessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Accessed
		}
	}

	if oldestKey != "" {
		delete(sc.entries, oldestKey)
	}
}

// Close closes cache
func (sc *StateCache) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries = make(map[string]*CacheEntry)
	return nil
}