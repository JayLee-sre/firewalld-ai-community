package ai

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache(1 * time.Hour)

	resp := &AnalysisResponse{
		IsMalicious: true,
		Confidence:  0.95,
		AttackType:  "sqli",
		Reasoning:   "SQL injection detected",
	}
	c.Set("key1", resp)

	got := c.Get("key1")
	if got == nil {
		t.Fatal("expected cached response, got nil")
	}
	if got.AttackType != "sqli" {
		t.Errorf("expected sqli, got %s", got.AttackType)
	}
}

func TestCache_Expiry(t *testing.T) {
	c := NewCache(50 * time.Millisecond)

	resp := &AnalysisResponse{IsMalicious: false, AttackType: "normal"}
	c.Set("key1", resp)

	// Should be cached immediately
	if c.Get("key1") == nil {
		t.Error("should be cached immediately")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	if c.Get("key1") != nil {
		t.Error("should be expired after TTL")
	}
}

func TestCache_Miss(t *testing.T) {
	c := NewCache(1 * time.Hour)
	if c.Get("nonexistent") != nil {
		t.Error("should return nil for missing key")
	}
}

func TestCacheKey_Deterministic(t *testing.T) {
	k1 := CacheKey("1.2.3.4", "GET", "/test", "a=1", "body")
	k2 := CacheKey("1.2.3.4", "GET", "/test", "a=1", "body")
	if k1 != k2 {
		t.Error("same inputs should produce same cache key")
	}

	k3 := CacheKey("1.2.3.4", "GET", "/test", "a=2", "body")
	if k1 == k3 {
		t.Error("different inputs should produce different cache keys")
	}
}
