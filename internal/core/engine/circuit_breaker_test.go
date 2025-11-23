package engine

import (
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name         string
		maxFailures  int
		resetTimeout time.Duration
	}{
		{
			name:         "valid breaker",
			maxFailures:  5,
			resetTimeout: time.Second,
		},
		{
			name:         "zero failures",
			maxFailures:  0,
			resetTimeout: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCircuitBreaker("test", tt.maxFailures, tt.resetTimeout)
			if cb == nil {
				t.Fatal("NewCircuitBreaker returned nil")
			}
			if cb.name != "test" {
				t.Errorf("Expected name 'test', got '%s'", cb.name)
			}
			if cb.maxFailures != tt.maxFailures {
				t.Errorf("Expected maxFailures %d, got %d", tt.maxFailures, cb.maxFailures)
			}
			if cb.resetTimeout != tt.resetTimeout {
				t.Errorf("Expected resetTimeout %v, got %v", tt.resetTimeout, cb.resetTimeout)
			}
			if cb.halfOpenLimit != 3 {
				t.Errorf("Expected halfOpenLimit 3, got %d", cb.halfOpenLimit)
			}
			if cb.State() != StateClosed {
				t.Error("Circuit breaker should start in Closed state")
			}
		})
	}
}

func TestCircuitBreaker_Execute_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, time.Second)

	tests := []struct {
		name      string
		fn        func() error
		wantErr   bool
		wantState CircuitState
	}{
		{
			name: "successful execution",
			fn: func() error {
				return nil
			},
			wantErr:   false,
			wantState: StateClosed,
		},
		{
			name: "failing execution",
			fn: func() error {
				return errors.New("operation failed")
			},
			wantErr:   true,
			wantState: StateClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cb.Execute(tt.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cb.State() != tt.wantState {
				t.Errorf("State() = %v, want %v", cb.State(), tt.wantState)
			}
		})
	}
}

func TestCircuitBreaker_OpenState(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 100*time.Millisecond)

	// Cause failures to open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Circuit should be open
	if cb.State() != StateOpen {
		t.Errorf("Expected StateOpen, got %v", cb.State())
	}

	// Execute should fail immediately
	err := cb.Execute(func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 50*time.Millisecond)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Fatal("Circuit should be open")
	}

	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)

	// First successful call should transition to half-open
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("Expected StateHalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreaker_RecoveryFromHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 50*time.Millisecond)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)

	// Execute 3 successful calls to close the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
	}

	// Circuit should be closed
	if cb.State() != StateClosed {
		t.Errorf("Expected StateClosed, got %v", cb.State())
	}

	stats := cb.Stats()
	if stats.Failures != 0 {
		t.Errorf("Expected failures to be reset, got %d", stats.Failures)
	}
}

func TestCircuitBreaker_FailureInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 50*time.Millisecond)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Wait for reset timeout
	time.Sleep(100 * time.Millisecond)

	// Transition to half-open
	_ = cb.Execute(func() error {
		return nil
	})

	if cb.State() != StateHalfOpen {
		t.Fatal("Circuit should be in half-open state")
	}

	// Failure in half-open should return to open
	err := cb.Execute(func() error {
		return errors.New("failure")
	})
	if err == nil {
		t.Error("Expected error from failing function")
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected StateOpen, got %v", cb.State())
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker("test", 3, time.Second)

	// Initial stats
	stats := cb.Stats()
	if stats.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", stats.Name)
	}
	if stats.State != StateClosed {
		t.Errorf("Expected StateClosed, got %v", stats.State)
	}

	// Cause a failure
	_ = cb.Execute(func() error {
		return errors.New("failure")
	})

	stats = cb.Stats()
	if stats.Failures == 0 {
		t.Error("Expected failures to be > 0")
	}
	if stats.LastFailure.IsZero() {
		t.Error("Expected LastFailure to be set")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker("test", 10, time.Second)

	// Concurrent executions
	done := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		go func(id int) {
			err := cb.Execute(func() error {
				if id%2 == 0 {
					return nil
				}
				return errors.New("failure")
			})
			done <- (err == nil)
		}(i)
	}

	// Wait for all executions
	successCount := 0
	for i := 0; i < 20; i++ {
		if <-done {
			successCount++
		}
	}

	// Verify circuit breaker state is consistent
	state := cb.State()
	if state != StateClosed && state != StateOpen && state != StateHalfOpen {
		t.Errorf("Invalid state: %v", state)
	}

	stats := cb.Stats()
	if stats.Failures < 0 {
		t.Error("Failures should not be negative")
	}
}

func TestCircuitBreaker_ResetTimeout(t *testing.T) {
	cb := NewCircuitBreaker("test", 2, 100*time.Millisecond)

	// Open the circuit
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Fatal("Circuit should be open")
	}

	// Try immediately - should fail
	err := cb.Execute(func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Now should transition to half-open
	err = cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.State() != StateHalfOpen {
		t.Errorf("Expected StateHalfOpen, got %v", cb.State())
	}
}
