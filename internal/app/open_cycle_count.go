package app

import (
	"context"
	"fmt"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

// OpenCycleCountUseCase orquestra a abertura de tarefas de contagem cíclica
type OpenCycleCountUseCase struct {
	repo           fulfillment.Repository
	eventPublisher EventPublisher
	logger         Logger
}

// NewOpenCycleCountUseCase cria uma nova instância do caso de uso
func NewOpenCycleCountUseCase(repo fulfillment.Repository, eventPublisher EventPublisher, logger Logger) *OpenCycleCountUseCase {
	return &OpenCycleCountUseCase{
		repo:           repo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// OpenCycleCount cria uma nova tarefa de contagem cíclica
func (uc *OpenCycleCountUseCase) OpenCycleCount(ctx context.Context, location string, skus []string) (*fulfillment.CycleCountTask, error) {
	task, err := fulfillment.NewCycleCountTask(location, skus)
	if err != nil {
		return nil, fmt.Errorf("failed to create cycle count task: %w", err)
	}

	if err := uc.repo.CreateCycleCount(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to persist cycle count task: %w", err)
	}

	// Publica evento
	if err := uc.eventPublisher.PublishCycleCountOpened(ctx, task); err != nil {
		uc.logger.Error("Failed to publish cycle count opened event", "error", err)
	}

	uc.logger.Info("Cycle count task opened", "id", task.ID, "location", location)
	return task, nil
}

