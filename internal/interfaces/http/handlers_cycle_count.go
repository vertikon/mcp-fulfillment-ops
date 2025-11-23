package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vertikon/mcp-fulfillment-ops/internal/app"
	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

type OpenCycleCountRequest struct {
	Location string   `json:"location" binding:"required"`
	SKUs     []string `json:"skus" binding:"required"`
}

type SubmitCycleCountRequest struct {
	TaskID       string             `json:"task_id" binding:"required"`
	CountedItems []fulfillment.Item `json:"counted_items" binding:"required"`
}

func handleOpenCycleCount(uc *app.OpenCycleCountUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req OpenCycleCountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		task, err := uc.OpenCycleCount(c.Request.Context(), req.Location, req.SKUs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, task)
	}
}

func handleSubmitCycleCount(uc *app.SubmitCycleCountUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SubmitCycleCountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := uc.SubmitCycleCount(c.Request.Context(), req.TaskID, req.CountedItems); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "submitted"})
	}
}
