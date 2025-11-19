package grpc

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// AIServer implements gRPC server for AI operations
type AIServer struct {
	aiService *services.AIAppService
	logger    *zap.Logger
}

// NewAIServer creates a new AI gRPC server
func NewAIServer(aiService *services.AIAppService) *AIServer {
	return &AIServer{
		aiService: aiService,
		logger:    logger.WithContext(nil),
	}
}

// RegisterService registers the AI service with gRPC server
func (s *AIServer) RegisterService(grpcServer *grpc.Server) {
	// TODO: Register protobuf service
	s.logger.Info("AI gRPC service registered")
}

// Chat handles Chat gRPC request
func (s *AIServer) Chat(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.Info("Chat gRPC call")
	// TODO: Implement
	return nil, nil
}

// Generate handles Generate gRPC request
func (s *AIServer) Generate(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.Info("Generate gRPC call")
	// TODO: Implement
	return nil, nil
}
