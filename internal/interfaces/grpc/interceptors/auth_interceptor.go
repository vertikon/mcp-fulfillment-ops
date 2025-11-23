package interceptors

import (
	"context"
	"strings"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthManager defines interface for authentication
type AuthManager interface {
	ValidateToken(token string) (string, error) // Returns user ID
	HasPermission(userID string, resource string, action string) bool
}

// AuthInterceptor creates authentication interceptor for gRPC
func AuthInterceptor(authManager AuthManager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata not provided")
		}

		// Extract authorization token
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization header required")
		}

		authHeader := authHeaders[0]
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		token := parts[1]

		// Validate token
		userID, err := authManager.ValidateToken(token)
		if err != nil {
			logger.Warn("Token validation failed", zap.Error(err))
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		// Add user ID to context
		ctx = context.WithValue(ctx, "user_id", userID)

		return handler(ctx, req)
	}
}

// RBACInterceptor creates RBAC interceptor for gRPC
func RBACInterceptor(authManager AuthManager, resource string, action string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Get user ID from context
		userID, ok := ctx.Value("user_id").(string)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "user not authenticated")
		}

		// Check permission
		if !authManager.HasPermission(userID, resource, action) {
			return nil, status.Error(codes.PermissionDenied, "insufficient permissions")
		}

		return handler(ctx, req)
	}
}
