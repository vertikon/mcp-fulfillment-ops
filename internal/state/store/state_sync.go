package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// StateSync handles synchronization between distributed nodes
type StateSync interface {
	SyncWithPeer(ctx context.Context, peer string) error
	BroadcastUpdate(ctx context.Context, state *VersionedState) error
	SubscribeToUpdates(ctx context.Context) (<-chan *VersionedState, error)
	GetSyncStatus(ctx context.Context) (*SyncStatus, error)
}

// SyncStatus represents synchronization status
type SyncStatus struct {
	NodeID       string                  `json:"node_id"`
	Peers        []string                `json:"peers"`
	SyncProgress map[string]SyncProgress `json:"sync_progress"`
	LastSync     *time.Time              `json:"last_sync"`
	TotalUpdates int64                   `json:"total_updates"`
	QueueSize    int                     `json:"queue_size"`
}

// SyncProgress represents sync progress with a peer
type SyncProgress struct {
	PeerID       string    `json:"peer_id"`
	Status       string    `json:"status"` // syncing, synced, error
	LastSync     time.Time `json:"last_sync"`
	UpdatesCount int64     `json:"updates_count"`
	Error        string    `json:"error,omitempty"`
}

// SyncConfig represents synchronization configuration
type SyncConfig struct {
	NodeID       string        `json:"node_id"`
	Peers        []string      `json:"peers"`
	SyncInterval time.Duration `json:"sync_interval"`
	MaxRetries   int           `json:"max_retries"`
	RetryDelay   time.Duration `json:"retry_delay"`
	QueueSize    int           `json:"queue_size"`
	BatchSize    int           `json:"batch_size"`
	SyncTimeout  time.Duration `json:"sync_timeout"`
	Compression  bool          `json:"compression"`
	Encryption   bool          `json:"encryption"`
}

// DefaultSyncConfig returns default sync configuration
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		NodeID:       fmt.Sprintf("sync-node-%d", time.Now().Unix()),
		Peers:        []string{},
		SyncInterval: 30 * time.Second,
		MaxRetries:   3,
		RetryDelay:   5 * time.Second,
		QueueSize:    10000, // Aumentado de 1000 para 10000
		BatchSize:    1000,  // Aumentado de 100 para 1000
		SyncTimeout:  30 * time.Second,
		Compression:  true,
		Encryption:   false,
	}
}

// StateSyncImpl implements StateSync interface
type StateSyncImpl struct {
	config      *SyncConfig
	store       DistributedStore
	updateCh    chan *VersionedState
	subscribers []chan *VersionedState
	mu          sync.RWMutex
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	stats       *SyncStatus
}

// NewStateSync creates a new state synchronization implementation
func NewStateSync(config *SyncConfig, store DistributedStore) *StateSyncImpl {
	if config == nil {
		config = DefaultSyncConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	sync := &StateSyncImpl{
		config:      config,
		store:       store,
		updateCh:    make(chan *VersionedState, config.QueueSize*2), // Buffer maior
		subscribers: []chan *VersionedState{},
		stats: &SyncStatus{
			NodeID:       config.NodeID,
			Peers:        config.Peers,
			SyncProgress: make(map[string]SyncProgress),
			TotalUpdates: 0,
			QueueSize:    0,
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger.Get(),
	}

	// Initialize sync progress for each peer
	for _, peer := range config.Peers {
		sync.stats.SyncProgress[peer] = SyncProgress{
			PeerID: peer,
			Status: "not_started",
		}
	}

	// Start background sync processes
	go sync.backgroundSync()

	sync.logger.Info("State synchronization initialized",
		zap.String("node_id", config.NodeID),
		zap.Strings("peers", config.Peers),
		zap.Duration("sync_interval", config.SyncInterval))

	return sync
}

// SyncWithPeer syncs state with a specific peer
func (s *StateSyncImpl) SyncWithPeer(ctx context.Context, peer string) error {
	s.mu.Lock()
	progress, exists := s.stats.SyncProgress[peer]
	if !exists {
		progress = SyncProgress{
			PeerID: peer,
			Status: "not_started",
		}
		s.stats.SyncProgress[peer] = progress
	}
	progress.Status = "syncing"
	progress.LastSync = time.Now()
	s.stats.SyncProgress[peer] = progress
	s.mu.Unlock()

	s.logger.Info("Starting sync with peer",
		zap.String("peer", peer))

	// In a real implementation, this would:
	// 1. Connect to peer
	// 2. Exchange state versions
	// 3. Determine missing updates
	// 4. Transfer missing updates
	// 5. Apply updates to local store

	// Simulate sync process
	time.Sleep(1 * time.Second)

	// Update progress
	s.mu.Lock()
	progress.Status = "synced"
	progress.UpdatesCount = 10 // Simulated
	s.stats.SyncProgress[peer] = progress
	s.stats.LastSync = &time.Time{}
	*s.stats.LastSync = time.Now()
	s.mu.Unlock()

	s.logger.Info("Sync with peer completed",
		zap.String("peer", peer),
		zap.Int64("updates_count", progress.UpdatesCount))

	return nil
}

// BroadcastUpdate broadcasts a state update to all peers
func (s *StateSyncImpl) BroadcastUpdate(ctx context.Context, state *VersionedState) error {
	s.mu.Lock()
	s.stats.TotalUpdates++
	s.mu.Unlock()

	s.logger.Debug("Broadcasting update to peers",
		zap.String("key", state.Key),
		zap.Uint64("version", state.Version),
		zap.Int("peers_count", len(s.config.Peers)))

	// Send update to all peers concurrently
	var wg sync.WaitGroup
	errCh := make(chan error, len(s.config.Peers))

	for _, peer := range s.config.Peers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			if err := s.sendUpdateToPeer(ctx, p, state); err != nil {
				s.logger.Error("Failed to send update to peer",
					zap.String("peer", p),
					zap.Error(err))
				errCh <- err
			}
		}(peer)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errCh)

	// Check for errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed for %d peers: %v", len(errors), errors[0])
	}

	return nil
}

// sendUpdateToPeer sends update to a specific peer
func (s *StateSyncImpl) sendUpdateToPeer(ctx context.Context, peer string, state *VersionedState) error {
	// In a real implementation, this would:
	// 1. Serialize state
	// 2. Compress if enabled
	// 3. Encrypt if enabled
	// 4. Send via network
	// 5. Handle retries

	s.logger.Debug("Sending update to peer",
		zap.String("peer", peer),
		zap.String("key", state.Key))

	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	return nil
}

// SubscribeToUpdates subscribes to state updates
func (s *StateSyncImpl) SubscribeToUpdates(ctx context.Context) (<-chan *VersionedState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updateCh := make(chan *VersionedState, s.config.QueueSize)
	s.subscribers = append(s.subscribers, updateCh)

	s.logger.Debug("New subscriber added",
		zap.Int("total_subscribers", len(s.subscribers)))

	return updateCh, nil
}

// GetSyncStatus returns synchronization status
func (s *StateSyncImpl) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Update queue size
	s.stats.QueueSize = len(s.updateCh)

	// Return a copy of stats
	status := *s.stats
	status.SyncProgress = make(map[string]SyncProgress)
	for k, v := range s.stats.SyncProgress {
		status.SyncProgress[k] = v
	}

	return &status, nil
}

// NotifyUpdate notifies subscribers of state updates
func (s *StateSyncImpl) NotifyUpdate(ctx context.Context, state *VersionedState) error {
	select {
	case s.updateCh <- state:
		s.logger.Debug("Update queued for broadcasting",
			zap.String("key", state.Key),
			zap.Uint64("version", state.Version))
		return nil
	default:
		return fmt.Errorf("update queue is full")
	}
}

// backgroundSync runs background synchronization processes
func (s *StateSyncImpl) backgroundSync() {
	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Background sync stopped")
			return

		case <-ticker.C:
			s.performPeriodicSync()

		case state := <-s.updateCh:
			s.broadcastState(state)
		}
	}
}

// performPeriodicSync performs periodic synchronization with all peers
func (s *StateSyncImpl) performPeriodicSync() {
	s.logger.Debug("Performing periodic sync")

	var wg sync.WaitGroup
	for _, peer := range s.config.Peers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			if err := s.SyncWithPeer(s.ctx, p); err != nil {
				s.logger.Error("Periodic sync failed",
					zap.String("peer", p),
					zap.Error(err))
			}
		}(peer)
	}

	wg.Wait()
}

// broadcastState broadcasts state to all subscribers
func (s *StateSyncImpl) broadcastState(state *VersionedState) {
	s.mu.RLock()
	subscribers := make([]chan *VersionedState, len(s.subscribers))
	copy(subscribers, s.subscribers)
	s.mu.RUnlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- state:
		default:
			s.logger.Warn("Subscriber channel full, dropping update")
		}
	}
}

// Stop stops the state synchronization
func (s *StateSyncImpl) Stop() {
	s.cancel()
	close(s.updateCh)

	// Close all subscriber channels
	s.mu.Lock()
	for _, subscriber := range s.subscribers {
		close(subscriber)
	}
	s.subscribers = []chan *VersionedState{}
	s.mu.Unlock()

	s.logger.Info("State synchronization stopped")
}
