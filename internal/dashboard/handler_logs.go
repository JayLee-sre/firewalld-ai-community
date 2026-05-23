package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/store"
)

func (s *Server) handleListLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	filter := store.LogFilter{
		ClientIP: r.URL.Query().Get("client_ip"),
		SiteID:   r.URL.Query().Get("site_id"),
		Severity: r.URL.Query().Get("severity"),
		Source:   r.URL.Query().Get("source"),
	}

	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			filter.Since = t
		}
	}

	logs, total, err := s.store.ListAttackLogs(offset, limit, filter)
	if err != nil {
		http.Error(w, `{"error":"failed to list logs"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (s *Server) handleGetLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	logEntry, err := s.store.GetAttackLog(id)
	if err != nil {
		http.Error(w, `{"error":"failed to get log"}`, http.StatusInternalServerError)
		return
	}
	if logEntry == nil {
		http.Error(w, `{"error":"log not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, logEntry)
}

func (s *Server) handleMarkLogFalsePositive(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	id := chi.URLParam(r, "id")
	var body struct {
		AddWhitelist bool   `json:"add_whitelist"`
		Note         string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	logEntry, err := s.store.GetAttackLog(id)
	if err != nil {
		http.Error(w, `{"error":"failed to get log"}`, http.StatusInternalServerError)
		return
	}
	if logEntry == nil {
		http.Error(w, `{"error":"log not found"}`, http.StatusNotFound)
		return
	}

	if err := s.store.MarkAttackLogReview(id, true); err != nil {
		http.Error(w, `{"error":"failed to mark log"}`, http.StatusInternalServerError)
		return
	}
	if body.AddWhitelist {
		note := body.Note
		if note == "" {
			note = "AI 误报学习自动加入白名单"
		}
		if err := s.store.AddIPEntry(model.IPEntry{
			ID:        uuid.New().String(),
			IPAddress: logEntry.ClientIP,
			ListType:  "whitelist",
			Note:      note,
		}); err != nil {
			http.Error(w, `{"error":"failed to add whitelist"}`, http.StatusInternalServerError)
			return
		}
		if s.OnIPListChanged != nil {
			s.OnIPListChanged()
		}
	}

	s.recordAudit("admin", dashboardClientIP(r), "ai_false_positive", "success", "log="+id+" ip="+logEntry.ClientIP)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "updated",
		"add_whitelist": body.AddWhitelist,
	})
}

func (s *Server) handleMarkLogReviewed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.MarkAttackLogReview(id, false); err != nil {
		http.Error(w, `{"error":"failed to mark log"}`, http.StatusInternalServerError)
		return
	}
	s.recordAudit("admin", dashboardClientIP(r), "ai_review", "success", "log="+id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleExportLogs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	filter := store.LogFilter{
		ClientIP: r.URL.Query().Get("client_ip"),
		SiteID:   r.URL.Query().Get("site_id"),
		Severity: r.URL.Query().Get("severity"),
		Source:   r.URL.Query().Get("source"),
	}
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			filter.Since = t
		}
	}

	// Fetch all matching logs in batches
	var allLogs []model.AttackLog
	offset := 0
	for {
		batch, total, err := s.store.ListAttackLogs(offset, 1000, filter)
		if err != nil {
			http.Error(w, `{"error":"failed to fetch logs"}`, http.StatusInternalServerError)
			return
		}
		allLogs = append(allLogs, batch...)
		offset += len(batch)
		if offset >= total || len(batch) == 0 {
			break
		}
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=attack-logs-"+time.Now().Format("20060102")+".json")
		json.NewEncoder(w).Encode(allLogs)
		return
	}

	// CSV format
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=attack-logs-"+time.Now().Format("20060102")+".csv")

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	csvWriter.Write([]string{"timestamp", "client_ip", "region", "method", "path", "rule_id", "rule_name", "severity", "source", "action"})
	for _, l := range allLogs {
		csvWriter.Write([]string{
			l.Timestamp.Format(time.RFC3339),
			l.ClientIP,
			l.Region,
			l.Method,
			l.Path,
			l.RuleID,
			l.RuleName,
			l.Severity,
			l.Source,
			l.Action,
		})
	}
}
