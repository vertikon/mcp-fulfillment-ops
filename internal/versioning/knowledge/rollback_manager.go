package knowledge

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// RollbackOperation represents a rollback operation
type RollbackOperation struct {
	ID            string                 `json:"id"`
	SourceVersion string                 `json:"source_version"`
	TargetVersion string                 `json:"target_version"`
	Status        RollbackStatus         `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RollbackStatus represents the status of a rollback
type RollbackStatus string

const (
	RollbackStatusPending   RollbackStatus = "pending"
	RollbackStatusRunning   RollbackStatus = "running"
	RollbackStatusCompleted RollbackStatus = "completed"
	RollbackStatusFailed    RollbackStatus = "failed"
)

// RollbackManager interface for managing rollbacks
type RollbackManager interface {
	// RollbackToVersion rolls back to a specific version
	RollbackToVersion(ctx context.Context, knowledgeID string, targetVersionID string) (*RollbackOperation, error)

	// GetRollbackOperation retrieves a rollback operation
	GetRollbackOperation(ctx context.Context, operationID string) (*RollbackOperation, error)

	// ListRollbackOperations lists rollback operations for a knowledge base
	ListRollbackOperations(ctx context.Context, knowledgeID string) ([]*RollbackOperation, error)

	// ValidateRollback validates if a rollback is safe
	ValidateRollback(ctx context.Context, knowledgeID string, targetVersionID string) error

	// CancelRollback cancels a pending rollback
	CancelRollback(ctx context.Context, operationID string) error
}

// InMemoryRollbackManager implements RollbackManager
type InMemoryRollbackManager struct {
	versioning KnowledgeVersioning
	operations map[string]*RollbackOperation
	mu         sync.RWMutex
	logger     *zap.Logger
}

var (
	_ RollbackManager = (*InMemoryRollbackManager)(nil)
)

// NewInMemoryRollbackManager creates a new rollback manager
func NewInMemoryRollbackManager(versioning KnowledgeVersioning) *InMemoryRollbackManager {
	return &InMemoryRollbackManager{
		versioning: versioning,
		operations: make(map[string]*RollbackOperation),
		logger:     logger.WithContext(context.Background()),
	}
}

// RollbackToVersion rolls back to a specific version
func (rm *InMemoryRollbackManager) RollbackToVersion(ctx context.Context, knowledgeID string, targetVersionID string) (*RollbackOperation, error) {
	logger := logger.WithContext(ctx)
	logger.Info("Starting rollback",
		zap.String("knowledge_id", knowledgeID),
		zap.String("target_version_id", targetVersionID))

	// Validate rollback
	if err := rm.ValidateRollback(ctx, knowledgeID, targetVersionID); err != nil {
		return nil, fmt.Errorf("rollback validation failed: %w", err)
	}

	// Get current version
	currentVersion, err := rm.versioning.GetLatestVersion(ctx, knowledgeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// Get target version
	targetVersion, err := rm.versioning.GetVersion(ctx, targetVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target version: %w", err)
	}

	// Verify knowledge IDs match
	if targetVersion.KnowledgeID != knowledgeID {
		return nil, fmt.Errorf("target version does not belong to knowledge base %s", knowledgeID)
	}

	// Create rollback operation
	operation := &RollbackOperation{
		ID:            fmt.Sprintf("rollback-%d", time.Now().UnixNano()),
		SourceVersion: currentVersion.ID,
		TargetVersion: targetVersionID,
		Status:        RollbackStatusPending,
		CreatedAt:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	rm.mu.Lock()
	rm.operations[operation.ID] = operation
	rm.mu.Unlock()

	// Execute rollback
	operation.Status = RollbackStatusRunning
	if err := rm.executeRollback(ctx, knowledgeID, targetVersionID); err != nil {
		operation.Status = RollbackStatusFailed
		operation.Error = err.Error()
		rm.mu.Lock()
		rm.operations[operation.ID] = operation
		rm.mu.Unlock()
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	// Mark as completed
	now := time.Now()
	operation.Status = RollbackStatusCompleted
	operation.CompletedAt = &now

	rm.mu.Lock()
	rm.operations[operation.ID] = operation
	rm.mu.Unlock()

	logger.Info("Rollback completed",
		zap.String("operation_id", operation.ID),
		zap.String("target_version_id", targetVersionID))

	return operation, nil
}

// GetRollbackOperation retrieves a rollback operation
func (rm *InMemoryRollbackManager) GetRollbackOperation(ctx context.Context, operationID string) (*RollbackOperation, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	operation, exists := rm.operations[operationID]
	if !exists {
		return nil, fmt.Errorf("rollback operation %s not found", operationID)
	}

	return operation, nil
}

// ListRollbackOperations lists rollback operations for a knowledge base
func (rm *InMemoryRollbackManager) ListRollbackOperations(ctx context.Context, knowledgeID string) ([]*RollbackOperation, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var operations []*RollbackOperation
	for _, op := range rm.operations {
		// Get source version to check knowledge ID
		sourceVersion, err := rm.versioning.GetVersion(ctx, op.SourceVersion)
		if err == nil && sourceVersion.KnowledgeID == knowledgeID {
			operations = append(operations, op)
		}
	}

	return operations, nil
}

// ValidateRollback validates if a rollback is safe
func (rm *InMemoryRollbackManager) ValidateRollback(ctx context.Context, knowledgeID string, targetVersionID string) error {
	// Get target version
	targetVersion, err := rm.versioning.GetVersion(ctx, targetVersionID)
	if err != nil {
		return fmt.Errorf("target version not found: %w", err)
	}

	// Verify knowledge ID matches
	if targetVersion.KnowledgeID != knowledgeID {
		return fmt.Errorf("target version does not belong to knowledge base %s", knowledgeID)
	}

	// Check if version is deleted
	if deleted, ok := targetVersion.Metadata["deleted"].(bool); ok && deleted {
		return fmt.Errorf("cannot rollback to deleted version")
	}

	// Get current version
	currentVersion, err := rm.versioning.GetLatestVersion(ctx, knowledgeID)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Check if already at target version
	if currentVersion.ID == targetVersionID {
		return fmt.Errorf("already at target version")
	}

	return nil
}

// CancelRollback cancels a pending rollback
func (rm *InMemoryRollbackManager) CancelRollback(ctx context.Context, operationID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	operation, exists := rm.operations[operationID]
	if !exists {
		return fmt.Errorf("rollback operation %s not found", operationID)
	}

	if operation.Status != RollbackStatusPending {
		return fmt.Errorf("cannot cancel rollback in status %s", operation.Status)
	}

	operation.Status = RollbackStatusFailed
	operation.Error = "cancelled by user"
	rm.operations[operationID] = operation

	return nil
}

// executeRollback executes the actual rollback
func (rm *InMemoryRollbackManager) executeRollback(ctx context.Context, knowledgeID string, targetVersionID string) error {
	logger := logger.WithContext(ctx)
	logger.Info("Executing rollback",
		zap.String("knowledge_id", knowledgeID),
		zap.String("target_version_id", targetVersionID))

	// In a real implementation, this would:
	// 1. Create a new version based on target version
	// 2. Copy all documents from target version
	// 3. Update knowledge base to use new version
	// 4. Validate integrity

	// For now, we just validate that the target version exists and is accessible
	_, err := rm.versioning.GetVersion(ctx, targetVersionID)
	if err != nil {
		return err
	}

	// Verify documents exist
	docs, err := rm.versioning.ListDocuments(ctx, targetVersionID)
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		logger.Warn("Rolling back to version with no documents",
			zap.String("target_version_id", targetVersionID))
	}

	logger.Info("Rollback execution completed",
		zap.String("target_version_id", targetVersionID),
		zap.Int("document_count", len(docs)))

	return nil
}
