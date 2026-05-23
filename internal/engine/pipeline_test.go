package engine

import (
	"context"
	"net/url"
	"regexp"
	"testing"
	"time"

	"zhiyuwaf/internal/model"
)

func makeRequest(ip, method, path, query, body string) *model.ParsedRequest {
	return &model.ParsedRequest{
		ID:        "test-1",
		Timestamp: time.Now(),
		ClientIP:  ip,
		Method:    method,
		URL:       path + "?" + query,
		Path:      path,
		QueryParams: map[string][]string{
			"q": {query},
		},
		Body:        []byte(body),
		BodyPreview: body,
	}
}

func TestDefaultRulesDoNotBlockNormalLoginPath(t *testing.T) {
	rs := NewRuleSet()
	if err := rs.LoadFromDir("../../configs/rules"); err != nil {
		t.Fatalf("load default rules: %v", err)
	}
	p := NewPipeline(rs, 1000, 1000)

	req := makeRequest("10.0.0.1", "GET", "/login", "", "")
	res := p.Inspect(context.Background(), req)
	if res != nil {
		t.Fatalf("normal login path should pass, got blocked by %s: %s", res.RuleID, res.Message)
	}
}

func TestDefaultRulesDetectDoubleEncodedPayload(t *testing.T) {
	rs := NewRuleSet()
	if err := rs.LoadFromDir("../../configs/rules"); err != nil {
		t.Fatalf("load default rules: %v", err)
	}
	p := NewPipeline(rs, 1000, 1000)

	req := makeRequest("10.0.0.1", "GET", "/search", "%253Cscript%253Ealert(1)%253C%252Fscript%253E", "")
	res := p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Fatal("double-encoded XSS payload should be blocked")
	}
}

func TestRuleMatchIncludesQueryKeys(t *testing.T) {
	rs := NewRuleSet()
	rs.AddRule(&PatternRule{
		BaseRule: BaseRule{
			RuleID:           "TEST-QUERY-KEY",
			RuleName:         "Test Query Key",
			RuleSeverity:     "medium",
			CompiledPatterns: mustCompile(t, `(?i)debug_admin=true`),
			MatchLocations:   []string{"query"},
		},
	})
	p := NewPipeline(rs, 1000, 1000)

	req := makeRequest("10.0.0.1", "GET", "/test", "", "")
	req.QueryParams = url.Values{"debug_admin": {"true"}}
	res := p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Fatal("query parameter names should be inspected with values")
	}
}

func TestPipeline_IPWhitelist(t *testing.T) {
	rs := NewRuleSet()
	p := NewPipeline(rs, 60, 10)
	p.UpdateIPLists(map[string]bool{"10.0.0.1": true}, nil)

	req := makeRequest("10.0.0.1", "GET", "/test", "normal", "")
	res := p.Inspect(context.Background(), req)
	if res != nil {
		t.Errorf("whitelisted IP should pass, got blocked: %+v", res)
	}
}

func TestPipeline_IPBlacklist(t *testing.T) {
	rs := NewRuleSet()
	p := NewPipeline(rs, 60, 10)
	p.UpdateIPLists(nil, map[string]bool{"192.168.1.100": true})

	req := makeRequest("192.168.1.100", "GET", "/test", "normal", "")
	res := p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Error("blacklisted IP should be blocked")
	}
	if res.RuleID != "IP-BLACK" {
		t.Errorf("expected IP-BLACK rule, got %s", res.RuleID)
	}
}

func TestPipeline_RateLimit(t *testing.T) {
	rs := NewRuleSet()
	p := NewPipeline(rs, 2, 1) // 2 per minute, burst 1

	req := makeRequest("10.0.0.5", "GET", "/test", "", "")
	// First 2 requests should pass (burst=1 + 1 token)
	p.Inspect(context.Background(), req)
	p.Inspect(context.Background(), req)
	// Third should be rate limited
	res := p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Error("should be rate limited after exceeding limit")
	}
	if res.RuleID != "RATE-LIMIT" {
		t.Errorf("expected RATE-LIMIT rule, got %s", res.RuleID)
	}
}

func TestPipeline_RuleMatch(t *testing.T) {
	rs := NewRuleSet()
	// Add a simple SQL injection rule
	compiled := mustCompile(t, `(?i)(\bunion\b.*\bselect\b|\bselect\b.*\bfrom\b)`)
	rs.AddRule(&PatternRule{
		BaseRule: BaseRule{
			RuleID:           "TEST-SQLI",
			RuleName:         "Test SQLi",
			RuleSeverity:     "high",
			CompiledPatterns: compiled,
			MatchLocations:   []string{"query", "body"},
		},
	})

	p := NewPipeline(rs, 60, 10)

	// Normal request should pass
	req := makeRequest("10.0.0.1", "GET", "/search", "hello world", "")
	res := p.Inspect(context.Background(), req)
	if res != nil {
		t.Errorf("normal request should pass, got: %+v", res)
	}

	// SQL injection should be blocked
	req = makeRequest("10.0.0.1", "GET", "/search", "1 UNION SELECT * FROM users", "")
	res = p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Error("SQL injection should be blocked")
	}
	if res.RuleID != "TEST-SQLI" {
		t.Errorf("expected TEST-SQLI, got %s", res.RuleID)
	}
}

func TestPipeline_RuleMatchBody(t *testing.T) {
	rs := NewRuleSet()
	compiled := mustCompile(t, `(?i)<script[^>]*>`)
	rs.AddRule(&PatternRule{
		BaseRule: BaseRule{
			RuleID:           "TEST-XSS",
			RuleName:         "Test XSS",
			RuleSeverity:     "high",
			CompiledPatterns: compiled,
			MatchLocations:   []string{"body"},
		},
	})

	p := NewPipeline(rs, 60, 10)

	req := makeRequest("10.0.0.1", "POST", "/comment", "", `<script>alert(1)</script>`)
	res := p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Error("XSS in body should be blocked")
	}
}

func TestPipeline_AIAnalyzer(t *testing.T) {
	rs := NewRuleSet()
	p := NewPipeline(rs, 60, 10)

	// Mock AI analyzer that always blocks
	mock := &mockAIAnalyzer{result: &model.DetectionResult{
		Blocked:  true,
		RuleID:   "AI-TEST",
		RuleName: "AI Test",
		Severity: "high",
		Source:   "ai",
	}}
	p.SetAIAnalyzer(mock)

	req := makeRequest("10.0.0.1", "GET", "/suspicious", "eval(base64_decode(...))", "")
	res := p.Inspect(context.Background(), req)
	if res == nil || !res.Blocked {
		t.Error("AI analyzer should block suspicious request")
	}
	if res.Source != "ai" {
		t.Errorf("expected ai source, got %s", res.Source)
	}
}

func TestPipeline_AIAnalyzerNil(t *testing.T) {
	rs := NewRuleSet()
	p := NewPipeline(rs, 60, 10)
	// No AI analyzer set

	req := makeRequest("10.0.0.1", "GET", "/test", "normal", "")
	res := p.Inspect(context.Background(), req)
	if res != nil {
		t.Errorf("should pass without AI analyzer, got: %+v", res)
	}
}

func TestPipeline_AttackLogChannel(t *testing.T) {
	rs := NewRuleSet()
	p := NewPipeline(rs, 60, 10)
	p.UpdateIPLists(nil, map[string]bool{"1.2.3.4": true})

	req := makeRequest("1.2.3.4", "GET", "/test", "", "")
	p.Inspect(context.Background(), req)

	select {
	case log := <-p.AttackLogChan():
		if log.ClientIP != "1.2.3.4" {
			t.Errorf("expected IP 1.2.3.4, got %s", log.ClientIP)
		}
		if log.RuleID != "IP-BLACK" {
			t.Errorf("expected IP-BLACK, got %s", log.RuleID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected attack log in channel")
	}
}

// mockAIAnalyzer is a test double for AI analysis.
type mockAIAnalyzer struct {
	result *model.DetectionResult
}

func (m *mockAIAnalyzer) AnalyzeAsync(ctx context.Context, req *model.ParsedRequest) <-chan *model.DetectionResult {
	ch := make(chan *model.DetectionResult, 1)
	ch <- m.result
	close(ch)
	return ch
}

func mustCompile(t *testing.T, patterns ...string) []*regexp.Regexp {
	t.Helper()
	var result []*regexp.Regexp
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			t.Fatalf("failed to compile pattern %q: %v", p, err)
		}
		result = append(result, re)
	}
	return result
}
