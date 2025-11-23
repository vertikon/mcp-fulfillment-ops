// Package streaming provides NATS JetStream streaming implementation
package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// JetStreamClient provides NATS JetStream operations
type JetStreamClient interface {
	// Publish publishes a message to a subject
	Publish(ctx context.Context, subject string, data []byte) error

	// Subscribe creates a durable consumer subscription
	Subscribe(ctx context.Context, subject string, durableName string, handler func(*Message) error) error

	// CreateStream creates a new JetStream stream
	CreateStream(ctx context.Context, config *StreamConfig) error

	// DeleteStream deletes a JetStream stream
	DeleteStream(ctx context.Context, streamName string) error

	// Close closes the connection
	Close() error
}

// Message represents a NATS message
type Message struct {
	Subject string
	Data    []byte
	Headers map[string]string
}

// StreamConfig represents a JetStream stream configuration
type StreamConfig struct {
	Name      string
	Subjects  []string
	Replicas  int
	MaxAge    time.Duration
	MaxBytes  int64
	MaxMsgs   int64
	Retention nats.RetentionPolicy
}

// natsJetStreamClient implements JetStreamClient using NATS
type natsJetStreamClient struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewNATSJetStreamClient creates a new NATS JetStream client
func NewNATSJetStreamClient(urls []string, user, password string) (JetStreamClient, error) {
	opts := []nats.Option{
		nats.Name("mcp-fulfillment-ops-jetstream"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Error("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	}

	if user != "" && password != "" {
		opts = append(opts, nats.UserInfo(user, password))
	}

	// Connect to NATS
	var conn *nats.Conn
	var err error
	if len(urls) > 0 {
		conn, err = nats.Connect(urls[0], opts...)
	} else {
		conn, err = nats.Connect(nats.DefaultURL, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Get JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &natsJetStreamClient{
		conn: conn,
		js:   js,
	}, nil
}

// Publish publishes a message to a subject
func (c *natsJetStreamClient) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := c.js.Publish(subject, data)
	if err != nil {
		logger.Error("Failed to publish message",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	logger.Debug("Published message",
		zap.String("subject", subject),
		zap.Int("size", len(data)),
	)

	return nil
}

// Subscribe creates a durable consumer subscription
func (c *natsJetStreamClient) Subscribe(ctx context.Context, subject string, durableName string, handler func(*Message) error) error {
	_, err := c.js.Subscribe(subject, func(msg *nats.Msg) {
		message := &Message{
			Subject: msg.Subject,
			Data:    msg.Data,
			Headers: make(map[string]string),
		}

		// Copy headers if present
		if msg.Header != nil {
			for key, values := range msg.Header {
				if len(values) > 0 {
					message.Headers[key] = values[0]
				}
			}
		}

		if err := handler(message); err != nil {
			logger.Error("Message handler failed",
				zap.String("subject", subject),
				zap.Error(err),
			)
			// Don't ack on error - let NATS retry
			return
		}

		// Ack successful processing
		msg.Ack()
	}, nats.Durable(durableName), nats.ManualAck())

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	logger.Info("Subscribed to subject",
		zap.String("subject", subject),
		zap.String("durable", durableName),
	)

	return nil
}

// CreateStream creates a new JetStream stream
func (c *natsJetStreamClient) CreateStream(ctx context.Context, config *StreamConfig) error {
	streamConfig := &nats.StreamConfig{
		Name:      config.Name,
		Subjects:  config.Subjects,
		Replicas:  config.Replicas,
		MaxAge:    config.MaxAge,
		MaxBytes:  config.MaxBytes,
		MaxMsgs:   config.MaxMsgs,
		Retention: config.Retention,
	}

	_, err := c.js.AddStream(streamConfig)
	if err != nil {
		// Verifica se o stream já existe tentando obtê-lo
		if _, getErr := c.js.StreamInfo(config.Name); getErr == nil {
			logger.Debug("Stream already exists", zap.String("stream", config.Name))
			return nil
		}
		return fmt.Errorf("failed to create stream %s: %w", config.Name, err)
	}

	logger.Info("Created stream",
		zap.String("stream", config.Name),
		zap.Strings("subjects", config.Subjects),
	)

	return nil
}

// DeleteStream deletes a JetStream stream
func (c *natsJetStreamClient) DeleteStream(ctx context.Context, streamName string) error {
	err := c.js.DeleteStream(streamName)
	if err != nil {
		return fmt.Errorf("failed to delete stream %s: %w", streamName, err)
	}

	logger.Info("Deleted stream", zap.String("stream", streamName))
	return nil
}

// Close closes the connection
func (c *natsJetStreamClient) Close() error {
	if c.conn != nil && c.conn.IsConnected() {
		c.conn.Close()
		logger.Info("Closed NATS connection")
	}
	return nil
}

// PublishJSON publishes a JSON-encoded message
func PublishJSON(ctx context.Context, client JetStreamClient, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	return client.Publish(ctx, subject, data)
}

// SubscribeJSON creates a subscription that unmarshals JSON messages
func SubscribeJSON(ctx context.Context, client JetStreamClient, subject string, durableName string, handler func(*Message, interface{}) error, payloadType interface{}) error {
	return client.Subscribe(ctx, subject, durableName, func(msg *Message) error {
		var payload interface{}
		if payloadType != nil {
			payload = payloadType
		} else {
			payload = make(map[string]interface{})
		}

		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		return handler(msg, payload)
	})
}
