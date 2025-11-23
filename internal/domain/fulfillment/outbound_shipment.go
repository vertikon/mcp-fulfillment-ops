package fulfillment

import (
	"time"

	"github.com/google/uuid"
)

// OutboundShipment: Expedição de Saída (pode ser usado para tracking detalhado)
type OutboundShipment struct {
	ID                 string     `json:"id"`
	FulfillmentOrderID string     `json:"fulfillment_order_id"`
	TrackingNumber     string     `json:"tracking_number,omitempty"`
	Carrier            string     `json:"carrier,omitempty"`
	Status             Status     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ShippedAt          *time.Time `json:"shipped_at,omitempty"`
}

// NewOutboundShipment cria uma nova instância de OutboundShipment
func NewOutboundShipment(fulfillmentOrderID, trackingNumber, carrier string) *OutboundShipment {
	now := time.Now()
	return &OutboundShipment{
		ID:                 uuid.New().String(),
		FulfillmentOrderID: fulfillmentOrderID,
		TrackingNumber:     trackingNumber,
		Carrier:            carrier,
		Status:             StatusPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Ship confirma a expedição
func (o *OutboundShipment) Ship() error {
	if o.Status != StatusPending && o.Status != StatusInProgress {
		return ErrInvalidStateTransition
	}
	now := time.Now()
	o.Status = StatusCompleted
	o.UpdatedAt = now
	o.ShippedAt = &now
	return nil
}
