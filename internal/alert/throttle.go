package alert

import (
	"sync"
	"time"
)

// ThrottleMap prevents duplicate alerts within a time window.
type ThrottleMap struct {
	mu     sync.Mutex
	items  map[string]time.Time
	window time.Duration
}

// NewThrottleMap creates a new throttle map with the given window in minutes.
func NewThrottleMap(windowMinutes int) *ThrottleMap {
	if windowMinutes <= 0 {
		windowMinutes = 10
	}
	return &ThrottleMap{
		items:  make(map[string]time.Time),
		window: time.Duration(windowMinutes) * time.Minute,
	}
}

// ShouldSend returns true if this key has not been sent within the window.
func (t *ThrottleMap) ShouldSend(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	if last, ok := t.items[key]; ok {
		if now.Sub(last) < t.window {
			return false
		}
	}
	t.items[key] = now

	// Periodic cleanup: remove entries older than 2x window
	for k, v := range t.items {
		if now.Sub(v) > t.window*2 {
			delete(t.items, k)
		}
	}
	return true
}
