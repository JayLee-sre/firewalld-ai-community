package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

func (s *Server) handleGetAIStats(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 || hours > 24*30 {
		hours = 24
	}
	stats, err := s.store.GetAttackStatsBySite(time.Now().Add(-time.Duration(hours)*time.Hour), r.URL.Query().Get("site_id"))
	if err != nil {
		http.Error(w, `{"error":"failed to get AI stats"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hours":                hours,
		"ai_count":             stats.AICount,
		"ai_false_positive":    stats.AIFalsePositiveCount,
		"ai_effective_blocked": stats.AICount - stats.AIFalsePositiveCount,
		"ai_reviewed":          stats.AIReviewedCount,
		"by_source":            stats.BySource,
	})
}

func (s *Server) handleListAISuggestions(w http.ResponseWriter, r *http.Request) {
	hours, _ := strconv.Atoi(r.URL.Query().Get("hours"))
	if hours <= 0 || hours > 24*30 {
		hours = 24 * 7
	}
	minCount, _ := strconv.Atoi(r.URL.Query().Get("min_count"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	suggestions, err := s.store.GetAIRuleSuggestionsBySite(time.Now().Add(-time.Duration(hours)*time.Hour), minCount, limit, r.URL.Query().Get("site_id"))
	if err != nil {
		http.Error(w, `{"error":"failed to list AI suggestions"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  suggestions,
		"hours": hours,
	})
}

func (s *Server) handlePromoteAISuggestion(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Severity    string   `json:"severity"`
		Patterns    []string `json:"patterns"`
		Enabled     bool     `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if body.Name == "" || len(body.Patterns) == 0 {
		http.Error(w, `{"error":"name and patterns are required"}`, http.StatusBadRequest)
		return
	}
	if body.Severity == "" {
		body.Severity = "medium"
	}
	if body.Description == "" {
		body.Description = "由 AI 重复命中沉淀的建议规则，建议人工复核后启用"
	}

	rule := model.Rule{
		ID:             uuid.New().String(),
		Name:           body.Name,
		Description:    body.Description,
		Severity:       body.Severity,
		Enabled:        body.Enabled,
		Patterns:       body.Patterns,
		MatchLocations: []string{"path"},
	}
	if err := s.store.CreateRule(rule); err != nil {
		http.Error(w, `{"error":"failed to create rule"}`, http.StatusInternalServerError)
		return
	}
	if s.OnRulesChanged != nil {
		s.OnRulesChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "ai_rule_promote", "success", "rule="+rule.ID)
	writeJSON(w, http.StatusCreated, rule)
}
