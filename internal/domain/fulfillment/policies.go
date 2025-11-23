package fulfillment

import "time"

// Policy define políticas de workflow físico
type Policy struct {
	// SLA máximo para operações (em minutos)
	MaxInboundDurationMinutes    int
	MaxOutboundDurationMinutes   int
	MaxTransferDurationMinutes   int
	MaxReturnDurationMinutes     int
	MaxCycleCountDurationMinutes int
}

// DefaultPolicy retorna a política padrão
func DefaultPolicy() *Policy {
	return &Policy{
		MaxInboundDurationMinutes:    120, // 2 horas para recebimento
		MaxOutboundDurationMinutes:   60,  // 1 hora para expedição
		MaxTransferDurationMinutes:   180, // 3 horas para transferência
		MaxReturnDurationMinutes:     90,  // 1.5 horas para devolução
		MaxCycleCountDurationMinutes: 240, // 4 horas para contagem
	}
}

// ValidateStateTransition valida se uma transição de estado é válida
func ValidateStateTransition(from, to Status) bool {
	validTransitions := map[Status][]Status{
		StatusPending:    {StatusInProgress, StatusCancelled},
		StatusInProgress: {StatusCompleted, StatusFailed, StatusCancelled, StatusBlocked},
		StatusBlocked:    {StatusInProgress, StatusCancelled},
		StatusFailed:     {StatusPending, StatusCancelled},
		StatusCancelled:  {}, // Estado final
		StatusCompleted:  {}, // Estado final
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == to {
			return true
		}
	}

	return false
}

// CheckSLA verifica se uma operação está dentro do SLA
func CheckSLA(createdAt time.Time, maxDurationMinutes int) bool {
	elapsed := time.Since(createdAt)
	maxDuration := time.Duration(maxDurationMinutes) * time.Minute
	return elapsed <= maxDuration
}

// IsIdempotencyKeyValid valida se uma chave de idempotência é válida
func IsIdempotencyKeyValid(key string) bool {
	return len(key) > 0 && len(key) <= 255
}
