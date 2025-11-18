package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-hulk/internal/application/dtos"
	"github.com/vertikon/mcp-hulk/internal/services"
	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// MCPHandler handles HTTP requests for MCP operations
type MCPHandler struct {
	mcpService *services.MCPAppService
	logger     *zap.Logger
}

// NewMCPHandler creates a new MCP HTTP handler
func NewMCPHandler(mcpService *services.MCPAppService) *MCPHandler {
	return &MCPHandler{
		mcpService: mcpService,
		logger:     logger.WithContext(nil),
	}
}

// RegisterRoutes registers MCP routes
func (h *MCPHandler) RegisterRoutes(e *echo.Group) {
	// MCP CRUD operations
	e.POST("/mcps", h.CreateMCP)
	e.GET("/mcps", h.ListMCPs)
	e.GET("/mcps/:id", h.GetMCP)
	e.PUT("/mcps/:id", h.UpdateMCP)
	e.DELETE("/mcps/:id", h.DeleteMCP)

	// MCP generation
	e.POST("/mcps/generate", h.GenerateMCP)
	e.POST("/mcps/:id/validate", h.ValidateMCP)
}

// CreateMCP handles POST /mcps
func (h *MCPHandler) CreateMCP(c echo.Context) error {
	var req dtos.CreateMCPRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Call service
	mcp, err := h.mcpService.CreateMCP(c.Request().Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create MCP", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create MCP",
		})
	}

	return c.JSON(http.StatusCreated, mcp)
}

// ListMCPs handles GET /mcps
func (h *MCPHandler) ListMCPs(c echo.Context) error {
	// Get query parameters
	limit := c.QueryParam("limit")
	offset := c.QueryParam("offset")

	h.logger.Info("Listing MCPs",
		zap.String("limit", limit),
		zap.String("offset", offset),
	)

	// Call service
	mcps, err := h.mcpService.ListMCPs(c.Request().Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to list MCPs", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list MCPs",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"mcps":   mcps,
		"total":  len(mcps),
		"limit":  limit,
		"offset": offset,
	})
}

// GetMCP handles GET /mcps/:id
func (h *MCPHandler) GetMCP(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "MCP ID is required",
		})
	}

	// Call service
	mcp, err := h.mcpService.GetMCP(c.Request().Context(), id)
	if err != nil {
		h.logger.Error("Failed to get MCP", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "MCP not found",
		})
	}

	return c.JSON(http.StatusOK, mcp)
}

// UpdateMCP handles PUT /mcps/:id
func (h *MCPHandler) UpdateMCP(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "MCP ID is required",
		})
	}

	var req dtos.UpdateMCPRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Call service
	mcp, err := h.mcpService.UpdateMCP(c.Request().Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update MCP", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update MCP",
		})
	}

	return c.JSON(http.StatusOK, mcp)
}

// DeleteMCP handles DELETE /mcps/:id
func (h *MCPHandler) DeleteMCP(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "MCP ID is required",
		})
	}

	// Call service
	err := h.mcpService.DeleteMCP(c.Request().Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete MCP", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete MCP",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// GenerateMCP handles POST /mcps/generate
func (h *MCPHandler) GenerateMCP(c echo.Context) error {
	var req dtos.GenerateMCPRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Call service
	result, err := h.mcpService.GenerateMCP(c.Request().Context(), &req)
	if err != nil {
		h.logger.Error("Failed to generate MCP", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate MCP",
		})
	}

	return c.JSON(http.StatusAccepted, result)
}

// ValidateMCP handles POST /mcps/:id/validate
func (h *MCPHandler) ValidateMCP(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "MCP ID is required",
		})
	}

	// Call service
	result, err := h.mcpService.ValidateMCP(c.Request().Context(), id)
	if err != nil {
		h.logger.Error("Failed to validate MCP", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to validate MCP",
		})
	}

	return c.JSON(http.StatusOK, result)
}
