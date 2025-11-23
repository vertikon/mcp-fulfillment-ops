package fulfillment

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// CycleCountTask: Tarefa de contagem cíclica/inventário físico
type CycleCountTask struct {
	ID             string     `json:"id"`
	Location       string     `json:"location"`
	SKUs           []string   `json:"skus"` // Lista de SKUs para contagem
	Status         Status     `json:"status"`
	CountedItems   []Item     `json:"counted_items"` // Itens contados fisicamente
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// NewCycleCountTask cria uma nova instância de CycleCountTask
func NewCycleCountTask(location string, skus []string) (*CycleCountTask, error) {
	if len(skus) == 0 {
		return nil, errors.New("cycle count task must have at least one SKU")
	}
	now := time.Now()
	id := uuid.New().String()
	return &CycleCountTask{
		ID:             id,
		Location:       location,
		SKUs:           skus,
		Status:         StatusPending,
		CountedItems:   []Item{},
		IdempotencyKey: id,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// StartCounting inicia a contagem física
func (c *CycleCountTask) StartCounting() error {
	if c.Status != StatusPending {
		return ErrInvalidStateTransition
	}
	c.Status = StatusInProgress
	c.UpdatedAt = time.Now()
	return nil
}

// SubmitCount registra os itens contados
func (c *CycleCountTask) SubmitCount(items []Item) error {
	if c.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	c.CountedItems = items
	c.UpdatedAt = time.Now()
	return nil
}

// Complete finaliza a contagem
func (c *CycleCountTask) Complete() error {
	if c.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	now := time.Now()
	c.Status = StatusCompleted
	c.UpdatedAt = now
	c.CompletedAt = &now
	return nil
}

// Cancel cancela a tarefa de contagem
func (c *CycleCountTask) Cancel() error {
	if c.Status == StatusCompleted {
		return ErrInvalidStateTransition
	}
	c.Status = StatusCancelled
	c.UpdatedAt = time.Now()
	return nil
}
