package ai

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	response *AnalysisResponse
	expiry   time.Time
}

type LRUCache struct {
	mu       sync.RWMutex
	data     map[string]*cacheEntry
	ttl      time.Duration
	stopOnce sync.Once
	stopCh   chan struct{}
}

func NewCache(ttl time.Duration) *LRUCache {
	c := &LRUCache{
		data:   make(map[string]*cacheEntry),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}
	go c.cleanup()
	return c
}

func (c *LRUCache) Get(key string) *AnalysisResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok || time.Now().After(entry.expiry) {
		return nil
	}
	return entry.response
}

func (c *LRUCache) Set(key string, resp *AnalysisResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = &cacheEntry{
		response: resp,
		expiry:   time.Now().Add(c.ttl),
	}
}

func (c *LRUCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.data {
				if now.After(v.expiry) {
					delete(c.data, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

func (c *LRUCache) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

func CacheKey(clientIP, method, path, query, bodyPreview string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%s:%s", clientIP, method, path, query, bodyPreview)))
	return fmt.Sprintf("%x", h)
}
