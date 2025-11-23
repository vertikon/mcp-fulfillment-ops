// Package messaging provides event routing functionality
package messaging

import (
	"context"
	"testing"
)

func TestNewEventRouter(t *testing.T) {
	router := NewEventRouter()
	if router == nil {
		t.Error("NewEventRouter() returned nil")
	}
}

func TestEventRouter_RegisterHandler(t *testing.T) {
	router := NewEventRouter()

	tests := []struct {
		name    string
		pattern string
		handler EventHandler
		wantErr bool
	}{
		{
			name:    "valid handler",
			pattern: "test.pattern",
			handler: func(ctx context.Context, event *Event) error { return nil },
			wantErr: false,
		},
		{
			name:    "empty pattern",
			pattern: "",
			handler: func(ctx context.Context, event *Event) error { return nil },
			wantErr: true,
		},
		{
			name:    "nil handler",
			pattern: "test.pattern",
			handler: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.RegisterHandler(tt.pattern, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterHandler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEventRouter_Route(t *testing.T) {
	router := NewEventRouter()

	handlerCalled := false
	router.RegisterHandler("test.subject", func(ctx context.Context, event *Event) error {
		handlerCalled = true
		return nil
	})

	ctx := context.Background()
	event := &Event{
		Subject:   "test.subject",
		Type:      "test",
		Payload:   "test",
		Timestamp: 1234567890,
	}

	err := router.Route(ctx, event)
	if err != nil {
		t.Errorf("Route() error = %v", err)
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestEventRouter_matchPattern(t *testing.T) {
	router := NewEventRouter().(*eventRouter)

	tests := []struct {
		name    string
		pattern string
		subject string
		want    bool
	}{
		{
			name:    "exact match",
			pattern: "test.subject",
			subject: "test.subject",
			want:    true,
		},
		{
			name:    "wildcard single token",
			pattern: "test.*",
			subject: "test.anything",
			want:    true,
		},
		{
			name:    "wildcard all remaining",
			pattern: "test.>",
			subject: "test.anything.else",
			want:    true,
		},
		{
			name:    "no match",
			pattern: "test.subject",
			subject: "other.subject",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.matchPattern(tt.pattern, tt.subject)
			if got != tt.want {
				t.Errorf("matchPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
