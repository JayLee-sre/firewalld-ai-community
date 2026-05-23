package ai

import (
	"context"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"zhiyuwaf/internal/model"
)

type asyncAnalyzer struct {
	mu             sync.RWMutex
	provider       Provider
	cache          *LRUCache
	rateLimiter    *rate.Limiter
	perIPLimiter   *PerIPLimiter
	circuitBreaker *CircuitBreaker
	timeout        time.Duration
	failOpen       bool
	highRisk       []string
	onCall         func()
	checkAllowed   func() bool
}

func NewAnalyzer(provider Provider, cacheTTL time.Duration, maxReqPerMin int, timeout int, failOpen bool, highRiskPaths []string, perIPRate, perIPBurst, circuitThreshold int, circuitResetSec int) Analyzer {
	if maxReqPerMin <= 0 {
		maxReqPerMin = 60
	}
	burst := maxReqPerMin / 10
	if burst < 1 {
		burst = 1
	}
	if timeout <= 0 {
		timeout = 5
	}

	return &asyncAnalyzer{
		provider:       provider,
		cache:          NewCache(cacheTTL),
		rateLimiter:    rate.NewLimiter(rate.Limit(maxReqPerMin)/60.0, burst),
		perIPLimiter:   NewPerIPLimiter(perIPRate, perIPBurst),
		circuitBreaker: NewCircuitBreaker(circuitThreshold, time.Duration(circuitResetSec)*time.Second),
		timeout:        time.Duration(timeout) * time.Second,
		failOpen:       failOpen,
		highRisk:       normalizeHighRiskPaths(highRiskPaths),
	}
}

func (a *asyncAnalyzer) SetProvider(p Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.provider = p
}

func (a *asyncAnalyzer) SetOnCall(fn func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onCall = fn
}

func (a *asyncAnalyzer) SetAllowedCheck(fn func() bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.checkAllowed = fn
}

// Stop shuts down background goroutines (cache cleanup, per-IP limiter cleanup).
func (a *asyncAnalyzer) Stop() {
	a.cache.Stop()
	a.perIPLimiter.Stop()
}

func (a *asyncAnalyzer) AnalyzeAsync(ctx context.Context, req *model.ParsedRequest) <-chan *model.DetectionResult {
	ch := make(chan *model.DetectionResult, 1)

	go func() {
		defer close(ch)

		a.mu.RLock()
		provider := a.provider
		failOpen := a.failOpen
		onCall := a.onCall
		checkAllowed := a.checkAllowed
		a.mu.RUnlock()

		if provider == nil {
			return
		}

		query := url.Values(req.QueryParams).Encode()
		highRisk := a.isHighRisk(req.Path)
		failOpenForReq := failOpen && !highRisk

		// Check cache
		key := CacheKey(req.ClientIP, req.Method, req.Path, query, req.BodyPreview)
		if cached := a.cache.Get(key); cached != nil {
			if cached.IsMalicious {
				ch <- &model.DetectionResult{
					Blocked:  true,
					RuleID:   "AI-" + cached.AttackType,
					RuleName: "AI Detection: " + cached.AttackType,
					Severity: severityFromConfidence(cached.Confidence),
					Message:  cached.Reasoning,
					Source:   "ai",
				}
			}
			return
		}

		// Circuit breaker: if open, skip AI entirely (rules engine still runs)
		if !a.circuitBreaker.Allow() {
			log.Printf("AI circuit breaker open, skipping analysis for %s", req.ClientIP)
			return
		}

		// Per-IP rate limit: if exceeded, always block (fail_closed for this IP)
		if !a.perIPLimiter.Allow(req.ClientIP) {
			log.Printf("AI per-IP rate limit exceeded for %s", req.ClientIP)
			ch <- aiUnavailableResult("AI per-IP rate limit exceeded for " + req.ClientIP)
			return
		}

		// Global rate limit: if exceeded, behavior depends on fail_open
		if !a.rateLimiter.Allow() {
			log.Println("AI global rate limit exceeded, skipping analysis")
			if !failOpenForReq {
				ch <- aiUnavailableResult("AI rate limit exceeded")
			}
			return
		}

		// Community daily limit check
		if checkAllowed != nil && !checkAllowed() {
			log.Println("AI community daily limit reached, blocking")
			ch <- aiUnavailableResult("AI 每日调用次数已达上限")
			return
		}

		// Call provider with timeout
		aiCtx, cancel := context.WithTimeout(ctx, a.timeout)
		defer cancel()

		analysisReq := AnalysisRequest{
			ClientIP:    req.ClientIP,
			Method:      req.Method,
			Path:        req.Path,
			Query:       query,
			Context:     businessContext(req.Method, req.Path),
			HighRisk:    highRisk,
			Headers:     req.Headers,
			BodyPreview: req.BodyPreview,
		}

		resp, err := provider.Analyze(aiCtx, analysisReq)
		if err != nil {
			log.Printf("AI analysis error: %v", err)
			a.circuitBreaker.RecordFailure()
			if !failOpenForReq {
				ch <- aiUnavailableResult("AI analysis unavailable")
			}
			return
		}

		// Provider call succeeded
		a.circuitBreaker.RecordSuccess()
		if onCall != nil {
			onCall()
		}

		// Cache the result
		a.cache.Set(key, resp)

		if resp.IsMalicious {
			ch <- &model.DetectionResult{
				Blocked:  true,
				RuleID:   "AI-" + resp.AttackType,
				RuleName: "AI Detection: " + resp.AttackType,
				Severity: severityFromConfidence(resp.Confidence),
				Message:  resp.Reasoning,
				Source:   "ai",
			}
		}
	}()

	return ch
}

func normalizeHighRiskPaths(paths []string) []string {
	if len(paths) == 0 {
		paths = []string{"/admin", "/api/admin", "/api/v1/admin", "/login", "/api/v1/auth/login", "/upload", "/payment", "/pay", "/checkout"}
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (a *asyncAnalyzer) isHighRisk(path string) bool {
	a.mu.RLock()
	highRisk := a.highRisk
	a.mu.RUnlock()

	path = strings.ToLower(path)
	for _, prefix := range highRisk {
		if path == prefix || strings.HasPrefix(path, strings.TrimRight(prefix, "/")+"/") {
			return true
		}
	}
	return false
}

func businessContext(method, path string) string {
	p := strings.ToLower(path)
	switch {
	case strings.Contains(p, "login") || strings.Contains(p, "auth"):
		return "authentication endpoint: watch for credential stuffing, brute force signals, SQL injection, account enumeration, and abnormal automation"
	case strings.Contains(p, "admin") || strings.Contains(p, "manage"):
		return "administration endpoint: apply stricter scrutiny to privilege bypass, command execution, traversal, and unsafe state changes"
	case strings.Contains(p, "upload") || strings.Contains(p, "file") || strings.Contains(p, "import"):
		return "file handling endpoint: watch for webshells, unsafe extensions, path traversal, archive abuse, and content-type mismatch"
	case strings.Contains(p, "pay") || strings.Contains(p, "payment") || strings.Contains(p, "checkout") || strings.Contains(p, "order"):
		return "payment/order endpoint: watch for tampering, replay, parameter manipulation, and privilege/order ownership bypass"
	case method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE":
		return "state-changing endpoint: inspect parameters and body more strictly than read-only traffic"
	default:
		return "general web request"
	}
}

func aiUnavailableResult(message string) *model.DetectionResult {
	return &model.DetectionResult{
		Blocked:  true,
		RuleID:   "AI-UNAVAILABLE",
		RuleName: "AI Detection Unavailable",
		Severity: "medium",
		Message:  message,
		Source:   "ai",
	}
}

func severityFromConfidence(c float64) string {
	if c >= 0.9 {
		return "critical"
	}
	if c >= 0.7 {
		return "high"
	}
	if c >= 0.5 {
		return "medium"
	}
	return "low"
}
