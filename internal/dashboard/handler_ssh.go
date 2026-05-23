package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleGetSSHStats(w http.ResponseWriter, r *http.Request) {
	since := time.Now().Add(-24 * time.Hour)
	if q := r.URL.Query().Get("hours"); q != "" {
		if h, err := strconv.Atoi(q); err == nil && h > 0 {
			since = time.Now().Add(-time.Duration(h) * time.Hour)
		}
	}

	stats, err := s.store.GetSSHStats(since)
	if err != nil {
		http.Error(w, `{"error":"获取SSH统计失败"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleListSSHEvents(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit
	clientIP := r.URL.Query().Get("client_ip")
	eventType := r.URL.Query().Get("event_type")
	username := r.URL.Query().Get("username")

	events, total, err := s.store.ListSSHEvents(offset, limit, clientIP, eventType, username)
	if err != nil {
		http.Error(w, `{"error":"获取SSH事件失败"}`, 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  events,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
