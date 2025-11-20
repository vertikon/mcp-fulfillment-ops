package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

type StartInboundRequest struct {
	ReferenceID string            `json:"reference_id" binding:"required"`
	Origin      string            `json:"origin" binding:"required"`
	Destination string            `json:"destination" binding:"required"`
	Items       []fulfillment.Item `json:"items" binding:"required"`
}

type ConfirmInboundRequest struct {
	ShipmentID string `json:"shipment_id" binding:"required"`
}

func handleStartInbound(uc *app.ReceiveGoodsUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req StartInboundRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		shipment, err := uc.StartInbound(c.Request.Context(), req.ReferenceID, req.Origin, req.Destination, req.Items)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, shipment)
	}
}

func handleConfirmInbound(uc *app.ReceiveGoodsUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ConfirmInboundRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uc.ConfirmReceipt(c.Request.Context(), req.ShipmentID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
	}
}

