// Package value_objects provides value objects tests
package value_objects

import (
	"testing"
)

func TestStackType_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		stack StackType
		want  bool
	}{
		{"Go Premium", StackTypeGoPremium, true},
		{"TinyGo", StackTypeTinyGo, true},
		{"Web", StackTypeWeb, true},
		{"Invalid", StackType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stack.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewStackType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"Valid Go Premium", "go-premium", false},
		{"Valid TinyGo", "tinygo", false},
		{"Valid Web", "web", false},
		{"Invalid", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack, err := NewStackType(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStackType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && stack.String() != tt.value {
				t.Errorf("NewStackType() = %v, want %v", stack.String(), tt.value)
			}
		})
	}
}
