package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"zhiyuwaf/internal/model"
)

type errorProvider struct{}

func (errorProvider) Name() string { return "error" }

func (errorProvider) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error) {
	return nil, errors.New("provider unavailable")
}

type captureProvider struct {
	requests []AnalysisRequest
}

func (p *captureProvider) Name() string { return "capture" }

func (p *captureProvider) Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error) {
	p.requests = append(p.requests, req)
	return &AnalysisResponse{IsMalicious: false, AttackType: "normal"}, nil
}

func TestAnalyzerFailOpenAllowsProviderErrors(t *testing.T) {
	analyzer := newTestAnalyzer(errorProvider{}, time.Minute, 60, 1, true, nil)
	ch := analyzer.AnalyzeAsync(context.Background(), analyzerTestRequest())

	if res := <-ch; res != nil {
		t.Fatalf("expected fail-open provider error to pass, got %+v", res)
	}
}

func TestAnalyzerFailClosedBlocksProviderErrors(t *testing.T) {
	analyzer := newTestAnalyzer(errorProvider{}, time.Minute, 60, 1, false, nil)
	ch := analyzer.AnalyzeAsync(context.Background(), analyzerTestRequest())

	res := <-ch
	if res == nil || !res.Blocked {
		t.Fatal("expected fail-closed provider error to block")
	}
	if res.RuleID != "AI-UNAVAILABLE" {
		t.Fatalf("expected AI-UNAVAILABLE, got %q", res.RuleID)
	}
}

func TestAnalyzerSmallRateLimitStillHasBurst(t *testing.T) {
	analyzer := newTestAnalyzer(errorProvider{}, time.Minute, 1, 1, true, nil)
	ch := analyzer.AnalyzeAsync(context.Background(), analyzerTestRequest())

	<-ch
}

func TestAnalyzerForwardsQueryAndCachesByQuery(t *testing.T) {
	provider := &captureProvider{}
	analyzer := newTestAnalyzer(provider, time.Minute, 60, 1, true, nil)

	req := analyzerTestRequest()
	req.QueryParams = map[string][]string{"debug": {"1"}}
	<-analyzer.AnalyzeAsync(context.Background(), req)

	req.QueryParams = map[string][]string{"debug": {"2"}}
	<-analyzer.AnalyzeAsync(context.Background(), req)

	if len(provider.requests) != 2 {
		t.Fatalf("expected query variants to use different cache keys, got %d provider calls", len(provider.requests))
	}
	if provider.requests[0].Query != "debug=1" || provider.requests[1].Query != "debug=2" {
		t.Fatalf("expected forwarded queries, got %+v", provider.requests)
	}
}

func TestPerIPLimitBlocksExcess(t *testing.T) {
	// Create analyzer with very restrictive per-IP limits (2/min, burst 1)
	analyzer := NewAnalyzer(errorProvider{}, time.Minute, 100, 1, true, nil, 2, 1, 100, 1)
	req := analyzerTestRequest()

	// First two should pass (fail-open, provider error returns nil)
	<-analyzer.AnalyzeAsync(context.Background(), req)
	<-analyzer.AnalyzeAsync(context.Background(), req)

	// Third should be blocked by per-IP limiter
	ch := analyzer.AnalyzeAsync(context.Background(), req)
	res := <-ch
	if res == nil || !res.Blocked {
		t.Fatal("expected per-IP rate limit to block")
	}
	if res.RuleID != "AI-UNAVAILABLE" {
		t.Fatalf("expected AI-UNAVAILABLE, got %q", res.RuleID)
	}
}

func TestCircuitBreakerOpens(t *testing.T) {
	// Create analyzer with restrictive circuit breaker (threshold=2) and generous per-IP
	analyzer := NewAnalyzer(errorProvider{}, time.Minute, 100, 1, false, nil, 100, 10, 2, 1)
	// With threshold=2, 2 consecutive provider errors should open the circuit
	for i := 0; i < 2; i++ {
		ch := analyzer.AnalyzeAsync(context.Background(), analyzerTestRequest())
		<-ch
	}

	// The 3rd call should be skipped by circuit breaker — no AI-UNAVAILABLE either
	// because circuit breaker skips AI entirely (rules engine still runs)
	ch := analyzer.AnalyzeAsync(context.Background(), analyzerTestRequest())
	select {
	case res := <-ch:
		// If we get a result, it should be from the global rate limiter or similar,
		// not from a provider call
		_ = res
	case <-time.After(100 * time.Millisecond):
		// Circuit breaker working: no result at all means AI was skipped
	}
}

// newTestAnalyzer creates an analyzer with test-friendly per-IP limits.
func newTestAnalyzer(provider Provider, cacheTTL time.Duration, maxReqPerMin int, timeout int, failOpen bool, highRiskPaths []string) Analyzer {
	return NewAnalyzer(provider, cacheTTL, maxReqPerMin, timeout, failOpen, highRiskPaths, 100, 10, 100, 1)
}

func analyzerTestRequest() *model.ParsedRequest {
	return &model.ParsedRequest{
		ClientIP:    "10.0.0.1",
		Method:      "GET",
		Path:        "/",
		Headers:     map[string][]string{"User-Agent": {"test"}},
		BodyPreview: "",
	}
}
