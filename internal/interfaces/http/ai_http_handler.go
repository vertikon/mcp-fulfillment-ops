package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-fulfillment-ops/internal/application/dtos"
	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// AIHandler handles HTTP requests for AI operations
type AIHandler struct {
	aiService *services.AIAppService
	logger    *zap.Logger
}

// NewAIHandler creates a new AI HTTP handler
func NewAIHandler(aiService *services.AIAppService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
		logger:    logger.WithContext(nil),
	}
}

// RegisterRoutes registers AI routes
func (h *AIHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/ai/chat", h.Chat)
	e.POST("/ai/generate", h.Generate)
	e.POST("/ai/embed", h.Embed)
	e.GET("/ai/models", h.ListModels)
}

// Chat handles POST /ai/chat
func (h *AIHandler) Chat(c echo.Context) error {
	var req dtos.ChatRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	// Call service
	response, err := h.aiService.Chat(c.Request().Context(), &req)
	if err != nil {
		h.logger.Error("Failed to process chat", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process chat"})
	}
	return c.JSON(http.StatusOK, response)
}

// Generate handles POST /ai/generate
func (h *AIHandler) Generate(c echo.Context) error {
	var req dtos.GenerateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	// Call service
	response, err := h.aiService.Generate(c.Request().Context(), &req)
	if err != nil {
		h.logger.Error("Failed to generate content", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate content"})
	}
	return c.JSON(http.StatusOK, response)
}

// Embed handles POST /ai/embed
func (h *AIHandler) Embed(c echo.Context) error {
	var req dtos.EmbedRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	// Call service
	response, err := h.aiService.Embed(c.Request().Context(), &req)
	if err != nil {
		h.logger.Error("Failed to generate embedding", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate embedding"})
	}
	return c.JSON(http.StatusOK, response)
}

// ListModels handles GET /ai/models
func (h *AIHandler) ListModels(c echo.Context) error {
	// Call service
	models, err := h.aiService.ListModels(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to list models", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list models"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"models": models})
}
