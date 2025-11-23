package interceptors

import (
	"context"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor creates logging interceptor for gRPC
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Process request
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Get status code
		code := status.Code(err)
		if err == nil {
			code = status.Code(nil)
		}

		// Log request
		logger.Info("gRPC request",
			zap.String("method", info.FullMethod),
			zap.String("code", code.String()),
			zap.Duration("duration", duration),
			zap.Error(err),
		)

		return resp, err
	}
}
