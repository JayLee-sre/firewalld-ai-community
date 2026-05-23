package builtin

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen int64
}

type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rpm      int
	burst    int
	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewRateLimiter(requestsPerMinute, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rpm:      requestsPerMinute,
		burst:    burst,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &rateLimiterEntry{
			limiter: rate.NewLimiter(rate.Limit(rl.rpm)/60.0, rl.burst),
		}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now().UnixNano()
	limiter := entry.limiter
	rl.mu.Unlock()

	return limiter.Allow()
}

// cleanupLoop periodically removes IP entries not seen for 10 minutes.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute).UnixNano()
			for ip, e := range rl.limiters {
				if e.lastSeen < cutoff {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stopCh) })
}
