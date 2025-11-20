package app

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// CompleteTransferUseCase orquestra transferências entre locais
type CompleteTransferUseCase struct {
	repo            fulfillment.Repository
	inventoryClient InventoryClient
	eventPublisher  EventPublisher
	logger          Logger
}

// NewCompleteTransferUseCase cria uma nova instância do caso de uso
func NewCompleteTransferUseCase(repo fulfillment.Repository, inventoryClient InventoryClient, eventPublisher EventPublisher, logger Logger) *CompleteTransferUseCase {
	return &CompleteTransferUseCase{
		repo:            repo,
		inventoryClient: inventoryClient,
		eventPublisher:  eventPublisher,
		logger:          logger,
	}
}

// CreateTransfer cria uma nova TransferOrder
func (uc *CompleteTransferUseCase) CreateTransfer(ctx context.Context, locationFrom, locationTo string, items []fulfillment.Item) (*fulfillment.TransferOrder, error) {
	transfer, err := fulfillment.NewTransferOrder(locationFrom, locationTo, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer order: %w", err)
	}

	if err := uc.repo.CreateTransfer(ctx, transfer); err != nil {
		return nil, fmt.Errorf("failed to persist transfer order: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishTransferCreated(ctx, transfer); err != nil {
		uc.logger.Error("Failed to publish transfer created event", "error", err)
	}

	uc.logger.Info("Transfer order created", "id", transfer.ID, "from", locationFrom, "to", locationTo)
	return transfer, nil
}

// CompleteTransfer finaliza a transferência (baixa origem + entrada destino)
func (uc *CompleteTransferUseCase) CompleteTransfer(ctx context.Context, transferID string) error {
	transfer, err := uc.repo.GetTransferByID(ctx, transferID)
	if err != nil {
		return fmt.Errorf("failed to get transfer order: %w", err)
	}

	// Inicia transferência
	if err := transfer.StartTransfer(); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	if err := uc.repo.UpdateTransfer(ctx, transfer); err != nil {
		return fmt.Errorf("failed to update transfer status: %w", err)
	}

	// Chama Core para saída de origem
	for _, item := range transfer.Items {
		// Saída (quantidade negativa)
		if err := uc.inventoryClient.AdjustStock(ctx, transfer.LocationFrom, item.SKU, -item.Quantity, item.Batch); err != nil {
			uc.logger.Error("Failed to adjust stock (outbound) in core inventory", "error", err, "sku", item.SKU)
			transfer.Status = fulfillment.StatusFailed
			uc.repo.UpdateTransfer(ctx, transfer)
			return fmt.Errorf("failed to adjust stock (outbound) for SKU %s: %w", item.SKU, err)
		}

		// Entrada no destino
		if err := uc.inventoryClient.AdjustStock(ctx, transfer.LocationTo, item.SKU, item.Quantity, item.Batch); err != nil {
			uc.logger.Error("Failed to adjust stock (inbound) in core inventory", "error", err, "sku", item.SKU)
			transfer.Status = fulfillment.StatusFailed
			uc.repo.UpdateTransfer(ctx, transfer)
			return fmt.Errorf("failed to adjust stock (inbound) for SKU %s: %w", item.SKU, err)
		}
	}

	// Completa a transferência
	if err := transfer.Complete(); err != nil {
		return fmt.Errorf("failed to complete transfer: %w", err)
	}

	if err := uc.repo.UpdateTransfer(ctx, transfer); err != nil {
		return fmt.Errorf("failed to update transfer status: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishTransferCompleted(ctx, transfer); err != nil {
		uc.logger.Error("Failed to publish transfer completed event", "error", err)
	}

	uc.logger.Info("Transfer order completed", "id", transferID)
	return nil
}

