package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
)

type StartPickingRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

type ShipOrderRequest struct {
	OrderID string `json:"order_id" binding:"required"`
}

func handleStartPicking(uc *app.ShipOrderUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req StartPickingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uc.StartPicking(c.Request.Context(), req.OrderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "picking_started"})
	}
}

func handleShipOrder(uc *app.ShipOrderUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ShipOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uc.Ship(c.Request.Context(), req.OrderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "shipped"})
	}
}
