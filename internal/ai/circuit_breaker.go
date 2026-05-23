package ai

import (
	"sync/atomic"
	"time"
)

const (
	cbClosed   int32 = 0
	cbOpen     int32 = 1
	cbHalfOpen int32 = 2
)

// CircuitBreaker tracks consecutive AI provider failures and temporarily
// disables AI calls when the failure threshold is reached.
type CircuitBreaker struct {
	state         int32
	failCount     int32
	threshold     int
	resetAfter    time.Duration
	lastFailTime  int64 // unix nano
	lastProbeTime int64 // unix nano
}

func NewCircuitBreaker(threshold int, resetAfter time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if resetAfter <= 0 {
		resetAfter = 30 * time.Second
	}
	return &CircuitBreaker{
		threshold:  threshold,
		resetAfter: resetAfter,
	}
}

// Allow returns true if a request should be sent to the AI provider.
func (cb *CircuitBreaker) Allow() bool {
	state := atomic.LoadInt32(&cb.state)
	switch state {
	case cbClosed:
		return true
	case cbOpen:
		// Check if enough time has passed to try a probe
		now := time.Now().UnixNano()
		lastProbe := atomic.LoadInt64(&cb.lastProbeTime)
		if now-lastProbe > int64(cb.resetAfter) {
			if atomic.CompareAndSwapInt32(&cb.state, cbOpen, cbHalfOpen) {
				atomic.StoreInt64(&cb.lastProbeTime, now)
				return true
			}
		}
		return false
	case cbHalfOpen:
		return false // Only one probe at a time
	}
	return true
}

// RecordSuccess resets the failure counter and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failCount, 0)
	atomic.StoreInt32(&cb.state, cbClosed)
}

// RecordFailure increments the failure counter and may open the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	atomic.StoreInt64(&cb.lastFailTime, time.Now().UnixNano())
	count := atomic.AddInt32(&cb.failCount, 1)
	state := atomic.LoadInt32(&cb.state)

	if state == cbHalfOpen {
		// Probe failed, re-open
		atomic.StoreInt32(&cb.state, cbOpen)
		return
	}

	if int(count) >= cb.threshold {
		atomic.StoreInt32(&cb.state, cbOpen)
	}
}

// State returns the current circuit breaker state as a string.
func (cb *CircuitBreaker) State() string {
	switch atomic.LoadInt32(&cb.state) {
	case cbClosed:
		return "closed"
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half-open"
	}
	return "unknown"
}
