package engine

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"sync"
	"time"

	"zhiyuwaf/internal/engine/builtin"
	"zhiyuwaf/internal/model"
)

// AnalyzerInterface allows the pipeline to use AI analysis without importing the ai package.
type AnalyzerInterface interface {
	AnalyzeAsync(ctx context.Context, req *model.ParsedRequest) <-chan *model.DetectionResult
}

// GeoResolver resolves an IP to a country code.
type GeoResolver interface {
	GetCountry(ip string) string
	GetCountryCode(ip string) string
}

type Pipeline struct {
	ruleSet         *RuleSet
	ipChecker       *builtin.IPListChecker
	rateLimiter     *builtin.RateLimiter
	geoChecker      *builtin.GeoChecker
	geoResolver     GeoResolver
	aiAnalyzer      AnalyzerInterface
	attackLogCh     chan model.AttackLog
	observationMode bool
	mu              sync.RWMutex
}

func NewPipeline(ruleSet *RuleSet, rpm, burst int) *Pipeline {
	return &Pipeline{
		ruleSet:     ruleSet,
		ipChecker:   builtin.NewIPListChecker(),
		rateLimiter: builtin.NewRateLimiter(rpm, burst),
		geoChecker:  builtin.NewGeoChecker(),
		attackLogCh: make(chan model.AttackLog, 1000),
	}
}

// Close shuts down the pipeline and its background resources.
func (p *Pipeline) Close() {
	p.rateLimiter.Stop()
	close(p.attackLogCh)
}

func (p *Pipeline) AttackLogChan() <-chan model.AttackLog {
	return p.attackLogCh
}

func (p *Pipeline) UpdateIPLists(whitelist, blacklist map[string]bool) {
	p.ipChecker.UpdateLists(whitelist, blacklist)
}

func (p *Pipeline) SetGeoResolver(r GeoResolver) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.geoResolver = r
}

func (p *Pipeline) UpdateGeoRules(blocked []string) {
	p.geoChecker.Update(blocked)
}

func (p *Pipeline) SetAIAnalyzer(a AnalyzerInterface) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.aiAnalyzer = a
}

func (p *Pipeline) UpdateRules(rs *RuleSet) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ruleSet = rs
}

func (p *Pipeline) SetObservationMode(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observationMode = enabled
}

func (p *Pipeline) IsObservationMode() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.observationMode
}

func (p *Pipeline) Inspect(ctx context.Context, req *model.ParsedRequest) *DetectionResult {
	start := time.Now()

	// Stage 1: IP Whitelist - fast pass
	if p.ipChecker.IsWhitelisted(req.ClientIP) {
		return nil // PASS
	}

	// Stage 2: IP Blacklist
	if res := p.ipChecker.Check(ctx, req); res != nil {
		res.Latency = time.Since(start).Nanoseconds()
		p.emitLog(req, res)
		return res
	}

	// Stage 2.5: Geo-Location Blocking
	p.mu.RLock()
	resolver := p.geoResolver
	p.mu.RUnlock()
	if resolver != nil {
		countryCode := resolver.GetCountryCode(req.ClientIP)
		if res := p.geoChecker.Check(ctx, req, countryCode); res != nil {
			res.Latency = time.Since(start).Nanoseconds()
			p.emitLog(req, res)
			return res
		}
	}

	// Stage 3: Rate Limit (skip for localhost in iptables REDIRECT mode)
	if req.ClientIP != "127.0.0.1" && req.ClientIP != "::1" && !p.rateLimiter.Allow(req.ClientIP) {
		res := &DetectionResult{
			Blocked:  true,
			RuleID:   "RATE-LIMIT",
			RuleName: "Rate Limit Exceeded",
			Severity: "medium",
			Message:  "rate limit exceeded for " + req.ClientIP,
			Source:   "rule_engine",
			Latency:  time.Since(start).Nanoseconds(),
		}
		p.emitLog(req, res)
		return res
	}

	// Stage 4: Rule Engine (optimized: pre-extract text per location)
	p.mu.RLock()
	byLocation := p.ruleSet.RulesByLocation()
	aiAnalyzer := p.aiAnalyzer
	p.mu.RUnlock()

	// Extract and normalize text per location once, then match all rules for that location
	for loc, rules := range byLocation {
		select {
		case <-ctx.Done():
			return &DetectionResult{
				Blocked:  true,
				RuleID:   "TIMEOUT",
				RuleName: "Request Timeout",
				Severity: "medium",
				Message:  "request inspection timed out",
				Source:   "rule_engine",
				Latency:  time.Since(start).Nanoseconds(),
			}
		default:
		}

		texts := ExtractLocationText(loc, req)
		for _, rule := range rules {
			if res := matchRulePatterns(rule, texts); res != nil {
				res.Latency = time.Since(start).Nanoseconds()
				p.emitLog(req, res)
				return res
			}
		}
	}

	// Stage 5: AI Analysis
	if aiAnalyzer != nil && !req.SkipAI {
		aiCh := aiAnalyzer.AnalyzeAsync(ctx, req)
		select {
		case res := <-aiCh:
			if res != nil && res.Blocked {
				res.Latency = time.Since(start).Nanoseconds()
				p.emitLog(req, res)
				return res
			}
		case <-ctx.Done():
			return &DetectionResult{
				Blocked:  true,
				RuleID:   "TIMEOUT",
				RuleName: "AI Analysis Timeout",
				Severity: "medium",
				Message:  "AI analysis timed out",
				Source:   "ai",
				Latency:  time.Since(start).Nanoseconds(),
			}
		}
	}

	return nil // PASS
}

func (p *Pipeline) emitLog(req *model.ParsedRequest, res *DetectionResult) {
	headersJSON := ""
	if len(req.Headers) > 0 {
		if b, err := json.Marshal(req.Headers); err == nil {
			headersJSON = string(b)
		}
	}

	l := model.AttackLog{
		ID:          req.ID,
		Timestamp:   req.Timestamp,
		ClientIP:    req.ClientIP,
		SiteID:      req.SiteID,
		SiteName:    req.SiteName,
		Domain:      req.Domain,
		Method:      req.Method,
		Path:        req.Path,
		Headers:     headersJSON,
		BodyPreview: req.BodyPreview,
		RuleID:      res.RuleID,
		RuleName:    res.RuleName,
		Severity:    res.Severity,
		Source:      res.Source,
		Action:      "blocked",
		AIReasoning: "",
	}
	if res.Source == "ai" {
		l.AIReasoning = res.Message
	}

	select {
	case p.attackLogCh <- l:
	default:
		log.Println("attack log channel full, dropping log")
	}
}

// matchRulePatterns runs a rule's compiled regex patterns against pre-extracted text variants.
func matchRulePatterns(rule Rule, texts []string) *DetectionResult {
	type patternHolder interface {
		Patterns() []*regexp.Regexp
	}
	holder, ok := rule.(patternHolder)
	if !ok {
		return nil
	}
	patterns := holder.Patterns()
	for _, text := range texts {
		for _, re := range patterns {
			if match := re.FindString(text); match != "" {
				return &DetectionResult{
					Blocked:  true,
					RuleID:   rule.ID(),
					RuleName: rule.Name(),
					Severity: rule.Severity(),
					Message:  "blocked: " + match,
					Source:   "rule_engine",
				}
			}
		}
	}
	return nil
}
