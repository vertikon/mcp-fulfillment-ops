package domain_test

import (
	"testing"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

func TestValidateStateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    fulfillment.Status
		to      fulfillment.Status
		want    bool
	}{
		{
			name: "pending -> in_progress",
			from: fulfillment.StatusPending,
			to:   fulfillment.StatusInProgress,
			want: true,
		},
		{
			name: "pending -> cancelled",
			from: fulfillment.StatusPending,
			to:   fulfillment.StatusCancelled,
			want: true,
		},
		{
			name: "in_progress -> completed",
			from: fulfillment.StatusInProgress,
			to:   fulfillment.StatusCompleted,
			want: true,
		},
		{
			name: "in_progress -> failed",
			from: fulfillment.StatusInProgress,
			to:   fulfillment.StatusFailed,
			want: true,
		},
		{
			name: "pending -> completed (invalid)",
			from: fulfillment.StatusPending,
			to:   fulfillment.StatusCompleted,
			want: false,
		},
		{
			name: "completed -> in_progress (invalid)",
			from: fulfillment.StatusCompleted,
			to:   fulfillment.StatusInProgress,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fulfillment.ValidateStateTransition(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("ValidateStateTransition(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestCheckSLA(t *testing.T) {
	tests := []struct {
		name              string
		createdAt         time.Time
		maxDurationMinutes int
		want              bool
	}{
		{
			name:              "within SLA",
			createdAt:         time.Now().Add(-30 * time.Minute),
			maxDurationMinutes: 60,
			want:              true,
		},
		{
			name:              "exceeded SLA",
			createdAt:         time.Now().Add(-90 * time.Minute),
			maxDurationMinutes: 60,
			want:              false,
		},
		{
			name:              "exactly at SLA limit",
			createdAt:         time.Now().Add(-60 * time.Minute),
			maxDurationMinutes: 60,
			want:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fulfillment.CheckSLA(tt.createdAt, tt.maxDurationMinutes)
			if got != tt.want {
				t.Errorf("CheckSLA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIdempotencyKeyValid(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "valid key",
			key:  "order-123",
			want: true,
		},
		{
			name: "empty key",
			key:  "",
			want: false,
		},
		{
			name: "key too long",
			key:  string(make([]byte, 256)),
			want: false,
		},
		{
			name: "key at max length",
			key:  string(make([]byte, 255)),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fulfillment.IsIdempotencyKeyValid(tt.key)
			if got != tt.want {
				t.Errorf("IsIdempotencyKeyValid(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

