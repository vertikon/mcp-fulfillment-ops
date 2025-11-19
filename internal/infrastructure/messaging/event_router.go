// Package messaging provides event routing functionality
package messaging

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// EventRouter provides semantic event routing
type EventRouter interface {
	// RegisterHandler registers a handler for a pattern
	RegisterHandler(pattern string, handler EventHandler) error

	// Route routes an event to appropriate handlers
	Route(ctx context.Context, event *Event) error

	// UnregisterHandler removes a handler
	UnregisterHandler(pattern string) error
}

// EventHandler handles events matching a pattern
type EventHandler func(ctx context.Context, event *Event) error

// Event represents an event to be routed
type Event struct {
	Subject   string
	Type      string
	Payload   interface{}
	Metadata  map[string]interface{}
	Timestamp int64
}

// eventRouter implements EventRouter
type eventRouter struct {
	handlers map[string]EventHandler
	mu       sync.RWMutex
}

// NewEventRouter creates a new event router
func NewEventRouter() EventRouter {
	return &eventRouter{
		handlers: make(map[string]EventHandler),
	}
}

// RegisterHandler registers a handler for a pattern
func (r *eventRouter) RegisterHandler(pattern string, handler EventHandler) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[pattern]; exists {
		logger.Warn("Handler already registered for pattern, overwriting",
			zap.String("pattern", pattern),
		)
	}

	r.handlers[pattern] = handler
	logger.Info("Registered event handler",
		zap.String("pattern", pattern),
	)

	return nil
}

// Route routes an event to appropriate handlers
func (r *eventRouter) Route(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := false
	for pattern, handler := range r.handlers {
		if r.matchPattern(pattern, event.Subject) {
			matched = true
			if err := handler(ctx, event); err != nil {
				logger.Error("Event handler failed",
					zap.String("pattern", pattern),
					zap.String("subject", event.Subject),
					zap.Error(err),
				)
				// Continue to other handlers even if one fails
			}
		}
	}

	if !matched {
		logger.Debug("No handler matched event",
			zap.String("subject", event.Subject),
		)
	}

	return nil
}

// UnregisterHandler removes a handler
func (r *eventRouter) UnregisterHandler(pattern string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[pattern]; !exists {
		return fmt.Errorf("handler not found for pattern: %s", pattern)
	}

	delete(r.handlers, pattern)
	logger.Info("Unregistered event handler",
		zap.String("pattern", pattern),
	)

	return nil
}

// matchPattern matches a subject against a pattern
// Supports wildcards: * (single token), > (all remaining tokens)
func (r *eventRouter) matchPattern(pattern, subject string) bool {
	// Exact match
	if pattern == subject {
		return true
	}

	// Wildcard patterns
	patternParts := strings.Split(pattern, ".")
	subjectParts := strings.Split(subject, ".")

	// Handle > wildcard (matches all remaining tokens)
	if len(patternParts) > 0 && patternParts[len(patternParts)-1] == ">" {
		prefix := strings.Join(patternParts[:len(patternParts)-1], ".")
		return strings.HasPrefix(subject, prefix+".")
	}

	// Handle * wildcard (matches single token)
	if len(patternParts) != len(subjectParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if patternPart == "*" {
			continue // Matches any single token
		}
		if patternPart != subjectParts[i] {
			return false
		}
	}

	return true
}
