package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

type CreateTransferRequest struct {
	LocationFrom string             `json:"location_from" binding:"required"`
	LocationTo   string             `json:"location_to" binding:"required"`
	Items        []fulfillment.Item `json:"items" binding:"required"`
}

type CompleteTransferRequest struct {
	TransferID string `json:"transfer_id" binding:"required"`
}

func handleCreateTransfer(uc *app.CompleteTransferUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateTransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		transfer, err := uc.CreateTransfer(c.Request.Context(), req.LocationFrom, req.LocationTo, req.Items)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, transfer)
	}
}

func handleCompleteTransfer(uc *app.CompleteTransferUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CompleteTransferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uc.CompleteTransfer(c.Request.Context(), req.TransferID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	}
}
