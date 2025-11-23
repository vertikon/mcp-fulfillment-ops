package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// EventPublisher implementa o contrato EventPublisher para publicação de eventos NATS
type EventPublisher struct {
	js     jetstream.JetStream
	logger Logger
}

// Logger is defined in logger_adapter.go

// NewEventPublisher cria uma nova instância do publisher
func NewEventPublisher(js jetstream.JetStream, logger Logger) *EventPublisher {
	return &EventPublisher{
		js:     js,
		logger: logger,
	}
}

// PublishInboundReceived publica evento de recebimento confirmado
func (p *EventPublisher) PublishInboundReceived(ctx context.Context, shipment *fulfillment.InboundShipment) error {
	event := map[string]interface{}{
		"shipment_id":   shipment.ID,
		"reference_id":  shipment.ReferenceID,
		"destination":   shipment.Destination,
		"items":         shipment.Items,
		"completed_at":  shipment.CompletedAt,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.inbound.received.v1", event)
}

// PublishOutboundShipped publica evento de expedição confirmada
func (p *EventPublisher) PublishOutboundShipped(ctx context.Context, order *fulfillment.FulfillmentOrder) error {
	event := map[string]interface{}{
		"order_id":      order.ID,
		"oms_order_id":  order.OrderID,
		"destination":   order.Destination,
		"items":         order.Items,
		"shipped_at":    order.ShippedAt,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.outbound.shipped.v1", event)
}

// PublishPickingStarted publica evento de início de picking
func (p *EventPublisher) PublishPickingStarted(ctx context.Context, order *fulfillment.FulfillmentOrder) error {
	event := map[string]interface{}{
		"order_id":      order.ID,
		"oms_order_id":  order.OrderID,
		"priority":      order.Priority,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.outbound.picking_started.v1", event)
}

// PublishReturnRegistered publica evento de devolução registrada
func (p *EventPublisher) PublishReturnRegistered(ctx context.Context, returnOrder *fulfillment.ReturnOrder) error {
	event := map[string]interface{}{
		"return_id":         returnOrder.ID,
		"original_order_id": returnOrder.OriginalOrderID,
		"reason":            returnOrder.Reason,
		"items":             returnOrder.Items,
		"timestamp":         time.Now().UTC(),
		"event_version":     "v1",
	}

	return p.publishEvent(ctx, "fulfillment.return.registered.v1", event)
}

// PublishReturnCompleted publica evento de devolução completada
func (p *EventPublisher) PublishReturnCompleted(ctx context.Context, returnOrder *fulfillment.ReturnOrder) error {
	event := map[string]interface{}{
		"return_id":         returnOrder.ID,
		"original_order_id": returnOrder.OriginalOrderID,
		"completed_at":      returnOrder.CompletedAt,
		"timestamp":         time.Now().UTC(),
		"event_version":     "v1",
	}

	return p.publishEvent(ctx, "fulfillment.return.completed.v1", event)
}

// PublishTransferCreated publica evento de transferência criada
func (p *EventPublisher) PublishTransferCreated(ctx context.Context, transfer *fulfillment.TransferOrder) error {
	event := map[string]interface{}{
		"transfer_id":   transfer.ID,
		"location_from": transfer.LocationFrom,
		"location_to":   transfer.LocationTo,
		"items":         transfer.Items,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.transfer.created.v1", event)
}

// PublishTransferCompleted publica evento de transferência completada
func (p *EventPublisher) PublishTransferCompleted(ctx context.Context, transfer *fulfillment.TransferOrder) error {
	event := map[string]interface{}{
		"transfer_id":   transfer.ID,
		"location_from": transfer.LocationFrom,
		"location_to":   transfer.LocationTo,
		"completed_at":  transfer.CompletedAt,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.transfer.completed.v1", event)
}

// PublishCycleCountOpened publica evento de contagem cíclica aberta
func (p *EventPublisher) PublishCycleCountOpened(ctx context.Context, task *fulfillment.CycleCountTask) error {
	event := map[string]interface{}{
		"task_id":       task.ID,
		"location":      task.Location,
		"skus":          task.SKUs,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.cycle_count.opened.v1", event)
}

// PublishCycleCountCompleted publica evento de contagem cíclica completada
func (p *EventPublisher) PublishCycleCountCompleted(ctx context.Context, task *fulfillment.CycleCountTask) error {
	event := map[string]interface{}{
		"task_id":       task.ID,
		"location":      task.Location,
		"counted_items": task.CountedItems,
		"completed_at":  task.CompletedAt,
		"timestamp":     time.Now().UTC(),
		"event_version": "v1",
	}

	return p.publishEvent(ctx, "fulfillment.cycle_count.completed.v1", event)
}

// publishEvent publica um evento no NATS JetStream
func (p *EventPublisher) publishEvent(ctx context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = p.js.Publish(ctx, subject, data)
	if err != nil {
		p.logger.Error("Failed to publish event", zap.String("subject", subject), zap.Error(err))
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Info("Event published", zap.String("subject", subject))
	return nil
}
