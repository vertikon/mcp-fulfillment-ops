package app

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// ReceiveGoodsUseCase orquestra o recebimento físico de mercadorias
type ReceiveGoodsUseCase struct {
	repo            fulfillment.Repository
	inventoryClient InventoryClient
	eventPublisher  EventPublisher
	logger          Logger
}

// NewReceiveGoodsUseCase cria uma nova instância do caso de uso
func NewReceiveGoodsUseCase(repo fulfillment.Repository, inventoryClient InventoryClient, eventPublisher EventPublisher, logger Logger) *ReceiveGoodsUseCase {
	return &ReceiveGoodsUseCase{
		repo:            repo,
		inventoryClient: inventoryClient,
		eventPublisher:  eventPublisher,
		logger:          logger,
	}
}

// StartInbound cria um novo InboundShipment (previsto)
func (uc *ReceiveGoodsUseCase) StartInbound(ctx context.Context, refID, origin, dest string, items []fulfillment.Item) (*fulfillment.InboundShipment, error) {
	shipment, err := fulfillment.NewInboundShipment(refID, origin, dest, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create inbound shipment: %w", err)
	}

	// Verifica idempotência
	existing, err := uc.repo.GetInboundByReferenceID(ctx, refID)
	if err == nil && existing != nil {
		uc.logger.Warn("Inbound shipment already exists (idempotency)", "reference_id", refID)
		return existing, nil
	}

	if err := uc.repo.CreateInbound(ctx, shipment); err != nil {
		return nil, fmt.Errorf("failed to persist inbound shipment: %w", err)
	}

	uc.logger.Info("Inbound shipment created", "id", shipment.ID, "reference_id", refID)
	return shipment, nil
}

// ConfirmReceipt confirma o recebimento físico e chama o Core Inventory
func (uc *ReceiveGoodsUseCase) ConfirmReceipt(ctx context.Context, shipmentID string) error {
	shipment, err := uc.repo.GetInboundByID(ctx, shipmentID)
	if err != nil {
		return fmt.Errorf("failed to get inbound shipment: %w", err)
	}

	// Valida estado
	if err := shipment.StartReceiving(); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	if err := uc.repo.UpdateInbound(ctx, shipment); err != nil {
		return fmt.Errorf("failed to update inbound status: %w", err)
	}

	// Chama mcp-core-inventory para entrada de estoque
	for _, item := range shipment.Items {
		if err := uc.inventoryClient.AdjustStock(ctx, shipment.Destination, item.SKU, item.Quantity, item.Batch); err != nil {
			uc.logger.Error("Failed to adjust stock in core inventory", "error", err, "sku", item.SKU)
			// Marca como failed
			shipment.Status = fulfillment.StatusFailed
			uc.repo.UpdateInbound(ctx, shipment)
			return fmt.Errorf("failed to adjust stock for SKU %s: %w", item.SKU, err)
		}
	}

	// Completa o recebimento
	if err := shipment.Complete(); err != nil {
		return fmt.Errorf("failed to complete shipment: %w", err)
	}

	if err := uc.repo.UpdateInbound(ctx, shipment); err != nil {
		return fmt.Errorf("failed to update inbound status: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishInboundReceived(ctx, shipment); err != nil {
		uc.logger.Error("Failed to publish inbound received event", "error", err)
		// Não falha a operação se o evento não for publicado
	}

	uc.logger.Info("Inbound shipment confirmed", "id", shipmentID)
	return nil
}

