package app

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// InventoryClient define o contrato para comunicação com mcp-core-inventory
type InventoryClient interface {
	AdjustStock(ctx context.Context, location string, sku string, quantity int, batch string) error
	ConfirmReservation(ctx context.Context, orderID string, items []fulfillment.Item) error
	GetAvailableStock(ctx context.Context, location string, sku string) (int, error)
}

// EventPublisher define o contrato para publicação de eventos
type EventPublisher interface {
	PublishInboundReceived(ctx context.Context, shipment *fulfillment.InboundShipment) error
	PublishOutboundShipped(ctx context.Context, order *fulfillment.FulfillmentOrder) error
	PublishPickingStarted(ctx context.Context, order *fulfillment.FulfillmentOrder) error
	PublishReturnRegistered(ctx context.Context, returnOrder *fulfillment.ReturnOrder) error
	PublishReturnCompleted(ctx context.Context, returnOrder *fulfillment.ReturnOrder) error
	PublishTransferCreated(ctx context.Context, transfer *fulfillment.TransferOrder) error
	PublishTransferCompleted(ctx context.Context, transfer *fulfillment.TransferOrder) error
	PublishCycleCountOpened(ctx context.Context, task *fulfillment.CycleCountTask) error
	PublishCycleCountCompleted(ctx context.Context, task *fulfillment.CycleCountTask) error
}

// Logger define o contrato para logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

