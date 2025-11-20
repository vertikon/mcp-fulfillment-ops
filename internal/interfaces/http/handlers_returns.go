package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

type RegisterReturnRequest struct {
	OriginalOrderID string            `json:"original_order_id" binding:"required"`
	Reason          string            `json:"reason"`
	Location        string            `json:"location" binding:"required"`
	Items           []fulfillment.Item `json:"items" binding:"required"`
}

type CompleteReturnRequest struct {
	ReturnID  string `json:"return_id" binding:"required"`
	Location  string `json:"location" binding:"required"`
}

func handleRegisterReturn(uc *app.RegisterReturnUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterReturnRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		returnOrder, err := uc.RegisterReturn(c.Request.Context(), req.OriginalOrderID, req.Reason, req.Location, req.Items)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, returnOrder)
	}
}

func handleCompleteReturn(uc *app.RegisterReturnUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CompleteReturnRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uc.CompleteReturn(c.Request.Context(), req.ReturnID, req.Location); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	}
}

