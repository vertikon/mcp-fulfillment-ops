package fulfillment

import (
	"time"

	"github.com/google/uuid"
)

// InboundShipment: Recebimento de Mercadoria (Compras/Transferência)
type InboundShipment struct {
	ID             string    `json:"id"`
	ReferenceID    string    `json:"reference_id"` // Ex: ID do Pedido de Compra (B7)
	Origin         string    `json:"origin"`       // Ex: Fornecedor X
	Destination    string    `json:"destination"`  // Ex: CD-SP
	Status         Status    `json:"status"`
	Items          []Item    `json:"items"`
	IdempotencyKey string    `json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// NewInboundShipment cria uma nova instância de InboundShipment
func NewInboundShipment(refID, origin, dest string, items []Item) (*InboundShipment, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}
	now := time.Now()
	return &InboundShipment{
		ID:             uuid.New().String(),
		ReferenceID:    refID,
		Origin:         origin,
		Destination:    dest,
		Status:         StatusPending,
		Items:          items,
		IdempotencyKey: refID, // Usa ReferenceID como chave de idempotência
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// StartReceiving inicia o processo de recebimento físico
func (i *InboundShipment) StartReceiving() error {
	if i.Status != StatusPending {
		return ErrInvalidStateTransition
	}
	i.Status = StatusInProgress
	i.UpdatedAt = time.Now()
	return nil
}

// Complete finaliza o recebimento físico
func (i *InboundShipment) Complete() error {
	if i.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	now := time.Now()
	i.Status = StatusCompleted
	i.UpdatedAt = now
	i.CompletedAt = &now
	return nil
}

// Cancel cancela o recebimento
func (i *InboundShipment) Cancel() error {
	if i.Status == StatusCompleted {
		return ErrInvalidStateTransition
	}
	i.Status = StatusCancelled
	i.UpdatedAt = time.Now()
	return nil
}

