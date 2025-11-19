// Package events provides NATS JetStream event bindings
package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// EventPublisher publishes events to NATS JetStream
type EventPublisher struct {
	js nats.JetStreamContext
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(js nats.JetStreamContext) *EventPublisher {
	return &EventPublisher{js: js}
}

// PublishTaskCreated publishes a task created event
func (ep *EventPublisher) PublishTaskCreated(ctx context.Context, taskID string, metadata map[string]interface{}) error {
	event := TaskEvent{
		ID:        taskID,
		Type:      "created",
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	return ep.publishEvent(ctx, "hulk.task.created", event)
}

// PublishTaskCompleted publishes a task completed event
func (ep *EventPublisher) PublishTaskCompleted(ctx context.Context, taskID string, result interface{}) error {
	event := TaskEvent{
		ID:        taskID,
		Type:      "completed",
		Timestamp: time.Now(),
		Result:    result,
	}

	return ep.publishEvent(ctx, "hulk.task.completed", event)
}

// PublishTaskFailed publishes a task failed event
func (ep *EventPublisher) PublishTaskFailed(ctx context.Context, taskID string, error string) error {
	event := TaskEvent{
		ID:        taskID,
		Type:      "failed",
		Timestamp: time.Now(),
		Error:     error,
	}

	return ep.publishEvent(ctx, "hulk.task.failed", event)
}

// PublishRuntimeHealth publishes a runtime health event
func (ep *EventPublisher) PublishRuntimeHealth(ctx context.Context, healthy bool, details map[string]interface{}) error {
	event := HealthEvent{
		Healthy:   healthy,
		Timestamp: time.Now(),
		Details:   details,
	}

	return ep.publishEvent(ctx, "hulk.runtime.health", event)
}

// publishEvent publishes a generic event
func (ep *EventPublisher) publishEvent(ctx context.Context, subject string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = ep.js.Publish(subject, data)
	if err != nil {
		logger.Error("Failed to publish event",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// TaskEvent represents a task event
type TaskEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// HealthEvent represents a health event
type HealthEvent struct {
	Healthy   bool                   `json:"healthy"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

