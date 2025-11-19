package grpc

import (
	"context"

	"github.com/vertikon/mcp-fulfillment-ops/internal/services"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// MonitoringServer implements gRPC server for monitoring operations
type MonitoringServer struct {
	monitoringService *services.MonitoringAppService
	logger            *zap.Logger
}

// NewMonitoringServer creates a new monitoring gRPC server
func NewMonitoringServer(monitoringService *services.MonitoringAppService) *MonitoringServer {
	return &MonitoringServer{
		monitoringService: monitoringService,
		logger:            logger.WithContext(nil),
	}
}

// RegisterService registers the monitoring service with gRPC server
func (s *MonitoringServer) RegisterService(grpcServer *grpc.Server) {
	// TODO: Register protobuf service
	s.logger.Info("Monitoring gRPC service registered")
}

// GetMetrics handles GetMetrics gRPC request
func (s *MonitoringServer) GetMetrics(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.Info("GetMetrics gRPC call")
	// TODO: Implement
	return nil, nil
}

// GetHealth handles GetHealth gRPC request
func (s *MonitoringServer) GetHealth(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.Info("GetHealth gRPC call")
	// TODO: Implement
	return nil, nil
}
