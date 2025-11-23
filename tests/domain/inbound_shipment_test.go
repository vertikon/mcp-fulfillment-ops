package domain_test

import (
	"testing"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/fulfillment"
)

func TestNewInboundShipment(t *testing.T) {
	tests := []struct {
		name        string
		refID       string
		origin      string
		dest        string
		items       []fulfillment.Item
		wantErr     bool
		errContains string
	}{
		{
			name:   "valid shipment",
			refID:  "REF-001",
			origin: "Fornecedor A",
			dest:   "CD-SP",
			items: []fulfillment.Item{
				{SKU: "SKU-001", Quantity: 10},
			},
			wantErr: false,
		},
		{
			name:        "empty items",
			refID:       "REF-002",
			origin:      "Fornecedor B",
			dest:        "CD-RJ",
			items:       []fulfillment.Item{},
			wantErr:     true,
			errContains: "empty items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shipment, err := fulfillment.NewInboundShipment(tt.refID, tt.origin, tt.dest, tt.items)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if shipment.ReferenceID != tt.refID {
				t.Errorf("ReferenceID = %v, want %v", shipment.ReferenceID, tt.refID)
			}
			if shipment.Origin != tt.origin {
				t.Errorf("Origin = %v, want %v", shipment.Origin, tt.origin)
			}
			if shipment.Destination != tt.dest {
				t.Errorf("Destination = %v, want %v", shipment.Destination, tt.dest)
			}
			if shipment.Status != fulfillment.StatusPending {
				t.Errorf("Status = %v, want %v", shipment.Status, fulfillment.StatusPending)
			}
		})
	}
}

func TestInboundShipment_StartReceiving(t *testing.T) {
	shipment, _ := fulfillment.NewInboundShipment("REF-001", "Origin", "Dest", []fulfillment.Item{{SKU: "SKU-001", Quantity: 10}})

	tests := []struct {
		name       string
		setup      func(*fulfillment.InboundShipment)
		wantErr    bool
		wantStatus fulfillment.Status
	}{
		{
			name: "valid transition pending -> in_progress",
			setup: func(s *fulfillment.InboundShipment) {
				s.Status = fulfillment.StatusPending
			},
			wantErr:    false,
			wantStatus: fulfillment.StatusInProgress,
		},
		{
			name: "invalid transition completed -> in_progress",
			setup: func(s *fulfillment.InboundShipment) {
				s.Status = fulfillment.StatusCompleted
			},
			wantErr:    true,
			wantStatus: fulfillment.StatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(shipment)
			err := shipment.StartReceiving()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if shipment.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", shipment.Status, tt.wantStatus)
			}
		})
	}
}

func TestInboundShipment_Complete(t *testing.T) {
	shipment, _ := fulfillment.NewInboundShipment("REF-001", "Origin", "Dest", []fulfillment.Item{{SKU: "SKU-001", Quantity: 10}})

	tests := []struct {
		name       string
		setup      func(*fulfillment.InboundShipment)
		wantErr    bool
		wantStatus fulfillment.Status
	}{
		{
			name: "valid transition in_progress -> completed",
			setup: func(s *fulfillment.InboundShipment) {
				s.Status = fulfillment.StatusInProgress
			},
			wantErr:    false,
			wantStatus: fulfillment.StatusCompleted,
		},
		{
			name: "invalid transition pending -> completed",
			setup: func(s *fulfillment.InboundShipment) {
				s.Status = fulfillment.StatusPending
			},
			wantErr:    true,
			wantStatus: fulfillment.StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(shipment)
			err := shipment.Complete()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if shipment.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", shipment.Status, tt.wantStatus)
			}

			if shipment.CompletedAt == nil {
				t.Error("CompletedAt should be set")
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
