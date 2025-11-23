package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// ServiceDiscovery handles service discovery functionality
type ServiceDiscovery struct {
	registry *MCPRegistry
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(registry *MCPRegistry) *ServiceDiscovery {
	return &ServiceDiscovery{
		registry: registry,
		logger:   logger.Get(),
	}
}

// DiscoverServices discovers services in the registry
func (sd *ServiceDiscovery) DiscoverServices(ctx context.Context, query string) ([]*ServiceInfo, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	var results []*ServiceInfo

	for _, service := range sd.registry.services {
		if matchesQuery(service, query) {
			results = append(results, service)
		}
	}

	return results, nil
}

// matchesQuery checks if a service matches the search query
func matchesQuery(service *ServiceInfo, query string) bool {
	if query == "" {
		return true
	}

	// Simple string matching - could be enhanced with regex or fuzzy matching
	return fmt.Sprintf("%s %s %s",
		service.Name,
		service.Type,
		service.Version) == query
}

// WatchServices watches for changes in services
func (sd *ServiceDiscovery) WatchServices(ctx context.Context) (<-chan *ServiceEvent, error) {
	eventChan := make(chan *ServiceEvent, 100)

	go func() {
		defer close(eventChan)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Poll for changes
				sd.pollForChanges(eventChan)
			}
		}
	}()

	return eventChan, nil
}

// pollForChanges polls for changes in the registry
func (sd *ServiceDiscovery) pollForChanges(eventChan chan<- *ServiceEvent) {
	// Implementation would check for changes and send events
	// This is a placeholder for the actual polling logic
}

// ServiceEvent represents a service discovery event
type ServiceEvent struct {
	Type      string       `json:"type"` // "added", "removed", "updated"
	Service   *ServiceInfo `json:"service"`
	Timestamp time.Time    `json:"timestamp"`
}
