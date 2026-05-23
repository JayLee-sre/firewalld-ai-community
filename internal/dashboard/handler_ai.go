package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"zhiyuwaf/internal/ai"
	"zhiyuwaf/internal/ai/openai"
	"zhiyuwaf/internal/store"
)

const communityDailyAILimit = 50

// LoadAISettingsFromDB loads persisted AI settings from DB into cfg.
// Call this on startup before initializing the AI analyzer.
// NOTE: ai_enabled is NOT loaded from DB — YAML config is the source of truth for on/off.
// DB only overrides supplementary settings (provider, model, key, timeouts).
func (s *Server) LoadAISettingsFromDB() {
	m, err := s.store.ListSettings()
	if err != nil {
		log.Printf("warning: failed to load AI settings from DB: %v", err)
		return
	}

	// ai_enabled intentionally skipped — YAML config controls this
	if v, ok := m["ai_provider"]; ok {
		s.cfg.AI.Provider = v
	}
	if s.cfg.AI.Provider != "openai" {
		s.cfg.AI.Provider = "openai"
		_ = s.store.SetSetting("ai_provider", "openai")
	}
	if v, ok := m["ai_async_timeout"]; ok {
		fmt.Sscanf(v, "%d", &s.cfg.AI.AsyncTimeout)
	}
	if v, ok := m["ai_cache_ttl"]; ok {
		fmt.Sscanf(v, "%d", &s.cfg.AI.CacheTTL)
	}
	if v, ok := m["ai_max_requests"]; ok {
		fmt.Sscanf(v, "%d", &s.cfg.AI.MaxRequests)
	}
	if v, ok := m["ai_fail_open"]; ok {
		s.cfg.AI.FailOpen = v == "true"
	}
	if v, ok := m["ai_high_risk_paths"]; ok {
		s.cfg.AI.HighRiskPaths = splitCSV(v)
	}
	if v, ok := m["ai_openai_api_key"]; ok {
		s.cfg.AI.Providers.OpenAI.APIKey = v
	}
	if v, ok := m["ai_openai_model"]; ok {
		s.cfg.AI.Providers.OpenAI.Model = v
	}
	if v, ok := m["ai_openai_base_url"]; ok {
		s.cfg.AI.Providers.OpenAI.BaseURL = v
	}

	log.Printf("AI settings loaded from DB: enabled=%v provider=%s", s.cfg.AI.Enabled, s.cfg.AI.Provider)
}

func (s *Server) handleGetAIProviders(w http.ResponseWriter, r *http.Request) {
	openaiKey := maskAPIKey(s.cfg.AI.Providers.OpenAI.APIKey)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":              s.cfg.AI.Enabled,
		"provider":             "openai",
		"async_timeout":        s.cfg.AI.AsyncTimeout,
		"cache_ttl":            s.cfg.AI.CacheTTL,
		"max_requests_per_min": s.cfg.AI.MaxRequests,
		"fail_open":            s.cfg.AI.FailOpen,
		"high_risk_paths":      s.cfg.AI.HighRiskPaths,
		"providers": map[string]interface{}{
			"openai": map[string]interface{}{
				"api_key":  openaiKey,
				"model":    s.cfg.AI.Providers.OpenAI.Model,
				"base_url": s.cfg.AI.Providers.OpenAI.BaseURL,
			},
		},
	})
}

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

func (s *Server) handleUpdateAIProvider(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	name := chi.URLParam(r, "name")
	switch name {
	case "openai":
		if v, ok := update["api_key"].(string); ok {
			if !isMaskedAPIKey(v) {
				s.cfg.AI.Providers.OpenAI.APIKey = v
				s.store.SetSetting("ai_openai_api_key", v)
			}
		}
		if v, ok := update["model"].(string); ok {
			s.cfg.AI.Providers.OpenAI.Model = v
			s.store.SetSetting("ai_openai_model", v)
		}
		if v, ok := update["base_url"].(string); ok {
			s.cfg.AI.Providers.OpenAI.BaseURL = v
			s.store.SetSetting("ai_openai_base_url", v)
		}
	default:
		http.Error(w, `{"error":"unknown provider"}`, http.StatusBadRequest)
		return
	}

	// Reinitialize AI analyzer with new settings
	if s.OnAIConfigChanged != nil {
		s.OnAIConfigChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "ai_provider_update", "success", "provider="+name)

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func isMaskedAPIKey(key string) bool {
	return strings.HasPrefix(key, "****")
}

func (s *Server) handleUpdateAIGlobal(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if v, ok := update["enabled"].(bool); ok {
		s.cfg.AI.Enabled = v
		s.store.SetSetting("ai_enabled", fmt.Sprintf("%v", v))
	}
	if v, ok := update["provider"].(string); ok {
		if v != "openai" {
			http.Error(w, `{"error":"only openai-compatible provider is supported"}`, http.StatusBadRequest)
			return
		}
		s.cfg.AI.Provider = v
		s.store.SetSetting("ai_provider", v)
	}
	if v, ok := update["async_timeout"].(float64); ok {
		s.cfg.AI.AsyncTimeout = int(v)
		s.store.SetSetting("ai_async_timeout", fmt.Sprintf("%d", int(v)))
	}
	if v, ok := update["cache_ttl"].(float64); ok {
		s.cfg.AI.CacheTTL = int(v)
		s.store.SetSetting("ai_cache_ttl", fmt.Sprintf("%d", int(v)))
	}
	if v, ok := update["max_requests_per_min"].(float64); ok {
		s.cfg.AI.MaxRequests = int(v)
		s.store.SetSetting("ai_max_requests", fmt.Sprintf("%d", int(v)))
	}
	if v, ok := update["fail_open"].(bool); ok {
		s.cfg.AI.FailOpen = v
		s.store.SetSetting("ai_fail_open", fmt.Sprintf("%v", v))
	}
	if v, ok := update["high_risk_paths"].(string); ok {
		s.cfg.AI.HighRiskPaths = splitCSV(v)
		s.store.SetSetting("ai_high_risk_paths", strings.Join(s.cfg.AI.HighRiskPaths, ","))
	}
	if raw, ok := update["high_risk_paths"].([]interface{}); ok {
		paths := make([]string, 0, len(raw))
		for _, item := range raw {
			if path, ok := item.(string); ok && strings.TrimSpace(path) != "" {
				paths = append(paths, strings.TrimSpace(path))
			}
		}
		s.cfg.AI.HighRiskPaths = paths
		s.store.SetSetting("ai_high_risk_paths", strings.Join(paths, ","))
	}

	if s.OnAIConfigChanged != nil {
		s.OnAIConfigChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "ai_global_update", "success", "provider="+s.cfg.AI.Provider)

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *Server) handleTestAI(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if !s.cfg.AI.Enabled {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "disabled",
			"message": "AI 检测未启用",
		})
		return
	}

	var provider ai.Provider
	switch s.cfg.AI.Provider {
	case "openai":
		provider = openai.NewClient(
			s.cfg.AI.Providers.OpenAI.APIKey,
			s.cfg.AI.Providers.OpenAI.Model,
			s.cfg.AI.Providers.OpenAI.BaseURL,
		)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "未知的 AI 提供商: " + s.cfg.AI.Provider,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	testReq := ai.AnalysisRequest{
		ClientIP:    "127.0.0.1",
		Method:      "GET",
		Path:        "/test",
		Headers:     map[string][]string{"User-Agent": {"ZhiYu-WAF-test"}},
		BodyPreview: "",
	}

	start := time.Now()
	resp, err := provider.Analyze(ctx, testReq)
	latency := time.Since(start)

	if err != nil {
		log.Printf("AI test failed: %v", err)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("连接失败: %v", err),
			"latency": latency.Milliseconds(),
		})
		return
	}

	var model string
	switch s.cfg.AI.Provider {
	case "openai":
		model = s.cfg.AI.Providers.OpenAI.Model
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"message":  fmt.Sprintf("连接成功 (%s, %dms)", s.cfg.AI.Provider, latency.Milliseconds()),
		"provider": s.cfg.AI.Provider,
		"model":    model,
		"latency":  latency.Milliseconds(),
		"response": resp,
	})
}

// getTodayAIUsageKey returns the settings key for today's AI call counter.
func getTodayAIUsageKey() string {
	return fmt.Sprintf("ai_usage_%s", time.Now().Format("20060102"))
}

// GetTodayAIUsage returns how many AI calls have been made today.
func (s *Server) GetTodayAIUsage() int {
	key := getTodayAIUsageKey()
	val, err := s.store.GetSetting(key)
	if err != nil || val == "" {
		return 0
	}
	var count int
	fmt.Sscanf(val, "%d", &count)
	return count
}

// IncrementAIUsage increments the daily AI call counter.
func (s *Server) IncrementAIUsage() {
	key := getTodayAIUsageKey()
	val, _ := s.store.GetSetting(key)
	var count int
	fmt.Sscanf(val, "%d", &count)
	count++
	s.store.SetSetting(key, fmt.Sprintf("%d", count))
}

// IsCommunityAIAllowed checks if community edition can make an AI call.
func (s *Server) IsCommunityAIAllowed() bool {
	if s.currentEdition() == "pro" {
		return true
	}
	return s.GetTodayAIUsage() < communityDailyAILimit
}

func (s *Server) handleGetAIUsage(w http.ResponseWriter, r *http.Request) {
	edition := s.currentEdition()
	usage := s.GetTodayAIUsage()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"edition":      edition,
		"today_used":   usage,
		"daily_limit":  communityDailyAILimit,
		"remaining":    communityDailyAILimit - usage,
		"is_pro":       edition == "pro",
	})
}

func (s *Server) handleGenerateRule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Description) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请提供规则描述"})
		return
	}

	if !s.cfg.AI.Enabled || s.cfg.AI.Providers.OpenAI.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "AI 引擎未启用或未配置 API Key"})
		return
	}

	provider := openai.NewClient(
		s.cfg.AI.Providers.OpenAI.APIKey,
		s.cfg.AI.Providers.OpenAI.Model,
		s.cfg.AI.Providers.OpenAI.BaseURL,
	)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	genReq := ai.AnalysisRequest{
		ClientIP:    "127.0.0.1",
		Method:      "POST",
		Path:        "/ai/generate-rule",
		BodyPreview: fmt.Sprintf("请根据以下描述生成 WAF 检测规则的正则表达式：%s", req.Description),
	}

	resp, err := provider.Analyze(ctx, genReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("AI 生成失败: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"description": req.Description,
		"ai_response": resp,
	})
}

func (s *Server) handleThreatProfile(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if v := r.URL.Query().Get("hours"); v != "" {
		fmt.Sscanf(v, "%d", &hours)
	}
	if hours < 1 || hours > 168 {
		hours = 24
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	stats, err := s.store.GetAttackStats(since)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "获取统计数据失败"})
		return
	}

	// Top attacker IPs
	logs, _, err := s.store.ListAttackLogs(0, 100, store.LogFilter{Since: since})
	var topIPs []map[string]interface{}
	ipCount := make(map[string]int)
	if err == nil {
		for _, l := range logs {
			ipCount[l.ClientIP]++
		}
		type ipStat struct {
			IP    string
			Count int
		}
		var sorted []ipStat
		for ip, c := range ipCount {
			sorted = append(sorted, ipStat{ip, c})
		}
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].Count > sorted[i].Count {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		limit := 10
		if len(sorted) < limit {
			limit = len(sorted)
		}
		for _, s := range sorted[:limit] {
			topIPs = append(topIPs, map[string]interface{}{
				"ip":    s.IP,
				"count": s.Count,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hours":            hours,
		"stats":            stats,
		"top_attacker_ips": topIPs,
	})
}
