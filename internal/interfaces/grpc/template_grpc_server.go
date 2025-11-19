package grpc

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// TemplateServer implements gRPC server for template operations
type TemplateServer struct {
	templateService *services.TemplateAppService
	logger          *zap.Logger
}

// NewTemplateServer creates a new template gRPC server
func NewTemplateServer(templateService *services.TemplateAppService) *TemplateServer {
	return &TemplateServer{
		templateService: templateService,
		logger:          logger.WithContext(nil),
	}
}

// RegisterService registers the template service with gRPC server
func (s *TemplateServer) RegisterService(grpcServer *grpc.Server) {
	// TODO: Register protobuf service
	s.logger.Info("Template gRPC service registered")
}

// CreateTemplate handles CreateTemplate gRPC request
func (s *TemplateServer) CreateTemplate(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.Info("CreateTemplate gRPC call")
	// TODO: Implement
	return nil, nil
}

// ListTemplates handles ListTemplates gRPC request
func (s *TemplateServer) ListTemplates(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.Info("ListTemplates gRPC call")
	// TODO: Implement
	return nil, nil
}
