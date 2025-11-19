// Package engine provides the execution engine with worker pools for concurrent task processing.
package engine

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/vertikon/mcp-fulfillment-ops/pkg/logger"
	"go.uber.org/zap"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int32

const (
	// StateClosed means the circuit is closed and requests pass through
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests are rejected
	StateOpen
	// StateHalfOpen means the circuit is half-open and testing recovery
	StateHalfOpen
)

// CircuitBreaker implements a circuit breaker pattern
type CircuitBreaker struct {
	name          string
	maxFailures   int
	timeout       time.Duration
	resetTimeout  time.Duration
	state         int32 // atomic access
	failures      int32 // atomic access
	lastFailure   time.Time
	mu            sync.RWMutex
	successCount  int32 // atomic access for half-open state
	halfOpenLimit int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:          name,
		maxFailures:   maxFailures,
		resetTimeout:  resetTimeout,
		halfOpenLimit: 3, // Allow 3 successful calls in half-open to close
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	state := cb.getState()

	switch state {
	case StateOpen:
		if cb.shouldAttemptReset() {
			cb.setState(StateHalfOpen)
			logger.Info("Circuit breaker entering half-open state",
				zap.String("name", cb.name),
			)
		} else {
			return ErrCircuitOpen
		}
	case StateHalfOpen:
		// Continue to execute
	case StateClosed:
		// Continue to execute
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// getState returns the current circuit breaker state
func (cb *CircuitBreaker) getState() CircuitState {
	return CircuitState(atomic.LoadInt32(&cb.state))
}

// setState sets the circuit breaker state
func (cb *CircuitBreaker) setState(state CircuitState) {
	atomic.StoreInt32(&cb.state, int32(state))
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	cb.lastFailure = time.Now()
	failures := atomic.AddInt32(&cb.failures, 1)
	cb.mu.Unlock()

	state := cb.getState()

	if state == StateHalfOpen {
		// Failed in half-open, go back to open
		cb.setState(StateOpen)
		logger.Warn("Circuit breaker back to open state",
			zap.String("name", cb.name),
		)
		return
	}

	if failures >= int32(cb.maxFailures) {
		cb.setState(StateOpen)
		logger.Warn("Circuit breaker opened",
			zap.String("name", cb.name),
			zap.Int32("failures", failures),
		)
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	state := cb.getState()

	if state == StateHalfOpen {
		successes := atomic.AddInt32(&cb.successCount, 1)
		if successes >= int32(cb.halfOpenLimit) {
			cb.setState(StateClosed)
			atomic.StoreInt32(&cb.failures, 0)
			atomic.StoreInt32(&cb.successCount, 0)
			logger.Info("Circuit breaker closed after recovery",
				zap.String("name", cb.name),
			)
		}
	} else if state == StateOpen {
		// Should not happen, but reset if it does
		cb.setState(StateClosed)
		atomic.StoreInt32(&cb.failures, 0)
	} else {
		// Closed state - reset failure count on success
		atomic.StoreInt32(&cb.failures, 0)
	}
}

// shouldAttemptReset checks if enough time has passed to attempt reset
func (cb *CircuitBreaker) shouldAttemptReset() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return time.Since(cb.lastFailure) >= cb.resetTimeout
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	return cb.getState()
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		Name:        cb.name,
		State:       cb.getState(),
		Failures:    atomic.LoadInt32(&cb.failures),
		LastFailure: cb.lastFailure,
	}
}

// CircuitBreakerStats represents circuit breaker statistics
type CircuitBreakerStats struct {
	Name        string
	State       CircuitState
	Failures    int32
	LastFailure time.Time
}

// Errors
var (
	ErrCircuitOpen = &Error{Message: "circuit breaker is open"}
)
