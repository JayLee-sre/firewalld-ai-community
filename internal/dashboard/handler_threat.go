package dashboard

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleGetThreatIntelStatus(w http.ResponseWriter, r *http.Request) {
	// Get threat intel IPs from blacklist (synced by threat intel syncer)
	var threatIPs []map[string]interface{}
	if s.store != nil {
		entries, err := s.store.ListIPEntries("blacklist")
		if err == nil {
			for _, e := range entries {
				if strings.Contains(e.Note, "威胁情报") || strings.Contains(e.Note, "auto-synced") || strings.Contains(e.Note, "abuseipdb") {
					threatIPs = append(threatIPs, map[string]interface{}{
						"ip":         e.IPAddress,
						"note":       e.Note,
						"created_at": e.CreatedAt,
					})
				}
			}
		}
	}

	if s.ThreatSyncerStatus == nil {
		writeJSON(w, 200, map[string]interface{}{
			"provider":   "abuseipdb",
			"last_sync":  nil,
			"ip_count":   0,
			"threat_ips": threatIPs,
		})
		return
	}
	lastSync, count := s.ThreatSyncerStatus()
	writeJSON(w, 200, map[string]interface{}{
		"provider":   "abuseipdb",
		"last_sync":  lastSync,
		"ip_count":   count,
		"threat_ips": threatIPs,
	})
}

func (s *Server) handleSyncThreatIntel(w http.ResponseWriter, r *http.Request) {
	if s.ThreatSyncerSync == nil {
		writeJSON(w, 400, map[string]interface{}{
			"error":   "threat intelligence not configured",
			"message": "请先配置 AbuseIPDB API Key",
		})
		return
	}
	go s.ThreatSyncerSync()
	writeJSON(w, 200, map[string]interface{}{
		"status":  "syncing",
		"message": "同步已触发，请稍后刷新状态查看结果",
	})
}

func (s *Server) handleUpdateThreatIntelConfig(w http.ResponseWriter, r *http.Request) {
	var cfg struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	if cfg.APIKey != "" {
		s.store.SetSetting("threatintel_api_key", cfg.APIKey)
	}
	if s.OnThreatIntelChanged != nil {
		s.OnThreatIntelChanged()
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
