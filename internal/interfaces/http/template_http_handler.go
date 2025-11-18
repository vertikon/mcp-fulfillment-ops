package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-hulk/internal/application/dtos"
	"github.com/vertikon/mcp-hulk/internal/services"
	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// TemplateHandler handles HTTP requests for template operations
type TemplateHandler struct {
	templateService *services.TemplateAppService
	logger          *zap.Logger
}

// NewTemplateHandler creates a new template HTTP handler
func NewTemplateHandler(templateService *services.TemplateAppService) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
		logger:          logger.WithContext(nil),
	}
}

// RegisterRoutes registers template routes
func (h *TemplateHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/templates", h.CreateTemplate)
	e.GET("/templates", h.ListTemplates)
	e.GET("/templates/:id", h.GetTemplate)
	e.PUT("/templates/:id", h.UpdateTemplate)
	e.DELETE("/templates/:id", h.DeleteTemplate)
}

// CreateTemplate handles POST /templates
func (h *TemplateHandler) CreateTemplate(c echo.Context) error {
	var req dtos.CreateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	// Call service
	template, err := h.templateService.CreateTemplate(c.Request().Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create template", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create template"})
	}
	return c.JSON(http.StatusCreated, template)
}

// ListTemplates handles GET /templates
func (h *TemplateHandler) ListTemplates(c echo.Context) error {
	// Call service
	templates, err := h.templateService.ListTemplates(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to list templates", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list templates"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"templates": templates, "total": len(templates)})
}

// GetTemplate handles GET /templates/:id
func (h *TemplateHandler) GetTemplate(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Template ID is required"})
	}
	// Call service
	template, err := h.templateService.GetTemplate(c.Request().Context(), id)
	if err != nil {
		h.logger.Error("Failed to get template", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}
	return c.JSON(http.StatusOK, template)
}

// UpdateTemplate handles PUT /templates/:id
func (h *TemplateHandler) UpdateTemplate(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Template ID is required"})
	}
	var req dtos.UpdateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	// Call service
	template, err := h.templateService.UpdateTemplate(c.Request().Context(), id, &req)
	if err != nil {
		h.logger.Error("Failed to update template", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update template"})
	}
	return c.JSON(http.StatusOK, template)
}

// DeleteTemplate handles DELETE /templates/:id
func (h *TemplateHandler) DeleteTemplate(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Template ID is required"})
	}
	// Call service
	err := h.templateService.DeleteTemplate(c.Request().Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete template", zap.String("id", id), zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete template"})
	}
	return c.NoContent(http.StatusNoContent)
}
