package app

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// ShipOrderUseCase orquestra a expedição física de pedidos
type ShipOrderUseCase struct {
	repo            fulfillment.Repository
	inventoryClient InventoryClient
	eventPublisher  EventPublisher
	logger          Logger
}

// NewShipOrderUseCase cria uma nova instância do caso de uso
func NewShipOrderUseCase(repo fulfillment.Repository, inventoryClient InventoryClient, eventPublisher EventPublisher, logger Logger) *ShipOrderUseCase {
	return &ShipOrderUseCase{
		repo:            repo,
		inventoryClient: inventoryClient,
		eventPublisher:  eventPublisher,
		logger:          logger,
	}
}

// CreateOrder cria uma nova FulfillmentOrder a partir de um evento OMS
func (uc *ShipOrderUseCase) CreateOrder(ctx context.Context, orderID, customer, destination string, items []fulfillment.Item, priority int) (*fulfillment.FulfillmentOrder, error) {
	order, err := fulfillment.NewFulfillmentOrder(orderID, customer, destination, items, priority)
	if err != nil {
		return nil, fmt.Errorf("failed to create fulfillment order: %w", err)
	}

	// Verifica idempotência
	existing, err := uc.repo.GetOrderByOrderID(ctx, orderID)
	if err == nil && existing != nil {
		uc.logger.Warn("Fulfillment order already exists (idempotency)", "order_id", orderID)
		return existing, nil
	}

	if err := uc.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to persist fulfillment order: %w", err)
	}

	uc.logger.Info("Fulfillment order created", "id", order.ID, "order_id", orderID)
	return order, nil
}

// StartPicking inicia o processo de separação (picking)
func (uc *ShipOrderUseCase) StartPicking(ctx context.Context, orderID string) error {
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get fulfillment order: %w", err)
	}

	if err := order.StartPicking(); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishPickingStarted(ctx, order); err != nil {
		uc.logger.Error("Failed to publish picking started event", "error", err)
	}

	uc.logger.Info("Picking started", "order_id", orderID)
	return nil
}

// Ship confirma a expedição física e chama o Core Inventory
func (uc *ShipOrderUseCase) Ship(ctx context.Context, orderID string) error {
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get fulfillment order: %w", err)
	}

	// Valida que está em progresso (picking/packing completo)
	if order.Status != fulfillment.StatusInProgress {
		return fmt.Errorf("order must be in progress to ship, current status: %s", order.Status)
	}

	// Chama mcp-core-inventory para confirmar reservas e aplicar baixa definitiva
	if err := uc.inventoryClient.ConfirmReservation(ctx, order.OrderID, order.Items); err != nil {
		uc.logger.Error("Failed to confirm reservation in core inventory", "error", err)
		order.Status = fulfillment.StatusFailed
		uc.repo.UpdateOrder(ctx, order)
		return fmt.Errorf("failed to confirm reservation: %w", err)
	}

	// Completa a expedição
	if err := order.Ship(); err != nil {
		return fmt.Errorf("failed to ship order: %w", err)
	}

	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishOutboundShipped(ctx, order); err != nil {
		uc.logger.Error("Failed to publish outbound shipped event", "error", err)
		// Não falha a operação se o evento não for publicado
	}

	uc.logger.Info("Order shipped", "order_id", orderID)
	return nil
}
