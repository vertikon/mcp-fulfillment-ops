package app

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// RegisterReturnUseCase orquestra o registro e processamento de devoluções
type RegisterReturnUseCase struct {
	repo            fulfillment.Repository
	inventoryClient InventoryClient
	eventPublisher  EventPublisher
	logger          Logger
}

// NewRegisterReturnUseCase cria uma nova instância do caso de uso
func NewRegisterReturnUseCase(repo fulfillment.Repository, inventoryClient InventoryClient, eventPublisher EventPublisher, logger Logger) *RegisterReturnUseCase {
	return &RegisterReturnUseCase{
		repo:            repo,
		inventoryClient: inventoryClient,
		eventPublisher:  eventPublisher,
		logger:          logger,
	}
}

// RegisterReturn registra uma devolução física
func (uc *RegisterReturnUseCase) RegisterReturn(ctx context.Context, originalOrderID, reason, location string, items []fulfillment.Item) (*fulfillment.ReturnOrder, error) {
	returnOrder, err := fulfillment.NewReturnOrder(originalOrderID, reason, items)
	if err != nil {
		return nil, fmt.Errorf("failed to create return order: %w", err)
	}

	if err := uc.repo.CreateReturn(ctx, returnOrder); err != nil {
		return nil, fmt.Errorf("failed to persist return order: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishReturnRegistered(ctx, returnOrder); err != nil {
		uc.logger.Error("Failed to publish return registered event", "error", err)
	}

	uc.logger.Info("Return order registered", "id", returnOrder.ID, "original_order_id", originalOrderID)
	return returnOrder, nil
}

// CompleteReturn finaliza o processamento da devolução
func (uc *RegisterReturnUseCase) CompleteReturn(ctx context.Context, returnID, location string) error {
	returnOrder, err := uc.repo.GetReturnByID(ctx, returnID)
	if err != nil {
		return fmt.Errorf("failed to get return order: %w", err)
	}

	// Inicia processamento
	if err := returnOrder.StartProcessing(); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	if err := uc.repo.UpdateReturn(ctx, returnOrder); err != nil {
		return fmt.Errorf("failed to update return status: %w", err)
	}

	// Aplica lógica de reaproveitamento e chama Core para entrada/ajuste
	// Por padrão, devoluções voltam para estoque vendável
	for _, item := range returnOrder.Items {
		if err := uc.inventoryClient.AdjustStock(ctx, location, item.SKU, item.Quantity, item.Batch); err != nil {
			uc.logger.Error("Failed to adjust stock in core inventory", "error", err, "sku", item.SKU)
			returnOrder.Status = fulfillment.StatusFailed
			uc.repo.UpdateReturn(ctx, returnOrder)
			return fmt.Errorf("failed to adjust stock for SKU %s: %w", item.SKU, err)
		}
	}

	// Completa a devolução
	if err := returnOrder.Complete(); err != nil {
		return fmt.Errorf("failed to complete return: %w", err)
	}

	if err := uc.repo.UpdateReturn(ctx, returnOrder); err != nil {
		return fmt.Errorf("failed to update return status: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishReturnCompleted(ctx, returnOrder); err != nil {
		uc.logger.Error("Failed to publish return completed event", "error", err)
	}

	uc.logger.Info("Return order completed", "id", returnID)
	return nil
}

