package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ServiceRegistry manages registration and discovery of services
type ServiceRegistry struct {
	services map[string]*ServiceInfo
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*ServiceInfo),
		logger:   logger.GetLogger(),
	}
}

// RegisterService registers a new service
func (sr *ServiceRegistry) RegisterService(ctx context.Context, service *ServiceInfo) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	service.LastChecked = time.Now()
	sr.services[service.ID] = service
	
	sr.logger.Info("Service registered",
		zap.String("id", service.ID),
		zap.String("name", service.Name),
		zap.String("type", service.Type))
	
	return nil
}

// UnregisterService removes a service from the registry
func (sr *ServiceRegistry) UnregisterService(ctx context.Context, serviceID string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if _, exists := sr.services[serviceID]; !exists {
		return fmt.Errorf("service %s not found", serviceID)
	}

	delete(sr.services, serviceID)
	sr.logger.Info("Service unregistered", zap.String("id", serviceID))
	
	return nil
}

// GetService retrieves a service by ID
func (sr *ServiceRegistry) GetService(ctx context.Context, serviceID string) (*ServiceInfo, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	service, exists := sr.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service %s not found", serviceID)
	}

	return service, nil
}

// ListServices returns all registered services
func (sr *ServiceRegistry) ListServices(ctx context.Context) ([]*ServiceInfo, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(sr.services))
	for _, service := range sr.services {
		services = append(services, service)
	}

	return services, nil
}

// UpdateServiceStatus updates the status of a service
func (sr *ServiceRegistry) UpdateServiceStatus(ctx context.Context, serviceID, status string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	service, exists := sr.services[serviceID]
	if !exists {
		return fmt.Errorf("service %s not found", serviceID)
	}

	service.Status = status
	service.LastChecked = time.Now()
	
	sr.logger.Info("Service status updated",
		zap.String("id", serviceID),
		zap.String("status", status))
	
	return nil
}