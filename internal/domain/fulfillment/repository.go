package fulfillment

import "context"

// Repository define a interface de persistência para entidades de fulfillment
type Repository interface {
	// Inbound
	CreateInbound(ctx context.Context, shipment *InboundShipment) error
	GetInboundByID(ctx context.Context, id string) (*InboundShipment, error)
	GetInboundByReferenceID(ctx context.Context, referenceID string) (*InboundShipment, error)
	UpdateInboundStatus(ctx context.Context, id string, status Status) error
	UpdateInbound(ctx context.Context, shipment *InboundShipment) error

	// Outbound
	CreateOrder(ctx context.Context, order *FulfillmentOrder) error
	GetOrderByID(ctx context.Context, id string) (*FulfillmentOrder, error)
	GetOrderByOrderID(ctx context.Context, orderID string) (*FulfillmentOrder, error)
	UpdateOrderStatus(ctx context.Context, id string, status Status) error
	UpdateOrder(ctx context.Context, order *FulfillmentOrder) error

	// Transfer
	CreateTransfer(ctx context.Context, transfer *TransferOrder) error
	GetTransferByID(ctx context.Context, id string) (*TransferOrder, error)
	UpdateTransferStatus(ctx context.Context, id string, status Status) error
	UpdateTransfer(ctx context.Context, transfer *TransferOrder) error

	// Return
	CreateReturn(ctx context.Context, returnOrder *ReturnOrder) error
	GetReturnByID(ctx context.Context, id string) (*ReturnOrder, error)
	UpdateReturnStatus(ctx context.Context, id string, status Status) error
	UpdateReturn(ctx context.Context, returnOrder *ReturnOrder) error

	// Cycle Count
	CreateCycleCount(ctx context.Context, task *CycleCountTask) error
	GetCycleCountByID(ctx context.Context, id string) (*CycleCountTask, error)
	UpdateCycleCountStatus(ctx context.Context, id string, status Status) error
	UpdateCycleCount(ctx context.Context, task *CycleCountTask) error
}
