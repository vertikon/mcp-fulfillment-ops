package fulfillment

import (
	"time"

	"github.com/google/uuid"
)

// ReturnOrder: Logística Reversa
type ReturnOrder struct {
	ID             string    `json:"id"`
	OriginalOrderID string   `json:"original_order_id"`
	Reason         string    `json:"reason"`
	Status         Status    `json:"status"`
	Items          []Item    `json:"items"`
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// NewReturnOrder cria uma nova instância de ReturnOrder
func NewReturnOrder(originalOrderID, reason string, items []Item) (*ReturnOrder, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	now := time.Now()
	id := uuid.New().String()
	return &ReturnOrder{
		ID:             id,
		OriginalOrderID: originalOrderID,
		Reason:         reason,
		Status:         StatusPending,
		Items:          items,
		IdempotencyKey: id,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// StartProcessing inicia o processamento da devolução
func (r *ReturnOrder) StartProcessing() error {
	if r.Status != StatusPending {
		return ErrInvalidStateTransition
	}
	r.Status = StatusInProgress
	r.UpdatedAt = time.Now()
	return nil
}

// Complete finaliza a devolução
func (r *ReturnOrder) Complete() error {
	if r.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	now := time.Now()
	r.Status = StatusCompleted
	r.UpdatedAt = now
	r.CompletedAt = &now
	return nil
}

// Cancel cancela a devolução
func (r *ReturnOrder) Cancel() error {
	if r.Status == StatusCompleted {
		return ErrInvalidStateTransition
	}
	r.Status = StatusCancelled
	r.UpdatedAt = time.Now()
	return nil
}

