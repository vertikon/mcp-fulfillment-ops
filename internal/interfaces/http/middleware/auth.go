package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// AuthManager defines interface for authentication
type AuthManager interface {
	ValidateToken(token string) (string, error) // Returns user ID
	HasPermission(userID string, resource string, action string) bool
}

// AuthMiddleware creates authentication middleware
func AuthMiddleware(authManager AuthManager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authorization header required",
				})
			}

			// Extract Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid authorization header format",
				})
			}

			token := parts[1]

			// Validate token
			userID, err := authManager.ValidateToken(token)
			if err != nil {
				logger.Warn("Token validation failed", zap.Error(err))
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired token",
				})
			}

			// Set user ID in context
			c.Set("user_id", userID)

			return next(c)
		}
	}
}

// RBACMiddleware creates RBAC middleware
func RBACMiddleware(authManager AuthManager, resource string, action string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID, ok := c.Get("user_id").(string)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "User not authenticated",
				})
			}

			// Check permission
			if !authManager.HasPermission(userID, resource, action) {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Insufficient permissions",
				})
			}

			return next(c)
		}
	}
}
