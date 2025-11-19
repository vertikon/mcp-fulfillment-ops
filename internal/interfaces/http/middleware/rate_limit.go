package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// RateLimiter defines interface for rate limiting
type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) bool
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(rateLimiter RateLimiter, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get client identifier (IP or user ID)
			clientID := c.RealIP()
			if userID, ok := c.Get("user_id").(string); ok {
				clientID = userID
			}

			// Check rate limit
			if !rateLimiter.Allow(clientID, limit, window) {
				logger.Warn("Rate limit exceeded",
					zap.String("client_id", clientID),
					zap.String("path", c.Path()),
				)
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Rate limit exceeded",
				})
			}

			return next(c)
		}
	}
}
