package fulfillment

import (
	"time"

	"github.com/google/uuid"
)

// FulfillmentOrder: Expedição de Venda (Outbound)
type FulfillmentOrder struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"` // Ex: ID do Pedido OMS (B10)
	Customer       string    `json:"customer"`
	Destination    string    `json:"destination"` // Endereço
	Status         Status    `json:"status"`
	Items          []Item    `json:"items"`
	Priority       int       `json:"priority"` // 0-Normal, 1-Express
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ShippedAt      *time.Time `json:"shipped_at,omitempty"`
}

// NewFulfillmentOrder cria uma nova instância de FulfillmentOrder
func NewFulfillmentOrder(orderID, customer, destination string, items []Item, priority int) (*FulfillmentOrder, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	now := time.Now()
	return &FulfillmentOrder{
		ID:             uuid.New().String(),
		OrderID:        orderID,
		Customer:       customer,
		Destination:    destination,
		Status:         StatusPending,
		Items:          items,
		Priority:       priority,
		IdempotencyKey: orderID, // Usa OrderID como chave de idempotência
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// StartPicking inicia o processo de separação (picking)
func (f *FulfillmentOrder) StartPicking() error {
	if f.Status != StatusPending {
		return ErrInvalidStateTransition
	}
	f.Status = StatusInProgress
	f.UpdatedAt = time.Now()
	return nil
}

// Ship confirma a expedição física
func (f *FulfillmentOrder) Ship() error {
	if f.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	now := time.Now()
	f.Status = StatusCompleted
	f.UpdatedAt = now
	f.ShippedAt = &now
	return nil
}

// Cancel cancela a ordem de fulfillment
func (f *FulfillmentOrder) Cancel() error {
	if f.Status == StatusCompleted {
		return ErrInvalidStateTransition
	}
	f.Status = StatusCancelled
	f.UpdatedAt = time.Now()
	return nil
}

