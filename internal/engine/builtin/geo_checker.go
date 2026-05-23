package builtin

import (
	"context"
	"sync"

	"zhiyuwaf/internal/model"
)

type GeoChecker struct {
	mu      sync.RWMutex
	blocked map[string]bool // country code -> true
}

func NewGeoChecker() *GeoChecker {
	return &GeoChecker{
		blocked: make(map[string]bool),
	}
}

// Update sets the blocked country codes.
func (g *GeoChecker) Update(blockedCodes []string) {
	m := make(map[string]bool, len(blockedCodes))
	for _, c := range blockedCodes {
		if c != "" {
			m[c] = true
		}
	}
	g.mu.Lock()
	g.blocked = m
	g.mu.Unlock()
}

// Check returns a DetectionResult if the country code is blocked.
func (g *GeoChecker) Check(ctx context.Context, req *model.ParsedRequest, countryCode string) *model.DetectionResult {
	if countryCode == "" {
		return nil
	}

	g.mu.RLock()
	blocked := g.blocked[countryCode]
	g.mu.RUnlock()

	if blocked {
		return &model.DetectionResult{
			Blocked:  true,
			RuleID:   "GEO-BLOCK",
			RuleName: "Geo-Location Blocked",
			Severity: "medium",
			Message:  "blocked country: " + countryCode,
			Source:   "rule_engine",
		}
	}
	return nil
}
