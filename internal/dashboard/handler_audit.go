package dashboard

import (
	"net/http"
	"strconv"
	"time"

	"zhiyuwaf/internal/store"
)

func (s *Server) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := store.AuditFilter{
		Action: r.URL.Query().Get("action"),
		Status: r.URL.Query().Get("status"),
		Actor:  r.URL.Query().Get("actor"),
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			filter.Since = t
		}
	}
	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
			filter.Until = t
		}
	}

	events, total, err := s.store.ListAuditEvents((page-1)*limit, limit, filter)
	if err != nil {
		http.Error(w, `{"error":"failed to list audit events"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  events,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
