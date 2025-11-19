package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// LoggingMiddleware creates logging middleware
func LoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Log request
			logger.Info("HTTP request",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Path()),
				zap.String("remote_addr", c.RealIP()),
				zap.Int("status", c.Response().Status),
				zap.Duration("duration", duration),
				zap.String("user_agent", c.Request().UserAgent()),
			)

			return err
		}
	}
}
