package fulfillment

import (
	"time"

	"github.com/google/uuid"
)

// TransferOrder: Transferência entre locais (CD↔loja, loja↔loja)
type TransferOrder struct {
	ID             string     `json:"id"`
	LocationFrom   string     `json:"location_from"`
	LocationTo     string     `json:"location_to"`
	Status         Status     `json:"status"`
	Items          []Item     `json:"items"`
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// NewTransferOrder cria uma nova instância de TransferOrder
func NewTransferOrder(locationFrom, locationTo string, items []Item) (*TransferOrder, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	now := time.Now()
	id := uuid.New().String()
	return &TransferOrder{
		ID:             id,
		LocationFrom:   locationFrom,
		LocationTo:     locationTo,
		Status:         StatusPending,
		Items:          items,
		IdempotencyKey: id,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// StartTransfer inicia o processo de transferência
func (t *TransferOrder) StartTransfer() error {
	if t.Status != StatusPending {
		return ErrInvalidStateTransition
	}
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now()
	return nil
}

// Complete finaliza a transferência
func (t *TransferOrder) Complete() error {
	if t.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	now := time.Now()
	t.Status = StatusCompleted
	t.UpdatedAt = now
	t.CompletedAt = &now
	return nil
}

// Cancel cancela a transferência
func (t *TransferOrder) Cancel() error {
	if t.Status == StatusCompleted {
		return ErrInvalidStateTransition
	}
	t.Status = StatusCancelled
	t.UpdatedAt = time.Now()
	return nil
}
