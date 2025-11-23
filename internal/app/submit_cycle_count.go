package app

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// SubmitCycleCountUseCase orquestra o envio e processamento de contagens cíclicas
type SubmitCycleCountUseCase struct {
	repo            fulfillment.Repository
	inventoryClient InventoryClient
	eventPublisher  EventPublisher
	logger          Logger
}

// NewSubmitCycleCountUseCase cria uma nova instância do caso de uso
func NewSubmitCycleCountUseCase(repo fulfillment.Repository, inventoryClient InventoryClient, eventPublisher EventPublisher, logger Logger) *SubmitCycleCountUseCase {
	return &SubmitCycleCountUseCase{
		repo:            repo,
		inventoryClient: inventoryClient,
		eventPublisher:  eventPublisher,
		logger:          logger,
	}
}

// SubmitCycleCount processa a contagem física e gera ajustes
func (uc *SubmitCycleCountUseCase) SubmitCycleCount(ctx context.Context, taskID string, countedItems []fulfillment.Item) error {
	task, err := uc.repo.GetCycleCountByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get cycle count task: %w", err)
	}

	// Inicia contagem
	if err := task.StartCounting(); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	// Registra itens contados
	if err := task.SubmitCount(countedItems); err != nil {
		return fmt.Errorf("failed to submit count: %w", err)
	}

	if err := uc.repo.UpdateCycleCount(ctx, task); err != nil {
		return fmt.Errorf("failed to update cycle count task: %w", err)
	}

	// Compara contagem física vs ledger (via Core) e calcula diferenças
	for _, countedItem := range countedItems {
		ledgerQuantity, err := uc.inventoryClient.GetAvailableStock(ctx, task.Location, countedItem.SKU)
		if err != nil {
			uc.logger.Error("Failed to get available stock from core inventory", "error", err, "sku", countedItem.SKU)
			continue
		}

		physicalQuantity := countedItem.Quantity
		difference := physicalQuantity - ledgerQuantity

		if difference != 0 {
			uc.logger.Warn("Stock discrepancy found", "sku", countedItem.SKU, "ledger", ledgerQuantity, "physical", physicalQuantity, "difference", difference)

			// Gera ajuste via mcp-core-inventory
			if err := uc.inventoryClient.AdjustStock(ctx, task.Location, countedItem.SKU, difference, countedItem.Batch); err != nil {
				uc.logger.Error("Failed to adjust stock in core inventory", "error", err, "sku", countedItem.SKU)
				task.Status = fulfillment.StatusFailed
				uc.repo.UpdateCycleCount(ctx, task)
				return fmt.Errorf("failed to adjust stock for SKU %s: %w", countedItem.SKU, err)
			}
		}
	}

	// Completa a contagem
	if err := task.Complete(); err != nil {
		return fmt.Errorf("failed to complete cycle count: %w", err)
	}

	if err := uc.repo.UpdateCycleCount(ctx, task); err != nil {
		return fmt.Errorf("failed to update cycle count status: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishCycleCountCompleted(ctx, task); err != nil {
		uc.logger.Error("Failed to publish cycle count completed event", "error", err)
	}

	uc.logger.Info("Cycle count task completed", "id", taskID)
	return nil
}
