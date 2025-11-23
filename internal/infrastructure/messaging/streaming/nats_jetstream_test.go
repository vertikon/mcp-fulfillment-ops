// Package streaming provides NATS JetStream streaming implementation
package streaming

import (
	"context"
	"testing"
	"time"
)

func TestNewNATSJetStreamClient(t *testing.T) {
	tests := []struct {
		name     string
		urls     []string
		user     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid connection",
			urls:     []string{"nats://localhost:4222"},
			user:     "",
			password: "",
			wantErr:  false,
		},
		{
			name:     "with credentials",
			urls:     []string{"nats://localhost:4222"},
			user:     "test",
			password: "test",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewNATSJetStreamClient(tt.urls, tt.user, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNATSJetStreamClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if client != nil {
				defer client.Close()
			}
		})
	}
}

func TestNATSJetStreamClient_Publish(t *testing.T) {
	// Skip if NATS not available
	t.Skip("Requires NATS server running")

	client, err := NewNATSJetStreamClient([]string{"nats://localhost:4222"}, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	err = client.Publish(ctx, "test.subject", []byte("test message"))
	if err != nil {
		t.Errorf("Publish() error = %v", err)
	}
}

func TestNATSJetStreamClient_CreateStream(t *testing.T) {
	// Skip if NATS not available
	t.Skip("Requires NATS server running")

	client, err := NewNATSJetStreamClient([]string{"nats://localhost:4222"}, "", "")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	config := &StreamConfig{
		Name:     "test-stream",
		Subjects: []string{"test.>"},
		Replicas: 1,
		MaxAge:   1 * time.Hour,
	}

	err = client.CreateStream(ctx, config)
	if err != nil {
		t.Errorf("CreateStream() error = %v", err)
	}
}
