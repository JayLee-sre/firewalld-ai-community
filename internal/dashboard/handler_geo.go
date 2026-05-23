package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"zhiyuwaf/internal/model"
)

func (s *Server) handleListGeoRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.store.ListGeoRules()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, 200, rules)
}

func (s *Server) handleAddGeoRule(w http.ResponseWriter, r *http.Request) {
	var rule model.GeoRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	if rule.Country == "" {
		http.Error(w, "country is required", 400)
		return
	}
	if rule.Action == "" {
		rule.Action = "block"
	}
	// Default to enabled for new rules (unless explicitly set to false)
	if !rule.Enabled {
		rule.Enabled = true
	}
	if err := s.store.AddGeoRule(rule); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if s.OnGeoRulesChanged != nil {
		s.OnGeoRulesChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "geo_add", "success", rule.Country+" "+rule.Action)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleUpdateGeoRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var rule model.GeoRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	rule.ID = id
	if err := s.store.UpdateGeoRule(rule); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if s.OnGeoRulesChanged != nil {
		s.OnGeoRulesChanged()
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleRemoveGeoRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.RemoveGeoRule(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if s.OnGeoRulesChanged != nil {
		s.OnGeoRulesChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "geo_remove", "success", "id="+id)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
