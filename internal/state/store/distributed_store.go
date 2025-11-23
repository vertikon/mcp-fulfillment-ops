package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// VersionedState represents a versioned state entry
type VersionedState struct {
	Key     string                 `json:"key"`
	Value   interface{}            `json:"value"`
	Version uint64                 `json:"version"`
	TTL     *time.Time             `json:"ttl,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// DistributedStore interface for distributed state management
type DistributedStore interface {
	// Basic operations
	Get(ctx context.Context, key string) (*VersionedState, error)
	Set(ctx context.Context, key string, value interface{}, ttl *time.Time) (*VersionedState, error)
	Delete(ctx context.Context, key string) error

	// Compare and set
	CompareAndSet(ctx context.Context, key string, expectedVersion uint64, value interface{}, ttl *time.Time) (*VersionedState, error)

	// Distributed locks
	AcquireLock(ctx context.Context, lockKey string, ttlSeconds int) (bool, error)
	ReleaseLock(ctx context.Context, lockKey string) error

	// Snapshots
	Snapshot(ctx context.Context) error
	Restore(ctx context.Context, snapshotID string) error

	// State synchronization
	SyncFrom(ctx context.Context, peers []string) error
	NotifyUpdate(ctx context.Context, state *VersionedState) error

	// Health and status
	Health(ctx context.Context) (StoreHealth, error)
	Stats(ctx context.Context) (StoreStats, error)

	// Snapshot support
	GetAllKeys(ctx context.Context) ([]string, error)
}

// StoreHealth represents store health status
type StoreHealth struct {
	Status    string    `json:"status"`
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
	Peers     []string  `json:"peers"`
	Size      int64     `json:"size"`
}

// StoreStats represents store statistics
type StoreStats struct {
	TotalKeys      int64      `json:"total_keys"`
	ReadOps        int64      `json:"read_ops"`
	WriteOps       int64      `json:"write_ops"`
	ConflictOps    int64      `json:"conflict_ops"`
	SnapshotOps    int64      `json:"snapshot_ops"`
	LastSnapshot   *time.Time `json:"last_snapshot"`
	TotalSizeBytes int64      `json:"total_size_bytes"`
}

// StoreConfig represents store configuration
type StoreConfig struct {
	NodeID             string        `json:"node_id"`
	Peers              []string      `json:"peers"`
	SnapshotInterval   time.Duration `json:"snapshot_interval"`
	SnapshotRetention  int           `json:"snapshot_retention"`
	ConflictResolution string        `json:"conflict_resolution"`
	LockTimeout        time.Duration `json:"lock_timeout"`
	StorageBackend     string        `json:"storage_backend"`
	StoragePath        string        `json:"storage_path"`
}

// DefaultStoreConfig returns default store configuration
func DefaultStoreConfig() *StoreConfig {
	return &StoreConfig{
		NodeID:             fmt.Sprintf("node-%d", time.Now().Unix()),
		Peers:              []string{},
		SnapshotInterval:   5 * time.Minute,
		SnapshotRetention:  10,
		ConflictResolution: "last-write-wins",
		LockTimeout:        30 * time.Second,
		StorageBackend:     "memory",
		StoragePath:        "./data/store",
	}
}

// InMemoryDistributedStore implements DistributedStore in memory
type InMemoryDistributedStore struct {
	config *StoreConfig
	data   map[string]*VersionedState
	locks  map[string]*Lock
	mu     sync.RWMutex
	logger *zap.Logger

	// Channel for notifications
	updateCh chan *VersionedState

	// Statistics
	stats *StoreStats

	// Background context
	ctx    context.Context
	cancel context.CancelFunc
}

// Lock represents a distributed lock
type Lock struct {
	Key        string                 `json:"key"`
	NodeID     string                 `json:"node_id"`
	AcquiredAt time.Time              `json:"acquired_at"`
	TTL        time.Duration          `json:"ttl"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// NewInMemoryDistributedStore creates a new in-memory distributed store
func NewInMemoryDistributedStore(config *StoreConfig) *InMemoryDistributedStore {
	if config == nil {
		config = DefaultStoreConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	store := &InMemoryDistributedStore{
		config:   config,
		data:     make(map[string]*VersionedState),
		locks:    make(map[string]*Lock),
		updateCh: make(chan *VersionedState, 10000), // Aumentado de 1000 para 10000
		stats:    &StoreStats{},
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger.Get(),
	}

	// Start background processes
	go store.backgroundProcesses()

	store.logger.Info("In-memory distributed store initialized",
		zap.String("node_id", config.NodeID),
		zap.Strings("peers", config.Peers))

	return store
}

// Get retrieves a value by key
func (s *InMemoryDistributedStore) Get(ctx context.Context, key string) (*VersionedState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check TTL
	if state.TTL != nil && state.TTL.Before(time.Now()) {
		delete(s.data, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}

	s.stats.ReadOps++

	s.logger.Debug("Key retrieved",
		zap.String("key", key),
		zap.Uint64("version", state.Version))

	return state, nil
}

// Set stores a value with versioning
func (s *InMemoryDistributedStore) Set(ctx context.Context, key string, value interface{}, ttl *time.Time) (*VersionedState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current version
	currentVersion := uint64(0)
	if currentState, exists := s.data[key]; exists {
		currentVersion = currentState.Version
	}

	// Create new version
	newState := &VersionedState{
		Key:     key,
		Value:   value,
		Version: currentVersion + 1,
		TTL:     ttl,
		Meta: map[string]interface{}{
			"node_id":    s.config.NodeID,
			"created_at": time.Now(),
		},
	}

	// Store state
	s.data[key] = newState
	s.stats.WriteOps++

	// Send update notification
	select {
	case s.updateCh <- newState:
	default:
		s.logger.Warn("Update channel full, dropping update",
			zap.String("key", key))
	}

	s.logger.Debug("Key set",
		zap.String("key", key),
		zap.Uint64("version", newState.Version))

	return newState, nil
}

// Delete removes a key
func (s *InMemoryDistributedStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)

	s.logger.Debug("Key deleted", zap.String("key", key))
	return nil
}

// CompareAndSet performs atomic compare-and-set operation
func (s *InMemoryDistributedStore) CompareAndSet(ctx context.Context, key string, expectedVersion uint64, value interface{}, ttl *time.Time) (*VersionedState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentState, exists := s.data[key]

	// Check if expected version matches
	if !exists && expectedVersion != 0 {
		s.stats.ConflictOps++
		return nil, fmt.Errorf("expected version %d but key does not exist", expectedVersion)
	}

	if exists && currentState.Version != expectedVersion {
		s.stats.ConflictOps++
		return nil, fmt.Errorf("expected version %d but current version is %d", expectedVersion, currentState.Version)
	}

	// Create new version
	newState := &VersionedState{
		Key:     key,
		Value:   value,
		Version: expectedVersion + 1,
		TTL:     ttl,
		Meta: map[string]interface{}{
			"node_id":    s.config.NodeID,
			"created_at": time.Now(),
			"cas":        true,
		},
	}

	// Store state
	s.data[key] = newState
	s.stats.WriteOps++

	// Send update notification
	select {
	case s.updateCh <- newState:
	default:
		s.logger.Warn("Update channel full, dropping update",
			zap.String("key", key))
	}

	s.logger.Debug("CAS operation successful",
		zap.String("key", key),
		zap.Uint64("expected_version", expectedVersion),
		zap.Uint64("new_version", newState.Version))

	return newState, nil
}

// AcquireLock acquires a distributed lock
func (s *InMemoryDistributedStore) AcquireLock(ctx context.Context, lockKey string, ttlSeconds int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if lock exists and is still valid
	if existingLock, exists := s.locks[lockKey]; exists {
		// Check if lock has expired
		if existingLock.AcquiredAt.Add(existingLock.TTL).After(time.Now()) {
			// Lock is still held by another node
			return false, nil
		}
		// Lock has expired, can acquire
	}

	// Create new lock
	newLock := &Lock{
		Key:        lockKey,
		NodeID:     s.config.NodeID,
		AcquiredAt: time.Now(),
		TTL:        time.Duration(ttlSeconds) * time.Second,
		Meta: map[string]interface{}{
			"acquired_at": time.Now(),
		},
	}

	s.locks[lockKey] = newLock

	s.logger.Debug("Lock acquired",
		zap.String("lock_key", lockKey),
		zap.String("node_id", s.config.NodeID),
		zap.Duration("ttl", newLock.TTL))

	return true, nil
}

// ReleaseLock releases a distributed lock
func (s *InMemoryDistributedStore) ReleaseLock(ctx context.Context, lockKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, exists := s.locks[lockKey]
	if !exists {
		return fmt.Errorf("lock not found: %s", lockKey)
	}

	// Check if this node owns the lock
	if lock.NodeID != s.config.NodeID {
		return fmt.Errorf("lock not owned by this node: %s", lockKey)
	}

	delete(s.locks, lockKey)

	s.logger.Debug("Lock released",
		zap.String("lock_key", lockKey),
		zap.String("node_id", s.config.NodeID))

	return nil
}

// Snapshot creates a snapshot of the current state
func (s *InMemoryDistributedStore) Snapshot(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create snapshot
	snapshotID := fmt.Sprintf("snapshot-%d", time.Now().Unix())
	snapshot := make(map[string]*VersionedState)

	for k, v := range s.data {
		snapshot[k] = v
	}

	// In a real implementation, this would be persisted
	s.logger.Info("Snapshot created",
		zap.String("snapshot_id", snapshotID),
		zap.Int("keys_count", len(snapshot)))

	s.stats.SnapshotOps++
	s.stats.LastSnapshot = &time.Time{}
	*s.stats.LastSnapshot = time.Now()

	return nil
}

// Restore restores state from a snapshot
func (s *InMemoryDistributedStore) Restore(ctx context.Context, snapshotID string) error {
	s.logger.Info("Restore from snapshot",
		zap.String("snapshot_id", snapshotID))

	// In a real implementation, this would load from persistence
	return nil
}

// SyncFrom syncs state from peers
func (s *InMemoryDistributedStore) SyncFrom(ctx context.Context, peers []string) error {
	s.logger.Info("Syncing from peers",
		zap.Strings("peers", peers))

	// In a real implementation, this would sync state from peer nodes
	return nil
}

// NotifyUpdate notifies subscribers of state updates
func (s *InMemoryDistributedStore) NotifyUpdate(ctx context.Context, state *VersionedState) error {
	// Send update notification
	select {
	case s.updateCh <- state:
	default:
		return fmt.Errorf("update channel full")
	}

	return nil
}

// Health returns store health status
func (s *InMemoryDistributedStore) Health(ctx context.Context) (StoreHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return StoreHealth{
		Status:    "healthy",
		NodeID:    s.config.NodeID,
		Timestamp: time.Now(),
		Peers:     s.config.Peers,
		Size:      int64(len(s.data)),
	}, nil
}

// Stats returns store statistics
func (s *InMemoryDistributedStore) Stats(ctx context.Context) (StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := *s.stats
	stats.TotalKeys = int64(len(s.data))
	stats.TotalSizeBytes = s.calculateSize(s.data)

	return stats, nil
}

// GetAllKeys returns all keys in the store
func (s *InMemoryDistributedStore) GetAllKeys(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}

	return keys, nil
}

// calculateSize estimates the total size of stored data
func (s *InMemoryDistributedStore) calculateSize(data map[string]*VersionedState) int64 {
	var size int64
	for _, state := range data {
		size += int64(len(state.Key)) + 100 // Rough estimate
	}
	return size
}

// backgroundProcesses runs background maintenance tasks
func (s *InMemoryDistributedStore) backgroundProcesses() {
	ticker := time.NewTicker(s.config.SnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Background processes stopped")
			return

		case state := <-s.updateCh:
			s.handleUpdate(state)

		case <-ticker.C:
			s.backgroundSnapshot()
		}
	}
}

// handleUpdate handles state update notifications
func (s *InMemoryDistributedStore) handleUpdate(state *VersionedState) {
	s.logger.Debug("Handling update notification",
		zap.String("key", state.Key),
		zap.Uint64("version", state.Version))

	// In a real implementation, this would notify subscribers
}

// backgroundSnapshot performs automatic snapshots
func (s *InMemoryDistributedStore) backgroundSnapshot() {
	if err := s.Snapshot(s.ctx); err != nil {
		s.logger.Error("Background snapshot failed", zap.Error(err))
	}
}

// Close closes the store
func (s *InMemoryDistributedStore) Close() error {
	s.cancel()
	close(s.updateCh)

	s.logger.Info("In-memory distributed store closed")
	return nil
}
