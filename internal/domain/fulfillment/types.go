package fulfillment

import "errors"

// OperationType representa o tipo de operação de fulfillment
type OperationType string

const (
	OpInbound    OperationType = "INBOUND"
	OpOutbound   OperationType = "OUTBOUND"
	OpTransfer   OperationType = "TRANSFER"
	OpReturn     OperationType = "RETURN"
	OpCycleCount OperationType = "CYCLE_COUNT"
)

// Status do Workflow (Máquina de Estados)
type Status string

const (
	StatusPending    Status = "PENDING"     // Criado, aguardando início
	StatusInProgress Status = "IN_PROGRESS" // Sendo bipado/separado
	StatusCompleted  Status = "COMPLETED"   // Finalizado com sucesso
	StatusCancelled  Status = "CANCELLED"   // Cancelado
	StatusFailed     Status = "FAILED"      // Erro sistêmico ou divergência
	StatusBlocked    Status = "BLOCKED"     // Bloqueado (ex: divergência física)
)

var (
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrOrderNotFound          = errors.New("fulfillment order not found")
	ErrEmptyItems             = errors.New("order must have items")
	ErrShipmentNotFound       = errors.New("inbound shipment not found")
	ErrTransferNotFound       = errors.New("transfer order not found")
	ErrReturnNotFound         = errors.New("return order not found")
	ErrCycleCountNotFound     = errors.New("cycle count task not found")
)
