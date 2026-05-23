package ai

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type perIPEntry struct {
	limiter  *rate.Limiter
	lastSeen int64 // unix nano
}

// PerIPLimiter provides per-IP rate limiting for AI analysis calls.
type PerIPLimiter struct {
	mu       sync.RWMutex
	limits   map[string]*perIPEntry
	rate     rate.Limit
	burst    int
	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewPerIPLimiter(requestsPerMinute, burst int) *PerIPLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 10
	}
	if burst <= 0 {
		burst = 2
	}
	pl := &PerIPLimiter{
		limits: make(map[string]*perIPEntry),
		rate:   rate.Limit(requestsPerMinute) / 60.0,
		burst:  burst,
		stopCh: make(chan struct{}),
	}
	go pl.cleanup()
	return pl
}

// Allow returns true if the given IP has not exceeded its rate limit.
func (pl *PerIPLimiter) Allow(ip string) bool {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	e, ok := pl.limits[ip]
	if !ok {
		e = &perIPEntry{
			limiter: rate.NewLimiter(pl.rate, pl.burst),
		}
		pl.limits[ip] = e
	}
	e.lastSeen = time.Now().UnixNano()
	return e.limiter.Allow()
}

// cleanup periodically removes stale IP entries (no requests for 5 minutes).
func (pl *PerIPLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			pl.mu.Lock()
			cutoff := time.Now().Add(-5 * time.Minute).UnixNano()
			for ip, e := range pl.limits {
				if e.lastSeen < cutoff {
					delete(pl.limits, ip)
				}
			}
			pl.mu.Unlock()
		case <-pl.stopCh:
			return
		}
	}
}

func (pl *PerIPLimiter) Stop() {
	pl.stopOnce.Do(func() { close(pl.stopCh) })
}
