package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

func (s *Server) handleListIP(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	if listType == "" {
		listType = "blacklist"
	}

	entries, err := s.store.ListIPEntries(listType)
	if err != nil {
		http.Error(w, `{"error":"failed to list IP entries"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleAddIP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var entry model.IPEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	entry.ID = uuid.New().String()
	if err := s.store.AddIPEntry(entry); err != nil {
		http.Error(w, `{"error":"failed to add IP entry"}`, http.StatusInternalServerError)
		return
	}
	if s.OnIPListChanged != nil {
		s.OnIPListChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "ip_add", "success", entry.IPAddress+" "+entry.ListType)

	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) handleRemoveIP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.RemoveIPEntry(id); err != nil {
		http.Error(w, `{"error":"failed to remove IP entry"}`, http.StatusInternalServerError)
		return
	}
	if s.OnIPListChanged != nil {
		s.OnIPListChanged()
	}
	s.recordAudit("admin", dashboardClientIP(r), "ip_remove", "success", "id="+id)

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleBatchAddIP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB max
	var req struct {
		ListType string `json:"list_type"`
		Entries  string `json:"entries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ListType != "blacklist" && req.ListType != "whitelist" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "list_type must be blacklist or whitelist"})
		return
	}

	imported := 0
	skipped := 0
	lines := strings.Split(req.Entries, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "IP optional note"
		parts := strings.SplitN(line, " ", 2)
		ipStr := parts[0]
		note := ""
		if len(parts) > 1 {
			note = strings.TrimSpace(parts[1])
		}

		// Validate IP (also accept CIDR)
		if net.ParseIP(ipStr) == nil {
			if _, _, err := net.ParseCIDR(ipStr); err != nil {
				skipped++
				continue
			}
		}

		entry := model.IPEntry{
			ID:        uuid.New().String(),
			IPAddress: ipStr,
			ListType:  req.ListType,
			Note:      note,
		}
		if err := s.store.AddIPEntry(entry); err != nil {
			skipped++
		} else {
			imported++
		}
	}

	if imported > 0 && s.OnIPListChanged != nil {
		s.OnIPListChanged()
	}

	s.recordAudit("admin", dashboardClientIP(r), "ip_batch_import", "success",
		fmt.Sprintf("type=%s imported=%d skipped=%d", req.ListType, imported, skipped))

	writeJSON(w, http.StatusOK, map[string]int{"imported": imported, "skipped": skipped})
}

func (s *Server) handleExportIPList(w http.ResponseWriter, r *http.Request) {
	listType := r.URL.Query().Get("type")
	if listType == "" {
		listType = "blacklist"
	}

	entries, err := s.store.ListIPEntries(listType)
	if err != nil {
		http.Error(w, `{"error":"failed to list IP entries"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=iplist-%s.csv", listType))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"ip_address", "note", "created_at"})
	for _, e := range entries {
		writer.Write([]string{e.IPAddress, e.Note, e.CreatedAt.Format("2006-01-02 15:04:05")})
	}
}
