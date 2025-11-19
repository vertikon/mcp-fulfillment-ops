// Package entities provides domain entities tests
package entities

import (
	"testing"

	"github.com/vertikon/mcp-fulfillment-ops/internal/domain/value_objects"
)

func TestNewMCP(t *testing.T) {
	tests := []struct {
		name        string
		description string
		stack       value_objects.StackType
		wantErr     bool
	}{
		{
			name:        "Valid MCP",
			description: "Test MCP",
			stack:       value_objects.StackTypeGoPremium,
			wantErr:     false,
		},
		{
			name:        "",
			description: "Test MCP",
			stack:       value_objects.StackTypeGoPremium,
			wantErr:     true,
		},
		{
			name:        "Invalid Stack",
			description: "Test MCP",
			stack:       value_objects.StackType("invalid"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcp, err := NewMCP(tt.name, tt.description, tt.stack)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMCP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && mcp == nil {
				t.Errorf("NewMCP() returned nil MCP")
			}
			if !tt.wantErr && mcp.ID() == "" {
				t.Errorf("NewMCP() returned MCP with empty ID")
			}
		})
	}
}

func TestMCP_SetPath(t *testing.T) {
	mcp, _ := NewMCP("Test", "Description", value_objects.StackTypeGoPremium)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"Valid path", "/path/to/mcp", false},
		{"Empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mcp.SetPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && mcp.Path() != tt.path {
				t.Errorf("SetPath() path = %v, want %v", mcp.Path(), tt.path)
			}
		})
	}
}

func TestMCP_AddFeature(t *testing.T) {
	mcp, _ := NewMCP("Test", "Description", value_objects.StackTypeGoPremium)
	feature, _ := value_objects.NewFeature("test-feature", value_objects.FeatureStatusEnabled, "Test feature")

	if err := mcp.AddFeature(feature); err != nil {
		t.Errorf("AddFeature() error = %v", err)
	}

	if len(mcp.Features()) != 1 {
		t.Errorf("AddFeature() features count = %v, want 1", len(mcp.Features()))
	}

	// Test duplicate
	if err := mcp.AddFeature(feature); err == nil {
		t.Errorf("AddFeature() should fail on duplicate")
	}
}

func TestMCP_AddContext(t *testing.T) {
	mcp, _ := NewMCP("Test", "Description", value_objects.StackTypeGoPremium)

	tests := []struct {
		name        string
		knowledgeID string
		documents   []string
		wantErr     bool
	}{
		{"Valid context", "knowledge-1", []string{"doc1"}, false},
		{"Empty knowledge ID", "", []string{"doc1"}, true},
		{"Empty documents", "knowledge-1", []string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mcp.AddContext(tt.knowledgeID, tt.documents, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddContext() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !mcp.HasContext() {
				t.Errorf("AddContext() context not set")
			}
		})
	}
}

