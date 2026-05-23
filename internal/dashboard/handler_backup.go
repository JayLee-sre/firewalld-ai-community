package dashboard

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"zhiyuwaf/internal/model"
)

type BackupData struct {
	ExportedAt time.Time         `json:"exported_at"`
	Version    string            `json:"version"`
	Rules      []model.Rule      `json:"rules"`
	IPEntries  []model.IPEntry   `json:"ip_entries"`
	Sites      []model.Site      `json:"sites"`
	GeoRules   []model.GeoRule   `json:"geo_rules"`
	Settings   map[string]string `json:"settings"`
}

var sensitiveSettings = map[string]bool{
	"admin_password_hash": true,
	"license_token":       true,
	"license_key":         true,
}

func (s *Server) handleExportBackup(w http.ResponseWriter, r *http.Request) {
	rules, _ := s.store.ListRules()
	if rules == nil {
		rules = []model.Rule{}
	}

	var ipEntries []model.IPEntry
	bl, _ := s.store.ListIPEntries("blacklist")
	wl, _ := s.store.ListIPEntries("whitelist")
	ipEntries = append(ipEntries, bl...)
	ipEntries = append(ipEntries, wl...)
	if ipEntries == nil {
		ipEntries = []model.IPEntry{}
	}

	sites, _ := s.store.ListSites()
	if sites == nil {
		sites = []model.Site{}
	}
	geoRules, _ := s.store.ListGeoRules()
	if geoRules == nil {
		geoRules = []model.GeoRule{}
	}

	settings, _ := s.store.ListSettings()
	// Filter out sensitive settings
	cleanSettings := make(map[string]string)
	for k, v := range settings {
		if !sensitiveSettings[k] {
			cleanSettings[k] = v
		}
	}

	backup := BackupData{
		ExportedAt: time.Now(),
		Version:    "1.0",
		Rules:      rules,
		IPEntries:  ipEntries,
		Sites:      sites,
		GeoRules:   geoRules,
		Settings:   cleanSettings,
	}

	s.recordAudit("admin", dashboardClientIP(r), "backup_export", "success", "configuration exported")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=zhiyu-waf-backup-"+time.Now().Format("20060102")+".json")
	json.NewEncoder(w).Encode(backup)
}

func (s *Server) handleImportBackup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB max

	var backup BackupData
	if err := json.NewDecoder(r.Body).Decode(&backup); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid backup file: " + err.Error()})
		return
	}

	imported := map[string]int{}
	errors := []string{}

	// Import rules
	for _, rule := range backup.Rules {
		if rule.ID == "" || rule.Name == "" {
			continue
		}
		if err := s.store.CreateRule(rule); err != nil {
			errors = append(errors, "rule "+rule.Name+": "+err.Error())
		} else {
			imported["rules"]++
		}
	}
	if imported["rules"] > 0 && s.OnRulesChanged != nil {
		s.OnRulesChanged()
	}

	// Import IP entries
	for _, entry := range backup.IPEntries {
		if entry.IPAddress == "" || entry.ListType == "" {
			continue
		}
		if err := s.store.AddIPEntry(entry); err != nil {
			errors = append(errors, "ip "+entry.IPAddress+": "+err.Error())
		} else {
			imported["ip_entries"]++
		}
	}
	if imported["ip_entries"] > 0 && s.OnIPListChanged != nil {
		s.OnIPListChanged()
	}

	// Import sites
	for _, site := range backup.Sites {
		if site.ID == "" || site.Name == "" {
			continue
		}
		if err := s.store.CreateSite(site); err != nil {
			errors = append(errors, "site "+site.Name+": "+err.Error())
		} else {
			imported["sites"]++
		}
	}
	if imported["sites"] > 0 && s.OnSitesChanged != nil {
		s.OnSitesChanged()
	}

	// Import geo rules
	for _, rule := range backup.GeoRules {
		if rule.Country == "" {
			continue
		}
		if err := s.store.AddGeoRule(rule); err != nil {
			errors = append(errors, "geo "+rule.Country+": "+err.Error())
		} else {
			imported["geo_rules"]++
		}
	}
	if imported["geo_rules"] > 0 && s.OnGeoRulesChanged != nil {
		s.OnGeoRulesChanged()
	}

	// Import settings
	for k, v := range backup.Settings {
		if sensitiveSettings[k] {
			continue
		}
		if err := s.store.SetSetting(k, v); err != nil {
			errors = append(errors, "setting "+k+": "+err.Error())
		} else {
			imported["settings"]++
		}
	}

	s.recordAudit("admin", dashboardClientIP(r), "backup_import", "success",
		fmt.Sprintf("rules=%d ips=%d sites=%d", imported["rules"], imported["ip_entries"], imported["sites"]))

	log.Printf("backup import: %+v, errors: %d", imported, len(errors))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"imported": imported,
		"errors":   errors,
	})
}
