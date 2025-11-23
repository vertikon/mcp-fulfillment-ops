package store

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// SnapshotType represents type of snapshot
type SnapshotType string

const (
	FullSnapshot        SnapshotType = "full"
	IncrementalSnapshot SnapshotType = "incremental"
	AutoSnapshot        SnapshotType = "auto"
	ManualSnapshot      SnapshotType = "manual"
)

// SnapshotInfo represents metadata about a snapshot
type SnapshotInfo struct {
	ID             string                 `json:"id"`
	Type           SnapshotType           `json:"type"`
	CreatedAt      time.Time              `json:"created_at"`
	Size           int64                  `json:"size"`
	KeysCount      int                    `json:"keys_count"`
	BaseSnapshotID string                 `json:"base_snapshot_id,omitempty"`
	NodeID         string                 `json:"node_id"`
	Version        uint64                 `json:"version"`
	Checksum       string                 `json:"checksum"`
	Compression    bool                   `json:"compression"`
	Encrypted      bool                   `json:"encrypted"`
	Meta           map[string]interface{} `json:"meta,omitempty"`
}

// SnapshotData represents actual snapshot data
type SnapshotData struct {
	Info    SnapshotInfo               `json:"info"`
	State   map[string]*VersionedState `json:"state"`
	Deleted []string                   `json:"deleted,omitempty"`
	Changes map[string]*StateChange    `json:"changes,omitempty"`
}

// StateChange represents a change to a state entry
type StateChange struct {
	OldState *VersionedState `json:"old_state,omitempty"`
	NewState *VersionedState `json:"new_state,omitempty"`
	Type     string          `json:"type"` // created, updated, deleted
	Time     time.Time       `json:"time"`
	NodeID   string          `json:"node_id"`
}

// SnapshotManager interface for managing state snapshots
type SnapshotManager interface {
	CreateSnapshot(ctx context.Context, snapshotType SnapshotType, baseSnapshotID string) (*SnapshotInfo, error)
	RestoreSnapshot(ctx context.Context, snapshotID string) error
	DeleteSnapshot(ctx context.Context, snapshotID string) error
	ListSnapshots(ctx context.Context) ([]*SnapshotInfo, error)
	GetSnapshotInfo(ctx context.Context, snapshotID string) (*SnapshotInfo, error)
	IncrementalSnapshot(ctx context.Context, baseSnapshotID string) (*SnapshotInfo, error)
	ScheduleAutoSnapshot(ctx context.Context, interval time.Duration) error
	GetSnapshotStats() SnapshotStats
}

// SnapshotStats represents snapshot statistics
type SnapshotStats struct {
	TotalSnapshots       int64         `json:"total_snapshots"`
	FullSnapshots        int64         `json:"full_snapshots"`
	IncrementalSnapshots int64         `json:"incremental_snapshots"`
	TotalSize            int64         `json:"total_size"`
	AverageSize          int64         `json:"average_size"`
	LastSnapshot         *time.Time    `json:"last_snapshot"`
	SnapshotFrequency    time.Duration `json:"snapshot_frequency"`
	SnapshotRetention    int           `json:"snapshot_retention"`
	CompressionRatio     float64       `json:"compression_ratio"`
	RestoreTime          time.Duration `json:"average_restore_time"`
}

// SnapshotConfig represents snapshot configuration
type SnapshotConfig struct {
	SnapshotPath     string        `json:"snapshot_path"`
	Compression      bool          `json:"compression"`
	Encryption       bool          `json:"encryption"`
	AutoSnapshot     bool          `json:"auto_snapshot"`
	SnapshotInterval time.Duration `json:"snapshot_interval"`
	RetentionCount   int           `json:"retention_count"`
	RetentionDays    int           `json:"retention_days"`
	MaxSnapshotSize  int64         `json:"max_snapshot_size"`
	BackupPath       string        `json:"backup_path"`
	NodeID           string        `json:"node_id"`
}

// DefaultSnapshotConfig returns default snapshot configuration
func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		SnapshotPath:     "./data/snapshots",
		Compression:      true,
		Encryption:       false,
		AutoSnapshot:     true,
		SnapshotInterval: 5 * time.Minute,
		RetentionCount:   10,
		RetentionDays:    30,
		MaxSnapshotSize:  1024 * 1024 * 1024, // 1GB
		BackupPath:       "./data/backups",
		NodeID:           fmt.Sprintf("snapshot-%d", time.Now().Unix()),
	}
}

// SnapshotManagerImpl implements SnapshotManager interface
type SnapshotManagerImpl struct {
	config     *SnapshotConfig
	store      DistributedStore
	logger     *zap.Logger
	mu         sync.RWMutex
	stats      *SnapshotStats
	ctx        context.Context
	cancel     context.CancelFunc
	snapshotCh chan *SnapshotInfo
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(config *SnapshotConfig, store DistributedStore) *SnapshotManagerImpl {
	if config == nil {
		config = DefaultSnapshotConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger := logger.Get()

	// Create snapshot directory if it doesn't exist
	if err := os.MkdirAll(config.SnapshotPath, 0755); err != nil {
		logger.Error("Failed to create snapshot directory",
			zap.String("path", config.SnapshotPath),
			zap.Error(err))
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(config.BackupPath, 0755); err != nil {
		logger.Error("Failed to create backup directory",
			zap.String("path", config.BackupPath),
			zap.Error(err))
	}

	manager := &SnapshotManagerImpl{
		config:     config,
		store:      store,
		snapshotCh: make(chan *SnapshotInfo, 1000), // Aumentado de 100 para 1000
		stats: &SnapshotStats{
			SnapshotFrequency: config.SnapshotInterval,
			SnapshotRetention: config.RetentionCount,
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}

	// Start background processes
	go manager.backgroundProcesses()

	if config.AutoSnapshot {
		go manager.autoSnapshot()
	}

	manager.logger.Info("Snapshot manager initialized",
		zap.String("snapshot_path", config.SnapshotPath),
		zap.Bool("compression", config.Compression),
		zap.Duration("auto_snapshot_interval", config.SnapshotInterval))

	return manager
}

// CreateSnapshot creates a new snapshot
func (sm *SnapshotManagerImpl) CreateSnapshot(ctx context.Context, snapshotType SnapshotType, baseSnapshotID string) (*SnapshotInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	startTime := time.Now()
	snapshotID := fmt.Sprintf("%s-%d", snapshotType, time.Now().Unix())

	sm.logger.Info("Creating snapshot",
		zap.String("snapshot_id", snapshotID),
		zap.String("type", string(snapshotType)),
		zap.String("base_snapshot_id", baseSnapshotID))

	// Get current state from store
	var state map[string]*VersionedState
	var err error

	if snapshotType == IncrementalSnapshot && baseSnapshotID != "" {
		state, err = sm.createIncrementalState(ctx, baseSnapshotID)
	} else {
		state, err = sm.captureFullState(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to capture state: %w", err)
	}

	// Create snapshot data
	snapshotData := &SnapshotData{
		Info: SnapshotInfo{
			ID:             snapshotID,
			Type:           snapshotType,
			CreatedAt:      time.Now(),
			KeysCount:      len(state),
			BaseSnapshotID: baseSnapshotID,
			NodeID:         sm.config.NodeID,
			Compression:    sm.config.Compression,
			Encrypted:      sm.config.Encryption,
			Meta: map[string]interface{}{
				"creation_duration": time.Since(startTime).String(),
				"created_by":        "snapshot-manager",
			},
		},
		State: state,
	}

	// Serialize and save snapshot
	filePath := filepath.Join(sm.config.SnapshotPath, snapshotID+".snap")
	size, err := sm.saveSnapshotToFile(snapshotData, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Update snapshot info
	snapshotData.Info.Size = size
	snapshotData.Info.Checksum = sm.calculateChecksum(snapshotData)

	// Update stats
	sm.updateSnapshotStats(&snapshotData.Info)

	duration := time.Since(startTime)
	sm.logger.Info("Snapshot created successfully",
		zap.String("snapshot_id", snapshotID),
		zap.String("type", string(snapshotType)),
		zap.Int("keys_count", len(state)),
		zap.Int64("size_bytes", size),
		zap.Duration("creation_duration", duration))

	return &snapshotData.Info, nil
}

// RestoreSnapshot restores state from a snapshot
func (sm *SnapshotManagerImpl) RestoreSnapshot(ctx context.Context, snapshotID string) error {
	startTime := time.Now()

	sm.logger.Info("Restoring snapshot",
		zap.String("snapshot_id", snapshotID))

	// Load snapshot file
	filePath := filepath.Join(sm.config.SnapshotPath, snapshotID+".snap")
	snapshotData, err := sm.loadSnapshotFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Verify checksum
	if snapshotData.Info.Checksum != sm.calculateChecksum(snapshotData) {
		return fmt.Errorf("snapshot checksum verification failed: %s", snapshotID)
	}

	// Restore state to store
	if err := sm.restoreStateToStore(ctx, snapshotData); err != nil {
		return fmt.Errorf("failed to restore state: %w", err)
	}

	duration := time.Since(startTime)
	sm.logger.Info("Snapshot restored successfully",
		zap.String("snapshot_id", snapshotID),
		zap.Int("keys_count", len(snapshotData.State)),
		zap.Duration("restore_duration", duration))

	return nil
}

// DeleteSnapshot deletes a snapshot
func (sm *SnapshotManagerImpl) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.logger.Info("Deleting snapshot",
		zap.String("snapshot_id", snapshotID))

	// Delete snapshot file
	filePath := filepath.Join(sm.config.SnapshotPath, snapshotID+".snap")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete snapshot file: %w", err)
	}

	// Update stats
	sm.stats.TotalSnapshots--
	if sm.stats.TotalSize > 0 {
		// Get file size before deletion
		if info, err := os.Stat(filePath); err == nil {
			sm.stats.TotalSize -= info.Size()
		}
	}

	sm.logger.Info("Snapshot deleted successfully",
		zap.String("snapshot_id", snapshotID))

	return nil
}

// ListSnapshots lists all available snapshots
func (sm *SnapshotManagerImpl) ListSnapshots(ctx context.Context) ([]*SnapshotInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	files, err := filepath.Glob(filepath.Join(sm.config.SnapshotPath, "*.snap"))
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot files: %w", err)
	}

	var snapshots []*SnapshotInfo
	for _, file := range files {
		snapshotData, err := sm.loadSnapshotFromFile(file)
		if err != nil {
			sm.logger.Error("Failed to load snapshot info",
				zap.String("file", file),
				zap.Error(err))
			continue
		}
		snapshots = append(snapshots, &snapshotData.Info)
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(snapshots)-1; i++ {
		for j := i + 1; j < len(snapshots); j++ {
			if snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt) {
				snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
			}
		}
	}

	return snapshots, nil
}

// GetSnapshotInfo returns information about a specific snapshot
func (sm *SnapshotManagerImpl) GetSnapshotInfo(ctx context.Context, snapshotID string) (*SnapshotInfo, error) {
	filePath := filepath.Join(sm.config.SnapshotPath, snapshotID+".snap")
	snapshotData, err := sm.loadSnapshotFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	return &snapshotData.Info, nil
}

// IncrementalSnapshot creates an incremental snapshot
func (sm *SnapshotManagerImpl) IncrementalSnapshot(ctx context.Context, baseSnapshotID string) (*SnapshotInfo, error) {
	return sm.CreateSnapshot(ctx, IncrementalSnapshot, baseSnapshotID)
}

// ScheduleAutoSnapshot schedules automatic snapshots
func (sm *SnapshotManagerImpl) ScheduleAutoSnapshot(ctx context.Context, interval time.Duration) error {
	sm.mu.Lock()
	sm.config.SnapshotInterval = interval
	sm.config.AutoSnapshot = true
	sm.mu.Unlock()

	// Restart auto snapshot with new interval
	go sm.autoSnapshot()

	sm.logger.Info("Auto snapshot scheduled",
		zap.Duration("interval", interval))

	return nil
}

// GetSnapshotStats returns snapshot statistics
func (sm *SnapshotManagerImpl) GetSnapshotStats() SnapshotStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := *sm.stats
	return stats
}

// Private helper methods

func (sm *SnapshotManagerImpl) captureFullState(ctx context.Context) (map[string]*VersionedState, error) {
	// Get all keys from store
	keys, err := sm.store.GetAllKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all keys: %w", err)
	}

	// Build state map by retrieving each key
	state := make(map[string]*VersionedState, len(keys))
	for _, key := range keys {
		versionedState, err := sm.store.Get(ctx, key)
		if err != nil {
			// Skip keys that no longer exist (race condition)
			sm.logger.Debug("Key not found during snapshot capture, skipping",
				zap.String("key", key),
				zap.Error(err))
			continue
		}
		state[key] = versionedState
	}

	sm.logger.Debug("Full state captured",
		zap.Int("keys_count", len(state)))

	return state, nil
}

func (sm *SnapshotManagerImpl) createIncrementalState(ctx context.Context, baseSnapshotID string) (map[string]*VersionedState, error) {
	// Load base snapshot
	basePath := filepath.Join(sm.config.SnapshotPath, baseSnapshotID+".snap")
	baseData, err := sm.loadSnapshotFromFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load base snapshot: %w", err)
	}

	// Get current state
	currentState, err := sm.captureFullState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to capture current state: %w", err)
	}

	// Calculate changes
	changes := make(map[string]*StateChange)
	incrementalState := make(map[string]*VersionedState)

	// Find updates and new keys
	for key, newState := range currentState {
		if oldState, exists := baseData.State[key]; exists {
			// Key exists in both, check for changes
			if !sm.statesEqual(oldState, newState) {
				changes[key] = &StateChange{
					OldState: oldState,
					NewState: newState,
					Type:     "updated",
					Time:     time.Now(),
					NodeID:   sm.config.NodeID,
				}
				incrementalState[key] = newState
			}
		} else {
			// New key
			changes[key] = &StateChange{
				OldState: nil,
				NewState: newState,
				Type:     "created",
				Time:     time.Now(),
				NodeID:   sm.config.NodeID,
			}
			incrementalState[key] = newState
		}
	}

	// Find deleted keys
	for key, oldState := range baseData.State {
		if _, exists := currentState[key]; !exists {
			changes[key] = &StateChange{
				OldState: oldState,
				NewState: nil,
				Type:     "deleted",
				Time:     time.Now(),
				NodeID:   sm.config.NodeID,
			}
		}
	}

	// Store changes in the incremental state metadata
	// In a real implementation, this would be stored separately

	return incrementalState, nil
}

func (sm *SnapshotManagerImpl) saveSnapshotToFile(snapshotData *SnapshotData, filePath string) (int64, error) {
	// Serialize snapshot data
	data, err := json.Marshal(snapshotData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Compress if enabled
	if sm.config.Compression {
		data, err = sm.compressData(data)
		if err != nil {
			return 0, fmt.Errorf("failed to compress data: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return 0, fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return int64(len(data)), nil
}

func (sm *SnapshotManagerImpl) loadSnapshotFromFile(filePath string) (*SnapshotData, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	// Decompress if needed
	if sm.config.Compression {
		data, err = sm.decompressData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
	}

	// Deserialize
	var snapshotData SnapshotData
	if err := json.Unmarshal(data, &snapshotData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshotData, nil
}

func (sm *SnapshotManagerImpl) restoreStateToStore(ctx context.Context, snapshotData *SnapshotData) error {
	// For each state entry, restore to store
	for key, state := range snapshotData.State {
		if _, err := sm.store.Set(ctx, key, state.Value, state.TTL); err != nil {
			return fmt.Errorf("failed to restore key %s: %w", key, err)
		}
	}

	return nil
}

func (sm *SnapshotManagerImpl) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (sm *SnapshotManagerImpl) decompressData(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

func (sm *SnapshotManagerImpl) calculateChecksum(snapshotData *SnapshotData) string {
	// Simple checksum - in a real implementation, use SHA-256 or similar
	data, _ := json.Marshal(snapshotData)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (sm *SnapshotManagerImpl) statesEqual(state1, state2 *VersionedState) bool {
	if state1.Version != state2.Version {
		return false
	}
	// In a real implementation, compare values more thoroughly
	return true
}

func (sm *SnapshotManagerImpl) updateSnapshotStats(snapshotInfo *SnapshotInfo) {
	sm.stats.TotalSnapshots++
	sm.stats.TotalSize += snapshotInfo.Size

	switch snapshotInfo.Type {
	case FullSnapshot:
		sm.stats.FullSnapshots++
	case IncrementalSnapshot:
		sm.stats.IncrementalSnapshots++
	}

	if sm.stats.TotalSnapshots > 0 {
		sm.stats.AverageSize = sm.stats.TotalSize / sm.stats.TotalSnapshots
	}

	sm.stats.LastSnapshot = &time.Time{}
	*sm.stats.LastSnapshot = snapshotInfo.CreatedAt
}

func (sm *SnapshotManagerImpl) backgroundProcesses() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.cleanupOldSnapshots()
		}
	}
}

func (sm *SnapshotManagerImpl) autoSnapshot() {
	ticker := time.NewTicker(sm.config.SnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			if _, err := sm.CreateSnapshot(sm.ctx, AutoSnapshot, ""); err != nil {
				sm.logger.Error("Auto snapshot failed", zap.Error(err))
			}
		}
	}
}

func (sm *SnapshotManagerImpl) cleanupOldSnapshots() {
	snapshots, err := sm.ListSnapshots(sm.ctx)
	if err != nil {
		sm.logger.Error("Failed to list snapshots for cleanup", zap.Error(err))
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -sm.config.RetentionDays)

	for _, snapshot := range snapshots {
		if snapshot.CreatedAt.Before(cutoffTime) {
			if err := sm.DeleteSnapshot(sm.ctx, snapshot.ID); err != nil {
				sm.logger.Error("Failed to delete old snapshot",
					zap.String("snapshot_id", snapshot.ID),
					zap.Error(err))
			}
		}
	}
}

// Close closes the snapshot manager
func (sm *SnapshotManagerImpl) Close() error {
	sm.cancel()
	close(sm.snapshotCh)

	sm.logger.Info("Snapshot manager closed")
	return nil
}
