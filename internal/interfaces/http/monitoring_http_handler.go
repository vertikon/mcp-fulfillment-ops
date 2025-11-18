package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-hulk/internal/services"
	"github.com/vertikon/mcp-hulk/pkg/logger"
	"go.uber.org/zap"
)

// MonitoringHandler handles HTTP requests for monitoring operations
type MonitoringHandler struct {
	monitoringService *services.MonitoringAppService
	logger            *zap.Logger
}

// NewMonitoringHandler creates a new monitoring HTTP handler
func NewMonitoringHandler(monitoringService *services.MonitoringAppService) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
		logger:            logger.WithContext(nil),
	}
}

// RegisterRoutes registers monitoring routes
func (h *MonitoringHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/metrics", h.GetMetrics)
	e.GET("/health", h.GetHealth)
	e.GET("/status", h.GetStatus)
}

// GetMetrics handles GET /metrics
func (h *MonitoringHandler) GetMetrics(c echo.Context) error {
	// Call service
	metrics, err := h.monitoringService.GetMetrics(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to get metrics", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get metrics"})
	}
	return c.JSON(http.StatusOK, metrics)
}

// GetHealth handles GET /health
func (h *MonitoringHandler) GetHealth(c echo.Context) error {
	// Call service
	health, err := h.monitoringService.GetHealth(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to get health", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get health"})
	}
	return c.JSON(http.StatusOK, health)
}

// GetStatus handles GET /status
func (h *MonitoringHandler) GetStatus(c echo.Context) error {
	// Call service
	status, err := h.monitoringService.GetStatus(c.Request().Context())
	if err != nil {
		h.logger.Error("Failed to get status", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get status"})
	}
	return c.JSON(http.StatusOK, status)
}
