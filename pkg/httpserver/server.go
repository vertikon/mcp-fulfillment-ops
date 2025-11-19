// Package httpserver provides HTTP server with Echo and Vertikon middlewares
package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vertikon/mcp-fulfillment-ops/internal/observability"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Server represents an HTTP server
type Server struct {
	e      *echo.Echo
	config Config
	metrics *observability.Metrics
}

// Config represents server configuration
type Config struct {
	Port         int
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NewServer creates a new HTTP server
func NewServer(config Config, metrics *observability.Metrics) *Server {
	e := echo.New()

	// Hide Echo banner
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(otelMiddleware())
	e.Use(loggingMiddleware())
	e.Use(metricsMiddleware(metrics))

	// CORS
	e.Use(middleware.CORS())

	// Health endpoints
	e.GET("/health", healthHandler)
	e.GET("/ready", readyHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	return &Server{
		e:       e,
		config:  config,
		metrics: metrics,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	logger.Info("Starting HTTP server",
		zap.String("address", addr),
	)

	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	return s.e.StartServer(srv)
}

// Stop stops the HTTP server gracefully
func (s *Server) Stop(ctx context.Context) error {
	logger.Info("Stopping HTTP server")
	return s.e.Shutdown(ctx)
}

// RegisterRoute registers a route
func (s *Server) RegisterRoute(method, path string, handler echo.HandlerFunc, middlewares ...echo.MiddlewareFunc) {
	s.e.Add(method, path, handler, middlewares...)
}

// GetEcho returns the underlying Echo instance
func (s *Server) GetEcho() *echo.Echo {
	return s.e
}

// healthHandler handles health check requests
func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now(),
	})
}

// readyHandler handles readiness check requests
func readyHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ready",
		"timestamp": time.Now(),
	})
}

// otelMiddleware adds OpenTelemetry tracing
func otelMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Tracing is handled by OpenTelemetry instrumentation
			return next(c)
		}
	}
}

// loggingMiddleware adds structured logging
func loggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			logger.Info("HTTP request",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
				zap.Int("status", c.Response().Status),
				zap.Duration("duration", duration),
			)

			return err
		}
	}
}

// metricsMiddleware adds Prometheus metrics
func metricsMiddleware(m *observability.Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			status := fmt.Sprintf("%d", c.Response().Status)
			m.RecordRequest(c.Request().Method, c.Path(), status, duration.Seconds())

			return err
		}
	}
}

