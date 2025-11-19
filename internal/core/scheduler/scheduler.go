// Package scheduler provides task scheduling with NATS JetStream integration
package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// Scheduler manages scheduled tasks with NATS JetStream
type Scheduler struct {
	js      nats.JetStreamContext
	streams map[string]*nats.StreamConfig
}

// NewScheduler creates a new scheduler with NATS JetStream
func NewScheduler(js nats.JetStreamContext) *Scheduler {
	return &Scheduler{
		js:      js,
		streams: make(map[string]*nats.StreamConfig),
	}
}

// InitializeStreams creates required NATS JetStream streams
func (s *Scheduler) InitializeStreams(ctx context.Context) error {
	streams := []struct {
		name    string
		subjects []string
	}{
		{
			name:     "hulk.engine.tasks",
			subjects: []string{"hulk.task.created", "hulk.task.completed", "hulk.task.failed"},
		},
		{
			name:     "hulk.engine.events",
			subjects: []string{"hulk.engine.*"},
		},
		{
			name:     "hulk.scheduler.queue",
			subjects: []string{"hulk.scheduler.tick"},
		},
		{
			name:     "hulk.errors",
			subjects: []string{"hulk.error.*"},
		},
	}

	for _, stream := range streams {
		cfg := &nats.StreamConfig{
			Name:      stream.name,
			Subjects: stream.subjects,
			Replicas:  1,
			MaxAge:    24 * time.Hour,
		}

		_, err := s.js.AddStream(cfg)
		if err != nil {
			if err == nats.ErrStreamNameExist {
				logger.Debug("Stream already exists", zap.String("stream", stream.name))
				continue
			}
			return err
		}

		s.streams[stream.name] = cfg
		logger.Info("Stream created", zap.String("stream", stream.name))
	}

	return nil
}

// PublishTick publishes a scheduler tick event
func (s *Scheduler) PublishTick(ctx context.Context) error {
	tick := TickEvent{
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(tick)
	if err != nil {
		return err
	}

	_, err = s.js.Publish("hulk.scheduler.tick", data)
	if err != nil {
		return err
	}

	return nil
}

// SubscribeToTicks subscribes to scheduler tick events
func (s *Scheduler) SubscribeToTicks(ctx context.Context, handler func(*TickEvent) error) (*nats.Subscription, error) {
	sub, err := s.js.Subscribe("hulk.scheduler.tick", func(msg *nats.Msg) {
		var tick TickEvent
		if err := json.Unmarshal(msg.Data, &tick); err != nil {
			logger.Error("Failed to unmarshal tick event", zap.Error(err))
			return
		}

		if err := handler(&tick); err != nil {
			logger.Error("Tick handler failed", zap.Error(err))
		}

		msg.Ack()
	}, nats.Durable("scheduler-tick-consumer"))

	return sub, err
}

// TickEvent represents a scheduler tick event
type TickEvent struct {
	Timestamp time.Time `json:"timestamp"`
}

