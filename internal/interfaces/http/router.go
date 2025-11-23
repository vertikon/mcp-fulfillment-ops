package http

import (
	"github.com/gin-gonic/gin"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
)

// Router configura as rotas HTTP do fulfillment-ops
func Router(
	receiveGoodsUC *app.ReceiveGoodsUseCase,
	shipOrderUC *app.ShipOrderUseCase,
	registerReturnUC *app.RegisterReturnUseCase,
	completeTransferUC *app.CompleteTransferUseCase,
	openCycleCountUC *app.OpenCycleCountUseCase,
	submitCycleCountUC *app.SubmitCycleCountUseCase,
) *gin.Engine {
	r := gin.Default()

	// Middleware de observabilidade
	r.Use(observabilityMiddleware())

	// Grupo de rotas v1
	v1 := r.Group("/v1")

	// Inbound (Entrada)
	inbound := v1.Group("/inbound")
	{
		inbound.POST("/start", handleStartInbound(receiveGoodsUC))
		inbound.POST("/confirm", handleConfirmInbound(receiveGoodsUC))
	}

	// Outbound (Saída)
	outbound := v1.Group("/outbound")
	{
		outbound.POST("/start_picking", handleStartPicking(shipOrderUC))
		outbound.POST("/ship", handleShipOrder(shipOrderUC))
	}

	// Transferências
	transfer := v1.Group("/transfer")
	{
		transfer.POST("/create", handleCreateTransfer(completeTransferUC))
		transfer.POST("/complete", handleCompleteTransfer(completeTransferUC))
	}

	// Devoluções
	returns := v1.Group("/returns")
	{
		returns.POST("/register", handleRegisterReturn(registerReturnUC))
		returns.POST("/complete", handleCompleteReturn(registerReturnUC))
	}

	// Contagem Cíclica
	cycleCount := v1.Group("/cycle_count")
	{
		cycleCount.POST("/open", handleOpenCycleCount(openCycleCountUC))
		cycleCount.POST("/submit", handleSubmitCycleCount(submitCycleCountUC))
	}

	// Health check
	r.GET("/health", handleHealth())

	return r
}

func observabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implementar middleware de observabilidade (logs, métricas, trace)
		c.Next()
	}
}

func handleHealth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	}
}
