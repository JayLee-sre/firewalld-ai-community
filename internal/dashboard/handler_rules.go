package dashboard

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/model"
)

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListRules()
	if err != nil {
		http.Error(w, `{"error":"failed to list rules"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var rule model.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	rule.ID = uuid.New().String()
	if !s.validateRulePayload(w, &rule) {
		return
	}
	if err := s.store.CreateRule(rule); err != nil {
		http.Error(w, `{"error":"failed to create rule"}`, http.StatusInternalServerError)
		return
	}
	s.afterRulesChanged("rule_create", rule.ID)

	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	id := chi.URLParam(r, "id")
	var rule model.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	rule.ID = id
	if !s.validateRulePayload(w, &rule) {
		return
	}
	if err := s.store.UpdateRule(rule); err != nil {
		http.Error(w, `{"error":"failed to update rule"}`, http.StatusInternalServerError)
		return
	}
	s.afterRulesChanged("rule_update", rule.ID)

	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteRule(id); err != nil {
		http.Error(w, `{"error":"failed to delete rule"}`, http.StatusInternalServerError)
		return
	}
	s.afterRulesChanged("rule_delete", id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) validateRulePayload(w http.ResponseWriter, rule *model.Rule) bool {
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" || len(rule.Patterns) == 0 || len(rule.MatchLocations) == 0 {
		http.Error(w, `{"error":"name, patterns and match_locations are required"}`, http.StatusBadRequest)
		return false
	}
	if rule.Severity == "" {
		rule.Severity = "medium"
	}
	for _, p := range rule.Patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			http.Error(w, `{"error":"patterns cannot contain empty lines"}`, http.StatusBadRequest)
			return false
		}
		if _, err := regexp.Compile(p); err != nil {
			http.Error(w, `{"error":"invalid regexp pattern: `+err.Error()+`"}`, http.StatusBadRequest)
			return false
		}
	}
	return true
}

func (s *Server) afterRulesChanged(action, id string) {
	if s.OnRulesChanged != nil {
		s.OnRulesChanged()
	}
	s.recordAudit("admin", "", action, "success", "rule="+id)
}

// handleTestRule tests a rule pattern against sample input.
// POST /api/v1/rules/test { "pattern": "...", "text": "..." }
func (s *Server) handleTestRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pattern string `json:"pattern"`
		Text    string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Pattern == "" || req.Text == "" {
		http.Error(w, `{"error":"pattern and text are required"}`, http.StatusBadRequest)
		return
	}
	re, err := regexp.Compile(req.Pattern)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":  "invalid pattern: " + err.Error(),
			"match":  false,
		})
		return
	}
	match := re.FindString(req.Text)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"matched":      match != "",
		"match_sample": match,
	})
}

// handleRulePreview tests a full rule config against a sample request.
// POST /api/v1/rules/preview
func (s *Server) handleRulePreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Patterns       []string `json:"patterns"`
		MatchLocations []string `json:"match_locations"`
		URL            string   `json:"url"`
		Body           string   `json:"body"`
		Headers        string   `json:"headers"`
		UserAgent      string   `json:"user_agent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	compiled := make([]*regexp.Regexp, 0, len(req.Patterns))
	for _, p := range req.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error": "invalid pattern: " + p + " — " + err.Error(),
			})
			return
		}
		compiled = append(compiled, re)
	}

	parsed := &model.ParsedRequest{
		URL:       req.URL,
		Path:      req.URL,
		Body:      []byte(req.Body),
		UserAgent: req.UserAgent,
		Headers:   map[string][]string{"User-Agent": {req.UserAgent}},
	}

	for _, loc := range req.MatchLocations {
		texts := engine.ExtractLocationText(loc, parsed)
		for _, re := range compiled {
			for _, text := range texts {
				if match := re.FindString(text); match != "" {
					writeJSON(w, http.StatusOK, map[string]interface{}{
						"matched":  true,
						"location": loc,
						"sample":   match,
					})
					return
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"matched": false,
	})
}
