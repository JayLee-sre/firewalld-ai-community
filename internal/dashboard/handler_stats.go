package dashboard

import (
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	sinceStr := r.URL.Query().Get("since")
	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	stats, err := s.store.GetAttackStatsBySite(since, r.URL.Query().Get("site_id"))
	if err != nil {
		http.Error(w, `{"error":"failed to get stats"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleGetTimeSeries(w http.ResponseWriter, r *http.Request) {
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
		hours = h
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	stats, err := s.store.GetAttackStatsBySite(since, r.URL.Query().Get("site_id"))
	if err != nil {
		http.Error(w, `{"error":"failed to get timeseries"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
