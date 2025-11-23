package interceptors

import (
	"context"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// RateLimiter defines interface for rate limiting
type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) bool
}

// RateLimitInterceptor creates rate limiting interceptor for gRPC
func RateLimitInterceptor(rateLimiter RateLimiter, limit int, window time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Get client identifier
		clientID := "unknown"

		// Try to get user ID from context first
		if userID, ok := ctx.Value("user_id").(string); ok {
			clientID = userID
		} else {
			// Fallback to IP from metadata
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				ips := md.Get("x-forwarded-for")
				if len(ips) > 0 {
					clientID = ips[0]
				} else {
					ips = md.Get("x-real-ip")
					if len(ips) > 0 {
						clientID = ips[0]
					}
				}
			}
		}

		// Check rate limit
		if !rateLimiter.Allow(clientID, limit, window) {
			logger.Warn("Rate limit exceeded",
				zap.String("client_id", clientID),
				zap.String("method", info.FullMethod),
			)
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
